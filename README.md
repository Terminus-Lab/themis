# Themis

**Themis** is an evaluation service for AI agent responses. It uses configurable LLM judges running in parallel to assess response quality across multiple dimensions.

## What Is This?

Themis is an **evaluation service/framework** that can be deployed in multiple modes for different use cases:

- **HTTP API** - Fast synchronous evaluation for real-time checks
- **API + Streaming** - Unified service running HTTP API and Redis stream consumer together
- **CLI (Batch)** - Offline evaluation with statistical validation (Kendall's tau)
- **MCP Server** - Integration with coding assistants (Claude Code, Claude Desktop, Cursor)

It evaluates AI responses using a two-stage pipeline: fast heuristics (prechecks) followed by parallel LLM judge evaluation.

## Philosophy

### 1. Configurable Multi-Provider Judges

Each judge is defined in `configs/judges.yaml` with complete flexibility:

- **Different LLM providers**: Mix AWS Bedrock Claude, Azure OpenAI GPT, and OpenAI Platform in the same evaluation
- **Different models per judge**: Each judge can use its own model (e.g., Judge A uses Claude Sonnet, Judge B uses GPT-4o-mini)
- **Custom prompts**: Edit evaluation prompts without code changes
- **Weighted scoring**: Each judge contributes differently to the final score

**Example configuration**: Run the same prompt with both Claude and GPT, or run different prompts for different quality dimensions.

### 2. Multiple Deployment Modes

**API Mode** (Default) - For fast synchronous checks:
```bash
go run cmd/api/main.go
curl -X POST http://localhost:18082/api/v1/evaluate -d '{...}'
```

**API + Streaming Mode** - Unified service for both HTTP and stream processing:
```bash
STREAMING_ENABLED=true go run cmd/api/main.go
```
- HTTP API available for manual testing and metrics (`/metrics` endpoint)
- Redis stream consumer runs in background for continuous monitoring
- Single deployment, single metrics endpoint, single health check

**Benefits of unified mode:**
- ✅ Prometheus metrics for both API and streaming
- ✅ Manual testing via API while streaming runs
- ✅ Simpler deployment (one service instead of two)
- ✅ Graceful shutdown handles both HTTP and streaming

**CLI Batch Mode** - For offline evaluation and judge validation:
```bash
go run cmd/batch/main.go -input dataset.jsonl -output results.jsonl
```
Unique features only available in CLI mode:
- Kendall's tau correlation analysis against human annotations
- Statistical validation reports (JSON output)
- Summary generation with aggregated statistics

**MCP Mode** - For local coding assistant integration:
```bash
go run cmd/mcp/main.go
```

**Horizontal Scaling** - Multiple streaming workers sharing load:
```bash
# Worker 1
STREAMING_ENABLED=true REDIS_CONSUMER_NAME=worker-1 go run cmd/api/main.go

# Worker 2 (different port for metrics)
STREAMING_ENABLED=true REDIS_CONSUMER_NAME=worker-2 EVAL_AGENT_API_PORT=18083 go run cmd/api/main.go

# Each worker processes messages from same Redis stream
# Redis consumer groups ensure no duplicate processing
```

### 3. Results Handling

Results are currently **logged** via structured logging (zerolog). Depending on your use case, you can extend the service to:
- Save results to a database
- Publish to Kafka topics for downstream processing
- Stream to observability platforms
- Integrate with your existing monitoring infrastructure

## Configuration

Secrets and configuration are loaded from a `.env` file:

```env
# AWS Bedrock credentials (if using Claude models)
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your_key
AWS_SECRET_ACCESS_KEY=your_secret

# Azure OpenAI credentials (if using Azure-hosted GPT models)
OPEN_AI_KEY=your_azure_openai_api_key
AZURE_OPENAI_ENDPOINT=https://...openai.azure.com/...

# OpenAI Platform credentials (if using direct OpenAI API)
OPEN_AI_KEY=sk-proj-...  # Standard OpenAI API key

# Service configuration
EVAL_AGENT_API_PORT=18082
EARLY_EXIT_THRESHOLD=0.2
PRECHECK_WEIGHT=0.3
LLM_JUDGE_WEIGHT=0.7

# Streaming configuration (for API + Streaming mode)
STREAMING_ENABLED=false           # Set to true to enable Redis stream consumer
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_STREAM_KEY=eval-events
REDIS_CONSUMER_GROUP=eval-group
REDIS_CONSUMER_NAME=consumer-1    # Unique per instance for horizontal scaling

# Verdict thresholds
VERDICT_PASS_THRESHOLD=0.8      # Confidence > 0.8 → "pass"
VERDICT_REVIEW_THRESHOLD=0.5    # Confidence > 0.5 → "review", else "fail"

# Aggregation configuration
ENABLE_PRECHECK=true                           # Enable/disable Stage 1 prechecks (default: true)
JUDGE_AGGREGATION_METHOD=weighted_average      # Stage 2 method: weighted_average, harmonic_mean, median, weighted_product
```

Judge configurations (prompts, models, weights) are defined in `configs/judges.yaml`:

```yaml
judges:
  default_model:
    modelFamily: "openai_platform"  # Options: anthropic, openai, openai_platform
    modelID: gpt-4o-mini            # Default model for all judges

  evaluators:
    - name: relevance
      enabled: true
      weight: 0.25
      model:
        modelFamily: "anthropic"
        modelID: us.anthropic.claude-3-5-sonnet-20241022-v2:0
      prompt: |
        You are an evaluation judge...

    - name: coherence
      enabled: true
      weight: 0.15
      model:
        modelFamily: "openai_platform"
        modelID: gpt-4o-mini
      prompt: |
        Evaluate logical consistency...
```

## Quick Start

### Prerequisites
- Go 1.21+
- One of the following LLM provider credentials:
  - **OpenAI Platform** API key (simplest - just `OPEN_AI_KEY`)
  - **AWS Bedrock** access (for Claude models)
  - **Azure OpenAI** access (for Azure-hosted GPT models)

### Run API Server
```bash
# Create .env file with credentials
cp .env.example .env  # Edit with your credentials

# Start API server
go run cmd/api/main.go
```

### Evaluate a Response
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-1",
    "event_type": "agent_response",
    "agent": {"name": "my-agent", "version": "1.0"},
    "interaction": {
      "user_query": "What is the capital of France?",
      "answer": "Paris"
    }
  }'
