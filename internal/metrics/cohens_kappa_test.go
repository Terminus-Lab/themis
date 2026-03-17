package metrics

import (
	"testing"
)

func TestComputeCohensKappa_PerfectAgreement(t *testing.T) {
	// All predictions match actual
	actual := []Label{LabelFail, LabelReview, LabelPass, LabelPass, LabelFail}
	predicted := []Label{LabelFail, LabelReview, LabelPass, LabelPass, LabelFail}

	cm, _ := Build(actual, predicted)
	kappa, err := ComputeCohensKappa(cm)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if kappa != 1.0 {
		t.Errorf("perfect agreement should give κ = 1.0, got %.3f", kappa)
	}
}

func TestComputeCohensKappa_ModerateAgreement(t *testing.T) {
	// Realistic scenario: 75% accuracy with some confusion
	actual := []Label{
		LabelFail, LabelFail, LabelFail,
		LabelReview, LabelReview, LabelReview, LabelReview,
		LabelPass, LabelPass, LabelPass, LabelPass,
	}
	predicted := []Label{
		LabelFail, LabelFail, LabelReview,      // 2 correct fail, 1 fail→review
		LabelFail, LabelReview, LabelReview, LabelPass, // 1 review→fail, 2 correct review, 1 review→pass
		LabelReview, LabelPass, LabelPass, LabelPass, // 1 pass→review, 3 correct pass
	}

	cm, _ := Build(actual, predicted)
	kappa, err := ComputeCohensKappa(cm)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Expected κ ≈ 0.4-0.6 (moderate agreement)
	if kappa < 0.3 || kappa > 0.7 {
		t.Errorf("expected moderate agreement (0.3-0.7), got κ = %.3f", kappa)
	}
}

func TestComputeCohensKappa_ImbalancedData(t *testing.T) {
	// 90% pass, judge always predicts pass
	actual := make([]Label, 100)
	predicted := make([]Label, 100)

	for i := 0; i < 90; i++ {
		actual[i] = LabelPass
		predicted[i] = LabelPass
	}
	for i := 90; i < 100; i++ {
		actual[i] = LabelFail
		predicted[i] = LabelPass // Judge misses all failures
	}

	cm, _ := Build(actual, predicted)
	kappa, err := ComputeCohensKappa(cm)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// High accuracy (90%) but κ should be low (judge just exploiting class imbalance)
	if kappa > 0.2 {
		t.Errorf("expected low κ for biased judge, got κ = %.3f", kappa)
	}
}

func TestComputeCohensKappa_SingleClass(t *testing.T) {
	// All samples are the same class — happens when validating a homogeneous sample
	// (e.g. all "pass" evaluations with all "pass" annotations).
	// Expected: kappa = 1.0 (perfect agreement by convention, not 0/0 error).
	actual := []Label{LabelPass, LabelPass, LabelPass, LabelPass}
	predicted := []Label{LabelPass, LabelPass, LabelPass, LabelPass}

	cm, _ := Build(actual, predicted)
	kappa, err := ComputeCohensKappa(cm)

	if err != nil {
		t.Fatalf("expected no error for single-class input, got %v", err)
	}
	if kappa != 1.0 {
		t.Errorf("single-class perfect agreement should give κ = 1.0, got %.3f", kappa)
	}
}

func TestComputeCohensKappa_EmptyMatrix(t *testing.T) {
	cm, _ := Build([]Label{}, []Label{})
	_, err := ComputeCohensKappa(cm)

	if err == nil {
		t.Fatal("expected error for empty matrix, got nil")
	}
}

func TestInterpretKappa(t *testing.T) {
	tests := []struct {
		kappa          float64
		expectedInterp string
	}{
		{-0.1, "Poor"},
		{0.15, "Slight"},
		{0.35, "Fair"},
		{0.55, "Moderate"},
		{0.75, "Substantial"},
		{0.95, "Almost perfect"},
	}

	for _, tt := range tests {
		got := InterpretKappa(tt.kappa)
		if got != tt.expectedInterp {
			t.Errorf("InterpretKappa(%.2f) = %s, want %s", tt.kappa, got, tt.expectedInterp)
		}
	}
}
