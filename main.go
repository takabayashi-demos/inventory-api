package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	defaultProcessTimeout = 5 * time.Second
	httpReadTimeout       = 10 * time.Second
	httpWriteTimeout      = 10 * time.Second
	listenAddr            = ":8080"
)

// ProcessRequest represents an incoming warehouse processing request.
type ProcessRequest struct {
	ItemID    string `json:"item_id"`
	Action    string `json:"action"`
	Quantity  int    `json:"quantity"`
	Warehouse string `json:"warehouse"`
}

// ProcessResponse represents the result of a warehouse processing operation.
type ProcessResponse struct {
	Status    string `json:"status"`
	Component string `json:"component"`
	LatencyMs int64  `json:"latency_ms"`
	ItemID    string `json:"item_id,omitempty"`
}

// ErrorResponse represents an API error.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

// ServiceStats holds aggregated metrics for the service.
type ServiceStats struct {
	Requests     int64   `json:"requests"`
	Errors       int64   `json:"errors"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
}

// WarehouseService handles warehouse operations.
type WarehouseService struct {
	mu      sync.RWMutex
	cache   map[string]ProcessResponse
	logger  *slog.Logger
	metrics struct {
		Requests       int64
		Errors         int64
		TotalLatencyMs float64
	}
}

// NewWarehouseService creates a new service instance.
func NewWarehouseService(logger *slog.Logger) *WarehouseService {
	return &WarehouseService{
		cache:  make(map[string]ProcessResponse),
		logger: logger,
	}
}

// Process handles a warehouse request with timeout.
func (s *WarehouseService) Process(ctx context.Context, req ProcessRequest) (*ProcessResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultProcessTimeout)
	defer cancel()

	start := time.Now()
	s.recordRequest()

	select {
	case <-ctx.Done():
		s.recordError()
		s.logger.ErrorContext(ctx, "warehouse processing timed out",
			"item_id", req.ItemID,
			"action", req.Action,
		)
		return nil, fmt.Errorf("warehouse processing timed out: %w", ctx.Err())
	default:
		latency := time.Since(start).Milliseconds()
		resp := &ProcessResponse{
			Status:    "ok",
			Component: "warehouse",
			LatencyMs: latency,
			ItemID:    req.ItemID,
		}

		s.recordLatency(float64(latency))
		s.logger.InfoContext(ctx, "processed warehouse request",
			"item_id", req.ItemID,
			"action", req.Action,
			"latency_ms", latency,
		)

		return resp, nil
	}
}

func (s *WarehouseService) recordRequest() {
	s.mu.Lock()
	s.metrics.Requests++
	s.mu.Unlock()
}

func (s *WarehouseService) recordError() {
	s.mu.Lock()
	s.metrics.Errors++
	s.mu.Unlock()
}

func (s *WarehouseService) recordLatency(ms float64) {
	s.mu.Lock()
	s.metrics.TotalLatencyMs += ms
	s.mu.Unlock()
}

// GetStats returns service metrics.
func (s *WarehouseService) GetStats() ServiceStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var avgLatency float64
	if s.metrics.Requests > 0 {
		avgLatency = s.metrics.TotalLatencyMs / float64(s.metrics.Requests)
	}

	return ServiceStats{
		Requests:     s.metrics.Requests,
		Errors:       s.metrics.Errors,
		AvgLatencyMs: avgLatency,
	}
}

// HandleProcess is the HTTP handler for warehouse processing requests.
func (s *WarehouseService) HandleProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error: "method not allowed",
			Code:  http.StatusMethodNotAllowed,
		})
		return
	}

	var req ProcessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "invalid request body",
			Code:  http.StatusBadRequest,
		})
		return
	}

	resp, err := s.Process(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusGatewayTimeout, ErrorResponse{
			Error: err.Error(),
			Code:  http.StatusGatewayTimeout,
		})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// HandleStats is the HTTP handler for service metrics.
func (s *WarehouseService) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error: "method not allowed",
			Code:  http.StatusMethodNotAllowed,
		})
		return
	}

	writeJSON(w, http.StatusOK, s.GetStats())
}

// HandleHealth is a basic liveness probe handler.
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	svc := NewWarehouseService(logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/process", svc.HandleProcess)
	mux.HandleFunc("/stats", svc.HandleStats)
	mux.HandleFunc("/healthz", HandleHealth)

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  httpReadTimeout,
		WriteTimeout: httpWriteTimeout,
	}

	logger.Info("starting inventory-api", "addr", listenAddr)
	if err := server.ListenAndServe(); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}
