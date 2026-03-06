package batch

import (
	"context"
	"fmt"
	"testing"

	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/rs/zerolog"
)

// Mock executor for testing
type mockExecutor struct {
	called int
}

func (m *mockExecutor) Execute(ctx context.Context, evalCtx models.EvaluationContext) models.EvaluationResult {
	m.called++
	return models.EvaluationResult{
		ID:      evalCtx.RequestID,
		Verdict: models.VerdictPass,
	}
}

func TestProcessor_Process(t *testing.T) {
	logger := zerolog.Nop()
	executor := &mockExecutor{}
	processor := NewProcessor(executor, 2, &logger)

	records := []InputRecord{
		{LineNumber: 1, Request: models.EvaluationRequest{EventID: "1"}},
		{LineNumber: 2, Request: models.EvaluationRequest{EventID: "2"}},
		{LineNumber: 3, Request: models.EvaluationRequest{EventID: "3"}},
	}

	ctx := context.Background()
	results := processor.Process(ctx, records)

	count := 0
	for range results {
		count++
	}

	if count != 3 {
		t.Errorf("expected 3 results, got %d", count)
	}

	if executor.called != 3 {
		t.Errorf("expected executor called 3 times, got %d", executor.called)
	}
}

func TestProcessor_SkipsErrorRecords(t *testing.T) {
	logger := zerolog.Nop()
	executor := &mockExecutor{}
	processor := NewProcessor(executor, 2, &logger)

	records := []InputRecord{
		{LineNumber: 1, Request: models.EvaluationRequest{EventID: "1"}},
		{LineNumber: 2, Error: fmt.Errorf("parse error")}, // Should skip
		{LineNumber: 3, Request: models.EvaluationRequest{EventID: "3"}},
	}

	ctx := context.Background()
	results := processor.Process(ctx, records)

	count := 0
	for range results {
		count++
	}

	// Only 2 valid records processed
	if count != 2 {
		t.Errorf("expected 2 results, got %d", count)
	}

	if executor.called != 2 {
		t.Errorf("expected executor called 2 times, got %d", executor.called)
	}
}
