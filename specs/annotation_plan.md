## Annotation

## Overview

Add **persistent storage**, **query API**, and **annotation UI** to enable human-in-the-loop validation and continuous judge calibration.

### Goals

1. **Persist evaluation results** from API/streaming modes to PostgreSQL
2. **Query API** to retrieve results with filters (agent, judge, date)
3. **Annotation UI** (Streamlit) to collect human judgments
4. **Feedback loop** to validate judge accuracy against human annotations
5. **Automated TTL** to expire old results

---

## Architecture

```
┌─────────────────┐
│   API Request   │
│  /api/v1/eval   │
└────────┬────────┘
         │
         v
┌─────────────────┐
│  Eval Pipeline  │
│ (judges + agg)  │
└────────┬────────┘
         │
         ├─► Return to client
         │
         v
┌─────────────────┐
│   PostgreSQL    │
│  eval_results   │
│  (partitioned)  │
└────────┬────────┘
         │
         v
┌─────────────────┐     ┌──────────────────┐
│   Query API     │────►│  Streamlit UI    │
│ GET /results    │     │  (annotation)    │
└─────────────────┘     └──────────────────┘
                               │
                               v
                        ┌──────────────────┐
                        │ human_annotations│
                        │    (postgres)    │
                        └──────────────────┘
```

---

### 1. Database Schema

**Technology**: PostgreSQL with partitioning (for TTL)

#### Table: `eval_results`

```sql
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
    stage_scores JSONB NOT NULL,  -- [{"name": "relevance-judge", "score": 0.9, "reason": "..."}]

    -- Timestamps and TTL
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,  -- created_at + retention_period

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
```

#### Table: `human_annotations`

```sql
CREATE TABLE human_annotations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    eval_result_id UUID NOT NULL REFERENCES eval_results(id) ON DELETE CASCADE,

    -- Human judgment
    human_verdict TEXT NOT NULL,  -- pass, review, fail
    human_confidence FLOAT,  -- optional 0.0-1.0
    human_notes TEXT,

    -- Annotator metadata
    annotator_id TEXT NOT NULL,
    annotated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Quality checks
    annotation_time_seconds INTEGER,  -- how long they spent

    UNIQUE(eval_result_id, annotator_id)  -- one annotation per evaluator
);

CREATE INDEX idx_eval_result ON human_annotations (eval_result_id);
CREATE INDEX idx_annotator ON human_annotations (annotator_id, annotated_at DESC);
```

**Design decisions**:
- **Partitioning by `created_at`**: Enables efficient TTL by dropping old partitions
- **JSONB for stage_scores**: Flexible schema as judges evolve
- **No separate judge results table**: Denormalized for query performance
- **Cascading deletes**: Annotations deleted when eval results expire

**Retention policy**: Configurable (default 90 days)

---

### 3. Partition Cleanup Strategy

**Recommended: PostgreSQL pg_cron extension**

**Why pg_cron?**
- ✅ Built into managed Postgres (AWS RDS, Azure, Google Cloud SQL)
- ✅ No external orchestration needed
- ✅ Runs inside database (no app code needed)
- ✅ Atomic operations (no race conditions)

**Setup**:

```sql
-- Enable extension (one-time)
CREATE EXTENSION pg_cron;

-- Create cleanup function
CREATE OR REPLACE FUNCTION drop_expired_partitions()
RETURNS void AS $$
DECLARE
    partition_name TEXT;
    drop_before TIMESTAMP;
BEGIN
    -- Drop partitions older than 90 days
    drop_before := NOW() - INTERVAL '90 days';

    FOR partition_name IN
        SELECT tablename
        FROM pg_tables
        WHERE schemaname = 'public'
        AND tablename LIKE 'eval_results_%'
        AND tablename::text < 'eval_results_' || TO_CHAR(drop_before, 'YYYY_MM')
    LOOP
        EXECUTE 'DROP TABLE IF EXISTS ' || partition_name || ' CASCADE';
        RAISE NOTICE 'Dropped partition: %', partition_name;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Schedule daily at 2 AM UTC
SELECT cron.schedule(
    'drop-old-eval-partitions',
    '0 2 * * *',
    'SELECT drop_expired_partitions();'
);
```

**Alternative: Kubernetes CronJob** (if not using managed Postgres)

