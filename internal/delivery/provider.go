package delivery

import (
	"context"

	"github.com/insiderone/notifier/internal/domain"
)

// Result represents the outcome of a delivery attempt.
type Result struct {
	StatusCode   int
	ResponseBody string
	Error        error
	DurationMs   int64
}

// Provider defines the interface for delivering notifications.
type Provider interface {
	// Send delivers a notification and returns the result.
	Send(ctx context.Context, notification *domain.Notification) Result
	// Channel returns which channel this provider handles.
	Channel() domain.Channel
}
