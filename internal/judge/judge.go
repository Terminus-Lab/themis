package judge

import (
	"context"

	"github.com/Terminus-Lab/themis/internal/models"
)

type Judge interface {
	Name() string
	Evaluate(ctx context.Context, evaluationContext models.EvaluationContext) models.StageResult
	RequiresExpectedOutput() bool
}
