package redis

import (
	"testing"
	"time"

	"github.com/Terminus-Lab/themis/internal/models"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name     string
		request  models.EvaluationRequest
		validate func(*testing.T, models.EvaluationContext)
	}{
		{
			name: "complete request with conversation_id",
			request: models.EvaluationRequest{
				EventID:        "evt-123",
				ConversationID: "conv-456",
				EventType:      models.EventTypeAgentResponse,
				Agent: models.Agent{
					Name:    "test-agent",
					Type:    "rag",
					Version: "v1.0",
				},
				Interaction: models.Interaction{
					UserQuery:      "What is AI?",
					Context:        "AI context",
					Answer:         "AI is Artificial Intelligence",
					ExpectedOutput: "Expected answer",
				},
			},
			validate: func(t *testing.T, ctx models.EvaluationContext) {
				if ctx.RequestID != "evt-123" {
					t.Errorf("Expected RequestID='evt-123', got '%s'", ctx.RequestID)
				}
				if ctx.ConversationID != "conv-456" {
					t.Errorf("Expected ConversationID='conv-456', got '%s'", ctx.ConversationID)
				}
				if ctx.AgentName != "test-agent" {
					t.Errorf("Expected AgentName='test-agent', got '%s'", ctx.AgentName)
				}
				if ctx.AgentVersion != "v1.0" {
					t.Errorf("Expected AgentVersion='v1.0', got '%s'", ctx.AgentVersion)
				}
				if ctx.Query != "What is AI?" {
					t.Errorf("Expected Query='What is AI?', got '%s'", ctx.Query)
				}
				if ctx.Context != "AI context" {
					t.Errorf("Expected Context='AI context', got '%s'", ctx.Context)
				}
				if ctx.Answer != "AI is Artificial Intelligence" {
					t.Errorf("Expected Answer='AI is Artificial Intelligence', got '%s'", ctx.Answer)
				}
				if ctx.ExpectedOutput != "Expected answer" {
					t.Errorf("Expected ExpectedOutput='Expected answer', got '%s'", ctx.ExpectedOutput)
				}
				if ctx.CreatedAt.IsZero() {
					t.Error("Expected CreatedAt to be set")
				}
			},
		},
		{
			name: "request without conversation_id",
			request: models.EvaluationRequest{
				EventID:   "evt-789",
				EventType: models.EventTypeAgentResponse,
				Agent: models.Agent{
					Name:    "other-agent",
					Version: "v2.0",
				},
				Interaction: models.Interaction{
					UserQuery: "Test query",
					Answer:    "Test answer",
				},
			},
			validate: func(t *testing.T, ctx models.EvaluationContext) {
				if ctx.RequestID != "evt-789" {
					t.Errorf("Expected RequestID='evt-789', got '%s'", ctx.RequestID)
				}
				if ctx.ConversationID != "" {
					t.Errorf("Expected empty ConversationID, got '%s'", ctx.ConversationID)
				}
				if ctx.AgentName != "other-agent" {
					t.Errorf("Expected AgentName='other-agent', got '%s'", ctx.AgentName)
				}
				if ctx.AgentVersion != "v2.0" {
					t.Errorf("Expected AgentVersion='v2.0', got '%s'", ctx.AgentVersion)
				}
			},
		},
		{
			name: "multi-turn conversation",
			request: models.EvaluationRequest{
				EventID:        "turn-2",
				ConversationID: "conv-multi",
				EventType:      models.EventTypeAgentResponse,
				Agent: models.Agent{
					Name:    "assistant",
					Version: "v1.0",
				},
				Interaction: models.Interaction{
					UserQuery: "Follow-up question",
					Context:   "Previous context",
					Answer:    "Follow-up answer",
				},
			},
			validate: func(t *testing.T, ctx models.EvaluationContext) {
				if ctx.ConversationID != "conv-multi" {
					t.Errorf("Expected ConversationID='conv-multi', got '%s'", ctx.ConversationID)
				}
				if ctx.RequestID != "turn-2" {
					t.Errorf("Expected RequestID='turn-2', got '%s'", ctx.RequestID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalize(tt.request)
			tt.validate(t, result)
		})
	}
}

func TestNormalize_CreatedAtTimestamp(t *testing.T) {
	request := models.EvaluationRequest{
		EventID: "evt-time-test",
		Agent: models.Agent{
			Name:    "agent",
			Version: "v1.0",
		},
		Interaction: models.Interaction{
			UserQuery: "Test",
			Answer:    "Test",
		},
	}

	before := time.Now()
	result := normalize(request)
	after := time.Now()

	if result.CreatedAt.Before(before) || result.CreatedAt.After(after) {
		t.Errorf("CreatedAt timestamp not within expected range: %v (should be between %v and %v)",
			result.CreatedAt, before, after)
	}
}
