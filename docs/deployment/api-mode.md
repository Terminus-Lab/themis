# API Mode

HTTP REST server for synchronous, on-demand evaluation.

## Start

```bash
./bin/themis-api
# Expected: "INFO Starting Themis Server address=:18082"

# With Redis streaming:
EVENTS_STREAMING_ENABLED=true ./bin/themis-api
```

---

## Endpoints

### POST `/api/v1/evaluate`

Full two-stage pipeline evaluation.

```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "eval-001",
    "event_type": "agent_response",
    "agent": {"name": "my-agent", "version": "1.0"},
    "interaction": {
      "user_query": "What is the capital of France?",
      "context": "France is a country in Europe...",
      "answer": "Paris"
    }
  }'
```

Response:
```json
{
  "id": "eval-001",
  "confidence": 0.92,
  "verdict": "pass",
  "stage_scores": [...],
  "metrics": {
    "stage1_avg": 0.95,
    "stage2_weighted_avg": 0.92,
    "aggregation_method": "weighted_average"
  }
}
```

### POST `/api/v1/evaluate/judge/{name}`

Single judge evaluation (faster). Available names: `relevance`, `faithfulness`, `coherence`, `completeness`, `instruction`, `correctness`.

```bash
curl -X POST "http://localhost:18082/api/v1/evaluate/judge/relevance?threshold=0.8" \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "eval-002",
    "agent": {"name": "my-agent"},
    "interaction": {"user_query": "What is AI?", "answer": "Artificial intelligence."}
  }'
```

Query param: `threshold` — custom pass/fail threshold (default: 0.7).

### GET `/api/v1/results`

Query past evaluations.

Query params:
- `agent_name` — exact match
- `verdict` — `pass`, `review`, or `fail`
- `limit` — results per page (default: 50)
- `offset` — pagination offset (default: 0)

```bash
curl "http://localhost:18082/api/v1/results?agent_name=my-agent&verdict=fail&limit=20"
```

### GET `/api/v1/results/{event_id}`

Single evaluation by ID.

### GET `/api/v1/conversations`

List all conversations with summary metrics (turn count, avg confidence, verdict distribution).

### GET `/api/v1/conversations/{id}`

All turns for a conversation with detailed evaluations.

### POST `/api/v1/validation/sample/events/download`

Sample a percentage of individual event evaluations for human annotation. Returns JSONL with **interaction data only** — no Themis scores, so annotators are unbiased.

```bash
curl -X POST http://localhost:18082/api/v1/validation/sample/events/download \
  -H "Content-Type: application/json" \
  -d '{
    "start_date": "2026-01-01T00:00:00Z",
    "end_date": "2026-03-31T23:59:59Z",
    "percentage": 25,
    "min_size": 100,
    "max_size": 2500
  }' -o events-sample.jsonl
```

Response: `application/x-ndjson`, one event per line:
```json
{"event_id":"evt-001","conversation_id":"conv-abc","agent":{"name":"my-agent","version":"1.0"},"interaction":{"user_query":"...","answer":"...","context":"..."}}
```

**Annotation workflow:**
```bash
# 1. Download sample
curl -X POST .../api/v1/validation/sample/events/download \
  -d '{"start_date":"...","end_date":"...","percentage":25}' \
  -o events-sample.jsonl

# 2. Annotators add "human_annotation": "pass|review|fail" to each line

# 3. Validate event-level judges
./bin/themis-cli validate-events -i annotated-events.jsonl -c 0.3
```

### POST `/api/v1/evaluate/conversation`

Evaluate a full multi-turn conversation using conversation-scoped judges (`scope: conversation` in `judges.yaml`).

```bash
curl -X POST http://localhost:18082/api/v1/evaluate/conversation \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "conv-001",
    "agent": {"name": "my-agent", "version": "1.0"},
    "turns": [
      {"turn_index": 1, "user_query": "What is Python?", "answer": "A programming language."},
      {"turn_index": 2, "user_query": "Is it hard to learn?", "answer": "No, it is beginner-friendly."}
    ]
  }'
```

