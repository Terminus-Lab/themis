package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Terminus-Lab/themis/internal/executor"
	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type Consumer struct {
	client       *redis.Client
	stream       string
	groupID      string
	consumerName string
	executor     *executor.Executor
	logger       *zerolog.Logger
}

func NewConsumer(client *redis.Client, stream string, groupID string, consumerName string, exec *executor.Executor, logger *zerolog.Logger) *Consumer {
	return &Consumer{
		client:       client,
		stream:       stream,
		groupID:      groupID,
		consumerName: consumerName,
		executor:     exec,
		logger:       logger,
	}
}

func (c *Consumer) Setup(ctx context.Context) error {
	err := c.client.XGroupCreateMkStream(ctx, c.stream, c.groupID, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info().
		Str("stream", c.stream).
		Str("group", c.groupID).
		Str("consumer", c.consumerName).
		Msg("Consumer started")

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
				// timeout, no message -> loop again
				continue
			}

			if ctx.Err() != nil {
				return ctx.Err() // context cancelled during block
			}

			c.logger.Error().Err(err).Msg("Failed to read from stream")
			continue
		}

		for _, msg := range msgs[0].Messages {
			c.process(ctx, msg)
		}
	}
}

func (c *Consumer) Stop() error {
	// No-op
	return nil

}

func (c *Consumer) process(ctx context.Context, msg redis.XMessage) {
	c.logger.Info().Str("id", msg.ID).Msg("Message received")

	// decode json
	payload, ok := msg.Values["payload"].(string)
	if !ok {
		c.logger.Error().Str("id", msg.ID).Msg("Missing payload field")
		c.ack(ctx, msg.ID)
		return
	}

	var evalRequest models.EvaluationRequest
	if err := json.Unmarshal([]byte(payload), &evalRequest); err != nil {
		c.logger.Error().Err(err).Str("id", msg.ID).Msg("Failed to decode message")
		c.ack(ctx, msg.ID) // bad message — ACK to skip it
		return
	}

	evalCtx := normalize(evalRequest)
	result := c.executor.Execute(ctx, evalCtx)

	c.logger.Info().
		Str("id", msg.ID).
		Str("verdict", string(result.Verdict)).
		Float64("confidence", result.Confidence).
		Msg("Evaluation complete")

	c.ack(ctx, msg.ID)

}

func (c *Consumer) ack(ctx context.Context, msgID string) {
	if err := c.client.XAck(ctx, c.stream, c.groupID, msgID).Err(); err != nil {
		c.logger.Error().Err(err).Str("id", msgID).Msg("Failed to ACK message")
	}
}

func normalize(req models.EvaluationRequest) models.EvaluationContext {
	return models.EvaluationContext{
		RequestID:      req.EventID,
		ConversationID: req.ConversationID,
		AgentName:      req.Agent.Name,
		AgentVersion:   req.Agent.Version,
		Query:          req.Interaction.UserQuery,
		Context:        req.Interaction.Context,
		Answer:         req.Interaction.Answer,
		ExpectedOutput: req.Interaction.ExpectedOutput,
		CreatedAt:      time.Now(),
	}
}
