package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/insiderone/notifier/internal/domain"
	"github.com/insiderone/notifier/internal/repository"
	"github.com/insiderone/notifier/internal/validator"
)

type QueueProducer interface {
	Enqueue(ctx context.Context, notificationID uuid.UUID, channel domain.Channel, priority domain.Priority) error
}

type NotificationService struct {
	repo     *repository.NotificationRepository
	tmplSvc  *TemplateService
	producer QueueProducer
	logger   *slog.Logger
	cfg      NotificationServiceConfig
}

type NotificationServiceConfig struct {
	DefaultMaxRetries int
}

func NewNotificationService(
	repo *repository.NotificationRepository,
	tmplSvc *TemplateService,
	producer QueueProducer,
	logger *slog.Logger,
	cfg NotificationServiceConfig,
) *NotificationService {
	if cfg.DefaultMaxRetries == 0 {
		cfg.DefaultMaxRetries = 5
	}
	return &NotificationService{
		repo:     repo,
		tmplSvc:  tmplSvc,
		producer: producer,
		logger:   logger,
		cfg:      cfg,
	}
}

func (s *NotificationService) Create(ctx context.Context, req domain.CreateNotificationRequest) (*domain.Notification, error) {
	if err := validator.ValidateStruct(req); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrValidation, err)
	}
	if !req.Channel.Valid() {
		return nil, domain.ErrInvalidChannel
	}
	if !req.Priority.Valid() {
		return nil, domain.ErrInvalidPriority
	}

	n := &domain.Notification{
		IdempotencyKey: req.IdempotencyKey,
		Channel:        req.Channel,
		Priority:       req.Priority,
		Recipient:      req.Recipient,
		Subject:        req.Subject,
		Body:           req.Body,
		TemplateID:     req.TemplateID,
		TemplateVars:   req.TemplateVars,
		MaxRetries:     s.cfg.DefaultMaxRetries,
	}

	if req.MaxRetries != nil {
		n.MaxRetries = *req.MaxRetries
	}

	// Render template if specified
	if req.TemplateID != nil {
		tmpl, err := s.tmplSvc.GetByID(ctx, *req.TemplateID)
		if err != nil {
			return nil, fmt.Errorf("fetching template: %w", err)
		}
		subject, body, err := s.tmplSvc.Render(tmpl, req.TemplateVars)
		if err != nil {
			return nil, err
		}
		n.Body = body
		if subject != "" {
			n.Subject = subject
		}
	}

	// Validate content for the channel
	if err := validator.ValidateNotificationContent(n.Channel, n.Recipient, n.Subject, n.Body); err != nil {
		return nil, err
	}

	// Set status based on scheduling
	if req.ScheduledAt != nil {
		n.Status = domain.StatusScheduled
		n.ScheduledAt = req.ScheduledAt
	} else {
		n.Status = domain.StatusPending
	}

	if err := s.repo.Create(ctx, n); err != nil {
		return nil, err
	}

	s.logger.Info("notification created", "id", n.ID, "channel", n.Channel, "status", n.Status)

	// Enqueue for immediate delivery
	if n.Status == domain.StatusPending && s.producer != nil {
		if err := s.producer.Enqueue(ctx, n.ID, n.Channel, n.Priority); err != nil {
			s.logger.Error("failed to enqueue notification", "id", n.ID, "error", err)
			// Don't fail the request — scheduler will pick it up
		} else {
			if err := s.repo.UpdateStatus(ctx, n.ID, domain.StatusQueued, ""); err != nil {
				s.logger.Error("failed to update status to queued", "id", n.ID, "error", err)
			}
			n.Status = domain.StatusQueued
		}
	}

	return n, nil
}

func (s *NotificationService) CreateBatch(ctx context.Context, req domain.BatchCreateRequest) ([]domain.Notification, error) {
	if err := validator.ValidateStruct(req); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrValidation, err)
	}
	if len(req.Notifications) > 1000 {
		return nil, domain.ErrBatchTooLarge
	}

	batchID := uuid.Must(uuid.NewV7())
	notifications := make([]*domain.Notification, 0, len(req.Notifications))

	for i, r := range req.Notifications {
		if !r.Channel.Valid() {
			return nil, fmt.Errorf("%w at index %d", domain.ErrInvalidChannel, i)
		}
		if !r.Priority.Valid() {
			return nil, fmt.Errorf("%w at index %d", domain.ErrInvalidPriority, i)
		}

		n := &domain.Notification{
			IdempotencyKey: r.IdempotencyKey,
			BatchID:        &batchID,
			Channel:        r.Channel,
			Priority:       r.Priority,
			Recipient:      r.Recipient,
			Subject:        r.Subject,
			Body:           r.Body,
			TemplateID:     r.TemplateID,
			TemplateVars:   r.TemplateVars,
			MaxRetries:     s.cfg.DefaultMaxRetries,
		}

		if r.MaxRetries != nil {
			n.MaxRetries = *r.MaxRetries
		}

		if r.TemplateID != nil {
			tmpl, err := s.tmplSvc.GetByID(ctx, *r.TemplateID)
			if err != nil {
				return nil, fmt.Errorf("template at index %d: %w", i, err)
			}
			subject, body, err := s.tmplSvc.Render(tmpl, r.TemplateVars)
			if err != nil {
				return nil, fmt.Errorf("template render at index %d: %w", i, err)
			}
			n.Body = body
			if subject != "" {
				n.Subject = subject
			}
		}

		if err := validator.ValidateNotificationContent(n.Channel, n.Recipient, n.Subject, n.Body); err != nil {
			return nil, fmt.Errorf("validation at index %d: %w", i, err)
		}

		if r.ScheduledAt != nil {
			n.Status = domain.StatusScheduled
			n.ScheduledAt = r.ScheduledAt
		} else {
			n.Status = domain.StatusPending
		}

		notifications = append(notifications, n)
	}

	if err := s.repo.CreateBatch(ctx, notifications); err != nil {
		return nil, err
	}

	s.logger.Info("batch created", "batch_id", batchID, "count", len(notifications))

	// Enqueue pending notifications
	if s.producer != nil {
		for _, n := range notifications {
			if n.Status == domain.StatusPending {
				if err := s.producer.Enqueue(ctx, n.ID, n.Channel, n.Priority); err != nil {
					s.logger.Error("failed to enqueue batch notification", "id", n.ID, "error", err)
				} else {
					s.repo.UpdateStatus(ctx, n.ID, domain.StatusQueued, "")
					n.Status = domain.StatusQueued
				}
			}
		}
	}

	result := make([]domain.Notification, len(notifications))
	for i, n := range notifications {
		result[i] = *n
	}
	return result, nil
}

func (s *NotificationService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *NotificationService) List(ctx context.Context, filter domain.NotificationFilter) (*domain.NotificationList, error) {
	return s.repo.List(ctx, filter)
}

func (s *NotificationService) Cancel(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Cancel(ctx, id); err != nil {
		return err
	}
	s.logger.Info("notification cancelled", "id", id)
	return nil
}

func (s *NotificationService) GetDeliveryAttempts(ctx context.Context, notificationID uuid.UUID) ([]domain.DeliveryAttempt, error) {
	// Verify notification exists
	if _, err := s.repo.GetByID(ctx, notificationID); err != nil {
		return nil, err
	}
	return s.repo.GetDeliveryAttempts(ctx, notificationID)
}
