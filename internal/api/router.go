package api

import (
	"log/slog"
	"net/http"

	"github.com/insiderone/notifier/internal/api/handler"
	"github.com/insiderone/notifier/internal/api/middleware"
	"github.com/insiderone/notifier/internal/api/websocket"
)

func NewRouter(
	notificationHandler *handler.NotificationHandler,
	templateHandler *handler.TemplateHandler,
	healthHandler *handler.HealthHandler,
	wsHub *websocket.Hub,
	logger *slog.Logger,
) http.Handler {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", healthHandler.Health)
	mux.HandleFunc("GET /ready", healthHandler.Ready)

	// Metrics
	mux.Handle("GET /metrics", handler.MetricsHandler())

	// Notifications
	mux.HandleFunc("POST /api/v1/notifications", notificationHandler.Create)
	mux.HandleFunc("POST /api/v1/notifications/batch", notificationHandler.CreateBatch)
	mux.HandleFunc("GET /api/v1/notifications/{id}", notificationHandler.Get)
	mux.HandleFunc("GET /api/v1/notifications", notificationHandler.List)
	mux.HandleFunc("GET /api/v1/batches/{batchId}/notifications", notificationHandler.ListByBatch)
	mux.HandleFunc("POST /api/v1/notifications/{id}/cancel", notificationHandler.Cancel)
	mux.HandleFunc("GET /api/v1/notifications/{id}/attempts", notificationHandler.GetAttempts)

	// Templates
	mux.HandleFunc("POST /api/v1/templates", templateHandler.Create)
	mux.HandleFunc("GET /api/v1/templates", templateHandler.List)
	mux.HandleFunc("GET /api/v1/templates/{id}", templateHandler.Get)
	mux.HandleFunc("PUT /api/v1/templates/{id}", templateHandler.Update)
	mux.HandleFunc("DELETE /api/v1/templates/{id}", templateHandler.Delete)

	// WebSocket
	mux.HandleFunc("GET /ws/notifications/{id}", wsHub.HandleWebSocket)

	// Apply middleware
	var h http.Handler = mux
	h = middleware.Logging(logger)(h)
	h = middleware.Recovery(logger)(h)
	h = middleware.RequestID(h)

	return h
}
