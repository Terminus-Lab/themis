package azure

import (
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name           string
		apiKey         string
		model          string
		azureEndpoint  string
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name:          "valid configuration",
			apiKey:        "test-api-key",
			model:         "gpt-4o-mini",
			azureEndpoint: "https://test.openai.azure.com/openai/deployments/test",
			wantErr:       false,
		},
		{
			name:           "missing API key",
			apiKey:         "",
			model:          "gpt-4o-mini",
			azureEndpoint:  "https://test.openai.azure.com",
			wantErr:        true,
			expectedErrMsg: "azure OpenAI API key is required",
		},
		{
			name:           "missing model",
			apiKey:         "test-api-key",
			model:          "",
			azureEndpoint:  "https://test.openai.azure.com",
			wantErr:        true,
			expectedErrMsg: "azure OpenAI deployment name is required",
		},
		{
			name:           "missing endpoint",
			apiKey:         "test-api-key",
			model:          "gpt-4o-mini",
			azureEndpoint:  "",
			wantErr:        true,
			expectedErrMsg: "azure OpenAI endpoint is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.apiKey, tt.model, tt.azureEndpoint)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewClient() expected error, got nil")
					return
				}
				if err.Error() != tt.expectedErrMsg {
					t.Errorf("NewClient() error = %v, want %v", err.Error(), tt.expectedErrMsg)
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
			if client.ModelID != tt.model {
				t.Errorf("ModelID = %v, want %v", client.ModelID, tt.model)
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
		})
	}
}

func TestNewClient_ValidationOrder(t *testing.T) {
	// Test that validation checks happen in the right order
	_, err := NewClient("", "", "")
	if err == nil {
		t.Error("Expected error for all empty params")
	}
	// Should fail on API key first
	if err.Error() != "azure OpenAI API key is required" {
		t.Errorf("Expected API key error first, got: %v", err.Error())
	}

	_, err = NewClient("key", "", "")
	if err == nil {
		t.Error("Expected error for empty model and endpoint")
	}
	// Should fail on model second
	if err.Error() != "azure OpenAI deployment name is required" {
		t.Errorf("Expected model error second, got: %v", err.Error())
	}

	_, err = NewClient("key", "model", "")
	if err == nil {
		t.Error("Expected error for empty endpoint")
	}
	// Should fail on endpoint third
	if err.Error() != "azure OpenAI endpoint is required" {
		t.Errorf("Expected endpoint error third, got: %v", err.Error())
	}
}
