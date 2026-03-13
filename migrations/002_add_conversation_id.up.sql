-- Add conversation_id column to eval_results table
ALTER TABLE eval_results ADD COLUMN conversation_id TEXT NOT NULL DEFAULT '';

-- Create index for conversation queries
CREATE INDEX idx_conversation_id ON eval_results (conversation_id, created_at DESC);
