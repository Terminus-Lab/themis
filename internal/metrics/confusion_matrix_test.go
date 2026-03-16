package metrics

import (
	"testing"
)

func TestBuild_Success(t *testing.T) {
	actual := []Label{LabelFail, LabelReview, LabelPass}
	predicted := []Label{LabelFail, LabelReview, LabelPass}

	cm, err := Build(actual, predicted)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cm == nil {
		t.Fatal("expected non-nil confusion matrix")
	}

	// Check matrix is initialized with all label combinations
	for _, actualLabel := range []Label{LabelFail, LabelReview, LabelPass} {
		for _, predictedLabel := range []Label{LabelFail, LabelReview, LabelPass} {
			if _, exists := cm.Matrix[actualLabel][predictedLabel]; !exists {
				t.Errorf("matrix missing entry for [%s][%s]", actualLabel, predictedLabel)
			}
		}
	}

	// Check diagonal values (perfect agreement)
	if cm.Get(LabelFail, LabelFail) != 1 {
		t.Errorf("expected fail→fail = 1, got %d", cm.Get(LabelFail, LabelFail))
	}
	if cm.Get(LabelReview, LabelReview) != 1 {
		t.Errorf("expected review→review = 1, got %d", cm.Get(LabelReview, LabelReview))
	}
	if cm.Get(LabelPass, LabelPass) != 1 {
		t.Errorf("expected pass→pass = 1, got %d", cm.Get(LabelPass, LabelPass))
	}
}

func TestBuild_MismatchLength(t *testing.T) {
	actual := []Label{LabelFail, LabelReview}
	predicted := []Label{LabelFail}

	cm, err := Build(actual, predicted)

	if err == nil {
		t.Fatal("expected error for mismatched lengths, got nil")
	}

	if cm != nil {
		t.Fatal("expected nil confusion matrix on error")
	}
}

func TestBuild_EmptyInput(t *testing.T) {
	actual := []Label{}
	predicted := []Label{}

	cm, err := Build(actual, predicted)

	if err != nil {
		t.Fatalf("expected no error for empty input, got %v", err)
	}

	if cm.TotalSample() != 0 {
		t.Errorf("expected total samples = 0, got %d", cm.TotalSample())
	}

	if cm.TotalCorrect() != 0 {
		t.Errorf("expected total correct = 0, got %d", cm.TotalCorrect())
	}
}

func TestGet(t *testing.T) {
	actual := []Label{LabelFail, LabelFail, LabelReview}
	predicted := []Label{LabelPass, LabelFail, LabelReview}

	cm, _ := Build(actual, predicted)

	tests := []struct {
		actualLabel    Label
		predictedLabel Label
		expected       int
	}{
		{LabelFail, LabelPass, 1},   // 1 fail predicted as pass
		{LabelFail, LabelFail, 1},   // 1 fail predicted correctly
		{LabelReview, LabelReview, 1}, // 1 review predicted correctly
		{LabelPass, LabelFail, 0},   // 0 pass predicted as fail
	}

	for _, tt := range tests {
		got := cm.Get(tt.actualLabel, tt.predictedLabel)
		if got != tt.expected {
			t.Errorf("Get(%s, %s) = %d, want %d",
				tt.actualLabel, tt.predictedLabel, got, tt.expected)
		}
	}
}

func TestTotalPredict(t *testing.T) {
	// Create scenario:
	// Actual:    [fail, fail, review, pass, pass]
	// Predicted: [fail, review, review, pass, fail]
	actual := []Label{LabelFail, LabelFail, LabelReview, LabelPass, LabelPass}
	predicted := []Label{LabelFail, LabelReview, LabelReview, LabelPass, LabelFail}

	cm, _ := Build(actual, predicted)

	tests := []struct {
		label    Label
		expected int
	}{
		{LabelFail, 2},   // 2 predictions of fail
		{LabelReview, 2}, // 2 predictions of review
		{LabelPass, 1},   // 1 prediction of pass
	}

	for _, tt := range tests {
		got := cm.TotalPredict(tt.label)
		if got != tt.expected {
			t.Errorf("TotalPredict(%s) = %d, want %d", tt.label, got, tt.expected)
		}
	}
}

