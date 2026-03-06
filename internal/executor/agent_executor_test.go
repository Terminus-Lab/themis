package executor

import (
	"context"
	"testing"
	"time"

	"github.com/Terminus-Lab/themis/internal/executor/mocks"
	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/rs/zerolog"
	"go.uber.org/mock/gomock"
)

func newTestLogger() *zerolog.Logger {
	logger := zerolog.Nop()
	return &logger
}

func TestExecutor_Execute_FullPipeline_Pass(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPrecheck := mocks.NewMockPrecheckRunner(ctrl)
	mockJudge := mocks.NewMockJudgeRunner(ctrl)
	mockAgg := mocks.NewMockAggregator(ctrl)

	evalCtx := models.EvaluationContext{
		RequestID: "test-001",
		Query:     "What is Go?",
		Answer:    "Go is a programming language.",
		Context:   "Go documentation",
		CreatedAt: time.Now(),
	}

	// Set expectations
	precheckResults := []models.StageResult{
		{Name: "length", Score: 0.8, Reason: "good length", Duration: 100 * time.Millisecond},
		{Name: "overlap", Score: 0.7, Reason: "good overlap", Duration: 100 * time.Millisecond},
	}
	mockPrecheck.EXPECT().Run(evalCtx).Return(precheckResults)

	judgeResults := []models.StageResult{
		{Name: "relevance", Score: 0.9, Reason: "relevant", Duration: 1 * time.Second},
		{Name: "faithfulness", Score: 0.85, Reason: "faithful", Duration: 1 * time.Second},
	}
	mockJudge.EXPECT().Run(gomock.Any(), evalCtx).Return(judgeResults)

	expectedResult := models.EvaluationResult{
		ID:         "test-001",
		Stages:     append(precheckResults, judgeResults...),
		Confidence: 0.85,
		Verdict:    models.VerdictPass,
	}
	mockAgg.EXPECT().Aggregate("test-001", precheckResults, judgeResults).Return(expectedResult)

	executor := NewExecutor(mockPrecheck, mockJudge, mockAgg, 0.2, newTestLogger())

	result := executor.Execute(context.Background(), evalCtx)

	if result.ID != "test-001" {
		t.Errorf("expected ID test-001, got %s", result.ID)
	}
	if result.Verdict != models.VerdictPass {
		t.Errorf("expected verdict Pass, got %s", result.Verdict)
	}
	if result.Confidence != 0.85 {
		t.Errorf("expected confidence 0.85, got %.2f", result.Confidence)
	}
}

func TestExecutor_Execute_EarlyExit_LowScore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPrecheck := mocks.NewMockPrecheckRunner(ctrl)
	mockJudge := mocks.NewMockJudgeRunner(ctrl)
	mockAgg := mocks.NewMockAggregator(ctrl)

	evalCtx := models.EvaluationContext{
		RequestID: "test-002",
		Query:     "test",
		Answer:    "a",
		Context:   "",
		CreatedAt: time.Now(),
	}

	// Precheck returns low scores triggering early exit
	precheckResults := []models.StageResult{
		{Name: "length", Score: 0.1, Reason: "too short", Duration: 100 * time.Millisecond},
		{Name: "overlap", Score: 0.15, Reason: "no overlap", Duration: 100 * time.Millisecond},
	}
	mockPrecheck.EXPECT().Run(evalCtx).Return(precheckResults)

	executor := NewExecutor(mockPrecheck, mockJudge, mockAgg, 0.2, newTestLogger())

	result := executor.Execute(context.Background(), evalCtx)

	// Early exit: avg = (0.1 + 0.15) / 2 = 0.125 < 0.2 threshold
	if result.Verdict != models.VerdictFail {
		t.Errorf("expected early exit verdict Fail, got %s", result.Verdict)
	}
	if len(result.Stages) != 2 {
		t.Errorf("expected 2 precheck stages, got %d", len(result.Stages))
	}
}

