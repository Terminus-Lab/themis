package prechecks

import (
	"fmt"
	"time"

	"github.com/Terminus-Lab/themis/internal/models"
)

type LengthChecker struct {
}

func NewLengthChecker() *LengthChecker {
	return &LengthChecker{}
}

// LengthChecker scores an answer based on its length relative to the query.
// It computes the character ratio between answer and query, penalizing answers
// that are too short (score 0.0) or excessively long (score 0.5).
func (c *LengthChecker) Check(evaluationContext models.EvaluationContext) models.StageResult {
	const minRatio = 0.5
	const maxRatio = 10.0

	answerLength := len(evaluationContext.Answer)
	queryLength := len(evaluationContext.Query)

	result := models.StageResult{
		Name:     "length-checker",
		Score:    0.0,
		Reason:   "",
		Duration: 0,
	}

	if queryLength == 0 {
		result.Reason = "Empty query"
		return result
	}

	now := time.Now()
	ratio := float64(answerLength) / float64(queryLength)

	if ratio < minRatio {
		result.Reason = "The answer contains fewer characters then the user answer"
	} else if ratio > maxRatio {
		result.Score = 0.5
		result.Reason = fmt.Sprintf("The answer is too long. It's %1.f times longer than the query", ratio)
	} else {
		result.Score = 1.0
		result.Reason = "Answer Length is acceptable"
	}
	result.Duration = time.Since(now)
	return result
}
