package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthHandler(db *pgxpool.Pool, redis *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redis}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	checks := map[string]string{}

	if err := h.db.Ping(ctx); err != nil {
		checks["postgres"] = "unhealthy: " + err.Error()
	} else {
		checks["postgres"] = "ok"
	}

	if err := h.redis.Ping(ctx).Err(); err != nil {
		checks["redis"] = "unhealthy: " + err.Error()
	} else {
		checks["redis"] = "ok"
	}

	status := http.StatusOK
	for _, v := range checks {
		if v != "ok" {
			status = http.StatusServiceUnavailable
			break
		}
	}

	writeJSON(w, status, map[string]any{"status": statusText(status), "checks": checks})
}

func statusText(code int) string {
	if code == http.StatusOK {
		return "ok"
	}
	return "unhealthy"
}