func TestTotalActual(t *testing.T) {
	// Actual:    [fail, fail, review, pass, pass]
	// Predicted: [fail, review, review, pass, fail]
	actual := []Label{LabelFail, LabelFail, LabelReview, LabelPass, LabelPass}
	predicted := []Label{LabelFail, LabelReview, LabelReview, LabelPass, LabelFail}

	cm, _ := Build(actual, predicted)

	tests := []struct {
		label    Label
		expected int
	}{
		{LabelFail, 2},   // 2 actual failures
		{LabelReview, 1}, // 1 actual review
		{LabelPass, 2},   // 2 actual passes
	}

	for _, tt := range tests {
		got := cm.TotalActual(tt.label)
		if got != tt.expected {
			t.Errorf("TotalActual(%s) = %d, want %d", tt.label, got, tt.expected)
		}
	}
}

func TestTotalCorrect(t *testing.T) {
	// Perfect agreement: all diagonal
	actual := []Label{LabelFail, LabelReview, LabelPass}
	predicted := []Label{LabelFail, LabelReview, LabelPass}

	cm, _ := Build(actual, predicted)

	got := cm.TotalCorrect()
	expected := 3

	if got != expected {
		t.Errorf("TotalCorrect() = %d, want %d (perfect agreement)", got, expected)
	}

	// Mixed agreement
	actual = []Label{LabelFail, LabelFail, LabelReview, LabelPass, LabelPass}
	predicted = []Label{LabelFail, LabelReview, LabelReview, LabelPass, LabelFail}

	cm, _ = Build(actual, predicted)

	got = cm.TotalCorrect()
	expected = 3 // fail→fail (1), review→review (1), pass→pass (1)

	if got != expected {
		t.Errorf("TotalCorrect() = %d, want %d (mixed agreement)", got, expected)
	}
}

func TestTotalSample(t *testing.T) {
	actual := []Label{LabelFail, LabelReview, LabelPass, LabelPass, LabelFail}
	predicted := []Label{LabelPass, LabelReview, LabelPass, LabelFail, LabelFail}

	cm, _ := Build(actual, predicted)

	got := cm.TotalSample()
	expected := 5

	if got != expected {
		t.Errorf("TotalSample() = %d, want %d", got, expected)
	}

	// Empty case
	cm, _ = Build([]Label{}, []Label{})
	got = cm.TotalSample()
	expected = 0

	if got != expected {
		t.Errorf("TotalSample() for empty = %d, want %d", got, expected)
	}
}

func TestComputeClassMetrics_PerfectAgreement(t *testing.T) {
	// All predictions match actual
	actual := []Label{LabelFail, LabelFail, LabelReview, LabelReview, LabelPass, LabelPass, LabelPass}
	predicted := []Label{LabelFail, LabelFail, LabelReview, LabelReview, LabelPass, LabelPass, LabelPass}

	cm, _ := Build(actual, predicted)
	metrics := cm.ComputeClassMetrics()

	// All classes should have perfect precision, recall, F1
	for label, m := range metrics {
		if m.Precision != 1.0 {
			t.Errorf("%s: precision = %.3f, want 1.0", label, m.Precision)
		}
		if m.Recall != 1.0 {
			t.Errorf("%s: recall = %.3f, want 1.0", label, m.Recall)
		}
		if m.F1Score != 1.0 {
			t.Errorf("%s: F1 = %.3f, want 1.0", label, m.F1Score)
		}
	}

	// Check supports
	if metrics[LabelFail].Support != 2 {
		t.Errorf("fail support = %d, want 2", metrics[LabelFail].Support)
	}
	if metrics[LabelReview].Support != 2 {
		t.Errorf("review support = %d, want 2", metrics[LabelReview].Support)
	}
	if metrics[LabelPass].Support != 3 {
		t.Errorf("pass support = %d, want 3", metrics[LabelPass].Support)
	}
}

