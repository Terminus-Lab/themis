# Themis Architecture

## Overview

Themis is an evaluation framework for AI agent responses using a two-stage pipeline with multi-provider LLM support. The architecture prioritizes cost efficiency (early exit on poor responses), flexibility (YAML-driven configuration), and production readiness (parallel execution, graceful shutdown, conversation tracking).

**Module**: `github.com/Terminus-Lab/themis`

## Two-Stage Evaluation Pipeline

```
Request → Stage 1: Prechecks → Early Exit Check → Stage 2: LLM Judges → Aggregation → Result
          (heuristics)          (< 0.2 = fail)   (6 judges parallel)   (confidence)
          ~5ms                                     ~2-3s                 verdict
```

### Stage 1: Prechecks (Optional)
- **Purpose**: Fast heuristic filtering without LLM calls
- **Checkers**: Length, overlap, format
- **Early Exit**: If average score < 0.2, return `fail` verdict (saves 80% LLM cost)
- **Toggle**: `ENABLE_PRECHECK=false` to skip Stage 1

### Stage 2: LLM Judges (Parallel)
- **Purpose**: Deep quality evaluation across 6 dimensions
- **Judges**: Relevance, faithfulness, coherence, completeness, instruction, correctness
- **Execution**: Concurrent goroutines with 15s timeout per judge
- **Skip Logic**: Judges auto-skip if required fields missing (e.g., correctness needs `expected_output`)

### Aggregation
- **Configurable Methods**: `weighted_average`, `harmonic_mean`, `median`, `weighted_product`
- **All methods computed**: Returned in `metrics` for experimentation
- **Confidence Calculation**:
  - Prechecks enabled: `(stage1_avg × 0.3) + (stage2_selected × 0.7)`
  - Prechecks disabled: `stage2_selected`
- **Verdict Thresholds**: pass (>0.8), review (>0.5), fail (≤0.5)

## Directory Structure

```
themis/
├── cmd/                          # Entry points
│   ├── api/                      # HTTP API + optional streaming
│   ├── batch/                    # CLI batch processor
│   ├── mcp/                      # Model Context Protocol server
│   └── producer/                 # Test data generator
├── internal/
│   ├── executor/                 # Pipeline orchestration
│   │   ├── agent_executor.go    # Full two-stage pipeline
│   │   └── judge_executor.go    # Single judge execution
│   ├── judge/                    # LLM judge implementation
│   │   ├── factory.go           # Single judge creation
│   │   ├── pool.go              # Judge registry from config
│   │   ├── runner.go            # Parallel execution engine
│   │   └── llm_judge.go         # Core judge logic
│   ├── prechecks/                # Stage 1 heuristics
│   │   ├── runner.go            # Precheck orchestrator
│   │   └── checkers/            # Length, overlap, format
│   ├── aggregator/               # Stage combination logic
│   ├── llm/                      # Multi-provider LLM clients
│   │   ├── client.go            # LLMClient interface
│   │   ├── llm_client_factory.go # Registry pattern
│   │   ├── anthropic/           # AWS Bedrock Claude
│   │   ├── openai/              # Azure OpenAI
│   │   └── openaiplatform/      # OpenAI Platform (direct API)
│   ├── setup/                    # Dependency injection
│   │   └── wiring.go            # Central initialization point
│   ├── config/                   # YAML configuration parser
│   ├── storage/                  # Database abstraction
│   │   ├── interface.go         # Storage interface
│   │   ├── sqlite.go            # SQLite (default, in-memory)
│   │   └── postgres.go          # PostgreSQL (production)
│   ├── api/                      # HTTP handlers, routes
│   ├── stream/                   # Redis consumer
│   ├── batch/                    # CLI processing logic
│   └── models/                   # Shared types
├── configs/
│   └── judges.yaml               # Judge definitions (prompts, models)
├── static/
│   └── dashboard.html            # Web UI (single-page app)
└── migrations/                   # Database schema versions
```

## Key Components

### Dependency Injection (`internal/setup/wiring.go`)

Central initialization point with strict ordering:

1. Load `configs/judges.yaml` → discover required models
2. Create LLM client registry → instantiate per-model clients
3. Build prechecks runner → Stage 1 checkers
4. Build judge pool → LLMJudge instances from config
5. Create judge runner → parallel execution engine
6. Create aggregator → score combination logic
7. Wire executors → AgentExecutor (full pipeline), JudgeExecutor (single judge)

**Pattern**: Registry-based DI ensures single source of truth for configuration.

### Multi-Provider LLM Support (`internal/llm/`)

**Registry Pattern**: Each judge specifies `modelFamily` and `modelID` in `judges.yaml`. LLM client factory maintains per-model clients.

```yaml
judges:
  - name: relevance
    modelFamily: anthropic      # AWS Bedrock Claude
    modelID: us.anthropic.claude-sonnet-4-0-20250514-v1:0
  - name: coherence
    modelFamily: openai_platform # OpenAI Platform (simplest)
    modelID: gpt-4o-mini
  - name: completeness
    modelFamily: openai          # Azure OpenAI
    modelID: gpt-4o-2024-08-06
```

**Benefit**: Mix providers in single evaluation. Judge pool auto-selects correct client per judge.

