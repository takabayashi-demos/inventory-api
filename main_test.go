package main

import (
	"context"
	"sync"
	"testing"
)

func TestNewWarehouseService(t *testing.T) {
	svc := NewWarehouseService()
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.cache == nil {
		t.Fatal("expected initialized cache map")
	}
}

func TestProcess_ReturnsResult(t *testing.T) {
	svc := NewWarehouseService()

	req := map[string]interface{}{
		"sku":       "WMT-12345",
		"warehouse": "DC-042",
		"quantity":  100,
	}

	result, err := svc.Process(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", result["status"])
	}
	if result["component"] != "warehouse" {
		t.Errorf("expected component=warehouse, got %v", result["component"])
	}
	if _, ok := result["latency_ms"]; !ok {
		t.Error("expected latency_ms in result")
	}
}

func TestProcess_IncrementsRequestCount(t *testing.T) {
	svc := NewWarehouseService()

	for i := 0; i < 5; i++ {
		_, err := svc.Process(context.Background(), map[string]interface{}{"sku": "WMT-00001"})
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
	}

	stats := svc.GetStats()
	if got := stats["requests"].(int64); got != 5 {
		t.Errorf("expected 5 requests, got %d", got)
	}
}

func TestGetStats_ZeroState(t *testing.T) {
	svc := NewWarehouseService()
	stats := svc.GetStats()

	if stats["requests"].(int64) != 0 {
		t.Errorf("expected 0 requests, got %v", stats["requests"])
	}
	if stats["errors"].(int64) != 0 {
		t.Errorf("expected 0 errors, got %v", stats["errors"])
	}
	if stats["avg_latency_ms"].(float64) != 0 {
		t.Errorf("expected 0 avg latency, got %v", stats["avg_latency_ms"])
	}
}

func TestProcess_ConcurrentSafety(t *testing.T) {
	svc := NewWarehouseService()
	ctx := context.Background()

	var wg sync.WaitGroup
	const goroutines = 50

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = svc.Process(ctx, map[string]interface{}{"sku": "WMT-CONCURRENT"})
		}()
	}
	wg.Wait()

	stats := svc.GetStats()
	if got := stats["requests"].(int64); got != goroutines {
		t.Errorf("expected %d requests, got %d", goroutines, got)
	}
}
