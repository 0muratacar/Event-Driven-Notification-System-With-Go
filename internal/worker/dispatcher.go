package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/insiderone/notifier/internal/api/websocket"
	"github.com/insiderone/notifier/internal/delivery"
	"github.com/insiderone/notifier/internal/domain"
	"github.com/insiderone/notifier/internal/queue"
	"github.com/insiderone/notifier/internal/ratelimit"
	"github.com/insiderone/notifier/internal/repository"
)

var (
	notificationsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "notifications_processed_total",
		Help: "Total number of notifications processed",
	}, []string{"channel", "status"})

	notificationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "notification_delivery_duration_seconds",
		Help:    "Duration of notification delivery",
		Buckets: prometheus.DefBuckets,
	}, []string{"channel"})

	retryCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "notifications_retried_total",
		Help: "Total number of notifications retried",
	})

	dlqCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "notifications_dlq_total",
		Help: "Total number of notifications sent to DLQ",
	})
)

type Dispatcher struct {
	consumer  *queue.Consumer
	producer  *queue.Producer
	dlq       *queue.DLQ
	repo      *repository.NotificationRepository
	providers map[domain.Channel]delivery.Provider
	limiter   *ratelimit.Limiter
	wsHub     *websocket.Hub
	logger    *slog.Logger
}

func NewDispatcher(
	consumer *queue.Consumer,
	producer *queue.Producer,
	dlq *queue.DLQ,
	repo *repository.NotificationRepository,
	providers map[domain.Channel]delivery.Provider,
	limiter *ratelimit.Limiter,
	wsHub *websocket.Hub,
	logger *slog.Logger,
) *Dispatcher {
	return &Dispatcher{
		consumer:  consumer,
		producer:  producer,
		dlq:       dlq,
		repo:      repo,
		providers: providers,
		limiter:   limiter,
		wsHub:     wsHub,
		logger:    logger,
	}
}

// Work is the main work loop for a single worker goroutine.
func (d *Dispatcher) Work(ctx context.Context, workerID int) {
	logger := d.logger.With("worker_id", workerID)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		messages, err := d.consumer.ReadMessages(ctx, 1, 2*time.Second)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Error("reading messages", "error", err)
			time.Sleep(time.Second)
			continue
		}

		for _, msg := range messages {
			d.processMessage(ctx, logger, msg)
		}
	}
}

