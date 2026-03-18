package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/insiderone/notifier/internal/domain"
	"github.com/insiderone/notifier/internal/service"
)

type NotificationHandler struct {
	svc    *service.NotificationService
	logger *slog.Logger
}

func NewNotificationHandler(svc *service.NotificationService, logger *slog.Logger) *NotificationHandler {
	return &NotificationHandler{svc: svc, logger: logger}
}

func (h *NotificationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	n, err := h.svc.Create(r.Context(), req)
	if err != nil {
		h.logger.Error("create notification failed", "error", err)
		handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, n)
}

func (h *NotificationHandler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	var req domain.BatchCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	notifications, err := h.svc.CreateBatch(r.Context(), req)
	if err != nil {
		h.logger.Error("create batch failed", "error", err)
		handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"notifications": notifications,
		"count":         len(notifications),
	})
}

func (h *NotificationHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid notification ID")
		return
	}

	n, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, n)
}

func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := domain.NotificationFilter{}

	if s := r.URL.Query().Get("status"); s != "" {
		status := domain.Status(s)
		filter.Status = &status
	}
	if c := r.URL.Query().Get("channel"); c != "" {
		channel := domain.Channel(c)
		filter.Channel = &channel
	}
	if b := r.URL.Query().Get("batch_id"); b != "" {
		batchID, err := uuid.Parse(b)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid batch_id")
			return
		}
		filter.BatchID = &batchID
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		limit, err := strconv.Atoi(l)
		if err != nil || limit < 1 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		filter.Limit = limit
	}

	// Cursor pagination
	cursorTime := r.URL.Query().Get("cursor_time")
	cursorID := r.URL.Query().Get("cursor_id")
	if cursorTime != "" && cursorID != "" {
		t, err := time.Parse(time.RFC3339Nano, cursorTime)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid cursor_time")
			return
		}
		id, err := uuid.Parse(cursorID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid cursor_id")
			return
		}
		filter.Cursor = &domain.Cursor{CreatedAt: t, ID: id}
	}

	list, err := h.svc.List(r.Context(), filter)
	if err != nil {
		h.logger.Error("list notifications failed", "error", err)
		handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, list)
}

func (h *NotificationHandler) ListByBatch(w http.ResponseWriter, r *http.Request) {
	batchID, err := uuid.Parse(r.PathValue("batchId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid batch ID")
		return
	}

	filter := domain.NotificationFilter{BatchID: &batchID, Limit: 100}
	list, err := h.svc.List(r.Context(), filter)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, list)
}

func (h *NotificationHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid notification ID")
		return
	}

	if err := h.svc.Cancel(r.Context(), id); err != nil {
		handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (h *NotificationHandler) GetAttempts(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid notification ID")
		return
	}

	attempts, err := h.svc.GetDeliveryAttempts(r.Context(), id)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"attempts": attempts})
}
