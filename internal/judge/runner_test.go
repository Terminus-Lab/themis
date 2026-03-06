package judge

import (
	"context"
	"testing"
	"time"

	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/rs/zerolog"
)

// MockJudge is a test implementation of Judge interface
type MockJudge struct {
	name                   string
	requiresExpectedOutput bool
	wasCalled              bool
	resultToReturn         models.StageResult
}

func (m *MockJudge) Name() string {
	return m.name
}

func (m *MockJudge) Evaluate(ctx context.Context, evaluationContext models.EvaluationContext) models.StageResult {
	m.wasCalled = true
	return m.resultToReturn
}

func (m *MockJudge) RequiresExpectedOutput() bool {
	return m.requiresExpectedOutput
}

func TestJudgeRunner_Run_SkipsJudgeWhenExpectedOutputMissing(t *testing.T) {
	logger := zerolog.Nop()

	// Create a judge that requires expected_output
	correctnessJudge := &MockJudge{
		name:                   "correctness",
		requiresExpectedOutput: true,
		resultToReturn: models.StageResult{
			Name:     "correctness",
			Score:    0.9,
			Reason:   "should not be called",
			Duration: 100 * time.Millisecond,
		},
	}

	// Create a regular judge that doesn't require expected_output
	relevanceJudge := &MockJudge{
		name:                   "relevance",
		requiresExpectedOutput: false,
		resultToReturn: models.StageResult{
			Name:     "relevance",
			Score:    0.8,
			Reason:   "relevant",
			Duration: 100 * time.Millisecond,
		},
	}

	runner := NewJudgeRunner([]Judge{correctnessJudge, relevanceJudge}, &logger)

	// Create evaluation context WITHOUT expected_output
	evalCtx := models.EvaluationContext{
		RequestID: "test-123",
		Query:     "What is the capital of France?",
		Answer:    "Paris",
		// ExpectedOutput is empty
		CreatedAt: time.Now(),
	}

	results := runner.Run(context.Background(), evalCtx)

	// Should only have 1 result (relevance), not 2
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// Correctness judge should NOT have been called
	if correctnessJudge.wasCalled {
		t.Error("Expected correctness judge to be skipped, but it was called")
	}

	// Relevance judge SHOULD have been called
	if !relevanceJudge.wasCalled {
		t.Error("Expected relevance judge to be called, but it wasn't")
	}

	// Verify the result is from relevance judge
	if results[0].Name != "relevance" {
		t.Errorf("Expected result from 'relevance', got '%s'", results[0].Name)
	}
}

func TestJudgeRunner_Run_RunsJudgeWhenExpectedOutputProvided(t *testing.T) {
	logger := zerolog.Nop()

	// Create a judge that requires expected_output
	correctnessJudge := &MockJudge{
		name:                   "correctness",
		requiresExpectedOutput: true,
		resultToReturn: models.StageResult{
			Name:     "correctness",
			Score:    0.95,
			Reason:   "exact match",
			Duration: 100 * time.Millisecond,
		},
	}

	// Create a regular judge
	relevanceJudge := &MockJudge{
		name:                   "relevance",
		requiresExpectedOutput: false,
		resultToReturn: models.StageResult{
			Name:     "relevance",
			Score:    0.8,
			Reason:   "relevant",
			Duration: 100 * time.Millisecond,
		},
	}

	runner := NewJudgeRunner([]Judge{correctnessJudge, relevanceJudge}, &logger)

	// Create evaluation context WITH expected_output
	evalCtx := models.EvaluationContext{
		RequestID:      "test-123",
		Query:          "What is the capital of France?",
		Answer:         "Paris",
		ExpectedOutput: "Paris", // Provided
		CreatedAt:      time.Now(),
	}

	results := runner.Run(context.Background(), evalCtx)

	// Should have 2 results
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Both judges should have been called
	if !correctnessJudge.wasCalled {
		t.Error("Expected correctness judge to be called, but it wasn't")
	}

	if !relevanceJudge.wasCalled {
		t.Error("Expected relevance judge to be called, but it wasn't")
	}
}

func TestJudgeRunner_Run_AllJudgesSkipped(t *testing.T) {
	logger := zerolog.Nop()

	// Create judges that all require expected_output
	correctnessJudge := &MockJudge{
		name:                   "correctness",
		requiresExpectedOutput: true,
		resultToReturn: models.StageResult{
			Name:  "correctness",
			Score: 0.9,
		},
	}

	similarityJudge := &MockJudge{
		name:                   "similarity",
		requiresExpectedOutput: true,
		resultToReturn: models.StageResult{
			Name:  "similarity",
			Score: 0.85,
		},
	}

	runner := NewJudgeRunner([]Judge{correctnessJudge, similarityJudge}, &logger)

	// Create evaluation context WITHOUT expected_output
	evalCtx := models.EvaluationContext{
		RequestID: "test-123",
		Query:     "What is the capital of France?",
		Answer:    "Paris",
		// ExpectedOutput is empty
		CreatedAt: time.Now(),
	}

	results := runner.Run(context.Background(), evalCtx)

	// Should have 0 results
	if len(results) != 0 {
		t.Errorf("Expected 0 results (all judges skipped), got %d", len(results))
	}

	// Neither judge should have been called
	if correctnessJudge.wasCalled {
		t.Error("Expected correctness judge to be skipped")
	}

	if similarityJudge.wasCalled {
		t.Error("Expected similarity judge to be skipped")
	}
}

func TestJudgeRunner_Run_NoJudges(t *testing.T) {
	logger := zerolog.Nop()

	runner := NewJudgeRunner([]Judge{}, &logger)

	evalCtx := models.EvaluationContext{
		RequestID: "test-123",
		Query:     "What is the capital of France?",
		Answer:    "Paris",
		CreatedAt: time.Now(),
	}

	results := runner.Run(context.Background(), evalCtx)

	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty judge list, got %d", len(results))
	}
}
