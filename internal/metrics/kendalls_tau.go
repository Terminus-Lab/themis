package metrics

import (
	"fmt"
	"math"
)

func ComputeKendallsTau(humanAnnotations, llmPredictions []Label) (float64, error) {
	if len(humanAnnotations) != len(llmPredictions) {
		return 0, fmt.Errorf("mismatch lengths")
	}

	if len(humanAnnotations) < 2 {
		return 0, fmt.Errorf("need at least 2 pairs to compute correlation")
	}

	// Convert labels to ranks (fail=0, review=1, pass=2)
	humanRanks := make([]int, len(humanAnnotations))
	llmRanks := make([]int, len(llmPredictions))

	for i := range humanAnnotations {
		humanRanks[i] = labelToRank(humanAnnotations[i])
		llmRanks[i] = labelToRank(llmPredictions[i])

		if humanRanks[i] == -1 {
			return 0, fmt.Errorf("invalid human annotation: %s", humanAnnotations[i])
		}
		if llmRanks[i] == -1 {
			return 0, fmt.Errorf("invalid LLM prediction: %s", llmPredictions[i])
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

// InterpretTau provides human-readable interpretation of Kendall's tau value.
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

// labelToRank converts label to numeric rank for correlation computation.
// pass=2, review=1, fail=0, invalid=-1
func labelToRank(label Label) int {
	switch label {
	case LabelPass:
		return 2
	case LabelReview:
		return 1
	case LabelFail:
		return 0
	default:
		return -1
	}
}
