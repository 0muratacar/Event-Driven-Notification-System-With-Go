package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/insiderone/notifier/internal/domain"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

func handleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "resource not found")
	case errors.Is(err, domain.ErrDuplicate):
		writeError(w, http.StatusConflict, "duplicate entry")
	case errors.Is(err, domain.ErrIdempotencyConflict):
		writeError(w, http.StatusConflict, "idempotency key already exists")
	case errors.Is(err, domain.ErrCannotCancel):
		writeError(w, http.StatusUnprocessableEntity, "notification cannot be cancelled in current status")
	case errors.Is(err, domain.ErrBatchTooLarge):
		writeError(w, http.StatusBadRequest, "batch size exceeds maximum of 1000")
	case errors.Is(err, domain.ErrInvalidChannel):
		writeError(w, http.StatusBadRequest, "invalid channel")
	case errors.Is(err, domain.ErrInvalidPriority):
		writeError(w, http.StatusBadRequest, "invalid priority")
	case errors.Is(err, domain.ErrTemplateRender):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
