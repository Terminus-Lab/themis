package judge

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Terminus-Lab/themis/internal/config"
	"github.com/Terminus-Lab/themis/internal/llm"
	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/rs/zerolog"
)

func TestNewLLMJudge_Success(t *testing.T) {
	logger := zerolog.Nop()

	cfg := config.JudgeConfiguration{
		Name:        "test-judge",
		Enabled:     true,
		Description: "Test judge",
		Prompt:      "Score: {{.Answer}}",
		Model: &config.ModelConfig{
			MaxTokens:   256,
			Temperature: 0.0,
			Retry:       false,
		},
	}

	judge, err := NewLLMJudge(cfg, &MockLLMClient{}, &logger)
	if err != nil {
		t.Fatalf("NewLLMJudge failed: %v", err)
	}

	if judge.name != "test-judge" {
		t.Errorf("Expected name 'test-judge', got '%s'", judge.name)
	}
	if judge.modelConfig.MaxTokens != 256 {
		t.Errorf("Expected MaxTokens=256, got %d", judge.modelConfig.MaxTokens)
	}
}

func TestNewLLMJudge_InvalidTemplate(t *testing.T) {
	logger := zerolog.Nop()

	cfg := config.JudgeConfiguration{
		Name:   "test-judge",
		Prompt: "{{.Invalid", // Invalid template syntax
		Model: &config.ModelConfig{
			MaxTokens: 256,
		},
	}

	_, err := NewLLMJudge(cfg, &MockLLMClient{}, &logger)
	if err == nil {
		t.Error("Expected error for invalid template")
	}
}

func TestNewLLMJudge_NilModelConfig(t *testing.T) {
	logger := zerolog.Nop()

	cfg := config.JudgeConfiguration{
		Name:   "test-judge",
		Prompt: "test",
		Model:  nil, // Should not happen after config loading
	}

	_, err := NewLLMJudge(cfg, &MockLLMClient{}, &logger)
	if err == nil {
		t.Error("Expected error for nil model config")
	}
}

func TestLLMJudge_Evaluate_Success(t *testing.T) {
	logger := zerolog.Nop()

	cfg := config.JudgeConfiguration{
		Name:   "relevance",
		Prompt: "Query: {{.Query}}\nAnswer: {{.Answer}}",
		Model: &config.ModelConfig{
			MaxTokens:   256,
			Temperature: 0.0,
			Retry:       false,
		},
	}

	mockClient := &MockLLMClient{
		ResponseToReturn: &llm.LLMResponse{
			Content: `{"score": 0.85, "reason": "Good match"}`,
		},
	}

	judge, err := NewLLMJudge(cfg, mockClient, &logger)
	if err != nil {
		t.Fatalf("NewLLMJudge failed: %v", err)
	}

	evalCtx := models.EvaluationContext{
		Query:  "What is AI?",
		Answer: "AI is artificial intelligence",
	}

	result := judge.Evaluate(context.Background(), evalCtx)

	if result.Score != 0.85 {
		t.Errorf("Expected score=0.85, got %f", result.Score)
	}
	if result.Reason != "Good match" {
		t.Errorf("Expected reason='Good match', got '%s'", result.Reason)
	}
	if result.Name != "relevance-judge" {
		t.Errorf("Expected name='relevance-judge', got '%s'", result.Name)
	}
}

func TestLLMJudge_Evaluate_TemplateExecutionFails(t *testing.T) {
	logger := zerolog.Nop()

	cfg := config.JudgeConfiguration{
		Name:   "test",
		Prompt: "{{.NonExistentField}}", // Field doesn't exist in EvaluationContext
		Model: &config.ModelConfig{
			MaxTokens: 256,
		},
	}

	judge, _ := NewLLMJudge(cfg, &MockLLMClient{}, &logger)

	evalCtx := models.EvaluationContext{
		Query:  "test",
		Answer: "test",
	}

	result := judge.Evaluate(context.Background(), evalCtx)

	if result.Score != 0.0 {
		t.Errorf("Expected score=0.0 for template error, got %f", result.Score)
	}
	if result.Reason == "" || len(result.Reason) < 5 {
		t.Errorf("Expected error reason, got '%s'", result.Reason)
	}
}

