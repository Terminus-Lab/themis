# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Themis** - Evaluation service/framework for AI agent conversations using a two-phase pipeline:
1. **Phase A (Per-turn judges)**: relevance, coherence, completeness judges run on each turn → `turn_avg`
2. **Phase B (Holistic judge)**: conversation-flow judge evaluates the full conversation → `holistic_score`
3. **Final score**: `final_score = α × holistic_score + (1-α) × turn_avg` (α = `CONVERSATION_HOLISTIC_WEIGHT`, default 0.5)

Everything is a **conversation** — single-turn requests are represented as conversations with one turn.

Module path: `github.com/Terminus-Lab/themis`

## Core Commands

### Running Services
```bash
# API server (HTTP endpoints only)
go run cmd/api/main.go

# API + Conversation streaming (Redis consumer)
CONVERSATION_STREAMING_ENABLED=true go run cmd/api/main.go

# Batch evaluation (offline datasets)
go run cmd/batch/main.go evaluate -input dataset.jsonl -output results.jsonl

# With custom worker count (via env var)
THEMIS_BATCH_WORKERS=10 go run cmd/batch/main.go evaluate -input dataset.jsonl -output results.jsonl

# MCP server (Claude Code/Desktop integration)
go run cmd/mcp/main.go

# Redis producer (test data generation)
go run cmd/producer/main.go
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./internal/...

# Run specific package tests
go test ./internal/judge/...

# Run with coverage
go test -cover ./...
```

### Docker (MCP)
```bash
# Build MCP server image
docker build -t themis-mcp .

# Run with environment variables
docker run --env-file .env themis-mcp
```

## Architecture

### Two-Phase Conversation Evaluation Pipeline

**Flow**: Request → Phase A (per-turn judges) → Phase B (holistic judge) → Final Score → Verdict → Store

1. **Phase A — Per-turn judges** (`scope: "turn"` in judges.yaml):
   - Runs `relevance`, `coherence`, `completeness` judges on each turn concurrently
   - Each turn gets a `turn_score` (weighted average of judge scores)
   - `turn_avg` = average of all turn scores

2. **Phase B — Holistic judge** (`scope: "conversation"` in judges.yaml):
   - `conversation-flow` judge evaluates the entire conversation at once
   - Returns `holistic_score` and `holistic_reason`

3. **Final score**: `final_score = α × holistic_score + (1-α) × turn_avg`
   - α = `CONVERSATION_HOLISTIC_WEIGHT` (default 0.5)

4. **Verdict thresholds**:
   - `final_score > VERDICT_PASS_THRESHOLD (0.8)` → `pass`
   - `final_score > VERDICT_REVIEW_THRESHOLD (0.5)` → `review`
   - otherwise → `fail`

### Multi-Provider LLM Support

**Registry Pattern** (`internal/llm/`): Each judge specifies its own `modelFamily` and `modelID`.

- Supported families:
  - `anthropic` - AWS Bedrock Claude models
  - `openai` - Azure OpenAI GPT models
  - `openai_platform` - OpenAI Platform (direct API)
- LLM client registry (`internal/llm/llm_client_factory.go`) maintains per-model clients

### Dependency Injection

**Wiring** (`internal/setup/wiring.go`): Central dependency injection point.

Order of initialization:
1. Load `configs/judges.yaml`
2. Create LLM client registry
3. Build turn judges (`scope: "turn"`) via `JudgePool.BuildTurnJudgesFromConfig`
4. Build holistic judge (`scope: "conversation"`) via `JudgePool.BuildHolisticJudgeFromConfig`
5. Create `JudgeRunner` for turn judges
6. Create `ConversationEvaluator` with turn runner + holistic judge
7. Initialize storage (SQLite or PostgreSQL)

### Entry Points

Four entry points sharing core evaluation logic:

1. **API** (`cmd/api/main.go`): REST endpoints with go-restful framework
   - `POST /api/v1/conversations/evaluate` - evaluate a full conversation
   - `GET /api/v1/conversations` - list all conversations with summary metrics
   - `GET /api/v1/conversations/{id}` - get conversation with turn-level evaluations
   - `GET /api/v1/metrics/health` - health metrics (window query param: 1d, 7d, 30d)
   - `GET /api/v1/health` - health check
   - `GET /` - dashboard UI (static HTML at `static/dashboard.html`)
   - CORS enabled, structured logging
   - **Can run with conversation streaming** (`CONVERSATION_STREAMING_ENABLED=true`): HTTP + Redis consumer in same process

2. **CLI** (`cmd/batch/main.go`): Command-line interface with concurrent worker pool
   - `evaluate` command: JSONL input/output for batch conversation evaluation
   - Progress tracking, graceful shutdown

3. **MCP** (`cmd/mcp/main.go`): Model Context Protocol server
   - Stdio-based communication
   - Tools: `evaluate_conversation`, `get_conversation`
   - Docker deployment for Claude Code/Desktop/Cursor

4. **Producer** (`cmd/producer/main.go`): Test data generator for Redis streams

All entry points use same core dependencies via `setup.Wire()`.

## Configuration

### Environment Variables (.env)

