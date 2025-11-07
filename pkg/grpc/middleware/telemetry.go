package middleware

import (
	"context"
	"time"

	"github.com/evgeniySeleznev/nwHS/pkg/metrics"
	sentryobs "github.com/evgeniySeleznev/nwHS/pkg/observability/sentry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryTelemetryInterceptor добавляет метрики, ошибки и трассировку к gRPC обработчикам.
func UnaryTelemetryInterceptor(service string, collector *metrics.Collector, sentryClient *sentryobs.Client, logger *zap.Logger) grpc.UnaryServerInterceptor {
	lg := logger
	if lg == nil {
		lg = zap.NewNop()
	}

	tracer := otel.Tracer(service)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		started := time.Now()

		ctx, span := tracer.Start(ctx, info.FullMethod)
		span.SetAttributes(
			semconv.RPCSystemKey.String("grpc"),
			semconv.RPCServiceKey.String(service),
			semconv.RPCMethodKey.String(info.FullMethod),
		)

		resp, err := handler(ctx, req)

		code := status.Code(err)
		span.SetAttributes(attribute.String("rpc.status_code", code.String()))

		if err != nil {
			lg.Error("grpc handler error", zap.String("method", info.FullMethod), zap.String("status", code.String()), zap.Error(err))
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			if sentryClient != nil && sentryClient.Enabled() {
				sentryClient.CaptureError(err)
			}
		} else {
			span.SetStatus(codes.Ok, "")
		}

		if collector != nil {
			collector.TrackDuration(service, info.FullMethod, code.String(), started)
		}

		span.End()
		return resp, err
	}
}
