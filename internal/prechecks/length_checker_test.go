package prechecks

import (
	"strings"
	"testing"

	"github.com/Terminus-Lab/themis/internal/models"
)

func TestLengthChecker(t *testing.T) {
	checker := NewLengthChecker()

	tests := []struct {
		name       string
		query      string
		answer     string
		wantScore  float64
		wantReason string
	}{
		{
			name:       "empty query",
			query:      "",
			answer:     "anything",
			wantScore:  0,
			wantReason: "Empty query",
		},
		{
			name:       "empty answer",
			query:      "hello",
			answer:     "hi",
			wantScore:  0,
			wantReason: "fewer characters",
		},
		{
			name:       "no overlap",
			query:      "hi",
			answer:     "hello world",
			wantScore:  1.0,
			wantReason: "acceptable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := models.EvaluationContext{

				Query:  tt.query,
				Answer: tt.answer,
			}

			got := checker.Check(ctx)
			if got.Score != tt.wantScore {
				t.Errorf("Score: %v, want %v", got.Score, tt.wantScore)
			}
			if tt.wantReason != "" && !strings.Contains(got.Reason, tt.wantReason) {
				t.Errorf("Reason: %q, want substring %q", got.Reason, tt.wantReason)
			}
		})
	}

}
