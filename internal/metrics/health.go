package metrics

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// HealthChecker is an interface for components that can report health
type HealthChecker interface {
	// Check performs a health check and returns an error if unhealthy
	Check(ctx context.Context) error
}

// HealthHandler handles health check requests
type HealthHandler struct {
	checkers map[string]HealthChecker
}

// NewHealthHandler creates a new health handler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{
		checkers: make(map[string]HealthChecker),
	}
}

// RegisterChecker registers a health checker
func (h *HealthHandler) RegisterChecker(name string, checker HealthChecker) {
	h.checkers[name] = checker
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status  string            `json:"status"`
	Checks  map[string]string `json:"checks,omitempty"`
	Message string            `json:"message,omitempty"`
}

// ServeHTTP implements the /healthz endpoint (always returns OK if process is running)
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status: "ok",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ReadyHandler handles readiness check requests
type ReadyHandler struct {
	checkers map[string]HealthChecker
	timeout  time.Duration
}

// NewReadyHandler creates a new readiness handler
func NewReadyHandler(timeout time.Duration) *ReadyHandler {
	return &ReadyHandler{
		checkers: make(map[string]HealthChecker),
		timeout:  timeout,
	}
}

// RegisterChecker registers a readiness checker
func (r *ReadyHandler) RegisterChecker(name string, checker HealthChecker) {
	r.checkers[name] = checker
}

// ServeHTTP implements the /readyz endpoint (checks all registered checkers)
func (r *ReadyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), r.timeout)
	defer cancel()

	checks := make(map[string]string)
	allHealthy := true

	for name, checker := range r.checkers {
		if err := checker.Check(ctx); err != nil {
			checks[name] = "unhealthy: " + err.Error()
			allHealthy = false
		} else {
			checks[name] = "ok"
		}
	}

	response := HealthResponse{
		Checks: checks,
	}

	w.Header().Set("Content-Type", "application/json")

	if allHealthy {
		response.Status = "ready"
		w.WriteHeader(http.StatusOK)
	} else {
		response.Status = "not ready"
		response.Message = "one or more checks failed"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}
