package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// QueryService handles query operations.
type QueryService struct {
	mu      sync.RWMutex
	cache   map[string]interface{}
	metrics struct {
		Requests  int64
		Errors    int64
		LatencyMs float64
	}
}

// NewQueryService creates a new service instance.
func NewQueryService() *QueryService {
	return &QueryService{
		cache: make(map[string]interface{}),
	}
}

// Process handles a query request with timeout.
func (s *QueryService) Process(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	start := time.Now()
	s.mu.Lock()
	s.metrics.Requests++
	s.mu.Unlock()

	select {
	case <-ctx.Done():
		s.mu.Lock()
		s.metrics.Errors++
		s.mu.Unlock()
		return nil, fmt.Errorf("query processing timed out")
	default:
		// Process the request
		result := map[string]interface{}{
			"status":     "ok",
			"component":  "query",
			"latency_ms": time.Since(start).Milliseconds(),
		}

		s.mu.Lock()
		s.metrics.LatencyMs += float64(time.Since(start).Milliseconds())
		s.mu.Unlock()

		return result, nil
	}
}

// GetStats returns service metrics.
func (s *QueryService) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	avgLatency := float64(0)
	if s.metrics.Requests > 0 {
		avgLatency = s.metrics.LatencyMs / float64(s.metrics.Requests)
	}

	return map[string]interface{}{
		"requests":       s.metrics.Requests,
		"errors":         s.metrics.Errors,
		"avg_latency_ms": avgLatency,
	}
}
