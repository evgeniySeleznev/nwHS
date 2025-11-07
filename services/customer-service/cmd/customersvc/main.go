package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/evgeniySeleznev/nwHS/pkg/config"
	"github.com/evgeniySeleznev/nwHS/pkg/logger"
	"github.com/evgeniySeleznev/nwHS/services/customer-service/internal/app"
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

	if env := os.Getenv("APP_ENV"); env != "" {
		cfg.Observability.Sentry.Environment = env
	}
	if release := os.Getenv("APP_RELEASE"); release != "" {
		cfg.Observability.Sentry.Release = release
	}

	application, err := app.New(ctx, cfg, logger.Config{
		Level:       os.Getenv("LOG_LEVEL"),
		Environment: os.Getenv("APP_ENV"),
		Encoding:    "json",
	})
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
