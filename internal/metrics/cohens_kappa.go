package metrics

import "fmt"


func ComputeCohensKappa(cm *ConfusionMatrix) (float64, error) {
	total := cm.TotalSample()
	if total == 0 {
		return 0, fmt.Errorf("empty confusion matrix")
	}

	// Observed agreement: proportion of diagonal (actual matches)
	observedAgreement := float64(cm.TotalCorrect()) / float64(total)

	// Expected agreement: sum of (marginal_actual × marginal_predicted) for each label
	expectedAgreement := 0.0
	for _, label := range cm.Labels {
		pActual := float64(cm.TotalActual(label)) / float64(total)
		pPredicted := float64(cm.TotalPredict(label)) / float64(total)
		expectedAgreement += pActual * pPredicted
	}

	// Handle edge case: all samples fall into one class (expected agreement = 1.0).
	// When this happens, observed agreement is also 1.0 (perfect agreement by
	// definition), so kappa = 1.0 by convention rather than 0/0.
	if expectedAgreement >= 1.0 {
		return 1.0, nil
	}

	kappa := (observedAgreement - expectedAgreement) / (1.0 - expectedAgreement)
	return kappa, nil
}

// InterpretKappa returns human-readable interpretation of Cohen's Kappa value.
func InterpretKappa(kappa float64) string {
	switch {
	case kappa < 0.00:
		return "Poor"
	case kappa < 0.21:
		return "Slight"
	case kappa < 0.41:
		return "Fair"
	case kappa < 0.61:
		return "Moderate"
	case kappa < 0.81:
		return "Substantial"
	default:
		return "Almost perfect"
	}
}
