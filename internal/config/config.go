package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Server   ServerConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	Worker   WorkerConfig
	Delivery DeliveryConfig
	Tracing  TracingConfig
}

type ServerConfig struct {
	Host         string        `env:"SERVER_HOST" envDefault:"0.0.0.0"`
	Port         int           `env:"SERVER_PORT" envDefault:"8080"`
	ReadTimeout  time.Duration `env:"SERVER_READ_TIMEOUT" envDefault:"10s"`
	WriteTimeout time.Duration `env:"SERVER_WRITE_TIMEOUT" envDefault:"30s"`
	IdleTimeout  time.Duration `env:"SERVER_IDLE_TIMEOUT" envDefault:"60s"`
}

func (s ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

type PostgresConfig struct {
	DSN             string        `env:"POSTGRES_DSN" envDefault:"postgres://postgres:postgres@localhost:5432/notifier?sslmode=disable"`
	MaxConns        int32         `env:"POSTGRES_MAX_CONNS" envDefault:"25"`
	MinConns        int32         `env:"POSTGRES_MIN_CONNS" envDefault:"5"`
	MaxConnLifetime time.Duration `env:"POSTGRES_MAX_CONN_LIFETIME" envDefault:"1h"`
}

type RedisConfig struct {
	Addr     string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	Password string `env:"REDIS_PASSWORD" envDefault:""`
	DB       int    `env:"REDIS_DB" envDefault:"0"`
}

type WorkerConfig struct {
	PoolSize          int           `env:"WORKER_POOL_SIZE" envDefault:"10"`
	SchedulerInterval time.Duration `env:"WORKER_SCHEDULER_INTERVAL" envDefault:"5s"`
	BatchSize         int           `env:"WORKER_BATCH_SIZE" envDefault:"100"`
	MaxRetries        int           `env:"WORKER_MAX_RETRIES" envDefault:"5"`
	RateLimitPerSec   int           `env:"WORKER_RATE_LIMIT_PER_SEC" envDefault:"100"`
}

type DeliveryConfig struct {
	WebhookBaseURL string        `env:"DELIVERY_WEBHOOK_BASE_URL" envDefault:"https://webhook.site"`
	EmailPath      string        `env:"DELIVERY_EMAIL_PATH" envDefault:"/email"`
	SMSPath        string        `env:"DELIVERY_SMS_PATH" envDefault:"/sms"`
	PushPath       string        `env:"DELIVERY_PUSH_PATH" envDefault:"/push"`
	Timeout        time.Duration `env:"DELIVERY_TIMEOUT" envDefault:"10s"`
}

type TracingConfig struct {
	Enabled  bool   `env:"TRACING_ENABLED" envDefault:"false"`
	Endpoint string `env:"TRACING_ENDPOINT" envDefault:"localhost:4317"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return cfg, nil
}
