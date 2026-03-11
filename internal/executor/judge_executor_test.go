package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Terminus-Lab/themis/internal/executor/mocks"
	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/Terminus-Lab/themis/internal/storage/sqlite"
	"github.com/rs/zerolog"
	"go.uber.org/mock/gomock"
)

func testLogger() *zerolog.Logger {
	logger := zerolog.Nop()
	return &logger
}

// Setup SQLite in-memory database for testing
func setupTestDB(t *testing.T) (*sqlite.DB, func()) {
	t.Helper()

	db, err := sqlite.New(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}

	if err := db.InitSchema(context.Background()); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}
	cleanup := func() { _ = db.Close() }

	return db, cleanup
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

			// Setup SQLite in-memory database
			db, cleanup := setupTestDB(t)
			defer cleanup()
			repo := sqlite.NewEvalRepository(db, testLogger())

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
			executor := NewJudgeExecutor(mockJudgeFactory, repo, testLogger())
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

			// NEW: Verify result was stored in database
			stored, err := repo.QueryById(context.Background(), tt.evalCtx.RequestID)
			if err != nil {
				t.Fatalf("failed to query stored result: %v", err)
			}

			if stored.EventID != tt.evalCtx.RequestID {
				t.Errorf("stored EventID mismatch: expected %s, got %s", tt.evalCtx.RequestID, stored.EventID)
			}

			if stored.Confidence != tt.expectScore {
				t.Errorf("stored Confidence mismatch: expected %.2f, got %.2f", tt.expectScore, stored.Confidence)
			}

			if stored.Verdict != string(tt.expectVerdict) {
				t.Errorf("stored Verdict mismatch: expected %s, got %s", tt.expectVerdict, stored.Verdict)
			}

			if stored.UserQuery != tt.evalCtx.Query {
				t.Errorf("stored UserQuery mismatch: expected %s, got %s", tt.evalCtx.Query, stored.UserQuery)
			}

			if stored.Answer != tt.evalCtx.Answer {
				t.Errorf("stored Answer mismatch: expected %s, got %s", tt.evalCtx.Answer, stored.Answer)
			}

			if len(stored.StageScores) != 1 {
				t.Errorf("stored StageScores count mismatch: expected 1, got %d", len(stored.StageScores))
			}
		})
	}
}

func TestJudgeExecutor_Execute_ContextCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup SQLite in-memory database
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := sqlite.NewEvalRepository(db, testLogger())

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

	executor := NewJudgeExecutor(mockJudgeFactory, repo, testLogger())
	result, err := executor.Execute(ctx, "relevance", 0.7, evalCtx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return result even with cancelled context (judge handles it)
	if result.Verdict != models.VerdictFail {
		t.Errorf("expected verdict Fail for cancelled context, got %s", result.Verdict)
	}

	// Note: Storage may fail with cancelled context - this is expected behavior
	// The executor doesn't check Store errors (fire-and-forget pattern)
}

func TestJudgeExecutor_Execute_VerifyStorageIntegration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup SQLite in-memory database
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := sqlite.NewEvalRepository(db, testLogger())

	mockJudgeFactory := mocks.NewMockJudgeFactory(ctrl)
	mockJudge := mocks.NewMockJudge(ctrl)

	// Run multiple evaluations
	tests := []struct {
		requestID string
		judgeName string
		score     float64
		verdict   models.Verdict
	}{
		{"req-001", "relevance", 0.9, models.VerdictPass},
		{"req-002", "coherence", 0.3, models.VerdictFail},
		{"req-003", "faithfulness", 0.85, models.VerdictPass},
	}

	for _, tt := range tests {
		evalCtx := models.EvaluationContext{
			RequestID: tt.requestID,
			Query:     "test query",
			Answer:    "test answer",
			Context:   "test context",
			CreatedAt: time.Now(),
		}

		stageResult := models.StageResult{
			Name:     tt.judgeName,
			Score:    tt.score,
			Reason:   "test reason",
			Duration: 100 * time.Millisecond,
		}

		mockJudgeFactory.EXPECT().Get(tt.judgeName).Return(mockJudge, nil)
		mockJudge.EXPECT().Evaluate(gomock.Any(), evalCtx).Return(stageResult)

		executor := NewJudgeExecutor(mockJudgeFactory, repo, testLogger())
		_, err := executor.Execute(context.Background(), tt.judgeName, 0.7, evalCtx)
		if err != nil {
			t.Fatalf("execution failed: %v", err)
		}
	}

	// Verify all 3 evaluations are stored
	allResults, count, err := repo.Query(context.Background(), models.QueryFilters{Limit: 10})
	if err != nil {
		t.Fatalf("failed to query all results: %v", err)
	}

	if count != 3 {
		t.Errorf("expected 3 stored evaluations, got %d", count)
	}

	if len(allResults) != 3 {
		t.Errorf("expected 3 results, got %d", len(allResults))
	}

	// Verify we can query by verdict
	passResults, passCount, err := repo.Query(context.Background(), models.QueryFilters{
		Verdict: "pass",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("failed to query pass results: %v", err)
	}

	if passCount != 2 {
		t.Errorf("expected 2 pass verdicts, got %d", passCount)
	}

	if len(passResults) != 2 {
		t.Errorf("expected 2 pass results, got %d", len(passResults))
	}

	// Verify fail verdict
	failResults, failCount, err := repo.Query(context.Background(), models.QueryFilters{
		Verdict: "fail",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("failed to query fail results: %v", err)
	}

	if failCount != 1 {
		t.Errorf("expected 1 fail verdict, got %d", failCount)
	}

	if len(failResults) != 1 {
		t.Errorf("expected 1 fail result, got %d", len(failResults))
	}
}
