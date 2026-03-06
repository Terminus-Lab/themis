package llm

import "fmt"

type LLMFamily string

const (
	FamilyAnthropic LLMFamily = "anthropic"
	FamilyOpenAI    LLMFamily = "openai"
)

// LLMClientRegistry stores LLM clients organized by family and model ID
type LLMClientRegistry struct {
	clients map[LLMFamily]map[string]LLMClient // family -> modelID -> client
}

// NewLLMClientRegistry creates a new registry with the provided clients
// clients map is structured as: family -> (modelID -> client)
func NewLLMClientRegistry(clients map[LLMFamily]map[string]LLMClient) *LLMClientRegistry {
	if clients == nil {
		clients = make(map[LLMFamily]map[string]LLMClient)
	}
	return &LLMClientRegistry{
		clients: clients,
	}
}

// Get retrieves a client by family and model ID
func (r *LLMClientRegistry) Get(family LLMFamily, modelID string) (LLMClient, error) {
	familyClients, exists := r.clients[family]
	if !exists {
		return nil, fmt.Errorf("no clients registered for family: %s", family)
	}

	client, exists := familyClients[modelID]
	if !exists {
		return nil, fmt.Errorf("client not found for family %s and model %s", family, modelID)
	}

	return client, nil
}
