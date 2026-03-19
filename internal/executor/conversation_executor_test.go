package executor

import (
	"context"
	"testing"
	"time"

	"github.com/Terminus-Lab/themis/internal/executor/mocks"
	"github.com/Terminus-Lab/themis/internal/models"
	"go.uber.org/mock/gomock"
)

func TestConversationExecutor_Execute_StoresResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRunner := mocks.NewMockJudgeRunner(ctrl)
	mockAgg := mocks.NewMockAggregator(ctrl)
	repo := setupTestRepository(t)

	req := models.ConversationEvaluationRequest{
		ConversationID: "conv-exec-001",
		Agent:          models.Agent{Name: "test-agent", Version: "1.0"},
		Turns: []models.ConversationTurn{
			{TurnIndex: 0, UserQuery: "Hello?", Answer: "Hi there!"},
			{TurnIndex: 1, UserQuery: "How are you?", Answer: "I am fine."},
		},
	}

	judgeResults := []models.StageResult{
		{Name: "conversation-flow-judge", Score: 0.85, Reason: "coherent flow", Duration: 500 * time.Millisecond},
	}

	mockRunner.EXPECT().
		Run(gomock.Any(), gomock.AssignableToTypeOf(models.EvaluationContext{})).
		DoAndReturn(func(_ context.Context, ctx models.EvaluationContext) []models.StageResult {
			// Verify turns were passed in the evaluation context
			if len(ctx.Turns) != 2 {
				t.Errorf("expected 2 turns in context, got %d", len(ctx.Turns))
			}
			if ctx.ConversationID != "conv-exec-001" {
				t.Errorf("expected conversation_id=conv-exec-001, got %s", ctx.ConversationID)
			}
			return judgeResults
		})

	mockAgg.EXPECT().
		Aggregate(gomock.Any(), nil, judgeResults).
		Return(models.EvaluationResult{
			Confidence: 0.85,
			Verdict:    models.VerdictPass,
			Stages:     judgeResults,
		})

	exec := NewConversationExecutor(mockRunner, repo, mockAgg, newTestLogger())
	result := exec.Execute(context.Background(), req)

	if result.ConversationID != "conv-exec-001" {
		t.Errorf("expected ConversationID=conv-exec-001, got %s", result.ConversationID)
	}
	if result.TurnCount != 2 {
		t.Errorf("expected TurnCount=2, got %d", result.TurnCount)
	}
	if result.AgentName != "test-agent" {
		t.Errorf("expected AgentName=test-agent, got %s", result.AgentName)
	}
	if result.Verdict != models.VerdictPass {
		t.Errorf("expected verdict=pass, got %s", result.Verdict)
	}
	if result.Confidence != 0.85 {
		t.Errorf("expected confidence=0.85, got %.2f", result.Confidence)
	}
	if len(result.Stages) != 1 {
		t.Errorf("expected 1 stage, got %d", len(result.Stages))
	}

	// Verify persisted to DB
	stored, err := repo.GetConversationEval(context.Background(), "conv-exec-001")
	if err != nil {
		t.Fatalf("expected conversation eval to be stored, got error: %v", err)
	}
	if stored.ConversationID != "conv-exec-001" {
		t.Errorf("stored ConversationID mismatch: got %s", stored.ConversationID)
	}
	if stored.TurnCount != 2 {
		t.Errorf("stored TurnCount mismatch: expected 2, got %d", stored.TurnCount)
	}
	if stored.Verdict != "pass" {
		t.Errorf("stored Verdict mismatch: expected pass, got %s", stored.Verdict)
	}
}

