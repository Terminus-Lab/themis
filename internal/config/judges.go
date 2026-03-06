package config

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// JudgesConfig is the root configuration structure
type JudgesConfig struct {
	Judges Judges `yaml:"judges"`
}

// Judges contains default model config and list of evaluators
type Judges struct {
	DefaultModel ModelConfig          `yaml:"default_model"`
	Evaluators   []JudgeConfiguration `yaml:"evaluators"`
}

// JudgeConfiguration defines a single judge configuration
type JudgeConfiguration struct {
	Name                   string       `yaml:"name"`
	Enabled                bool         `yaml:"enabled"`
	Description            string       `yaml:"description"`
	RequiresContext        bool         `yaml:"requires_context"`
	RequiresExpectedOutput bool         `yaml:"requires_expected_output"` // For correctness evaluation
	Prompt                 string       `yaml:"prompt"`
	Model                  *ModelConfig `yaml:"model,omitempty"` // Optional override
	Weight                 float64      `yaml:"weight,omitempty"` // Weight for this judge (0.0-1.0)
}

// ModelConfig defines LLM model parameters
type ModelConfig struct {
	ModelID     string  `yaml:"modelID,omitempty"`
	ModelFamily string  `yaml:"modelFamily,omitempty"`
	MaxTokens   int     `yaml:"max_tokens,omitempty"`
	Temperature float64 `yaml:"temperature,omitempty"`
	Retry       bool    `yaml:"retry,omitempty"`
}

// LoadJudgesConfig loads and validates the judges configuration from YAML
func LoadJudgesConfig() (*JudgesConfig, error) {
	path := os.Getenv("JUDGES_CONFIG_PATH")
	if path == "" {
		path = "configs/judges.yaml"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg JudgesConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	applyDefaults(&cfg)

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

func applyDefaults(cfg *JudgesConfig) {
	if cfg.Judges.DefaultModel.MaxTokens == 0 {
		cfg.Judges.DefaultModel.MaxTokens = 256
	}
	if cfg.Judges.DefaultModel.Temperature == 0.0 {
		cfg.Judges.DefaultModel.Temperature = 0.0
	}

	// Set the default model family
	if cfg.Judges.DefaultModel.ModelFamily == "" {
		cfg.Judges.DefaultModel.ModelFamily = os.Getenv("DEFAULT_MODEL_FAMILY")
	}

	if cfg.Judges.DefaultModel.ModelID == "" {
		cfg.Judges.DefaultModel.ModelID = os.Getenv("DEFAULT_MODEL_ID")
	}

	// For each judge, apply defaults
	for i := range cfg.Judges.Evaluators {
		judge := &cfg.Judges.Evaluators[i]

		if judge.Model == nil {
			judge.Model = &ModelConfig{
				MaxTokens:   cfg.Judges.DefaultModel.MaxTokens,
				Temperature: cfg.Judges.DefaultModel.Temperature,
				Retry:       cfg.Judges.DefaultModel.Retry,
				ModelID:     cfg.Judges.DefaultModel.ModelID,
				ModelFamily: cfg.Judges.DefaultModel.ModelFamily,
			}
		} else {
			if judge.Model.MaxTokens == 0 {
				judge.Model.MaxTokens = cfg.Judges.DefaultModel.MaxTokens
			}
			if judge.Model.Temperature == 0.0 {
				judge.Model.Temperature = cfg.Judges.DefaultModel.Temperature
			}
			if judge.Model.ModelID == "" {
				judge.Model.ModelID = cfg.Judges.DefaultModel.ModelID
			}
			if judge.Model.ModelFamily == "" {
				judge.Model.ModelFamily = cfg.Judges.DefaultModel.ModelFamily
			}
		}
	}

	// Normalize weights: if no weights set or don't sum to 1.0, distribute equally
	normalizeJudgeWeights(cfg)
}

func normalizeJudgeWeights(cfg *JudgesConfig) {
	// Count enabled judges and sum existing weights
	enabledCount := 0
	weightSum := 0.0
	hasAnyWeight := false

	for i := range cfg.Judges.Evaluators {
		judge := &cfg.Judges.Evaluators[i]
		if judge.Enabled {
			enabledCount++
			if judge.Weight > 0 {
				hasAnyWeight = true
				weightSum += judge.Weight
			}
		}
	}

	if enabledCount == 0 {
		return
	}

	// If no weights specified, distribute equally
	if !hasAnyWeight {
		defaultWeight := 1.0 / float64(enabledCount)
		for i := range cfg.Judges.Evaluators {
			judge := &cfg.Judges.Evaluators[i]
			if judge.Enabled {
				judge.Weight = defaultWeight
			}
		}
		return
	}

	// If weights don't sum to ~1.0, normalize them
	const tolerance = 0.001
	if weightSum < (1.0-tolerance) || weightSum > (1.0+tolerance) {
		for i := range cfg.Judges.Evaluators {
			judge := &cfg.Judges.Evaluators[i]
			if judge.Enabled && judge.Weight > 0 {
				judge.Weight = judge.Weight / weightSum
			}
		}
	}
}

func (cfg *JudgesConfig) Validate() error {
	if len(cfg.Judges.Evaluators) == 0 {
		return fmt.Errorf("no judges configured in evaluators list")
	}

	seen := make(map[string]bool)

	for i, judge := range cfg.Judges.Evaluators {
		if judge.Name == "" {
			return fmt.Errorf("judge at index %d is missing name", i)
		}

		if seen[judge.Name] {
			return fmt.Errorf("duplicate judge name: %s", judge.Name)
		}
		seen[judge.Name] = true

		if judge.Prompt == "" {
			return fmt.Errorf("judge %s is missing prompt", judge.Name)
		}

		if _, err := template.New(judge.Name).Parse(judge.Prompt); err != nil {
			return fmt.Errorf("judge %s has invalid prompt template: %w", judge.Name, err)
		}

		// Validate that required fields are referenced in the prompt
		if judge.RequiresContext && !strings.Contains(judge.Prompt, "{{.Context}}") {
			return fmt.Errorf("judge %s requires context but prompt does not reference {{.Context}}", judge.Name)
		}

		if judge.RequiresExpectedOutput && !strings.Contains(judge.Prompt, "{{.ExpectedOutput}}") {
			return fmt.Errorf("judge %s requires expected_output but prompt does not reference {{.ExpectedOutput}}", judge.Name)
		}

		if judge.Model != nil {
			if judge.Model.MaxTokens < 0 {
				return fmt.Errorf("judge %s has negative max_tokens: %d", judge.Name, judge.Model.MaxTokens)
			}
			if judge.Model.Temperature < 0.0 || judge.Model.Temperature > 1.0 {
				return fmt.Errorf("judge %s has invalid temperature: %f (must be 0.0-1.0)", judge.Name, judge.Model.Temperature)
			}
		}

		// Validate weight (should be set by applyDefaults)
		if judge.Enabled && (judge.Weight < 0.0 || judge.Weight > 1.0) {
			return fmt.Errorf("judge %s has invalid weight: %f (must be 0.0-1.0)", judge.Name, judge.Weight)
		}
	}

	if cfg.Judges.DefaultModel.MaxTokens < 0 {
		return fmt.Errorf("default model has negative max_tokens: %d", cfg.Judges.DefaultModel.MaxTokens)
	}
	if cfg.Judges.DefaultModel.Temperature < 0.0 || cfg.Judges.DefaultModel.Temperature > 1.0 {
		return fmt.Errorf("default model has invalid temperature: %f (must be 0.0-1.0)", cfg.Judges.DefaultModel.Temperature)
	}

	return nil
}
