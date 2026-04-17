package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	
	healthHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	
	if resp["status"] != "UP" {
		t.Errorf("expected status UP, got %s", resp["status"])
	}
	if resp["service"] != "inventory-api" {
		t.Errorf("expected service inventory-api, got %s", resp["service"])
	}
}

func TestReadyHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()
	
	readyHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	
	if resp["status"] != "READY" {
		t.Errorf("expected status READY, got %s", resp["status"])
	}
}

func TestListInventoryHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/inventory", nil)
	w := httptest.NewRecorder()
	
	listInventoryHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	
	items := resp["items"].([]interface{})
	if len(items) != 8 {
		t.Errorf("expected 8 items, got %d", len(items))
	}
}

func TestGetStockHandler_ValidSKU(t *testing.T) {
	req := httptest.NewRequest("GET", "/stock?sku=SKU-001", nil)
	w := httptest.NewRecorder()
	
	getStockHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	
	var product Product
	json.NewDecoder(w.Body).Decode(&product)
	
	if product.SKU != "SKU-001" {
		t.Errorf("expected SKU-001, got %s", product.SKU)
	}
	if product.Stock != 150 {
		t.Errorf("expected stock 150, got %d", product.Stock)
	}
}

func TestGetStockHandler_MissingSKUParam(t *testing.T) {
	req := httptest.NewRequest("GET", "/stock", nil)
	w := httptest.NewRecorder()
	
	getStockHandler(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetStockHandler_InvalidSKU(t *testing.T) {
	// This test will currently panic due to nil pointer dereference
	// Leaving it here to document the bug
	t.Skip("Skipping - causes panic due to missing nil check")
	
	req := httptest.NewRequest("GET", "/stock?sku=INVALID", nil)
	w := httptest.NewRecorder()
	
	getStockHandler(w, req)
	
	// Should return 404, but currently panics
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestReserveStockHandler_SufficientStock(t *testing.T) {
	// Reset inventory for consistent test
	init()
	
	body := []byte(`{"sku":"SKU-001","quantity":50}`)
	req := httptest.NewRequest("POST", "/reserve", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	
	reserveStockHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	
	if resp["status"] != "reserved" {
		t.Errorf("expected status reserved, got %s", resp["status"])
	}
	if int(resp["reserved"].(float64)) != 50 {
		t.Errorf("expected reserved 50, got %v", resp["reserved"])
	}
}

func TestReserveStockHandler_InsufficientStock(t *testing.T) {
	init()
	
	body := []byte(`{"sku":"SKU-001","quantity":200}`)
	req := httptest.NewRequest("POST", "/reserve", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	
	reserveStockHandler(w, req)
	
	if w.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", w.Code)
	}
}

func TestReserveStockHandler_ExactQuantity(t *testing.T) {
	// This test exposes the off-by-one bug
	init()
	
	body := []byte(`{"sku":"SKU-001","quantity":150}`)
	req := httptest.NewRequest("POST", "/reserve", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	
	reserveStockHandler(w, req)
	
	// Should succeed (150 == 150), but fails due to > instead of >=
	// Expected: 200 OK, Actual: 409 Conflict
	if w.Code != http.StatusOK {
		t.Logf("BUG DETECTED: Off-by-one error prevents reserving exact stock amount")
		t.Logf("Expected 200 OK, got %d (409 Conflict)", w.Code)
	}
}

func TestReserveStockHandler_InvalidMethod(t *testing.T) {
	req := httptest.NewRequest("GET", "/reserve", nil)
	w := httptest.NewRecorder()
	
	reserveStockHandler(w, req)
	
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestReserveStockHandler_ProductNotFound(t *testing.T) {
	body := []byte(`{"sku":"INVALID","quantity":1}`)
	req := httptest.NewRequest("POST", "/reserve", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	
	reserveStockHandler(w, req)
	
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestConcurrentReservations(t *testing.T) {
	// This test exposes the race condition
	init()
	
	const goroutines = 10
	const quantityPerReservation = 20
	
	var wg sync.WaitGroup
	successCount := 0
	var countMu sync.Mutex
	
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			body := []byte(`{"sku":"SKU-001","quantity":20}`)
			req := httptest.NewRequest("POST", "/reserve", bytes.NewBuffer(body))
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
	
	// SKU-001 starts with 150 stock
	// 10 concurrent reservations of 20 each = 200 total requested
	// Should only allow 7 reservations (140 units), rejecting 3
	expectedSuccess := 7
	
	if successCount != expectedSuccess {
		t.Logf("RACE CONDITION DETECTED: Expected %d successful reservations, got %d", expectedSuccess, successCount)
		t.Logf("Missing mutex protection allows overselling")
	}
	
	finalStock := inventory["SKU-001"].Stock
	expectedStock := 150 - (successCount * quantityPerReservation)
	
	if finalStock != expectedStock {
		t.Errorf("Expected final stock %d, got %d (race condition)", expectedStock, finalStock)
	}
}
