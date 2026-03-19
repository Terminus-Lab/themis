package executor

import (
	"context"
	"time"

	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/Terminus-Lab/themis/internal/storage"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ConversationExecutor runs conversation-scoped judges over all turns of a conversation.
type ConversationExecutor struct {
	judgeRunner JudgeRunner
	repository  storage.Repository
	aggregator  Aggregator
	logger      *zerolog.Logger
}

func NewConversationExecutor(
	judgeRunner JudgeRunner,
	repository storage.Repository,
	aggregator Aggregator,
	logger *zerolog.Logger,
) *ConversationExecutor {
	return &ConversationExecutor{
		judgeRunner: judgeRunner,
		repository:  repository,
		aggregator:  aggregator,
		logger:      logger,
	}
}

// Execute evaluates a full conversation using conversation-scoped judges.
func (e *ConversationExecutor) Execute(ctx context.Context, req models.ConversationEvaluationRequest) models.ConversationEvaluationResult {
	id := uuid.New().String()

	e.logger.Info().
		Str("conversation_id", req.ConversationID).
		Int("turn_count", len(req.Turns)).
		Msg("starting conversation evaluation")

	result := models.ConversationEvaluationResult{
		ConversationID: req.ConversationID,
		AgentName:      req.Agent.Name,
		AgentVersion:   req.Agent.Version,
		TurnCount:      len(req.Turns),
		Stages:         []models.StageResult{},
	}

	// Build EvaluationContext with turns populated for conversation judges
	evalCtx := models.EvaluationContext{
		RequestID:      id,
		ConversationID: req.ConversationID,
		AgentName:      req.Agent.Name,
		AgentVersion:   req.Agent.Version,
		Turns:          req.Turns,
		CreatedAt:      time.Now(),
	}

	judgeResults := e.judgeRunner.Run(ctx, evalCtx)

	// Aggregate (use stage1=nil since there are no prechecks for conversation eval)
	aggregated := e.aggregator.Aggregate(id, nil, judgeResults)

	result.Stages = judgeResults
	result.Verdict = aggregated.Verdict
	result.Confidence = aggregated.Confidence

	// Store in conversation_eval_results table
	eval := &storage.ConversationEvaluation{
		ID:             id,
		ConversationID: req.ConversationID,
		AgentName:      req.Agent.Name,
		AgentVersion:   req.Agent.Version,
		TurnCount:      len(req.Turns),
		Confidence:     result.Confidence,
		Verdict:        string(result.Verdict),
		StageScores:    judgeResults,
	}

	if err := e.repository.StoreConversationEval(ctx, eval); err != nil {
		e.logger.Error().Err(err).Msg("unable to store conversation evaluation result")
	}

	e.logger.Info().
		Str("conversation_id", req.ConversationID).
		Str("verdict", string(result.Verdict)).
		Float64("confidence", result.Confidence).
		Msg("conversation evaluation complete")

	return result
}
