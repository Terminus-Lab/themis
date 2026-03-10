package prechecks

import (
	"github.com/Terminus-Lab/themis/internal/models"
)

type Checker interface {
	Check(evaluationContext models.EvaluationContext) models.StageResult
}
