package models

import (
	"time"
)

type Verdict string

const (
	VerdictPass   Verdict = "pass"
	VerdictFail   Verdict = "fail"
	VerdictReview Verdict = "review"
)

type Agent struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Version string `json:"version"`
}

// ConversationTurn represents a single turn in a multi-turn conversation.
type ConversationTurn struct {
	TurnIndex      int    `json:"turn_index"`
	UserQuery      string `json:"user_query"`
	Answer         string `json:"answer"`
	Context        string `json:"context,omitempty"`
	ExpectedOutput string `json:"expected_output,omitempty"`
}

// ConversationEvaluationRequest is the API input for conversation-level evaluation.
type ConversationEvaluationRequest struct {
	ConversationID string             `json:"conversation_id"` // Required
	Agent          Agent              `json:"agent"`
	Turns          []ConversationTurn `json:"turns"` // All turns in the conversation
}

// TurnEvaluationResult holds the evaluation scores for a single conversation turn.
type TurnEvaluationResult struct {
	TurnIndex int           `json:"turn_index"`
	UserQuery string        `json:"user_query"`
	Answer    string        `json:"answer"`
	Scores    []StageResult `json:"scores"`
	TurnScore float64       `json:"turn_score"` // average of Scores
}

// ConversationEvaluationResult is the output of conversation-level evaluation.
type ConversationEvaluationResult struct {
	ConversationID string                 `json:"conversation_id"`
	AgentName      string                 `json:"agent_name"`
	AgentVersion   string                 `json:"agent_version"`
	TurnCount      int                    `json:"turn_count"`
	TurnResults    []TurnEvaluationResult `json:"turn_results"`
	TurnAvg        float64                `json:"turn_avg"`
	HolisticScore  float64                `json:"holistic_score"`
	HolisticReason string                 `json:"holistic_reason"`
	FinalScore     float64                `json:"final_score"`
	Verdict        Verdict                `json:"verdict"`
	EvalErrors     []string               `json:"eval_errors,omitempty"`
}

// EvaluationContext is the normalized internal context passed to judges.
type EvaluationContext struct {
	RequestID      string             `json:"request_id"`
	ConversationID string             `json:"conversation_id"`
	AgentName      string             `json:"agent_name,omitempty"`
	AgentVersion   string             `json:"agent_version,omitempty"`
	Query          string             `json:"user_query"`
	Context        string             `json:"context,omitempty"`
	Answer         string             `json:"answer"`
	ExpectedOutput string             `json:"expected_output,omitempty"`
	Turns          []ConversationTurn `json:"turns,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
}

// StageResult is a single judge's output.
type StageResult struct {
	Name     string        `json:"name"`
	Score    float64       `json:"score"`
	Reason   string        `json:"reason"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration_ns"`
	Weight   float64       `json:"weight,omitempty"`
}

type QueryFilters struct {
	AgentName string
	Verdict   string
	Limit     int
	Offset    int
	Count     int
}
