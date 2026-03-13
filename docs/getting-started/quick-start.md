---
title: Quick Start
description: Get started with Themis in 5 minutes
version: 1.0.0
tags: [quick-start, tutorial, getting-started, examples]
related:
  - getting-started/installation.md
  - getting-started/configuration.md
  - deployment/api-mode.md
  - testing/api-tests.md
---

# Quick Start

Get Themis running and evaluate your first AI response in 5 minutes.

## Step 1: Download and Extract

```bash
# Download latest release (choose your platform)
# macOS Apple Silicon (M1/M2/M3):
curl -LO https://github.com/Terminus-Lab/themis/releases/download/v1.0.0/themis_1.0.0_darwin_arm64.tar.gz
tar -xzf themis_1.0.0_darwin_arm64.tar.gz
cd themis_1.0.0_darwin_arm64

# macOS Intel:
# curl -LO https://github.com/Terminus-Lab/themis/releases/download/v1.0.0/themis_1.0.0_darwin_amd64.tar.gz

# Linux:
# curl -LO https://github.com/Terminus-Lab/themis/releases/download/v1.0.0/themis_1.0.0_linux_amd64.tar.gz

# Windows PowerShell:
# Invoke-WebRequest -Uri "https://github.com/Terminus-Lab/themis/releases/download/v1.0.0/themis_1.0.0_windows_amd64.zip" -OutFile "themis_1.0.0_windows_amd64.zip"
# Expand-Archive themis_1.0.0_windows_amd64.zip

# Set up environment (OpenAI - simplest option)
cp .env.example .env
echo "OPEN_AI_KEY=sk-proj-YOUR_KEY_HERE" >> .env
```

## Step 2: Start the Server

```bash
./themis-api

# Windows:
# .\themis-api.exe
```

Expected output:
```
INFO judge created successfully judge=relevance
INFO judge created successfully judge=faithfulness
INFO judge pool built successfully total_judges=5
INFO Starting Themis Server address=:18082
```

## Step 3: Evaluate Your First Response

```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-001",
    "event_type": "agent_response",
    "agent": {
      "name": "my-agent",
      "version": "1.0"
    },
    "interaction": {
      "user_query": "What is the capital of France?",
      "context": "France is a country in Western Europe. Paris is its capital city.",
      "answer": "The capital of France is Paris."
    }
  }'
```

### Response Breakdown

```json
{
  "id": "test-001",
  "stages": [
    {"name": "length-checker", "score": 1.0},
    {"name": "overlap-checker", "score": 0.85},
    {"name": "format-checker", "score": 1.0},
    {"name": "relevance-judge", "score": 0.95},
    {"name": "faithfulness-judge", "score": 1.0},
    {"name": "coherence-judge", "score": 1.0},
    {"name": "completeness-judge", "score": 1.0},
    {"name": "instruction-judge", "score": 1.0}
  ],
  "confidence": 0.92,
  "verdict": "pass",
  "metrics": {
    "stage1_avg": 0.95,
    "stage2_weighted_avg": 0.99,
    "stage2_harmonic_mean": 0.98,
    "stage2_median": 1.0,
    "stage2_weighted_product": 0.98,
    "aggregation_method": "weighted_average"
  }
}
```

**Understanding the response:**
- **8 stages**: 3 prechecks (fast heuristics) + 5 LLM judges (parallel evaluation)
- **confidence**: Final aggregated score (0.92 = 92% confident the answer is good)
- **verdict**: `pass` (threshold > 0.8), `review` (0.5-0.8), or `fail` (< 0.5)
- **metrics**: All 4 aggregation methods computed for transparency

## Step 4: View Results in Dashboard

Open your browser to `http://localhost:18082`

The dashboard shows:
- Real-time evaluation results
- Filters by agent, verdict, pagination
- Expandable rows with detailed stage scores
- Auto-refresh every 10 seconds

## Common Use Cases

### Use Case 1: Check Answer Quality

Evaluate if an AI answer is high-quality:

```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "quality-check",
    "agent": {"name": "my-agent"},
    "interaction": {
      "user_query": "Explain quantum computing",
      "answer": "Yes."
    }
  }'
```

Expected: `verdict: "fail"`, early exit (only 3 precheck stages, no expensive LLM calls)

### Use Case 2: Detect Hallucinations

Check if answer contains facts not in context:

```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "hallucination-check",
    "agent": {"name": "my-agent"},
    "interaction": {
      "user_query": "What is the population of Tokyo?",
      "context": "Tokyo is the capital of Japan.",
      "answer": "Tokyo has 50 million people and is the largest city in China."
    }
  }'
```

Expected: Low faithfulness and coherence scores (catches hallucination about China)

