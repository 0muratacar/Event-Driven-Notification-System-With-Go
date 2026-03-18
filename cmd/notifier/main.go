package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/insiderone/notifier/internal/api"
	"github.com/insiderone/notifier/internal/api/handler"
	ws "github.com/insiderone/notifier/internal/api/websocket"
	"github.com/insiderone/notifier/internal/config"
	"github.com/insiderone/notifier/internal/delivery"
	"github.com/insiderone/notifier/internal/domain"
	"github.com/insiderone/notifier/internal/queue"
	"github.com/insiderone/notifier/internal/ratelimit"
	"github.com/insiderone/notifier/internal/repository"
	"github.com/insiderone/notifier/internal/service"
	"github.com/insiderone/notifier/internal/tracing"
	"github.com/insiderone/notifier/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(logger); err != nil {
		logger.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Setup tracing
	shutdownTracing, err := tracing.Setup(ctx, cfg.Tracing)
	if err != nil {
		return fmt.Errorf("setting up tracing: %w", err)
	}
	defer func() { _ = shutdownTracing(context.Background()) }()

	// Connect to Postgres
	poolCfg, err := pgxpool.ParseConfig(cfg.Postgres.DSN)
	if err != nil {
		return fmt.Errorf("parsing postgres config: %w", err)
	}
	poolCfg.MaxConns = cfg.Postgres.MaxConns
	poolCfg.MinConns = cfg.Postgres.MinConns
	poolCfg.MaxConnLifetime = cfg.Postgres.MaxConnLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return fmt.Errorf("connecting to postgres: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("pinging postgres: %w", err)
	}
	logger.Info("connected to postgres")

	// Run migrations
	if err := runMigrations(cfg.Postgres.DSN); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}
	logger.Info("migrations applied")

	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer func() { _ = rdb.Close() }()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("connecting to redis: %w", err)
	}
	logger.Info("connected to redis")

	// Initialize repositories
	notifRepo := repository.NewNotificationRepository(pool)
	tmplRepo := repository.NewTemplateRepository(pool)

	// Initialize queue components
	producer := queue.NewProducer(rdb)
	consumer := queue.NewConsumer(rdb, "worker-0", logger)
	dlq := queue.NewDLQ(rdb)

	// Ensure consumer groups exist
	if err := consumer.EnsureGroups(ctx); err != nil {
		return fmt.Errorf("ensuring consumer groups: %w", err)
	}

	// Initialize services
	tmplSvc := service.NewTemplateService(tmplRepo, logger)
	notifSvc := service.NewNotificationService(notifRepo, tmplSvc, producer, logger, service.NotificationServiceConfig{
		DefaultMaxRetries: cfg.Worker.MaxRetries,
	})

	// Initialize delivery providers
	providers := map[domain.Channel]delivery.Provider{
		domain.ChannelEmail: delivery.NewEmailProvider(cfg.Delivery.WebhookBaseURL, cfg.Delivery.EmailPath, cfg.Delivery.Timeout),
		domain.ChannelSMS:   delivery.NewSMSProvider(cfg.Delivery.WebhookBaseURL, cfg.Delivery.SMSPath, cfg.Delivery.Timeout),
		domain.ChannelPush:  delivery.NewPushProvider(cfg.Delivery.WebhookBaseURL, cfg.Delivery.PushPath, cfg.Delivery.Timeout),
	}

	// Initialize rate limiter
	limiter := ratelimit.NewLimiter(rdb, cfg.Worker.RateLimitPerSec)

	// Initialize WebSocket hub
	wsHub := ws.NewHub(logger)

	// Initialize handlers
	notifHandler := handler.NewNotificationHandler(notifSvc, logger)
	tmplHandler := handler.NewTemplateHandler(tmplSvc, logger)
	healthHandler := handler.NewHealthHandler(pool, rdb)

	// Setup router
	router := api.NewRouter(notifHandler, tmplHandler, healthHandler, wsHub, logger)

	// Initialize HTTP server
	server := api.NewServer(cfg.Server, router, logger)

	// Initialize workers
	dispatcher := worker.NewDispatcher(consumer, producer, dlq, notifRepo, providers, limiter, wsHub, logger)
	workerPool := worker.NewPool(cfg.Worker.PoolSize, logger)
	scheduler := worker.NewScheduler(notifRepo, producer, cfg.Worker.SchedulerInterval, cfg.Worker.BatchSize, logger)

	// Start workers
	workerPool.Start(ctx, dispatcher.Work)

	// Start delayed retry processor
	go dispatcher.ProcessDelayed(ctx)

	// Start scheduler
	go scheduler.Run(ctx)

	// Start HTTP server
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start()
	}()

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
	}

	// Graceful shutdown
	if err := server.Shutdown(context.Background()); err != nil {
		logger.Error("server shutdown error", "error", err)
	}

	// Cancel context to stop workers
	cancel()
	workerPool.Wait()

	logger.Info("shutdown complete")
	return nil
}

func runMigrations(dsn string) error {
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		return fmt.Errorf("creating migrator: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("applying migrations: %w", err)
	}
	return nil
}
