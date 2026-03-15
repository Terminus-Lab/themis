package metrics

type Label string

const (
	LabelFail   Label = "fail"
	LabelReview Label = "review"
	LabelPass   Label = "pass"
)

type ConfusionMatrix struct {
	Matrix map[Label]map[Label]int // actual (human) -> predict (llm) -> count
	Labels []Label
}

type ClassMetrics struct {
	Precision float64 `json:"precision"`
	Recall    float64 `json:"recall"`
	F1Score   float64 `json:"f1"`
	Support   int     `json:"support"` // number of actual instances
}

type ValidationResult struct {
	// Top-level fields
	Passed       bool    `json:"passed"`
	TotalRecords int     `json:"total_records"`
	Threshold    float64 `json:"threshold"`

	// Correlation metrics (PRIMARY - Pass/Fail decision)
	CorrelationMetrics CorrelationMetrics `json:"correlation_metrics"`

	// Agreement metrics (REPORT - Industry standard)
	AgreementMetrics AgreementMetrics `json:"agreement_metrics"`

	// Confusion matrix (DEBUG - Actionable insights)
	ConfusionMatrix map[Label]map[Label]int `json:"confusion_matrix"`

	// Per-class metrics (DEBUG - Precision/Recall/F1)
	PerClassMetrics map[Label]ClassMetrics `json:"per_class_metrics"`
}

type CorrelationMetrics struct {
	KendallsTau      float64 `json:"kendalls_tau"`
	Interpretation   string  `json:"interpretation"`
	PassedThreshold  bool    `json:"passed_threshold"`
}

type AgreementMetrics struct {
	CohensKappa    float64 `json:"cohens_kappa"`
	Interpretation string  `json:"interpretation"`
}