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
# API server (HTTP endpoints)
go run cmd/api/main.go

# Batch evaluation (offline datasets)
go run cmd/batch/main.go -input dataset.jsonl -output results.jsonl -workers 5

# Redis stream consumer (asynchronous processing)
go run cmd/streaming/main.go

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

1. **Prechecks** (`internal/prechecks/`): Fast heuristics without LLM calls
   - Length checker, overlap checker, format checker
   - If average score < 0.2, early exit with `fail` verdict (saves 80% LLM cost)

2. **LLM Judges** (`internal/judge/`): Parallel execution of 6 judges
   - Relevance, faithfulness, coherence, completeness, instruction, correctness
   - Each judge runs concurrently (15s timeout per judge)
   - Skip logic: Judges auto-skip if required fields missing (e.g., correctness needs `expected_output`)

3. **Aggregation** (`internal/aggregator/`): Weighted combination
   - `confidence = (avg_stage1 × 0.3) + (avg_stage2 × 0.7)`
   - Verdict: `pass` (>0.8), `review` (>0.5), `fail` (≤0.5)

### Multi-Provider LLM Support

**Registry Pattern** (`internal/llm/`): Single pipeline can use multiple LLM providers simultaneously.

- Each judge in `configs/judges.yaml` specifies its own `modelFamily` and `modelID`
- Supported families: `anthropic` (AWS Bedrock Claude), `openai` (Azure OpenAI GPT)
- LLM client registry (`internal/llm/llm_client_factory.go`) maintains per-model clients
- Judge pool (`internal/judge/pool.go`) automatically selects correct client per judge

**Example**: Judge A uses Claude Sonnet, Judge B uses GPT-4o-mini, Judge C uses Claude Haiku - all in same evaluation.

### Dependency Injection

**Wiring** (`internal/setup/wiring.go`): Central dependency injection point.

Order of initialization:
1. Load `configs/judges.yaml` to discover required models
2. Create LLM client registry with all referenced models (AWS Bedrock + Azure OpenAI)
3. Build prechecks stage runner
4. Build judge pool from config (creates LLMJudge instances)
5. Create judge runner (parallel execution) and judge factory (single execution)
6. Create aggregator
7. Wire executors (AgentExecutor for full pipeline, JudgeExecutor for single judge)

### Entry Points

Five independent entry points sharing core evaluation logic:

1. **API** (`cmd/api/main.go`): REST endpoints with go-restful framework
   - `POST /api/v1/evaluate` - full pipeline
   - `POST /api/v1/evaluate/judge/{name}` - single judge
   - CORS enabled, structured logging

2. **Batch** (`cmd/batch/main.go`): CLI with concurrent worker pool
   - JSONL input/output formats
   - Validation mode with Kendall's τ correlation
   - Progress tracking, graceful shutdown

3. **Streaming** (`cmd/streaming/main.go`): Redis Streams consumer
   - Long-running consumer with acknowledgment
   - Horizontal scaling support
   - Fault tolerance via Redis persistence

4. **MCP** (`cmd/mcp/main.go`): Model Context Protocol server
   - Stdio-based communication
   - Exposes `evaluate_response` and `evaluate_single_judge` tools
   - Docker deployment for Claude Code/Desktop/Cursor

5. **Producer** (`cmd/producer/main.go`): Test data generator for Redis

All entry points use same core dependencies via `setup.Wire()`.

## Configuration

### Environment Variables (.env)

Required credentials (provider-dependent):
```bash
# AWS Bedrock Claude
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...

# Azure OpenAI GPT
OPEN_AI_KEY=...
AZURE_OPENAI_ENDPOINT=https://...openai.azure.com/openai/deployments/...
```

Pipeline configuration:
```bash
EVAL_AGENT_API_PORT=18082          # API server port
EARLY_EXIT_THRESHOLD=0.2           # Precheck early exit threshold
PRECHECK_WEIGHT=0.3                # Stage 1 weight in aggregation
LLM_JUDGE_WEIGHT=0.7               # Stage 2 weight in aggregation
```

### Judge Configuration (configs/judges.yaml)

YAML-driven judge definitions - edit prompts and models without code changes.

Each judge specifies:
- `name`: Unique identifier (relevance, faithfulness, coherence, completeness, instruction, correctness)
- `enabled`: Toggle judge on/off
- `weight`: Contribution to final score
- `requires_context`: Whether judge needs retrieved context (for RAG evaluation)
- `requires_expected_output`: Whether judge needs ground truth (for correctness evaluation)
- `model.modelFamily`: "anthropic" or "openai"
- `model.modelID`: Specific model identifier
- `model.max_tokens`, `temperature`, `retry`: Model settings
- `prompt`: Judge-specific evaluation prompt (uses Go template syntax)

**Important**: Each judge can use a different model. Mix Claude and GPT in same pipeline.

### Skip Logic

Judges with `requires_context: true` or `requires_expected_output: true` automatically skip if required field is missing in request. This maintains backwards compatibility - existing requests without optional fields continue working unchanged.

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
- **`internal/models/`**: Shared types (EvaluationContext, EvaluationResult, StageResult)

## Development Notes

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
go run cmd/batch/main.go \
  -input human_annotated_sample.jsonl \
  -validate \
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

### Adding LLM Provider Support

To add a new LLM provider (e.g., Anthropic direct API, Vertex AI):
1. Create new package in `internal/llm/<provider>/`
2. Implement `LLMClient` interface from `internal/llm/client.go`
3. Add new `LLMFamily` constant to `internal/llm/types.go`
4. Update `createLLMClientRegistry()` in `internal/setup/wiring.go`
5. Add provider credentials to `.env` and `Config` struct

### Parallel Execution

LLM judges run concurrently via `judge.Runner.Run()` which spawns goroutines for each enabled judge. Synchronization via channels and wait groups. Timeout per judge: 15 seconds (configurable in judge runner).

### Error Handling

- LLM client failures: Retry with exponential backoff (if `retry: true` in judge config)
- JSON parsing errors: Logged and judge returns zero score
- Context timeout: Individual judge times out, others continue
- Early exit: Triggered by low precheck score, skips LLM calls entirely

## Documentation

Comprehensive test cases and usage guides in `docs/`:
- `docs/api_test_cases.md` - HTTP endpoint testing
- `docs/batch_evaluation_test_cases.md` - Offline evaluation and validation
- `docs/mcp_test_cases.md` - Claude Code/Desktop integration
- `docs/redis_test_cases.md` - Redis Streams setup and consumer patterns
