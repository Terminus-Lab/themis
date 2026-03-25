package mcpadapter

// EvaluateConversationInput is the MCP tool input schema for conversation evaluation.
type EvaluateConversationInput struct {
	ConversationID string          `json:"conversation_id" jsonschema:"required,unique conversation identifier"`
	AgentName      string          `json:"agent_name,omitempty" jsonschema:"optional name of the agent being evaluated"`
	AgentVersion   string          `json:"agent_version,omitempty" jsonschema:"optional version of the agent being evaluated"`
	Turns          []TurnInput     `json:"turns" jsonschema:"required,all turns in the conversation"`
}

// TurnInput is a single conversation turn in EvaluateConversationInput.
type TurnInput struct {
	TurnIndex int    `json:"turn_index"`
	UserQuery string `json:"user_query"`
	Answer    string `json:"answer"`
	Context   string `json:"context,omitempty"`
}

// ConversationInput is the MCP tool input schema for retrieving a conversation.
type ConversationInput struct {
	ConversationID string `json:"conversation_id" jsonschema:"conversation identifier to retrieve"`
}
