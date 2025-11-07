package app

import (
	"context"
	"fmt"
	"net"

	kafkaInfra "github.com/company/holo/services/customer-service/internal/infrastructure/kafka"
	repository "github.com/company/holo/services/customer-service/internal/infrastructure/repository"
	search "github.com/company/holo/services/customer-service/internal/infrastructure/search"
	grpciface "github.com/company/holo/services/customer-service/internal/interfaces/grpc"

	"github.com/company/holo/services/customer-service/internal/application/commands"
	"github.com/company/holo/services/customer-service/internal/application/queries"

	"github.com/company/holo/pkg/logger"
	"github.com/company/holo/pkg/metrics"
	"github.com/company/holo/pkg/tracing"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// App описывает корневой объект сервисного контейнера.
type App struct {
	cfg      Config
	log      *zap.Logger
	metrics  *metrics.Collector
	tracer   tracing.Provider
	pool     *pgxpool.Pool
	writer   *kafka.Writer
	indexer  *search.Indexer
	server   *grpciface.Transport
	listener net.Listener
	shutdown []func(context.Context) error
}

// Provider описывает зависимость для трейсинга (адаптер над OTEL провайдером).
type Provider interface {
	Shutdown(ctx context.Context) error
}

// New создаёт экземпляр приложения с переданными зависимостями.
func New(ctx context.Context, cfg Config, logCfg logger.Config, tracer tracing.Provider) (*App, error) {
	cfg.Defaults()

	zapLogger, err := logger.New(logCfg)
	if err != nil {
		return nil, fmt.Errorf("app: logger: %w", err)
	}

	collector := metrics.NewCollector()

	pool, err := newPostgresPool(ctx, cfg)
	if err != nil {
		return nil, err
	}

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  cfg.Kafka.Brokers,
		Topic:    cfg.Kafka.CustomerTopic,
		Balancer: &kafka.LeastBytes{},
	})

	osClient, err := opensearch.NewClient(opensearch.Config{Addresses: []string{cfg.Search.Endpoint}})
	if err != nil {
		return nil, fmt.Errorf("opensearch client: %w", err)
	}

	repo := repository.NewPostgresRepository(pool)
	indexer := search.NewIndexer(osClient, cfg.Search.Index)
	publisher := kafkaInfra.NewPublisher(writer, cfg.Kafka.CustomerTopic)

	registerHandler := commands.NewRegisterCustomerHandler(repo, indexer, publisher, zapLogger)
	getHandler := queries.NewGetCustomerHandler(repo)

	transport := grpciface.NewTransport(registerHandler, getHandler, zapLogger)

	address := fmt.Sprintf("%s:%d", cfg.GRPC.Host, cfg.GRPC.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", address, err)
	}

	app := &App{
		cfg:      cfg,
		log:      zapLogger,
		metrics:  collector,
		tracer:   tracer,
		pool:     pool,
		writer:   writer,
		indexer:  indexer,
		server:   transport,
		listener: listener,
	}

	app.shutdown = []func(context.Context) error{
		func(ctx context.Context) error {
			transport.Stop()
			return nil
		},
		func(ctx context.Context) error {
			listener.Close()
			return nil
		},
		func(ctx context.Context) error {
			pool.Close()
			return nil
		},
		func(ctx context.Context) error {
			return writer.Close()
		},
		func(ctx context.Context) error {
			return tracer.Shutdown(ctx)
		},
		func(ctx context.Context) error {
			return zapLogger.Sync()
		},
	}

	app.log.Info("customer service configured", zap.String("service", cfg.ServiceName))

	return app, nil
}

// Run запускает инфраструктурные слои. Здесь будет инициализация транспорта.
func (a *App) Run(ctx context.Context) error {
	a.log.Info("customer service started", zap.String("addr", a.listener.Addr().String()))

	errCh := make(chan error, 1)
	go func() {
		if err := a.server.Serve(a.listener); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

// Shutdown корректно останавливает зависимые ресурсы.
func (a *App) Shutdown(ctx context.Context) error {
	for _, fn := range a.shutdown {
		if err := fn(ctx); err != nil {
			a.log.Error("shutdown error", zap.Error(err))
		}
	}
	return nil
}

// Logger возвращает используемый zap.Logger.
func (a *App) Logger() *zap.Logger {
	return a.log
}

// Metrics возвращает коллекцию метрик.
func (a *App) Metrics() *metrics.Collector {
	return a.metrics
}

func newPostgresPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.Postgres.DSN)
	if err != nil {
		return nil, fmt.Errorf("pgx parse dsn: %w", err)
	}
	poolCfg.MaxConns = cfg.Postgres.MaxConns

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("pgx pool: %w", err)
	}

	return pool, nil
}