### Parallel Execution (`internal/judge/runner.go`)

- Spawns goroutines for each enabled judge
- Channels + WaitGroups for synchronization
- Per-judge timeout (15s default)
- Collects results into slice for aggregation

### Conversation Tracking (`internal/storage/`)

Multi-turn agent interactions grouped by `conversation_id`:

- **Fields**: `conversation_id`, `agent_name`, `agent_version`
- **Storage methods**: `GetConversation()`, `ListConversations()`
- **Aggregates**: Turn count, avg confidence, verdict distribution
- **API endpoints**: `GET /api/v1/conversations`, `GET /api/v1/conversations/{id}`
- **MCP tool**: `get_conversation` for Claude Code integration

## Entry Points

### 1. API Server (`cmd/api/main.go`)

**Two modes:**
- **API only** (default): HTTP endpoints only
- **API + Streaming** (`STREAMING_ENABLED=true`): HTTP + Redis consumer in same process

**Endpoints:**
- `POST /api/v1/evaluate` - Full pipeline evaluation
- `POST /api/v1/evaluate/judge/{name}` - Single judge
- `GET /api/v1/results` - Query with filters (agent_name, verdict, pagination)
- `GET /api/v1/results/{event_id}` - Single result
- `GET /api/v1/conversations` - List conversations with summary
- `GET /api/v1/conversations/{id}` - Conversation turns with evaluations
- `GET /` - Dashboard UI

### 2. Batch CLI (`cmd/batch/main.go`)

- JSONL input/output
- Concurrent worker pool
- Validation mode with Kendall's τ correlation
- Progress tracking, graceful shutdown

### 3. MCP Server (`cmd/mcp/main.go`)

- Stdio-based Model Context Protocol
- Tools: `evaluate_response`, `evaluate_single_judge`, `get_conversation`
- Docker deployment for Claude Code/Desktop/Cursor

### 4. Producer (`cmd/producer/main.go`)

- Test data generator for Redis Streams
- Development/testing only

## Data Flow

```
┌─────────────┐
│ Entry Point │ (API/Batch/MCP/Producer)
└──────┬──────┘
       │
       v
┌──────────────────┐
│ AgentExecutor    │ (Pipeline orchestration)
└──────┬───────────┘
       │
       v
┌──────────────────┐
│ Prechecks Runner │ Stage 1: Fast heuristics
└──────┬───────────┘
       │
       ├─> Early exit? (score < 0.2) → Return "fail"
       │
       v
┌──────────────────┐
│ Judge Runner     │ Stage 2: Parallel LLM judges
│ ┌──────────────┐ │
│ │ Judge Pool   │ │ (6 judges × LLM clients)
│ └──────────────┘ │
└──────┬───────────┘
       │
       v
┌──────────────────┐
│ Aggregator       │ Combine stages → confidence + verdict
└──────┬───────────┘
       │
       v
┌──────────────────┐
│ Storage          │ Save to SQLite/PostgreSQL
└──────────────────┘
```

## Configuration

### Environment Variables (`.env`)
- **Credentials**: `AWS_*`, `OPEN_AI_KEY`, `AZURE_OPENAI_ENDPOINT`
- **Pipeline**: `ENABLE_PRECHECK`, `JUDGE_AGGREGATION_METHOD`, `VERDICT_*_THRESHOLD`
- **Database**: `IN_MEMORY_DB` (true = SQLite, false = PostgreSQL)
- **Streaming**: `STREAMING_ENABLED`, `REDIS_*`

### Judge Configuration (`configs/judges.yaml`)
- **Per-judge settings**: name, enabled, weight, model family/ID, prompt
- **YAML-driven**: Edit prompts without code changes
- **Validation**: Schema validation on startup

## Database Schema

### `evaluations` table
- `id`, `event_id`, `user_query`, `answer`, `context`, `expected_output`
- `conversation_id`, `agent_name`, `agent_version` (nullable for backwards compat)
- `verdict`, `confidence`, `metrics` (JSON)
- `stages` (JSON array of stage results)
- `created_at`

**Indexes**: `event_id`, `conversation_id`, `verdict`, `created_at`

## Design Principles

1. **Cost Efficiency**: Early exit saves 80% LLM costs on poor responses
2. **Flexibility**: YAML-driven judges, multi-provider support, configurable aggregation
3. **Parallel by Default**: Judges run concurrently, minimize latency
4. **Production Ready**: Graceful shutdown, structured logging, metrics, conversation tracking
5. **Backwards Compatible**: Optional fields (conversation_id, context, expected_output)
6. **Simple Defaults**: SQLite in-memory (zero config), unified API+Streaming mode

## Adding New Components

### New LLM Provider
1. Create `internal/llm/<provider>/` package
2. Implement `LLMClient` interface
3. Add `LLMFamily` constant to factory
4. Update `createLLMClientRegistry()` in `wiring.go`

### New Judge
1. Add definition to `configs/judges.yaml`
2. Specify `modelFamily`, `modelID`, prompt
3. Restart service (auto-loaded from config)

### New Entry Point
1. Create `cmd/<name>/main.go`
2. Call `setup.Wire()` for DI
3. Use `AgentExecutor` or `JudgeExecutor`
4. Handle storage via `Storage` interface
