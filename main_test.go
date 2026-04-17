package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestReserveStock_Concurrent(t *testing.T) {
	// Reset inventory to known state
	inventory = map[string]*Product{
		"SKU-TEST": {SKU: "SKU-TEST", Name: "Test Product", Stock: 100, Price: 99.99, Warehouse: "test"},
	}

	var wg sync.WaitGroup
	concurrentRequests := 10
	quantityPerRequest := 10

	successCount := 0
	var countMu sync.Mutex

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			reqBody := map[string]interface{}{
				"sku":      "SKU-TEST",
				"quantity": quantityPerRequest,
			}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest("POST", "/api/reserve", bytes.NewReader(body))
			w := httptest.NewRecorder()

			reserveStockHandler(w, req)

			if w.Code == http.StatusOK {
				countMu.Lock()
				successCount++
				countMu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Exactly 10 reservations should succeed (100 stock / 10 per request)
	if successCount != 10 {
		t.Errorf("Expected 10 successful reservations, got %d", successCount)
	}

	// Final stock should be 0
	mu.Lock()
	finalStock := inventory["SKU-TEST"].Stock
	mu.Unlock()

	if finalStock != 0 {
		t.Errorf("Expected final stock to be 0, got %d", finalStock)
	}
}

func TestReserveStock_ExactStock(t *testing.T) {
	inventory = map[string]*Product{
		"SKU-EXACT": {SKU: "SKU-EXACT", Name: "Exact Test", Stock: 5, Price: 49.99, Warehouse: "test"},
	}

	reqBody := map[string]interface{}{
		"sku":      "SKU-EXACT",
		"quantity": 5,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/reserve", bytes.NewReader(body))
	w := httptest.NewRecorder()

	reserveStockHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 when reserving exact stock amount, got %d", w.Code)
	}

	mu.Lock()
	finalStock := inventory["SKU-EXACT"].Stock
	mu.Unlock()

	if finalStock != 0 {
		t.Errorf("Expected stock to be 0 after reserving all, got %d", finalStock)
	}
}
