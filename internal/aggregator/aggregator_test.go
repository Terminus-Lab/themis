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
	aggConfig := AggregationConfig{
		EnablePrecheck:         true,
		JudgeAggregationMethod: "weighted_average",
	}

	agg := NewAggregator(weights, VerdictThresholds{Pass: 0.8, Review: 0.5}, aggConfig, newTestLogger())

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

	aggConfig := AggregationConfig{
		EnablePrecheck:         true,
		JudgeAggregationMethod: "weighted_average",
	}

	agg := NewAggregator(weights, VerdictThresholds{Pass: 0.8, Review: 0.5}, aggConfig, newTestLogger())

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
	aggConfig := AggregationConfig{
		EnablePrecheck:         true,
		JudgeAggregationMethod: "weighted_average",
	}
	agg := NewAggregator(weights, VerdictThresholds{Pass: 0.8, Review: 0.5}, aggConfig, newTestLogger())

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
	aggConfig := AggregationConfig{
		EnablePrecheck:         true,
		JudgeAggregationMethod: "weighted_average",
	}

	agg := NewAggregator(weights, VerdictThresholds{Pass: 0.8, Review: 0.5}, aggConfig, newTestLogger())

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

// Test individual aggregation methods
func TestCalculateWeightedAverage(t *testing.T) {
	stages := []models.StageResult{
		{Score: 0.9, Weight: 0.5},
		{Score: 0.5, Weight: 0.3},
		{Score: 0.7, Weight: 0.2},
	}
	// (0.9*0.5 + 0.5*0.3 + 0.7*0.2) / 1.0 = 0.74
	result := calculateWeightedAverage(stages)
	expected := 0.74
	if abs(result-expected) > 0.01 {
		t.Errorf("expected %.2f, got %.2f", expected, result)
	}
}

func TestCalculateWeightedAverage_NoWeights(t *testing.T) {
	stages := []models.StageResult{
		{Score: 0.6, Weight: 0},
		{Score: 0.8, Weight: 0},
	}
	// Should fallback to simple average: (0.6 + 0.8) / 2 = 0.7
	result := calculateWeightedAverage(stages)
	expected := 0.7
	if abs(result-expected) > 0.01 {
		t.Errorf("expected %.2f, got %.2f", expected, result)
	}
}

func TestCalculateHarmonicMean(t *testing.T) {
	stages := []models.StageResult{
		{Score: 0.8, Weight: 1.0},
		{Score: 0.6, Weight: 1.0},
		{Score: 0.9, Weight: 1.0},
	}
	// H = 3 / (1/0.8 + 1/0.6 + 1/0.9) ≈ 0.744
	result := calculateHarmonicMean(stages)
	expected := 0.744
	if abs(result-expected) > 0.01 {
		t.Errorf("expected %.3f, got %.3f", expected, result)
	}
}

func TestCalculateHarmonicMean_ZeroScore(t *testing.T) {
	stages := []models.StageResult{
		{Score: 0.8, Weight: 1.0},
		{Score: 0.0, Weight: 1.0}, // Zero score
	}
	result := calculateHarmonicMean(stages)
	if result != 0.0 {
		t.Errorf("expected 0.0 for zero score, got %.3f", result)
	}
}

func TestCalculateMedian_OddCount(t *testing.T) {
	stages := []models.StageResult{
		{Score: 0.5},
		{Score: 0.9},
		{Score: 0.7},
	}
	// Sorted: [0.5, 0.7, 0.9], median = 0.7
	result := calculateMedian(stages)
	expected := 0.7
	if abs(result-expected) > 0.01 {
		t.Errorf("expected %.2f, got %.2f", expected, result)
	}
}

func TestCalculateMedian_EvenCount(t *testing.T) {
	stages := []models.StageResult{
		{Score: 0.6},
		{Score: 0.8},
		{Score: 0.4},
		{Score: 0.9},
	}
	// Sorted: [0.4, 0.6, 0.8, 0.9], median = (0.6 + 0.8) / 2 = 0.7
	result := calculateMedian(stages)
	expected := 0.7
	if abs(result-expected) > 0.01 {
		t.Errorf("expected %.2f, got %.2f", expected, result)
	}
}

func TestCalculateMedian_SingleScore(t *testing.T) {
	stages := []models.StageResult{
		{Score: 0.85},
	}
	result := calculateMedian(stages)
	expected := 0.85
	if abs(result-expected) > 0.01 {
		t.Errorf("expected %.2f, got %.2f", expected, result)
	}
}

func TestCalculateWeightedProduct(t *testing.T) {
	stages := []models.StageResult{
		{Score: 0.8, Weight: 1.0},
		{Score: 0.9, Weight: 1.0},
	}
	// Normalized weights: 0.5, 0.5
	// Product: 0.8^0.5 * 0.9^0.5 ≈ 0.849
	result := calculateWeightedProduct(stages)
	expected := 0.849
	if abs(result-expected) > 0.01 {
		t.Errorf("expected %.3f, got %.3f", expected, result)
	}
}

func TestCalculateWeightedProduct_OneLowScore(t *testing.T) {
	stages := []models.StageResult{
		{Score: 0.9, Weight: 1.0},
		{Score: 0.9, Weight: 1.0},
		{Score: 0.2, Weight: 1.0}, // Low score tanks result
	}
	// 0.9^(1/3) * 0.9^(1/3) * 0.2^(1/3) ≈ 0.545
	result := calculateWeightedProduct(stages)
	expected := 0.545
	if abs(result-expected) > 0.01 {
		t.Errorf("expected %.3f, got %.3f", expected, result)
	}
}

