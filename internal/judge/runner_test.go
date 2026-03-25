package judge

import (
	"context"
	"testing"
	"time"

	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/rs/zerolog"
)

// MockJudge is a test implementation of the Judge interface
type MockJudge struct {
	name           string
	wasCalled      bool
	resultToReturn models.StageResult
}

func (m *MockJudge) Name() string {
	return m.name
}

func (m *MockJudge) Evaluate(ctx context.Context, evaluationContext models.EvaluationContext) models.StageResult {
	m.wasCalled = true
	return m.resultToReturn
}

func TestJudgeRunner_Run_AllJudgesRun(t *testing.T) {
	logger := zerolog.Nop()

	judge1 := &MockJudge{
		name: "relevance",
		resultToReturn: models.StageResult{
			Name:   "relevance",
			Score:  0.8,
			Reason: "relevant",
		},
	}

	judge2 := &MockJudge{
		name: "coherence",
		resultToReturn: models.StageResult{
			Name:   "coherence",
			Score:  0.9,
			Reason: "coherent",
		},
	}

	runner := NewJudgeRunner([]Judge{judge1, judge2}, &logger)

	evalCtx := models.EvaluationContext{
		Query:     "What is the capital of France?",
		Answer:    "Paris",
		CreatedAt: time.Now(),
	}

	results := runner.Run(context.Background(), evalCtx)

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	if !judge1.wasCalled {
		t.Error("Expected judge1 to be called")
	}
	if !judge2.wasCalled {
		t.Error("Expected judge2 to be called")
	}
}

func TestJudgeRunner_Run_NoJudges(t *testing.T) {
	logger := zerolog.Nop()

	runner := NewJudgeRunner([]Judge{}, &logger)

	evalCtx := models.EvaluationContext{
		Query:     "What is the capital of France?",
		Answer:    "Paris",
		CreatedAt: time.Now(),
	}

	results := runner.Run(context.Background(), evalCtx)

	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty judge list, got %d", len(results))
	}
}

func TestJudgeRunner_Run_SingleJudge(t *testing.T) {
	logger := zerolog.Nop()

	j := &MockJudge{
		name: "completeness",
		resultToReturn: models.StageResult{
			Name:   "completeness",
			Score:  0.75,
			Reason: "mostly complete",
		},
	}

	runner := NewJudgeRunner([]Judge{j}, &logger)

	evalCtx := models.EvaluationContext{
		Query:     "Explain AI",
		Answer:    "AI is artificial intelligence",
		CreatedAt: time.Now(),
	}

	results := runner.Run(context.Background(), evalCtx)

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if results[0].Score != 0.75 {
		t.Errorf("Expected score=0.75, got %f", results[0].Score)
	}
	if !j.wasCalled {
		t.Error("Expected judge to be called")
	}
}
