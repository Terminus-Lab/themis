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

	// Use in-memory SQLite database
	db, err := sqlite.New(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}

	// Initialize schema
	if err := db.InitSchema(context.Background()); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func newTestLogger() *zerolog.Logger {
	logger := zerolog.Nop()
	return &logger
}

func TestEvalRepository_Store_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := sqlite.NewEvalRepository(db, newTestLogger())

	eval := &storage.Evaluation{
		EventID:      "evt-001",
		AgentName:    "test-agent",
		AgentVersion: "v1.0.0",
		UserQuery:    "What is Go?",
		Answer:       "Go is a programming language",
		Context:      "Go documentation",
		Confidence:   0.85,
		Verdict:      "pass",
		StageScores: []models.StageResult{
			{Name: "relevance", Score: 0.9, Reason: "relevant", Duration: 100 * time.Millisecond, Weight: 0.3},
			{Name: "faithfulness", Score: 0.8, Reason: "faithful", Duration: 200 * time.Millisecond, Weight: 0.7},
		},
	}

	err := repo.Store(context.Background(), eval)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Verify stored data by querying back
	results, count, err := repo.Query(context.Background(), models.QueryFilters{
		AgentName: "test-agent",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	stored := results[0]
	if stored.EventID != eval.EventID {
		t.Errorf("EventID mismatch: expected %s, got %s", eval.EventID, stored.EventID)
	}
	if stored.AgentName != eval.AgentName {
		t.Errorf("AgentName mismatch: expected %s, got %s", eval.AgentName, stored.AgentName)
	}
	if stored.Confidence != eval.Confidence {
		t.Errorf("Confidence mismatch: expected %.2f, got %.2f", eval.Confidence, stored.Confidence)
	}
	if len(stored.StageScores) != 2 {
		t.Errorf("Expected 2 stage scores, got %d", len(stored.StageScores))
	}
}

func TestEvalRepository_Query_FilterByAgentName(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := sqlite.NewEvalRepository(db, newTestLogger())

	// Insert test data
	testData := []storage.Evaluation{
		{
			EventID:      "evt-001",
			AgentName:    "agent-a",
			AgentVersion: "v1.0",
			UserQuery:    "query1",
			Answer:       "answer1",
			Confidence:   0.8,
			Verdict:      "pass",
			StageScores:  []models.StageResult{},
		},
		{
			EventID:      "evt-002",
			AgentName:    "agent-b",
			AgentVersion: "v1.0",
			UserQuery:    "query2",
			Answer:       "answer2",
			Confidence:   0.7,
			Verdict:      "fail",
			StageScores:  []models.StageResult{},
		},
		{
			EventID:      "evt-003",
			AgentName:    "agent-a",
			AgentVersion: "v2.0",
			UserQuery:    "query3",
			Answer:       "answer3",
			Confidence:   0.9,
			Verdict:      "pass",
			StageScores:  []models.StageResult{},
		},
	}

	for _, eval := range testData {
		if err := repo.Store(context.Background(), &eval); err != nil {
			t.Fatalf("Failed to store test data: %v", err)
		}
	}

	// Query by agent name
	results, count, err := repo.Query(context.Background(), models.QueryFilters{
		AgentName: "agent-a",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// Verify all results are for agent-a
	for _, result := range results {
		if result.AgentName != "agent-a" {
			t.Errorf("Expected agent-a, got %s", result.AgentName)
		}
	}
}

func TestEvalRepository_Query_FilterByVerdict(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := sqlite.NewEvalRepository(db, newTestLogger())

	// Insert test data with different verdicts
	testData := []storage.Evaluation{
		{EventID: "evt-001", AgentName: "agent", AgentVersion: "v1", UserQuery: "q1", Answer: "a1", Confidence: 0.9, Verdict: "pass", StageScores: []models.StageResult{}},
		{EventID: "evt-002", AgentName: "agent", AgentVersion: "v1", UserQuery: "q2", Answer: "a2", Confidence: 0.4, Verdict: "fail", StageScores: []models.StageResult{}},
		{EventID: "evt-003", AgentName: "agent", AgentVersion: "v1", UserQuery: "q3", Answer: "a3", Confidence: 0.6, Verdict: "review", StageScores: []models.StageResult{}},
		{EventID: "evt-004", AgentName: "agent", AgentVersion: "v1", UserQuery: "q4", Answer: "a4", Confidence: 0.3, Verdict: "fail", StageScores: []models.StageResult{}},
	}

	for _, eval := range testData {
		if err := repo.Store(context.Background(), &eval); err != nil {
			t.Fatalf("Failed to store test data: %v", err)
		}
	}

	// Query for fail verdicts
	results, count, err := repo.Query(context.Background(), models.QueryFilters{
		Verdict: "fail",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	for _, result := range results {
		if result.Verdict != "fail" {
			t.Errorf("Expected verdict 'fail', got %s", result.Verdict)
		}
	}
}

func TestEvalRepository_Query_Pagination(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := sqlite.NewEvalRepository(db, newTestLogger())

	// Insert 10 test records
	for i := 1; i <= 10; i++ {
		eval := storage.Evaluation{
			EventID:      "evt-" + string(rune('0'+i)),
			AgentName:    "agent",
			AgentVersion: "v1",
			UserQuery:    "query",
			Answer:       "answer",
			Confidence:   0.8,
			Verdict:      "pass",
			StageScores:  []models.StageResult{},
		}
		if err := repo.Store(context.Background(), &eval); err != nil {
			t.Fatalf("Failed to store test data: %v", err)
		}
	}

	// Test pagination: first page
	results, count, err := repo.Query(context.Background(), models.QueryFilters{
		Limit:  3,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if count != 10 {
		t.Errorf("Expected total count 10, got %d", count)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results in first page, got %d", len(results))
	}

	// Test pagination: second page
	results, count, err = repo.Query(context.Background(), models.QueryFilters{
		Limit:  3,
		Offset: 3,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if count != 10 {
		t.Errorf("Expected total count 10, got %d", count)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results in second page, got %d", len(results))
	}
}

func TestEvalRepository_Query_EmptyResults(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := sqlite.NewEvalRepository(db, newTestLogger())

	// Query empty database
	results, count, err := repo.Query(context.Background(), models.QueryFilters{
		AgentName: "nonexistent",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestEvalRepository_QueryById_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := sqlite.NewEvalRepository(db, newTestLogger())

	// Store test data
	eval := &storage.Evaluation{
		EventID:      "evt-123",
		AgentName:    "test-agent",
		AgentVersion: "v1.0.0",
		UserQuery:    "test query",
		Answer:       "test answer",
		Confidence:   0.85,
		Verdict:      "pass",
		StageScores: []models.StageResult{
			{Name: "test", Score: 0.9, Reason: "good", Duration: 100 * time.Millisecond},
		},
	}

	if err := repo.Store(context.Background(), eval); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Query by ID
	result, err := repo.QueryById(context.Background(), "evt-123")
	if err != nil {
		t.Fatalf("QueryById failed: %v", err)
	}

	if result.EventID != eval.EventID {
		t.Errorf("EventID mismatch: expected %s, got %s", eval.EventID, result.EventID)
	}
	if result.AgentName != eval.AgentName {
		t.Errorf("AgentName mismatch: expected %s, got %s", eval.AgentName, result.AgentName)
	}
	if result.Confidence != eval.Confidence {
		t.Errorf("Confidence mismatch: expected %.2f, got %.2f", eval.Confidence, result.Confidence)
	}
}

func TestEvalRepository_QueryById_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := sqlite.NewEvalRepository(db, newTestLogger())

	// Query non-existent ID
	_, err := repo.QueryById(context.Background(), "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent ID, got nil")
	}
	// Error is wrapped by repository, just check it's not nil
}

func TestEvalRepository_Store_DuplicateEventID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := sqlite.NewEvalRepository(db, newTestLogger())

	eval := &storage.Evaluation{
		EventID:      "evt-duplicate",
		AgentName:    "agent",
		AgentVersion: "v1",
		UserQuery:    "query",
		Answer:       "answer",
		Confidence:   0.8,
		Verdict:      "pass",
		StageScores:  []models.StageResult{},
	}

	// Store once - should succeed
	if err := repo.Store(context.Background(), eval); err != nil {
		t.Fatalf("First store failed: %v", err)
	}

	// Store again with same event_id - should succeed (SQLite generates unique id)
	// Note: event_id is NOT the primary key, id is
	if err := repo.Store(context.Background(), eval); err != nil {
		t.Fatalf("Second store failed: %v", err)
	}

	// Verify both records exist
	_, count, err := repo.Query(context.Background(), models.QueryFilters{Limit: 10})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 records with same event_id, got %d", count)
	}
}
