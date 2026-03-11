package openaiplatform

import (
	"context"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		modelID  string
		wantErr  bool
	}{
		{
			name:    "valid configuration",
			apiKey:  "sk-proj-test-key",
			modelID: "gpt-4o-mini",
			wantErr: false,
		},
		{
			name:    "empty API key still creates client",
			apiKey:  "",
			modelID: "gpt-4o-mini",
			wantErr: false,
		},
		{
			name:    "empty model ID still creates client",
			apiKey:  "sk-proj-test-key",
			modelID: "",
			wantErr: false,
		},
		{
			name:    "both empty still creates client",
			apiKey:  "",
			modelID: "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client, err := NewClient(ctx, tt.apiKey, tt.modelID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewClient() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewClient() unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Error("NewClient() returned nil client")
				return
			}

			// Verify client configuration
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

func TestNewClient_ContextCancellation(t *testing.T) {
	// Test that cancelled context doesn't prevent client creation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	client, err := NewClient(ctx, "sk-test", "gpt-4o-mini")
	if err != nil {
		t.Errorf("NewClient() with cancelled context should still succeed, got error: %v", err)
	}

	if client == nil {
		t.Error("NewClient() returned nil client even with cancelled context")
	}
}