func TestComputeClassMetrics_AllWrong(t *testing.T) {
	// No diagonal entries - all predictions wrong
	actual := []Label{LabelFail, LabelReview, LabelPass}
	predicted := []Label{LabelPass, LabelFail, LabelReview}

	cm, _ := Build(actual, predicted)
	metrics := cm.ComputeClassMetrics()

	// All classes should have 0 precision, recall, F1
	for label, m := range metrics {
		if m.Precision != 0.0 {
			t.Errorf("%s: precision = %.3f, want 0.0", label, m.Precision)
		}
		if m.Recall != 0.0 {
			t.Errorf("%s: recall = %.3f, want 0.0", label, m.Recall)
		}
		if m.F1Score != 0.0 {
			t.Errorf("%s: F1 = %.3f, want 0.0", label, m.F1Score)
		}
	}
}

func TestComputeClassMetrics_Mixed(t *testing.T) {
	// Realistic scenario with calculated expected values
	// Matrix:
	//              Predicted
	//          fail  review  pass
	// Actual fail    2      1      0
	//        review  1      2      1
	//        pass    0      1      3
	actual := []Label{
		LabelFail, LabelFail, LabelFail,        // 3 fail
		LabelReview, LabelReview, LabelReview, LabelReview, // 4 review
		LabelPass, LabelPass, LabelPass, LabelPass, // 4 pass
	}
	predicted := []Label{
		LabelFail, LabelFail, LabelReview,      // 2 correct fail, 1 fail→review
		LabelFail, LabelReview, LabelReview, LabelPass, // 1 review→fail, 2 correct review, 1 review→pass
		LabelReview, LabelPass, LabelPass, LabelPass, // 1 pass→review, 3 correct pass
	}

	cm, _ := Build(actual, predicted)
	metrics := cm.ComputeClassMetrics()

	// Fail class:
	// TP=2, FP=1 (1 review→fail), FN=1 (1 fail→review)
	// Precision = 2/(2+1) = 0.667
	// Recall = 2/(2+1) = 0.667
	// F1 = 2*(0.667*0.667)/(0.667+0.667) = 0.667
	failMetrics := metrics[LabelFail]
	assertFloatEqual(t, "fail precision", failMetrics.Precision, 0.667, 0.01)
	assertFloatEqual(t, "fail recall", failMetrics.Recall, 0.667, 0.01)
	assertFloatEqual(t, "fail F1", failMetrics.F1Score, 0.667, 0.01)
	if failMetrics.Support != 3 {
		t.Errorf("fail support = %d, want 3", failMetrics.Support)
	}

	// Review class:
	// TP=2, FP=2 (1 fail→review, 1 pass→review), FN=2 (1 review→fail, 1 review→pass)
	// Precision = 2/(2+2) = 0.5
	// Recall = 2/(2+2) = 0.5
	// F1 = 2*(0.5*0.5)/(0.5+0.5) = 0.5
	reviewMetrics := metrics[LabelReview]
	assertFloatEqual(t, "review precision", reviewMetrics.Precision, 0.5, 0.01)
	assertFloatEqual(t, "review recall", reviewMetrics.Recall, 0.5, 0.01)
	assertFloatEqual(t, "review F1", reviewMetrics.F1Score, 0.5, 0.01)
	if reviewMetrics.Support != 4 {
		t.Errorf("review support = %d, want 4", reviewMetrics.Support)
	}

	// Pass class:
	// TP=3, FP=1 (1 review→pass), FN=1 (1 pass→review)
	// Precision = 3/(3+1) = 0.75
	// Recall = 3/(3+1) = 0.75
	// F1 = 2*(0.75*0.75)/(0.75+0.75) = 0.75
	passMetrics := metrics[LabelPass]
	assertFloatEqual(t, "pass precision", passMetrics.Precision, 0.75, 0.01)
	assertFloatEqual(t, "pass recall", passMetrics.Recall, 0.75, 0.01)
	assertFloatEqual(t, "pass F1", passMetrics.F1Score, 0.75, 0.01)
	if passMetrics.Support != 4 {
		t.Errorf("pass support = %d, want 4", passMetrics.Support)
	}
}

