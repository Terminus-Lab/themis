package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Terminus-Lab/themis/internal/executor"
	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type ConversationConsumer struct {
	client       *redis.Client
	stream       string
	groupID      string
	consumerName string
	executor     *executor.ConversationExecutor
	logger       *zerolog.Logger
}

func NewConversationConsumer(client *redis.Client, stream string, groupID string, consumerName string, exec *executor.ConversationExecutor, logger *zerolog.Logger) *ConversationConsumer {
	return &ConversationConsumer{
		client:       client,
		stream:       stream,
		groupID:      groupID,
		consumerName: consumerName,
		executor:     exec,
		logger:       logger,
	}
}

func (c *ConversationConsumer) Setup(ctx context.Context) error {
	err := c.client.XGroupCreateMkStream(ctx, c.stream, c.groupID, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

func (c *ConversationConsumer) Start(ctx context.Context) error {
	c.logger.Info().
		Str("stream", c.stream).
		Str("group", c.groupID).
		Str("consumer", c.consumerName).
		Msg("Conversation consumer started")

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		msgs, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.groupID,
			Consumer: c.consumerName,
			Streams:  []string{c.stream, ">"},
			Count:    1,
			Block:    2 * time.Second,
		}).Result()

		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			c.logger.Error().Err(err).Msg("Failed to read from conversation stream")
			continue
		}

		for _, msg := range msgs[0].Messages {
			c.process(ctx, msg)
		}
	}
}

func (c *ConversationConsumer) Stop() error {
	return nil
}

func (c *ConversationConsumer) process(ctx context.Context, msg redis.XMessage) {
	c.logger.Info().Str("id", msg.ID).Msg("Conversation message received")

	req, err := decodeConversationPayload(msg)
	if err != nil {
		c.logger.Error().Err(err).Str("id", msg.ID).Msg("Failed to decode conversation message")
		c.ack(ctx, msg.ID)
		return
	}

	result := c.executor.Execute(ctx, req)

	c.logger.Info().
		Str("id", msg.ID).
		Str("conversation_id", result.ConversationID).
		Str("verdict", string(result.Verdict)).
		Float64("confidence", result.Confidence).
		Msg("Conversation evaluation complete")

	c.ack(ctx, msg.ID)
}

func decodeConversationPayload(msg redis.XMessage) (models.ConversationEvaluationRequest, error) {
	payload, ok := msg.Values["payload"].(string)
	if !ok {
		return models.ConversationEvaluationRequest{}, fmt.Errorf("missing or invalid payload field")
	}
	var req models.ConversationEvaluationRequest
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		return models.ConversationEvaluationRequest{}, fmt.Errorf("invalid JSON: %w", err)
	}
	return req, nil
}

func (c *ConversationConsumer) ack(ctx context.Context, msgID string) {
	if err := c.client.XAck(ctx, c.stream, c.groupID, msgID).Err(); err != nil {
		c.logger.Error().Err(err).Str("id", msgID).Msg("Failed to ACK conversation message")
	}
}
