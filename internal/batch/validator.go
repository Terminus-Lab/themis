package batch

import (
	"fmt"

	"github.com/Terminus-Lab/themis/internal/metrics"
	"github.com/Terminus-Lab/themis/internal/models"
)

// AnnotationPair represents a human annotation paired with LLM verdict
type AnnotationPair struct {
	EventID         string
	HumanAnnotation string
	LLMVerdict      models.Verdict
	Confidence      float64
}

// ValidationResult holds the outcome of correlation analysis
type ValidationResult struct {
	TotalRecords    int                `json:"total_records"`
	AgreementCount  int                `json:"agreement_count"`
	AgreementRate   float64            `json:"agreement_rate"`
	KendallTau      float64            `json:"kendall_tau"`
	Threshold       float64            `json:"threshold"`
	Passed          bool               `json:"passed"`
	ConfusionMatrix map[string]int     `json:"confusion_matrix"`
	Interpretation  string             `json:"interpretation"`
}

// ComputeKendallTau calculates Kendall's tau-b correlation coefficient
// between human annotations and LLM verdicts
func ComputeKendallTau(pairs []AnnotationPair) (float64, error) {
	if len(pairs) < 2 {
		return 0, fmt.Errorf("need at least 2 pairs to compute correlation")
	}

	// Convert AnnotationPairs to Label slices
	humanLabels := make([]metrics.Label, len(pairs))
	llmLabels := make([]metrics.Label, len(pairs))

	for i, pair := range pairs {
		humanLabels[i] = metrics.Label(pair.HumanAnnotation)
		llmLabels[i] = metrics.Label(string(pair.LLMVerdict))
	}

	// Use metrics package implementation
	return metrics.ComputeKendallsTau(humanLabels, llmLabels)
}

// GenerateConfusionMatrix creates a confusion matrix from annotation pairs
func GenerateConfusionMatrix(pairs []AnnotationPair) map[string]int {
	matrix := make(map[string]int)

	// Initialize all combinations
	verdicts := []string{"pass", "review", "fail"}
	for _, human := range verdicts {
		for _, llm := range verdicts {
			key := fmt.Sprintf("%s_%s", human, llm)
			matrix[key] = 0
		}
	}

	// Count occurrences
	for _, pair := range pairs {
		key := fmt.Sprintf("%s_%s", pair.HumanAnnotation, pair.LLMVerdict)
		matrix[key]++
	}

	return matrix
}

// ValidateAnnotations performs full validation analysis
func ValidateAnnotations(pairs []AnnotationPair, threshold float64) (*ValidationResult, error) {
	if len(pairs) == 0 {
		return nil, fmt.Errorf("no annotation pairs to validate")
	}

	// Compute Kendall's tau
	tau, err := ComputeKendallTau(pairs)
	if err != nil {
		return nil, fmt.Errorf("failed to compute Kendall's tau: %w", err)
	}

	// Count agreements
	agreementCount := 0
	for _, pair := range pairs {
		if pair.HumanAnnotation == string(pair.LLMVerdict) {
			agreementCount++
		}
	}

	// Generate confusion matrix
	confusionMatrix := GenerateConfusionMatrix(pairs)

	// Determine if validation passed
	passed := tau >= threshold

	// Interpretation
	interpretation := InterpretTau(tau)

	result := &ValidationResult{
		TotalRecords:    len(pairs),
		AgreementCount:  agreementCount,
		AgreementRate:   float64(agreementCount) / float64(len(pairs)),
		KendallTau:      tau,
		Threshold:       threshold,
		Passed:          passed,
		ConfusionMatrix: confusionMatrix,
		Interpretation:  interpretation,
	}

	return result, nil
}


// InterpretTau provides human-readable interpretation of Kendall's tau
func InterpretTau(tau float64) string {
	return metrics.InterpretTau(tau)
}
