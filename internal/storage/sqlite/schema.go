package sqlite

import "context"

const schema = `
  CREATE TABLE IF NOT EXISTS conversations (
      id TEXT PRIMARY KEY,
      conversation_id TEXT NOT NULL UNIQUE,
      agent_name TEXT NOT NULL,
      agent_version TEXT,
      turn_count INTEGER NOT NULL,
      turn_avg REAL NOT NULL,
      holistic_score REAL NOT NULL,
      holistic_reason TEXT,
      final_score REAL NOT NULL,
      verdict TEXT NOT NULL,
      turn_results TEXT NOT NULL,
      created_at DATETIME DEFAULT CURRENT_TIMESTAMP
  );

  CREATE INDEX IF NOT EXISTS idx_conversations_conversation_id ON conversations(conversation_id, created_at);
  CREATE INDEX IF NOT EXISTS idx_conversations_agent_name ON conversations(agent_name, created_at);
  CREATE INDEX IF NOT EXISTS idx_conversations_verdict ON conversations(verdict, created_at);
`

func (d *DB) InitSchema(ctx context.Context) error {
	_, err := d.client.ExecContext(ctx, schema)
	return err
}
