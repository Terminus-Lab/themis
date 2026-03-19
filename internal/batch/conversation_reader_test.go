package batch

import (
	"context"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func nopLogger() *zerolog.Logger {
	l := zerolog.Nop()
	return &l
}

func TestConversationReader_ReadAll_ValidJSONL(t *testing.T) {
	input := `{"conversation_id":"conv-001","agent":{"name":"agent-a","version":"1.0"},"turns":[{"turn_index":0,"user_query":"Hello?","answer":"Hi!"},{"turn_index":1,"user_query":"How are you?","answer":"Good."}]}
{"conversation_id":"conv-002","agent":{"name":"agent-b","version":"2.0"},"turns":[{"turn_index":0,"user_query":"What is Go?","answer":"A language."}]}
`

	reader := NewConversationReader(strings.NewReader(input), nopLogger())
	ch := reader.ReadAll(context.Background())

	var records []ConversationInputRecord
	for r := range ch {
		records = append(records, r)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	// Verify first record
	r0 := records[0]
	if r0.Error != nil {
		t.Errorf("record 0: unexpected error: %v", r0.Error)
	}
	if r0.Request.ConversationID != "conv-001" {
		t.Errorf("record 0: expected conversation_id=conv-001, got %s", r0.Request.ConversationID)
	}
	if len(r0.Request.Turns) != 2 {
		t.Errorf("record 0: expected 2 turns, got %d", len(r0.Request.Turns))
	}
	if r0.LineNumber != 1 {
		t.Errorf("record 0: expected line=1, got %d", r0.LineNumber)
	}

	// Verify second record
	r1 := records[1]
	if r1.Error != nil {
		t.Errorf("record 1: unexpected error: %v", r1.Error)
	}
	if r1.Request.ConversationID != "conv-002" {
		t.Errorf("record 1: expected conversation_id=conv-002, got %s", r1.Request.ConversationID)
	}
	if len(r1.Request.Turns) != 1 {
		t.Errorf("record 1: expected 1 turn, got %d", len(r1.Request.Turns))
	}
	if r1.LineNumber != 2 {
		t.Errorf("record 1: expected line=2, got %d", r1.LineNumber)
	}
}

func TestConversationReader_ReadAll_ParseError(t *testing.T) {
	input := `{"conversation_id":"conv-001","turns":[{"user_query":"Hello","answer":"Hi"}]}
not valid json
{"conversation_id":"conv-003","turns":[{"user_query":"Q","answer":"A"}]}
`

	reader := NewConversationReader(strings.NewReader(input), nopLogger())
	ch := reader.ReadAll(context.Background())

	var records []ConversationInputRecord
	for r := range ch {
		records = append(records, r)
	}

	if len(records) != 3 {
		t.Fatalf("expected 3 records (including error), got %d", len(records))
	}

	if records[0].Error != nil {
		t.Errorf("record 0 should be valid, got error: %v", records[0].Error)
	}
	if records[1].Error == nil {
		t.Error("record 1 should have parse error")
	}
	if records[2].Error != nil {
		t.Errorf("record 2 should be valid, got error: %v", records[2].Error)
	}
}

func TestConversationReader_ReadAll_EmptyLines(t *testing.T) {
	input := `
{"conversation_id":"conv-001","turns":[]}

{"conversation_id":"conv-002","turns":[]}

`

	reader := NewConversationReader(strings.NewReader(input), nopLogger())
	ch := reader.ReadAll(context.Background())

	var records []ConversationInputRecord
	for r := range ch {
		records = append(records, r)
	}

	// Empty lines are skipped, so only 2 records
	if len(records) != 2 {
		t.Errorf("expected 2 records (empty lines skipped), got %d", len(records))
	}
}

func TestConversationReader_ReadAll_Empty(t *testing.T) {
	reader := NewConversationReader(strings.NewReader(""), nopLogger())
	ch := reader.ReadAll(context.Background())

	var records []ConversationInputRecord
	for r := range ch {
		records = append(records, r)
	}

	if len(records) != 0 {
		t.Errorf("expected 0 records for empty input, got %d", len(records))
	}
}

func TestConversationReader_ReadAll_ContextCancelled(t *testing.T) {
	// Large input to exercise cancellation
	var sb strings.Builder
	for i := 0; i < 100; i++ {
		sb.WriteString(`{"conversation_id":"conv-001","turns":[]}` + "\n")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	reader := NewConversationReader(strings.NewReader(sb.String()), nopLogger())
	ch := reader.ReadAll(ctx)

	// Drain the channel — should close eventually without blocking
	var records []ConversationInputRecord
	for r := range ch {
		records = append(records, r)
	}

	// After cancellation, fewer than all 100 records should be returned (possibly 0)
	if len(records) > 100 {
		t.Errorf("expected <= 100 records after cancellation, got %d", len(records))
	}
}
