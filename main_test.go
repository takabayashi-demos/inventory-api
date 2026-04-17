package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestConcurrentReservations(t *testing.T) {
	// Reset inventory for test
	inventory = map[string]*Product{
		"TEST-SKU": {SKU: "TEST-SKU", Name: "Test Product", Stock: 10, Price: 99.99, Warehouse: "test"},
	}

	var wg sync.WaitGroup
	successCount := 0
	var countMu sync.Mutex

	// Attempt 15 concurrent reservations of 1 unit each
	// Only 10 should succeed
	for i := 0; i < 15; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			reqBody := map[string]interface{}{
				"sku":      "TEST-SKU",
				"quantity": 1,
			}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/api/reserve", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			reserveStockHandler(w, req)

			if w.Code == 200 {
				countMu.Lock()
				successCount++
				countMu.Unlock()
			}
		}()
	}

	wg.Wait()

	if successCount != 10 {
		t.Errorf("Expected exactly 10 successful reservations, got %d", successCount)
	}

	if inventory["TEST-SKU"].Stock != 0 {
		t.Errorf("Expected final stock to be 0, got %d", inventory["TEST-SKU"].Stock)
	}
}

func TestReserveExactStock(t *testing.T) {
	// Test that we can reserve exactly the remaining stock
	inventory = map[string]*Product{
		"TEST-SKU": {SKU: "TEST-SKU", Name: "Test Product", Stock: 5, Price: 99.99, Warehouse: "test"},
	}

	reqBody := map[string]interface{}{
		"sku":      "TEST-SKU",
		"quantity": 5,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/reserve", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	reserveStockHandler(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200 OK when reserving exact stock, got %d", w.Code)
	}

	if inventory["TEST-SKU"].Stock != 0 {
		t.Errorf("Expected stock to be 0 after reserving all, got %d", inventory["TEST-SKU"].Stock)
	}
}
