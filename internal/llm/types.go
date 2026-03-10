package llm

type LLMRequest struct {
	Prompt      string
	MaxTokens   int
	Temperature float64
}

type LLMResponse struct {
	Content    string
	StopReason string
}
