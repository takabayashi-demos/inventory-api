// Inventory API - Walmart Platform
// Real-time inventory management with intentional issues.
//
// FIXED ISSUES:
// - Removed hardcoded DB credentials (now using env vars)
// - Removed debug endpoint exposing internal config
//
// REMAINING ISSUES (for demo):
// - Off-by-one error in stock count (bug)
// - No mutex on concurrent stock updates (race condition)
// - Panic on nil map access (bug)
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// Database configuration from environment variables
var (
	DB_HOST     = getEnv("DB_HOST", "inventory-db.walmart.internal")
	DB_USER     = getEnv("DB_USER", "inventory_user")
	DB_PASSWORD = getEnv("DB_PASSWORD", "")
	DB_NAME     = getEnv("DB_NAME", "inventory_prod")
)

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

type Product struct {
	SKU       string  `json:"sku"`
	Name      string  `json:"name"`
	Stock     int     `json:"stock"`
	Price     float64 `json:"price"`
	Warehouse string  `json:"warehouse"`
	UpdatedAt string  `json:"updated_at"`
}

var (
	inventory map[string]*Product
	// ❌ BUG: mu declared but not always used - race condition
	mu sync.Mutex
)

func init() {
	inventory = map[string]*Product{
		"SKU-001": {SKU: "SKU-001", Name: "Samsung 65\" 4K TV", Stock: 150, Price: 599.99, Warehouse: "us-east-1"},
		"SKU-002": {SKU: "SKU-002", Name: "Apple iPhone 15 Pro", Stock: 500, Price: 999.99, Warehouse: "us-east-1"},
		"SKU-003": {SKU: "SKU-003", Name: "Sony WH-1000XM5", Stock: 300, Price: 349.99, Warehouse: "us-west-2"},
		"SKU-004": {SKU: "SKU-004", Name: "Nintendo Switch OLED", Stock: 200, Price: 349.99, Warehouse: "us-west-2"},
		"SKU-005": {SKU: "SKU-005", Name: "Dyson V15 Vacuum", Stock: 75, Price: 749.99, Warehouse: "eu-west-1"},
		"SKU-006": {SKU: "SKU-006", Name: "KitchenAid Mixer", Stock: 120, Price: 429.99, Warehouse: "us-east-1"},
		"SKU-007": {SKU: "SKU-007", Name: "Instant Pot Duo", Stock: 400, Price: 89.99, Warehouse: "us-east-1"},
		"SKU-008": {SKU: "SKU-008", Name: "Lego Star Wars Set", Stock: 250, Price: 159.99, Warehouse: "ap-southeast-1"},
	}

	if DB_PASSWORD == "" {
		log.Println("WARNING: DB_PASSWORD not set, database connection may fail")
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{
		"status": "UP", "service": "inventory-api", "version": "1.5.0",
	})
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "READY"})
}

func listInventoryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	items := make([]*Product, 0)
	for _, p := range inventory {
		items = append(items, p)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": items,
		"total": len(items),
	})
}

func getStockHandler(w http.ResponseWriter, r *http.Request) {
	sku := r.URL.Query().Get("sku")
	if sku == "" {
		http.Error(w, `{"error":"sku parameter required"}`, 400)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// ❌ BUG: No nil check - panics if SKU doesn't exist
	product := inventory[sku]
	json.NewEncoder(w).Encode(product)
}

func reserveStockHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var req struct {
		SKU      string `json:"sku"`
		Quantity int    `json:"quantity"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	product, exists := inventory[req.SKU]
	if !exists {
		http.Error(w, `{"error":"product not found"}`, 404)
		return
	}

	// ❌ BUG: No lock - race condition on concurrent reservations
	// ❌ BUG: Off-by-one error (should be >= not >)
	if product.Stock > req.Quantity {
		product.Stock -= req.Quantity

		// Simulate DB write latency
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

		product.UpdatedAt = time.Now().Format(time.RFC3339)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sku":             req.SKU,
			"reserved":        req.Quantity,
			"remaining_stock": product.Stock,
			"status":          "reserved",
		})
	} else {
		http.Error(w, `{"error":"insufficient stock"}`, 409)
	}
}

func main() {
	port := getEnv("PORT", "8080")

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ready", readyHandler)
	http.HandleFunc("/inventory", listInventoryHandler)
	http.HandleFunc("/stock", getStockHandler)
	http.HandleFunc("/reserve", reserveStockHandler)

	log.Printf("Inventory API starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
