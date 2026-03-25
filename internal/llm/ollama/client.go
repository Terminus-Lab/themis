package ollama

import (
	"context"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type Client struct {
	Client       *openai.Client
	ModelID      string
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

func NewClient(ctx context.Context, baseURL, modelID string) (*Client, error) {
	openaiClient := openai.NewClient(
		option.WithAPIKey("ollama"),
		option.WithBaseURL(baseURL),
		option.WithMaxRetries(3),
	)

	return &Client{
		Client:       &openaiClient,
		ModelID:      modelID,
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     12 * time.Second,
	}, nil
}
