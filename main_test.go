package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestService() *WarehouseService {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	return NewWarehouseService(logger)
}

func TestProcess(t *testing.T) {
	svc := newTestService()

	req := ProcessRequest{
		ItemID:    "SKU-12345",
		Action:    "restock",
		Quantity:  100,
		Warehouse: "DC-TX-01",
	}

	resp, err := svc.Process(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %s", resp.Status)
	}
	if resp.Component != "warehouse" {
		t.Errorf("expected component warehouse, got %s", resp.Component)
	}
	if resp.ItemID != "SKU-12345" {
		t.Errorf("expected item_id SKU-12345, got %s", resp.ItemID)
	}
}

func TestProcessCancelledContext(t *testing.T) {
	svc := newTestService()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := svc.Process(ctx, ProcessRequest{ItemID: "SKU-99"})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestGetStats(t *testing.T) {
	svc := newTestService()

	for i := 0; i < 3; i++ {
		svc.Process(context.Background(), ProcessRequest{ItemID: "test"})
	}

	stats := svc.GetStats()
	if stats.Requests != 3 {
		t.Errorf("expected 3 requests, got %d", stats.Requests)
	}
	if stats.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", stats.Errors)
	}
}

func TestGetStatsEmpty(t *testing.T) {
	svc := newTestService()
	stats := svc.GetStats()

	if stats.Requests != 0 {
		t.Errorf("expected 0 requests, got %d", stats.Requests)
	}
	if stats.AvgLatencyMs != 0 {
		t.Errorf("expected 0 avg latency, got %f", stats.AvgLatencyMs)
	}
}

func TestHandleProcess(t *testing.T) {
	svc := newTestService()

	body, _ := json.Marshal(ProcessRequest{
		ItemID:   "SKU-99",
		Action:   "pick",
		Quantity: 5,
	})

	req := httptest.NewRequest(http.MethodPost, "/process", bytes.NewReader(body))
	w := httptest.NewRecorder()

	svc.HandleProcess(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp ProcessResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %s", resp.Status)
	}
}

func TestHandleProcessBadBody(t *testing.T) {
	svc := newTestService()

	req := httptest.NewRequest(http.MethodPost, "/process", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()

	svc.HandleProcess(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleProcessMethodNotAllowed(t *testing.T) {
	svc := newTestService()

	req := httptest.NewRequest(http.MethodGet, "/process", nil)
	w := httptest.NewRecorder()

	svc.HandleProcess(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleStats(t *testing.T) {
	svc := newTestService()

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	w := httptest.NewRecorder()

	svc.HandleStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var stats ServiceStats
	json.NewDecoder(w.Body).Decode(&stats)
	if stats.Requests != 0 {
		t.Errorf("expected 0 requests, got %d", stats.Requests)
	}
}

func TestHandleStatsMethodNotAllowed(t *testing.T) {
	svc := newTestService()

	req := httptest.NewRequest(http.MethodPost, "/stats", nil)
	w := httptest.NewRecorder()

	svc.HandleStats(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	HandleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
