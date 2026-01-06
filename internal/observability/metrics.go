package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the service.
type Metrics struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	ActiveRequests  prometheus.Gauge
	ClaudeDuration  prometheus.Histogram
	ErrorsTotal     *prometheus.CounterVec
}

var (
	// DefaultMetrics is the default metrics instance.
	DefaultMetrics *Metrics
)

// InitMetrics initializes and registers all Prometheus metrics.
func InitMetrics() *Metrics {
	metrics := &Metrics{
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "chat_completions_requests_total",
				Help: "Total number of chat completion requests",
			},
			[]string{"status", "stream"},
		),
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "chat_completions_duration_seconds",
				Help:    "Duration of chat completion requests in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"stream"},
		),
		ActiveRequests: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "chat_completions_active_requests",
				Help: "Number of active chat completion requests",
			},
		),
		ClaudeDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "claude_cli_duration_seconds",
				Help:    "Duration of Claude CLI executions in seconds",
				Buckets: prometheus.DefBuckets,
			},
		),
		ErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "chat_completions_errors_total",
				Help: "Total number of errors in chat completions",
			},
			[]string{"type"},
		),
	}

	DefaultMetrics = metrics
	return metrics
}

// RecordRequest records a completed request.
func (m *Metrics) RecordRequest(status string, stream bool, duration float64) {
	streamLabel := "false"
	if stream {
		streamLabel = "true"
	}
	m.RequestsTotal.WithLabelValues(status, streamLabel).Inc()
	m.RequestDuration.WithLabelValues(streamLabel).Observe(duration)
}

// RecordClaudeDuration records Claude CLI execution duration.
func (m *Metrics) RecordClaudeDuration(duration float64) {
	m.ClaudeDuration.Observe(duration)
}

// RecordError records an error.
func (m *Metrics) RecordError(errorType string) {
	m.ErrorsTotal.WithLabelValues(errorType).Inc()
}

// IncrementActive increments the active requests gauge.
func (m *Metrics) IncrementActive() {
	m.ActiveRequests.Inc()
}

// DecrementActive decrements the active requests gauge.
func (m *Metrics) DecrementActive() {
	m.ActiveRequests.Dec()
}
