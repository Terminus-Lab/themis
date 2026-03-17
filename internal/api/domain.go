package api

import (
	"time"

	"github.com/Terminus-Lab/themis/internal/storage"
)

type HealthResponse struct {
	Status  string `json:"status" description:"Service status"`
	Version string `json:"version" description:"API version"`
}

// EvaluationDTO is the API response model for evaluation results
type EvaluationDTO struct {
	EventID        string       `json:"event_id" description:"Unique event identifier"`
	ConversationID string       `json:"conversation_id" description:"Conversation ID"`
	AgentName      string       `json:"agent_name" description:"Name of the agent"`
	AgentVersion   string       `json:"agent_version" description:"Version of the agent"`
	UserQuery      string       `json:"user_query" description:"User's original query"`
	Answer         string       `json:"answer" description:"Agent's answer"`
	Context        string       `json:"context,omitempty" description:"Retrieved context (optional)"`
	Confidence     float64      `json:"confidence" description:"Overall confidence score"`
	Verdict        string       `json:"verdict" description:"Evaluation verdict (pass, review, fail)"`
	StageScores    []StageScore `json:"stage_scores" description:"Individual stage evaluation scores"`
}

// StageScore represents an individual evaluation stage result
type StageScore struct {
	Name   string  `json:"name" description:"Stage/judge name"`
	Score  float64 `json:"score" description:"Score (0.0-1.0)"`
	Reason string  `json:"reason,omitempty" description:"Evaluation reasoning (optional)"`
	Weight float64 `json:"weight,omitempty" description:"Weight of this judge"`
}

type QueryResultsResponse struct {
	Results []EvaluationDTO `json:"results" description:"List of evaluation results"`
	Total   int             `json:"total" description:"Total number of matching results"`
	Count   int             `json:"count" description:"Number of results in this response"`
	Limit   int             `json:"limit" description:"Limit used in query"`
	Offset  int             `json:"offset" description:"Offset used in query"`
	HasMore bool            `json:"has_more" description:"Whether there are more results available"`
}

type EvaluationResponse struct {
	Evaluation EvaluationDTO `json:"evaluation" description:"Evaluation result"`
}

type ConversationDetailResponse struct {
	ConversationID string          `json:"conversation_id" description:"Conversation ID"`
	TurnCount      int             `json:"turn_count" description:"The number of interactions/events from the conversation"`
	AvgConfidence  float64         `json:"avg_confidence" description:"Average confidence across all turns"`
	AgentName      string          `json:"agent_name" description:"Agent name"`
	AgentVersion   string          `json:"agent_version" description:"Agent version"`
	Turns          []EvaluationDTO `json:"turns"` // Full Evaluation
}

type ConversationSummaryDTO struct {
	ConversationID string  `json:"conversation_id"`
	TurnCount      int     `json:"turn_count"`
	AvgConfidence  float64 `json:"avg_confidence"`
	VerdictCounts  struct {
		Pass   int `json:"pass"`
		Review int `json:"review"`
		Fail   int `json:"fail"`
	} `json:"verdict_counts"`
	FirstTurnAt  time.Time `json:"first_turn_at"`
	LastTurnAt   time.Time `json:"last_turn_at"`
	AgentName    string    `json:"agent_name"`
	AgentVersion string    `json:"agent_version"`
}

type ConversationListResponse struct {
	Conversations []ConversationSummaryDTO `json:"conversations"`
	Total         int                      `json:"total"`
}

// HealthMetricsResponse is the response for GET /api/v1/metrics/health
type HealthMetricsResponse struct {
	Window           string  `json:"window"`
	TotalEvaluations int     `json:"total_evaluations"`
	AvgConfidence    float64 `json:"avg_confidence"`
}

