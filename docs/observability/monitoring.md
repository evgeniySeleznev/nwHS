# Observability Blueprint

This system is designed to keep MTTR below 5 minutes and sustain 99.97% SLA under
microservice growth. Observability covers metrics, logs, tracing, and errors.

## Stack

- **Prometheus** collects platform and business metrics. Every service exposes `/metrics`
  built on the shared `pkg/metrics` collector.
- **Grafana** visualises dashboards. See
  `docs/observability/grafana/customer-overview.json` for the baseline board that tracks gRPC
  latency, Kafka lag, PostgreSQL saturation, and error rates.
- **Jaeger** (OTLP) provides distributed traces via `pkg/tracing`.
- **Sentry** captures exceptions through `pkg/observability/sentry`. The shared gRPC interceptor
  reports every failed RPC.
- **Mongo-backed Kafka DLQ** captures failed publications (see `services/customer-service/internal/infrastructure/mongo`).
  Метаданные и payload события пишутся в коллекцию DLQ, что позволяет операторам вручную
  переигрывать события без потери данных.

## Service configuration

```yaml
service_name: customer-service
observability:
  metrics:
    addr: ":9100"
  tracing:
    endpoint: "jaeger-collector.observability:4317"
    insecure: true
  kafka:
    dlq:
      mongo_uri: "mongodb://mongodb.observability:27017"
      database: "holo_dlq"
      collection: "customer_events"
  sentry:
    dsn: "https://public@example.ingest.sentry.io/1"
    environment: "production"
    release: "customer-service@1.0.0"
    sample_rate: 1.0
    traces_sample_rate: 1.0
```

### Environment variables

- `CUSTOMER_OBSERVABILITY_TRACING_ENDPOINT` — OTLP collector endpoint (Jaeger/Tempo/OTEL).
- `CUSTOMER_OBSERVABILITY_SENTRY_DSN` — project DSN used by Sentry SDK.
- `CUSTOMER_OBSERVABILITY_METRICS_ADDR` — Prometheus scrape address (`:9100` default).
- `APP_ENV`, `APP_RELEASE` — propagated into logs, traces, and Sentry scopes.

## Alerting guidelines

- Prometheus Alertmanager: alert on `holo_request_latency_seconds` p95 > SLA, gRPC error rate,
  Kafka consumer lag, PostgreSQL connection saturation.
- Sentry: alert rules for error frequency regressions and high-severity issues.
- Grafana: dashboards include annotations from Sentry and Jaeger to cut diagnosis time.

Following this setup keeps troubleshooting contextual and fast even as service count grows.