func TestComputeClassMetrics_ImbalancedClasses(t *testing.T) {
	// 90% pass, 5% fail, 5% review (realistic imbalance)
	actual := make([]Label, 100)
	predicted := make([]Label, 100)

	// 90 pass (all correct)
	for i := 0; i < 90; i++ {
		actual[i] = LabelPass
		predicted[i] = LabelPass
	}

	// 5 fail (4 correct, 1 wrong → review)
	for i := 90; i < 95; i++ {
		actual[i] = LabelFail
		if i == 94 {
			predicted[i] = LabelReview
		} else {
			predicted[i] = LabelFail
		}
	}

	// 5 review (3 correct, 2 wrong → pass)
	for i := 95; i < 100; i++ {
		actual[i] = LabelReview
		if i >= 98 {
			predicted[i] = LabelPass
		} else {
			predicted[i] = LabelReview
		}
	}

	cm, _ := Build(actual, predicted)
	metrics := cm.ComputeClassMetrics()

	// Verify fail class metrics
	// TP=4, FP=0, FN=1
	// Precision = 4/4 = 1.0, Recall = 4/5 = 0.8
	failMetrics := metrics[LabelFail]
	assertFloatEqual(t, "fail precision", failMetrics.Precision, 1.0, 0.01)
	assertFloatEqual(t, "fail recall", failMetrics.Recall, 0.8, 0.01)
	if failMetrics.Support != 5 {
		t.Errorf("fail support = %d, want 5", failMetrics.Support)
	}

	// Verify pass class metrics
	// TP=90, FP=2 (from review), FN=0
	// Precision = 90/92 = 0.978, Recall = 90/90 = 1.0
	passMetrics := metrics[LabelPass]
	assertFloatEqual(t, "pass precision", passMetrics.Precision, 0.978, 0.01)
	assertFloatEqual(t, "pass recall", passMetrics.Recall, 1.0, 0.01)
	if passMetrics.Support != 90 {
		t.Errorf("pass support = %d, want 90", passMetrics.Support)
	}
}

func TestComputeClassMetrics_ZeroDivision(t *testing.T) {
	// Edge case: class exists in actual but never predicted
	actual := []Label{LabelFail, LabelReview, LabelPass}
	predicted := []Label{LabelPass, LabelPass, LabelPass}

	cm, _ := Build(actual, predicted)
	metrics := cm.ComputeClassMetrics()

	// Fail and Review should have 0 precision (TP=0, FP=0 → 0/0 handled)
	// but non-zero support
	if metrics[LabelFail].Precision != 0.0 {
		t.Errorf("fail precision with zero predictions should be 0.0, got %.3f",
			metrics[LabelFail].Precision)
	}

	if metrics[LabelReview].Precision != 0.0 {
		t.Errorf("review precision with zero predictions should be 0.0, got %.3f",
			metrics[LabelReview].Precision)
	}

	// Pass should have FP > 0, so precision calculation works
	if metrics[LabelPass].Precision == 0.0 {
		t.Error("pass precision should be non-zero")
	}
}

// Helper function for float comparison with tolerance
func assertFloatEqual(t *testing.T, name string, got, want, tolerance float64) {
	t.Helper()
	diff := got - want
	if diff < 0 {
		diff = -diff
	}
	if diff > tolerance {
		t.Errorf("%s = %.3f, want %.3f (tolerance %.3f)", name, got, want, tolerance)
	}
}
