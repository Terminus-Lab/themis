package api

import (
	"time"

	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/Terminus-Lab/themis/internal/storage"
)

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type HealthMetricsResponse struct {
	Window           string  `json:"window"`
	TotalEvaluations int     `json:"total_evaluations"`
	AvgConfidence    float64 `json:"avg_confidence"`
}

// ConversationEvalRequest is the API request body for POST /api/v1/conversations/evaluate.
type ConversationEvalRequest struct {
	ConversationID string `json:"conversation_id"`
	Agent          struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"agent"`
	Turns []ConversationTurnRequest `json:"turns"`
}

// ConversationTurnRequest is a single turn in a ConversationEvalRequest.
type ConversationTurnRequest struct {
	TurnIndex      int    `json:"turn_index"`
	UserQuery      string `json:"user_query"`
	Answer         string `json:"answer"`
	Context        string `json:"context,omitempty"`
	ExpectedOutput string `json:"expected_output,omitempty"`
}

// TurnScoreDTO represents evaluation scores for a single turn.
type TurnScoreDTO struct {
	TurnIndex int        `json:"turn_index"`
	UserQuery string     `json:"user_query"`
	Answer    string     `json:"answer"`
	Scores    []ScoreDTO `json:"scores"`
	TurnScore float64    `json:"turn_score"`
}

// ScoreDTO represents a single judge's score.
type ScoreDTO struct {
	Name   string  `json:"name"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason,omitempty"`
	Weight float64 `json:"weight,omitempty"`
}

// ConversationEvalResponse is the API response for POST /api/v1/conversations/evaluate.
type ConversationEvalResponse struct {
	ConversationID string        `json:"conversation_id"`
	AgentName      string        `json:"agent_name"`
	AgentVersion   string        `json:"agent_version"`
	TurnCount      int           `json:"turn_count"`
	TurnResults    []TurnScoreDTO `json:"turn_results"`
	TurnAvg        float64       `json:"turn_avg"`
	HolisticScore  float64       `json:"holistic_score"`
	HolisticReason string        `json:"holistic_reason"`
	FinalScore     float64       `json:"final_score"`
	Verdict        string        `json:"verdict"`
}

// ConversationDetailResponse is the API response for GET /api/v1/conversations/{id}.
type ConversationDetailResponse struct {
	ConversationID string        `json:"conversation_id"`
	AgentName      string        `json:"agent_name"`
	AgentVersion   string        `json:"agent_version"`
	TurnCount      int           `json:"turn_count"`
	TurnResults    []TurnScoreDTO `json:"turn_results"`
	TurnAvg        float64       `json:"turn_avg"`
	HolisticScore  float64       `json:"holistic_score"`
	HolisticReason string        `json:"holistic_reason"`
	FinalScore     float64       `json:"final_score"`
	Verdict        string        `json:"verdict"`
	CreatedAt      time.Time     `json:"created_at"`
}

// ConversationSummaryDTO is a single item in the list endpoint.
type ConversationSummaryDTO struct {
	ConversationID string    `json:"conversation_id"`
	AgentName      string    `json:"agent_name"`
	AgentVersion   string    `json:"agent_version"`
	TurnCount      int       `json:"turn_count"`
	FinalScore     float64   `json:"final_score"`
	Verdict        string    `json:"verdict"`
	CreatedAt      time.Time `json:"created_at"`
}

type ConversationListResponse struct {
	Conversations []ConversationSummaryDTO `json:"conversations"`
	Total         int                      `json:"total"`
}

// toConversationEvalResponse converts models.ConversationEvaluationResult → API DTO.
func toConversationEvalResponse(r models.ConversationEvaluationResult) ConversationEvalResponse {
	turns := make([]TurnScoreDTO, len(r.TurnResults))
	for i, tr := range r.TurnResults {
		turns[i] = toTurnScoreDTO(tr)
	}
	return ConversationEvalResponse{
		ConversationID: r.ConversationID,
		AgentName:      r.AgentName,
		AgentVersion:   r.AgentVersion,
		TurnCount:      r.TurnCount,
		TurnResults:    turns,
		TurnAvg:        r.TurnAvg,
		HolisticScore:  r.HolisticScore,
		HolisticReason: r.HolisticReason,
		FinalScore:     r.FinalScore,
		Verdict:        string(r.Verdict),
	}
}

// toConversationDetailResponse converts storage.ConversationRecord → API DTO.
func toConversationDetailResponse(r *storage.ConversationRecord) ConversationDetailResponse {
	turns := make([]TurnScoreDTO, len(r.TurnResults))
	for i, tr := range r.TurnResults {
		turns[i] = toTurnScoreDTO(tr)
	}
	return ConversationDetailResponse{
		ConversationID: r.ConversationID,
		AgentName:      r.AgentName,
		AgentVersion:   r.AgentVersion,
		TurnCount:      r.TurnCount,
		TurnResults:    turns,
		TurnAvg:        r.TurnAvg,
		HolisticScore:  r.HolisticScore,
		HolisticReason: r.HolisticReason,
		FinalScore:     r.FinalScore,
		Verdict:        r.Verdict,
		CreatedAt:      r.CreatedAt,
	}
}

func toTurnScoreDTO(tr models.TurnEvaluationResult) TurnScoreDTO {
	scores := make([]ScoreDTO, len(tr.Scores))
	for i, s := range tr.Scores {
		scores[i] = ScoreDTO{
			Name:   s.Name,
			Score:  s.Score,
			Reason: s.Reason,
			Weight: s.Weight,
		}
	}
	return TurnScoreDTO{
		TurnIndex: tr.TurnIndex,
		UserQuery: tr.UserQuery,
		Answer:    tr.Answer,
		Scores:    scores,
		TurnScore: tr.TurnScore,
	}
}
