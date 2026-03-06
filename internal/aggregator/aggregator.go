package aggregator

import (
	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/rs/zerolog"
)

type Weights struct {
	PreChecks float64
	LLMJudge  float64
}

type VerdictThresholds struct {
	Pass   float64 // Confidence > Pass → "pass"
	Review float64 // Confidence > Review → "review", else "fail"
}

type Aggregator struct {
	Weights    Weights
	Thresholds VerdictThresholds
	logger     *zerolog.Logger
}

func NewAggregator(weights Weights, thresholds VerdictThresholds, logger *zerolog.Logger) *Aggregator {
	return &Aggregator{
		Weights:    weights,
		Thresholds: thresholds,
		logger:     logger,
	}
}

func (a *Aggregator) Aggregate(id string, stage1 []models.StageResult, stage2 []models.StageResult) models.EvaluationResult {
	result := models.EvaluationResult{
		ID:     id,
		Stages: append(stage1, stage2...),
	}

	if len(stage1) == 0 || len(stage2) == 0 {
		result.Verdict = models.VerdictFail
		return result
	}

	// Stage 1: Simple average (prechecks have no weights)
	stage1Score := 0.0
	for _, stage := range stage1 {
		stage1Score += stage.Score
	}
	stage1Avg := stage1Score / float64(len(stage1))

	// Stage 2: Weighted average (LLM judges have per-judge weights)
	stage2WeightedScore := 0.0
	stage2TotalWeight := 0.0
	for _, stage := range stage2 {
		stage2WeightedScore += stage.Score * stage.Weight
		stage2TotalWeight += stage.Weight
	}

	// Use weighted average if weights are set, otherwise fall back to simple average
	stage2Avg := 0.0
	if stage2TotalWeight > 0 {
		stage2Avg = stage2WeightedScore / stage2TotalWeight
	} else {
		// Fallback to simple average if no weights set
		stage2Score := 0.0
		for _, stage := range stage2 {
			stage2Score += stage.Score
		}
		stage2Avg = stage2Score / float64(len(stage2))
	}

	confidence := (stage1Avg * a.Weights.PreChecks) + (stage2Avg * a.Weights.LLMJudge)

	result.Confidence = confidence
	result.Verdict = a.calculateVerdict(confidence)

	a.logger.
		Info().
		Float64("stage1_avg", stage1Avg).
		Float64("stage2_weighted_avg", stage2Avg).
		Float64("confidence", confidence).
		Str("verdict", string(result.Verdict)).
		Msg("aggregation complete")
	return result
}

func (a *Aggregator) calculateVerdict(confidence float64) models.Verdict {
	if confidence > a.Thresholds.Pass {
		return models.VerdictPass
	}
	if confidence > a.Thresholds.Review {
		return models.VerdictReview
	}
	return models.VerdictFail
}
