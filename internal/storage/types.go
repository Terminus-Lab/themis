package storage

import "github.com/Terminus-Lab/themis/internal/models"

type QueryFilters struct {
	AgentName string
	Verdict   string
	Limit     int
	Offset    int
	Count     int
}

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
