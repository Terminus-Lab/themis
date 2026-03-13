package storage

import (
	"time"

	"github.com/Terminus-Lab/themis/internal/models"
)

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
