# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Themis** - Evaluation service/framework for AI agent responses using a two-stage pipeline:
1. **Stage 1 (Prechecks)**: Fast heuristics (length, overlap, format) - no LLM calls
2. **Stage 2 (LLM Judges)**: Parallel LLM evaluation across 6 quality dimensions

Key differentiator: Built-in Kendall's τ validation against human annotations to ensure judge accuracy before production deployment.

Module path: `github.com/Terminus-Lab/themis`

## Core Commands

### Running Services
```bash
# API server (HTTP endpoints only)
go run cmd/api/main.go

# API + Streaming (unified service - recommended for production)
STREAMING_ENABLED=true go run cmd/api/main.go

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

### Two-Stage Evaluation Pipeline

**Flow**: Request → Prechecks → Early Exit Check → LLM Judges (parallel) → Aggregation → Result

1. **Prechecks** (`internal/prechecks/`): Fast heuristics without LLM calls (optional)
   - Length checker, overlap checker, format checker
   - If average score < 0.2, early exit with `fail` verdict (saves 80% LLM cost)
   - Can be disabled via `ENABLE_PRECHECK=false` to use Stage 2 only

2. **LLM Judges** (`internal/judge/`): Parallel execution of 6 judges
   - Relevance, faithfulness, coherence, completeness, instruction, correctness
   - Each judge runs concurrently (15s timeout per judge)
   - Skip logic: Judges auto-skip if required fields missing (e.g., correctness needs `expected_output`)

3. **Aggregation** (`internal/aggregator/`): Configurable combination
   - Stage 1 can be disabled via `ENABLE_PRECHECK=false`
   - Stage 2 aggregation method configurable: `weighted_average`, `harmonic_mean`, `median`, `weighted_product`
   - All 4 Stage 2 methods computed and returned in metrics
   - `confidence = (avg_stage1 × 0.3) + (stage2_selected × 0.7)` if prechecks enabled, else `confidence = stage2_selected`
   - Verdict thresholds are configurable via env vars (defaults: pass=0.8, review=0.5)
   - Verdict: `pass` (> pass threshold), `review` (> review threshold), `fail` (≤ review threshold)

### Multi-Provider LLM Support

**Registry Pattern** (`internal/llm/`): Single pipeline can use multiple LLM providers simultaneously.

- Each judge in `configs/judges.yaml` specifies its own `modelFamily` and `modelID`
- Supported families:
  - `anthropic` - AWS Bedrock Claude models
  - `openai` - Azure OpenAI GPT models
  - `openai_platform` - OpenAI Platform (direct API with standard API key)
- LLM client registry (`internal/llm/llm_client_factory.go`) maintains per-model clients
- Judge pool (`internal/judge/pool.go`) automatically selects correct client per judge

**Example**: Judge A uses Claude Sonnet, Judge B uses GPT-4o-mini from OpenAI Platform, Judge C uses Azure-hosted GPT-4 - all in same evaluation.

### Dependency Injection

**Wiring** (`internal/setup/wiring.go`): Central dependency injection point.

Order of initialization:
1. Load `configs/judges.yaml` to discover required models
2. Create LLM client registry with all referenced models (AWS Bedrock + Azure OpenAI + OpenAI Platform)
3. Build prechecks stage runner
4. Build judge pool from config (creates LLMJudge instances)
5. Create judge runner (parallel execution) and judge factory (single execution)
6. Create aggregator
7. Wire executors (AgentExecutor for full pipeline, JudgeExecutor for single judge)

### Entry Points

Four entry points sharing core evaluation logic:

1. **API** (`cmd/api/main.go`): REST endpoints with go-restful framework
   - `POST /api/v1/evaluate` - full pipeline evaluation
   - `POST /api/v1/evaluate/judge/{name}` - single judge evaluation
   - `GET /api/v1/results` - query evaluation results with filters (agent_name, verdict, limit, offset)
   - `GET /api/v1/results/{event_id}` - get single evaluation result by ID
   - `GET /api/v1/conversations` - list all conversations with summary metrics
   - `GET /api/v1/conversations/{id}` - get conversation turns with detailed evaluations
   - `GET /` - dashboard UI (static HTML at `static/dashboard.html`)
   - CORS enabled, structured logging
   - **Can run in two modes:**
     - **API only** (default): HTTP endpoints only
     - **API + Streaming** (`STREAMING_ENABLED=true`): HTTP + Redis consumer in same process
   - **Unified mode benefits:**
     - Single deployment for both HTTP and streaming
     - Manual testing via API while streaming runs
     - Graceful shutdown handles both HTTP and streaming
     - Horizontal scaling with multiple consumer instances
   - **Dashboard UI:**
     - Dark terminal theme (Claude Code-inspired)
     - Two-tab interface: Results and Conversations
     - **Results tab**: Real-time visualization with filtering by agent/verdict, expandable rows
     - **Conversations tab**: Multi-turn conversation tracking with turn-by-turn drill-down
     - Clickable turns navigate to full evaluation details
     - URL-based navigation with browser back/forward support
     - Auto-refresh every 10 seconds
     - No authentication required (local dev only)

2. **CLI** (`cmd/batch/main.go`): Command-line interface with concurrent worker pool
   - JSONL input/output formats
   - Validation mode with Kendall's τ correlation
   - Progress tracking, graceful shutdown

3. **MCP** (`cmd/mcp/main.go`): Model Context Protocol server
   - Stdio-based communication
   - Exposes `evaluate_response`, `evaluate_single_judge`, and `get_conversation` tools
   - Supports conversation tracking with `conversation_id`, `agent_name`, `agent_version` fields
   - Docker deployment for Claude Code/Desktop/Cursor

4. **Producer** (`cmd/producer/main.go`): Test data generator for Redis

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

# OpenAI Platform (modelFamily: "openai_platform")
OPEN_AI_KEY=sk-proj-...  # Standard OpenAI API key - simplest option
```

