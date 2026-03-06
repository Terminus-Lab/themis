package batch

import (
	"testing"

	"github.com/Terminus-Lab/themis/internal/models"
)

func TestVerdictToRank(t *testing.T) {
	tests := []struct {
		name     string
		verdict  string
		expected int
	}{
		{"pass", "pass", 2},
		{"review", "review", 1},
		{"fail", "fail", 0},
		{"invalid", "invalid", -1},
		{"empty", "", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := verdictToRank(tt.verdict)
			if result != tt.expected {
				t.Errorf("verdictToRank(%q) = %d, want %d", tt.verdict, result, tt.expected)
			}
		})
	}
}

func TestComputeKendallTau_PerfectAgreement(t *testing.T) {
	pairs := []AnnotationPair{
		{"1", "pass", models.VerdictPass, 0.9},
		{"2", "pass", models.VerdictPass, 0.85},
		{"3", "review", models.VerdictReview, 0.65},
		{"4", "fail", models.VerdictFail, 0.3},
		{"5", "fail", models.VerdictFail, 0.15},
	}

	tau, err := ComputeKendallTau(pairs)
	if err != nil {
		t.Fatalf("ComputeKendallTau failed: %v", err)
	}

	// Perfect agreement should give high positive tau (ties present, so not exactly 1.0)
	if tau < 0.7 {
		t.Errorf("Expected tau >= 0.7 for perfect agreement, got %f", tau)
	}
}

func TestComputeKendallTau_CompleteDisagreement(t *testing.T) {
	pairs := []AnnotationPair{
		{"1", "pass", models.VerdictFail, 0.3},
		{"2", "pass", models.VerdictFail, 0.2},
		{"3", "review", models.VerdictFail, 0.1},
		{"4", "fail", models.VerdictPass, 0.9},
		{"5", "fail", models.VerdictPass, 0.85},
	}

	tau, err := ComputeKendallTau(pairs)
	if err != nil {
		t.Fatalf("ComputeKendallTau failed: %v", err)
	}

	// Complete disagreement should give strong negative tau (ties present, so not exactly -1.0)
	if tau > -0.5 {
		t.Errorf("Expected tau <= -0.5 for complete disagreement, got %f", tau)
	}
}

func TestComputeKendallTau_ModerateAgreement(t *testing.T) {
	pairs := []AnnotationPair{
		{"1", "pass", models.VerdictPass, 0.9},
		{"2", "pass", models.VerdictReview, 0.7},  // Disagreement
		{"3", "review", models.VerdictReview, 0.65},
		{"4", "fail", models.VerdictFail, 0.3},
		{"5", "fail", models.VerdictReview, 0.55}, // Disagreement
	}

	tau, err := ComputeKendallTau(pairs)
	if err != nil {
		t.Fatalf("ComputeKendallTau failed: %v", err)
	}

	// Should be positive but not perfect
	if tau <= 0 || tau >= 1.0 {
		t.Errorf("Expected moderate positive tau, got %f", tau)
	}
}

func TestComputeKendallTau_InsufficientData(t *testing.T) {
	pairs := []AnnotationPair{
		{"1", "pass", models.VerdictPass, 0.9},
	}

	_, err := ComputeKendallTau(pairs)
	if err == nil {
		t.Error("Expected error for single pair, got nil")
	}
}

func TestComputeKendallTau_InvalidHumanAnnotation(t *testing.T) {
	pairs := []AnnotationPair{
		{"1", "invalid", models.VerdictPass, 0.9},
		{"2", "pass", models.VerdictPass, 0.85},
	}

	_, err := ComputeKendallTau(pairs)
	if err == nil {
		t.Error("Expected error for invalid human annotation, got nil")
	}
}

func TestGenerateConfusionMatrix(t *testing.T) {
	pairs := []AnnotationPair{
		{"1", "pass", models.VerdictPass, 0.9},
		{"2", "pass", models.VerdictPass, 0.85},
		{"3", "pass", models.VerdictReview, 0.7},
		{"4", "review", models.VerdictReview, 0.65},
		{"5", "fail", models.VerdictFail, 0.3},
	}

	matrix := GenerateConfusionMatrix(pairs)

	// Check expected values
	expected := map[string]int{
		"pass_pass":     2,
		"pass_review":   1,
		"pass_fail":     0,
		"review_pass":   0,
		"review_review": 1,
		"review_fail":   0,
		"fail_pass":     0,
		"fail_review":   0,
		"fail_fail":     1,
	}

	for key, expectedCount := range expected {
		if matrix[key] != expectedCount {
			t.Errorf("matrix[%s] = %d, want %d", key, matrix[key], expectedCount)
		}
	}
}

func TestValidateAnnotations(t *testing.T) {
	pairs := []AnnotationPair{
		{"1", "pass", models.VerdictPass, 0.9},
		{"2", "pass", models.VerdictPass, 0.85},
		{"3", "review", models.VerdictReview, 0.65},
		{"4", "fail", models.VerdictFail, 0.3},
	}

	result, err := ValidateAnnotations(pairs, 0.3)
	if err != nil {
		t.Fatalf("ValidateAnnotations failed: %v", err)
	}

	// Check basic fields
	if result.TotalRecords != 4 {
		t.Errorf("TotalRecords = %d, want 4", result.TotalRecords)
	}

	if result.AgreementCount != 4 {
		t.Errorf("AgreementCount = %d, want 4", result.AgreementCount)
	}

	if result.AgreementRate != 1.0 {
		t.Errorf("AgreementRate = %f, want 1.0", result.AgreementRate)
	}

	// With ties, tau won't be exactly 1.0, but should be high
	if result.KendallTau < 0.7 {
		t.Errorf("KendallTau = %f, want >= 0.7", result.KendallTau)
	}

	if !result.Passed {
		t.Error("Validation should pass with high tau and threshold=0.3")
	}

	if result.Interpretation != "Strong agreement" && result.Interpretation != "Moderate to strong agreement" {
		t.Errorf("Interpretation = %q, want strong agreement", result.Interpretation)
	}
}

func TestValidateAnnotations_BelowThreshold(t *testing.T) {
	// Create pairs with low correlation
	pairs := []AnnotationPair{
		{"1", "pass", models.VerdictFail, 0.3},
		{"2", "fail", models.VerdictPass, 0.9},
		{"3", "pass", models.VerdictFail, 0.2},
		{"4", "fail", models.VerdictPass, 0.85},
	}

	result, err := ValidateAnnotations(pairs, 0.5)
	if err != nil {
		t.Fatalf("ValidateAnnotations failed: %v", err)
	}

	// Should fail validation
	if result.Passed {
		t.Error("Validation should fail with negative correlation and threshold=0.5")
	}

	// Agreement count should be 0 (all disagree)
	if result.AgreementCount != 0 {
		t.Errorf("AgreementCount = %d, want 0", result.AgreementCount)
	}
}

func TestInterpretTau(t *testing.T) {
	tests := []struct {
		tau      float64
		expected string
	}{
		{0.8, "Strong agreement"},
		{0.6, "Moderate to strong agreement"},
		{0.4, "Moderate agreement"},
		{0.2, "Weak agreement"},
		{0.05, "Very weak or no agreement"},
		{-0.8, "Strong agreement"},    // Absolute value
		{-0.4, "Moderate agreement"},  // Absolute value
	}

	for _, tt := range tests {
		result := InterpretTau(tt.tau)
		if result != tt.expected {
			t.Errorf("InterpretTau(%f) = %q, want %q", tt.tau, result, tt.expected)
		}
	}
}