```yaml
# k8s/partition-cleanup-cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: themis-partition-cleanup
spec:
  schedule: "0 2 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: cleanup
            image: terminus-lab/themis:latest
            command: ["/app/themis", "cleanup", "--partitions"]
            env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: themis-db
                  key: url
          restartPolicy: OnFailure
```

**NOT recommended: Goroutine in API app**
- ❌ Multiple API instances = multiple cleanup jobs (wasteful)
- ❌ Requires distributed locks to prevent race conditions
- ❌ Mixing maintenance with serving traffic (poor separation of concerns)

---

### 4. Storage Layer Implementation

**File**: `internal/storage/postgres/results.go`

```go
package postgres

import (
    "context"
    "time"
    "github.com/jackc/pgx/v5/pgxpool"
    "themis/internal/models"
)

type ResultsRepository struct {
    pool *pgxpool.Pool
    retentionDays int
}

func NewResultsRepository(pool *pgxpool.Pool, retentionDays int) *ResultsRepository {
    return &ResultsRepository{
        pool: pool,
        retentionDays: retentionDays,
    }
}

// Store saves evaluation result to database
func (r *ResultsRepository) Store(ctx context.Context, result *models.EvaluationResult) error {
    query := `
        INSERT INTO eval_results (
            event_id, event_type, agent_name, agent_version,
            user_query, answer, context, expected_output,
            confidence, verdict, stage_scores,
            created_at, expires_at, metadata
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
        )
    `

    expiresAt := time.Now().AddDate(0, 0, r.retentionDays)

    _, err := r.pool.Exec(ctx, query,
        result.ID, result.EventType,
        result.Agent.Name, result.Agent.Version,
        result.Interaction.Query, result.Interaction.Answer,
        result.Interaction.Context, result.Interaction.ExpectedOutput,
        result.Confidence, result.Verdict,
        result.Stages, // JSONB
        result.CreatedAt, expiresAt,
        result.Metadata,
    )

    return err
}

// Query retrieves results with filters
func (r *ResultsRepository) Query(ctx context.Context, filters QueryFilters) ([]models.EvaluationResult, error) {
    // Implementation in next section
}
```

**Integration point**: Update API handler to persist after evaluation

```go
// internal/api/handler.go
func (h *Handler) Evaluate(c *gin.Context) {
    // ... existing evaluation logic ...

    result := h.executor.Execute(ctx, req)

    // NEW: Persist to database
    if err := h.storage.Store(ctx, result); err != nil {
        log.Error("Failed to store result", "error", err)
        // Don't fail the request - storage is async concern
    }

    c.JSON(200, result)
}
```

**Configuration**: Add to `.env`

```env
# Storage
DATABASE_URL=postgres://user:pass@localhost:5432/themis
STORAGE_ENABLED=true
RETENTION_DAYS=90
```

---

### 5. Query API

**Endpoint**: `GET /api/v1/results`

**Query parameters**:
- `agent_name` - Filter by agent name
- `agent_version` - Filter by specific version
- `verdict` - Filter by verdict (pass, review, fail)
- `from_date` - Start date (ISO 8601)
- `to_date` - End date (ISO 8601)
- `limit` - Results per page (default: 50, max: 500)
- `offset` - Pagination offset
- `order` - Sort order (created_at_desc, created_at_asc, confidence_desc)

**Example requests**:

```bash
# All results for a specific agent
curl "http://localhost:18082/api/v1/results?agent_name=rag-agent&limit=100"

# Failed evaluations in last 7 days
curl "http://localhost:18082/api/v1/results?verdict=fail&from_date=2026-02-27"

# Specific agent version, ordered by confidence
curl "http://localhost:18082/api/v1/results?agent_name=rag-agent&agent_version=1.2.0&order=confidence_desc"
```

**Response format**:

```json
{
  "results": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "event_id": "evt-123",
      "agent": {
        "name": "rag-agent",
        "version": "1.2.0"
      },
      "interaction": {
        "user_query": "What is the capital of France?",
        "answer": "Paris is the capital of France.",
        "context": "France is a country in Europe."
      },
      "confidence": 0.95,
      "verdict": "pass",
      "stages": [
        {"name": "relevance-judge", "score": 0.9, "reason": "Directly answers query"},
        {"name": "faithfulness-judge", "score": 1.0, "reason": "Grounded in context"}
      ],
      "created_at": "2026-03-06T10:30:00Z"
    }
  ],
  "pagination": {
    "total": 1523,
    "limit": 50,
    "offset": 0,
    "has_more": true
  }
}
```

