package queue

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/insiderone/notifier/internal/domain"
)

const (
	ConsumerGroup = "notifier-workers"
)

type Message struct {
	StreamID       string
	MessageID      string
	NotificationID string
	Channel        string
}

type Consumer struct {
	client       *redis.Client
	consumerName string
	logger       *slog.Logger
}

func NewConsumer(client *redis.Client, consumerName string, logger *slog.Logger) *Consumer {
	return &Consumer{
		client:       client,
		consumerName: consumerName,
		logger:       logger,
	}
}

// EnsureGroups creates consumer groups for all priority streams if they don't exist.
func (c *Consumer) EnsureGroups(ctx context.Context) error {
	streams := []string{
		domain.PriorityHigh.StreamKey(),
		domain.PriorityNormal.StreamKey(),
		domain.PriorityLow.StreamKey(),
	}
	for _, stream := range streams {
		err := c.client.XGroupCreateMkStream(ctx, stream, ConsumerGroup, "0").Err()
		if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
			return fmt.Errorf("creating consumer group for %s: %w", stream, err)
		}
	}
	return nil
}

// ReadMessages reads from priority streams in order: high, normal, low.
// Returns messages from the highest-priority stream that has pending work.
func (c *Consumer) ReadMessages(ctx context.Context, count int64, blockTimeout time.Duration) ([]Message, error) {
	// Priority order: high first, then normal, then low
	streams := []string{
		domain.PriorityHigh.StreamKey(),
		domain.PriorityNormal.StreamKey(),
		domain.PriorityLow.StreamKey(),
	}

	// Try to read from all streams, Redis will return from whichever has data
	streamArgs := make([]string, 0, len(streams)*2)
	streamArgs = append(streamArgs, streams...)
	for range streams {
		streamArgs = append(streamArgs, ">")
	}

	results, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    ConsumerGroup,
		Consumer: c.consumerName,
		Streams:  streamArgs,
		Count:    count,
		Block:    blockTimeout,
	}).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("XREADGROUP: %w", err)
	}

	var messages []Message
	for _, stream := range results {
		for _, msg := range stream.Messages {
			messages = append(messages, Message{
				StreamID:       stream.Stream,
				MessageID:      msg.ID,
				NotificationID: msg.Values["notification_id"].(string),
				Channel:        msg.Values["channel"].(string),
			})
		}
	}
	return messages, nil
}

// Ack acknowledges a message in its stream.
func (c *Consumer) Ack(ctx context.Context, stream, messageID string) error {
	return c.client.XAck(ctx, stream, ConsumerGroup, messageID).Err()
}

// ClaimStale reclaims messages that have been pending for longer than minIdle.
func (c *Consumer) ClaimStale(ctx context.Context, minIdle time.Duration) ([]Message, error) {
	streams := []string{
		domain.PriorityHigh.StreamKey(),
		domain.PriorityNormal.StreamKey(),
		domain.PriorityLow.StreamKey(),
	}

	var messages []Message
	for _, stream := range streams {
		result, _, err := c.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream:   stream,
			Group:    ConsumerGroup,
			Consumer: c.consumerName,
			MinIdle:  minIdle,
			Start:    "0-0",
			Count:    10,
		}).Result()
		if err != nil {
			c.logger.Warn("XAUTOCLAIM error", "stream", stream, "error", err)
			continue
		}
		for _, msg := range result {
			messages = append(messages, Message{
				StreamID:       stream,
				MessageID:      msg.ID,
				NotificationID: msg.Values["notification_id"].(string),
				Channel:        msg.Values["channel"].(string),
			})
		}
	}
	return messages, nil
}
