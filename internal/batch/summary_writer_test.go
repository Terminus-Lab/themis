package batch

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/rs/zerolog"
)

func TestSummaryWriter(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.Nop()
	writer := NewSummaryWriter(&buf, &logger)

	// Write mix of pass, fail, review
	writer.Write(models.EvaluationResult{ID: "1", Verdict: models.VerdictPass, Confidence: 0.9})
	writer.Write(models.EvaluationResult{ID: "2", Verdict: models.VerdictFail, Confidence: 0.3})
	writer.Write(models.EvaluationResult{ID: "3", Verdict: models.VerdictReview, Confidence: 0.6})

	err := writer.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	var stats SummaryStats
	if err := json.Unmarshal(buf.Bytes(), &stats); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if stats.Total != 3 {
		t.Errorf("Total: got %d, want 3", stats.Total)
	}
	if stats.PassCount != 1 {
		t.Errorf("PassCount: got %d, want 1", stats.PassCount)
	}
	if stats.FailCount != 1 {
		t.Errorf("FailCount: got %d, want 1", stats.FailCount)
	}
	if stats.ReviewCount != 1 {
		t.Errorf("ReviewCount: got %d, want 1", stats.ReviewCount)
	}
	wantAvg := (0.9 + 0.3 + 0.6) / 3
	if stats.AvgConfidence != wantAvg {
		t.Errorf("AvgConfidence: got %v, want %v", stats.AvgConfidence, wantAvg)
	}
}
