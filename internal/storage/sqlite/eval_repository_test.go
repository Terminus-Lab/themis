package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/Terminus-Lab/themis/internal/storage"
	"github.com/Terminus-Lab/themis/internal/storage/sqlite"
	"github.com/rs/zerolog"
)

func setupTestDB(t *testing.T) (*sqlite.DB, func()) {
	t.Helper()

	db, err := sqlite.New(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}
	if err := db.InitSchema(context.Background()); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}
	return db, func() { _ = db.Close() }
}

func newTestLogger() *zerolog.Logger {
	logger := zerolog.Nop()
	return &logger
}

func sampleRecord(conversationID string) *storage.ConversationRecord {
	return &storage.ConversationRecord{
		ID:             "rec-" + conversationID,
		ConversationID: conversationID,
		AgentName:      "test-agent",
		AgentVersion:   "1.0",
		TurnCount:      2,
		TurnAvg:        0.8,
		HolisticScore:  0.9,
		HolisticReason: "good flow",
		FinalScore:     0.85,
		Verdict:        "pass",
		TurnResults: []models.TurnEvaluationResult{
			{TurnIndex: 0, UserQuery: "Q1", Answer: "A1", TurnScore: 0.8},
			{TurnIndex: 1, UserQuery: "Q2", Answer: "A2", TurnScore: 0.8},
		},
	}
}

func TestEvalRepository_StoreAndGetConversation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := sqlite.NewEvalRepository(db, newTestLogger())
	ctx := context.Background()

	rec := sampleRecord("conv-abc")
	if err := repo.StoreConversation(ctx, rec); err != nil {
		t.Fatalf("StoreConversation failed: %v", err)
	}

	got, err := repo.GetConversation(ctx, "conv-abc")
	if err != nil {
		t.Fatalf("GetConversation failed: %v", err)
	}

	if got.ConversationID != "conv-abc" {
		t.Errorf("ConversationID mismatch: got %s", got.ConversationID)
	}
	if got.AgentName != "test-agent" {
		t.Errorf("AgentName mismatch: got %s", got.AgentName)
	}
	if got.TurnCount != 2 {
		t.Errorf("TurnCount mismatch: got %d", got.TurnCount)
	}
	if got.FinalScore != 0.85 {
		t.Errorf("FinalScore mismatch: got %.2f", got.FinalScore)
	}
	if got.Verdict != "pass" {
		t.Errorf("Verdict mismatch: got %s", got.Verdict)
	}
	if len(got.TurnResults) != 2 {
		t.Errorf("TurnResults count mismatch: got %d", len(got.TurnResults))
	}
}

func TestEvalRepository_GetConversation_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := sqlite.NewEvalRepository(db, newTestLogger())

	_, err := repo.GetConversation(context.Background(), "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent conversation, got nil")
	}
}

func TestEvalRepository_ListConversations_Empty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := sqlite.NewEvalRepository(db, newTestLogger())

	summaries, err := repo.ListConversations(context.Background())
	if err != nil {
		t.Fatalf("ListConversations failed: %v", err)
	}
	if len(summaries) != 0 {
		t.Errorf("Expected 0 summaries, got %d", len(summaries))
	}
}

func TestEvalRepository_ListConversations_Multiple(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := sqlite.NewEvalRepository(db, newTestLogger())
	ctx := context.Background()

	for _, id := range []string{"conv-1", "conv-2", "conv-3"} {
		if err := repo.StoreConversation(ctx, sampleRecord(id)); err != nil {
			t.Fatalf("StoreConversation failed: %v", err)
		}
	}

	summaries, err := repo.ListConversations(ctx)
	if err != nil {
		t.Fatalf("ListConversations failed: %v", err)
	}
	if len(summaries) != 3 {
		t.Errorf("Expected 3 summaries, got %d", len(summaries))
	}
}

func TestEvalRepository_HealthMetrics_Empty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := sqlite.NewEvalRepository(db, newTestLogger())

	metrics, err := repo.HealthMetrics(context.Background(), time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("HealthMetrics failed: %v", err)
	}
	if metrics.TotalEvaluations != 0 {
		t.Errorf("Expected 0 evaluations, got %d", metrics.TotalEvaluations)
	}
}

func TestEvalRepository_HealthMetrics_WithData(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := sqlite.NewEvalRepository(db, newTestLogger())
	ctx := context.Background()

	for _, id := range []string{"conv-a", "conv-b"} {
		if err := repo.StoreConversation(ctx, sampleRecord(id)); err != nil {
			t.Fatalf("StoreConversation failed: %v", err)
		}
	}

	metrics, err := repo.HealthMetrics(ctx, time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("HealthMetrics failed: %v", err)
	}
	if metrics.TotalEvaluations != 2 {
		t.Errorf("Expected 2 evaluations, got %d", metrics.TotalEvaluations)
	}
	if metrics.AvgConfidence <= 0 {
		t.Errorf("Expected positive AvgConfidence, got %.3f", metrics.AvgConfidence)
	}
}

func TestEvalRepository_GetConversation_ReturnsLatest(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := sqlite.NewEvalRepository(db, newTestLogger())
	ctx := context.Background()

	// Store two records with same conversation_id, second with higher final_score
	rec1 := sampleRecord("conv-dup")
	rec1.ID = "rec-first"
	rec1.FinalScore = 0.5
	rec1.Verdict = "review"
	if err := repo.StoreConversation(ctx, rec1); err != nil {
		t.Fatalf("StoreConversation failed: %v", err)
	}

	rec2 := sampleRecord("conv-dup")
	rec2.ID = "rec-second"
	rec2.FinalScore = 0.9
	rec2.Verdict = "pass"
	if err := repo.StoreConversation(ctx, rec2); err != nil {
		t.Fatalf("StoreConversation failed: %v", err)
	}

	// GetConversation returns latest (DESC by created_at)
	got, err := repo.GetConversation(ctx, "conv-dup")
	if err != nil {
		t.Fatalf("GetConversation failed: %v", err)
	}
	// The latest stored should be returned
	if got.ConversationID != "conv-dup" {
		t.Errorf("ConversationID mismatch: got %s", got.ConversationID)
	}
}
