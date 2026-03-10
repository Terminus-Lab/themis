package batch

import "github.com/Terminus-Lab/themis/internal/models"

type InputRecord struct {
	LineNumber int
	Request    models.EvaluationRequest
	Error      error
}
