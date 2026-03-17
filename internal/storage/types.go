package storage

import (
	"time"

	"github.com/Terminus-Lab/themis/internal/models"
)

// HealthMetricsData holds aggregated metrics for the health endpoint
type HealthMetricsData struct {
	TotalEvaluations int
	AvgConfidence    float64
}

// SampleFilters defines the parameters for sampling evaluation results
type SampleFilters struct {
	StartDate  time.Time
	EndDate    time.Time
	Percentage int // 1-100
	MinSize    int // minimum sample size (0 = no minimum)
	MaxSize    int // maximum sample size (0 = no maximum)
}

type Evaluation struct {
	EventID        string
	ConversationID string
	AgentName      string
	AgentVersion   string
	UserQuery      string
	Answer         string
	Context        string
	Confidence     float64
	Verdict        string
	StageScores    []models.StageResult
}

// ConversationSummary provides aggregated metrics for a conversation
type ConversationSummary struct {
	ConversationID string
	TurnCount      int
	AvgConfidence  float64
	PassCount      int
	ReviewCount    int
	FailCount      int
	FirstTurnAt    time.Time
	LastTurnAt     time.Time
	AgentName      string
	AgentVersion   string
}

// ConversationDetail provides full conversation data with all turns
type ConversationDetail struct {
	ConversationID string
	TurnCount      int
	AvgConfidence  float64
	AgentName      string
	AgentVersion   string
	Turns          []Evaluation
}
