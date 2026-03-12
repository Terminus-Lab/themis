package executor

import (
	"context"

	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/Terminus-Lab/themis/internal/storage"
	"github.com/rs/zerolog"
)

// PrecheckRunner runs precheck stage evaluations
type PrecheckRunner interface {
	Run(evalCtx models.EvaluationContext) []models.StageResult
}

// JudgeRunner runs LLM judge evaluations
type JudgeRunner interface {
	Run(ctx context.Context, evalCtx models.EvaluationContext) []models.StageResult
}

// Aggregator aggregates stage results into final evaluation
type Aggregator interface {
	Aggregate(id string, stage1 []models.StageResult, stage2 []models.StageResult) models.EvaluationResult
}

type Executor struct {
	precheckStageRunner PrecheckRunner
	repository          storage.Repository
	judgeRunner         JudgeRunner
	aggregator          Aggregator
	earlyExitThreshold  float64
	logger              *zerolog.Logger
}

func NewExecutor(
	prechecks PrecheckRunner,
	repository storage.Repository,
	judgeRunner JudgeRunner,
	aggregator Aggregator,
	earlyExitThreshold float64,
	logger *zerolog.Logger,
) *Executor {
	return &Executor{
		precheckStageRunner: prechecks,
		repository:          repository,
		judgeRunner:         judgeRunner,
		aggregator:          aggregator,
		earlyExitThreshold:  earlyExitThreshold,
		logger:              logger,
	}
}

func (e *Executor) Execute(ctx context.Context, evalCtx models.EvaluationContext) models.EvaluationResult {
	id := evalCtx.RequestID
	e.logger.Info().Str("requestID", id).Msg("starting evaluation")

	result := models.EvaluationResult{
		ID:             id,
		ConversationID: evalCtx.ConversationID,
		Stages:         []models.StageResult{},
		Confidence:     0,
		Verdict:        "",
	}

	stageEvalResults := e.precheckStageRunner.Run(evalCtx)

	if len(stageEvalResults) == 0 {
		result.Verdict = models.VerdictFail
		return result
	}

	stageEvalScore := 0.0
	for _, stageEval := range stageEvalResults {
		stageEvalScore += stageEval.Score
	}

	stageEvalAvgScore := stageEvalScore / float64(len(stageEvalResults))

	if stageEvalAvgScore < e.earlyExitThreshold {
		result.Stages = append(result.Stages, stageEvalResults...)
		result.Verdict = models.VerdictFail
		result.Confidence = stageEvalAvgScore // Set confidence for early exit
		e.logger.Info().Float64("avgScore", stageEvalAvgScore).Msg("early exit triggered")

		// Store early exit result
		evaluationResult := storage.Evaluation{
			EventID:        result.ID,
			ConversationID: result.ConversationID,
			AgentName:      evalCtx.AgentName,
			AgentVersion:   evalCtx.AgentVersion,
			UserQuery:      evalCtx.Query,
			Answer:         evalCtx.Answer,
			Context:        evalCtx.Context,
			Confidence:     result.Confidence,
			Verdict:        string(result.Verdict),
			StageScores:    stageEvalResults,
		}

		err := e.repository.Store(ctx, &evaluationResult)
		if err != nil {
			e.logger.Error().Err(err).Msg("unable to store early exit evaluation result")
		}

		return result
	}

	judgeEvaResults := e.judgeRunner.Run(ctx, evalCtx)

	finalResult := e.aggregator.Aggregate(id, stageEvalResults, judgeEvaResults)
	finalResult.ConversationID = result.ConversationID

	evaluationResult := storage.Evaluation{
		EventID:        finalResult.ID,
		ConversationID: result.ConversationID,
		AgentName:      evalCtx.AgentName,
		AgentVersion:   evalCtx.AgentVersion,
		UserQuery:      evalCtx.Query,
		Answer:         evalCtx.Answer,
		Context:        evalCtx.Context,
		Confidence:     finalResult.Confidence,
		Verdict:        string(finalResult.Verdict),
		StageScores:    judgeEvaResults,
	}

	err := e.repository.Store(ctx, &evaluationResult)
	if err != nil {
		e.logger.Error().Err(err).Msg("unable to store evaluation result in the repository")
		return finalResult
	}

	e.logger.
		Info().
		Str("verdict", string(finalResult.Verdict)).
		Float64("confidence", result.Confidence).
		Msg("evaluation complete")
	return finalResult
}
