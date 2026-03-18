package queue

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/insiderone/notifier/internal/domain"
)

type Producer struct {
	client *redis.Client
}

func NewProducer(client *redis.Client) *Producer {
	return &Producer{client: client}
}

func (p *Producer) Enqueue(ctx context.Context, notificationID uuid.UUID, channel domain.Channel, priority domain.Priority) error {
	streamKey := priority.StreamKey()

	_, err := p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]any{
			"notification_id": notificationID.String(),
			"channel":         string(channel),
		},
	}).Result()
	if err != nil {
		return fmt.Errorf("XADD to %s: %w", streamKey, err)
	}
	return nil
}

// EnqueueDelayed adds a notification to the delayed retry sorted set.
// The score is the Unix timestamp when the notification should be retried.
func (p *Producer) EnqueueDelayed(ctx context.Context, notificationID uuid.UUID, channel domain.Channel, priority domain.Priority, retryAt float64) error {
	member := fmt.Sprintf("%s:%s:%s", notificationID.String(), string(channel), string(priority))
	_, err := p.client.ZAdd(ctx, "notifications:delayed", redis.Z{
		Score:  retryAt,
		Member: member,
	}).Result()
	if err != nil {
		return fmt.Errorf("ZADD delayed: %w", err)
	}
	return nil
}

// PopDelayedDue atomically removes and returns members from the delayed set
// whose score (retry timestamp) is <= now.
func (p *Producer) PopDelayedDue(ctx context.Context, now float64, count int64) ([]string, error) {
	// Use ZRANGEBYSCORE + ZREM in a pipeline
	pipe := p.client.Pipeline()
	rangeCmd := pipe.ZRangeByScore(ctx, "notifications:delayed", &redis.ZRangeBy{
		Min:   "-inf",
		Max:   fmt.Sprintf("%f", now),
		Count: count,
	})
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("pipeline exec: %w", err)
	}

	members := rangeCmd.Val()
	if len(members) == 0 {
		return nil, nil
	}

	// Remove fetched members
	memberIfaces := make([]any, len(members))
	for i, m := range members {
		memberIfaces[i] = m
	}
	p.client.ZRem(ctx, "notifications:delayed", memberIfaces...)

	return members, nil
}