Required credentials (provider-dependent):
```bash
# AWS Bedrock Claude (modelFamily: "anthropic")
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...

# Azure OpenAI GPT (modelFamily: "openai")
OPEN_AI_KEY=...
AZURE_OPENAI_ENDPOINT=https://...openai.azure.com/openai/deployments/...

# OpenAI Platform (modelFamily: "openai_platform") — simplest to get started
OPEN_AI_KEY=sk-proj-...
```

Pipeline configuration:
```bash
EVAL_AGENT_API_PORT=18082          # API server port

# Conversation scoring
CONVERSATION_HOLISTIC_WEIGHT=0.5   # α: weight for holistic score (0.0–1.0)
VERDICT_PASS_THRESHOLD=0.8         # final_score > this → "pass"
VERDICT_REVIEW_THRESHOLD=0.5       # final_score > this → "review", else "fail"

# Database
IN_MEMORY_DB=true                  # Use SQLite in-memory (default: true) - zero setup required
THEMIS_DB_URL=                     # PostgreSQL connection string (only if IN_MEMORY_DB=false)

# Streaming (optional)
CONVERSATION_STREAMING_ENABLED=false      # Enable Redis consumer
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_CONVERSATION_STREAM_KEY=eval-conversations
REDIS_CONVERSATION_GROUP=eval-conv-group
REDIS_CONSUMER_NAME=consumer-1

# CLI
THEMIS_BATCH_WORKERS=5             # Number of concurrent workers for batch evaluation
```

### Judge Configuration (configs/judges.yaml)

YAML-driven judge definitions — 4 judges total:

| Judge | Scope | Weight | Purpose |
|-------|-------|--------|---------|
| `relevance` | turn | 0.35 | Is the answer relevant to the query? |
| `coherence` | turn | 0.30 | Is the answer coherent and well-formed? |
| `completeness` | turn | 0.35 | Does the answer fully address the query? |
| `conversation-flow` | conversation | 1.0 | Does the conversation flow naturally? |

Each judge specifies:
- `scope`: `"turn"` for per-turn evaluation, `"conversation"` for holistic evaluation
- `weight`: Contribution to turn_avg (for turn judges) or holistic_score (for conversation judges)
- `model.modelFamily`: `"anthropic"`, `"openai"`, or `"openai_platform"`
- `model.modelID`: Specific model identifier
- `prompt`: Go template string for the judge prompt

## Key Packages

- **`internal/executor/`**: `ConversationEvaluator` — two-phase evaluation orchestration
- **`internal/judge/`**: Judge implementation, factory, pool (`BuildTurnJudgesFromConfig`, `BuildHolisticJudgeFromConfig`), runner
- **`internal/llm/`**: Multi-provider LLM client abstraction and registry
- **`internal/setup/`**: Dependency injection, configuration loading, wiring
- **`internal/config/`**: YAML configuration parsing (judges.yaml)
- **`internal/batch/`**: Batch processing for conversations, JSONL reader/writer
- **`internal/api/`**: HTTP handlers, routes, middleware
- **`internal/stream/`**: Redis Streams consumer implementation (conversation only)
- **`internal/storage/`**: Repository interface with SQLite (default) and PostgreSQL implementations
- **`internal/models/`**: Shared types (`ConversationEvaluationRequest`, `ConversationEvaluationResult`, `TurnEvaluationResult`)

## Development Notes

### Database Storage

**SQLite (Default)** - Automatically used for development and testing:
- Set `IN_MEMORY_DB=true` (default) to use SQLite in-memory storage
- Zero configuration required
- Schema: single `conversations` table with `turn_avg`, `holistic_score`, `final_score`, `turn_results` (JSON)

**PostgreSQL (Production)** - For persistent storage:
- Set `IN_MEMORY_DB=false`
- Provide `THEMIS_DB_URL` connection string
- Run migrations: `migrate -path ./migrations -database "$THEMIS_DB_URL" up`

### Adding a New Judge

1. Add judge definition to `configs/judges.yaml`
2. Set `scope: "turn"` or `scope: "conversation"` depending on type
3. Set `enabled: true` and specify `modelFamily`/`modelID`
4. Restart service — judge pool automatically builds from config
5. No code changes needed (YAML-driven)

### Parallel Execution

Turn judges run concurrently via goroutines + WaitGroup in `ConversationEvaluator.evaluateTurns()`.
Each turn spawns a goroutine that runs all turn judges via `JudgeRunner.Run()` (which also runs judges concurrently).
The holistic judge runs after all turns complete, with a 30-second timeout.

### Error Handling

- LLM client failures: Retry with exponential backoff (if `retry: true` in judge config)
- JSON parsing errors: Logged and judge returns zero score
- Context timeout: Individual judge times out, others continue
- Storage failures: Logged but don't affect the returned result

## Testing

### API Tests
```bash
go test ./internal/api/...
```
Tests cover: health check, list conversations (empty), get conversation (404), evaluate validation errors, health metrics.
Integration test (`TestAPI_EvaluateConversation_Integration`) requires `OPEN_AI_KEY` and is skipped automatically.

### Storage Tests
```bash
go test ./internal/storage/sqlite/...
```
Tests cover: store/get conversation, list conversations, health metrics, not-found handling.

### Judge Pool Tests
```bash
go test ./internal/judge/...
```
Tests cover: build turn judges from config, skip disabled judges, nil config, no enabled judges error, invalid template error.
