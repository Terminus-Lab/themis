---
title: API Mode
description: Deploy Themis as an HTTP API server for synchronous evaluation
version: 1.0.0
tags: [deployment, api, http, rest, synchronous]
related:
  - getting-started/quick-start.md
  - getting-started/configuration.md
  - testing/api-tests.md
  - testing/streaming-tests.md
---

# API Mode

Deploy Themis as an HTTP REST API server for synchronous, on-demand evaluation.

## Overview

**API Mode** provides HTTP endpoints for:
- Real-time evaluation of AI responses
- Single judge evaluation
- Result querying and filtering
- Web dashboard for visualization
- Prometheus metrics

**Best for**:
- Interactive applications
- Real-time quality checks
- Development and testing
- Manual evaluation via dashboard

## Quick Start

### 1. Configure Environment

```bash
# Minimal .env configuration
OPEN_AI_KEY=sk-proj-...
EVAL_AGENT_API_PORT=18082
```

### 2. Start Server

```bash
go run cmd/api/main.go
```

**Expected output**:
```
INFO judge pool built successfully total_judges=5
INFO Starting Themis Server address=:18082 streaming_enabled=false
```

### 3. Test Endpoint

```bash
curl http://localhost:18082/
# Should return dashboard HTML
```

## Available Endpoints

### 1. Full Pipeline Evaluation

**POST** `/api/v1/evaluate`

Runs complete two-stage evaluation pipeline.

**Request**:
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "eval-001",
    "event_type": "agent_response",
    "agent": {
      "name": "my-agent",
      "version": "1.0"
    },
    "interaction": {
      "user_query": "What is the capital of France?",
      "context": "France is a country in Europe...",
      "answer": "Paris"
    }
  }'
```

**Response**:
```json
{
  "id": "eval-001",
  "stages": [
    {"name": "length-checker", "score": 1.0},
    {"name": "relevance-judge", "score": 0.95},
    ...
  ],
  "confidence": 0.92,
  "verdict": "pass",
  "metrics": {
    "stage1_avg": 0.95,
    "stage2_weighted_avg": 0.92,
    "aggregation_method": "weighted_average"
  }
}
```

### 2. Single Judge Evaluation

**POST** `/api/v1/evaluate/judge/{name}`

Run only one specific judge (faster).

**Available judges**: relevance, faithfulness, coherence, completeness, instruction, correctness

**Request**:
```bash
curl -X POST "http://localhost:18082/api/v1/evaluate/judge/relevance?threshold=0.8" \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "eval-002",
    "agent": {"name": "my-agent"},
    "interaction": {
      "user_query": "What is AI?",
      "answer": "AI is artificial intelligence."
    }
  }'
```

**Query parameters**:
- `threshold` (optional) - Custom pass/fail threshold (default: 0.8)

**Response**:
```json
{
  "id": "eval-002",
  "stages": [
    {"name": "relevance-judge", "score": 0.95, "reason": "Answer directly addresses query"}
  ],
  "confidence": 0.95,
  "verdict": "pass"
}
```

### 3. Query Results

**GET** `/api/v1/results`

Retrieve past evaluation results with filters.

**Query parameters**:
- `agent_name` - Filter by agent name (exact match)
- `verdict` - Filter by verdict: `pass`, `review`, or `fail`
- `limit` - Results per page (default: 50, max: 100)
- `offset` - Pagination offset (default: 0)

**Request**:
```bash
# All results for agent
curl "http://localhost:18082/api/v1/results?agent_name=my-agent&limit=10"

# Failed evaluations only
curl "http://localhost:18082/api/v1/results?verdict=fail&limit=20"

# Pagination
curl "http://localhost:18082/api/v1/results?limit=50&offset=100"
```

**Response**:
```json
{
  "results": [
    {
      "event_id": "eval-001",
      "agent_name": "my-agent",
      "agent_version": "1.0",
      "user_query": "What is the capital of France?",
      "answer": "Paris",
      "context": "France is...",
      "confidence": 0.92,
      "verdict": "pass",
      "stage_scores": [...],
      "created_at": "2024-03-10T10:00:00Z"
    }
  ],
  "total": 150,
  "count": 10,
  "limit": 10,
  "offset": 0,
  "has_more": true
}
```

### 4. Get Single Result

**GET** `/api/v1/results/{event_id}`

Retrieve specific evaluation by ID.

**Request**:
```bash
curl "http://localhost:18082/api/v1/results/eval-001"
```

**Response**: Single evaluation result object.

### 5. Dashboard

**GET** `/`

Web dashboard for visualizing results.

**Access**: Open `http://localhost:18082` in browser

