package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadJudgesConfig_Success(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "judges.yaml")

	configContent := `judges:
  default_model:
    max_tokens: 256
    temperature: 0.0
    retry: true

  evaluators:
    - name: relevance
      enabled: true
      description: "Checks relevance"
      requires_context: false
      prompt: |
        Score the answer: {{.Answer}}
        {"score": <float>, "reason": "<string>"}
      model:
        max_tokens: 128
        retry: false

    - name: faithfulness
      enabled: true
      description: "Checks faithfulness"
      requires_context: true
      prompt: |
        Context: {{.Context}}
        Answer: {{.Answer}}
        {"score": <float>, "reason": "<string>"}
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Set env var to point to test config
	os.Setenv("JUDGES_CONFIG_PATH", configPath)
	defer os.Unsetenv("JUDGES_CONFIG_PATH")

	// Load config
	cfg, err := LoadJudgesConfig()
	if err != nil {
		t.Fatalf("LoadJudgesConfig() failed: %v", err)
	}

	// Verify structure
	if len(cfg.Judges.Evaluators) != 2 {
		t.Errorf("Expected 2 evaluators, got %d", len(cfg.Judges.Evaluators))
	}

	// Check default model
	if cfg.Judges.DefaultModel.MaxTokens != 256 {
		t.Errorf("Expected default max_tokens=256, got %d", cfg.Judges.DefaultModel.MaxTokens)
	}
	if cfg.Judges.DefaultModel.Temperature != 0.0 {
		t.Errorf("Expected default temperature=0.0, got %f", cfg.Judges.DefaultModel.Temperature)
	}
	if !cfg.Judges.DefaultModel.Retry {
		t.Error("Expected default retry=true")
	}

	// Check first judge (has model override)
	relevance := cfg.Judges.Evaluators[0]
	if relevance.Name != "relevance" {
		t.Errorf("Expected judge name 'relevance', got '%s'", relevance.Name)
	}
	if !relevance.Enabled {
		t.Error("Expected relevance to be enabled")
	}
	if relevance.RequiresContext {
		t.Error("Expected relevance.requires_context=false")
	}

	// Check model override was applied
	if relevance.Model.MaxTokens != 128 {
		t.Errorf("Expected relevance max_tokens=128, got %d", relevance.Model.MaxTokens)
	}
	if relevance.Model.Retry {
		t.Error("Expected relevance retry=false")
	}
	// Temperature should inherit from default (merged in applyDefaults)
	if relevance.Model.Temperature != 0.0 {
		t.Errorf("Expected relevance temperature=0.0 (inherited), got %f", relevance.Model.Temperature)
	}

	// Check second judge (no model override - should use defaults)
	faithfulness := cfg.Judges.Evaluators[1]
	if faithfulness.Name != "faithfulness" {
		t.Errorf("Expected judge name 'faithfulness', got '%s'", faithfulness.Name)
	}
	if !faithfulness.RequiresContext {
		t.Error("Expected faithfulness.requires_context=true")
	}

	// Model should be populated with defaults
	if faithfulness.Model == nil {
		t.Fatal("Expected faithfulness.Model to be populated with defaults")
	}
	if faithfulness.Model.MaxTokens != 256 {
		t.Errorf("Expected faithfulness max_tokens=256 (default), got %d", faithfulness.Model.MaxTokens)
	}
	if faithfulness.Model.Temperature != 0.0 {
		t.Errorf("Expected faithfulness temperature=0.0 (default), got %f", faithfulness.Model.Temperature)
	}
	if !faithfulness.Model.Retry {
		t.Error("Expected faithfulness retry=true (default)")
	}
}

func TestLoadJudgesConfig_DefaultPath(t *testing.T) {
	// Test that default path is used when env var not set
	os.Unsetenv("JUDGES_CONFIG_PATH")

	// This will fail since configs/judges.yaml may not exist in test environment
	// But we're testing the path resolution logic
	_, err := LoadJudgesConfig()

	// We expect an error about file not found or parse error
	if err == nil {
		// If no error, the file exists and we loaded it successfully
		// This is fine - means the actual config file is present
		t.Log("Default config file loaded successfully")
	} else {
		// Check that error mentions the default path
		if !contains(err.Error(), "configs/judges.yaml") {
			t.Errorf("Expected error to mention default path 'configs/judges.yaml', got: %v", err)
		}
	}
}

func TestLoadJudgesConfig_FileNotFound(t *testing.T) {
	os.Setenv("JUDGES_CONFIG_PATH", "/nonexistent/path/judges.yaml")
	defer os.Unsetenv("JUDGES_CONFIG_PATH")

	_, err := LoadJudgesConfig()
	if err == nil {
		t.Error("Expected error for nonexistent config file")
	}

	if !contains(err.Error(), "failed to read config file") {
		t.Errorf("Expected 'failed to read config file' error, got: %v", err)
	}
}

func TestLoadJudgesConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// Invalid YAML
	invalidContent := `judges:
  evaluators:
    - name: test
      prompt: "test"
      invalid_indent:
    wrong_level