// Test full aggregation with different methods
func TestAggregate_AllMethods(t *testing.T) {
	stage1 := []models.StageResult{
		{Name: "precheck1", Score: 0.8, Weight: 1.0},
		{Name: "precheck2", Score: 0.9, Weight: 1.0},
	}
	stage2 := []models.StageResult{
		{Name: "judge1", Score: 0.9, Weight: 0.4},
		{Name: "judge2", Score: 0.8, Weight: 0.3},
		{Name: "judge3", Score: 0.7, Weight: 0.3},
	}

	methods := []models.AggregationMethod{
		models.MethodWeightedAverage,
		models.MethodHarmonicMean,
		models.MethodMedian,
		models.MethodWeightedProduct,
	}

	weights := Weights{PreChecks: 0.3, LLMJudge: 0.7}
	thresholds := VerdictThresholds{Pass: 0.8, Review: 0.5}

	for _, method := range methods {
		t.Run(string(method), func(t *testing.T) {
			config := AggregationConfig{
				EnablePrecheck:         true,
				JudgeAggregationMethod: method,
			}
			agg := NewAggregator(weights, thresholds, config, newTestLogger())
			result := agg.Aggregate("test", stage1, stage2)

			// Verify all metrics are populated
			if result.Metrics.Stage1Avg == 0 {
				t.Error("Stage1Avg should be populated")
			}
			if result.Metrics.Stage2WeightedAvg == 0 {
				t.Error("Stage2WeightedAvg should be populated")
			}
			if result.Metrics.Stage2HarmonicMean == 0 {
				t.Error("Stage2HarmonicMean should be populated")
			}
			if result.Metrics.Stage2Median == 0 {
				t.Error("Stage2Median should be populated")
			}
			if result.Metrics.Stage2WeightedProduct == 0 {
				t.Error("Stage2WeightedProduct should be populated")
			}
			if result.Metrics.MethodUsed != method {
				t.Errorf("Expected method %s, got %s", method, result.Metrics.MethodUsed)
			}
			if result.Confidence == 0 {
				t.Error("Confidence should be non-zero")
			}
			if result.Metrics.FinalConfidence != result.Confidence {
				t.Error("FinalConfidence should match Confidence")
			}

			t.Logf("Method: %s, Confidence: %.3f, Verdict: %s", method, result.Confidence, result.Verdict)
		})
	}
}

// Test precheck disabled
func TestAggregate_PrecheckDisabled(t *testing.T) {
	stage1 := []models.StageResult{
		{Name: "precheck", Score: 0.1, Weight: 1.0}, // Very low, but should be ignored
	}
	stage2 := []models.StageResult{
		{Name: "judge", Score: 0.9, Weight: 1.0},
	}

	config := AggregationConfig{
		EnablePrecheck:         false, // Disabled
		JudgeAggregationMethod: models.MethodWeightedAverage,
	}
	weights := Weights{PreChecks: 0.3, LLMJudge: 0.7}
	thresholds := VerdictThresholds{Pass: 0.8, Review: 0.5}

	agg := NewAggregator(weights, thresholds, config, newTestLogger())
	result := agg.Aggregate("test", stage1, stage2)

	// With precheck disabled, confidence should equal stage2 score
	if abs(result.Confidence-0.9) > 0.01 {
		t.Errorf("Expected confidence 0.9 (stage2 only), got %.2f", result.Confidence)
	}

	if result.Verdict != models.VerdictPass {
		t.Errorf("Expected Pass with 0.9 confidence, got %s", result.Verdict)
	}

	// Stage1Avg should be 0 when disabled
	if result.Metrics.Stage1Avg != 0 {
		t.Errorf("Expected Stage1Avg = 0 when disabled, got %.2f", result.Metrics.Stage1Avg)
	}
}

// Test that different methods produce different results
func TestAggregate_MethodsDiffer(t *testing.T) {
	stage1 := []models.StageResult{{Name: "p", Score: 0.8, Weight: 1.0}}
	// Deliberately varied scores with wide variance
	stage2 := []models.StageResult{
		{Name: "j1", Score: 0.95, Weight: 0.4},
		{Name: "j2", Score: 0.4, Weight: 0.3},
		{Name: "j3", Score: 0.75, Weight: 0.3},
	}

	config := AggregationConfig{
		EnablePrecheck:         true,
		JudgeAggregationMethod: models.MethodWeightedAverage,
	}
	weights := Weights{PreChecks: 0.3, LLMJudge: 0.7}
	thresholds := VerdictThresholds{Pass: 0.8, Review: 0.5}

	agg := NewAggregator(weights, thresholds, config, newTestLogger())
	result := agg.Aggregate("test", stage1, stage2)

	// All methods should produce different results with varied scores
	metrics := result.Metrics
	values := map[string]float64{
		"WeightedAvg":     metrics.Stage2WeightedAvg,
		"HarmonicMean":    metrics.Stage2HarmonicMean,
		"Median":          metrics.Stage2Median,
		"WeightedProduct": metrics.Stage2WeightedProduct,
	}

	// Check all are different (within tolerance)
	names := []string{"WeightedAvg", "HarmonicMean", "Median", "WeightedProduct"}
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			diff := abs(values[names[i]] - values[names[j]])
			if diff < 0.001 {
				t.Errorf("Methods %s and %s produced nearly identical results: %.3f vs %.3f",
					names[i], names[j], values[names[i]], values[names[j]])
			}
		}
	}

	t.Logf("Stage2 scores: WeightedAvg=%.3f, HarmonicMean=%.3f, Median=%.3f, WeightedProduct=%.3f",
		metrics.Stage2WeightedAvg, metrics.Stage2HarmonicMean,
		metrics.Stage2Median, metrics.Stage2WeightedProduct)
}

// Helper function
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
