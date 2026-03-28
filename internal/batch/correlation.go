package batch

import (
	"math"

	"github.com/Terminus-Lab/themis/internal/models"
)

// Annotation holds human-provided ground truth for a single conversation.
type Annotation struct {
	HumanLabel  string   // "pass", "review", "fail" — empty = not provided
	HumanScore  *float64 // 0.0–1.0 — nil = not provided
	HumanReason string   // free-text explanation from annotator — empty = not provided
}

// ConfusionMatrixResult is a 3-class confusion matrix for fail/review/pass.
type ConfusionMatrixResult struct {
	Labels []string `json:"labels"`
	Matrix [][]int  `json:"matrix"` // [true][predicted]
}

// DisagreementEntry records a single per-conversation label mismatch between
// the human annotator and Themis.
type DisagreementEntry struct {
	ConversationID string   `json:"conversation_id"`
	HumanLabel     string   `json:"human_label"`
	ThemisLabel    string   `json:"themis_label"`
	HumanScore     *float64 `json:"human_score,omitempty"`
	ThemisScore    float64  `json:"themis_score"`
}

// CorrelationReport holds all computed correlation metrics.
type CorrelationReport struct {
	AnnotatedCount  int                    `json:"annotated_count"`
	KendallTau      *float64               `json:"kendall_tau,omitempty"`
	WeightedKappa   *float64               `json:"weighted_kappa,omitempty"`
	ConfusionMatrix *ConfusionMatrixResult `json:"confusion_matrix,omitempty"`
	Disagreements   []DisagreementEntry    `json:"disagreements,omitempty"`
}

// ComputeCorrelationReport joins results with annotations by ConversationID and computes
// correlation metrics between Themis scores/verdicts and human-provided ground truth.
func ComputeCorrelationReport(results []models.ConversationEvaluationResult, annotations map[string]Annotation) CorrelationReport {
	labels := []string{"fail", "review", "pass"}

	var (
		themisScores  []float64
		humanScores   []float64
		themisLabels  []string
		humanLabels   []string
		disagreements []DisagreementEntry
		annotated     int
	)

	for _, r := range results {
		ann, ok := annotations[r.ConversationID]
		if !ok {
			continue
		}
		annotated++

		if ann.HumanScore != nil {
			themisScores = append(themisScores, r.FinalScore)
			humanScores = append(humanScores, *ann.HumanScore)
		}

		if ann.HumanLabel != "" {
			themisLabels = append(themisLabels, string(r.Verdict))
			humanLabels = append(humanLabels, ann.HumanLabel)
			if ann.HumanLabel != string(r.Verdict) {
				disagreements = append(disagreements, DisagreementEntry{
					ConversationID: r.ConversationID,
					HumanLabel:     ann.HumanLabel,
					ThemisLabel:    string(r.Verdict),
					ThemisScore:    r.FinalScore,
					HumanScore:     ann.HumanScore,
				})
			}
		}
	}

	report := CorrelationReport{
		AnnotatedCount: annotated,
		Disagreements:  disagreements,
	}

	if len(themisScores) >= 2 {
		tau := kendallTauB(themisScores, humanScores)
		report.KendallTau = &tau
	}

	if len(themisLabels) > 0 {
		wkappa := weightedCohensKappa(themisLabels, humanLabels, labels)
		report.WeightedKappa = &wkappa

		report.ConfusionMatrix = buildConfusionMatrix(humanLabels, themisLabels, labels)
	}

	return report
}

// kendallTauB computes Kendall's τ-b with tie correction.
func kendallTauB(x, y []float64) float64 {
	n := len(x)
	concordant, discordant, tiesX, tiesY := 0, 0, 0, 0
	for i := 0; i < n-1; i++ {
		for j := i + 1; j < n; j++ {
			dx := x[i] - x[j]
			dy := y[i] - y[j]
			switch {
			case dx*dy > 0:
				concordant++
			case dx*dy < 0:
				discordant++
			case dx == 0 && dy != 0:
				tiesX++
			case dy == 0 && dx != 0:
				tiesY++
			}
		}
	}
	n0 := n * (n - 1) / 2
	denom := math.Sqrt(float64(n0-tiesX) * float64(n0-tiesY))
	if denom == 0 {
		return 0
	}
	return float64(concordant-discordant) / denom
}

// weightedCohensKappa computes linear-weighted Cohen's κ.
// weights[i][j] = 1 - |i-j|/(k-1), penalizing fail↔pass more than fail↔review.
func weightedCohensKappa(predicted, actual []string, labels []string) float64 {
	n := len(predicted)
	if n == 0 {
		return 0
	}
	labelIdx := make(map[string]int, len(labels))
	for i, l := range labels {
		labelIdx[l] = i
	}
	k := len(labels)

	// Build weight matrix: w[i][j] = 1 - |i-j|/(k-1)
	w := make([][]float64, k)
	for i := range w {
		w[i] = make([]float64, k)
		for j := range w[i] {
			diff := i - j
			if diff < 0 {
				diff = -diff
			}
			w[i][j] = 1.0 - float64(diff)/float64(k-1)
		}
	}

	cm := make([][]int, k)
	for i := range cm {
		cm[i] = make([]int, k)
	}
	valid := 0
	for i := range predicted {
		pi, ok1 := labelIdx[predicted[i]]
		ai, ok2 := labelIdx[actual[i]]
		if ok1 && ok2 {
			cm[ai][pi]++
			valid++
		}
	}
	if valid == 0 {
		return 0
	}

	// Observed weighted agreement
	po := 0.0
	for i := 0; i < k; i++ {
		for j := 0; j < k; j++ {
			po += w[i][j] * float64(cm[i][j])
		}
	}
	po /= float64(valid)

	// Expected weighted agreement
	pe := 0.0
	for i := 0; i < k; i++ {
		for j := 0; j < k; j++ {
			rowSum := 0
			colSum := 0
			for l := 0; l < k; l++ {
				rowSum += cm[i][l]
				colSum += cm[l][j]
			}
			pe += w[i][j] * float64(rowSum) * float64(colSum)
		}
	}
	pe /= float64(valid * valid)

	if 1-pe == 0 {
		return 1
	}
	return (po - pe) / (1 - pe)
}

// buildConfusionMatrix builds a ConfusionMatrixResult from actual (human) and predicted (themis) labels.
func buildConfusionMatrix(actual, predicted []string, labels []string) *ConfusionMatrixResult {
	k := len(labels)
	labelIdx := make(map[string]int, k)
	for i, l := range labels {
		labelIdx[l] = i
	}

	matrix := make([][]int, k)
	for i := range matrix {
		matrix[i] = make([]int, k)
	}

	for i := range actual {
		ai, ok1 := labelIdx[actual[i]]
		pi, ok2 := labelIdx[predicted[i]]
		if ok1 && ok2 {
			matrix[ai][pi]++
		}
	}

	return &ConfusionMatrixResult{
		Labels: labels,
		Matrix: matrix,
	}
}
