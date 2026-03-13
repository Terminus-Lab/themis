-- Remove conversation_id index
DROP INDEX IF EXISTS idx_conversation_id;

-- Remove conversation_id column
ALTER TABLE eval_results DROP COLUMN IF EXISTS conversation_id;
