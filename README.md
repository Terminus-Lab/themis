# Themis

AI agent conversation evaluation framework using configurable LLM judges.

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://go.dev/)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/Terminus-Lab/themis)

## Overview

Themis evaluates multi-turn AI agent conversations through a two-phase pipeline:

1. **Phase A — Per-turn judges** (`relevance`, `coherence`, `completeness`): each turn is scored independently, averaged into `turn_avg`
2. **Phase B — Holistic judge** (`conversation-flow`): the full conversation is evaluated for context-awareness and flow, producing `holistic_score`
3. **Final score**: `final_score = α × holistic_score + (1−α) × turn_avg` (α = `CONVERSATION_HOLISTIC_WEIGHT`, default 0.5)
4. **Verdict**: `pass` / `review` / `fail` based on configurable thresholds

Everything is a **conversation** — single-turn requests are represented as conversations with one turn.

## Build

```bash
# API server (HTTP endpoints + web dashboard)
go build -o bin/themis-api cmd/api/main.go

# CLI (batch evaluation)
go build -o bin/themis-cli cmd/batch/main.go

# MCP server (Claude Code / Claude Desktop integration)
go build -o bin/themis-mcp cmd/mcp/main.go
```

## Getting Started

```bash
# Clone and configure
cp .env.example .env
# Edit .env with your LLM provider credentials (see Configuration below)

# Start the API server
./bin/themis-api
# Open http://localhost:18082 for the web dashboard
```

## API

**Base URL**: `http://localhost:18082`

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/conversations/evaluate` | Evaluate a multi-turn conversation |
| `GET` | `/api/v1/conversations` | List all evaluated conversations |
| `GET` | `/api/v1/conversations/{id}` | Get conversation with turn-level detail |
| `GET` | `/api/v1/metrics/health?window=7d` | Health metrics (window: 1d, 7d, 30d) |
| `GET` | `/api/v1/health` | Service health check |
| `GET` | `/` | Web dashboard |

**Evaluate a conversation:**

```bash
curl -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "conv-001",
    "agent": {"name": "my-agent", "version": "1.0"},
    "turns": [
      {"turn_index": 0, "user_query": "What is AI?", "answer": "AI stands for Artificial Intelligence..."},
      {"turn_index": 1, "user_query": "Can you give an example?", "answer": "A common example is image recognition..."}
    ]
  }'
```

**Response:**

```json
{
  "conversation_id": "conv-001",
  "turn_avg": 0.87,
  "holistic_score": 0.91,
  "final_score": 0.89,
  "verdict": "pass",
  "holistic_reason": "The conversation flows naturally...",
  "turn_results": [...]
}
```

## CLI Batch Evaluation

```bash
# Evaluate a JSONL file
./bin/themis-cli evaluate -i resources/conversations.jsonl -o results.jsonl

# Summary output (no per-conversation JSONL)
./bin/themis-cli evaluate -i resources/conversations.jsonl -f summary

# Summary + separate results file
./bin/themis-cli evaluate -i resources/conversations.jsonl -o results.jsonl -s summary.json

# Scale workers (default: 5)
THEMIS_BATCH_WORKERS=10 ./bin/themis-cli evaluate -i dataset.jsonl -o results.jsonl
```

**Input format** (`conversations.jsonl`):

```json
{"conversation_id":"conv-001","agent":{"name":"my-agent","version":"1.0"},"turns":[{"turn_index":0,"user_query":"...","answer":"..."}]}
```

**Human annotation** — add `human_label` and/or `human_score` to get automatic correlation analysis:

```json
{"conversation_id":"conv-001","human_label":"pass","human_score":0.91,"agent":{...},"turns":[...]}
```

When annotations are present, the CLI appends a `correlation_report` line with Kendall's τ-b, Cohen's κ (unweighted + weighted), and a confusion matrix.

## MCP Integration

The MCP server exposes Themis as tools for Claude Code and Claude Desktop.

**Available tools:**
- `evaluate_conversation` — evaluate a multi-turn conversation
- `get_conversation` — retrieve a stored evaluation by `conversation_id`

**Setup:**

```bash
# Add to Claude Code
claude mcp add --transport stdio themis -- /absolute/path/to/bin/themis-mcp

# Add to Claude Desktop (claude_desktop_config.json)
{
  "mcpServers": {
    "themis": {
      "command": "/absolute/path/to/bin/themis-mcp"
    }
  }
}
```

**Docker:**

```bash
docker build -t themis-mcp .
docker run --env-file .env themis-mcp
```

## Configuration

### Environment Variables (`.env`)

```env
# LLM Provider — choose at least one
OPEN_AI_KEY=sk-proj-...                    # OpenAI Platform (simplest)
AWS_REGION=us-east-1                       # AWS Bedrock (Claude models)
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...
AZURE_OPENAI_ENDPOINT=https://...          # Azure OpenAI

# API server
EVAL_AGENT_API_PORT=18082

# Scoring
CONVERSATION_HOLISTIC_WEIGHT=0.5           # α: weight for holistic score
SCORING_FORMULA=linear                     # linear | geometric | min
VERDICT_PASS_THRESHOLD=0.8
VERDICT_REVIEW_THRESHOLD=0.5

# Database
IN_MEMORY_DB=true                          # SQLite in-memory (default)
THEMIS_DB_URL=                             # PostgreSQL (if IN_MEMORY_DB=false)

# Streaming (optional Redis consumer)
CONVERSATION_STREAMING_ENABLED=false
REDIS_ADDR=localhost:6379
REDIS_CONVERSATION_STREAM_KEY=eval-conversations
REDIS_CONVERSATION_GROUP=eval-conv-group
REDIS_CONSUMER_NAME=consumer-1

# CLI
THEMIS_BATCH_WORKERS=5
```

See `.env.example` for the full reference.

### Judge Configuration (`configs/judges.yaml`)

Four judges are included out of the box:

| Judge | Scope | Default Weight | Purpose |
|-------|-------|---------------|---------|
| `relevance` | turn | 0.35 | Is the answer relevant to the query? |
| `coherence` | turn | 0.30 | Is the answer coherent and well-formed? |
| `completeness` | turn | 0.35 | Does the answer fully address the query? |
| `conversation-flow` | conversation | 1.0 | Does the conversation flow naturally? |

To enable or disable a judge, set `enabled: true/false` in `judges.yaml`. Each judge can use a different LLM provider and model:

```yaml
judges:
  default_model:
    modelFamily: openai_platform
    modelID: gpt-4o-mini
  evaluators:
    - name: relevance
      enabled: true
      scope: turn
      weight: 0.35
      prompt: "..."
```

## Database

**SQLite** (default, zero setup):
```env
IN_MEMORY_DB=true
```

**PostgreSQL** (persistent, production):
```env
IN_MEMORY_DB=false
THEMIS_DB_URL=postgresql://user:password@localhost:5432/themis?sslmode=disable
```

Run migrations before first use:
```bash
migrate -path ./migrations -database "$THEMIS_DB_URL" up
```

## Testing

```bash
# Unit and integration tests
go test ./...

# Smoke test (requires .env with valid LLM credentials)
bash scripts/smoke-test.sh
```

See [docs/testing/test-guide.md](docs/testing/test-guide.md) for the full test guide.

## License

[Apache License 2.0](LICENSE)
