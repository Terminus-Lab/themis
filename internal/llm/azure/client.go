package azure

import (
	"fmt"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type Client struct {
	Client       openai.Client
	ModelID      string
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

func NewClient(apiKey string, model string, azureEndpoint string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Azure OpenAI API key is required")
	}
	if model == "" {
		return nil, fmt.Errorf("Azure OpenAI deployment name is required")
	}
	if azureEndpoint == "" {
		return nil, fmt.Errorf("Azure OpenAI endpoint is required")
	}

	openaiClient := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(azureEndpoint),
		option.WithMaxRetries(3),
	)

	return &Client{
		Client:       openaiClient,
		ModelID:      model,
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     12 * time.Second,
	}, nil
}
