package llm

import (
	"context"
)

// LLMClient is an interface for invoking LLM models
// This allows mocking in tests without making real API calls
type LLMClient interface {
	InvokeModel(ctx context.Context, request LLMRequest) (*LLMResponse, error)
	InvokeModelWithRetry(ctx context.Context, request LLMRequest) (*LLMResponse, error)
}
