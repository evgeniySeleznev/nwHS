package metrics

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestCollectorTrackDuration(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewCollector(WithRegistry(registry))

	start := time.Now().Add(-10 * time.Millisecond)
	collector.TrackDuration("customer", "RegisterCustomer", "200", start)

	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	if len(metrics) == 0 {
		t.Fatalf("expected metrics to be collected")
	}
}

func TestCollectorHandler(t *testing.T) {
	collector := NewCollector()
	collector.TrackDuration("customer", "RegisterCustomer", "200", time.Now())
	req := httptest.NewRequest("GET", "/metrics", nil)
	recorder := httptest.NewRecorder()

	collector.Handler().ServeHTTP(recorder, req)

	if recorder.Code != 200 {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if recorder.Body.Len() == 0 {
		t.Fatalf("expected body to contain metrics")
	}
}
