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
