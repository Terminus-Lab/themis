package judge

import (
	"fmt"

	"github.com/Terminus-Lab/themis/internal/config"
	"github.com/Terminus-Lab/themis/internal/llm"
	"github.com/rs/zerolog"
)

// JudgePool builds and manages a collection of judges from configuration
type JudgePool struct {
	registry *llm.LLMClientRegistry
	logger   *zerolog.Logger
}

// NewJudgePool creates a new judge pool builder
func NewJudgePool(registry *llm.LLMClientRegistry, logger *zerolog.Logger) *JudgePool {
	return &JudgePool{
		registry: registry,
		logger:   logger,
	}
}

func (p *JudgePool) BuildFromConfig(cfg *config.JudgesConfig) ([]Judge, error) {
	if cfg == nil {
		return nil, fmt.Errorf("judges config is nil")
	}

	var judges []Judge

	for _, judgeCfg := range cfg.Judges.Evaluators {
		// Skip disabled judges
		if !judgeCfg.Enabled {
			p.logger.Info().
				Str("judge", judgeCfg.Name).
				Msg("judge disabled in config, skipping")
			continue
		}

		// Get LLM client from registry based on judge's model family
		family := llm.LLMFamily(judgeCfg.Model.ModelFamily)
		llmClient, err := p.registry.Get(family, judgeCfg.Model.ModelID)
		if err != nil {
			return nil, fmt.Errorf("failed to get LLM client for judge %s (family=%s, model=%s): %w",
				judgeCfg.Name, judgeCfg.Model.ModelFamily, judgeCfg.Model.ModelID, err)
		}

		// Create LLM judge
		judge, err := NewLLMJudge(judgeCfg, llmClient, p.logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create judge %s: %w", judgeCfg.Name, err)
		}

		judges = append(judges, judge)

		p.logger.Info().
			Str("judge", judgeCfg.Name).
			Str("model_family", judgeCfg.Model.ModelFamily).
			Str("model_id", judgeCfg.Model.ModelID).
			Int("max_tokens", judgeCfg.Model.MaxTokens).
			Float64("temperature", judgeCfg.Model.Temperature).
			Bool("retry", judgeCfg.Model.Retry).
			Bool("requires_context", judgeCfg.RequiresContext).
			Msg("judge created successfully")
	}

	if len(judges) == 0 {
		return nil, fmt.Errorf("no enabled judges found in config")
	}

	p.logger.Info().
		Int("total_judges", len(judges)).
		Msg("judge pool built successfully")

	return judges, nil
}
