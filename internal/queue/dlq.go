package queue

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const DLQStream = "notifications:dlq"

type DLQ struct {
	client *redis.Client
}

func NewDLQ(client *redis.Client) *DLQ {
	return &DLQ{client: client}
}

func (d *DLQ) Push(ctx context.Context, notificationID uuid.UUID, channel, lastError string) error {
	_, err := d.client.XAdd(ctx, &redis.XAddArgs{
		Stream: DLQStream,
		Values: map[string]any{
			"notification_id": notificationID.String(),
			"channel":         channel,
			"error":           lastError,
		},
	}).Result()
	if err != nil {
		return fmt.Errorf("XADD to DLQ: %w", err)
	}
	return nil
}

func (d *DLQ) Read(ctx context.Context, count int64) ([]Message, error) {
	results, err := d.client.XRange(ctx, DLQStream, "-", "+").Result()
	if err != nil {
		return nil, fmt.Errorf("XRANGE DLQ: %w", err)
	}

	var messages []Message
	limit := int(count)
	for i, msg := range results {
		if i >= limit {
			break
		}
		messages = append(messages, Message{
			StreamID:       DLQStream,
			MessageID:      msg.ID,
			NotificationID: msg.Values["notification_id"].(string),
			Channel:        msg.Values["channel"].(string),
		})
	}
	return messages, nil
}
