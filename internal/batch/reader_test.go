package batch

import (
	"context"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func newTestLogger() *zerolog.Logger {
	logger := zerolog.Nop()
	return &logger
}

func TestReader_InvalidFile(t *testing.T) {
	file := strings.NewReader("invalid file content")

	reader := NewReader(file, newTestLogger())
	ctx := context.Background()
	ch := reader.ReadAll(ctx)

	for record := range ch {
		if record.Error == nil {
			t.Errorf("expected parse error for invalid JSON, but got none")
		}
	}
}

func TestReader_ValidFile(t *testing.T) {
	inputFile := `{"event_id":"1","event_type":"agent_response","agent":{"name":"test","type":"rag","version":"1.0"},"interaction":{"user_query":"test","context":"","answer":"response"}}
  {"event_id":"2","event_type":"agent_response","agent":{"name":"test","type":"rag","version":"1.0"},"interaction":{"user_query":"test2","context":"","answer":"response2"}}`

	file := strings.NewReader(inputFile)

	ctx := context.Background()
	reader := NewReader(file, newTestLogger())

	ch := reader.ReadAll(ctx)
	count := 0
	for record := range ch {
		count += 1
		if record.Error != nil {
			t.Errorf("Error reading the evaluation request record. Got: %s", record.Error)
		}
	}
	if count != 2 {
		t.Errorf("Expected 2 evaluation request messages. Got: %d", count)
	}
}

func TestReader_WithConversationID(t *testing.T) {
	inputFile := `{"event_id":"turn-1","conversation_id":"conv-123","event_type":"agent_response","agent":{"name":"test","type":"rag","version":"1.0"},"interaction":{"user_query":"What is AI?","context":"","answer":"AI is Artificial Intelligence"}}
{"event_id":"turn-2","conversation_id":"conv-123","event_type":"agent_response","agent":{"name":"test","type":"rag","version":"1.0"},"interaction":{"user_query":"How does it work?","context":"","answer":"It uses machine learning algorithms"}}`

	file := strings.NewReader(inputFile)
	ctx := context.Background()
	reader := NewReader(file, newTestLogger())

	ch := reader.ReadAll(ctx)
	records := []InputRecord{}
	for record := range ch {
		records = append(records, record)
		if record.Error != nil {
			t.Errorf("Error reading record: %s", record.Error)
		}
	}

	if len(records) != 2 {
		t.Errorf("Expected 2 records, got %d", len(records))
	}

	// Verify conversation_id is parsed correctly
	for _, record := range records {
		if record.Request.ConversationID != "conv-123" {
			t.Errorf("Expected conversation_id='conv-123', got '%s'", record.Request.ConversationID)
		}
	}

	// Verify event IDs are different but conversation ID is same
	if records[0].Request.EventID != "turn-1" {
		t.Errorf("Expected event_id='turn-1', got '%s'", records[0].Request.EventID)
	}
	if records[1].Request.EventID != "turn-2" {
		t.Errorf("Expected event_id='turn-2', got '%s'", records[1].Request.EventID)
	}
}

func TestReader_ContextCancellation(t *testing.T) {
	// Large input with many lines
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines,
			`{"event_id":"1","event_type":"agent_response","agent":{"name":"test","type":"rag","version":"1.0"},"interaction":{"user_query":"test","context":"","answer":"response"}}`)
	}
	file := strings.NewReader(strings.Join(lines, "\n"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reader := NewReader(file, newTestLogger())

	ch := reader.ReadAll(ctx)
	count := 0
	for range ch {
		count++
		if count == 5 {
			cancel() // Cancel after 5 records
			break
		}
	}

	// Should have stopped early
	if count >= 100 {
		t.Errorf("expected early cancellation, but read all records")
	}
}

func TestReader_LineNumbers(t *testing.T) {
	inputFile := `{"event_id":"1","event_type":"agent_response","agent":{"name":"test","type":"rag","version":"1.0"},"interaction":{"user_query":"test","context":"","answer":"response"}}

{"invalid json}
{"event_id":"2","event_type":"agent_response","agent":{"name":"test","type":"rag","version":"1.0"},"interaction":{"user_query":"test2","context":"","answer":"response2"}}`

	file := strings.NewReader(inputFile)
	reader := NewReader(file, newTestLogger())

	ch := reader.ReadAll(context.Background())
	records := []InputRecord{}
	for record := range ch {
		records = append(records, record)
	}

	// Check line numbers
	if records[0].LineNumber != 1 {
		t.Errorf("first record should be line 1, got %d", records[0].LineNumber)
	}
	if records[1].LineNumber != 3 {
		t.Errorf("error record should be line 3, got %d", records[1].LineNumber)
	}
	if records[2].LineNumber != 4 {
		t.Errorf("third record should be line 4, got %d", records[2].LineNumber)
	}
}
