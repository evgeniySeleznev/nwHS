package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/evgeniySeleznev/nwHS/services/customer-service/internal/application/commands"
	"github.com/evgeniySeleznev/nwHS/services/customer-service/internal/application/queries"
	kafkaInfra "github.com/evgeniySeleznev/nwHS/services/customer-service/internal/infrastructure/kafka"
	mongodlq "github.com/evgeniySeleznev/nwHS/services/customer-service/internal/infrastructure/mongo"
	repository "github.com/evgeniySeleznev/nwHS/services/customer-service/internal/infrastructure/repository"
	search "github.com/evgeniySeleznev/nwHS/services/customer-service/internal/infrastructure/search"
	grpciface "github.com/evgeniySeleznev/nwHS/services/customer-service/internal/interfaces/grpc"

	grpcmiddleware "github.com/evgeniySeleznev/nwHS/pkg/grpc/middleware"
	"github.com/evgeniySeleznev/nwHS/pkg/logger"
	"github.com/evgeniySeleznev/nwHS/pkg/metrics"
	sentryobs "github.com/evgeniySeleznev/nwHS/pkg/observability/sentry"
	"github.com/evgeniySeleznev/nwHS/pkg/tracing"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// App описывает корневой сервисный контейнер.
type App struct {
	cfg        Config
	log        *zap.Logger
	metrics    *metrics.Collector
	tracer     tracing.Provider
	sentry     *sentryobs.Client
	pool       *pgxpool.Pool
	writer     *kafka.Writer
	indexer    *search.Indexer
	server     *grpciface.Transport
	metricsSrv *http.Server
	listener   net.Listener
	mongo      *mongo.Client
	shutdown   []func(context.Context) error
}

// New создаёт экземпляр приложения с переданными зависимостями.
func New(ctx context.Context, cfg Config, logCfg logger.Config) (*App, error) {
	cfg.Defaults()

	zapLogger, err := logger.New(logCfg)
	if err != nil {
		return nil, fmt.Errorf("app: logger: %w", err)
	}

	collector := metrics.NewCollector()

	tracer, err := tracing.InitProvider(ctx, tracing.Config{
		Endpoint:    cfg.Observability.Tracing.Endpoint,
		Insecure:    cfg.Observability.Tracing.Insecure,
		Service:     cfg.ServiceName,
		Environment: cfg.Observability.Sentry.Environment,
	})
	if err != nil {
		return nil, fmt.Errorf("app: tracing: %w", err)
	}

	sentryClient, err := sentryobs.Init(sentryobs.Config{
		DSN:              cfg.Observability.Sentry.DSN,
		Environment:      cfg.Observability.Sentry.Environment,
		Release:          cfg.Observability.Sentry.Release,
		SampleRate:       cfg.Observability.Sentry.SampleRate,
		TracesSampleRate: cfg.Observability.Sentry.TracesSampleRate,
	})
	if err != nil {
		return nil, fmt.Errorf("app: sentry init: %w", err)
	}

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

	var (
		mongoClient *mongo.Client
		dlqRepo     *mongodlq.DeadLetterRepository
	)

	if cfg.Kafka.DLQ.MongoURI != "" {
		mongoClient, err = newMongoClient(ctx, cfg.Kafka.DLQ.MongoURI)
		if err != nil {
			return nil, fmt.Errorf("app: dlq mongo: %w", err)
		}
		dlqRepo = mongodlq.NewDeadLetterRepository(mongoClient, cfg.Kafka.DLQ.Database, cfg.Kafka.DLQ.Collection)
	}

	publisher := kafkaInfra.NewPublisher(writer, cfg.Kafka.CustomerTopic, dlqRepo)

	registerHandler := commands.NewRegisterCustomerHandler(repo, indexer, publisher, zapLogger)
	getHandler := queries.NewGetCustomerHandler(repo)

	telemetryInterceptor := grpcmiddleware.UnaryTelemetryInterceptor(cfg.ServiceName, collector, sentryClient, zapLogger)
	transport := grpciface.NewTransport(
		registerHandler,
		getHandler,
		zapLogger,
		grpc.UnaryInterceptor(telemetryInterceptor),
	)

	address := fmt.Sprintf("%s:%d", cfg.GRPC.Host, cfg.GRPC.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", address, err)
	}

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", collector.Handler())

	metricsSrv := &http.Server{
		Addr:              cfg.Observability.Metrics.Addr,
		Handler:           metricsMux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	app := &App{
		cfg:        cfg,
		log:        zapLogger,
		metrics:    collector,
		tracer:     tracer,
		sentry:     sentryClient,
		pool:       pool,
		writer:     writer,
		indexer:    indexer,
		server:     transport,
		metricsSrv: metricsSrv,
		listener:   listener,
		mongo:      mongoClient,
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
			if tracer != nil {
				return tracer.Shutdown(ctx)
			}
			return nil
		},
		func(ctx context.Context) error {
			if sentryClient != nil {
				sentryClient.Flush(ctx)
			}
			return nil
		},
		func(ctx context.Context) error {
			if metricsSrv != nil {
				return metricsSrv.Shutdown(ctx)
			}
			return nil
		},
		func(ctx context.Context) error {
			if mongoClient != nil {
				return mongoClient.Disconnect(ctx)
			}
			return nil
		},
		func(context.Context) error {
			return zapLogger.Sync()
		},
	}

	app.log.Info("customer service configured", zap.String("service", cfg.ServiceName))

	return app, nil
}

// Run запускает инфраструктурные слои.
func (a *App) Run(ctx context.Context) error {
	a.log.Info("customer service started", zap.String("addr", a.listener.Addr().String()))

	if a.metricsSrv != nil {
		a.log.Info("metrics server listening", zap.String("addr", a.metricsSrv.Addr))
		go func() {
			if err := a.metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				a.log.Error("metrics server error", zap.Error(err))
			}
		}()
	}

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

func newMongoClient(ctx context.Context, uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}

	return client, nil
}
