package batch

import (
	"context"
	"errors"
	"testing"

	"github.com/Terminus-Lab/themis/internal/models"
)

// mockConversationExecutor is a simple in-process executor for tests.
type mockConversationExecutor struct {
	result models.ConversationEvaluationResult
}

func (m *mockConversationExecutor) Execute(_ context.Context, req models.ConversationEvaluationRequest) models.ConversationEvaluationResult {
	m.result.ConversationID = req.ConversationID
	m.result.TurnCount = len(req.Turns)
	return m.result
}

func TestConversationProcessor_Process_AllValid(t *testing.T) {
	exec := &mockConversationExecutor{
		result: models.ConversationEvaluationResult{
			Verdict:    models.VerdictPass,
			Confidence: 0.9,
		},
	}

	processor := NewConversationProcessor(exec, 2, nopLogger())

	records := []ConversationInputRecord{
		{
			LineNumber: 1,
			Request: models.ConversationEvaluationRequest{
				ConversationID: "conv-a",
				Turns:          []models.ConversationTurn{{UserQuery: "Q", Answer: "A"}},
			},
		},
		{
			LineNumber: 2,
			Request: models.ConversationEvaluationRequest{
				ConversationID: "conv-b",
				Turns:          []models.ConversationTurn{{UserQuery: "Q2", Answer: "A2"}},
			},
		},
		{
			LineNumber: 3,
			Request: models.ConversationEvaluationRequest{
				ConversationID: "conv-c",
				Turns:          []models.ConversationTurn{{UserQuery: "Q3", Answer: "A3"}},
			},
		},
	}

	resultsCh := processor.Process(context.Background(), records)

	var results []models.ConversationEvaluationResult
	for r := range resultsCh {
		results = append(results, r)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestConversationProcessor_Process_SkipsErrorRecords(t *testing.T) {
	exec := &mockConversationExecutor{
		result: models.ConversationEvaluationResult{
			Verdict:    models.VerdictPass,
			Confidence: 0.9,
		},
	}

	processor := NewConversationProcessor(exec, 2, nopLogger())

	records := []ConversationInputRecord{
		{LineNumber: 1, Request: models.ConversationEvaluationRequest{ConversationID: "conv-ok"}},
		{LineNumber: 2, Error: errors.New("parse error on line 2")},
		{LineNumber: 3, Request: models.ConversationEvaluationRequest{ConversationID: "conv-ok-2"}},
	}

	resultsCh := processor.Process(context.Background(), records)

	var results []models.ConversationEvaluationResult
	for r := range resultsCh {
		results = append(results, r)
	}

	// Error record is skipped, so only 2 results
	if len(results) != 2 {
		t.Errorf("expected 2 results (error record skipped), got %d", len(results))
	}
}

func TestConversationProcessor_Process_AllErrors(t *testing.T) {
	exec := &mockConversationExecutor{}
	processor := NewConversationProcessor(exec, 1, nopLogger())

	records := []ConversationInputRecord{
		{LineNumber: 1, Error: errors.New("bad line 1")},
		{LineNumber: 2, Error: errors.New("bad line 2")},
	}

	resultsCh := processor.Process(context.Background(), records)

	var results []models.ConversationEvaluationResult
	for r := range resultsCh {
		results = append(results, r)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results when all records have errors, got %d", len(results))
	}
}

func TestConversationProcessor_Process_Empty(t *testing.T) {
	exec := &mockConversationExecutor{}
	processor := NewConversationProcessor(exec, 3, nopLogger())

	resultsCh := processor.Process(context.Background(), []ConversationInputRecord{})

	var results []models.ConversationEvaluationResult
	for r := range resultsCh {
		results = append(results, r)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results for empty input, got %d", len(results))
	}
}

func TestConversationProcessor_Process_SingleWorker(t *testing.T) {
	callCount := 0
	exec := &countingConversationExecutor{count: &callCount}
	processor := NewConversationProcessor(exec, 1, nopLogger())

	records := make([]ConversationInputRecord, 5)
	for i := range records {
		records[i] = ConversationInputRecord{
			LineNumber: i + 1,
			Request:    models.ConversationEvaluationRequest{ConversationID: "conv"},
		}
	}

	resultsCh := processor.Process(context.Background(), records)
	for range resultsCh {
	}

	if callCount != 5 {
		t.Errorf("expected executor called 5 times, got %d", callCount)
	}
}

type countingConversationExecutor struct {
	count *int
}

func (c *countingConversationExecutor) Execute(_ context.Context, req models.ConversationEvaluationRequest) models.ConversationEvaluationResult {
	*c.count++
	return models.ConversationEvaluationResult{ConversationID: req.ConversationID}
}
