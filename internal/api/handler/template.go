package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/insiderone/notifier/internal/domain"
	"github.com/insiderone/notifier/internal/service"
)

type TemplateHandler struct {
	svc    *service.TemplateService
	logger *slog.Logger
}

func NewTemplateHandler(svc *service.TemplateService, logger *slog.Logger) *TemplateHandler {
	return &TemplateHandler{svc: svc, logger: logger}
}

func (h *TemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	t, err := h.svc.Create(r.Context(), req)
	if err != nil {
		h.logger.Error("create template failed", "error", err)
		handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, t)
}

func (h *TemplateHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	t, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, t)
}

func (h *TemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	templates, err := h.svc.List(r.Context())
	if err != nil {
		h.logger.Error("list templates failed", "error", err)
		handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"templates": templates})
}

func (h *TemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	var req domain.UpdateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	t, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		h.logger.Error("update template failed", "error", err)
		handleDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, t)
}

func (h *TemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
