CREATE TABLE eval_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id TEXT NOT NULL,
    event_type TEXT NOT NULL,

    -- Agent metadata
    agent_name TEXT NOT NULL,
    agent_version TEXT NOT NULL,

    -- Evaluation inputs
    user_query TEXT NOT NULL,
    answer TEXT NOT NULL,
    context TEXT,
    expected_output TEXT,

    -- Evaluation outputs
    confidence FLOAT NOT NULL,
    verdict TEXT NOT NULL,  -- pass, review, fail

    -- Individual judge scores (JSONB for flexibility)
    stage_scores JSONB NOT NULL,

    -- Timestamps and TTL
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,

    -- Metadata
    metadata JSONB
) PARTITION BY RANGE (created_at);

-- Indexes
CREATE INDEX idx_agent_name ON eval_results (agent_name, created_at DESC);
CREATE INDEX idx_agent_version ON eval_results (agent_name, agent_version, created_at DESC);
CREATE INDEX idx_verdict ON eval_results (verdict, created_at DESC);
CREATE INDEX idx_expires ON eval_results (expires_at) WHERE expires_at > NOW();
CREATE INDEX idx_stage_scores ON eval_results USING GIN (stage_scores);

-- Example partitions (monthly)
CREATE TABLE eval_results_2026_03 PARTITION OF eval_results
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');

CREATE TABLE eval_results_2026_04 PARTITION OF eval_results
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');