Pipeline configuration:
```bash
EVAL_AGENT_API_PORT=18082          # API server port
EARLY_EXIT_THRESHOLD=0.2           # Precheck early exit threshold
PRECHECK_WEIGHT=0.3                # Stage 1 weight in aggregation
LLM_JUDGE_WEIGHT=0.7               # Stage 2 weight in aggregation
VERDICT_PASS_THRESHOLD=0.8         # Confidence > this → "pass"
VERDICT_REVIEW_THRESHOLD=0.5       # Confidence > this → "review", else "fail"
ENABLE_PRECHECK=true               # Enable Stage 1 prechecks
JUDGE_AGGREGATION_METHOD=weighted_average  # Stage 2: weighted_average, harmonic_mean, median, weighted_product

# Database configuration
IN_MEMORY_DB=true                  # Use SQLite in-memory (default: true) - zero setup required
THEMIS_DB_URL=                     # PostgreSQL connection string (only if IN_MEMORY_DB=false)

# Streaming configuration (unified API + Streaming mode)
STREAMING_ENABLED=false            # Enable Redis stream consumer alongside API
REDIS_ADDR=localhost:6379          # Redis server address
REDIS_PASSWORD=                    # Redis password (optional)
REDIS_STREAM_KEY=eval-events       # Redis stream key
REDIS_CONSUMER_GROUP=eval-group    # Consumer group name
REDIS_CONSUMER_NAME=consumer-1     # Unique consumer name (for horizontal scaling)

# CLI configuration
THEMIS_BATCH_WORKERS=5             # Number of concurrent workers for CLI evaluation (default: 5)
```

### Judge Configuration (configs/judges.yaml)

YAML-driven judge definitions - edit prompts and models without code changes.

Each judge specifies:
- `name`: Unique identifier (relevance, faithfulness, coherence, completeness, instruction, correctness)
- `enabled`: Toggle judge on/off
- `weight`: Contribution to final score
- `requires_context`: Whether judge needs retrieved context (for RAG evaluation)
- `requires_expected_output`: Whether judge needs ground truth (for correctness evaluation)
- `model.modelFamily`: "anthropic", "openai", or "openai_platform"
- `model.modelID`: Specific model identifier
- `model.max_tokens`, `temperature`, `retry`: Model settings
- `prompt`: Judge-specific evaluation prompt (uses Go template syntax)

