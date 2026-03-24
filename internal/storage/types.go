package storage

import (
	"time"

	"github.com/Terminus-Lab/themis/internal/models"
)

// HealthMetricsData holds aggregated metrics for the health endpoint.
type HealthMetricsData struct {
	TotalEvaluations int
	AvgConfidence    float64
}

// ConversationRecord is the database record for a full conversation evaluation.
type ConversationRecord struct {
	ID             string
	ConversationID string
	AgentName      string
	AgentVersion   string
	TurnCount      int
	TurnAvg        float64
	HolisticScore  float64
	HolisticReason string
	FinalScore     float64
	Verdict        string
	TurnResults    []models.TurnEvaluationResult
	CreatedAt      time.Time
}

// ConversationSummary provides aggregated metrics for the list endpoint.
type ConversationSummary struct {
	ConversationID string
	AgentName      string
	AgentVersion   string
	TurnCount      int
	FinalScore     float64
	Verdict        string
	CreatedAt      time.Time
}
