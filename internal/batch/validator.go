package batch

import (
	"fmt"
	"math"

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

	// Convert verdicts to ranks
	humanRanks := make([]int, len(pairs))
	llmRanks := make([]int, len(pairs))

	for i, pair := range pairs {
		humanRanks[i] = verdictToRank(pair.HumanAnnotation)
		llmRanks[i] = verdictToRank(string(pair.LLMVerdict))

		if humanRanks[i] == -1 {
			return 0, fmt.Errorf("invalid human annotation: %s", pair.HumanAnnotation)
		}
		if llmRanks[i] == -1 {
			return 0, fmt.Errorf("invalid LLM verdict: %s", pair.LLMVerdict)
		}
	}

	// Count concordant and discordant pairs
	concordant := 0
	discordant := 0

	for i := 0; i < len(humanRanks); i++ {
		for j := i + 1; j < len(humanRanks); j++ {
			humanDiff := humanRanks[i] - humanRanks[j]
			llmDiff := llmRanks[i] - llmRanks[j]

			if humanDiff*llmDiff > 0 {
				concordant++ // Same direction
			} else if humanDiff*llmDiff < 0 {
				discordant++ // Opposite direction
			}
			// If either diff is 0, it's a tie - don't count
		}
	}

	// Compute Kendall's tau
	totalPairs := len(humanRanks) * (len(humanRanks) - 1) / 2
	if totalPairs == 0 {
		return 0, fmt.Errorf("not enough pairs to compute correlation")
	}

	tau := float64(concordant-discordant) / float64(totalPairs)
	return tau, nil
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

// verdictToRank converts verdict string to numeric rank
// pass=2, review=1, fail=0
func verdictToRank(verdict string) int {
	switch verdict {
	case "pass":
		return 2
	case "review":
		return 1
	case "fail":
		return 0
	default:
		return -1 // Invalid
	}
}

// InterpretTau provides human-readable interpretation of Kendall's tau
func InterpretTau(tau float64) string {
	absTau := math.Abs(tau)

	switch {
	case absTau >= 0.7:
		return "Strong agreement"
	case absTau >= 0.5:
		return "Moderate to strong agreement"
	case absTau >= 0.3:
		return "Moderate agreement"
	case absTau >= 0.1:
		return "Weak agreement"
	default:
		return "Very weak or no agreement"
	}
}