func TestExecutor_Execute_EmptyPrechecks_Fail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPrecheck := mocks.NewMockPrecheckRunner(ctrl)
	mockJudge := mocks.NewMockJudgeRunner(ctrl)
	mockAgg := mocks.NewMockAggregator(ctrl)

	evalCtx := models.EvaluationContext{
		RequestID: "test-003",
		Query:     "test",
		Answer:    "test",
		Context:   "",
		CreatedAt: time.Now(),
	}

	// Empty prechecks
	mockPrecheck.EXPECT().Run(evalCtx).Return([]models.StageResult{})

	executor := NewExecutor(mockPrecheck, mockJudge, mockAgg, 0.2, newTestLogger())

	result := executor.Execute(context.Background(), evalCtx)

	if result.Verdict != models.VerdictFail {
		t.Errorf("expected verdict Fail for empty prechecks, got %s", result.Verdict)
	}
	if result.ID != "test-003" {
		t.Errorf("expected ID test-003, got %s", result.ID)
	}
}

func TestExecutor_Execute_PassPrecheck_FailJudge(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPrecheck := mocks.NewMockPrecheckRunner(ctrl)
	mockJudge := mocks.NewMockJudgeRunner(ctrl)
	mockAgg := mocks.NewMockAggregator(ctrl)

	evalCtx := models.EvaluationContext{
		RequestID: "test-004",
		Query:     "What is Go?",
		Answer:    "The sky is blue.",
		Context:   "",
		CreatedAt: time.Now(),
	}

	precheckResults := []models.StageResult{
		{Name: "length", Score: 0.9, Reason: "good", Duration: 100 * time.Millisecond},
	}
	mockPrecheck.EXPECT().Run(evalCtx).Return(precheckResults)

	judgeResults := []models.StageResult{
		{Name: "relevance", Score: 0.3, Reason: "not relevant", Duration: 1 * time.Second},
	}
	mockJudge.EXPECT().Run(gomock.Any(), evalCtx).Return(judgeResults)

	expectedResult := models.EvaluationResult{
		ID:         "test-004",
		Stages:     append(precheckResults, judgeResults...),
		Confidence: 0.48, // (0.9 * 0.3) + (0.3 * 0.7) = 0.48
		Verdict:    models.VerdictFail,
	}
	mockAgg.EXPECT().Aggregate("test-004", precheckResults, judgeResults).Return(expectedResult)

	executor := NewExecutor(mockPrecheck, mockJudge, mockAgg, 0.2, newTestLogger())

	result := executor.Execute(context.Background(), evalCtx)

	if result.Verdict != models.VerdictFail {
		t.Errorf("expected verdict Fail, got %s", result.Verdict)
	}
	if result.Confidence != 0.48 {
		t.Errorf("expected confidence 0.48, got %.2f", result.Confidence)
	}
}

func TestExecutor_Execute_EarlyExitThreshold(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name            string
		threshold       float64
		precheckScore   float64
		shouldEarlyExit bool
	}{
		{"below threshold - early exit", 0.5, 0.4, true},
		{"above threshold - continue", 0.5, 0.6, false},
		{"exact threshold - continue", 0.2, 0.2, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPrecheck := mocks.NewMockPrecheckRunner(ctrl)
			mockJudge := mocks.NewMockJudgeRunner(ctrl)
			mockAgg := mocks.NewMockAggregator(ctrl)

			evalCtx := models.EvaluationContext{
				RequestID: "test",
				Query:     "test",
				Answer:    "test",
				Context:   "",
				CreatedAt: time.Now(),
			}

			precheckResults := []models.StageResult{
				{Name: "test", Score: tt.precheckScore, Reason: "test", Duration: 100 * time.Millisecond},
			}
			mockPrecheck.EXPECT().Run(evalCtx).Return(precheckResults)

			if tt.shouldEarlyExit {
				// Judge and aggregator should NOT be called
			} else {
				// Judge and aggregator SHOULD be called
				judgeResults := []models.StageResult{
					{Name: "judge", Score: 0.9, Reason: "test", Duration: 1 * time.Second},
				}
				mockJudge.EXPECT().Run(gomock.Any(), evalCtx).Return(judgeResults)
				mockAgg.EXPECT().Aggregate("test", precheckResults, judgeResults).Return(models.EvaluationResult{
					ID:         "test",
					Confidence: 0.85,
					Verdict:    models.VerdictPass,
				})
			}

			executor := NewExecutor(mockPrecheck, mockJudge, mockAgg, tt.threshold, newTestLogger())

			result := executor.Execute(context.Background(), evalCtx)

			if tt.shouldEarlyExit && result.Verdict != models.VerdictFail {
				t.Errorf("expected early exit with Fail verdict, got %s", result.Verdict)
			}
		})
	}
}
