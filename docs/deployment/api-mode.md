# API Mode

HTTP REST server for synchronous, on-demand conversation evaluation.

## Start

```bash
go run cmd/api/main.go
# Expected: "INFO Starting Themis Server address=:18082"

# With Redis conversation streaming:
CONVERSATION_STREAMING_ENABLED=true go run cmd/api/main.go
```

---

## Endpoints

### POST `/api/v1/conversations/evaluate`

Evaluate a full conversation (one or more turns) using the two-phase pipeline.

```bash
curl -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "conv-001",
    "agent": {"name": "my-agent", "version": "1.0"},
    "turns": [
      {
        "turn_index": 1,
        "user_query": "What is the capital of France?",
        "answer": "The capital of France is Paris."
      },
      {
        "turn_index": 2,
        "user_query": "And Germany?",
        "answer": "The capital of Germany is Berlin."
      }
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
  "turn_results": [
    {
      "turn_index": 1,
      "user_query": "What is the capital of France?",
      "answer": "The capital of France is Paris.",
      "turn_score": 0.93,
      "scores": [
        {"name": "relevance", "score": 0.95, "reason": "..."},
        {"name": "coherence", "score": 0.91, "reason": "..."},
        {"name": "completeness", "score": 0.93, "reason": "..."}
      ]
    }
  ],
  "turn_avg": 0.92,
  "holistic_score": 0.89,
  "holistic_reason": "The conversation flows naturally...",
  "final_score": 0.905,
  "verdict": "pass"
}
```

### GET `/api/v1/conversations`

List all evaluated conversations with summary metrics.

```bash
curl "http://localhost:18082/api/v1/conversations"
```

Response:
```json
{
  "conversations": [
    {
      "conversation_id": "conv-001",
      "agent_name": "my-agent",
      "agent_version": "1.0",
      "turn_count": 2,
      "final_score": 0.905,
      "verdict": "pass",
      "created_at": "2026-03-24T10:00:00Z"
    }
  ],
  "total": 1
}
```

### GET `/api/v1/conversations/{conversation_id}`

Get full evaluation details for a conversation, including per-turn scores.

```bash
curl "http://localhost:18082/api/v1/conversations/conv-001"
```

Returns `404` if not found.

### GET `/api/v1/metrics/health`

Production health metrics over a time window.

```bash
curl "http://localhost:18082/api/v1/metrics/health?window=7d"
```

Supported window formats: `1d`, `7d`, `30d`, `24h`, etc.

Response:
```json
{
  "window": "7d",
  "total_evaluations": 42,
  "avg_confidence": 0.81
}
```

### GET `/api/v1/health`

Server health check.

```bash
curl http://localhost:18082/api/v1/health
# {"status":"ok"}
```

### GET `/`

Web dashboard at `http://localhost:18082`.

---

## Deployment Patterns

### Development

```bash
go run cmd/api/main.go
```

### Production Binary

```bash
go build -o bin/themis-api cmd/api/main.go
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

### API + Conversation Streaming

```bash
CONVERSATION_STREAMING_ENABLED=true ./bin/themis-api
```

One process handles both HTTP requests and Redis stream consumption.

---

## Performance

**Use lighter models** — `gpt-4o-mini` is significantly faster than `claude-3-5-sonnet` with comparable results for most use cases.

**Disable unused judges** in `configs/judges.yaml`:
```yaml
- name: coherence
  enabled: false
```

Response time is bounded by the slowest judge (~3–4s for 3 parallel per-turn judges + holistic judge).

---

## Security

API has **no authentication** by default — intended for internal/trusted network use.

For production:
- Deploy behind an API gateway with auth
- Use network policies (VPC, security groups)

---

## Troubleshooting

| Issue | Fix |
|-------|-----|
| Port in use | Set `EVAL_AGENT_API_PORT=18083` in `.env` |
| Slow response | Check LLM provider latency; try a faster model |
| Database errors (SQLite) | Check `IN_MEMORY_DB=true` |
| Database errors (PostgreSQL) | Run `psql "$THEMIS_DB_URL"` to verify; check migrations |
