package setup

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/Terminus-Lab/themis/internal/aggregator"
	"github.com/Terminus-Lab/themis/internal/config"
	"github.com/Terminus-Lab/themis/internal/executor"
	"github.com/Terminus-Lab/themis/internal/judge"
	"github.com/Terminus-Lab/themis/internal/llm"
	"github.com/Terminus-Lab/themis/internal/llm/aws"
	"github.com/Terminus-Lab/themis/internal/llm/azure"
	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/Terminus-Lab/themis/internal/prechecks"
	"github.com/rs/zerolog"
)

type Config struct {
	AWSRegion              string
	ClaudeModelID          string
	OpenAIKey              string
	OpenAIModelID          string
	AzureOpenAIEndpoint    string
	DefaultProvider        string
	EnablePrecheck         bool
	PrecheckWeight         float64
	LLMJudgeWeight         float64
	EarlyExitThreshold     float64
	VerdictPassThreshold   float64
	VerdictReviewThreshold float64
	JudgeAggregationMethod string
}

type Dependencies struct {
	Executor      *executor.Executor
	JudgeExecutor *executor.JudgeExecutor
	Logger        *zerolog.Logger
}

func LoadConfig() *Config {
	return &Config{
		AWSRegion:              getEnv("AWS_REGION", "us-east-1"),
		ClaudeModelID:          getEnv("CLAUDE_MODEL_ID", ""),
		OpenAIKey:              getEnv("OPEN_AI_KEY", ""),
		OpenAIModelID:          getEnv("OPEN_AI_MODEL_ID", ""),
		AzureOpenAIEndpoint:    getEnv("AZURE_OPENAI_ENDPOINT", ""),
		DefaultProvider:        getEnv("DEFAULT_LLM_PROVIDER", "bedrock"),
		EnablePrecheck:         getEnvBool("ENABLE_PRECHECK", true),
		PrecheckWeight:         getEnvFloat("PRECHECK_WEIGHT", 0.3),
		LLMJudgeWeight:         getEnvFloat("LLM_JUDGE_WEIGHT", 0.7),
		EarlyExitThreshold:     getEnvFloat("EARLY_EXIT_THRESHOLD", 0.2),
		VerdictPassThreshold:   getEnvFloat("VERDICT_PASS_THRESHOLD", 0.8),
		VerdictReviewThreshold: getEnvFloat("VERDICT_REVIEW_THRESHOLD", 0.5),
		JudgeAggregationMethod: getEnv("JUDGE_AGGREGATION_METHOD", ""),
	}
}

func Wire(ctx context.Context, cfg *Config, logger *zerolog.Logger) (*Dependencies, error) {
	// Load judges configuration from YAML first
	judgesConfig, err := config.LoadJudgesConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load judges config: %w", err)
	}

	// Create registry with all models referenced in judges config
	registry, err := createLLMClientRegistry(ctx, cfg, judgesConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client registry: %w", err)
	}

	// PreChecks
	stageRunner := prechecks.NewStageRunner([]prechecks.Checker{
		&prechecks.LengthChecker{},
		&prechecks.OverlapChecker{MinOverlapThreshold: 0.3},
		&prechecks.FormatChecker{},
	})

	// Create judge pool and build judges from config
	judgePool := judge.NewJudgePool(registry, logger)
	judges, err := judgePool.BuildFromConfig(judgesConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build judges from config: %w", err)
	}

	// Create judge runner with config-driven judges
	judgeRunner := judge.NewJudgeRunner(judges, logger)

	// Judge factory for single judge execution (reuses same judges)
	judgeFactory := judge.NewJudgeFactory(judges, logger)

	// Aggregator
	var judgeAggMethod models.AggregationMethod
	if cfg.JudgeAggregationMethod == "" {
		judgeAggMethod = models.MethodWeightedAverage
	} else {
		judgeAggMethod = models.AggregationMethod(cfg.JudgeAggregationMethod)
	}

	aggConfig := aggregator.AggregationConfig{
		EnablePrecheck:         cfg.EnablePrecheck,
		JudgeAggregationMethod: judgeAggMethod,
	}

	agg := aggregator.NewAggregator(
		aggregator.Weights{
			PreChecks: cfg.PrecheckWeight,
			LLMJudge:  cfg.LLMJudgeWeight,
		},
		aggregator.VerdictThresholds{
			Pass:   cfg.VerdictPassThreshold,
			Review: cfg.VerdictReviewThreshold,
		},
		aggConfig,
		logger,
	)

	// Executors
	agentExec := executor.NewExecutor(stageRunner, judgeRunner, agg, cfg.EarlyExitThreshold, logger)
	judgeExec := executor.NewJudgeExecutor(judgeFactory, logger)

	return &Dependencies{
		Executor:      agentExec,
		JudgeExecutor: judgeExec,
		Logger:        logger,
	}, nil

}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}

	return parsed
}

func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		value = defaultValue
	}

	return value
}

func getEnvFloat(key string, defaultValue float64) float64 {
	valueStr := os.Getenv(key)
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		value = defaultValue
	}

	return value
}

func createLLMClientRegistry(ctx context.Context, cfg *Config, judgesConfig *config.JudgesConfig) (*llm.LLMClientRegistry, error) {
	clients := make(map[llm.LLMFamily]map[string]llm.LLMClient)

	// Extract all unique models from judges config
	type modelKey struct {
		family  string
		modelID string
	}
	uniqueModels := make(map[modelKey]bool)

	for _, evaluator := range judgesConfig.Judges.Evaluators {
		if evaluator.Model != nil && evaluator.Model.ModelFamily != "" && evaluator.Model.ModelID != "" {
			uniqueModels[modelKey{evaluator.Model.ModelFamily, evaluator.Model.ModelID}] = true
		}
	}

	// Create clients for each unique model
	for model := range uniqueModels {
		family := llm.LLMFamily(model.family)

		switch family {
		case llm.FamilyAnthropic:
			if cfg.AWSRegion == "" {
				return nil, fmt.Errorf("AWS_REGION required for anthropic model %s", model.modelID)
			}
			client, err := aws.NewClient(ctx, cfg.AWSRegion, model.modelID)
			if err != nil {
				return nil, fmt.Errorf("failed to create Bedrock client for model %s: %w", model.modelID, err)
			}
			if clients[family] == nil {
				clients[family] = make(map[string]llm.LLMClient)
			}
			clients[family][model.modelID] = client

		case llm.FamilyOpenAI:
			if cfg.OpenAIKey == "" {
				return nil, fmt.Errorf("OPEN_AI_KEY required for openai model %s", model.modelID)
			}
			if cfg.AzureOpenAIEndpoint == "" {
				return nil, fmt.Errorf("AZURE_OPENAI_ENDPOINT required for openai model %s", model.modelID)
			}
			client, err := azure.NewClient(cfg.OpenAIKey, model.modelID, cfg.AzureOpenAIEndpoint)
			if err != nil {
				return nil, fmt.Errorf("failed to create Azure OpenAI client for model %s: %w", model.modelID, err)
			}
			if clients[family] == nil {
				clients[family] = make(map[string]llm.LLMClient)
			}
			clients[family][model.modelID] = client

		default:
			return nil, fmt.Errorf("unsupported model family: %s", model.family)
		}
	}

	if len(clients) == 0 {
		return nil, fmt.Errorf("no LLM clients configured - check judges.yaml has valid models with modelFamily and modelID")
	}

	return llm.NewLLMClientRegistry(clients), nil
}
