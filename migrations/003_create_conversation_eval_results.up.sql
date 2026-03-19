CREATE TABLE conversation_eval_results (
    id UUID PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    agent_name TEXT NOT NULL,
    agent_version TEXT NOT NULL,
    turn_count INTEGER NOT NULL,
    confidence FLOAT NOT NULL,
    verdict TEXT NOT NULL,
    stage_scores JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_conv_eval_conversation_id ON conversation_eval_results (conversation_id, created_at DESC);
