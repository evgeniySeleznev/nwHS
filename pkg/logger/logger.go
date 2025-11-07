package logger

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config описывает параметры инициализации зап-логгера.
type Config struct {
	Level       string
	Environment string
	Encoding    string
}

// New создаёт zap.Logger с учётом окружения и уровня логирования.
func New(cfg Config) (*zap.Logger, error) {
	zapCfg := zap.NewProductionConfig()

	if strings.EqualFold(cfg.Environment, "local") || strings.EqualFold(cfg.Environment, "dev") {
		zapCfg = zap.NewDevelopmentConfig()
	}

	if cfg.Encoding != "" {
		zapCfg.Encoding = cfg.Encoding
	}

	if cfg.Level != "" {
		lvl := new(zapcore.Level)
		if err := lvl.Set(cfg.Level); err != nil {
			return nil, fmt.Errorf("logger: invalid level %q: %w", cfg.Level, err)
		}
		zapCfg.Level = zap.NewAtomicLevelAt(*lvl)
	}

	zapLogger, err := zapCfg.Build(zap.AddCaller(), zap.AddCallerSkip(1))
	if err != nil {
		return nil, fmt.Errorf("logger: build: %w", err)
	}

	return zapLogger, nil
}

// Must создаёт логгер и паникует при ошибке.
func Must(cfg Config) *zap.Logger {
	logger, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return logger
}