**Features**:
- Real-time results table
- Filter by agent, verdict
- Expandable rows for details
- Auto-refresh every 10s
- Dark terminal theme

### 6. Health Check

**GET** `/health`

Server health status.

**Request**:
```bash
curl http://localhost:18082/health
```

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2024-03-10T10:00:00Z"
}
```

### 7. Metrics (Prometheus)

**GET** `/metrics`

Prometheus-compatible metrics endpoint.

**Request**:
```bash
curl http://localhost:18082/metrics
```

**Available metrics**:
- `themis_evaluations_total` - Total evaluations by verdict
- `themis_evaluation_duration_seconds` - Evaluation latency
- `themis_judge_scores` - Judge score distributions
- `themis_early_exits_total` - Early exit count

## Configuration

### Environment Variables

```env
# Server Configuration
EVAL_AGENT_API_PORT=18082          # HTTP port (default: 18082)

# Pipeline Settings
ENABLE_PRECHECK=true               # Enable Stage 1 (default: true)
EARLY_EXIT_THRESHOLD=0.2           # Precheck early exit (default: 0.2)

# Aggregation
PRECHECK_WEIGHT=0.3                # Stage 1 weight (default: 0.3)
LLM_JUDGE_WEIGHT=0.7               # Stage 2 weight (default: 0.7)
JUDGE_AGGREGATION_METHOD=weighted_average  # Stage 2 method

# Verdict Thresholds
VERDICT_PASS_THRESHOLD=0.8         # Pass threshold (default: 0.8)
VERDICT_REVIEW_THRESHOLD=0.5       # Review threshold (default: 0.5)

# Database
IN_MEMORY_DB=true                  # SQLite in-memory (default)
THEMIS_DB_URL=                     # PostgreSQL URL (if IN_MEMORY_DB=false)

# LLM Providers
OPEN_AI_KEY=sk-proj-...            # OpenAI Platform
# OR
AWS_REGION=us-east-1               # AWS Bedrock
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...
# OR
AZURE_OPENAI_ENDPOINT=...          # Azure OpenAI
OPEN_AI_KEY=...
```

## Deployment Patterns

### Pattern 1: Single Instance (Development)

Simple standalone deployment:

```bash
go run cmd/api/main.go
```

**Use case**: Local development, testing, demos

### Pattern 2: Docker Container

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o themis-api cmd/api/main.go

FROM alpine:latest
COPY --from=builder /app/themis-api /usr/local/bin/
COPY --from=builder /app/configs /configs
COPY --from=builder /app/static /static
CMD ["themis-api"]
```

```bash
docker build -t themis-api .
docker run -p 18082:18082 --env-file .env themis-api
```

**Use case**: Containerized deployments, Kubernetes

### Pattern 3: Load Balanced (High Availability)

Multiple API instances behind load balancer:

```bash
# Instance 1
EVAL_AGENT_API_PORT=18082 go run cmd/api/main.go

# Instance 2
EVAL_AGENT_API_PORT=18083 go run cmd/api/main.go

# Instance 3
EVAL_AGENT_API_PORT=18084 go run cmd/api/main.go

# Nginx/HAProxy load balancer
# Round-robin to 18082, 18083, 18084
```

**Use case**: High traffic, fault tolerance

### Pattern 4: Unified API + Streaming

Single service running both API and streaming consumer:

```bash
STREAMING_ENABLED=true go run cmd/api/main.go
```

**Benefits**:
- HTTP API for manual testing
- Redis consumer for async evaluation
- Single deployment
- Unified metrics endpoint

See [CLAUDE.md](../../CLAUDE.md) and [Streaming Tests](../testing/streaming-tests.md) for details.

