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
		Metrics: models.AggregationMetrics{
			Stage1Avg:             0.8,
			Stage2WeightedAvg:     0.9,
			Stage2HarmonicMean:    0.88,
			Stage2Median:          0.89,
			Stage2WeightedProduct: 0.87,
			FinalConfidence:       0.85,
			MethodUsed:            models.MethodWeightedAverage,
		},
	}

	err := writer.Write(result)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	got := buf.String()
	// Verify it's valid JSON with expected fields
	if !strings.Contains(got, `"id":"test-001"`) {
		t.Error("missing id field")
	}
	if !strings.Contains(got, `"confidence":0.85`) {
		t.Error("missing confidence field")
	}
	if !strings.Contains(got, `"verdict":"pass"`) {
		t.Error("missing verdict field")
	}
	if !strings.Contains(got, `"metrics"`) {
		t.Error("missing metrics field")
	}
	if !strings.Contains(got, `"stage2_weighted_avg":0.9`) {
		t.Error("missing stage2_weighted_avg in metrics")
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
