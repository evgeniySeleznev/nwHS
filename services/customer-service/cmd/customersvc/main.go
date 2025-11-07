package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/company/holo/pkg/config"
	"github.com/company/holo/pkg/logger"
	"github.com/company/holo/pkg/tracing"
	"github.com/company/holo/services/customer-service/internal/app"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var cfg app.Config
	loader := config.New(config.WithConfigPaths("./configs"), config.WithPrefix("CUSTOMER"))
	if err := loader.Load(&cfg); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	tracer, err := tracing.InitProvider(ctx, tracing.Config{
		Endpoint:    os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		Insecure:    os.Getenv("OTEL_EXPORTER_OTLP_INSECURE") == "true",
		Service:     cfg.ServiceName,
		Environment: os.Getenv("APP_ENV"),
	})
	if err != nil {
		log.Fatalf("failed to init tracing: %v", err)
	}

	application, err := app.New(ctx, cfg, logger.Config{
		Level:       os.Getenv("LOG_LEVEL"),
		Environment: os.Getenv("APP_ENV"),
		Encoding:    "json",
	}, tracer)
	if err != nil {
		log.Fatalf("failed to init app: %v", err)
	}

	if err := application.Run(ctx); err != nil && err != context.Canceled {
		application.Logger().Error("runtime error", zap.Error(err))
		os.Exit(1)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := application.Shutdown(shutdownCtx); err != nil {
		application.Logger().Error("graceful shutdown failed", zap.Error(err))
	}
}