`

	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	os.Setenv("JUDGES_CONFIG_PATH", configPath)
	defer os.Unsetenv("JUDGES_CONFIG_PATH")

	_, err := LoadJudgesConfig()
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}

	if !contains(err.Error(), "failed to parse YAML") {
		t.Errorf("Expected 'failed to parse YAML' error, got: %v", err)
	}
}

func TestValidate_NoJudges(t *testing.T) {
	cfg := &JudgesConfig{
		Judges: Judges{
			Evaluators: []JudgeConfiguration{},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for empty evaluators list")
	}

	if !contains(err.Error(), "no judges configured") {
		t.Errorf("Expected 'no judges configured' error, got: %v", err)
	}
}

func TestValidate_MissingName(t *testing.T) {
	cfg := &JudgesConfig{
		Judges: Judges{
			Evaluators: []JudgeConfiguration{
				{
					Name:   "",
					Prompt: "test",
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for missing name")
	}

	if !contains(err.Error(), "missing name") {
		t.Errorf("Expected 'missing name' error, got: %v", err)
	}
}

func TestValidate_MissingPrompt(t *testing.T) {
	cfg := &JudgesConfig{
		Judges: Judges{
			Evaluators: []JudgeConfiguration{
				{
					Name:   "test",
					Prompt: "",
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for missing prompt")
	}

	if !contains(err.Error(), "missing prompt") {
		t.Errorf("Expected 'missing prompt' error, got: %v", err)
	}
}

func TestValidate_InvalidPromptTemplate(t *testing.T) {
	cfg := &JudgesConfig{
		Judges: Judges{
			Evaluators: []JudgeConfiguration{
				{
					Name:   "test",
					Prompt: "{{.InvalidSyntax",
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for invalid template syntax")
	}

	if !contains(err.Error(), "invalid prompt template") {
		t.Errorf("Expected 'invalid prompt template' error, got: %v", err)
	}
}

func TestValidate_DuplicateNames(t *testing.T) {
	cfg := &JudgesConfig{
		Judges: Judges{
			Evaluators: []JudgeConfiguration{
				{
					Name:   "relevance",
					Prompt: "test1",
				},
				{
					Name:   "relevance",
					Prompt: "test2",
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for duplicate names")
	}

	if !contains(err.Error(), "duplicate judge name") {
		t.Errorf("Expected 'duplicate judge name' error, got: %v", err)
	}
}

func TestValidate_NegativeMaxTokens(t *testing.T) {
	cfg := &JudgesConfig{
		Judges: Judges{
			DefaultModel: ModelConfig{
				MaxTokens: -100,
			},
			Evaluators: []JudgeConfiguration{
				{
					Name:   "test",
					Prompt: "test",
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for negative max_tokens")
	}

	if !contains(err.Error(), "negative max_tokens") {
		t.Errorf("Expected 'negative max_tokens' error, got: %v", err)
	}
}

func TestValidate_InvalidTemperature(t *testing.T) {
	tests := []struct {
		name        string
		temperature float64
	}{
		{"negative", -0.1},
		{"too high", 1.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &JudgesConfig{
				Judges: Judges{
					DefaultModel: ModelConfig{
						Temperature: tt.temperature,
					},
					Evaluators: []JudgeConfiguration{
						{
							Name:   "test",
							Prompt: "test",
						},
					},
				},
			}

			err := cfg.Validate()
			if err == nil {
				t.Errorf("Expected validation error for temperature=%f", tt.temperature)
			}

			if !contains(err.Error(), "invalid temperature") {
				t.Errorf("Expected 'invalid temperature' error, got: %v", err)
			}
		})
	}
}

func TestApplyDefaults_PopulatesDefaultModel(t *testing.T) {
	cfg := &JudgesConfig{
		Judges: Judges{
			DefaultModel: ModelConfig{
				// All zero values - should get defaults
			},
			Evaluators: []JudgeConfiguration{
				{Name: "test", Prompt: "test"},
			},
		},
	}

	applyDefaults(cfg)

	if cfg.Judges.DefaultModel.MaxTokens != 256 {
		t.Errorf("Expected default max_tokens=256, got %d", cfg.Judges.DefaultModel.MaxTokens)
	}
	if cfg.Judges.DefaultModel.Temperature != 0.0 {
		t.Errorf("Expected default temperature=0.0, got %f", cfg.Judges.DefaultModel.Temperature)
	}
}

func TestApplyDefaults_CreatesModelForJudges(t *testing.T) {
	cfg := &JudgesConfig{
		Judges: Judges{
			DefaultModel: ModelConfig{
				MaxTokens:   300,
				Temperature: 0.7,
				Retry:       true,
			},
			Evaluators: []JudgeConfiguration{
				{Name: "test", Prompt: "test", Model: nil},
			},
		},
	}

	applyDefaults(cfg)

	judge := cfg.Judges.Evaluators[0]
	if judge.Model == nil {
		t.Fatal("Expected judge.Model to be created")
	}
	if judge.Model.MaxTokens != 300 {
		t.Errorf("Expected max_tokens=300, got %d", judge.Model.MaxTokens)
	}
	if judge.Model.Temperature != 0.7 {
		t.Errorf("Expected temperature=0.7, got %f", judge.Model.Temperature)
	}
	if !judge.Model.Retry {
		t.Error("Expected retry=true")
	}
}

func TestApplyDefaults_MergesPartialOverrides(t *testing.T) {
	cfg := &JudgesConfig{
		Judges: Judges{
			DefaultModel: ModelConfig{
				MaxTokens:   256,
				Temperature: 0.5,
				Retry:       true,
			},
			Evaluators: []JudgeConfiguration{
				{
					Name:   "test",
					Prompt: "test",
					Model: &ModelConfig{
						MaxTokens: 512, // Only override max_tokens
						// Temperature and Retry are zero values
					},
				},
			},
		},
	}

	applyDefaults(cfg)

	judge := cfg.Judges.Evaluators[0]
	if judge.Model.MaxTokens != 512 {
		t.Errorf("Expected max_tokens=512 (override), got %d", judge.Model.MaxTokens)
	}
	if judge.Model.Temperature != 0.5 {
		t.Errorf("Expected temperature=0.5 (merged from default), got %f", judge.Model.Temperature)
	}
}

func TestValidate_RequiresContextWithoutContextPlaceholder(t *testing.T) {
	cfg := &JudgesConfig{
		Judges: Judges{
			Evaluators: []JudgeConfiguration{
				{
					Name:            "faithfulness",
					Prompt:          "Score this answer: {{.Answer}}", // Missing {{.Context}}
					RequiresContext: true,
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for judge requiring context without {{.Context}} in prompt")
	}

	if !contains(err.Error(), "requires context") || !contains(err.Error(), "{{.Context}}") {
		t.Errorf("Expected error about missing {{.Context}}, got: %v", err)
	}
}

func TestValidate_RequiresContextWithContextPlaceholder(t *testing.T) {
	cfg := &JudgesConfig{
		Judges: Judges{
			Evaluators: []JudgeConfiguration{
				{
					Name:            "faithfulness",
					Prompt:          "Context: {{.Context}}\nAnswer: {{.Answer}}", // Has {{.Context}}
					RequiresContext: true,
				},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Expected validation to pass for judge with {{.Context}}, got: %v", err)
	}
}

func TestValidate_RequiresExpectedOutputWithoutPlaceholder(t *testing.T) {
	cfg := &JudgesConfig{
		Judges: Judges{
			Evaluators: []JudgeConfiguration{
				{
					Name:                   "correctness",
					Prompt:                 "Answer: {{.Answer}}\nScore it.", // Missing {{.ExpectedOutput}}
					RequiresExpectedOutput: true,
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for judge requiring expected_output without {{.ExpectedOutput}} in prompt")
	}

	if !contains(err.Error(), "requires expected_output") || !contains(err.Error(), "{{.ExpectedOutput}}") {
		t.Errorf("Expected error about missing {{.ExpectedOutput}}, got: %v", err)
	}
}

func TestValidate_RequiresExpectedOutputWithPlaceholder(t *testing.T) {
	cfg := &JudgesConfig{
		Judges: Judges{
			Evaluators: []JudgeConfiguration{
				{
					Name:                   "correctness",
					Prompt:                 "Answer: {{.Answer}}\nExpected: {{.ExpectedOutput}}", // Has {{.ExpectedOutput}}
					RequiresExpectedOutput: true,
				},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Expected validation to pass for judge with {{.ExpectedOutput}}, got: %v", err)
	}
}

func TestValidate_RequiresBothContextAndExpectedOutput(t *testing.T) {
	cfg := &JudgesConfig{
		Judges: Judges{
			Evaluators: []JudgeConfiguration{
				{
					Name: "complex",
					Prompt: `Context: {{.Context}}
Answer: {{.Answer}}
Expected: {{.ExpectedOutput}}`,
					RequiresContext:        true,
					RequiresExpectedOutput: true,
				},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Expected validation to pass for judge with both {{.Context}} and {{.ExpectedOutput}}, got: %v", err)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
