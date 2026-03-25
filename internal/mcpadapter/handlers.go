package mcpadapter

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/Terminus-Lab/themis/internal/executor"
	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/Terminus-Lab/themis/internal/storage"
)

// NewEvaluateConversationHandler returns a tool handler for conversation evaluation.
func NewEvaluateConversationHandler(eval *executor.ConversationEvaluator) func(context.Context, *mcp.CallToolRequest, EvaluateConversationInput) (*mcp.CallToolResult, models.ConversationEvaluationResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input EvaluateConversationInput) (*mcp.CallToolResult, models.ConversationEvaluationResult, error) {
		return EvaluateConversation(ctx, eval, req, input)
	}
}

// EvaluateConversation runs the conversation evaluation pipeline and returns the result.
func EvaluateConversation(
	ctx context.Context,
	eval *executor.ConversationEvaluator,
	req *mcp.CallToolRequest,
	input EvaluateConversationInput,
) (*mcp.CallToolResult, models.ConversationEvaluationResult, error) {
	if input.ConversationID == "" {
		return nil, models.ConversationEvaluationResult{}, fmt.Errorf("conversation_id is required")
	}
	if len(input.Turns) == 0 {
		return nil, models.ConversationEvaluationResult{}, fmt.Errorf("turns must not be empty")
	}

	turns := make([]models.ConversationTurn, len(input.Turns))
	for i, t := range input.Turns {
		turns[i] = models.ConversationTurn{
			TurnIndex: t.TurnIndex,
			UserQuery: t.UserQuery,
			Answer:    t.Answer,
			Context:   t.Context,
		}
	}

	convReq := models.ConversationEvaluationRequest{
		ConversationID: input.ConversationID,
		Agent: models.Agent{
			Name:    input.AgentName,
			Version: input.AgentVersion,
		},
		Turns: turns,
	}

	result := eval.Execute(ctx, convReq)
	return nil, result, nil
}

// NewGetConversationHandler returns a tool handler for retrieving a conversation.
func NewGetConversationHandler(repo storage.Repository) func(context.Context, *mcp.CallToolRequest, ConversationInput) (*mcp.CallToolResult, *storage.ConversationRecord, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ConversationInput) (*mcp.CallToolResult, *storage.ConversationRecord, error) {
		return GetConversation(ctx, repo, req, input)
	}
}

// GetConversation retrieves the conversation evaluation record by ID.
func GetConversation(
	ctx context.Context,
	repo storage.Repository,
	req *mcp.CallToolRequest,
	input ConversationInput,
) (*mcp.CallToolResult, *storage.ConversationRecord, error) {
	if input.ConversationID == "" {
		return nil, nil, fmt.Errorf("conversation_id is required")
	}

	record, err := repo.GetConversation(ctx, input.ConversationID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	return nil, record, nil
}
