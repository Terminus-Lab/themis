---
title: API Test Cases
description: Comprehensive test scenarios for Themis HTTP API endpoints
version: 1.0.0
tags: [testing, api, http, test-cases, endpoints]
related:
  - deployment/api-mode.md
  - testing/batch-tests.md
  - testing/mcp-tests.md
  - testing/streaming-tests.md
---

# API Test Cases

Comprehensive test scenarios for the Themis HTTP API.

**Important:** Expected responses show **representative results**. Actual LLM judge scores may vary slightly (±0.1) due to model variability, but the overall patterns (high/medium/low scores, verdicts, stage counts) should match. Focus on validating behavior patterns rather than exact numeric matches.

## Setup

### Prerequisites

Ensure your `.env` file is configured:

```env
# AWS Bedrock credentials (if using anthropic models)
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your_key
AWS_SECRET_ACCESS_KEY=your_secret

# OpenAI credentials (if using openai models)
OPEN_AI_KEY=sk-...

# API configuration
EVAL_AGENT_API_PORT=18082
EARLY_EXIT_THRESHOLD=0.2
```

Judges are configured in `configs/judges.yaml`. The system dynamically creates LLM clients based on models referenced in the configuration.

### Start the Server

```bash
cd themis
./bin/themis-api
```

Server runs on `http://localhost:18082`

**Expected startup logs:**
```
INFO judge created successfully judge=relevance model_family=anthropic model_id=us.anthropic.claude-3-5-haiku-20241022-v1:0
INFO judge created successfully judge=faithfulness model_family=anthropic model_id=us.anthropic.claude-3-5-haiku-20241022-v1:0
INFO judge created successfully judge=coherence model_family=anthropic model_id=us.anthropic.claude-3-5-haiku-20241022-v1:0
INFO judge created successfully judge=completeness model_family=anthropic model_id=us.anthropic.claude-3-5-haiku-20241022-v1:0
INFO judge created successfully judge=instruction model_family=anthropic model_id=us.anthropic.claude-3-5-haiku-20241022-v1:0
INFO judge pool built successfully total_judges=5
```

## Full Pipeline Evaluation Tests

### Test Case 1: Happy Path - High Quality Answer

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-001",
    "conversation_id": "conv-test-001",
    "event_type": "agent_response",
    "agent": {"name": "test-agent", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "What is the capital of France?",
      "context": "France is a country in Western Europe. Paris is its capital city and largest metropolis.",
      "answer": "The capital of France is Paris."
    }
  }'
```

**Expected:**
- Status Code: 200
- `confidence` > 0.8
- `verdict` = "pass"
- **8 stages** (3 prechecks + 5 judges)
- All scores high (answer is correct, relevant, and grounded)
- Response time: ~3-4 seconds (judges run in parallel)

### Test Case 2: Early Exit - Very Short Answer

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-002",
    "conversation_id": "conv-test-002",
    "event_type": "agent_response",
    "agent": {"name": "test-agent", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "Explain quantum computing in detail",
      "context": "Quantum computing uses quantum mechanics principles...",
      "answer": "Yes."
    }
  }'
```

**Expected:**
- Status Code: 200
- `confidence` < 0.2
- `verdict` = "fail"
- Only 3 stages (prechecks only, early exit triggered)
- No LLM judges called (cost savings)

### Test Case 3: Fail Verdict - Irrelevant Answer

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-003",
    "conversation_id": "conv-test-003",
    "event_type": "agent_response",
    "agent": {"name": "test-agent", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "What are the main causes of climate change?",
      "context": "Climate change is primarily caused by greenhouse gas emissions from human activities.",
      "answer": "There are various factors that contribute to weather patterns."
    }
  }'
```

**Expected:**
- Status Code: 200
- `confidence` ~0.35
- `verdict` = "fail"
- 8 stages (full pipeline - all judges run)
- Low scores across all judges:
  - **relevance: 0.2** - Answer discusses weather patterns, not climate change causes
  - **faithfulness: 0.2** - Doesn't mention greenhouse gases from context
  - **coherence: 0.3** - Vague but not contradictory
  - **completeness: 0.0** - Completely fails to address the query
  - **instruction: 0.4** - No specific instructions violated
- Total duration: ~3 seconds (5 parallel LLM calls)

### Test Case 4: Hallucination Detection

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-004",
    "conversation_id": "conv-test-004",
    "event_type": "agent_response",
    "agent": {"name": "test-agent", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "What is the population of Tokyo?",
      "context": "Tokyo is the capital of Japan.",
      "answer": "Tokyo has a population of 50 million people and is the largest city in China."
    }
  }'
```

