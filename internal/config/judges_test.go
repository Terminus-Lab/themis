package config

import (
	"os"
	"path/filepath"
	"strings"
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
      prompt: |
        Score the answer: {{.Answer}}
        {"score": <float>, "reason": "<string>"}
      model:
        max_tokens: 128
        retry: false

    - name: faithfulness
      enabled: true
      description: "Checks faithfulness"
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
		if !contains(err.Error(), "config/judges.yaml") {
			t.Errorf("Expected error to mention default path 'config/judges.yaml', got: %v", err)
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

func TestLoadJudgesConfig_SearchOrder(t *testing.T) {
	t.Run("returns error when no config found", func(t *testing.T) {
		os.Unsetenv("JUDGES_CONFIG_PATH")

		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		os.Chdir(tmpDir)
		defer os.Chdir(origDir)

		_, err := LoadJudgesConfig()
		if err == nil {
			t.Fatal("Expected error when no config found, got nil")
		}

		if !strings.Contains(err.Error(), "judges.yaml not found") {
			t.Errorf("Expected helpful error message, got: %v", err)
		}

		t.Logf("Correctly returned error: %v", err)
	})

	// Test that JUDGES_CONFIG_PATH takes priority
	t.Run("respects JUDGES_CONFIG_PATH override", func(t *testing.T) {
		// Create temp config file
		tmpDir := t.TempDir()
		customConfigPath := filepath.Join(tmpDir, "custom-judges.yaml")

		customConfig := `
judges:
  default_model:
    modelFamily: "openai_platform"
    modelID: "gpt-4o-mini"
    max_tokens: 256
    temperature: 0.0
  evaluators:
    - name: test-judge
      enabled: true
      weight: 1.0
      prompt: |
        Test prompt
        Query: {{.Query}}
        Answer: {{.Answer}}
        Output: {"score": 1.0, "reason": "test"}
`
		if err := os.WriteFile(customConfigPath, []byte(customConfig), 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		// Set env var
		os.Setenv("JUDGES_CONFIG_PATH", customConfigPath)
		defer os.Unsetenv("JUDGES_CONFIG_PATH")

		// Load config
		cfg, err := LoadJudgesConfig()
		if err != nil {
			t.Fatalf("Failed to load custom config: %v", err)
		}

		if len(cfg.Judges.Evaluators) != 1 {
			t.Errorf("Expected 1 judge from custom config, got %d", len(cfg.Judges.Evaluators))
		}

		if cfg.Judges.Evaluators[0].Name != "test-judge" {
			t.Errorf("Expected judge name 'test-judge', got '%s'", cfg.Judges.Evaluators[0].Name)
		}

		t.Log("Successfully loaded custom config via JUDGES_CONFIG_PATH")
	})
}

func TestNormalizeJudgeWeights_PerScope(t *testing.T) {
	cfg := &JudgesConfig{
		Judges: Judges{
			DefaultModel: ModelConfig{MaxTokens: 256, ModelFamily: "openai", ModelID: "gpt-4o-mini"},
			Evaluators: []JudgeConfiguration{
				{Name: "relevance", Enabled: true, Scope: "event", Weight: 0.6,
					Prompt: "Score: {{.Answer}}\n{\"score\": 0.0, \"reason\": \"\"}"},
				{Name: "coherence", Enabled: true, Scope: "event", Weight: 0.4,
					Prompt: "Score: {{.Answer}}\n{\"score\": 0.0, \"reason\": \"\"}"},
				{Name: "conv-flow", Enabled: true, Scope: "conversation", Weight: 1.0,
					Prompt: "{{range .Turns}}{{.UserQuery}}{{end}}\n{\"score\": 0.0, \"reason\": \"\"}"},
			},
		},
	}

	applyDefaults(cfg)

	// Event judges should sum to 1.0 (already do: 0.6+0.4)
	eventSum := 0.0
	convSum := 0.0
	for _, j := range cfg.Judges.Evaluators {
		scope := j.Scope
		if scope == "" {
			scope = "event"
		}
		if scope == "event" && j.Enabled {
			eventSum += j.Weight
		}
		if scope == "conversation" && j.Enabled {
			convSum += j.Weight
		}
	}

	const tol = 0.001
	if eventSum < 1.0-tol || eventSum > 1.0+tol {
		t.Errorf("event judge weights should sum to 1.0, got %.4f", eventSum)
	}
	if convSum < 1.0-tol || convSum > 1.0+tol {
		t.Errorf("conversation judge weights should sum to 1.0, got %.4f", convSum)
	}
}

func TestNormalizeJudgeWeights_ConversationScopeDefault(t *testing.T) {
	// Judges without explicit scope should default to event scope
	cfg := &JudgesConfig{
		Judges: Judges{
			DefaultModel: ModelConfig{MaxTokens: 256, ModelFamily: "openai", ModelID: "gpt-4o-mini"},
			Evaluators: []JudgeConfiguration{
				{Name: "a", Enabled: true, Scope: "", Weight: 0.5,
					Prompt: "{{.Answer}}\n{\"score\": 0.0, \"reason\": \"\"}"},
				{Name: "b", Enabled: true, Scope: "", Weight: 0.5,
					Prompt: "{{.Answer}}\n{\"score\": 0.0, \"reason\": \"\"}"},
			},
		},
	}
	applyDefaults(cfg)

	total := 0.0
	for _, j := range cfg.Judges.Evaluators {
		total += j.Weight
	}
	const tol = 0.001
	if total < 1.0-tol || total > 1.0+tol {
		t.Errorf("weights should sum to 1.0, got %.4f", total)
	}
}

func TestNormalizeJudgeWeights_ActualConfig(t *testing.T) {
	// Verify the actual configs/judges.yaml parses correctly with scopes
	os.Setenv("JUDGES_CONFIG_PATH", "../../configs/judges.yaml")
	defer os.Unsetenv("JUDGES_CONFIG_PATH")

	cfg, err := LoadJudgesConfig()
	if err != nil {
		t.Fatalf("Failed to load actual judges config: %v", err)
	}

	turnCount := 0
	convCount := 0
	turnWeightSum := 0.0
	convWeightSum := 0.0

	for _, j := range cfg.Judges.Evaluators {
		if !j.Enabled {
			continue
		}
		scope := j.Scope
		if scope == "" {
			scope = "turn"
		}
		if scope == "turn" {
			turnCount++
			turnWeightSum += j.Weight
		} else if scope == "conversation" {
			convCount++
			convWeightSum += j.Weight
		}
	}

	if turnCount == 0 {
		t.Error("Expected at least one enabled turn-scoped judge")
	}
	if convCount == 0 {
		t.Error("Expected at least one enabled conversation-scoped judge")
	}

	const tol = 0.001
	if turnWeightSum < 1.0-tol || turnWeightSum > 1.0+tol {
		t.Errorf("turn judge weights should sum to 1.0, got %.4f (count=%d)", turnWeightSum, turnCount)
	}
	if convWeightSum < 1.0-tol || convWeightSum > 1.0+tol {
		t.Errorf("conversation judge weights should sum to 1.0, got %.4f (count=%d)", convWeightSum, convCount)
	}

	t.Logf("turn judges=%d (sum=%.4f), conversation judges=%d (sum=%.4f)",
		turnCount, turnWeightSum, convCount, convWeightSum)
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
