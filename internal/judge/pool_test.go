package judge

import (
	"testing"

	"github.com/Terminus-Lab/themis/internal/config"
	"github.com/Terminus-Lab/themis/internal/llm"
	"github.com/rs/zerolog"
)

func createTestRegistry() *llm.LLMClientRegistry {
	mockClient := &MockLLMClient{}
	return llm.NewLLMClientRegistry(map[llm.LLMFamily]map[string]llm.LLMClient{
		llm.FamilyAnthropic: {
			"test-model": mockClient,
		},
	})
}

func TestNewJudgePool(t *testing.T) {
	logger := zerolog.Nop()
	registry := createTestRegistry()

	pool := NewJudgePool(registry, &logger)

	if pool == nil {
		t.Fatal("Expected pool to be created")
	}
	if pool.registry == nil {
		t.Error("Expected registry to be set")
	}
}

func TestJudgePool_BuildFromConfig_Success(t *testing.T) {
	logger := zerolog.Nop()
	registry := createTestRegistry()

	pool := NewJudgePool(registry, &logger)

	cfg := &config.JudgesConfig{
		Judges: config.Judges{
			DefaultModel: config.ModelConfig{
				MaxTokens:   256,
				Temperature: 0.0,
				Retry:       true,
				ModelFamily: "anthropic",
				ModelID:     "test-model",
			},
			Evaluators: []config.JudgeConfiguration{
				{
					Name:    "judge1",
					Enabled: true,
					Prompt:  "Score: {{.Answer}}",
					Model: &config.ModelConfig{
						MaxTokens:   256,
						ModelFamily: "anthropic",
						ModelID:     "test-model",
					},
				},
				{
					Name:    "judge2",
					Enabled: true,
					Prompt:  "Score: {{.Query}}",
					Model: &config.ModelConfig{
						MaxTokens:   128,
						ModelFamily: "anthropic",
						ModelID:     "test-model",
					},
				},
			},
		},
	}

	judges, err := pool.BuildFromConfig(cfg)
	if err != nil {
		t.Fatalf("BuildFromConfig failed: %v", err)
	}

	if len(judges) != 2 {
		t.Errorf("Expected 2 judges, got %d", len(judges))
	}
}

func TestJudgePool_BuildFromConfig_SkipsDisabledJudges(t *testing.T) {
	logger := zerolog.Nop()
	registry := createTestRegistry()

	pool := NewJudgePool(registry, &logger)

	cfg := &config.JudgesConfig{
		Judges: config.Judges{
			DefaultModel: config.ModelConfig{
				MaxTokens:   256,
				ModelFamily: "anthropic",
				ModelID:     "test-model",
			},
			Evaluators: []config.JudgeConfiguration{
				{
					Name:    "judge1",
					Enabled: true,
					Prompt:  "Score: {{.Answer}}",
					Model: &config.ModelConfig{
						MaxTokens:   256,
						ModelFamily: "anthropic",
						ModelID:     "test-model",
					},
				},
				{
					Name:    "judge2",
					Enabled: false, // Disabled
					Prompt:  "Score: {{.Query}}",
					Model: &config.ModelConfig{
						MaxTokens:   128,
						ModelFamily: "anthropic",
						ModelID:     "test-model",
					},
				},
				{
					Name:    "judge3",
					Enabled: true,
					Prompt:  "Score: {{.Context}}",
					Model: &config.ModelConfig{
						MaxTokens:   256,
						ModelFamily: "anthropic",
						ModelID:     "test-model",
					},
				},
			},
		},
	}

	judges, err := pool.BuildFromConfig(cfg)
	if err != nil {
		t.Fatalf("BuildFromConfig failed: %v", err)
	}

	if len(judges) != 2 {
		t.Errorf("Expected 2 enabled judges, got %d", len(judges))
	}
}

func TestJudgePool_BuildFromConfig_NilConfig(t *testing.T) {
	logger := zerolog.Nop()
	registry := createTestRegistry()

	pool := NewJudgePool(registry, &logger)

	_, err := pool.BuildFromConfig(nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}
	if err.Error() != "judges config is nil" {
		t.Errorf("Expected 'judges config is nil' error, got: %v", err)
	}
}

func TestJudgePool_BuildFromConfig_NoEnabledJudges(t *testing.T) {
	logger := zerolog.Nop()
	registry := createTestRegistry()

	pool := NewJudgePool(registry, &logger)

	cfg := &config.JudgesConfig{
		Judges: config.Judges{
			DefaultModel: config.ModelConfig{
				MaxTokens:   256,
				ModelFamily: "anthropic",
				ModelID:     "test-model",
			},
			Evaluators: []config.JudgeConfiguration{
				{
					Name:    "judge1",
					Enabled: false,
					Prompt:  "Score: {{.Answer}}",
					Model: &config.ModelConfig{
						MaxTokens:   256,
						ModelFamily: "anthropic",
						ModelID:     "test-model",
					},
				},
			},
		},
	}

	_, err := pool.BuildFromConfig(cfg)
	if err == nil {
		t.Error("Expected error for no enabled judges")
	}

	expectedMsg := "no enabled event-scoped judges found in config"
	if err.Error() != expectedMsg {
		t.Errorf("Expected '%s' error, got: %v", expectedMsg, err)
	}
}

func TestJudgePool_BuildFromConfig_InvalidJudge(t *testing.T) {
	logger := zerolog.Nop()
	registry := createTestRegistry()

	pool := NewJudgePool(registry, &logger)

	cfg := &config.JudgesConfig{
		Judges: config.Judges{
			DefaultModel: config.ModelConfig{
				MaxTokens:   256,
				ModelFamily: "anthropic",
				ModelID:     "test-model",
			},
			Evaluators: []config.JudgeConfiguration{
				{
					Name:    "bad-judge",
					Enabled: true,
					Prompt:  "{{.Invalid", // Invalid template
					Model: &config.ModelConfig{
						MaxTokens:   256,
						ModelFamily: "anthropic",
						ModelID:     "test-model",
					},
				},
			},
		},
	}

	_, err := pool.BuildFromConfig(cfg)
	if err == nil {
		t.Error("Expected error for invalid judge")
	}

	// Should mention the judge name in the error
	if !contains(err.Error(), "bad-judge") {
		t.Errorf("Expected error to mention 'bad-judge', got: %v", err)
	}
}
