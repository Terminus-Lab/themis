package prechecks

import (
	"regexp"
	"strings"
	"time"

	"github.com/Terminus-Lab/themis/internal/models"
)

type FormatChecker struct {
}

func NewFormatChecker() *FormatChecker {
	return &FormatChecker{}
}

var repeatedPunctuation = regexp.MustCompile(`[!?.]{3,}`)

func (c *FormatChecker) Check(evaluationContext models.EvaluationContext) models.StageResult {

	result := models.StageResult{
		Name:     "format-checker",
		Score:    0.0,
		Reason:   "",
		Duration: 0,
	}

	now := time.Now()
	answer := strings.TrimSpace(evaluationContext.Answer)

	if len(answer) == 0 {
		result.Reason = "Empty answer"
		result.Duration = time.Since(now)
		return result
	}

	if len(strings.Fields(answer)) < 2 {
		result.Reason = "Short answer"
		result.Duration = time.Since(now)
		return result
	}

	if matched := repeatedPunctuation.MatchString(answer); matched {
		result.Reason = "Answer contains repeatable characters"
		result.Score = 0.5
		result.Duration = time.Since(now)
		return result
	}

	result.Reason = "Valid Answer"
	result.Duration = time.Since(now)
	result.Score = 1.0

	return result
}
