package azure

import (
	"fmt"
	"net/http"
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
		return nil, fmt.Errorf("azure OpenAI API key is required")
	}
	if model == "" {
		return nil, fmt.Errorf("azure OpenAI deployment name is required")
	}
	if azureEndpoint == "" {
		return nil, fmt.Errorf("azure OpenAI endpoint is required")
	}

	// Construct proper Azure OpenAI base URL
	// Format: https://{resource}.openai.azure.com/openai/deployments/{deployment-name}
	// The SDK will append /chat/completions
	baseURL := fmt.Sprintf("%s/openai/deployments/%s", azureEndpoint, model)

	// Middleware to add api-version query parameter to all requests
	addAPIVersion := func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
		q := req.URL.Query()
		q.Set("api-version", "2024-12-01-preview")
		req.URL.RawQuery = q.Encode()
		return next(req)
	}

	openaiClient := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
		option.WithMiddleware(addAPIVersion),
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
