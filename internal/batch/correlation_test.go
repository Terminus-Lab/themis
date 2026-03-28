package batch

import (
	"testing"

	"github.com/Terminus-Lab/themis/internal/models"
)

func ptr(f float64) *float64 { return &f }

func TestComputeCorrelationReport_Disagreements(t *testing.T) {
	tests := []struct {
		name                  string
		results               []models.ConversationEvaluationResult
		annotations           map[string]Annotation
		wantDisagreementIDs   []string // conversation_ids expected in Disagreements, in order
		wantDisagreementCount int
	}{
		{
			name: "all verdicts match — no disagreements",
			results: []models.ConversationEvaluationResult{
				{ConversationID: "c1", Verdict: "pass", FinalScore: 0.9},
				{ConversationID: "c2", Verdict: "fail", FinalScore: 0.2},
			},
			annotations: map[string]Annotation{
				"c1": {HumanLabel: "pass", HumanScore: ptr(0.92)},
				"c2": {HumanLabel: "fail", HumanScore: ptr(0.18)},
			},
			wantDisagreementCount: 0,
		},
		{
			name: "one disagreement — fields populated correctly",
			results: []models.ConversationEvaluationResult{
				{ConversationID: "c1", Verdict: "pass", FinalScore: 0.85},
				{ConversationID: "c2", Verdict: "pass", FinalScore: 0.83},
			},
			annotations: map[string]Annotation{
				"c1": {HumanLabel: "pass", HumanScore: ptr(0.90)},
				"c2": {HumanLabel: "fail", HumanScore: ptr(0.18)},
			},
			wantDisagreementCount: 1,
			wantDisagreementIDs:   []string{"c2"},
		},
		{
			name: "human_score nil when not provided",
			results: []models.ConversationEvaluationResult{
				{ConversationID: "c1", Verdict: "review", FinalScore: 0.6},
			},
			annotations: map[string]Annotation{
				"c1": {HumanLabel: "pass"}, // no HumanScore
			},
			wantDisagreementCount: 1,
			wantDisagreementIDs:   []string{"c1"},
		},
		{
			name: "multiple disagreements — agreements excluded",
			results: []models.ConversationEvaluationResult{
				{ConversationID: "c1", Verdict: "pass", FinalScore: 0.9},
				{ConversationID: "c2", Verdict: "fail", FinalScore: 0.2},
				{ConversationID: "c3", Verdict: "pass", FinalScore: 0.85},
				{ConversationID: "c4", Verdict: "review", FinalScore: 0.55},
			},
			annotations: map[string]Annotation{
				"c1": {HumanLabel: "pass", HumanScore: ptr(0.91)},
				"c2": {HumanLabel: "pass", HumanScore: ptr(0.88)}, // disagree
				"c3": {HumanLabel: "fail", HumanScore: ptr(0.15)}, // disagree
				"c4": {HumanLabel: "review", HumanScore: ptr(0.5)},
			},
			wantDisagreementCount: 2,
			wantDisagreementIDs:   []string{"c2", "c3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := ComputeCorrelationReport(tt.results, tt.annotations)

			if len(report.Disagreements) != tt.wantDisagreementCount {
				t.Fatalf("got %d disagreements, want %d", len(report.Disagreements), tt.wantDisagreementCount)
			}

			for i, wantID := range tt.wantDisagreementIDs {
				d := report.Disagreements[i]
				if d.ConversationID != wantID {
					t.Errorf("disagreement[%d].ConversationID = %q, want %q", i, d.ConversationID, wantID)
				}
				ann := tt.annotations[wantID]
				if d.HumanLabel != ann.HumanLabel {
					t.Errorf("disagreement[%d].HumanLabel = %q, want %q", i, d.HumanLabel, ann.HumanLabel)
				}
				if ann.HumanScore == nil && d.HumanScore != nil {
					t.Errorf("disagreement[%d].HumanScore should be nil", i)
				}
				if ann.HumanScore != nil && d.HumanScore == nil {
					t.Errorf("disagreement[%d].HumanScore should not be nil", i)
				}
			}
		})
	}
}