```

Response:
```json
{
  "id": "test-1",
  "stages": [
    {"name": "length-checker", "score": 1.0},
    {"name": "overlap-checker", "score": 0.85},
    {"name": "relevance-judge", "score": 0.95},
    {"name": "coherence-judge", "score": 1.0}
  ],
  "confidence": 0.92,
  "verdict": "pass",
  "metrics": {
    "stage1_avg": 0.85,
    "stage2_weighted_avg": 0.92,
    "stage2_harmonic_mean": 0.90,
    "stage2_median": 0.93,
    "stage2_weighted_product": 0.91,
    "final_confidence": 0.92,
    "aggregation_method": "weighted_average"
  }
}
```

### Verdict Thresholds

The verdict thresholds determine how confidence scores map to verdicts:
- **Pass threshold (default 0.8)**: Confidence above this → `pass` verdict
- **Review threshold (default 0.5)**: Confidence above this but below pass → `review` verdict
- **Below review threshold** → `fail` verdict

**When to adjust:**
- **High-stakes decisions** (medical, financial): Increase pass threshold to 0.9
- **Exploratory use cases**: Decrease pass threshold to 0.7
- **Strict quality requirements**: Increase review threshold to 0.6-0.7
- **A/B testing**: Experiment with different thresholds to optimize for your metrics

### Stage 2 Aggregation Methods

Choose how LLM judge scores are combined using `JUDGE_AGGREGATION_METHOD`:

| Method | Behavior | Use Case |
|--------|----------|----------|
| **weighted_average** (default) | Linear combination using judge weights | Balanced, general purpose |
| **harmonic_mean** | Heavily penalizes low outliers | Quality control - one bad judge matters |
| **median** | Middle value, ignores weights | Robust to extreme scores |
| **weighted_product** | Multiplicative - one low score tanks result | Strict - all judges must agree |

**Example configurations:**
```env
# Strict evaluation - all judges must agree
JUDGE_AGGREGATION_METHOD=weighted_product

