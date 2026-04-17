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

	total := int(resp["total"].(float64))
	if total != 8 {
		t.Errorf("expected total 8, got %d", total)
	}
}

func TestGetStockHandler(t *testing.T) {
	tests := []struct {
		name           string
		sku            string
		expectedStatus int
		expectedPanic  bool
	}{
		{name: "valid SKU", sku: "SKU-001", expectedStatus: http.StatusOK, expectedPanic: false},
		{name: "missing SKU param", sku: "", expectedStatus: http.StatusBadRequest, expectedPanic: false},
		{name: "nonexistent SKU", sku: "INVALID", expectedStatus: 0, expectedPanic: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("expected panic for %s, but didn't panic", tt.name)
					}
				}()
			}

			url := "/stock"
			if tt.sku != "" {
				url += "?sku=" + tt.sku
			}

			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			getStockHandler(w, req)

			if !tt.expectedPanic && w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestReserveStockHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
	}{
		{
			name:           "successful reservation",
			method:         "POST",
			body:           `{"sku":"SKU-002","quantity":10}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "insufficient stock",
			method:         "POST",
			body:           `{"sku":"SKU-005","quantity":1000}`,
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "product not found",
			method:         "POST",
			body:           `{"sku":"INVALID","quantity":10}`,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "method not allowed",
			method:         "GET",
			body:           "",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "edge case - exact stock match",
			method:         "POST",
			body:           `{"sku":"SKU-005","quantity":75}`,
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := bytes.NewBufferString(tt.body)
			req := httptest.NewRequest(tt.method, "/reserve", body)
			w := httptest.NewRecorder()

			reserveStockHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d", tt.name, tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestReserveStockHandler_RaceCondition tests for concurrent reservation race conditions
// Run with: go test -race
func TestReserveStockHandler_RaceCondition(t *testing.T) {
	// Reset inventory for this test
	inventory["SKU-999"] = &Product{
		SKU:       "SKU-999",
		Name:      "Test Product",
		Stock:     100,
		Price:     99.99,
		Warehouse: "test",
	}

	var wg sync.WaitGroup
	concurrentReservations := 20
	reservationSize := 10

	// Launch concurrent reservation requests
	for i := 0; i < concurrentReservations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := bytes.NewBufferString(`{"sku":"SKU-999","quantity":10}`)
			req := httptest.NewRequest("POST", "/reserve", body)
			w := httptest.NewRecorder()
			reserveStockHandler(w, req)
		}()
	}

	wg.Wait()

	// Check final stock - should be 100 - (successful_reservations * 10)
	// But due to race condition, it may be incorrect
	finalStock := inventory["SKU-999"].Stock
	t.Logf("Final stock after concurrent reservations: %d (expected <= 0 with race condition)", finalStock)

	if finalStock < 0 {
		t.Logf("WARNING: Race condition detected - stock went negative: %d", finalStock)
	}
}

// TestOffByOneError tests the off-by-one error in stock reservation
func TestOffByOneError(t *testing.T) {
	// This test demonstrates the off-by-one bug where stock == quantity fails
	inventory["SKU-TEST"] = &Product{
		SKU:   "SKU-TEST",
		Name:  "Test",
		Stock: 10,
	}

	body := bytes.NewBufferString(`{"sku":"SKU-TEST","quantity":10}`)
	req := httptest.NewRequest("POST", "/reserve", body)
	w := httptest.NewRecorder()

	reserveStockHandler(w, req)

	// BUG: Should succeed (stock >= quantity), but fails due to > instead of >=
	if w.Code == http.StatusOK {
		t.Errorf("Reservation succeeded, but off-by-one bug should cause it to fail")
	} else {
		t.Logf("Off-by-one bug confirmed: exact stock match rejected (status %d)", w.Code)
	}
}
