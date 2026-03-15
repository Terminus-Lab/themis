package metrics

import "fmt"

func Build(humanAnnotations, llmPredictions []Label) (*ConfusionMatrix, error) {
	if len(humanAnnotations) != len(llmPredictions) {
		return nil, fmt.Errorf("mismatch lengths")
	}

	cn := &ConfusionMatrix{
		Matrix: make(map[Label]map[Label]int),
		Labels: []Label{LabelFail, LabelReview, LabelPass},
	}

	for _, actualLabel := range cn.Labels {
		cn.Matrix[actualLabel] = make(map[Label]int)
		for _, predictedLabel := range cn.Labels {
			cn.Matrix[actualLabel][predictedLabel] = 0
		}
	}

	for i := range humanAnnotations {
		cn.Matrix[humanAnnotations[i]][llmPredictions[i]] += 1
	}

	return cn, nil
}

func (cm *ConfusionMatrix) Get(humanLabel, predictLabel Label) int {
	return cm.Matrix[humanLabel][predictLabel]
}

// TotalPredicted returns column sum
func (cm *ConfusionMatrix) TotalPredict(label Label) int {
	sum := 0
	for _, actual := range cm.Labels {
		sum += cm.Matrix[actual][label]
	}

	return sum
}

// TotalActual returns row sum
func (cm *ConfusionMatrix) TotalActual(label Label) int {
	sum := 0
	for _, actual := range cm.Labels {
		sum += cm.Matrix[label][actual]
	}
	return sum
}

func (cm *ConfusionMatrix) TotalCorrect() int {
	sum := 0
	for _, label := range cm.Labels {
		sum += cm.Matrix[label][label]
	}
	return sum
}

func (cm *ConfusionMatrix) TotalSample() int {
	sum := 0
	for _, label := range cm.Labels {
		sum += cm.TotalActual(label)
	}
	return sum
}

func (cm *ConfusionMatrix) ComputeClassMetrics() map[Label]ClassMetrics {
	metrics := make(map[Label]ClassMetrics)

	for _, label := range cm.Labels {
		// true positive: diagonal cells
		tp := cm.Get(label, label)

		// false positive: column sum - tp
		fp := cm.TotalPredict(label) - tp

		// false negative: row run - tp
		fn := cm.TotalActual(label) - tp

		// Precision
		var precision float64
		if tp+fp > 0 {
			precision = float64(tp) / float64(tp+fp)
		}

		// Recall
		var recall float64
		if tp+fn > 0 {
			recall = float64(tp) / float64(tp+fn)
		}

		// F1 score
		var f1 float64
		if precision+recall > 0 {
			f1 = 2 * (precision * recall) / (precision + recall)
		}

		metrics[label] = ClassMetrics{
			Precision: precision,
			Recall:    recall,
			F1Score:   f1,
			Support:   cm.TotalActual(label),
		}
	}

	return metrics
}
