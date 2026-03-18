package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/insiderone/notifier/internal/queue"
	"github.com/insiderone/notifier/internal/repository"
)

type Scheduler struct {
	repo     *repository.NotificationRepository
	producer *queue.Producer
	interval time.Duration
	batchSize int
	logger   *slog.Logger
}

func NewScheduler(
	repo *repository.NotificationRepository,
	producer *queue.Producer,
	interval time.Duration,
	batchSize int,
	logger *slog.Logger,
) *Scheduler {
	return &Scheduler{
		repo:      repo,
		producer:  producer,
		interval:  interval,
		batchSize: batchSize,
		logger:    logger,
	}
}

// Run polls Postgres for due scheduled notifications and enqueues them.
func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.logger.Info("scheduler started", "interval", s.interval)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler stopped")
			return
		case <-ticker.C:
			s.poll(ctx)
		}
	}
}

func (s *Scheduler) poll(ctx context.Context) {
	notifications, err := s.repo.FetchScheduledDue(ctx, s.batchSize)
	if err != nil {
		s.logger.Error("fetching scheduled notifications", "error", err)
		return
	}

	if len(notifications) == 0 {
		return
	}

	s.logger.Info("enqueuing scheduled notifications", "count", len(notifications))

	for _, n := range notifications {
		if err := s.producer.Enqueue(ctx, n.ID, n.Channel, n.Priority); err != nil {
			s.logger.Error("enqueuing scheduled notification", "id", n.ID, "error", err)
		}
	}
}
