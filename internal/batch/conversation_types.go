package batch

import "github.com/Terminus-Lab/themis/internal/models"

// ConversationInputRecord is a single conversation record read from a JSONL input file.
type ConversationInputRecord struct {
	LineNumber int
	Request    models.ConversationEvaluationRequest
	Error      error
}
