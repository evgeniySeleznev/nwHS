package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Collector агрегирует пользовательские метрики и предоставляет HTTP-обработчик.
type Collector struct {
	registry *prometheus.Registry
	latency  *prometheus.HistogramVec
	counter  *prometheus.CounterVec
}

// Option конфигурирует сборщик метрик.
type Option func(*Collector)

// WithRegistry переопределяет prometheus.Registry.
func WithRegistry(registry *prometheus.Registry) Option {
	return func(c *Collector) {
		if registry != nil {
			c.registry = registry
		}
	}
}

// NewCollector создаёт новый экземпляр Collector.
func NewCollector(opts ...Option) *Collector {
	collector := &Collector{
		registry: prometheus.NewRegistry(),
	}

	for _, opt := range opts {
		opt(collector)
	}

	collector.latency = promauto.With(collector.registry).NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "holo",
		Name:      "request_latency_seconds",
		Help:      "Histogram of request latencies.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 15),
	}, []string{"service", "endpoint", "status"})

	collector.counter = promauto.With(collector.registry).NewCounterVec(prometheus.CounterOpts{
		Namespace: "holo",
		Name:      "requests_total",
		Help:      "Total number of requests by service and endpoint.",
	}, []string{"service", "endpoint", "status"})

	return collector
}

// TrackDuration записывает latency и счётчик.
func (c *Collector) TrackDuration(service, endpoint, status string, started time.Time) {
	c.latency.WithLabelValues(service, endpoint, status).Observe(time.Since(started).Seconds())
	c.counter.WithLabelValues(service, endpoint, status).Inc()
}

// Handler возвращает http.Handler для /metrics.
func (c *Collector) Handler() http.Handler {
	return promhttp.HandlerFor(c.registry, promhttp.HandlerOpts{})
}
