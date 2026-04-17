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
	// Reset inventory for test
	inventory = map[string]*Product{
		"SKU-TEST": {SKU: "SKU-TEST", Name: "Test Product", Stock: 10, Price: 99.99, Warehouse: "test"},
	}

	// Attempt 15 concurrent reservations of 1 unit each
	// Only 10 should succeed (initial stock = 10)
	var wg sync.WaitGroup
	successes := 0
	failures := 0
	var resultMu sync.Mutex

	for i := 0; i < 15; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			body := bytes.NewBuffer([]byte(`{"sku":"SKU-TEST","quantity":1}`))
			req := httptest.NewRequest("POST", "/reserve", body)
			w := httptest.NewRecorder()

			reserveStockHandler(w, req)

			resultMu.Lock()
			if w.Code == 200 {
				successes++
			} else if w.Code == 409 {
				failures++
			}
			resultMu.Unlock()
		}()
	}

	wg.Wait()

	if successes != 10 {
		t.Errorf("Expected exactly 10 successful reservations, got %d", successes)
	}
	if failures != 5 {
		t.Errorf("Expected exactly 5 failed reservations, got %d", failures)
	}
	if inventory["SKU-TEST"].Stock != 0 {
		t.Errorf("Expected final stock to be 0, got %d", inventory["SKU-TEST"].Stock)
	}
}

func TestReserveStock_ExactAmount(t *testing.T) {
	// Reset inventory for test
	inventory = map[string]*Product{
		"SKU-EXACT": {SKU: "SKU-EXACT", Name: "Exact Test", Stock: 5, Price: 49.99, Warehouse: "test"},
	}

	body := bytes.NewBuffer([]byte(`{"sku":"SKU-EXACT","quantity":5}`))
	req := httptest.NewRequest("POST", "/reserve", body)
	w := httptest.NewRecorder()

	reserveStockHandler(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200 when reserving exact stock amount, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["remaining_stock"].(float64) != 0 {
		t.Errorf("Expected remaining stock to be 0, got %v", resp["remaining_stock"])
	}
}

func TestReserveStock_InsufficientStock(t *testing.T) {
	inventory = map[string]*Product{
		"SKU-LOW": {SKU: "SKU-LOW", Name: "Low Stock", Stock: 3, Price: 29.99, Warehouse: "test"},
	}

	body := bytes.NewBuffer([]byte(`{"sku":"SKU-LOW","quantity":5}`))
	req := httptest.NewRequest("POST", "/reserve", body)
	w := httptest.NewRecorder()

	reserveStockHandler(w, req)

	if w.Code != 409 {
		t.Errorf("Expected status 409 for insufficient stock, got %d", w.Code)
	}

	if inventory["SKU-LOW"].Stock != 3 {
		t.Errorf("Stock should not change on failed reservation, got %d", inventory["SKU-LOW"].Stock)
	}
}
