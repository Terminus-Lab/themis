package setup

import (
	"context"
	"fmt"

	"github.com/Terminus-Lab/themis/internal/config"
	"github.com/Terminus-Lab/themis/internal/env"
	"github.com/Terminus-Lab/themis/internal/executor"
	"github.com/Terminus-Lab/themis/internal/judge"
	"github.com/Terminus-Lab/themis/internal/llm"
	"github.com/Terminus-Lab/themis/internal/llm/aws"
	"github.com/Terminus-Lab/themis/internal/llm/azure"
	"github.com/Terminus-Lab/themis/internal/llm/openaiplatform"
	"github.com/Terminus-Lab/themis/internal/storage"
	"github.com/Terminus-Lab/themis/internal/storage/postgres"
	"github.com/Terminus-Lab/themis/internal/storage/sqlite"
	"github.com/rs/zerolog"
)

type Config struct {
	AWSRegion              string
	OpenAIKey              string
	AzureOpenAIEndpoint    string
	HolisticWeight         float64
	VerdictPassThreshold   float64
	VerdictReviewThreshold float64
	ScoringFormula         string
	InMemoryDB             bool
	DBConnectionString     string
}

type Dependencies struct {
	ConversationEvaluator *executor.ConversationEvaluator
	Repository            storage.Repository
	Logger                *zerolog.Logger
}

func LoadConfig() *Config {
	return &Config{
		AWSRegion:              env.GetString("AWS_REGION", "us-east-1"),
		OpenAIKey:              env.GetString("OPEN_AI_KEY", ""),
		AzureOpenAIEndpoint:    env.GetString("AZURE_OPENAI_ENDPOINT", ""),
		HolisticWeight:         env.GetFloat("CONVERSATION_HOLISTIC_WEIGHT", 0.5),
		VerdictPassThreshold:   env.GetFloat("VERDICT_PASS_THRESHOLD", 0.8),
		VerdictReviewThreshold: env.GetFloat("VERDICT_REVIEW_THRESHOLD", 0.5),
		ScoringFormula:         env.GetString("SCORING_FORMULA", "linear"),
		InMemoryDB:             env.GetBool("IN_MEMORY_DB", true),
		DBConnectionString:     env.GetString("THEMIS_DB_URL", ""),
	}
}

func Wire(ctx context.Context, cfg *Config, logger *zerolog.Logger) (*Dependencies, error) {
	// Load judges configuration
	judgesConfig, err := config.LoadJudgesConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load judges config: %w", err)
	}

	// Create LLM client registry
	registry, err := createLLMClientRegistry(ctx, cfg, judgesConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client registry: %w", err)
	}

	judgePool := judge.NewJudgePool(registry, logger)

	// Build per-turn judges (scope: "turn")
	turnJudges, err := judgePool.BuildTurnJudgesFromConfig(judgesConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build turn judges: %w", err)
	}
	turnRunner := judge.NewJudgeRunner(turnJudges, logger)

	// Build holistic judge (scope: "conversation")
	holisticJudge, err := judgePool.BuildHolisticJudgeFromConfig(judgesConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build holistic judge: %w", err)
	}

	// Database
	db, err := getDatabaseClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create db client: %w", err)
	}

	repository, err := NewEvalRepository(db, logger)
	if err != nil {
		return nil, fmt.Errorf("unable to get evaluation repository: %w", err)
	}

	logger.Info().Str("scoring_formula", cfg.ScoringFormula).Msg("active scoring formula")

	convEvaluator := executor.NewConversationEvaluator(
		turnRunner,
		holisticJudge,
		repository,
		cfg.HolisticWeight,
		cfg.VerdictPassThreshold,
		cfg.VerdictReviewThreshold,
		cfg.ScoringFormula,
		logger,
	)

	return &Dependencies{
		ConversationEvaluator: convEvaluator,
		Repository:            repository,
		Logger:                logger,
	}, nil
}

func createLLMClientRegistry(ctx context.Context, cfg *Config, judgesConfig *config.JudgesConfig) (*llm.LLMClientRegistry, error) {
	clients := make(map[llm.LLMFamily]map[string]llm.LLMClient)

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

		case llm.FamilyOpenAIPlatform:
			if cfg.OpenAIKey == "" {
				return nil, fmt.Errorf("OPEN_AI_KEY required for openai_platform model %s", model.modelID)
			}
			client, err := openaiplatform.NewClient(ctx, cfg.OpenAIKey, model.modelID)
			if err != nil {
				return nil, fmt.Errorf("failed to create OpenAI Platform client for model %s: %w", model.modelID, err)
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

func getDatabaseClient(ctx context.Context, cfg *Config) (storage.DB, error) {
	if cfg.InMemoryDB {
		client, err := sqlite.New(ctx, ":memory:")
		if err != nil {
			return nil, err
		}
		if err := client.InitSchema(ctx); err != nil {
			return nil, fmt.Errorf("failed to initialize SQLite schema: %w", err)
		}
		return client, nil
	}

	client, err := postgres.New(ctx, cfg.DBConnectionString)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func NewEvalRepository(db storage.DB, logger *zerolog.Logger) (storage.Repository, error) {
	switch d := db.(type) {
	case *postgres.DB:
		return postgres.NewEvalRepository(d, logger), nil
	case *sqlite.DB:
		return sqlite.NewEvalRepository(d, logger), nil
	default:
		return nil, fmt.Errorf("unsupported db type %T", db)
	}
}
