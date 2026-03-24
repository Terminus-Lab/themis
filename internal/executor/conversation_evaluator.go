package executor

import (
	"context"
	"sync"
	"time"

	"github.com/Terminus-Lab/themis/internal/judge"
	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/Terminus-Lab/themis/internal/storage"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ConversationEvaluator runs a two-phase evaluation on a full conversation:
//
//	Phase A: per-turn judges (relevance, coherence, completeness) → turn_avg
//	Phase B: holistic judge over all turns → holistic_score
//	final_score = α × holistic_score + (1-α) × turn_avg
type ConversationEvaluator struct {
	turnRunner    *judge.JudgeRunner
	holisticJudge judge.Judge
	repository    storage.Repository
	holisticWeight float64 // α — weight for holistic score (0.0–1.0)
	passThreshold  float64
	reviewThreshold float64
	logger         *zerolog.Logger
}

func NewConversationEvaluator(
	turnRunner *judge.JudgeRunner,
	holisticJudge judge.Judge,
	repository storage.Repository,
	holisticWeight float64,
	passThreshold float64,
	reviewThreshold float64,
	logger *zerolog.Logger,
) *ConversationEvaluator {
	return &ConversationEvaluator{
		turnRunner:      turnRunner,
		holisticJudge:   holisticJudge,
		repository:      repository,
		holisticWeight:  holisticWeight,
		passThreshold:   passThreshold,
		reviewThreshold: reviewThreshold,
		logger:          logger,
	}
}

// Execute evaluates a full conversation and returns the result.
func (e *ConversationEvaluator) Execute(ctx context.Context, req models.ConversationEvaluationRequest) models.ConversationEvaluationResult {
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
		TurnResults:    []models.TurnEvaluationResult{},
	}

	// === Phase A: evaluate each turn in parallel ===
	turnResults := e.evaluateTurns(ctx, req)
	result.TurnResults = turnResults

	// Compute turn_avg (simple average of all turn scores)
	turnAvg := 0.0
	if len(turnResults) > 0 {
		total := 0.0
		for _, tr := range turnResults {
			total += tr.TurnScore
		}
		turnAvg = total / float64(len(turnResults))
	}
	result.TurnAvg = turnAvg

	// === Phase B: holistic judge over full conversation ===
	holisticCtx := models.EvaluationContext{
		RequestID:      id,
		ConversationID: req.ConversationID,
		AgentName:      req.Agent.Name,
		AgentVersion:   req.Agent.Version,
		Turns:          req.Turns,
		CreatedAt:      time.Now(),
	}

	judgeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	holisticResult := e.holisticJudge.Evaluate(judgeCtx, holisticCtx)
	result.HolisticScore = holisticResult.Score
	result.HolisticReason = holisticResult.Reason

	// === Final score ===
	α := e.holisticWeight
	result.FinalScore = α*result.HolisticScore + (1-α)*result.TurnAvg
	result.Verdict = e.verdict(result.FinalScore)

	// === Persist ===
	record := &storage.ConversationRecord{
		ID:             id,
		ConversationID: req.ConversationID,
		AgentName:      req.Agent.Name,
		AgentVersion:   req.Agent.Version,
		TurnCount:      len(req.Turns),
		TurnAvg:        result.TurnAvg,
		HolisticScore:  result.HolisticScore,
		HolisticReason: result.HolisticReason,
		FinalScore:     result.FinalScore,
		Verdict:        string(result.Verdict),
		TurnResults:    result.TurnResults,
	}
	if err := e.repository.StoreConversation(ctx, record); err != nil {
		e.logger.Error().Err(err).Msg("unable to store conversation evaluation result")
	}

	e.logger.Info().
		Str("conversation_id", req.ConversationID).
		Float64("turn_avg", result.TurnAvg).
		Float64("holistic_score", result.HolisticScore).
		Float64("final_score", result.FinalScore).
		Str("verdict", string(result.Verdict)).
		Msg("conversation evaluation complete")

	return result
}

// evaluateTurns runs per-turn judges on each turn concurrently.
func (e *ConversationEvaluator) evaluateTurns(ctx context.Context, req models.ConversationEvaluationRequest) []models.TurnEvaluationResult {
	type indexedResult struct {
		index int
		result models.TurnEvaluationResult
	}

	resultsCh := make(chan indexedResult, len(req.Turns))
	var wg sync.WaitGroup

	for i, turn := range req.Turns {
		wg.Add(1)
		go func(idx int, t models.ConversationTurn) {
			defer wg.Done()

			evalCtx := models.EvaluationContext{
				RequestID:      uuid.New().String(),
				ConversationID: req.ConversationID,
				AgentName:      req.Agent.Name,
				AgentVersion:   req.Agent.Version,
				Query:          t.UserQuery,
				Answer:         t.Answer,
				Context:        t.Context,
				ExpectedOutput: t.ExpectedOutput,
				CreatedAt:      time.Now(),
			}

			scores := e.turnRunner.Run(ctx, evalCtx)

			// Compute weighted average for this turn
			turnScore := weightedAverage(scores)

			resultsCh <- indexedResult{
				index: idx,
				result: models.TurnEvaluationResult{
					TurnIndex: t.TurnIndex,
					UserQuery: t.UserQuery,
					Answer:    t.Answer,
					Scores:    scores,
					TurnScore: turnScore,
				},
			}
		}(i, turn)
	}

	wg.Wait()
	close(resultsCh)

	// Reconstruct in original order
	ordered := make([]models.TurnEvaluationResult, len(req.Turns))
	for r := range resultsCh {
		ordered[r.index] = r.result
	}
	return ordered
}

func weightedAverage(scores []models.StageResult) float64 {
	if len(scores) == 0 {
		return 0.0
	}
	totalWeight := 0.0
	weightedSum := 0.0
	for _, s := range scores {
		w := s.Weight
		if w == 0 {
			w = 1.0
		}
		weightedSum += s.Score * w
		totalWeight += w
	}
	if totalWeight == 0 {
		return 0.0
	}
	return weightedSum / totalWeight
}

func (e *ConversationEvaluator) verdict(score float64) models.Verdict {
	if score > e.passThreshold {
		return models.VerdictPass
	}
	if score > e.reviewThreshold {
		return models.VerdictReview
	}
	return models.VerdictFail
}
