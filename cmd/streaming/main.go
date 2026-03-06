package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/Terminus-Lab/themis/internal/aggregator"
	"github.com/Terminus-Lab/themis/internal/config"
	"github.com/Terminus-Lab/themis/internal/executor"
	"github.com/Terminus-Lab/themis/internal/judge"
	"github.com/Terminus-Lab/themis/internal/llm"
	"github.com/Terminus-Lab/themis/internal/llm/aws"
	"github.com/Terminus-Lab/themis/internal/prechecks"
	"github.com/Terminus-Lab/themis/internal/stream"
	"github.com/Terminus-Lab/themis/internal/stream/redis"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	logger := log.Logger

	// Load env
	if err := godotenv.Load(); err != nil {
		log.Warn().Msg("No .env file found")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Initialize AWS Bedrock client
	region := os.Getenv("AWS_REGION")
	modelID := os.Getenv("DEFAULT_MODEL_ID")
	modelFamily := os.Getenv("DEFAULT_MODEL_FAMILY")
	if modelFamily == "" {
		modelFamily = "anthropic"
	}

	awsClient, err := aws.NewClient(ctx, region, modelID)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create AWS Bedrock client")
	}

	// Create LLM client registry
	registry := llm.NewLLMClientRegistry(map[llm.LLMFamily]map[string]llm.LLMClient{
		llm.LLMFamily(modelFamily): {
			modelID: awsClient,
		},
	})

	// Redis client
	streamCfg := &stream.StreamConfig{
		Provider: os.Getenv("STREAM_PROVIDER"),
		RedisConfig: redis.NewRedisStreamConfig(
			os.Getenv("REDIS_ADDR"),
			os.Getenv("REDIS_PASSWORD"),
			"eval-events",
			"eval-group",
			os.Getenv("HOSTNAME"),
		),
	}

	// Aggregator weights
	precheckWeight, err := strconv.ParseFloat(os.Getenv("PRECHECK_WEIGHT"), 64)
	if err != nil {
		precheckWeight = 0.3
	}
	llmJudgeWeight, err := strconv.ParseFloat(os.Getenv("LLM_JUDGE_WEIGHT"), 64)
	if err != nil {
		llmJudgeWeight = 0.7
	}

	// Wire Components
	// Stage 1 — PreChecks
	stageRunner := prechecks.NewStageRunner([]prechecks.Checker{
		&prechecks.LengthChecker{},
		&prechecks.OverlapChecker{MinOverlapThreshold: 0.3},
		&prechecks.FormatChecker{},
	})

	// Stage 2 — LLM Judges (from YAML config)
	judgesConfig, err := config.LoadJudgesConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load judges config")
	}

	judgePool := judge.NewJudgePool(registry, &logger)
	judges, err := judgePool.BuildFromConfig(judgesConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to build judges from config")
	}

	judgeRunner := judge.NewJudgeRunner(judges, &logger)

	// Aggregator
	verdictPassThreshold, _ := strconv.ParseFloat(os.Getenv("VERDICT_PASS_THRESHOLD"), 64)
	if verdictPassThreshold == 0 {
		verdictPassThreshold = 0.8
	}
	verdictReviewThreshold, _ := strconv.ParseFloat(os.Getenv("VERDICT_REVIEW_THRESHOLD"), 64)
	if verdictReviewThreshold == 0 {
		verdictReviewThreshold = 0.5
	}
	agg := aggregator.NewAggregator(
		aggregator.Weights{
			PreChecks: precheckWeight,
			LLMJudge:  llmJudgeWeight,
		},
		aggregator.VerdictThresholds{
			Pass:   verdictPassThreshold,
			Review: verdictReviewThreshold,
		},
		&logger,
	)

	// Executor
	earlyExit, _ := strconv.ParseFloat(os.Getenv("EARLY_EXIT_THRESHOLD"), 64)
	if earlyExit == 0 {
		earlyExit = 0.2
	}
	exec := executor.NewExecutor(stageRunner, judgeRunner, agg, earlyExit, &logger)

	consumer, err := stream.NewStreamConsumer(ctx, streamCfg, exec, &logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create stream consumer")
	}

	// Setup consumer
	err = consumer.Setup(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to setup consumer")
	}

	// Start consumer
	go func() {
		if err := consumer.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error().Err(err).Msg("Consumer stopped with error")
		}
	}()

	// Wait for context to be done
	<-ctx.Done()
	logger.Info().Msg("Shutting down...")

	log.Info().Msg("Eval Agent stopped")
}
