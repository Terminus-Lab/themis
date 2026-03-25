package executor

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/Terminus-Lab/themis/internal/judge"
	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/Terminus-Lab/themis/internal/storage"
	"github.com/rs/zerolog"
)

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// stubJudge implements judge.Judge for testing.
type stubJudge struct {
	name   string
	result models.StageResult
}

func (s *stubJudge) Evaluate(_ context.Context, _ models.EvaluationContext) models.StageResult {
	return s.result
}
func (s *stubJudge) Name() string { return s.name }

// noopRepository is a storage.Repository that does nothing.
type noopRepository struct{}

func (n *noopRepository) StoreConversation(_ context.Context, _ *storage.ConversationRecord) error {
	return nil
}
func (n *noopRepository) GetConversation(_ context.Context, _ string) (*storage.ConversationRecord, error) {
	return nil, nil
}
func (n *noopRepository) ListConversations(_ context.Context) ([]storage.ConversationSummary, error) {
	return nil, nil
}
func (n *noopRepository) HealthMetrics(_ context.Context, _ time.Time) (storage.HealthMetricsData, error) {
	return storage.HealthMetricsData{}, nil
}

func newTestEvaluator(turnJudges []judge.Judge, holisticJudge judge.Judge) *ConversationEvaluator {
	logger := zerolog.Nop()
	runner := judge.NewJudgeRunner(turnJudges, &logger)
	return &ConversationEvaluator{
		turnRunner:      runner,
		holisticJudge:   holisticJudge,
		repository:      &noopRepository{},
		holisticWeight:  0.5,
		passThreshold:   0.8,
		reviewThreshold: 0.5,
		scoringFormula:  "linear",
		logger:          &logger,
	}
}

// --- weightedAverage tests ---

func TestWeightedAverage_ErroredJudgeCountsAsZero(t *testing.T) {
	scores := []models.StageResult{
		{Name: "relevance", Score: 0.9, Weight: 0.35},
		{Name: "coherence", Score: 0.0, Weight: 0.30, Error: "LLM call failed: API error"},
		{Name: "completeness", Score: 0.8, Weight: 0.35},
	}

	got := weightedAverage(scores)

	// Errored coherence judge counts as score 0.0 (penalty).
	// (0.9*0.35 + 0.0*0.30 + 0.8*0.35) / (0.35+0.30+0.35)
	expected := (0.9*0.35 + 0.0*0.30 + 0.8*0.35) / (0.35 + 0.30 + 0.35)
	if !approxEqual(got, expected) {
		t.Errorf("weightedAverage = %f, want %f", got, expected)
	}
}

func TestWeightedAverage_AllErrored_ReturnsZero(t *testing.T) {
	scores := []models.StageResult{
		{Name: "relevance", Score: 0.0, Weight: 0.35, Error: "timeout"},
		{Name: "coherence", Score: 0.0, Weight: 0.30, Error: "LLM call failed"},
	}

	got := weightedAverage(scores)
	if got != 0.0 {
		t.Errorf("Expected 0.0 when all judges errored, got %f", got)
	}
}

func TestWeightedAverage_NoErrors_WorksAsNormal(t *testing.T) {
	scores := []models.StageResult{
		{Name: "relevance", Score: 1.0, Weight: 0.5},
		{Name: "coherence", Score: 0.5, Weight: 0.5},
	}

	got := weightedAverage(scores)
	expected := 0.75
	if got != expected {
		t.Errorf("weightedAverage = %f, want %f", got, expected)
	}
}

// --- ConversationEvaluator.Execute tests ---

func TestConversationEvaluator_EvalErrors_SurfacedWhenTurnJudgeFails(t *testing.T) {
	turnJudges := []judge.Judge{
		&stubJudge{name: "relevance", result: models.StageResult{
			Name: "relevance-judge", Score: 0.9, Weight: 0.35,
		}},
		&stubJudge{name: "coherence", result: models.StageResult{
			Name: "coherence-judge", Score: 0.0, Weight: 0.30,
			Error: "LLM call failed: API error",
		}},
	}
	holistic := &stubJudge{name: "conversation-flow", result: models.StageResult{
		Name: "conversation-flow-judge", Score: 0.8,
	}}

	evaluator := newTestEvaluator(turnJudges, holistic)

	req := models.ConversationEvaluationRequest{
		ConversationID: "test-conv-1",
		Agent:          models.Agent{Name: "test-agent", Version: "1.0"},
		Turns: []models.ConversationTurn{
			{TurnIndex: 0, UserQuery: "What is AI?", Answer: "AI is artificial intelligence"},
		},
	}

	result := evaluator.Execute(context.Background(), req)

	if len(result.EvalErrors) == 0 {
		t.Fatal("Expected eval_errors to be non-empty when a judge failed")
	}

	// Errored coherence judge counts as 0.0 (penalty).
	// turn_avg = (0.9*0.35 + 0.0*0.30) / (0.35+0.30)
	expected := (0.9*0.35 + 0.0*0.30) / (0.35 + 0.30)
	if !approxEqual(result.TurnAvg, expected) {
		t.Errorf("Expected turn_avg=%f (errored judge penalizes score), got %f", expected, result.TurnAvg)
	}
}

func TestConversationEvaluator_EvalErrors_HolisticJudgeFailure(t *testing.T) {
	turnJudges := []judge.Judge{
		&stubJudge{name: "relevance", result: models.StageResult{
			Name: "relevance-judge", Score: 0.9, Weight: 1.0,
		}},
	}
	holistic := &stubJudge{name: "conversation-flow", result: models.StageResult{
		Name:  "conversation-flow-judge",
		Score: 0.0,
		Error: "evaluation timed out after 30s",
	}}

	evaluator := newTestEvaluator(turnJudges, holistic)

	req := models.ConversationEvaluationRequest{
		ConversationID: "test-conv-2",
		Agent:          models.Agent{Name: "test-agent", Version: "1.0"},
		Turns: []models.ConversationTurn{
			{TurnIndex: 0, UserQuery: "What is AI?", Answer: "AI is artificial intelligence"},
		},
	}

	result := evaluator.Execute(context.Background(), req)

	if len(result.EvalErrors) == 0 {
		t.Fatal("Expected eval_errors to contain holistic judge failure")
	}
}

func TestConversationEvaluator_NoEvalErrors_WhenAllJudgesSucceed(t *testing.T) {
	turnJudges := []judge.Judge{
		&stubJudge{name: "relevance", result: models.StageResult{
			Name: "relevance-judge", Score: 0.9, Weight: 1.0,
		}},
	}
	holistic := &stubJudge{name: "conversation-flow", result: models.StageResult{
		Name: "conversation-flow-judge", Score: 0.85,
	}}

	evaluator := newTestEvaluator(turnJudges, holistic)

	req := models.ConversationEvaluationRequest{
		ConversationID: "test-conv-3",
		Agent:          models.Agent{Name: "test-agent", Version: "1.0"},
		Turns: []models.ConversationTurn{
			{TurnIndex: 0, UserQuery: "What is AI?", Answer: "AI is artificial intelligence"},
		},
	}

	result := evaluator.Execute(context.Background(), req)

	if len(result.EvalErrors) != 0 {
		t.Errorf("Expected no eval_errors when all judges succeed, got %v", result.EvalErrors)
	}
}
