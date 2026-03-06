package aggregator

import (
	"testing"
	"time"

	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/rs/zerolog"
)

func newTestLogger() *zerolog.Logger {
	logger := zerolog.Nop()
	return &logger
}

func TestAggregate_Pass(t *testing.T) {
	weights := Weights{PreChecks: 0.3, LLMJudge: 0.7}
	agg := NewAggregator(weights, VerdictThresholds{Pass: 0.8, Review: 0.5}, newTestLogger())

	stage1 := []models.StageResult{{Name: "precheck", Score: 0.8, Reason: "ok", Duration: 100 * time.Millisecond}}
	stage2 := []models.StageResult{{Name: "judge", Score: 0.9, Reason: "good", Duration: 1 * time.Second}}

	result := agg.Aggregate("test", stage1, stage2)

	// (0.8 * 0.3) + (0.9 * 0.7) = 0.87 > 0.8 → Pass
	if result.Verdict != models.VerdictPass {
		t.Errorf("expected Pass, got %s", result.Verdict)
	}
}

func TestAggregate_Review(t *testing.T) {
	weights := Weights{PreChecks: 0.3, LLMJudge: 0.7}
	agg := NewAggregator(weights, VerdictThresholds{Pass: 0.8, Review: 0.5}, newTestLogger())

	stage1 := []models.StageResult{{Name: "precheck", Score: 0.6, Reason: "ok", Duration: 100 * time.Millisecond}}
	stage2 := []models.StageResult{{Name: "judge", Score: 0.7, Reason: "ok", Duration: 1 * time.Second}}

	result := agg.Aggregate("test", stage1, stage2)

	// (0.6 * 0.3) + (0.7 * 0.7) = 0.67, 0.5 < 0.67 <= 0.8 → Review
	if result.Verdict != models.VerdictReview {
		t.Errorf("expected Review, got %s", result.Verdict)
	}
}

func TestAggregate_Fail(t *testing.T) {
	weights := Weights{PreChecks: 0.3, LLMJudge: 0.7}
	agg := NewAggregator(weights, VerdictThresholds{Pass: 0.8, Review: 0.5}, newTestLogger())

	stage1 := []models.StageResult{{Name: "precheck", Score: 0.2, Reason: "bad", Duration: 100 * time.Millisecond}}
	stage2 := []models.StageResult{{Name: "judge", Score: 0.4, Reason: "bad", Duration: 1 * time.Second}}

	result := agg.Aggregate("test", stage1, stage2)

	// (0.2 * 0.3) + (0.4 * 0.7) = 0.34 <= 0.5 → Fail
	if result.Verdict != models.VerdictFail {
		t.Errorf("expected Fail, got %s", result.Verdict)
	}
}

func TestAggregate_EmptyStages_Fail(t *testing.T) {
	weights := Weights{PreChecks: 0.3, LLMJudge: 0.7}
	agg := NewAggregator(weights, VerdictThresholds{Pass: 0.8, Review: 0.5}, newTestLogger())

	// Test empty stage1
	result := agg.Aggregate("test", []models.StageResult{}, []models.StageResult{{Name: "j", Score: 1.0, Reason: "ok", Duration: 1 * time.Second}})
	if result.Verdict != models.VerdictFail {
		t.Error("expected Fail for empty stage1")
	}

	// Test empty stage2
	result = agg.Aggregate("test", []models.StageResult{{Name: "p", Score: 1.0, Reason: "ok", Duration: 100 * time.Millisecond}}, []models.StageResult{})
	if result.Verdict != models.VerdictFail {
		t.Error("expected Fail for empty stage2")
	}
}