**Expected:**
- Status Code: 200
- `confidence` = 0.393
- `verdict` = "fail"
- 8 stages (full pipeline)
- **Excellent hallucination detection** - All judges identify the errors:
  - **faithfulness: 0.1** - Correctly flags factual errors not in context ("China" not mentioned)
  - **coherence: 0.2** - Identifies logical inconsistency (Tokyo cannot be in both Japan and China)
  - **relevance: 0.2** - Notes answer addresses population but with wrong facts
  - **completeness: 0.0** - Answer is factually wrong
  - **instruction: 0.4** - No specific format instructions, penalizes factual errors
- Total duration: ~2.3 seconds (judges run in parallel)
- System successfully detects:
  - Geographic hallucination (China vs Japan)
  - Population exaggeration (50M vs actual ~14M city, ~37M metro)
  - Contradictory claims

---

## Single Judge Evaluation Tests

### Test Case 5: Relevance Judge

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate/judge/relevance \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-005",
    "conversation_id": "conv-test-005",
    "event_type": "agent_response",
    "agent": {"name": "test", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "What is machine learning?",
      "context": "ML is a subset of AI.",
      "answer": "Machine learning is a method where computers learn from data without explicit programming."
    }
  }'
```

**Expected:**
- Status Code: 200
- Only 1 stage (relevance-judge)
- Score close to 1.0 for relevant answer

### Test Case 6: Custom Threshold

**Request:**
```bash
curl -X POST "http://localhost:18082/api/v1/evaluate/judge/faithfulness?threshold=0.9" \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-006",
    "conversation_id": "conv-test-006",
    "event_type": "agent_response",
    "agent": {"name": "test", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "What is the boiling point of water?",
      "context": "Water boils at 100°C at sea level.",
      "answer": "Water boils at 100 degrees Celsius."
    }
  }'
```

**Expected Response:**
- Status Code: 200
- High faithfulness score (grounded in context)
- `verdict` = "pass" (score > 0.9 threshold)

### Test Case 7: Invalid Judge Name

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate/judge/invalid-judge \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-007",
    "conversation_id": "conv-test-007",
    "event_type": "agent_response",
    "agent": {"name": "test", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "Test",
      "context": "Test",
      "answer": "Test"
    }
  }'
```

**Expected Response:**
- Status Code: 400 or 404
- Error message: "judge not found" or similar

---

## Error Handling Tests

### Test Case 8: Missing Required Fields

**Request (missing `conversation_id`):**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-008",
    "interaction": {
      "user_query": "Test",
      "answer": "Test answer"
    }
  }'
```

**Expected:**
- Status Code: 400
- Error: `"conversation_id is required"`

**Request (missing `answer`):**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-008b",
    "conversation_id": "conv-test-008",
    "interaction": {
      "user_query": "Test"
    }
  }'
```

**Expected:**
- Status Code: 400
- Error: `"answer is required"`

---

## Performance Tests

### Test Case 9: Concurrent Requests

**Script:**
```bash
# Send 10 concurrent requests
for i in {1..10}; do
  curl -X POST http://localhost:18082/api/v1/evaluate \
    -H "Content-Type: application/json" \
    -d "{\"event_id\":\"perf-$i\",\"conversation_id\":\"conv-perf-$i\",\"event_type\":\"agent_response\",\"agent\":{\"name\":\"test\",\"type\":\"rag\",\"version\":\"1.0\"},\"interaction\":{\"user_query\":\"Test\",\"context\":\"Test\",\"answer\":\"Test\"}}" &
done
wait
```

**Expected:**
- All 10 requests complete successfully
- Response times < 5 seconds per request
- No race conditions or errors

### Test Case 10: Large Context (10KB)

