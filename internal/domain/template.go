package domain

import (
	"time"

	"github.com/google/uuid"
)

type Template struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Channel   Channel   `json:"channel"`
	Subject   string    `json:"subject,omitempty"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateTemplateRequest struct {
	Name    string  `json:"name" validate:"required,min=1,max=255"`
	Channel Channel `json:"channel" validate:"required"`
	Subject string  `json:"subject" validate:"omitempty,max=500"`
	Body    string  `json:"body" validate:"required,max=50000"`
}

type UpdateTemplateRequest struct {
	Name    *string  `json:"name" validate:"omitempty,min=1,max=255"`
	Channel *Channel `json:"channel" validate:"omitempty"`
	Subject *string  `json:"subject" validate:"omitempty,max=500"`
	Body    *string  `json:"body" validate:"omitempty,max=50000"`
}
