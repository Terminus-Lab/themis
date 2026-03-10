CREATE EXTENSION IF NOT EXISTS "uuid-ossp";  -- For UUID generation

CREATE TABLE eval_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id TEXT NOT NULL,

    -- Agent metadata
    agent_name TEXT NOT NULL,
    agent_version TEXT NOT NULL,

    -- Evaluation inputs
    user_query TEXT NOT NULL,
    answer TEXT NOT NULL,
    context TEXT,

    -- Evaluation outputs
    confidence FLOAT NOT NULL,
    verdict TEXT NOT NULL,  -- pass, review, fail

    -- Individual judge scores (JSONB for flexibility)
    stage_scores JSONB NOT NULL,

    -- Timestamps and TTL
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_agent_name ON eval_results (agent_name, created_at DESC);
CREATE INDEX idx_verdict ON eval_results (verdict, created_at DESC);
CREATE INDEX idx_created_at ON eval_results (created_at DESC);