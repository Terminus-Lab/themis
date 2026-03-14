package metrics

type Label string

const (
	LabelFail    Label = "fail"
	LabelReview  Label = "review"
	LabelSuccess Label = "pass"
)

type ConfusionMatrix struct {
	Matrix map[Label]map[Label]int
	Label  []Label
}