## Performance Optimization

### 1. Enable Early Exit

Skip expensive LLM calls for obviously bad answers:

```env
ENABLE_PRECHECK=true
EARLY_EXIT_THRESHOLD=0.2  # Default, increase to 0.3 for more exits
```

**Impact**: ~80% cost savings on low-quality responses

### 2. Parallel Judge Execution

Judges run concurrently by default. Ensure sufficient resources:

```bash
# Check concurrent LLM calls
# 5 judges = 5 concurrent API calls to LLM providers
```

**Response time**: ~3-4 seconds (limited by slowest judge)

### 3. Use Lighter Models

Configure fast, cheap models in `judges.yaml`:

```yaml
default_model:
  modelFamily: "openai_platform"
  modelID: gpt-4o-mini  # Fast and cheap
```

**Trade-off**: Lower accuracy for faster response

### 4. Disable Unused Judges

Edit `judges.yaml`:

```yaml
- name: instruction
  enabled: false  # Skip if not needed
```

**Impact**: Fewer LLM calls, faster evaluation

### 5. Connection Pooling

Go HTTP client pools connections automatically. Configure limits:

```go
// In internal/llm/client.go
http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100
```

## Monitoring

### Prometheus Integration

Scrape metrics endpoint:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'themis'
    static_configs:
      - targets: ['localhost:18082']
    metrics_path: '/metrics'
```

### Key Metrics to Track

1. **Evaluation throughput**: `rate(themis_evaluations_total[5m])`
2. **P95 latency**: `histogram_quantile(0.95, themis_evaluation_duration_seconds)`
3. **Verdict distribution**: `themis_evaluations_total` by verdict label
4. **Early exit rate**: `rate(themis_early_exits_total[5m])`
5. **Judge scores**: `themis_judge_scores` histogram

### Grafana Dashboard

Sample queries:

```promql
# Evaluation rate
rate(themis_evaluations_total[5m])

# Average confidence by verdict
avg(themis_confidence_score) by (verdict)

# Early exit savings
rate(themis_early_exits_total[5m]) / rate(themis_evaluations_total[5m])
```

## Troubleshooting

### Issue: Port Already in Use

```bash
# Change port
EVAL_AGENT_API_PORT=18083 go run cmd/api/main.go
```

### Issue: Slow Response Times

Check:
1. Early exit working? (test with bad answer)
2. Judge timeouts? (default: 15s per judge)
3. LLM provider latency? (check provider status)

### Issue: CORS Errors in Dashboard

API enables CORS by default. If issues persist:

```go
// internal/api/server.go
cors := restful.CrossOriginResourceSharing{
    AllowedOrigins: []string{"*"},
    AllowedMethods: []string{"GET", "POST"},
    Container:      container,
}
container.Filter(cors.Filter)
```

### Issue: Database Connection Errors

**SQLite (default)**: Should work out of the box

**PostgreSQL**: Verify connection:
```bash
psql "$THEMIS_DB_URL"
```

Check migrations applied:
```bash
migrate -path ./migrations -database "$THEMIS_DB_URL" version
```

## Security Considerations

### 1. Authentication

API has **no authentication** by default (designed for internal use).

For production:
- Deploy behind API gateway with auth
- Use network policies (VPC, security groups)
- Add custom middleware:

```go
// Add API key middleware
container.Filter(apiKeyAuth)
```

### 2. Rate Limiting

Add rate limiting middleware:

```go
import "golang.org/x/time/rate"

limiter := rate.NewLimiter(10, 20)  // 10 req/s, burst 20
container.Filter(rateLimitFilter(limiter))
```

### 3. Input Validation

API validates required fields. For additional validation:
- Max query/answer length
- Content filtering
- Schema validation

## Next Steps

- [API Tests](../testing/api-tests.md) - Comprehensive test cases
- [Streaming Tests](../testing/streaming-tests.md) - Async evaluation with Redis
- [Batch Tests](../testing/batch-tests.md) - Offline evaluation
- [Configuration](../getting-started/configuration.md) - Tune your setup
- [CLAUDE.md](../../CLAUDE.md) - Complete documentation
