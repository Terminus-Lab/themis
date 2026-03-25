package ollama

import (
	"context"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		modelID string
	}{
		{
			name:    "valid configuration",
			baseURL: "http://localhost:11434/v1",
			modelID: "qwen2.5:7b",
		},
		{
			name:    "custom base URL",
			baseURL: "http://192.168.1.100:11434/v1",
			modelID: "llama3.1:8b",
		},
		{
			name:    "empty base URL falls back to default",
			baseURL: "",
			modelID: "qwen2.5:7b",
		},
		{
			name:    "empty model ID still creates client",
			baseURL: "http://localhost:11434/v1",
			modelID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(context.Background(), tt.baseURL, tt.modelID)

			if err != nil {
				t.Errorf("NewClient() unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Fatal("NewClient() returned nil client")
			}

			if client.ModelID != tt.modelID {
				t.Errorf("ModelID = %v, want %v", client.ModelID, tt.modelID)
			}

			if client.MaxRetries != 3 {
				t.Errorf("MaxRetries = %v, want 3", client.MaxRetries)
			}

			if client.InitialDelay != 100*time.Millisecond {
				t.Errorf("InitialDelay = %v, want 100ms", client.InitialDelay)
			}

			if client.MaxDelay != 12*time.Second {
				t.Errorf("MaxDelay = %v, want 12s", client.MaxDelay)
			}

			if client.Client == nil {
				t.Error("Client.Client (OpenAI client) is nil")
			}
		})
	}
}

func TestNewClient_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client, err := NewClient(ctx, "http://localhost:11434/v1", "qwen2.5:7b")
	if err != nil {
		t.Errorf("NewClient() with cancelled context should still succeed, got: %v", err)
	}
	if client == nil {
		t.Error("NewClient() returned nil client with cancelled context")
	}
}
