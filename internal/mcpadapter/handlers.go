package mcpadapter

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/Terminus-Lab/themis/internal/executor"
	"github.com/Terminus-Lab/themis/internal/models"
)

// EvaluateInput is the MCP tool input schema for full pipeline evaluation.
type EvaluateInput struct {
	EventID        string `json:"event_id" jsonschema:"unique event identifier"`
	Query          string `json:"user_query" jsonschema:"user's original query"`
	Answer         string `json:"answer" jsonschema:"agent response to evaluate"`
	Context        string `json:"context,omitempty" jsonschema:"optional context or retrieved documents"`
	ExpectedOutput string `json:"expected_output,omitempty" jsonschema:"optional ground truth for correctness evaluation"`
}

// EvaluateSingleJudgeInput is the MCP tool input schema for single judge evaluation.
type EvaluateSingleJudgeInput struct {
	EventID        string  `json:"event_id" jsonschema:"unique event identifier"`
	Query          string  `json:"user_query" jsonschema:"user's original query"`
	Answer         string  `json:"answer" jsonschema:"agent response to evaluate"`
	Context        string  `json:"context,omitempty" jsonschema:"optional context or retrieved documents"`
	ExpectedOutput string  `json:"expected_output,omitempty" jsonschema:"optional ground truth for correctness evaluation"`
	JudgeName      string  `json:"judge_name" jsonschema:"judge name: relevance, faithfulness, coherence, completeness, instruction, or correctness"`
	Threshold      float64 `json:"threshold,omitempty" jsonschema:"pass/fail threshold (0.0-1.0, default: 0.7)"`
}

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
