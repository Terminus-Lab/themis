package llm

import (
	"context"
	"testing"
)

// MockLLMClient for testing
type MockLLMClient struct {
	modelID string
}

func (m *MockLLMClient) InvokeModel(ctx context.Context, request LLMRequest) (*LLMResponse, error) {
	return &LLMResponse{
		Content:    "mock response from " + m.modelID,
		StopReason: "end_turn",
	}, nil
}

func (m *MockLLMClient) InvokeModelWithRetry(ctx context.Context, request LLMRequest) (*LLMResponse, error) {
	return m.InvokeModel(ctx, request)
}

func TestLLMClientRegistry_Basic(t *testing.T) {
	// Setup clients
	sonnet := &MockLLMClient{modelID: "claude-sonnet-4"}
	haiku := &MockLLMClient{modelID: "claude-haiku-3.5"}
	gpt4 := &MockLLMClient{modelID: "gpt-4"}

	// Create registry with clients
	registry := NewLLMClientRegistry(map[LLMFamily]map[string]LLMClient{
		FamilyAnthropic: {
			"claude-sonnet-4":   sonnet,
			"claude-haiku-3.5":  haiku,
		},
		FamilyOpenAI: {
			"gpt-4": gpt4,
		},
	})

	// Test Get
	client, err := registry.Get(FamilyAnthropic, "claude-sonnet-4")
	if err != nil {
		t.Fatalf("Failed to get client: %v", err)
	}
	if client != sonnet {
		t.Error("Got wrong client")
	}

	// Test Get OpenAI
	client, err = registry.Get(FamilyOpenAI, "gpt-4")
	if err != nil {
		t.Fatalf("Failed to get OpenAI client: %v", err)
	}
	if client != gpt4 {
		t.Error("Got wrong OpenAI client")
	}
}

func TestLLMClientRegistry_ErrorCases(t *testing.T) {
	registry := NewLLMClientRegistry(nil)

	// Test Get on empty registry
	_, err := registry.Get(FamilyAnthropic, "any-model")
	if err == nil {
		t.Error("Expected error for non-existent family")
	}

	// Create registry with one client
	sonnet := &MockLLMClient{modelID: "claude-sonnet-4"}
	registry = NewLLMClientRegistry(map[LLMFamily]map[string]LLMClient{
		FamilyAnthropic: {
			"claude-sonnet-4": sonnet,
		},
	})

	// Test Get with non-existent model
	_, err = registry.Get(FamilyAnthropic, "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent model")
	}

	// Test Get with non-existent family
	_, err = registry.Get(FamilyOpenAI, "gpt-4")
	if err == nil {
		t.Error("Expected error for non-existent family")
	}
}

func TestLLMClientRegistry_Usage(t *testing.T) {
	// Setup clients
	sonnet := &MockLLMClient{modelID: "claude-sonnet-4"}
	haiku := &MockLLMClient{modelID: "claude-haiku-3.5"}
	gpt4 := &MockLLMClient{modelID: "gpt-4"}

	// Create registry
	registry := NewLLMClientRegistry(map[LLMFamily]map[string]LLMClient{
		FamilyAnthropic: {
			"claude-sonnet-4":  sonnet,
			"claude-haiku-3.5": haiku,
		},
		FamilyOpenAI: {
			"gpt-4": gpt4,
		},
	})

	// Use anthropic client
	client, _ := registry.Get(FamilyAnthropic, "claude-sonnet-4")
	ctx := context.Background()
	resp, _ := client.InvokeModel(ctx, LLMRequest{
		Prompt:      "test",
		MaxTokens:   100,
		Temperature: 0.0,
	})

	if resp.Content != "mock response from claude-sonnet-4" {
		t.Errorf("Unexpected response: %s", resp.Content)
	}

	// Use different anthropic model
	client, _ = registry.Get(FamilyAnthropic, "claude-haiku-3.5")
	resp, _ = client.InvokeModel(ctx, LLMRequest{
		Prompt:      "test",
		MaxTokens:   100,
		Temperature: 0.0,
	})

	if resp.Content != "mock response from claude-haiku-3.5" {
		t.Errorf("Unexpected response: %s", resp.Content)
	}

	// Use OpenAI client
	client, _ = registry.Get(FamilyOpenAI, "gpt-4")
	resp, _ = client.InvokeModel(ctx, LLMRequest{
		Prompt:      "test",
		MaxTokens:   100,
		Temperature: 0.0,
	})

	if resp.Content != "mock response from gpt-4" {
		t.Errorf("Unexpected response: %s", resp.Content)
	}
}