**Request:**
```bash
# Generate large context
LARGE_CONTEXT=$(python3 -c "print('Context word. ' * 2000)")

curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d "{
    \"event_id\": \"test-010\",
    \"conversation_id\": \"conv-test-010\",
    \"event_type\": \"agent_response\",
    \"agent\": {\"name\": \"test\", \"type\": \"rag\", \"version\": \"1.0\"},
    \"interaction\": {
      \"user_query\": \"Summarize the context\",
      \"context\": \"$LARGE_CONTEXT\",
      \"answer\": \"This is a summary.\"
    }
  }"
```

**Expected:**
- Status Code: 200
- Evaluation completes without timeout
- All judges handle large context

---

## Edge Cases

### Test Case 11: Special Characters in Answer

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-011",
    "conversation_id": "conv-test-011",
    "event_type": "agent_response",
    "agent": {"name": "test", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "Show me code",
      "context": "Python example",
      "answer": "def hello():\n    print(\"Hello, World!\")\n    return True"
    }
  }'
```

**Expected:**
- Status Code: 200
- Evaluation handles newlines and special characters
- Format checker passes

### Test Case 12: Non-English Text

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-012",
    "conversation_id": "conv-test-012",
    "event_type": "agent_response",
    "agent": {"name": "test", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "Quelle est la capitale de la France?",
      "context": "La France est un pays en Europe.",
      "answer": "La capitale de la France est Paris."
    }
  }'
```

**Expected:**
- Status Code: 200
- Evaluation works with non-English text
- Judges provide appropriate scores

---

## Multi-Provider Testing

### Test Case 13: Mixed Provider Evaluation

**Setup:**
Update `configs/judges.yaml` to use different providers:

```yaml
judges:
  default_model:
    modelFamily: "anthropic"
    modelID: us.anthropic.claude-3-5-haiku-20241022-v1:0

  evaluators:
    - name: relevance
      model:
        modelFamily: "anthropic"
        modelID: us.anthropic.claude-3-5-haiku-20241022-v1:0
    - name: faithfulness
      model:
        modelFamily: "openai"
        modelID: gpt-4o-mini
```

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-multi-provider",
    "conversation_id": "conv-multi-provider",
    "event_type": "agent_response",
    "agent": {"name": "test-agent", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "What is machine learning?",
      "context": "ML is a subset of AI that enables systems to learn from data.",
      "answer": "Machine learning allows computers to learn patterns from data without explicit programming."
    }
  }'
```

**Expected:**
- Status Code: 200
- Startup logs show both Anthropic and OpenAI clients initialized
- Each judge uses its configured provider
- All judges complete successfully
- Mixed provider evaluation works seamlessly

---

## Result Query Tests

### Test Case 14: Query All Results

**Request:**
```bash
curl "http://localhost:18082/api/v1/results?limit=10&offset=0"
```

**Expected:**
- Status Code: 200
- Returns paginated results
- Includes `total`, `count`, `has_more` fields

### Test Case 15: Filter by Agent Name

**Request:**
```bash
curl "http://localhost:18082/api/v1/results?agent_name=my-agent&limit=20"
```

**Expected:**
- Status Code: 200
- Returns only results for specified agent
- All results have `agent_name` = "my-agent"

### Test Case 16: Filter by Verdict

**Request:**
```bash
curl "http://localhost:18082/api/v1/results?verdict=fail&limit=10"
```

**Expected:**
- Status Code: 200
- Returns only failed evaluations
- All results have `verdict` = "fail"

### Test Case 17: Get Single Result by ID

**Request:**
```bash
curl "http://localhost:18082/api/v1/results/test-001"
```

**Expected:**
- Status Code: 200 (if exists) or 404 (if not found)
- Returns single evaluation result with full details

---

## Dashboard Tests

### Test Case 18: Access Dashboard

**Request:**
```bash
curl http://localhost:18082/
```

**Expected:**
- Status Code: 200
- Returns HTML content
- Dashboard loads in browser at http://localhost:18082

### Test Case 19: Dashboard Filter Functionality

**Manual test in browser:**
1. Open http://localhost:18082
2. Enter agent name in filter
3. Click "Fetch Results"
4. Verify results filtered correctly

**Expected:**
- Results update based on filter
- Pagination works
- Expandable rows show details

---

## Conversation Tests

### Test Case 20: List All Conversations

**Request:**
```bash
curl "http://localhost:18082/api/v1/conversations"
```

**Expected:**
- Status Code: 200
- Body: `{"conversations": [...], "total": N}`
- Each entry includes `conversation_id`, `turn_count`, `avg_confidence`, `verdict_counts`, `agent_name`

### Test Case 21: Get Conversation by ID

**Request:**
```bash
curl "http://localhost:18082/api/v1/conversations/conv-abc123"
```

**Expected:**
- Status Code: 200 (if exists) or 404 (if not found)
- Body includes `turns` array with full evaluation details per turn, ordered by `created_at ASC`
- `avg_confidence` is the mean across all turns

---

## Health Metrics Tests

### Test Case 22: Health Metrics - Default Window

**Request:**
```bash
curl "http://localhost:18082/api/v1/metrics/health"
```

**Expected:**
- Status Code: 200
- `window` = "7d"
- Body:
```json
{
  "window": "7d",
  "total_evaluations": 42,
  "avg_confidence": 0.81,
  "avg_disagreement_rate": 0.12
}
```
- `avg_disagreement_rate` is population std-dev of judge scores per evaluation, averaged across all evaluations (range 0–0.5)

### Test Case 23: Health Metrics - Custom Window

**Request:**
```bash
curl "http://localhost:18082/api/v1/metrics/health?window=24h"
curl "http://localhost:18082/api/v1/metrics/health?window=30d"
```

**Expected:**
- Status Code: 200
- `window` echoes the requested value
- Supported units: `h` (hours), `d` (days)

### Test Case 24: Health Metrics - Invalid Window

**Request:**
```bash
curl "http://localhost:18082/api/v1/metrics/health?window=7w"
curl "http://localhost:18082/api/v1/metrics/health?window=abc"
```

**Expected:**
- Status Code: 400
- Body: `{"error": "invalid window \"7w\": use format like 7d or 24h"}`

---

## Validation Sampling Tests

### Test Case 25: Download Sample - Basic

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/validation/sample/events/download \
  -H "Content-Type: application/json" \
  -d '{
    "start_date": "2020-01-01T00:00:00Z",
    "end_date": "2099-01-01T00:00:00Z",
    "percentage": 25
  }' \
  -o sample.jsonl
```

