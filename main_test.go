package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestConcurrentStockReservation(t *testing.T) {
	// Reset inventory for test
	inventory = map[string]*Product{
		"SKU-TEST": {SKU: "SKU-TEST", Name: "Test Product", Stock: 100, Price: 99.99, Warehouse: "us-east-1"},
	}

	var wg sync.WaitGroup
	successCount := 0
	var countMu sync.Mutex

	// Simulate 100 concurrent requests each trying to reserve 1 unit
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			reqBody := map[string]interface{}{
				"sku":      "SKU-TEST",
				"quantity": 1,
			}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest("POST", "/reserve", bytes.NewReader(body))
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

	// All 100 requests should succeed
	if successCount != 100 {
		t.Errorf("Expected 100 successful reservations, got %d", successCount)
	}

	// Final stock should be 0
	if inventory["SKU-TEST"].Stock != 0 {
		t.Errorf("Expected stock to be 0, got %d", inventory["SKU-TEST"].Stock)
	}
}

func TestReserveExactStock(t *testing.T) {
	inventory = map[string]*Product{
		"SKU-EXACT": {SKU: "SKU-EXACT", Name: "Exact Stock Product", Stock: 5, Price: 49.99, Warehouse: "us-west-2"},
	}

	reqBody := map[string]interface{}{
		"sku":      "SKU-EXACT",
		"quantity": 5,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/reserve", bytes.NewReader(body))
	w := httptest.NewRecorder()

	reserveStockHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if inventory["SKU-EXACT"].Stock != 0 {
		t.Errorf("Expected stock to be 0, got %d", inventory["SKU-EXACT"].Stock)
	}
}