# Balanced evaluation (default)
JUDGE_AGGREGATION_METHOD=weighted_average

# Robust to outliers
JUDGE_AGGREGATION_METHOD=median

# Quality control - penalize bad scores
JUDGE_AGGREGATION_METHOD=harmonic_mean
```

**All methods are computed and returned in `metrics`** - you can experiment without re-running evaluations by examining different scores in the response.

## How It Works

### Two-Stage Pipeline

```
Request → [Stage 1: Prechecks] → Early Exit Check → [Stage 2: LLM Judges] → Aggregation → Result
```

**Stage 1 (Prechecks)**: Fast heuristics without LLM calls (optional - can be disabled)
- Length checker, overlap checker, format checker
- If average score < 0.2, returns `fail` verdict immediately (saves 80% LLM cost)
- Disable with `ENABLE_PRECHECK=false` to use Stage 2 only

**Stage 2 (LLM Judges)**: Parallel execution of configured judges
- Evaluates: relevance, faithfulness, coherence, completeness, instruction-following, correctness
- Each judge runs concurrently (15s timeout)
- Judges auto-skip if required fields are missing
- **4 aggregation methods available**: weighted_average, harmonic_mean, median, weighted_product

**Aggregation**: Configurable combination methods
```
# Stage 2 aggregation (choose method via JUDGE_AGGREGATION_METHOD)
stage2_score = weighted_average | harmonic_mean | median | weighted_product

# Final confidence (if prechecks enabled)
confidence = (avg_stage1 × 0.3) + (stage2_score × 0.7)

# Or if prechecks disabled (ENABLE_PRECHECK=false)
confidence = stage2_score

# Verdict (configurable thresholds)
verdict = "pass" if > pass_threshold, "review" if > review_threshold, else "fail"
```

**All 4 Stage 2 methods are computed and returned in `metrics`** for transparency and experimentation.

## Documentation

Comprehensive testing guides for each deployment mode:

| Mode | Documentation |
|------|---------------|
| **API** | [docs/api_test_cases.md](docs/api_test_cases.md) |
| **Redis Streams** | [docs/redis_test_cases.md](docs/redis_test_cases.md) |
| **Batch/CLI** | [docs/batch_evaluation_test_cases.md](docs/batch_evaluation_test_cases.md) |
| **MCP** | [docs/mcp_test_cases.md](docs/mcp_test_cases.md) |

## Key Features

- **Multi-Provider LLM Support**: Mix AWS Bedrock Claude, Azure OpenAI GPT, and OpenAI Platform in single pipeline
- **Parallel Judge Execution**: All judges run concurrently for sub-5s latency
- **YAML-Driven Configuration**: Edit prompts and models without code changes
- **Configurable Aggregation**: 4 methods (weighted_average, harmonic_mean, median, weighted_product)
- **Optional Prechecks**: Enable/disable Stage 1 heuristics
- **Configurable Thresholds**: Adjust pass/review/fail boundaries per use case
- **Transparent Metrics**: All aggregation methods computed and returned in response
- **Statistical Validation**: Kendall's tau correlation against human annotations (CLI mode only)
- **Early Exit Optimization**: Skip LLM calls for obviously poor responses
- **Flexible Deployment**: API, streaming, batch, and MCP modes
- **Extensible Results**: Log, store in DB, or publish to Kafka

## Development

### Run Tests
```bash
go test ./...
```

### Build Docker Image (MCP)
```bash
docker build -t themis-mcp .
docker run --env-file .env themis-mcp
```

### Validate Judge Configuration
```bash
go run cmd/batch/main.go \
  -input human_annotated_sample.jsonl \
  -validate \
  -correlation-threshold 0.3
```

## License

MIT