Response:
```json
{
  "conversation_id": "conv-001",
  "agent_name": "my-agent",
  "agent_version": "1.0",
  "turn_count": 2,
  "verdict": "pass",
  "confidence": 0.87,
  "stages": [{"name": "conversation-flow-judge", "score": 0.87, "reason": "...", "weight": 1.0}]
}
```

Returns `503` if no `scope: conversation` judges are configured in `configs/judges.yaml`.

---

### POST `/api/v1/validation/sample/conversations/download`

Sample a percentage of whole conversations for conversation-level human annotation. Picks N distinct `conversation_id`s and returns all their turns grouped — never returns a partial conversation.

```bash
curl -X POST http://localhost:18082/api/v1/validation/sample/conversations/download \
  -H "Content-Type: application/json" \
  -d '{
    "start_date": "2026-01-01T00:00:00Z",
    "end_date": "2026-03-31T23:59:59Z",
    "percentage": 25
  }' -o conversations-sample.jsonl
```

Response: `application/x-ndjson`, one conversation per line:
```json
{"conversation_id":"conv-abc","agent":{"name":"my-agent","version":"1.0"},"turns":[{"turn_index":1,"user_query":"...","answer":"..."},{"turn_index":2,"user_query":"...","answer":"..."}]}
```

Fields: same as events endpoint (`start_date`, `end_date`, `percentage`, `min_size`, `max_size`). Here `percentage` and `min/max_size` apply to the number of **conversations**, not individual events.

**Annotation workflow:**
```bash
# 1. Download conversation sample
curl -X POST .../api/v1/validation/sample/conversations/download \
  -d '{"start_date":"...","end_date":"...","percentage":25}' \
  -o conversations-sample.jsonl

# 2. Annotators add "human_annotation": "pass|review|fail" per conversation line

# 3. Validate conversation-level judges (coming soon)
./bin/themis-cli validate-conversations -i annotated-conversations.jsonl -c 0.3
```

### GET `/api/v1/metrics/health`

Production health metrics.

```bash
curl "http://localhost:18082/api/v1/metrics/health?window=7d"
```

### GET `/api/v1/health`

Server health check.

```bash
curl http://localhost:18082/api/v1/health
# {"status":"ok","version":"1.0.0"}
```

### GET `/`

Web dashboard at `http://localhost:18082`.

---

## Deployment Patterns

### Single Instance (Development)
```bash
./bin/themis-api
```

### Docker
```bash
docker build -t themis-api .
docker run -p 18082:18082 --env-file .env themis-api
```

### Load Balanced
```bash
EVAL_AGENT_API_PORT=18082 ./bin/themis-api &
EVAL_AGENT_API_PORT=18083 ./bin/themis-api &
EVAL_AGENT_API_PORT=18084 ./bin/themis-api &
# Configure Nginx/HAProxy → round-robin to 18082–18084
```

### Unified API + Streaming
```bash
EVENTS_STREAMING_ENABLED=true ./bin/themis-api
```
One process handles both HTTP requests and Redis stream consumption.

---

## Performance

**Enable early exit** — skips LLM calls for obviously low-quality responses:
```env
ENABLE_PRECHECK=true
EARLY_EXIT_THRESHOLD=0.2   # ~80% cost savings on bad answers
```

**Use lighter models** — `gpt-4o-mini` is significantly faster than `claude-3-5-sonnet` with comparable results for most use cases.

**Disable unused judges** in `configs/judges.yaml`:
```yaml
- name: instruction
  enabled: false
```

Response time is bounded by the slowest judge (~3–4s for 5 parallel judges).

---

## Security

API has **no authentication** by default — intended for internal/trusted network use.

For production:
- Deploy behind an API gateway with auth
- Use network policies (VPC, security groups)
- See [SECURITY.md](../../SECURITY.md) for full guidance

---

## Troubleshooting

| Issue | Fix |
|-------|-----|
| Port in use | Set `EVAL_AGENT_API_PORT=18083` in `.env` |
| Slow response | Check early exit is working; verify LLM provider latency |
| Database errors (SQLite) | Should work out of the box; check `IN_MEMORY_DB=true` |
| Database errors (PostgreSQL) | Run `psql "$THEMIS_DB_URL"` to verify; check migrations with `migrate ... version` |