// SampleRecord is the JSONL record format for the sample download.
// It contains only interaction data for human annotation — no Themis evaluation results.
type SampleRecord struct {
	EventID        string `json:"event_id"`
	ConversationID string `json:"conversation_id"`
	Agent          struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"agent"`
	Interaction struct {
		UserQuery string `json:"user_query"`
		Context   string `json:"context,omitempty"`
		Answer    string `json:"answer"`
	} `json:"interaction"`
}

// SampleRequest is the request body for POST /api/v1/validation/sample/download
type SampleRequest struct {
	StartDate  string `json:"start_date" description:"Start of date range (RFC3339, e.g. 2026-01-01T00:00:00Z)"`
	EndDate    string `json:"end_date" description:"End of date range (RFC3339, e.g. 2026-03-31T23:59:59Z)"`
	Percentage int    `json:"percentage" description:"Percentage of records to sample (1-100, default: 25)"`
	MinSize    int    `json:"min_size,omitempty" description:"Minimum sample size (0 = no minimum)"`
	MaxSize    int    `json:"max_size,omitempty" description:"Maximum sample size (0 = no maximum)"`
}

// toEvaluationDTO converts storage.Evaluation to API DTO
func toEvaluationDTO(e storage.Evaluation) EvaluationDTO {
	stageScores := make([]StageScore, len(e.StageScores))
	for i, s := range e.StageScores {
		stageScores[i] = StageScore{
			Name:   s.Name,
			Score:  s.Score,
			Reason: s.Reason,
			Weight: s.Weight,
		}
	}

	return EvaluationDTO{
		EventID:        e.EventID,
		ConversationID: e.ConversationID,
		AgentName:      e.AgentName,
		AgentVersion:   e.AgentVersion,
		UserQuery:      e.UserQuery,
		Answer:         e.Answer,
		Context:        e.Context,
		Confidence:     e.Confidence,
		Verdict:        e.Verdict,
		StageScores:    stageScores,
	}
}

// toEvaluationDTOs converts slice of storage.Evaluation to DTOs
func toEvaluationDTOs(evaluations []storage.Evaluation) []EvaluationDTO {
	dtos := make([]EvaluationDTO, len(evaluations))
	for i, e := range evaluations {
		dtos[i] = toEvaluationDTO(e)
	}
	return dtos
}

// toConversationDetailResponse converts storage.ConversationDetail to API response DTO
func toConversationDetailResponse(detail storage.ConversationDetail) ConversationDetailResponse {
	dtos := make([]EvaluationDTO, len(detail.Turns))
	for i, turn := range detail.Turns {
		dtos[i] = toEvaluationDTO(turn)
	}

	return ConversationDetailResponse{
		ConversationID: detail.ConversationID,
		TurnCount:      detail.TurnCount,
		AvgConfidence:  detail.AvgConfidence,
		AgentName:      detail.AgentName,
		AgentVersion:   detail.AgentVersion,
		Turns:          dtos,
	}
}

func toConversationSummaryDTO(conversationSummaries storage.ConversationSummary) ConversationSummaryDTO {
	return ConversationSummaryDTO{
		ConversationID: conversationSummaries.ConversationID,
		TurnCount:      conversationSummaries.TurnCount,
		AvgConfidence:  conversationSummaries.AvgConfidence,
		VerdictCounts: struct {
			Pass   int `json:"pass"`
			Review int `json:"review"`
			Fail   int `json:"fail"`
		}{
			Pass:   conversationSummaries.PassCount,
			Review: conversationSummaries.ReviewCount,
			Fail:   conversationSummaries.FailCount,
		},
		FirstTurnAt:  conversationSummaries.FirstTurnAt,
		LastTurnAt:   conversationSummaries.LastTurnAt,
		AgentName:    conversationSummaries.AgentName,
		AgentVersion: conversationSummaries.AgentVersion,
	}
}

func toConversationSummaryDTOs(summaries []storage.ConversationSummary) []ConversationSummaryDTO {
	dtos := make([]ConversationSummaryDTO, len(summaries))
	for i, summary := range summaries {
		dtos[i] = toConversationSummaryDTO(summary)
	}
	return dtos
}