func TestLLMJudge_Evaluate_LLMCallFails(t *testing.T) {
	logger := zerolog.Nop()

	cfg := config.JudgeConfiguration{
		Name:   "test",
		Prompt: "Score: {{.Answer}}",
		Model: &config.ModelConfig{
			MaxTokens: 256,
			Retry:     false,
		},
	}

	mockClient := &MockLLMClient{
		ErrorToReturn: errors.New("API error"),
	}

	judge, _ := NewLLMJudge(cfg, mockClient, &logger)

	evalCtx := models.EvaluationContext{
		Answer: "test",
	}

	result := judge.Evaluate(context.Background(), evalCtx)

	if result.Score != 0.0 {
		t.Errorf("Expected score=0.0 for LLM error, got %f", result.Score)
	}
	if result.Reason != "Failed to call LLM" {
		t.Errorf("Expected LLM error reason, got '%s'", result.Reason)
	}
}

func TestLLMJudge_Evaluate_WithRetry(t *testing.T) {
	logger := zerolog.Nop()

	cfg := config.JudgeConfiguration{
		Name:   "test",
		Prompt: "Score: {{.Answer}}",
		Model: &config.ModelConfig{
			MaxTokens: 256,
			Retry:     true, // Should use InvokeModelWithRetry
		},
	}

	mockClient := &MockLLMClient{
		ResponseToReturn: &llm.LLMResponse{
			Content: `{"score": 0.9, "reason": "test"}`,
		},
	}

	judge, _ := NewLLMJudge(cfg, mockClient, &logger)

	evalCtx := models.EvaluationContext{
		Answer: "test",
	}

	result := judge.Evaluate(context.Background(), evalCtx)

	if result.Score != 0.9 {
		t.Errorf("Expected score=0.9, got %f", result.Score)
	}
	// Note: Cannot verify InvokeModelWithRetry was called vs InvokeModel
	// without modifying the existing MockLLMClient
}

func TestLLMJudge_Evaluate_InvalidJSON(t *testing.T) {
	logger := zerolog.Nop()

	cfg := config.JudgeConfiguration{
		Name:   "test",
		Prompt: "Score: {{.Answer}}",
		Model: &config.ModelConfig{
			MaxTokens: 256,
		},
	}

	mockClient := &MockLLMClient{
		ResponseToReturn: &llm.LLMResponse{
			Content: `not valid json`,
		},
	}

	judge, _ := NewLLMJudge(cfg, mockClient, &logger)

	evalCtx := models.EvaluationContext{
		Answer: "test",
	}

	result := judge.Evaluate(context.Background(), evalCtx)

	if result.Score != 0.0 {
		t.Errorf("Expected score=0.0 for invalid JSON, got %f", result.Score)
	}
	if result.Reason != "Failed to deserialize LLM response" {
		t.Errorf("Expected deserialization error, got '%s'", result.Reason)
	}
}

func TestLLMJudge_Evaluate_EmptyScoreAndReason(t *testing.T) {
	logger := zerolog.Nop()

	cfg := config.JudgeConfiguration{
		Name:   "test",
		Prompt: "Score: {{.Answer}}",
		Model: &config.ModelConfig{
			MaxTokens: 256,
		},
	}

	mockClient := &MockLLMClient{
		ResponseToReturn: &llm.LLMResponse{
			Content: `{"score": 0.0, "reason": ""}`,
		},
	}

	judge, _ := NewLLMJudge(cfg, mockClient, &logger)

	evalCtx := models.EvaluationContext{
		Answer: "test",
	}

	result := judge.Evaluate(context.Background(), evalCtx)

	if result.Score != 0.0 {
		t.Errorf("Expected score=0.0, got %f", result.Score)
	}
	if result.Reason != "Invalid LLM response: missing score and reason" {
		t.Errorf("Expected invalid response error, got '%s'", result.Reason)
	}
}

func TestLLMJudge_Evaluate_ScoreOutOfRange(t *testing.T) {
	tests := []struct {
		name  string
		score float64
	}{
		{"negative", -0.5},
		{"too high", 1.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zerolog.Nop()

			cfg := config.JudgeConfiguration{
				Name:   "test",
				Prompt: "Score: {{.Answer}}",
				Model: &config.ModelConfig{
					MaxTokens: 256,
				},
			}

			mockClient := &MockLLMClient{
				ResponseToReturn: &llm.LLMResponse{
					Content: fmt.Sprintf(`{"score": %f, "reason": "test"}`, tt.score),
				},
			}

			judge, _ := NewLLMJudge(cfg, mockClient, &logger)

			evalCtx := models.EvaluationContext{
				Answer: "test",
			}

			result := judge.Evaluate(context.Background(), evalCtx)

			if result.Score != 0.0 {
				t.Errorf("Expected score=0.0 for out of range score, got %f", result.Score)
			}
			if !contains(result.Reason, "out of range") {
				t.Errorf("Expected out of range error, got '%s'", result.Reason)
			}
		})
	}
}