### Use Case 3: Validate Against Ground Truth

Compare answer with expected output:

```bash
# First, enable correctness judge in configs/judges.yaml
# Set correctness.enabled: true

curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "correctness-check",
    "agent": {"name": "my-agent"},
    "interaction": {
      "user_query": "What is 2+2?",
      "answer": "The answer is four",
      "expected_output": "4"
    }
  }'
```

Expected: Correctness judge scores ~0.9 (semantic match despite different format)

### Use Case 4: Evaluate Single Quality Dimension

Test only one specific judge (faster):

```bash
curl -X POST http://localhost:18082/api/v1/evaluate/judge/relevance \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "relevance-only",
    "agent": {"name": "my-agent"},
    "interaction": {
      "user_query": "What is machine learning?",
      "answer": "Machine learning allows computers to learn from data."
    }
  }'
```

Expected: Only 1 stage (relevance-judge), response time ~1-2 seconds

### Use Case 5: Query Past Results

Retrieve previous evaluation results:

```bash
# Get all results for specific agent
curl "http://localhost:18082/api/v1/results?agent_name=my-agent&limit=10"

# Filter by verdict
curl "http://localhost:18082/api/v1/results?verdict=fail&limit=20"

# Get specific result by ID
curl "http://localhost:18082/api/v1/results/test-001"
```

## Next Steps

### Deploy Different Modes

- **[API Mode](../deployment/api-mode.md)** - HTTP API for synchronous evaluation
- **See CLAUDE.md** for batch, MCP, and streaming mode documentation

### Customize Configuration

- **[Configuration Guide](configuration.md)** - Environment variables and thresholds
- **See CLAUDE.md** for adding judges and aggregation methods

### Run Tests

- **[API Tests](../testing/api-tests.md)** - HTTP endpoint test cases
- **[Batch Tests](../testing/batch-tests.md)** - CLI batch processing tests
- **[MCP Tests](../testing/mcp-tests.md)** - MCP integration tests
- **[Streaming Tests](../testing/streaming-tests.md)** - Redis consumer tests

## Common Issues

### "judge not found" Error

Ensure `configs/judges.yaml` exists and is properly formatted. Check logs for:
```
INFO judge created successfully judge=relevance
```

### Low Confidence on Good Answers

Adjust verdict thresholds in `.env`:
```env
VERDICT_PASS_THRESHOLD=0.7  # Lower from default 0.8
VERDICT_REVIEW_THRESHOLD=0.4  # Lower from default 0.5
```

### Slow Response Times

Check if early exit is working:
```bash
# Test with obviously bad answer
curl -X POST http://localhost:18082/api/v1/evaluate \
  -d '{"interaction":{"user_query":"Long question here","answer":"No"}}'

# Should return quickly with only 3 stages
```

### Dashboard Not Loading

Verify API server is running:
```bash
curl http://localhost:18082/
# Should return HTML
```

Check browser console for errors and ensure no CORS issues.

## Performance Tips

1. **Enable Early Exit** - Bad answers skip expensive LLM calls:
   ```env
   ENABLE_PRECHECK=true  # Default
   EARLY_EXIT_THRESHOLD=0.2
   ```

2. **Disable Unused Judges** - Edit `configs/judges.yaml`:
   ```yaml
   - name: instruction
     enabled: false  # Skip if not needed
   ```

3. **Use Lighter Models** - Faster, cheaper evaluation:
   ```yaml
   default_model:
     modelFamily: "openai_platform"
     modelID: gpt-4o-mini  # Fast and cheap
   ```

4. **Horizontal Scaling (Streaming)** - Multiple consumers:
   ```bash
   # Consumer 1
   STREAMING_ENABLED=true REDIS_CONSUMER_NAME=worker-1 ./themis-api

   # Consumer 2
   STREAMING_ENABLED=true REDIS_CONSUMER_NAME=worker-2 EVAL_AGENT_API_PORT=18083 ./themis-api
   ```

## Building from Source (Development)

If you want to contribute or modify Themis:

```bash
# Clone repository
git clone https://github.com/Terminus-Lab/themis.git
cd themis

# Install dependencies
go mod download

# Build binaries
go build -o bin/themis-api cmd/api/main.go
go build -o bin/themis-mcp cmd/mcp/main.go
go build -o bin/themis-cli cmd/batch/main.go

# Run from source
go run cmd/api/main.go
```

## Getting Help

- **Documentation**: See [CLAUDE.md](../../CLAUDE.md) for complete documentation
- **Test Cases**: See [API Tests](../testing/api-tests.md) for comprehensive test scenarios
- **Issues**: Report bugs at https://github.com/Terminus-Lab/themis/issues
