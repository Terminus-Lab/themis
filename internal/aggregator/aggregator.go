package aggregator

import (
	"math"
	"slices"

	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/rs/zerolog"
)

type AggregationConfig struct {
	EnablePrecheck         bool
	JudgeAggregationMethod models.AggregationMethod
}

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
	Config     AggregationConfig
	logger     *zerolog.Logger
}

func NewAggregator(weights Weights, thresholds VerdictThresholds, config AggregationConfig, logger *zerolog.Logger) *Aggregator {
	return &Aggregator{
		Weights:    weights,
		Thresholds: thresholds,
		Config:     config,
		logger:     logger,
	}
}

func (a *Aggregator) Aggregate(id string, stage1 []models.StageResult, stage2 []models.StageResult) models.EvaluationResult {
	metrics := models.AggregationMetrics{}

	result := models.EvaluationResult{
		ID:     id,
		Stages: append(stage1, stage2...),
	}

	if len(stage1) == 0 || len(stage2) == 0 {
		result.Verdict = models.VerdictFail
		return result
	}

	stage1Avg := 0.0
	// Stage 1: Simple average (prechecks have no weights)
	if a.Config.EnablePrecheck && len(stage1) > 0 {
		stage1Score := 0.0
		for _, stage := range stage1 {
			stage1Score += stage.Score
		}
		stage1Avg = stage1Score / float64(len(stage1))
	}
	metrics.Stage1Avg = stage1Avg

	// Compute all scores for defined AggregationMethod
	metrics.Stage2WeightedAvg = calculateWeightedAverage(stage2)
	metrics.Stage2HarmonicMean = calculateHarmonicMean(stage2)
	metrics.Stage2Median = calculateMedian(stage2)
	metrics.Stage2WeightedProduct = calculateWeightedProduct(stage2)

	var stage2Avg float64
	switch a.Config.JudgeAggregationMethod {
	case models.MethodWeightedAverage:
		stage2Avg = metrics.Stage2WeightedAvg
	case models.MethodHarmonicMean:
		stage2Avg = metrics.Stage2HarmonicMean
	case models.MethodMedian:
		stage2Avg = metrics.Stage2Median
	case models.MethodWeightedProduct:
		stage2Avg = metrics.Stage2WeightedProduct
	}
	metrics.MethodUsed = a.Config.JudgeAggregationMethod

	if a.Config.EnablePrecheck {
		metrics.FinalConfidence = (stage1Avg * a.Weights.PreChecks) + (stage2Avg * a.Weights.LLMJudge)
	} else {
		metrics.FinalConfidence = stage2Avg
	}

	result.Confidence = metrics.FinalConfidence
	result.Verdict = a.calculateVerdict(metrics.FinalConfidence)
	result.Metrics = metrics

	a.logger.
		Info().
		Float64("stage1_avg", stage1Avg).
		Float64("stage2_avg", stage2Avg).
		Float64("confidence", result.Confidence).
		Str("verdict", string(result.Verdict)).
		Msg("aggregation complete")
	return result
}

func calculateWeightedAverage(stages []models.StageResult) float64 {
	stageWeightedScore := 0.0
	stageTotalWeight := 0.0
	for _, stage := range stages {
		stageWeightedScore += stage.Score * stage.Weight
		stageTotalWeight += stage.Weight
	}

	// Use weighted average if weights are set, otherwise fall back to simple average
	stageAvg := 0.0
	if stageTotalWeight > 0 {
		stageAvg = stageWeightedScore / stageTotalWeight
	} else {
		// Fallback to simple average if no weights set
		stageScore := 0.0
		for _, stage := range stages {
			stageScore += stage.Score
		}
		stageAvg = stageScore / float64(len(stages))
	}
	return stageAvg
}

func calculateHarmonicMean(stages []models.StageResult) float64 {
	stageTotalWeight := 0.0
	stageSumWeightPerScore := 0.0

	for _, stage := range stages {
			if stage.Score == 0 {
					return 0  // If any score is 0, harmonic mean is 0
			}
			stageTotalWeight += stage.Weight
			stageSumWeightPerScore += stage.Weight / stage.Score
	}

	if stageSumWeightPerScore == 0 {
			return 0
	}

	return stageTotalWeight / stageSumWeightPerScore
}

func calculateMedian(stages []models.StageResult) float64 {
	scores := make([]float64, 0, len(stages))

	for _, stage := range stages {
		scores = append(scores, stage.Score)
	}

	slices.Sort(scores)
	mid := len(scores) / 2
	if len(scores)%2 == 0 {
		return (scores[mid-1] + scores[mid]) / 2
	}
	return scores[mid]
}

func calculateWeightedProduct(stages []models.StageResult) float64 {
	sumWeights := 0.0
	for _, stage := range stages {
		sumWeights += stage.Weight
	}

	result := 1.0
	for _, stage := range stages {
		p := stage.Weight / sumWeights
		result *= math.Pow(stage.Score, p)
	}

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
