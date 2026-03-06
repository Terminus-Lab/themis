package judge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/Terminus-Lab/themis/internal/config"
	"github.com/Terminus-Lab/themis/internal/llm"
	"github.com/Terminus-Lab/themis/internal/models"
	"github.com/rs/zerolog"
)

// LLMJudge is a generic judge implementation that uses LLM with configurable prompts.
type LLMJudge struct {
	name                   string
	promptTemplate         *template.Template
	modelConfig            config.ModelConfig
	requiresContext        bool
	requiresExpectedOutput bool
	weight                 float64
	llmClient              llm.LLMClient
	logger                 *zerolog.Logger
}

func NewLLMJudge(
	judgeCfg config.JudgeConfiguration,
	llmClient llm.LLMClient,
	logger *zerolog.Logger,
) (*LLMJudge, error) {
	tmpl, err := template.New(judgeCfg.Name).Parse(judgeCfg.Prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prompt template for judge %s: %w", judgeCfg.Name, err)
	}

	if judgeCfg.Model == nil {
		return nil, fmt.Errorf("judge %s has nil model config (should be populated by config loader)", judgeCfg.Name)
	}

	return &LLMJudge{
		name:                   judgeCfg.Name,
		promptTemplate:         tmpl,
		modelConfig:            *judgeCfg.Model,
		requiresContext:        judgeCfg.RequiresContext,
		requiresExpectedOutput: judgeCfg.RequiresExpectedOutput,
		weight:                 judgeCfg.Weight,
		llmClient:              llmClient,
		logger:                 logger,
	}, nil
}

// Evaluate executes the judge evaluation
func (j *LLMJudge) Evaluate(ctx context.Context, evalCtx models.EvaluationContext) models.StageResult {
	now := time.Now()

	result := models.StageResult{
		Name:   fmt.Sprintf("%s-judge", j.name),
		Score:  0.0,
		Weight: j.weight,
	}

	// Check if context is required but missing
	if j.requiresContext && evalCtx.Context == "" {
		j.logger.Warn().
			Str("judge", j.name).
			Msg("judge requires context but none provided")
		result.Reason = "Context required but not provided"
		result.Duration = time.Since(now)
		return result
	}

	// Build prompt from template
	prompt, err := j.buildPrompt(evalCtx)
	if err != nil {
		j.logger.Error().
			Err(err).
			Str("judge", j.name).
			Msg("failed to build prompt from template")
		result.Reason = fmt.Sprintf("Failed to build prompt: %v", err)
		result.Duration = time.Since(now)
		return result
	}

	// Call LLM
	var resp *llm.LLMResponse
	if j.modelConfig.Retry {
		resp, err = j.llmClient.InvokeModelWithRetry(ctx, llm.LLMRequest{
			Prompt:      prompt,
			MaxTokens:   j.modelConfig.MaxTokens,
			Temperature: j.modelConfig.Temperature,
		})
	} else {
		resp, err = j.llmClient.InvokeModel(ctx, llm.LLMRequest{
			Prompt:      prompt,
			MaxTokens:   j.modelConfig.MaxTokens,
			Temperature: j.modelConfig.Temperature,
		})
	}

	if err != nil {
		j.logger.Error().
			Err(err).
			Str("judge", j.name).
			Msg("LLM call failed")
		result.Reason = "Failed to call LLM"
		result.Duration = time.Since(now)
		return result
	}

	// Parse LLM response (strip markdown code blocks if present)
	content := stripMarkdownCodeBlock(resp.Content)
	var llmResponse judgeResponse
	if err := json.Unmarshal([]byte(content), &llmResponse); err != nil {
		j.logger.Error().
			Err(err).
			Str("judge", j.name).
			Str("content", resp.Content).
			Msg("failed to deserialize LLM response")
		result.Reason = "Failed to deserialize LLM response"
		result.Duration = time.Since(now)
		return result
	}

	// Validate response
	if llmResponse.Score == 0.0 && llmResponse.Reason == "" {
		j.logger.Error().
			Str("judge", j.name).
			Msg("LLM returned empty score and reason")
		result.Reason = "Invalid LLM response: missing score and reason"
		result.Duration = time.Since(now)
		return result
	}

	if llmResponse.Score < 0.0 || llmResponse.Score > 1.0 {
		j.logger.Error().
			Str("judge", j.name).
			Float64("score", llmResponse.Score).
			Msg("LLM returned invalid score")
		result.Reason = fmt.Sprintf("Invalid LLM response: score %f out of range [0.0, 1.0]", llmResponse.Score)
		result.Duration = time.Since(now)
		return result
	}

	// Success
	result.Score = llmResponse.Score
	result.Reason = llmResponse.Reason
	result.Duration = time.Since(now)

	j.logger.Info().
		Str("judge", j.name).
		Float64("score", result.Score).
		Dur("duration", result.Duration).
		Msg("judge completed")

	return result
}

// Name returns the judge's name
func (j *LLMJudge) Name() string {
	return j.name
}

// RequiresExpectedOutput returns true if this judge needs expected_output
func (j *LLMJudge) RequiresExpectedOutput() bool {
	return j.requiresExpectedOutput
}

// buildPrompt executes the template with the evaluation context
func (j *LLMJudge) buildPrompt(evalCtx models.EvaluationContext) (string, error) {
	var buf bytes.Buffer
	if err := j.promptTemplate.Execute(&buf, evalCtx); err != nil {
		return "", fmt.Errorf("template execution failed: %w", err)
	}
	return buf.String(), nil
}

// stripMarkdownCodeBlock removes markdown code block formatting if present
func stripMarkdownCodeBlock(content string) string {
	content = strings.TrimSpace(content)

	// Check for markdown code blocks (```json ... ``` or ``` ... ```)
	if strings.HasPrefix(content, "```") {
		// Find the first newline (after the opening ```)
		firstNewline := strings.Index(content, "\n")
		if firstNewline == -1 {
			return content
		}

		// Find the closing ```
		closingBackticks := strings.LastIndex(content, "```")
		if closingBackticks == -1 || closingBackticks <= firstNewline {
			return content
		}

		// Extract the content between the code blocks
		content = content[firstNewline+1 : closingBackticks]
		content = strings.TrimSpace(content)
	}

	return content
}
