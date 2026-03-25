package batch

import "github.com/Terminus-Lab/themis/internal/models"

// ConversationInputRecord is a single conversation record read from a JSONL input file.
type ConversationInputRecord struct {
	LineNumber int
	Request    models.ConversationEvaluationRequest
	HumanLabel string   // optional: "pass", "review", "fail"
	HumanScore *float64 // optional: 0.0–1.0
	Error      error
}