**Implementation**:

```go
// internal/storage/postgres/query.go
type QueryFilters struct {
    AgentName    string
    AgentVersion string
    Verdict      string
    FromDate     time.Time
    ToDate       time.Time
    Limit        int
    Offset       int
    OrderBy      string
}

func (r *ResultsRepository) Query(ctx context.Context, filters QueryFilters) (*QueryResult, error) {
    query := `
        SELECT
            id, event_id, event_type,
            agent_name, agent_version,
            user_query, answer, context, expected_output,
            confidence, verdict, stage_scores,
            created_at
        FROM eval_results
        WHERE 1=1
    `

    args := []interface{}{}
    argIdx := 1

    if filters.AgentName != "" {
        query += fmt.Sprintf(" AND agent_name = $%d", argIdx)
        args = append(args, filters.AgentName)
        argIdx++
    }

    if filters.AgentVersion != "" {
        query += fmt.Sprintf(" AND agent_version = $%d", argIdx)
        args = append(args, filters.AgentVersion)
        argIdx++
    }

    if filters.Verdict != "" {
        query += fmt.Sprintf(" AND verdict = $%d", argIdx)
        args = append(args, filters.Verdict)
        argIdx++
    }

    if !filters.FromDate.IsZero() {
        query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
        args = append(args, filters.FromDate)
        argIdx++
    }

    if !filters.ToDate.IsZero() {
        query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
        args = append(args, filters.ToDate)
        argIdx++
    }

    // Order by
    switch filters.OrderBy {
    case "created_at_asc":
        query += " ORDER BY created_at ASC"
    case "confidence_desc":
        query += " ORDER BY confidence DESC, created_at DESC"
    default:
        query += " ORDER BY created_at DESC"
    }

    // Pagination
    query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
    args = append(args, filters.Limit, filters.Offset)

    // Execute query...
}
```

**API handler**:

```go
// internal/api/handler.go
func (h *Handler) QueryResults(c *gin.Context) {
    filters := QueryFilters{
        AgentName:    c.Query("agent_name"),
        AgentVersion: c.Query("agent_version"),
        Verdict:      c.Query("verdict"),
        Limit:        getIntQuery(c, "limit", 50, 500),
        Offset:       getIntQuery(c, "offset", 0, 0),
        OrderBy:      c.Query("order"),
    }

    if fromDate := c.Query("from_date"); fromDate != "" {
        filters.FromDate, _ = time.Parse(time.RFC3339, fromDate)
    }

    if toDate := c.Query("to_date"); toDate != "" {
        filters.ToDate, _ = time.Parse(time.RFC3339, toDate)
    }

    result, err := h.storage.Query(c.Request.Context(), filters)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to query results"})
        return
    }

    c.JSON(200, result)
}
```

**Route registration**:

```go
// internal/api/routes.go
func RegisterRoutes(r *gin.Engine, handler *Handler) {
    api := r.Group("/api/v1")

    // Existing
    api.POST("/evaluate", handler.Evaluate)
    api.POST("/evaluate/judge/:judge_name", handler.EvaluateSingleJudge)

    // NEW
    api.GET("/results", handler.QueryResults)
    api.GET("/results/:id", handler.GetResult)
}
```

---

### 6. Annotation UI (Streamlit)

**File structure**:

```
themis/
  ui/
    annotate.py          # Main Streamlit app
    components/
      result_card.py     # Display evaluation result
      annotation_form.py # Collect human judgment
    database.py          # DB connection
    requirements.txt
```

**Main app**: `ui/annotate.py`

