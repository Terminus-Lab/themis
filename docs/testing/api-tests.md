---
title: API Test Cases
description: Comprehensive test scenarios for Themis HTTP API endpoints
version: 2.0.0
tags: [testing, api, http, test-cases, endpoints]
related:
  - deployment/api-mode.md
  - testing/batch-tests.md
  - testing/mcp-tests.md
  - testing/streaming-tests.md
---

# API Test Cases

Comprehensive test scenarios for the Themis HTTP API.

**Important:** Expected responses show **representative results**. Actual LLM judge scores may vary slightly (±0.1) due to model variability, but overall patterns (pass/review/fail verdicts, score ranges) should match.

## Setup

### Prerequisites

Ensure your `.env` file is configured:

```env
OPEN_AI_KEY=sk-proj-...        # or AWS/Azure credentials
EVAL_AGENT_API_PORT=18082
CONVERSATION_HOLISTIC_WEIGHT=0.5
```

### Start the Server

```bash
go run cmd/api/main.go
```

Server runs on `http://localhost:18082`

**Expected startup logs:**
```
INFO judge created successfully judge=relevance scope=turn
INFO judge created successfully judge=coherence scope=turn
INFO judge created successfully judge=completeness scope=turn
INFO judge created successfully judge=conversation-flow scope=conversation
INFO judge pool built successfully total_judges=3 scope=turn
```

---

## Health Check

### Test Case 1: Health Endpoint

**Request:**
```bash
curl http://localhost:18082/api/v1/health
```

**Expected:**
- Status Code: 200
- Body: `{"status": "ok"}`

---

## Conversation Evaluation Tests

### Test Case 2: Single-Turn Conversation — High Quality

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "test-001",
    "agent": {"name": "test-agent", "version": "1.0"},
    "turns": [
      {
        "turn_index": 1,
        "user_query": "What is the capital of France?",
        "answer": "The capital of France is Paris."
      }
    ]
  }'
```

**Expected:**
- Status Code: 200
- `final_score` > 0.8
- `verdict` = "pass"
- `turn_results` has 1 entry with high scores
- Response time: ~3-4 seconds

### Test Case 3: Multi-Turn Conversation — Pass

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "test-002",
    "agent": {"name": "test-agent", "version": "1.0"},
    "turns": [
      {
        "turn_index": 1,
        "user_query": "What is the capital of France?",
        "answer": "The capital of France is Paris."
      },
      {
        "turn_index": 2,
        "user_query": "And what is the capital of Germany?",
        "answer": "The capital of Germany is Berlin."
      }
    ]
  }'
```

**Expected:**
- Status Code: 200
- `turn_avg` > 0.8
- `holistic_score` > 0.8 (conversation flows well)
- `final_score` > 0.8
- `verdict` = "pass"
- `turn_results` has 2 entries

**Sample response structure:**
```json
{
  "conversation_id": "test-002",
  "agent_name": "test-agent",
  "agent_version": "1.0",
  "turn_count": 2,
  "turn_results": [
    {
      "turn_index": 1,
      "user_query": "What is the capital of France?",
      "answer": "The capital of France is Paris.",
      "turn_score": 0.92,
      "scores": [
        {"name": "relevance", "score": 0.95, "reason": "..."},
        {"name": "coherence", "score": 0.90, "reason": "..."},
        {"name": "completeness", "score": 0.90, "reason": "..."}
      ]
    },
    {
      "turn_index": 2,
      "user_query": "And what is the capital of Germany?",
      "answer": "The capital of Germany is Berlin.",
      "turn_score": 0.91,
      "scores": [...]
    }
  ],
  "turn_avg": 0.915,
  "holistic_score": 0.88,
  "holistic_reason": "The conversation flows naturally...",
  "final_score": 0.897,
  "verdict": "pass"
}
```

### Test Case 4: Low Quality Answer — Fail

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "test-003",
    "agent": {"name": "test-agent", "version": "1.0"},
    "turns": [
      {
        "turn_index": 1,
        "user_query": "What are the main causes of climate change?",
        "answer": "There are various factors."
      }
    ]
  }'
```

**Expected:**
- Status Code: 200
- `final_score` < 0.5
- `verdict` = "fail"
- Low `turn_score` (vague, incomplete answer)

---

## Validation Error Tests

### Test Case 5: Missing conversation_id

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{"turns":[{"turn_index":1,"user_query":"hi","answer":"hello"}]}'
```

**Expected:**
- Status Code: 400
- Body: `{"error": "conversation_id is required"}`

### Test Case 6: Empty turns

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{"conversation_id":"conv-1","turns":[]}'
```

**Expected:**
- Status Code: 400
- Body: `{"error": "turns must not be empty"}`

---

## Conversation Query Tests

### Test Case 7: List Conversations — Empty

**Request:**
```bash
curl "http://localhost:18082/api/v1/conversations"
```

**Expected:**
- Status Code: 200
- Body: `{"conversations": [], "total": 0}`

### Test Case 8: Get Conversation by ID — Not Found

**Request:**
```bash
curl "http://localhost:18082/api/v1/conversations/does-not-exist"
```

**Expected:**
- Status Code: 404

### Test Case 9: Get Conversation by ID — Found

After submitting an evaluation for `conversation_id: "conv-abc"`:

**Request:**
```bash
curl "http://localhost:18082/api/v1/conversations/conv-abc"
```

**Expected:**
- Status Code: 200
- Body includes `conversation_id`, `turn_count`, `turn_results`, `final_score`, `verdict`

---

## Health Metrics Tests

### Test Case 10: Health Metrics — Default Window

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
  "total_evaluations": 0,
  "avg_confidence": 0
}
```

### Test Case 11: Health Metrics — Custom Window

**Request:**
```bash
curl "http://localhost:18082/api/v1/metrics/health?window=1d"
curl "http://localhost:18082/api/v1/metrics/health?window=30d"
```

**Expected:**
- Status Code: 200
- `window` echoes the requested value

### Test Case 12: Health Metrics — Invalid Window

**Request:**
```bash
curl "http://localhost:18082/api/v1/metrics/health?window=invalid"
curl "http://localhost:18082/api/v1/metrics/health?window=7w"
```

**Expected:**
- Status Code: 400
- Body: `{"error": "invalid window ..."}`

---

## Performance Tests

### Test Case 13: Concurrent Requests

```bash
for i in {1..5}; do
  curl -s -X POST http://localhost:18082/api/v1/conversations/evaluate \
    -H "Content-Type: application/json" \
    -d "{\"conversation_id\":\"perf-$i\",\"agent\":{\"name\":\"test\",\"version\":\"1.0\"},\"turns\":[{\"turn_index\":1,\"user_query\":\"What is Go?\",\"answer\":\"A programming language.\"}]}" &
done
wait
```

**Expected:**
- All 5 requests complete successfully
- Each response contains valid `final_score` and `verdict`
- No race conditions

---

## Next Steps

- [Batch Test Cases](batch-tests.md) - CLI evaluation testing
- [MCP Test Cases](mcp-tests.md) - Claude Code integration
- [Streaming Test Cases](streaming-tests.md) - Redis consumer testing
- [Configuration](../getting-started/configuration.md) - Tune your setup
