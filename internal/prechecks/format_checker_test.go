package prechecks

import (
	"testing"

	"github.com/Terminus-Lab/themis/internal/models"
)

func TestFormatChecker(t *testing.T) {
	result := NewFormatChecker()

	tests := []struct {
		name   string
		answer string
		score  float64
		reason string
	}{
		{
			name:   "Empty answer",
			answer: "",
			score:  0.0,
			reason: "Empty answer",
		},
		{
			name:   "Single word",
			answer: "ok",
			score:  0.0,
			reason: "Short answer",
		},
		{
			name:   "Repeated punctuation",
			answer: "Hello!!! How are you?",
			score:  0.5,
			reason: "Answer contains repeatable characters",
		},
		{
			name:   "Valid Answer",
			answer: "This is a valid answer",
			score:  1.0,
			reason: "Valid Answer",
		},
		{
			name:   "Exact two words check",
			answer: "Hi there",
			score:  1.0,
			reason: "Valid Answer",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := models.EvaluationContext{
				Answer: test.answer,
			}
			response := result.Check(ctx)
			if response.Score != test.score {
				t.Errorf("Score: %f, want: %f", response.Score, test.score)
			}

			if response.Reason != test.reason {
				t.Errorf("Reason: %s, want: %s", response.Reason, test.reason)
			}
		})

	}
}
