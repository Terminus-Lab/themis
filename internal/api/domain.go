package api

import "github.com/Terminus-Lab/themis/internal/storage"

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