**Expected:**
- Status Code: 200
- `Content-Type: application/x-ndjson`
- `Content-Disposition` header with filename `sample-<timestamp>.jsonl`
- Body: one JSON evaluation object per line (JSONL format)
- Each line parseable as JSON with fields: `event_id`, `agent_name`, `confidence`, `verdict`, `stage_scores`

### Test Case 26: Download Sample - With Size Constraints

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/validation/sample/events/download \
  -H "Content-Type: application/json" \
  -d '{
    "start_date": "2020-01-01T00:00:00Z",
    "end_date": "2099-01-01T00:00:00Z",
    "percentage": 50,
    "min_size": 100,
    "max_size": 2500
  }' \
  -o sample.jsonl
```

**Expected:**
- Status Code: 200
- Line count between 100 and 2500 regardless of total records
- Percentage applied first, then min/max clamp

### Test Case 27: Download Sample - Missing Dates

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/validation/sample/events/download \
  -H "Content-Type: application/json" \
  -d '{"percentage": 25}'
```

**Expected:**
- Status Code: 400
- Body: `{"error": "start_date and end_date are required"}`

### Test Case 28: Download Sample - Invalid Date Format

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/validation/sample/events/download \
  -H "Content-Type: application/json" \
  -d '{"start_date": "2026-01-01", "end_date": "2026-03-31", "percentage": 25}'
```

**Expected:**
- Status Code: 400 (dates must be RFC3339 format with time and timezone)

### Test Case 29: Download Sample - Empty Result Set

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/validation/sample/events/download \
  -H "Content-Type: application/json" \
  -d '{
    "start_date": "2000-01-01T00:00:00Z",
    "end_date": "2000-12-31T23:59:59Z",
    "percentage": 25
  }' \
  -o sample.jsonl
```

**Expected:**
- Status Code: 200
- Empty body (no records in that date range)

---

## Next Steps

- [Batch Test Cases](batch-tests.md) - CLI evaluation testing
- [MCP Test Cases](mcp-tests.md) - Claude Code integration
- [Streaming Test Cases](streaming-tests.md) - Redis consumer testing
- [API Mode Deployment](../deployment/api-mode.md) - Deployment guide
- [Configuration](../getting-started/configuration.md) - Tune your setup
