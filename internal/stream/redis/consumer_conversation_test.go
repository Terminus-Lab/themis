package redis

import (
	"encoding/json"
	"testing"

	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/redis/go-redis/v9"
)

func TestDecodeConversationPayload(t *testing.T) {
	turns := []models.ConversationTurn{
		{TurnIndex: 0, UserQuery: "What is AI?", Answer: "AI is Artificial Intelligence."},
		{TurnIndex: 1, UserQuery: "How does it work?", Answer: "It uses machine learning."},
	}

	tests := []struct {
		name        string
		msg         redis.XMessage
		wantErr     bool
		validate    func(*testing.T, models.ConversationEvaluationRequest)
	}{
		{
			name: "valid conversation payload",
			msg: xMessage(t, models.ConversationEvaluationRequest{
				ConversationID: "conv-123",
				Agent:          models.Agent{Name: "test-agent", Version: "v1.0"},
				Turns:          turns,
			}),
			wantErr: false,
			validate: func(t *testing.T, req models.ConversationEvaluationRequest) {
				if req.ConversationID != "conv-123" {
					t.Errorf("expected ConversationID='conv-123', got '%s'", req.ConversationID)
				}
				if req.Agent.Name != "test-agent" {
					t.Errorf("expected Agent.Name='test-agent', got '%s'", req.Agent.Name)
				}
				if len(req.Turns) != 2 {
					t.Errorf("expected 2 turns, got %d", len(req.Turns))
				}
				if req.Turns[0].UserQuery != "What is AI?" {
					t.Errorf("expected first turn query='What is AI?', got '%s'", req.Turns[0].UserQuery)
				}
			},
		},
		{
			name: "single turn conversation",
			msg: xMessage(t, models.ConversationEvaluationRequest{
				ConversationID: "conv-single",
				Agent:          models.Agent{Name: "agent", Version: "v2.0"},
				Turns: []models.ConversationTurn{
					{TurnIndex: 0, UserQuery: "Hello", Answer: "Hi there"},
				},
			}),
			wantErr: false,
			validate: func(t *testing.T, req models.ConversationEvaluationRequest) {
				if len(req.Turns) != 1 {
					t.Errorf("expected 1 turn, got %d", len(req.Turns))
				}
			},
		},
		{
			name: "missing payload field",
			msg: redis.XMessage{
				ID:     "1-0",
				Values: map[string]any{},
			},
			wantErr: true,
		},
		{
			name: "payload is not a string",
			msg: redis.XMessage{
				ID:     "1-0",
				Values: map[string]any{"payload": 12345},
			},
			wantErr: true,
		},
		{
			name: "invalid JSON payload",
			msg: redis.XMessage{
				ID:     "1-0",
				Values: map[string]any{"payload": `{invalid json}`},
			},
			wantErr: true,
		},
		{
			name: "empty JSON object",
			msg: redis.XMessage{
				ID:     "1-0",
				Values: map[string]any{"payload": `{}`},
			},
			wantErr: false,
			validate: func(t *testing.T, req models.ConversationEvaluationRequest) {
				if req.ConversationID != "" {
					t.Errorf("expected empty ConversationID, got '%s'", req.ConversationID)
				}
				if len(req.Turns) != 0 {
					t.Errorf("expected 0 turns, got %d", len(req.Turns))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := decodeConversationPayload(tt.msg)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.validate != nil {
				tt.validate(t, req)
			}
		})
	}
}

// xMessage serialises req as JSON and wraps it in a Redis XMessage.
func xMessage(t *testing.T, req models.ConversationEvaluationRequest) redis.XMessage {
	t.Helper()
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	return redis.XMessage{
		ID:     "1-0",
		Values: map[string]any{"payload": string(b)},
	}
}