func (d *Dispatcher) processMessage(ctx context.Context, logger *slog.Logger, msg queue.Message) {
	notificationID, err := uuid.Parse(msg.NotificationID)
	if err != nil {
		logger.Error("invalid notification ID", "id", msg.NotificationID, "error", err)
		d.consumer.Ack(ctx, msg.StreamID, msg.MessageID)
		return
	}

	channel := domain.Channel(msg.Channel)
	logger = logger.With("notification_id", notificationID, "channel", channel)

	// Rate limit check
	for {
		allowed, err := d.limiter.Allow(ctx, string(channel))
		if err != nil {
			logger.Error("rate limiter error", "error", err)
			break
		}
		if allowed {
			break
		}
		// Rate limited — brief sleep and retry
		select {
		case <-ctx.Done():
			return
		case <-time.After(50 * time.Millisecond):
		}
	}

	// Fetch notification from DB
	n, err := d.repo.GetByID(ctx, notificationID)
	if err != nil {
		logger.Error("fetching notification", "error", err)
		d.consumer.Ack(ctx, msg.StreamID, msg.MessageID)
		return
	}

	// Skip if already delivered, cancelled, or failed
	if n.Status == domain.StatusDelivered || n.Status == domain.StatusCancelled || n.Status == domain.StatusFailed {
		d.consumer.Ack(ctx, msg.StreamID, msg.MessageID)
		return
	}

	// Update status to processing
	d.repo.UpdateStatus(ctx, n.ID, domain.StatusProcessing, "")

	// Get provider
	provider, ok := d.providers[n.Channel]
	if !ok {
		logger.Error("no provider for channel", "channel", n.Channel)
		d.repo.UpdateStatus(ctx, n.ID, domain.StatusFailed, fmt.Sprintf("no provider for channel: %s", n.Channel))
		d.consumer.Ack(ctx, msg.StreamID, msg.MessageID)
		return
	}

	// Deliver
	start := time.Now()
	result := provider.Send(ctx, n)
	duration := time.Since(start).Seconds()

	notificationDuration.WithLabelValues(string(n.Channel)).Observe(duration)

	// Record attempt
	d.repo.IncrementAttempt(ctx, n.ID)
	attempt := &domain.DeliveryAttempt{
		NotificationID: n.ID,
		AttemptNumber:  n.AttemptCount + 1,
		StatusCode:     result.StatusCode,
		ResponseBody:   result.ResponseBody,
		DurationMs:     result.DurationMs,
	}
	if result.Error != nil {
		attempt.Error = result.Error.Error()
	}
	d.repo.CreateDeliveryAttempt(ctx, attempt)

	if result.Error == nil {
		// Success
		d.repo.UpdateStatus(ctx, n.ID, domain.StatusDelivered, "")
		d.consumer.Ack(ctx, msg.StreamID, msg.MessageID)
		notificationsProcessed.WithLabelValues(string(n.Channel), "delivered").Inc()

		// Broadcast via WebSocket
		d.wsHub.Broadcast(n.ID, "delivered")

		logger.Info("notification delivered")
	} else {
		// Failure
		errMsg := result.Error.Error()
		logger.Warn("delivery failed", "error", errMsg, "attempt", n.AttemptCount+1)

		if delivery.ShouldRetry(n.AttemptCount+1, n.MaxRetries) {
			// Schedule retry
			retryAt := delivery.RetryAt(n.AttemptCount + 1)
			d.repo.UpdateStatus(ctx, n.ID, domain.StatusQueued, errMsg)
			d.producer.EnqueueDelayed(ctx, n.ID, n.Channel, n.Priority, float64(retryAt.UnixMilli()))
			retryCount.Inc()

			d.wsHub.Broadcast(n.ID, fmt.Sprintf("retry_scheduled:%s", retryAt.Format(time.RFC3339)))
		} else {
			// Max retries exhausted → DLQ
			d.repo.UpdateStatus(ctx, n.ID, domain.StatusFailed, errMsg)
			d.dlq.Push(ctx, n.ID, string(n.Channel), errMsg)
			dlqCount.Inc()
			notificationsProcessed.WithLabelValues(string(n.Channel), "failed").Inc()

			d.wsHub.Broadcast(n.ID, "failed")
		}
		d.consumer.Ack(ctx, msg.StreamID, msg.MessageID)
	}
}

// ProcessDelayed checks the delayed sorted set and re-enqueues notifications whose retry time has arrived.
func (d *Dispatcher) ProcessDelayed(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.processDelayedBatch(ctx)
		}
	}
}

func (d *Dispatcher) processDelayedBatch(ctx context.Context) {
	now := float64(time.Now().UnixMilli())

	// This is a simplified approach - in production you'd use ZPOPMIN or a Lua script
	// to atomically pop and process
	results, err := d.producer.PopDelayedDue(ctx, now, 50)
	if err != nil {
		d.logger.Error("processing delayed queue", "error", err)
		return
	}

	for _, member := range results {
		parts := parseMember(member)
		if parts == nil {
			continue
		}

		notificationID, _ := uuid.Parse(parts[0])
		channel := domain.Channel(parts[1])
		priority := domain.Priority(parts[2])

		if err := d.producer.Enqueue(ctx, notificationID, channel, priority); err != nil {
			d.logger.Error("re-enqueuing delayed notification", "id", notificationID, "error", err)
		}
	}
}

func parseMember(s string) []string {
	// Format: "uuid:channel:priority"
	parts := make([]string, 0, 3)
	start := 0
	count := 0
	for i, c := range s {
		if c == ':' {
			parts = append(parts, s[start:i])
			start = i + 1
			count++
			if count == 2 {
				break
			}
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	if len(parts) != 3 {
		return nil
	}
	return parts
}
