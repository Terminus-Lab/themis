package sqlite

import "context"

const schema = `
  CREATE TABLE IF NOT EXISTS eval_results (
      id TEXT PRIMARY KEY,
      event_id TEXT NOT NULL,
      agent_name TEXT NOT NULL,
      agent_version TEXT NOT NULL,
      user_query TEXT NOT NULL,
      answer TEXT NOT NULL,
      context TEXT,
      confidence REAL NOT NULL,
      verdict TEXT NOT NULL,
      stage_scores TEXT NOT NULL,
      conversation_id TEXT,
      created_at DATETIME DEFAULT CURRENT_TIMESTAMP
  );

  CREATE INDEX IF NOT EXISTS idx_agent_name ON eval_results(agent_name, created_at);
  CREATE INDEX IF NOT EXISTS idx_verdict ON eval_results(verdict, created_at);
  CREATE INDEX IF NOT EXISTS idx_created_at ON eval_results(created_at);
  CREATE INDEX IF NOT EXISTS idx_conversation_id ON eval_results(conversation_id, created_at);
`

func (d *DB) InitSchema(ctx context.Context) error {
	_, err := d.client.ExecContext(ctx, schema)
	return err
}
