package storage

import "math"

// PopulationStdDev computes the population standard deviation of a slice of
// judge scores. It measures how much individual judges disagree with each
// other for a single evaluation.
//
// Uses population std-dev (divides by N, not N-1) because the scores represent
// the full set of judges — not a sample.
//
// Returns 0 if fewer than 2 scores are provided (no disagreement possible).
// Range: [0, 0.5] for scores in [0, 1].
func PopulationStdDev(scores []float64) float64 {
	if len(scores) < 2 {
		return 0
	}
	mean := 0.0
	for _, v := range scores {
		mean += v
	}
	mean /= float64(len(scores))

	variance := 0.0
	for _, v := range scores {
		variance += (v - mean) * (v - mean)
	}
	return math.Sqrt(variance / float64(len(scores)))
}
