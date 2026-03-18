package websocket

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

type Hub struct {
	mu          sync.RWMutex
	subscribers map[uuid.UUID]map[*websocket.Conn]struct{}
	logger      *slog.Logger
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		subscribers: make(map[uuid.UUID]map[*websocket.Conn]struct{}),
		logger:      logger,
	}
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid notification ID", http.StatusBadRequest)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Allow all origins in dev
	})
	if err != nil {
		h.logger.Error("websocket accept failed", "error", err)
		return
	}

	h.subscribe(id, conn)
	defer h.unsubscribe(id, conn)

	h.logger.Info("websocket client connected", "notification_id", id)

	// Keep connection alive, read messages to detect close
	ctx := r.Context()
	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			break
		}
	}
}

func (h *Hub) subscribe(id uuid.UUID, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.subscribers[id] == nil {
		h.subscribers[id] = make(map[*websocket.Conn]struct{})
	}
	h.subscribers[id][conn] = struct{}{}
}

func (h *Hub) unsubscribe(id uuid.UUID, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conns, ok := h.subscribers[id]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.subscribers, id)
		}
	}
	conn.Close(websocket.StatusNormalClosure, "")
}

// Broadcast sends a status update to all WebSocket clients watching a notification.
func (h *Hub) Broadcast(notificationID uuid.UUID, status string) {
	h.mu.RLock()
	conns := h.subscribers[notificationID]
	h.mu.RUnlock()

	if len(conns) == 0 {
		return
	}

	msg := []byte(`{"notification_id":"` + notificationID.String() + `","status":"` + status + `"}`)

	h.mu.RLock()
	defer h.mu.RUnlock()

	for conn := range conns {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := conn.Write(ctx, websocket.MessageText, msg); err != nil {
			h.logger.Warn("websocket write failed", "notification_id", notificationID, "error", err)
		}
		cancel()
	}
}
