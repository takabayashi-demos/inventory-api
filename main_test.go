package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestReserveStockConcurrent(t *testing.T) {
	// Reset inventory to known state
	inventory["SKU-TEST"] = &Product{
		SKU:   "SKU-TEST",
		Name:  "Test Product",
		Stock: 100,
		Price: 99.99,
	}

	var wg sync.WaitGroup
	successCount := 0
	var countMu sync.Mutex

	// Launch 20 concurrent reservations of 10 units each
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			body := bytes.NewBufferString(`{"sku":"SKU-TEST","quantity":10}`)
			req := httptest.NewRequest("POST", "/reserve", body)
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

	// With 100 stock and 10 units per reservation, exactly 10 should succeed
	if successCount != 10 {
		t.Errorf("Expected 10 successful reservations, got %d", successCount)
	}

	if inventory["SKU-TEST"].Stock != 0 {
		t.Errorf("Expected final stock to be 0, got %d", inventory["SKU-TEST"].Stock)
	}
}

func TestReserveExactStock(t *testing.T) {
	// Test the boundary condition: reserving exact available stock
	inventory["SKU-BOUNDARY"] = &Product{
		SKU:   "SKU-BOUNDARY",
		Name:  "Boundary Test",
		Stock: 50,
		Price: 49.99,
	}

	body := bytes.NewBufferString(`{"sku":"SKU-BOUNDARY","quantity":50}`)
	req := httptest.NewRequest("POST", "/reserve", body)
	w := httptest.NewRecorder()

	reserveStockHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 when reserving exact stock, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["remaining_stock"].(float64) != 0 {
		t.Errorf("Expected remaining stock to be 0, got %v", resp["remaining_stock"])
	}
}
