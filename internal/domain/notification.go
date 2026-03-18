package domain

import (
	"time"

	"github.com/google/uuid"
)

type Channel string

const (
	ChannelSMS   Channel = "sms"
	ChannelEmail Channel = "email"
	ChannelPush  Channel = "push"
)

func (c Channel) Valid() bool {
	switch c {
	case ChannelSMS, ChannelEmail, ChannelPush:
		return true
	}
	return false
}

type Priority string

const (
	PriorityHigh   Priority = "high"
	PriorityNormal Priority = "normal"
	PriorityLow    Priority = "low"
)

func (p Priority) Valid() bool {
	switch p {
	case PriorityHigh, PriorityNormal, PriorityLow:
		return true
	}
	return false
}

func (p Priority) StreamKey() string {
	return "notifications:" + string(p)
}

type Status string

const (
	StatusPending    Status = "pending"
	StatusScheduled  Status = "scheduled"
	StatusQueued     Status = "queued"
	StatusProcessing Status = "processing"
	StatusDelivered  Status = "delivered"
	StatusFailed     Status = "failed"
	StatusCancelled  Status = "cancelled"
)

type Notification struct {
	ID             uuid.UUID          `json:"id"`
	IdempotencyKey string             `json:"idempotency_key,omitempty"`
	BatchID        *uuid.UUID         `json:"batch_id,omitempty"`
	Channel        Channel            `json:"channel"`
	Priority       Priority           `json:"priority"`
	Recipient      string             `json:"recipient"`
	Subject        string             `json:"subject,omitempty"`
	Body           string             `json:"body"`
	TemplateID     *uuid.UUID         `json:"template_id,omitempty"`
	TemplateVars   map[string]string  `json:"template_vars,omitempty"`
	Status         Status             `json:"status"`
	ScheduledAt    *time.Time         `json:"scheduled_at,omitempty"`
	AttemptCount   int                `json:"attempt_count"`
	MaxRetries     int                `json:"max_retries"`
	LastError      string             `json:"last_error,omitempty"`
	DeliveredAt    *time.Time         `json:"delivered_at,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

type CreateNotificationRequest struct {
	IdempotencyKey string            `json:"idempotency_key" validate:"omitempty,max=255"`
	Channel        Channel           `json:"channel" validate:"required"`
	Priority       Priority          `json:"priority" validate:"required"`
	Recipient      string            `json:"recipient" validate:"required,max=500"`
	Subject        string            `json:"subject" validate:"omitempty,max=500"`
	Body           string            `json:"body" validate:"required_without=TemplateID,max=10000"`
	TemplateID     *uuid.UUID        `json:"template_id" validate:"omitempty"`
	TemplateVars   map[string]string `json:"template_vars" validate:"omitempty"`
	ScheduledAt    *time.Time        `json:"scheduled_at" validate:"omitempty"`
	MaxRetries     *int              `json:"max_retries" validate:"omitempty,min=0,max=10"`
}

type BatchCreateRequest struct {
	Notifications []CreateNotificationRequest `json:"notifications" validate:"required,min=1,max=1000,dive"`
}

type NotificationFilter struct {
	Status    *Status    `json:"status"`
	Channel   *Channel   `json:"channel"`
	BatchID   *uuid.UUID `json:"batch_id"`
	Cursor    *Cursor    `json:"cursor"`
	Limit     int        `json:"limit"`
}

type Cursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        uuid.UUID `json:"id"`
}

type NotificationList struct {
	Notifications []Notification `json:"notifications"`
	NextCursor    *Cursor        `json:"next_cursor,omitempty"`
}

type DeliveryAttempt struct {
	ID             uuid.UUID `json:"id"`
	NotificationID uuid.UUID `json:"notification_id"`
	AttemptNumber  int       `json:"attempt_number"`
	StatusCode     int       `json:"status_code"`
	ResponseBody   string    `json:"response_body,omitempty"`
	Error          string    `json:"error,omitempty"`
	DurationMs     int64     `json:"duration_ms"`
	CreatedAt      time.Time `json:"created_at"`
}
