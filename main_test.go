package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestGetStockHandler_Success(t *testing.T) {
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
