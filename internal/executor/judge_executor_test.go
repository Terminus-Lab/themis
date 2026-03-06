package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Terminus-Lab/themis/internal/executor/mocks"
	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/rs/zerolog"
	"go.uber.org/mock/gomock"
)

func testLogger() *zerolog.Logger {
	logger := zerolog.Nop()
	return &logger
}

func TestJudgeExecutor_Execute(t *testing.T) {
	tests := []struct {
		name          string
		judgeName     string
		threshold     float64
		stageResult   models.StageResult
		judgeErr      error
		evalCtx       models.EvaluationContext
		expectErr     error
		expectVerdict models.Verdict
		expectScore   float64
	}{
		{
			name:      "score above threshold - pass",
			judgeName: "relevance",
			threshold: 0.7,
			stageResult: models.StageResult{
				Name:     "relevance",
				Score:    0.85,
				Reason:   "Relevant",
				Duration: 100 * time.Millisecond,
			},
			evalCtx: models.EvaluationContext{
				RequestID: "test-001",
				Query:     "What is Go?",
				Answer:    "Go is a programming language.",
				Context:   "Go documentation",
				CreatedAt: time.Now(),
			},
			expectErr:     nil,
			expectVerdict: models.VerdictPass,
			expectScore:   0.85,
		},
		{
			name:      "score below threshold - fail",
			judgeName: "coherence",
			threshold: 0.6,
			stageResult: models.StageResult{
				Name:     "coherence",
				Score:    0.4,
				Reason:   "Incoherent response",
				Duration: 120 * time.Millisecond,
			},
			evalCtx: models.EvaluationContext{
				RequestID: "test-002",
				Query:     "Explain Docker?",
				Answer:    "Random unrelated text.",
				Context:   "Docker documentation",
				CreatedAt: time.Now(),
			},
			expectErr:     nil,
			expectVerdict: models.VerdictFail,
			expectScore:   0.4,
		},
		{
			name:      "score equal to threshold - fail",
			judgeName: "faithfulness",
			threshold: 0.75,
			stageResult: models.StageResult{
				Name:     "faithfulness",
				Score:    0.75,
				Reason:   "Borderline case",
				Duration: 90 * time.Millisecond,
			},
			evalCtx: models.EvaluationContext{
				RequestID: "test-003",
				Query:     "What is Redis?",
				Answer:    "Redis is a data store.",
				Context:   "Redis documentation",
				CreatedAt: time.Now(),
			},
			expectErr:     nil,
			expectVerdict: models.VerdictFail,
			expectScore:   0.75,
		},
		{
			name:      "judge not found - error",
			judgeName: "unknown-judge",
			threshold: 0.5,
			judgeErr:  errors.New("judge not found"),
			evalCtx: models.EvaluationContext{
				RequestID: "test-004",
				Query:     "Test query",
				Answer:    "Test answer",
				Context:   "Test context",
				CreatedAt: time.Now(),
			},
			expectErr: ErrJudgeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockJudgeFactory := mocks.NewMockJudgeFactory(ctrl)
			mockJudge := mocks.NewMockJudge(ctrl)

			// Setup expectations
			if tt.judgeErr != nil {
				mockJudgeFactory.EXPECT().Get(tt.judgeName).Return(nil, tt.judgeErr)
			} else {
				mockJudgeFactory.EXPECT().Get(tt.judgeName).Return(mockJudge, nil)
				mockJudge.EXPECT().Evaluate(gomock.Any(), tt.evalCtx).Return(tt.stageResult)
			}

			// Execute
			executor := NewJudgeExecutor(mockJudgeFactory, testLogger())
			result, err := executor.Execute(context.Background(), tt.judgeName, tt.threshold, tt.evalCtx)

			// Assert error
			if tt.expectErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.expectErr)
				} else if !errors.Is(err, tt.expectErr) {
					t.Errorf("expected error %v, got %v", tt.expectErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Assert result fields
			if result.ID != tt.evalCtx.RequestID {
				t.Errorf("expected ID %s, got %s", tt.evalCtx.RequestID, result.ID)
			}

			if result.Verdict != tt.expectVerdict {
				t.Errorf("expected verdict %s, got %s", tt.expectVerdict, result.Verdict)
			}

			if result.Confidence != tt.expectScore {
				t.Errorf("expected confidence %.2f, got %.2f", tt.expectScore, result.Confidence)
			}

			// Assert stages
			if len(result.Stages) != 1 {
				t.Fatalf("expected 1 stage, got %d", len(result.Stages))
			}

			stage := result.Stages[0]
			if stage.Name != tt.stageResult.Name {
				t.Errorf("expected stage name %s, got %s", tt.stageResult.Name, stage.Name)
			}

			if stage.Score != tt.stageResult.Score {
				t.Errorf("expected stage score %.2f, got %.2f", tt.stageResult.Score, stage.Score)
			}

			if stage.Reason != tt.stageResult.Reason {
				t.Errorf("expected stage reason %s, got %s", tt.stageResult.Reason, stage.Reason)
			}
		})
	}
}

func TestJudgeExecutor_Execute_ContextCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockJudgeFactory := mocks.NewMockJudgeFactory(ctrl)
	mockJudge := mocks.NewMockJudge(ctrl)

	evalCtx := models.EvaluationContext{
		RequestID: "test-cancel",
		Query:     "Test query",
		Answer:    "Test answer",
		Context:   "Test context",
		CreatedAt: time.Now(),
	}

	// Cancel context before judge evaluation
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stageResult := models.StageResult{
		Name:     "relevance",
		Score:    0.0,
		Reason:   "Context cancelled",
		Duration: 0,
	}

	mockJudgeFactory.EXPECT().Get("relevance").Return(mockJudge, nil)
	mockJudge.EXPECT().Evaluate(ctx, evalCtx).Return(stageResult)

	executor := NewJudgeExecutor(mockJudgeFactory, testLogger())
	result, err := executor.Execute(ctx, "relevance", 0.7, evalCtx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return result even with cancelled context (judge handles it)
	if result.Verdict != models.VerdictFail {
		t.Errorf("expected verdict Fail for cancelled context, got %s", result.Verdict)
	}
}