**Important**: Each judge can use a different model and provider. Mix Claude, Azure GPT, and OpenAI Platform GPT in same pipeline.

### Skip Logic

Judges with `requires_context: true` or `requires_expected_output: true` automatically skip if required field is missing in request. This maintains backwards compatibility - existing requests without optional fields continue working unchanged.

### Stage 2 Aggregation Methods

Four methods available for combining LLM judge scores (`JUDGE_AGGREGATION_METHOD`):

1. **weighted_average** (default): Linear combination using judge weights from `judges.yaml`
   - Formula: `sum(score × weight) / sum(weight)`
   - Best for: General purpose, balanced evaluation

2. **harmonic_mean**: Weighted harmonic mean - heavily penalizes low scores
   - Formula: `sum(weight) / sum(weight/score)`
   - Best for: Quality control where one bad judge matters
   - Returns 0 if any score is 0

3. **median**: Middle score value (ignores weights)
   - Formula: Middle value after sorting, or average of two middle values if even count
   - Best for: Robust to outliers, simple evaluation

4. **weighted_product**: Multiplicative combination - one low score tanks everything
   - Formula: `product(score^normalized_weight)`
   - Best for: Strict evaluation where all judges must agree

**All methods are computed on every request** and returned in `metrics` field for transparency and experimentation.

## Key Packages

- **`internal/executor/`**: Pipeline orchestration (agent_executor.go, judge_executor.go)
- **`internal/judge/`**: Judge implementation, factory, pool, runner
- **`internal/prechecks/`**: Fast heuristic checkers (no LLM)
- **`internal/aggregator/`**: Stage result aggregation into final confidence + verdict
- **`internal/llm/`**: Multi-provider LLM client abstraction and registry
- **`internal/setup/`**: Dependency injection, configuration loading, wiring
- **`internal/config/`**: YAML configuration parsing (judges.yaml)
- **`internal/batch/`**: Batch processing, validation, JSONL reader/writer
- **`internal/api/`**: HTTP handlers, routes, middleware
- **`internal/stream/`**: Redis Streams consumer implementation
- **`internal/storage/`**: Database abstraction layer with SQLite (default) and PostgreSQL implementations
- **`internal/models/`**: Shared types (EvaluationContext, EvaluationResult, StageResult)

## Development Notes

### Database Storage

**SQLite (Default)** - Automatically used for development and testing:
- Set `IN_MEMORY_DB=true` (default) to use SQLite in-memory storage
- Zero configuration required
- Perfect for local development and testing
- Evaluation results stored temporarily during service runtime

**PostgreSQL (Production)** - For persistent storage:
- Set `IN_MEMORY_DB=false`
- Provide `THEMIS_DB_URL` connection string
- Run migrations: `migrate -path ./migrations -database "$THEMIS_DB_URL" up`
- Use for production deployments requiring data persistence

### Query API Usage

Query evaluation results programmatically:
```bash
# Filter by agent name
curl "http://localhost:18082/api/v1/results?agent_name=my-agent&limit=10"

# Filter by verdict
curl "http://localhost:18082/api/v1/results?verdict=pass&limit=20&offset=0"

# Get specific result
curl "http://localhost:18082/api/v1/results/event-123"
```

Query parameters:
- `agent_name` - Filter by agent name (exact match)
- `verdict` - Filter by verdict: "pass", "review", or "fail"
- `limit` - Number of results per page (default: 50)
- `offset` - Pagination offset (default: 0)

### Dashboard UI

Access the web dashboard at `http://localhost:18082` after starting the API server.

**Location**: `static/dashboard.html` - single-page HTML application

**Features**:
- Dark terminal theme with Claude Code aesthetic
- Real-time updates (auto-refresh every 10s)
- Filter by agent name, verdict, or limit results
- Click rows to expand and view full details
- Pagination with Previous/Next controls
- No build step required - pure HTML/CSS/JS

