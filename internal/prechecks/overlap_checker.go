package prechecks

import (
	"fmt"
	"strings"
	"time"

	"github.com/Terminus-Lab/themis/internal/models"
)

type OverlapChecker struct {
	MinOverlapThreshold float64
}

func NewOverlapChecker() *OverlapChecker {
	return &OverlapChecker{}
}

// OverlapChecker scores an answer based on keyword overlap with the query.
// It tokenizes both strings, computes the ratio of shared unique words,
// and returns a low score if the answer doesn't share enough terms with the query.
func (c *OverlapChecker) Check(evaluationContext models.EvaluationContext) models.StageResult {

	if c.MinOverlapThreshold == 0.0 {
		// set default value
		c.MinOverlapThreshold = 0.1
	}

	result := models.StageResult{
		Name:     "overlap-checker",
		Score:    0.0,
		Reason:   "",
		Duration: 0,
	}
	now := time.Now()

	if len(evaluationContext.Query) == 0 {
		result.Reason = "Empty Query"
		result.Duration = time.Since(now)
		return result
	}

	if len(evaluationContext.Answer) == 0 {
		result.Reason = "Empty Answer"
		result.Duration = time.Since(now)
		return result
	}

	queryTokens := c.stringTokenizer(evaluationContext.Query)
	answerTokens := c.stringTokenizer(evaluationContext.Answer)

	uniqueQueryTokens := extractUniqueTokens(queryTokens)
	uniqueAnswerTokens := extractUniqueTokens(answerTokens)

	count := 0
	for token := range uniqueQueryTokens {
		if _, exists := uniqueAnswerTokens[token]; exists {
			count++
		}
	}

	score := float64(count) / float64(len(uniqueQueryTokens))
	if score < c.MinOverlapThreshold {
		result.Reason = fmt.Sprintf("Low keyword overlap: %.0f%% of query terms found in answer", score*100)
		result.Score = score
	} else {
		result.Reason = "There is a good overlap"
		result.Score = score
	}

	result.Duration = time.Since(now)
	return result

}

func extractUniqueTokens(tokens []string) map[string]bool {
	unique := make(map[string]bool, len(tokens))
	for _, t := range tokens {
		unique[t] = true
	}
	return unique
}

var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "is": true, "are": true,
	"was": true, "were": true, "be": true, "been": true, "being": true,
	"have": true, "has": true, "had": true, "do": true, "does": true,
	"did": true, "will": true, "would": true, "could": true, "should": true,
	"of": true, "at": true, "by": true, "for": true, "with": true,
	"about": true, "against": true, "between": true, "into": true,
	"through": true, "during": true, "before": true, "after": true,
	"to": true, "from": true, "in": true, "on": true,
}

func (c *OverlapChecker) stringTokenizer(s string) []string {
	s = strings.ToLower(s)
	s = removePunctuation(s)

	tokens := []string{}
	for word := range strings.FieldsSeq(s) {
		if !stopWords[word] && len(word) > 1 {
			tokens = append(tokens, word)
		}
	}
	return tokens

}

func removePunctuation(s string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(".,!?;:()[]{}\"'", r) {
			return -1 // Remove this rune
		}
		return r
	}, s)
}