```python
import streamlit as st
import psycopg2
from datetime import datetime, timedelta

st.set_page_config(page_title="Themis Annotation", layout="wide")

# Database connection
@st.cache_resource
def get_connection():
    return psycopg2.connect(st.secrets["database_url"])

# Sidebar filters
st.sidebar.header("Filters")
agent_name = st.sidebar.text_input("Agent Name")
verdict = st.sidebar.selectbox("Verdict", ["All", "pass", "review", "fail"])
days_back = st.sidebar.slider("Days back", 1, 30, 7)

# Query results
from_date = datetime.now() - timedelta(days=days_back)
query = """
    SELECT r.id, r.event_id, r.agent_name, r.agent_version,
           r.user_query, r.answer, r.context,
           r.confidence, r.verdict, r.stage_scores,
           r.created_at,
           CASE WHEN h.id IS NOT NULL THEN true ELSE false END as annotated
    FROM eval_results r
    LEFT JOIN human_annotations h ON r.id = h.eval_result_id
    WHERE r.created_at >= %s
"""

params = [from_date]
if agent_name:
    query += " AND r.agent_name = %s"
    params.append(agent_name)
if verdict != "All":
    query += " AND r.verdict = %s"
    params.append(verdict)

query += " ORDER BY r.created_at DESC LIMIT 50"

conn = get_connection()
results = conn.cursor().execute(query, params).fetchall()

# Display results
st.title("Themis Evaluation Annotation")
st.write(f"Found {len(results)} results")

for row in results:
    result_id, event_id, agent_name, agent_version, query_text, answer, context, \
        confidence, verdict, stage_scores, created_at, annotated = row

    with st.expander(f"{event_id} - {verdict.upper()} ({confidence:.2f})" +
                     (" ✅ Annotated" if annotated else "")):

        # Display result
        col1, col2 = st.columns([2, 1])

        with col1:
            st.subheader("Query")
            st.write(query_text)

            st.subheader("Answer")
            st.write(answer)

            if context:
                st.subheader("Context")
                st.text(context[:500] + "..." if len(context) > 500 else context)

        with col2:
            st.metric("Confidence", f"{confidence:.2f}")
            st.metric("Verdict", verdict)
            st.metric("Agent", f"{agent_name} v{agent_version}")

            st.subheader("Judge Scores")
            for stage in stage_scores:
                st.write(f"**{stage['name']}**: {stage['score']:.2f}")
                st.caption(stage.get('reason', ''))

        # Annotation form
        if not annotated:
            st.divider()
            st.subheader("Your Annotation")

            human_verdict = st.radio(
                "Your verdict",
                ["pass", "review", "fail"],
                key=f"verdict_{result_id}",
                horizontal=True
            )

            human_confidence = st.slider(
                "Your confidence",
                0.0, 1.0, 0.5, 0.1,
                key=f"confidence_{result_id}"
            )

            human_notes = st.text_area(
                "Notes (optional)",
                key=f"notes_{result_id}"
            )

            if st.button("Submit Annotation", key=f"submit_{result_id}"):
                # Save annotation
                insert_query = """
                    INSERT INTO human_annotations
                    (eval_result_id, human_verdict, human_confidence, human_notes, annotator_id)
                    VALUES (%s, %s, %s, %s, %s)
                """
                annotator_id = st.session_state.get('annotator_id', 'default')

                conn.cursor().execute(insert_query, (
                    result_id, human_verdict, human_confidence, human_notes, annotator_id
                ))
                conn.commit()

                st.success("Annotation saved!")
                st.rerun()
        else:
            st.info("Already annotated")
```

**Configuration**: `ui/.streamlit/secrets.toml`

```toml
database_url = "postgresql://user:pass@localhost:5432/themis"
```

**Run**:

```bash
cd themis/ui
pip install -r requirements.txt
streamlit run annotate.py
```

**Features**:
- ✅ Filter by agent, verdict, date range
- ✅ Display all evaluation details (query, answer, context, scores)
- ✅ Collect human verdict + confidence + notes
- ✅ Mark already-annotated results
- ✅ Pagination support

---

### 7. Feedback Loop: Validation Against Annotations

**CLI command**: `themis validate --annotated`

```bash
# Validate judges against all human annotations
go run cmd/validate/main.go \
  --annotated \
  --threshold 0.3 \
  --output validation_report.json
```

**What it does**:

1. Query all records with human annotations:
```sql
SELECT r.id, r.confidence, r.verdict, h.human_verdict
FROM eval_results r
JOIN human_annotations h ON r.id = h.eval_result_id
```

2. Compute Kendall's τ between system verdicts and human verdicts

