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
func ValidateAnnotations(pairs []AnnotationPair, threshold float64) (*metrics.ValidationResult, error) {
	if len(pairs) == 0 {
		return nil, fmt.Errorf("no annotation pairs to validate")
	}

	// Convert AnnotationPairs to Label slices
	humanLabels := make([]metrics.Label, len(pairs))
	llmLabels := make([]metrics.Label, len(pairs))

	for i, pair := range pairs {
		humanLabels[i] = metrics.Label(pair.HumanAnnotation)
		llmLabels[i] = metrics.Label(string(pair.LLMVerdict))
	}

	// Build confusion matrix
	cm, err := metrics.Build(humanLabels, llmLabels)
	if err != nil {
		return nil, fmt.Errorf("failed to build confusion matrix: %w", err)
	}

	// Compute Kendall's tau
	tau, err := metrics.ComputeKendallsTau(humanLabels, llmLabels)
	if err != nil {
		return nil, fmt.Errorf("failed to compute Kendall's tau: %w", err)
	}

	// Compute Cohen's Kappa
	kappa, err := metrics.ComputeCohensKappa(cm)
	if err != nil {
		return nil, fmt.Errorf("failed to compute Cohen's Kappa: %w", err)
	}

	// Compute per-class metrics
	perClassMetrics := cm.ComputeClassMetrics()

	// Determine if validation passed (based on Kendall's tau threshold)
	passed := tau >= threshold

	// Build result
	result := &metrics.ValidationResult{
		Passed:       passed,
		TotalRecords: len(pairs),
		Threshold:    threshold,
		CorrelationMetrics: metrics.CorrelationMetrics{
			KendallsTau:     tau,
			Interpretation:  metrics.InterpretTau(tau),
			PassedThreshold: passed,
		},
		AgreementMetrics: metrics.AgreementMetrics{
			CohensKappa:    kappa,
			Interpretation: metrics.InterpretKappa(kappa),
		},
		ConfusionMatrix: cm.Matrix,
		PerClassMetrics: perClassMetrics,
	}

	return result, nil
}


// InterpretTau provides human-readable interpretation of Kendall's tau
func InterpretTau(tau float64) string {
	return metrics.InterpretTau(tau)
}
