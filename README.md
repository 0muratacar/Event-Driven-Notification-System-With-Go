# Notification Service

Event-driven notification system for sending millions of notifications daily across SMS, Email, and Push channels.

## Architecture

```
HTTP API → Service Layer → PostgreSQL (persistence)
                         → Redis Streams (async queue)

Workers (consumer group) → Rate Limiter → Delivery Providers → webhook.site
                         → Retry (exponential backoff via Redis sorted set)
                         → DLQ (dead letter queue)

Scheduler → Polls Postgres for due scheduled notifications → Redis Streams

WebSocket Hub → Real-time status updates to connected clients
```

### Key Design Decisions

- **Redis Streams** with consumer groups for at-least-once delivery and crash recovery (XAUTOCLAIM)
- **3 priority streams** (high/normal/low) — workers drain high before normal before low
- **Delayed retry via sorted set** — Redis Streams lack native delay; sorted set with score = delivery timestamp
- **Single binary** — HTTP server + workers in one process; scale by running multiple instances sharing the same consumer group
- **Cursor pagination** — `(created_at, id)` for consistent O(1) pagination
- **Idempotency via DB unique constraint** — race-safe, no distributed locks

## Quick Start

```bash
# Start Postgres + Redis + App
docker compose up --build

# Or run locally (requires Postgres + Redis running)
cp .env.example .env
# Edit .env with your webhook.site UUIDs
make run
```

## API Overview

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/notifications` | Create single notification |
| POST | `/api/v1/notifications/batch` | Create batch (up to 1000) |
| GET | `/api/v1/notifications/{id}` | Get by ID |
| GET | `/api/v1/notifications` | List with filters + cursor pagination |
| GET | `/api/v1/notifications/batch/{batchId}` | List by batch |
| POST | `/api/v1/notifications/{id}/cancel` | Cancel pending notification |
| GET | `/api/v1/notifications/{id}/attempts` | Get delivery attempts |
| POST/GET/PUT/DELETE | `/api/v1/templates[/{id}]` | Template CRUD |
| GET | `/health` | Liveness |
| GET | `/ready` | Readiness (Postgres + Redis) |
| GET | `/metrics` | Prometheus metrics |
| GET | `/ws/notifications/{id}` | WebSocket status updates |

Full OpenAPI spec: `api/openapi.yaml`

### Example: Create Notification

```bash
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "channel": "email",
    "priority": "high",
    "recipient": "user@example.com",
    "subject": "Welcome!",
    "body": "Hello and welcome to our platform."
  }'
```

### Example: Batch Create

```bash
curl -X POST http://localhost:8080/api/v1/notifications/batch \
  -H "Content-Type: application/json" \
  -d '{
    "notifications": [
      {"channel": "sms", "priority": "normal", "recipient": "+12025551234", "body": "Your code: 1234"},
      {"channel": "email", "priority": "low", "recipient": "user@example.com", "subject": "Update", "body": "New features available."}
    ]
  }'
```

## Configuration

All configuration via environment variables. See `.env.example` for the full list.

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8080 | HTTP server port |
| `POSTGRES_DSN` | `postgres://postgres:postgres@localhost:5432/notifier?sslmode=disable` | PostgreSQL connection string |
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `WORKER_POOL_SIZE` | 10 | Number of worker goroutines |
| `WORKER_MAX_RETRIES` | 5 | Default max retry attempts |
| `WORKER_RATE_LIMIT_PER_SEC` | 100 | Rate limit per channel per second |
| `DELIVERY_WEBHOOK_BASE_URL` | `https://webhook.site` | Base URL for delivery providers |

## Development

```bash
make build          # Build binary
make test           # Run all tests
make test-short     # Run unit tests only
make lint           # Run linter
make tidy           # go mod tidy
make docker-up      # Start with Docker Compose
make docker-down    # Stop Docker Compose
```

## Testing

```bash
# Unit tests (no external dependencies)
make test-short

# Full tests (requires Postgres + Redis)
make test
```
