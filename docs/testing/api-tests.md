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
go run cmd/api/main.go
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

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-008",
    "interaction": {
      "user_query": "Test"
    }
  }'
```

**Expected:**
- Status Code: 400
- Error message about missing required field

---

## Performance Tests

### Test Case 9: Concurrent Requests

**Script:**
```bash
# Send 10 concurrent requests
for i in {1..10}; do
  curl -X POST http://localhost:18082/api/v1/evaluate \
    -H "Content-Type: application/json" \
    -d "{\"event_id\":\"perf-$i\",\"event_type\":\"agent_response\",\"agent\":{\"name\":\"test\",\"type\":\"rag\",\"version\":\"1.0\"},\"interaction\":{\"user_query\":\"Test\",\"context\":\"Test\",\"answer\":\"Test\"}}" &
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

## Next Steps

- [Batch Test Cases](batch-tests.md) - CLI evaluation testing
- [MCP Test Cases](mcp-tests.md) - Claude Code integration
- [Streaming Test Cases](streaming-tests.md) - Redis consumer testing
- [API Mode Deployment](../deployment/api-mode.md) - Deployment guide
- [Configuration](../getting-started/configuration.md) - Tune your setup