// MockLLMClient for testing
type MockLLMClient struct {
	ResponseToReturn *llm.LLMResponse
	ErrorToReturn    error
	WasCalled        bool
	LastRequest      *llm.LLMRequest
}

func (m *MockLLMClient) InvokeModel(ctx context.Context, request llm.LLMRequest) (*llm.LLMResponse, error) {
	m.WasCalled = true
	m.LastRequest = &request
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}
	return m.ResponseToReturn, nil
}

func (m *MockLLMClient) InvokeModelWithRetry(ctx context.Context, request llm.LLMRequest) (*llm.LLMResponse, error) {
	m.WasCalled = true
	m.LastRequest = &request
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}
	return m.ResponseToReturn, nil
}

// Helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestStripMarkdownCodeBlock(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain JSON without markdown",
			input:    `{"score": 0.85, "reason": "Good answer"}`,
			expected: `{"score": 0.85, "reason": "Good answer"}`,
		},
		{
			name:     "JSON wrapped in markdown code block with json tag",
			input:    "```json\n{\"score\": 0.85, \"reason\": \"Good answer\"}\n```",
			expected: `{"score": 0.85, "reason": "Good answer"}`,
		},
		{
			name:     "JSON wrapped in markdown code block without language tag",
			input:    "```\n{\"score\": 0.85, \"reason\": \"Good answer\"}\n```",
			expected: `{"score": 0.85, "reason": "Good answer"}`,
		},
		{
			name:     "JSON with extra whitespace in markdown block",
			input:    "```json\n  {\"score\": 0.85, \"reason\": \"Good answer\"}  \n```",
			expected: `{"score": 0.85, "reason": "Good answer"}`,
		},
		{
			name:     "multiline JSON in markdown block",
			input:    "```json\n{\n  \"score\": 0.85,\n  \"reason\": \"Good answer\"\n}\n```",
			expected: "{\n  \"score\": 0.85,\n  \"reason\": \"Good answer\"\n}",
		},
		{
			name:     "JSON with leading/trailing whitespace",
			input:    "  \n{\"score\": 0.85, \"reason\": \"Good answer\"}\n  ",
			expected: `{"score": 0.85, "reason": "Good answer"}`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only markdown markers",
			input:    "```\n```",
			expected: "",
		},
		{
			name:     "incomplete markdown block (only opening)",
			input:    "```json\n{\"score\": 0.85}",
			expected: "```json\n{\"score\": 0.85}",
		},
		{
			name:     "JSON with backticks in the content",
			input:    "```json\n{\"score\": 0.85, \"reason\": \"Use ` for code\"}\n```",
			expected: "{\"score\": 0.85, \"reason\": \"Use ` for code\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripMarkdownCodeBlock(tt.input)
			if result != tt.expected {
				t.Errorf("stripMarkdownCodeBlock() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestLLMJudge_Evaluate_MarkdownWrappedJSON(t *testing.T) {
	logger := zerolog.Nop()

	cfg := config.JudgeConfiguration{
		Name:   "test",
		Prompt: "Score: {{.Answer}}",
		Model: &config.ModelConfig{
			MaxTokens: 256,
		},
	}

	// Simulate LLM returning markdown-wrapped JSON (common behavior)
	mockClient := &MockLLMClient{
		ResponseToReturn: &llm.LLMResponse{
			Content: "```json\n{\"score\": 0.85, \"reason\": \"Good answer\"}\n```",
		},
	}

	judge, _ := NewLLMJudge(cfg, mockClient, &logger)

	evalCtx := models.EvaluationContext{
		Answer: "test",
	}

	result := judge.Evaluate(context.Background(), evalCtx)

	// Should successfully parse despite markdown wrapping
	if result.Score != 0.85 {
		t.Errorf("Expected score=0.85, got %f", result.Score)
	}
	if result.Reason != "Good answer" {
		t.Errorf("Expected reason='Good answer', got '%s'", result.Reason)
	}
}
