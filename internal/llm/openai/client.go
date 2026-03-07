package openaipaltform

import (
	"context"
	"time"

	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/openai/openai-go"
)

type Client struct {
	Client       *openai
	ModelID      string
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

func NewClient(ctx context.Context, openAiKey string, modelID string) (*Client, error) {
	openaiClient := openai.NewClient(option.WithAPIKey(openAiKey))

	return &Client{
		Client:       openaiClient,
		ModelID:      modelID,
		MaxRetries:   0,
		InitialDelay: 0,
		MaxDelay:     0,
	}
}
