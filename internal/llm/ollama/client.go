package ollama

import (
	"context"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const defaultBaseURL = "http://localhost:11434/v1"

type Client struct {
	Client       *openai.Client
	ModelID      string
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

func NewClient(ctx context.Context, baseURL, modelID string) (*Client, error) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

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
