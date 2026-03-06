package azure

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/Terminus-Lab/themis/internal/llm"
)

func (c *Client) InvokeModel(ctx context.Context, request llm.LLMRequest) (*llm.LLMResponse, error) {

	message := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(request.Prompt),
		},
		MaxCompletionTokens: openai.Int(int64(request.MaxTokens)),
		Temperature:         openai.Float(request.Temperature),
		Model:               openai.ChatModel(c.ModelID),
	}

	output, err := c.Client.Chat.Completions.New(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("unable to invoke Azure OpenAI model. Error: %w", err)

	}

	if len(output.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	response := output.Choices[0]
	return &llm.LLMResponse{
		Content:    response.Message.Content,
		StopReason: fmt.Sprint(response.FinishReason),
	}, nil
}

func (c *Client) InvokeModelWithRetry(ctx context.Context, request llm.LLMRequest) (*llm.LLMResponse, error) {
	return c.InvokeModel(ctx, request)
}