func TestConversationExecutor_Execute_SingleTurn(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRunner := mocks.NewMockJudgeRunner(ctrl)
	mockAgg := mocks.NewMockAggregator(ctrl)
	repo := setupTestRepository(t)

	req := models.ConversationEvaluationRequest{
		ConversationID: "conv-single-001",
		Agent:          models.Agent{Name: "agent", Version: "2.0"},
		Turns: []models.ConversationTurn{
			{TurnIndex: 0, UserQuery: "What is Go?", Answer: "A programming language."},
		},
	}

	judgeResults := []models.StageResult{
		{Name: "conversation-flow-judge", Score: 0.6, Reason: "acceptable", Duration: 200 * time.Millisecond},
	}

	mockRunner.EXPECT().Run(gomock.Any(), gomock.Any()).Return(judgeResults)
	mockAgg.EXPECT().Aggregate(gomock.Any(), nil, judgeResults).Return(models.EvaluationResult{
		Confidence: 0.6,
		Verdict:    models.VerdictReview,
		Stages:     judgeResults,
	})

	exec := NewConversationExecutor(mockRunner, repo, mockAgg, newTestLogger())
	result := exec.Execute(context.Background(), req)

	if result.TurnCount != 1 {
		t.Errorf("expected TurnCount=1, got %d", result.TurnCount)
	}
	if result.Verdict != models.VerdictReview {
		t.Errorf("expected verdict=review, got %s", result.Verdict)
	}
	if result.AgentVersion != "2.0" {
		t.Errorf("expected AgentVersion=2.0, got %s", result.AgentVersion)
	}
}

func TestConversationExecutor_Execute_FailVerdict(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRunner := mocks.NewMockJudgeRunner(ctrl)
	mockAgg := mocks.NewMockAggregator(ctrl)
	repo := setupTestRepository(t)

	req := models.ConversationEvaluationRequest{
		ConversationID: "conv-fail-001",
		Agent:          models.Agent{Name: "bad-agent", Version: "0.1"},
		Turns: []models.ConversationTurn{
			{TurnIndex: 0, UserQuery: "Q1", Answer: "A1"},
			{TurnIndex: 1, UserQuery: "Q2", Answer: "A2"},
			{TurnIndex: 2, UserQuery: "Q3", Answer: "A3"},
		},
	}

	judgeResults := []models.StageResult{
		{Name: "conversation-flow-judge", Score: 0.2, Reason: "incoherent", Duration: 300 * time.Millisecond},
	}

	mockRunner.EXPECT().Run(gomock.Any(), gomock.Any()).Return(judgeResults)
	mockAgg.EXPECT().Aggregate(gomock.Any(), nil, judgeResults).Return(models.EvaluationResult{
		Confidence: 0.2,
		Verdict:    models.VerdictFail,
		Stages:     judgeResults,
	})

	exec := NewConversationExecutor(mockRunner, repo, mockAgg, newTestLogger())
	result := exec.Execute(context.Background(), req)

	if result.TurnCount != 3 {
		t.Errorf("expected TurnCount=3, got %d", result.TurnCount)
	}
	if result.Verdict != models.VerdictFail {
		t.Errorf("expected verdict=fail, got %s", result.Verdict)
	}
	if result.Confidence != 0.2 {
		t.Errorf("expected confidence=0.2, got %.2f", result.Confidence)
	}
}

func TestConversationExecutor_Execute_MultipleJudges(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRunner := mocks.NewMockJudgeRunner(ctrl)
	mockAgg := mocks.NewMockAggregator(ctrl)
	repo := setupTestRepository(t)

	req := models.ConversationEvaluationRequest{
		ConversationID: "conv-multi-001",
		Agent:          models.Agent{Name: "agent", Version: "1.0"},
		Turns: []models.ConversationTurn{
			{TurnIndex: 0, UserQuery: "Q", Answer: "A"},
		},
	}

	judgeResults := []models.StageResult{
		{Name: "conversation-flow-judge", Score: 0.9, Reason: "great flow", Duration: 400 * time.Millisecond, Weight: 0.5},
		{Name: "topic-consistency-judge", Score: 0.8, Reason: "consistent topic", Duration: 350 * time.Millisecond, Weight: 0.5},
	}

	mockRunner.EXPECT().Run(gomock.Any(), gomock.Any()).Return(judgeResults)
	mockAgg.EXPECT().Aggregate(gomock.Any(), nil, judgeResults).Return(models.EvaluationResult{
		Confidence: 0.85,
		Verdict:    models.VerdictPass,
		Stages:     judgeResults,
	})

	exec := NewConversationExecutor(mockRunner, repo, mockAgg, newTestLogger())
	result := exec.Execute(context.Background(), req)

	if len(result.Stages) != 2 {
		t.Errorf("expected 2 stages, got %d", len(result.Stages))
	}
}
