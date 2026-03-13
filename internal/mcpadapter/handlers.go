package mcpadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/Terminus-Lab/themis/internal/executor"
	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/Terminus-Lab/themis/internal/storage"
)

// NewEvaluateHandler returns a tool handler that uses the given executor.
// Pass the returned function to mcp.AddTool.
func NewEvaluateHandler(exec *executor.Executor) func(context.Context, *mcp.CallToolRequest, EvaluateInput) (*mcp.CallToolResult, models.EvaluationResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input EvaluateInput) (*mcp.CallToolResult, models.EvaluationResult, error) {
		return EvaluateResponse(ctx, exec, req, input)
	}
}

// EvaluateResponse runs the full evaluation pipeline and returns the result.
func EvaluateResponse(
	ctx context.Context,
	exec *executor.Executor,
	req *mcp.CallToolRequest,
	input EvaluateInput,
) (*mcp.CallToolResult, models.EvaluationResult, error) {
	evalCtx := models.EvaluationContext{
		RequestID:      input.EventID,
		ConversationID: input.ConversationID,
		AgentName:      input.AgentName,
		AgentVersion:   input.AgentVersion,
		Query:          input.Query,
		Context:        input.Context,
		Answer:         input.Answer,
		ExpectedOutput: input.ExpectedOutput,
		CreatedAt:      time.Now(),
	}

	result := exec.Execute(ctx, evalCtx)
	return nil, result, nil
}

// NewEvaluateSingleJudgeHandler returns a tool handler for single judge evaluation.
// Pass the returned function to mcp.AddTool.
func NewEvaluateSingleJudgeHandler(judgeExec *executor.JudgeExecutor) func(context.Context, *mcp.CallToolRequest, EvaluateSingleJudgeInput) (*mcp.CallToolResult, models.EvaluationResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input EvaluateSingleJudgeInput) (*mcp.CallToolResult, models.EvaluationResult, error) {
		return EvaluateSingleJudge(ctx, judgeExec, req, input)
	}
}

// EvaluateSingleJudge runs evaluation with a single judge and returns the result.
func EvaluateSingleJudge(
	ctx context.Context,
	judgeExec *executor.JudgeExecutor,
	req *mcp.CallToolRequest,
	input EvaluateSingleJudgeInput,
) (*mcp.CallToolResult, models.EvaluationResult, error) {
	evalCtx := models.EvaluationContext{
		RequestID:      input.EventID,
		ConversationID: input.ConversationID,
		AgentName:      input.AgentName,
		AgentVersion:   input.AgentVersion,
		Query:          input.Query,
		Context:        input.Context,
		Answer:         input.Answer,
		ExpectedOutput: input.ExpectedOutput,
		CreatedAt:      time.Now(),
	}

	// Default threshold to 0.7 if not provided
	threshold := input.Threshold
	if threshold == 0.0 {
		threshold = 0.7
	}

	result, err := judgeExec.Execute(ctx, input.JudgeName, threshold, evalCtx)

	return nil, result, err
}

// NewGetConversationHandler returns a tool handler for retrieving conversation turns.
func NewGetConversationHandler(repo storage.Repository) func(context.Context, *mcp.CallToolRequest, ConversationInput) (*mcp.CallToolResult, storage.ConversationDetail, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ConversationInput) (*mcp.CallToolResult, storage.ConversationDetail, error) {
		return GetConversation(ctx, repo, req, input)
	}
}

// GetConversation retrieves all turns for a given conversation ID.
func GetConversation(
	ctx context.Context,
	repo storage.Repository,
	req *mcp.CallToolRequest,
	input ConversationInput,
) (*mcp.CallToolResult, storage.ConversationDetail, error) {
	if input.ConversationID == "" {
		return nil, storage.ConversationDetail{}, fmt.Errorf("conversation_id is required")
	}

	turns, err := repo.GetConversation(ctx, input.ConversationID)
	if err != nil {
		return nil, storage.ConversationDetail{}, fmt.Errorf("failed to get conversation: %w", err)
	}

	if len(turns) == 0 {
		return nil, storage.ConversationDetail{}, fmt.Errorf("conversation not found: %s", input.ConversationID)
	}

	// Calculate average confidence
	totalConfidence := 0.0
	for _, turn := range turns {
		totalConfidence += turn.Confidence
	}
	avgConfidence := totalConfidence / float64(len(turns))

	// Use agent info from first turn (should be same across all turns)
	result := storage.ConversationDetail{
		ConversationID: input.ConversationID,
		TurnCount:      len(turns),
		AvgConfidence:  avgConfidence,
		AgentName:      turns[0].AgentName,
		AgentVersion:   turns[0].AgentVersion,
		Turns:          turns,
	}

	return nil, result, nil
}
