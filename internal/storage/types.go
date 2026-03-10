package storage

import "github.com/Terminus-Lab/themis/internal/models"

type Evaluation struct {
	EventID      string
	AgentName    string
	AgentVersion string
	UserQuery    string
	Answer       string
	Context      string
	Confidence   float64
	Verdict      string
	StageScores  []models.StageResult
}
