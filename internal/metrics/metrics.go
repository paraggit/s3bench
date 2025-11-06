package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// Operation counters
	OpsTotal  *prometheus.CounterVec
	OpLatency *prometheus.HistogramVec

	// Data transfer
	BytesWritten prometheus.Counter
	BytesRead    prometheus.Counter

	// Verification
	VerifyFailures prometheus.Counter
	VerifyTotal    prometheus.Counter

	// Retries
	Retries *prometheus.CounterVec

	// Workers
	ActiveWorkers prometheus.Gauge

	// Rate limiter
	RateLimiterTokens prometheus.Gauge

	// Circuit breaker
	CircuitBreakerOpen prometheus.Gauge

	registry *prometheus.Registry
}

// NewMetrics creates and registers all metrics
func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()

	m := &Metrics{
		OpsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "s3_ops_total",
				Help: "Total number of S3 operations by type and status",
			},
			[]string{"op", "status"},
		),

		OpLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "s3_op_latency_seconds",
				Help: "Latency of S3 operations in seconds",
				Buckets: []float64{
					0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0,
				},
			},
			[]string{"op"},
		),

		BytesWritten: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "s3_bytes_written_total",
				Help: "Total bytes written to S3",
			},
		),

		BytesRead: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "s3_bytes_read_total",
				Help: "Total bytes read from S3",
			},
		),

		VerifyFailures: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "s3_verify_failures_total",
				Help: "Total number of verification failures",
			},
		),

		VerifyTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "s3_verify_total",
				Help: "Total number of verifications attempted",
			},
		),

		Retries: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "s3_retries_total",
				Help: "Total number of retries by operation",
			},
			[]string{"op"},
		),

		ActiveWorkers: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "s3_active_workers",
				Help: "Number of currently active workers",
			},
		),

		RateLimiterTokens: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "s3_rate_limiter_tokens",
				Help: "Current number of available rate limiter tokens",
			},
		),

		CircuitBreakerOpen: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "s3_circuit_breaker_open",
				Help: "Circuit breaker state (1 = open, 0 = closed)",
			},
		),

		registry: reg,
	}

	// Register all metrics
	reg.MustRegister(
		m.OpsTotal,
		m.OpLatency,
		m.BytesWritten,
		m.BytesRead,
		m.VerifyFailures,
		m.VerifyTotal,
		m.Retries,
		m.ActiveWorkers,
		m.RateLimiterTokens,
		m.CircuitBreakerOpen,
	)

	return m
}

// Handler returns an HTTP handler for Prometheus metrics
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// RecordOp records an operation with its status and latency
func (m *Metrics) RecordOp(op string, status string, duration time.Duration) {
	m.OpsTotal.WithLabelValues(op, status).Inc()
	m.OpLatency.WithLabelValues(op).Observe(duration.Seconds())
}

// RecordBytesWritten records bytes written
func (m *Metrics) RecordBytesWritten(bytes int64) {
	m.BytesWritten.Add(float64(bytes))
}

// RecordBytesRead records bytes read
func (m *Metrics) RecordBytesRead(bytes int64) {
	m.BytesRead.Add(float64(bytes))
}

// RecordVerifyFailure records a verification failure
func (m *Metrics) RecordVerifyFailure() {
	m.VerifyFailures.Inc()
	m.VerifyTotal.Inc()
}

// RecordVerifySuccess records a successful verification
func (m *Metrics) RecordVerifySuccess() {
	m.VerifyTotal.Inc()
}

// RecordRetry records a retry
func (m *Metrics) RecordRetry(op string) {
	m.Retries.WithLabelValues(op).Inc()
}

// SetActiveWorkers sets the number of active workers
func (m *Metrics) SetActiveWorkers(count int) {
	m.ActiveWorkers.Set(float64(count))
}

// SetRateLimiterTokens sets the rate limiter token count
func (m *Metrics) SetRateLimiterTokens(tokens float64) {
	m.RateLimiterTokens.Set(tokens)
}

// SetCircuitBreakerOpen sets the circuit breaker state
func (m *Metrics) SetCircuitBreakerOpen(open bool) {
	if open {
		m.CircuitBreakerOpen.Set(1)
	} else {
		m.CircuitBreakerOpen.Set(0)
	}
}

// OpStatus represents operation status
type OpStatus string

const (
	StatusSuccess OpStatus = "success"
	StatusError   OpStatus = "error"
	StatusTimeout OpStatus = "timeout"
	StatusRetry   OpStatus = "retry"
)

// OpType represents operation type
type OpType string

const (
	OpPut    OpType = "put"
	OpGet    OpType = "get"
	OpDelete OpType = "delete"
	OpCopy   OpType = "copy"
	OpList   OpType = "list"
	OpHead   OpType = "head"
)
