package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/Terminus-Lab/themis/internal/llm"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

type claudeMessageRequest struct {
	AnthropicVersion string          `json:"anthropic_version"`
	MaxTokens        int             `json:"max_tokens"`
	Temperature      float64         `json:"temperature"`
	Messages         []claudeMessage `json:"messages"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeMessageResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
}

var anthropicVersion = "bedrock-2023-05-31"

func (c *Client) InvokeModel(ctx context.Context, request llm.LLMRequest) (*llm.LLMResponse, error) {
	payload := claudeMessageRequest{
		AnthropicVersion: anthropicVersion,
		MaxTokens:        request.MaxTokens,
		Temperature:      request.Temperature,
		Messages: []claudeMessage{
			{
				Role:    "user",
				Content: request.Prompt,
			},
		},
	}

	byes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("unable to serialize claude request. Error: %w", err)
	}

	output, err := c.Client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     &c.ModelID,
		Body:        byes,
		Accept:      aws.String("application/json"),
		ContentType: aws.String("application/json"),
	})

	if err != nil {
		return nil, fmt.Errorf("unable to invoke claude model. Error: %w", err)
	}

	var response claudeMessageResponse
	if err := json.Unmarshal(output.Body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bedrock response. Error: %w", err)
	}

	// Extract the response
	var content string
	if len(response.Content) > 0 {
		content = response.Content[0].Text
	}

	return &llm.LLMResponse{
		Content:    content,
		StopReason: response.StopReason,
	}, nil
}

func (c *Client) InvokeModelWithRetry(ctx context.Context, request llm.LLMRequest) (*llm.LLMResponse, error) {
	var lastErr error

	for attempt := 0; attempt < c.MaxRetries; attempt++ {
		response, err := c.InvokeModel(ctx, request)
		if err == nil {
			return response, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return nil, fmt.Errorf("non-retryable error: %w", err)
		}

		delay := calculateBackoff(attempt, c.InitialDelay, c.MaxDelay)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			continue
		}
	}

	return nil, fmt.Errorf("max retries %d exceeded: %w", c.MaxRetries, lastErr)
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// 1. Throttling errors
	if strings.Contains(errStr, "ThrottlingException") ||
		strings.Contains(errStr, "TooManyRequestsException") ||
		strings.Contains(errStr, "Rate exceeded") {
		return true
	}

	// 2. Service errors (5xx)
	if strings.Contains(errStr, "InternalServerException") ||
		strings.Contains(errStr, "ServiceUnavailableException") ||
		strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "503") {
		return true
	}

	// 3. Network errors
	if strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "EOF") ||
		strings.Contains(errStr, "timeout") {
		return true
	}

	// Non-retryable errors (4xx client errors, validation errors, etc.)
	return false
}

func calculateBackoff(attempt int, initialDelay, maxDelay time.Duration) time.Duration {
	backoff := float64(initialDelay) + math.Pow(2, float64(attempt))

	if backoff > float64(maxDelay) {
		backoff = float64(maxDelay)
	}

	jitter := backoff * 0.2 * (2*rand.Float64() - 1) // Random value between -20% and +20%
	backoff += jitter

	return time.Duration(backoff)
}
