package mcpadapter

// EvaluateInput is the MCP tool input schema for full pipeline evaluation.
type EvaluateInput struct {
	EventID        string `json:"event_id" jsonschema:"unique event identifier"`
	ConversationID string `json:"conversation_id" jsonschema:"required conversation identifier to group multi-turn interactions"`
	AgentName      string `json:"agent_name,omitempty" jsonschema:"optional name of the agent being evaluated"`
	AgentVersion   string `json:"agent_version,omitempty" jsonschema:"optional version of the agent being evaluated"`
	Query          string `json:"user_query" jsonschema:"user's original query"`
	Answer         string `json:"answer" jsonschema:"agent response to evaluate"`
	Context        string `json:"context,omitempty" jsonschema:"optional context or retrieved documents"`
	ExpectedOutput string `json:"expected_output,omitempty" jsonschema:"optional ground truth for correctness evaluation"`
}

// EvaluateSingleJudgeInput is the MCP tool input schema for single judge evaluation.
type EvaluateSingleJudgeInput struct {
	EventID        string  `json:"event_id" jsonschema:"unique event identifier"`
	ConversationID string  `json:"conversation_id" jsonschema:"required conversation identifier to group multi-turn interactions"`
	AgentName      string  `json:"agent_name,omitempty" jsonschema:"optional name of the agent being evaluated"`
	AgentVersion   string  `json:"agent_version,omitempty" jsonschema:"optional version of the agent being evaluated"`
	Query          string  `json:"user_query" jsonschema:"user's original query"`
	Answer         string  `json:"answer" jsonschema:"agent response to evaluate"`
	Context        string  `json:"context,omitempty" jsonschema:"optional context or retrieved documents"`
	ExpectedOutput string  `json:"expected_output,omitempty" jsonschema:"optional ground truth for correctness evaluation"`
	JudgeName      string  `json:"judge_name" jsonschema:"judge name: relevance, faithfulness, coherence, completeness, instruction, or correctness"`
	Threshold      float64 `json:"threshold,omitempty" jsonschema:"pass/fail threshold (0.0-1.0, default: 0.7)"`
}

// ConversationInput is the MCP tool input schema for retrieving conversation turns.
type ConversationInput struct {
	ConversationID string `json:"conversation_id" jsonschema:"conversation identifier to retrieve all turns"`
}