3. Generate report:
```json
{
  "total_annotated": 156,
  "agreement_count": 132,
  "agreement_rate": 0.846,
  "kendall_tau": 0.68,
  "threshold": 0.3,
  "passed": true,
  "confusion_matrix": {
    "pass_pass": 85,
    "pass_review": 8,
    "pass_fail": 2,
    "review_review": 22,
    "review_fail": 7,
    "fail_fail": 32
  },
  "interpretation": "Strong agreement"
}
```

4. If τ < 0.3, prompts user to update judge prompts in `configs/judges.yaml`

**Implementation**: Reuse existing batch validation logic in `internal/batch/validator.go`, but fetch data from DB instead of JSONL.

---

## Implementation Phases

### Phase 2A: Storage Layer (Week 1)
- [ ] Database schema and migrations
- [ ] `internal/storage/postgres` package
- [ ] Integration with API handler (persist after eval)
- [ ] pg_cron setup for partition cleanup
- [ ] Configuration (DATABASE_URL, RETENTION_DAYS)
- [ ] Unit tests for storage layer

### Phase 2B: Query API (Week 2)
- [ ] `GET /api/v1/results` endpoint
- [ ] `GET /api/v1/results/:id` endpoint
- [ ] Query filtering logic
- [ ] Pagination support
- [ ] Integration tests
- [ ] API documentation update

### Phase 2C: Annotation UI (Week 3)
- [ ] Streamlit app structure
- [ ] Result browsing with filters
- [ ] Annotation form and submission
- [ ] Database integration (read results, write annotations)
- [ ] Session management (annotator tracking)
- [ ] Deployment guide

### Phase 2D: Validation Against Annotations (Week 4)
- [ ] CLI command: `themis validate --annotated`
- [ ] Query annotated results from DB
- [ ] Compute Kendall's τ correlation
- [ ] Generate validation report
- [ ] Integration with existing validation logic
- [ ] Documentation update

### Phase 2E: Documentation & Rename (Week 5)
- [ ] Repository rename: eval-agent → themis
- [ ] README rewrite emphasizing purpose
- [ ] Architecture diagrams
- [ ] Deployment guide (DB setup, UI deployment)
- [ ] Migration guide for existing users
- [ ] Update all code references

---

## Configuration Changes

### New Environment Variables

```env
# Storage (Phase 2A)
DATABASE_URL=postgres://user:pass@localhost:5432/themis
STORAGE_ENABLED=true
RETENTION_DAYS=90

# UI (Phase 2C)
UI_ENABLED=true
UI_PORT=8501
ANNOTATOR_ID=default  # Can be overridden per session

# Validation (Phase 2D)
MIN_ANNOTATIONS=25  # Minimum annotations before validation
```

---

## Success Metrics

**Phase 2A (Storage)**:
- ✅ 100% of API evaluations persisted to DB
- ✅ Partitions created automatically each month
- ✅ Old partitions dropped per retention policy
- ✅ Query performance < 100ms for filtered queries

**Phase 2B (Query API)**:
- ✅ All filters working correctly
- ✅ Pagination handles large result sets (10k+ records)
- ✅ API response time < 200ms (p95)

**Phase 2C (Annotation UI)**:
- ✅ Annotators can review 50+ results per hour
- ✅ Annotations persist correctly
- ✅ UI loads in < 2 seconds

**Phase 2D (Validation)**:
- ✅ Kendall's τ computed correctly
- ✅ Validation report generated
- ✅ Clear guidance when τ < 0.3

**Phase 2E (Documentation)**:
- ✅ README clearly explains Themis purpose
- ✅ Migration guide for existing users
- ✅ All code references updated

---

## Open Questions

1. **Multi-tenancy**: Do we need to isolate results per organization/team?
2. **Annotation workflow**: Should we support multiple annotators per result (consensus)?
3. **Export format**: Do we need to export annotations to Label Studio/Argilla?
4. **Real-time metrics**: Should we add Prometheus/Grafana dashboards for evaluation metrics?
5. **Batch re-evaluation**: Should we support re-running judges on historical data after prompt updates?

---

## Next Steps

1. **Review this plan** - Confirm architecture and priorities
2. **Set up development database** - PostgreSQL with test data
3. **Start Phase 2A** - Storage layer implementation
4. **Iterate** - Adjust based on findings

---

**Document Version**: 1.0
**Last Updated**: 2026-03-06
**Owner**: @povarna