**Customization**:
- Edit `static/dashboard.html` directly to modify UI
- Colors and styling defined in `<style>` block
- API calls use relative URLs (works with any port)
- Auto-detects API base URL from `window.location.origin`

**Use cases**:
- Visual monitoring during development
- Debugging judge scores and reasoning
- Quick inspection of evaluation results
- Demo and stakeholder presentations

### Adding a New Judge

1. Add judge definition to `configs/judges.yaml` with prompt and model settings
2. Set `enabled: true` and specify `modelFamily`/`modelID`
3. Restart service - judge pool automatically builds from config
4. No code changes needed (YAML-driven)

### Validating Judge Accuracy

Before deploying judge changes to production:
```bash
# 1. Collect human annotations for sample dataset (25% recommended)
# 2. Run validation mode
go run cmd/batch/main.go validate-events \
  -input human_annotated_sample.jsonl \
  -correlation-threshold 0.3

# 3. Check Kendall's τ ≥ 0.3 in JSON output
# 4. If passed, deploy updated configs/judges.yaml
```

### Testing Judge Changes

Iterative prompt tuning workflow:
1. Edit prompts in `configs/judges.yaml`
2. Run batch evaluation on validation set
3. Compare Kendall's τ with previous version
4. Deploy if τ ≥ 0.3 and improved

### Testing Aggregation Methods

Experiment with different Stage 2 aggregation methods:
```bash
# Test with weighted average (default)
JUDGE_AGGREGATION_METHOD=weighted_average go run cmd/api/main.go

# Test with harmonic mean (penalizes low scores)
JUDGE_AGGREGATION_METHOD=harmonic_mean go run cmd/api/main.go

# Test with median (robust to outliers)
JUDGE_AGGREGATION_METHOD=median go run cmd/api/main.go

# Test with weighted product (strict - all must agree)
JUDGE_AGGREGATION_METHOD=weighted_product go run cmd/api/main.go
```

Check metrics in response to compare all methods:
```json
{
  "metrics": {
    "stage2_weighted_avg": 0.85,
    "stage2_harmonic_mean": 0.82,
    "stage2_median": 0.87,
    "stage2_weighted_product": 0.79,
    "aggregation_method": "weighted_average"
  }
}
```

### Adding LLM Provider Support

To add a new LLM provider (e.g., Anthropic direct API, Vertex AI, Cohere):
1. Create new package in `internal/llm/<provider>/` (e.g., `internal/llm/vertexai/`)
2. Implement `LLMClient` interface from `internal/llm/client.go`
3. Add new `LLMFamily` constant to `internal/llm/llm_client_factory.go`
4. Update `createLLMClientRegistry()` in `internal/setup/wiring.go` with new case
5. Add provider credentials to `.env` and `Config` struct in `internal/setup/wiring.go`

**Example**: See `internal/llm/openaiplatform/` for reference implementation.

### Parallel Execution

LLM judges run concurrently via `judge.Runner.Run()` which spawns goroutines for each enabled judge. Synchronization via channels and wait groups. Timeout per judge: 15 seconds (configurable in judge runner).

### Error Handling

- LLM client failures: Retry with exponential backoff (if `retry: true` in judge config)
- JSON parsing errors: Logged and judge returns zero score
- Context timeout: Individual judge times out, others continue
- Early exit: Triggered by low precheck score, skips LLM calls entirely

## Documentation

Comprehensive guides and test cases in `docs/`:
- `docs/getting-started/installation.md` - Prerequisites and setup
- `docs/getting-started/quick-start.md` - 5-minute tutorial
- `docs/getting-started/configuration.md` - Environment variables and judges.yaml
- `docs/deployment/api-mode.md` - HTTP API deployment
- `docs/testing/api-tests.md` - HTTP endpoint test cases
- `docs/testing/batch-tests.md` - CLI batch processing test cases
- `docs/testing/mcp-tests.md` - Claude Code/Desktop integration test cases
- `docs/testing/streaming-tests.md` - Redis consumer test cases
