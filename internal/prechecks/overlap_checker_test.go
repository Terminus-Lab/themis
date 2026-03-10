package prechecks

import (
	"math"
	"strings"
	"testing"

	"github.com/Terminus-Lab/themis/internal/models"
)

func TestOverlapChecker(t *testing.T) {

	result := NewOverlapChecker()

	tests := []struct {
		name   string
		query  string
		answer string
		score  float64
		reason string
	}{
		{
			name:   "Empty Query",
			query:  "",
			answer: "anything",
			score:  0.0,
			reason: "Empty Query",
		},
		{
			name:   "Empty Answer",
			query:  "anything",
			answer: "",
			score:  0.0,
			reason: "Empty Answer",
		},
		{
			name:   "No overlap",
			query:  "apple banana",
			answer: "orange grape",
			score:  0.0,
			reason: "Low keyword overlap",
		},
		{
			name:   "Full overlap",
			query:  "encryption security",
			answer: "encryption and security matter",
			score:  1.0,
			reason: "There is a good overlap",
		},
		{
			name:   "Partial overlap",
			query:  "foo bar baz",
			answer: "foo bar",
			score:  2.0 / 3.0,
			reason: "There is a good overlap",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := models.EvaluationContext{
				Query:  test.query,
				Answer: test.answer,
			}

			response := result.Check(ctx)

			if math.Abs(response.Score-test.score) > 1e-9 {
				t.Errorf("Score: %f, want: %f", response.Score, test.score)
			}

			if !strings.Contains(response.Reason, test.reason) {
				t.Errorf("Reason: %s, want: %s", response.Reason, test.reason)
			}
		})
	}
}
