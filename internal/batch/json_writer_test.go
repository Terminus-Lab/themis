package batch

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/rs/zerolog"
)

func TestJSONLWriter_Write(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.Nop()
	writer := NewJSONLWriter(&buf, &logger)

	result := models.EvaluationResult{
		ID:         "test-001",
		Stages:     []models.StageResult{},
		Confidence: 0.85,
		Verdict:    models.VerdictPass,
	}

	err := writer.Write(result)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	got := buf.String()
	want := `{"id":"test-001","stages":[],"confidence":0.85,"verdict":"pass"}` + "\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestJSONLWriter_MultipleWrites(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.Nop()
	writer := NewJSONLWriter(&buf, &logger)

	writer.Write(models.EvaluationResult{ID: "1", Verdict: models.VerdictPass})
	writer.Write(models.EvaluationResult{ID: "2", Verdict: models.VerdictFail})

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}
