package metrics

import (
	"testing"
)

func TestComputeKendallsTau_PerfectAgreement(t *testing.T) {
	// Use unique ranks to get exactly τ = 1.0
	human := []Label{LabelFail, LabelReview, LabelPass}
	llm := []Label{LabelFail, LabelReview, LabelPass}

	tau, err := ComputeKendallsTau(human, llm)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if tau != 1.0 {
		t.Errorf("perfect agreement should give τ = 1.0, got %.3f", tau)
	}
}

func TestComputeKendallsTau_StrongCorrelation(t *testing.T) {
	human := []Label{LabelFail, LabelFail, LabelReview, LabelReview, LabelPass, LabelPass}
	llm := []Label{LabelFail, LabelReview, LabelReview, LabelPass, LabelPass, LabelPass}

	tau, err := ComputeKendallsTau(human, llm)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should have strong positive correlation (most pairs agree)
	if tau < 0.5 || tau > 1.0 {
		t.Errorf("expected strong positive correlation, got τ = %.3f", tau)
	}
}

func TestComputeKendallsTau_NegativeCorrelation(t *testing.T) {
	human := []Label{LabelFail, LabelReview, LabelPass}
	llm := []Label{LabelPass, LabelReview, LabelFail}

	tau, err := ComputeKendallsTau(human, llm)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Perfect inverse correlation
	if tau != -1.0 {
		t.Errorf("perfect disagreement should give τ = -1.0, got %.3f", tau)
	}
}

func TestComputeKendallsTau_MismatchLength(t *testing.T) {
	human := []Label{LabelFail, LabelReview}
	llm := []Label{LabelFail}

	_, err := ComputeKendallsTau(human, llm)

	if err == nil {
		t.Fatal("expected error for mismatched lengths, got nil")
	}
}

func TestComputeKendallsTau_TooFewPairs(t *testing.T) {
	human := []Label{LabelFail}
	llm := []Label{LabelFail}

	_, err := ComputeKendallsTau(human, llm)

	if err == nil {
		t.Fatal("expected error for too few pairs, got nil")
	}
}

func TestInterpretTau(t *testing.T) {
	tests := []struct {
		tau      float64
		expected string
	}{
		{0.85, "Strong agreement"},
		{0.65, "Moderate to strong agreement"},
		{0.45, "Moderate agreement"},
		{0.25, "Weak agreement"},
		{0.05, "Very weak or no agreement"},
		{-0.75, "Strong agreement"}, // Negative but strong
	}

	for _, tt := range tests {
		got := InterpretTau(tt.tau)
		if got != tt.expected {
			t.Errorf("InterpretTau(%.2f) = %s, want %s", tt.tau, got, tt.expected)
		}
	}
}
