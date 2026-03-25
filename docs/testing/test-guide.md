---
title: Themis Test Guide
description: Comprehensive test cases for all Themis features — API, CLI, MCP, Streaming, and configuration
version: 2.0.0
tags: [testing, api, cli, mcp, streaming, configuration]
---

# Themis Test Guide

Complete test coverage for every feature: API endpoints, batch CLI, MCP tools, Redis streaming, judge configuration, and the dashboard.

> **Score variability notice:** LLM judge scores vary by ±0.05–0.10 between runs. Test for ranges and verdict labels, not exact values.

---

## Prerequisites

### 1. Build binaries

```bash
go build -o bin/themis-api cmd/api/main.go
go build -o bin/themis-cli cmd/batch/main.go
go build -o bin/themis-mcp cmd/mcp/main.go
go build -o bin/themis-producer cmd/producer/main.go
```

### 2. Configure credentials

```bash
cp .env.example .env
# Edit .env — minimum required: OPEN_AI_KEY or AWS credentials
```

Minimum `.env`:
```env
OPEN_AI_KEY=sk-proj-...
EVAL_AGENT_API_PORT=18082
CONVERSATION_HOLISTIC_WEIGHT=0.5
VERDICT_PASS_THRESHOLD=0.8
VERDICT_REVIEW_THRESHOLD=0.5
IN_MEMORY_DB=true
```

### 3. Verify dependencies

```bash
# Go version
go version   # must be 1.24+

# jq (for JSON formatting in examples)
jq --version

# Redis (for streaming tests only)
redis-cli PING  # should return PONG
```

---

## Part 1 — Unit & Integration Tests

These run against your local codebase with no external services required.

### Run all tests

```bash
go test ./...
```

**Expected:** all packages pass, zero failures.

### Run with verbose output

```bash
go test -v ./internal/...
```

### Run with coverage

```bash
go test -cover ./...
```

### Run specific packages

```bash
go test ./internal/api/...       # API handler tests
go test ./internal/judge/...     # Judge pool tests
go test ./internal/config/...    # Config loading and validation
go test ./internal/storage/sqlite/...  # Storage layer tests
go test ./internal/batch/...     # Batch processing tests
```

### Test Case 1.1 — API handler unit tests

```bash
go test -v ./internal/api/...
```

**Verified:**
- Health check returns `{"status":"ok","version":"2.0.0"}`
- List conversations returns empty list `{"conversations":[],"total":0}`
- Get conversation by unknown ID returns 404
- Evaluate with missing `conversation_id` returns 400
- Evaluate with empty `turns` returns 400
- Health metrics with default window returns valid JSON

### Test Case 1.2 — Storage unit tests

```bash
go test -v ./internal/storage/sqlite/...
```

**Verified:**
- Store and retrieve conversation round-trip
- Get conversation not found returns `sql.ErrNoRows`
- List conversations returns ordered results
- Health metrics empty database returns zeros
- Health metrics with data returns correct totals and averages
- Duplicate conversation_id stores (latest wins)

### Test Case 1.3 — Judge pool unit tests

```bash
go test -v ./internal/judge/...
```

**Verified:**
- `BuildTurnJudgesFromConfig` returns 3 turn-scoped judges from default config
- Disabled judge is skipped
- Config with no enabled turn judges returns error
- Invalid prompt template returns error

### Test Case 1.4 — Config unit tests

```bash
go test -v ./internal/config/...
```

**Verified:**
- Loads `configs/judges.yaml` successfully
- `JUDGES_CONFIG_PATH` override loads alternate file
- Missing config file returns descriptive error
- Weight normalization: turn judges sum to 1.0
- Weight normalization: conversation judges sum to 1.0

### Test Case 1.5 — Integration test (requires LLM key)

```bash
OPEN_AI_KEY=sk-proj-... go test -v -run Integration ./internal/api/...
```

**Verified:**
- Full evaluation pipeline executes against real LLM
- Scores are in range [0.0, 1.0]
- Verdict is one of "pass", "review", "fail"

---

## Part 2 — API Tests

Start the server before running these tests:

```bash
./bin/themis-api
# Expected startup logs:
# INFO judge created successfully judge=relevance scope=turn
# INFO judge created successfully judge=coherence scope=turn
# INFO judge created successfully judge=completeness scope=turn
# INFO judge created successfully judge=conversation-flow scope=conversation
# INFO Starting Themis Server address=:18082
```

---

### Health

#### Test Case 2.1 — Server health check

```bash
curl -s http://localhost:18082/api/v1/health | jq .
```

**Expected response (200):**
```json
{
  "status": "ok",
  "version": "2.0.0"
}
```

---

### Evaluate Conversation

#### Test Case 2.2 — Single-turn, high quality answer

```bash
curl -s -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "tc-001",
    "agent": {"name": "test-agent", "version": "1.0"},
    "turns": [
      {
        "turn_index": 1,
        "user_query": "What is the capital of France?",
        "answer": "The capital of France is Paris."
      }
    ]
  }' | jq .
```

**Expected (200):**
- `final_score` > 0.80
- `verdict` = `"pass"`
- `turn_results` has exactly 1 entry
- Each entry has `scores` array with `relevance`, `coherence`, `completeness`
- `holistic_score` present (conversation-flow judge ran)
- Response time: 3–6 seconds

#### Test Case 2.3 — Multi-turn conversation, good quality

```bash
curl -s -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "tc-002",
    "agent": {"name": "test-agent", "version": "1.0"},
    "turns": [
      {
        "turn_index": 1,
        "user_query": "What is the capital of France?",
        "answer": "The capital of France is Paris."
      },
      {
        "turn_index": 2,
        "user_query": "What is it known for?",
        "answer": "Paris is known for the Eiffel Tower, the Louvre museum, and world-class cuisine."
      },
      {
        "turn_index": 3,
        "user_query": "What is the population?",
        "answer": "Paris has approximately 2.1 million people in the city proper, and over 12 million in the greater metropolitan area."
      }
    ]
  }' | jq '{verdict, final_score, turn_avg, holistic_score, turn_count}'
```

**Expected (200):**
- `turn_count` = 3
- `turn_avg` > 0.80
- `holistic_score` > 0.80 (coherent multi-turn flow)
- `final_score` > 0.80
- `verdict` = `"pass"`
- `turn_results` has 3 entries

#### Test Case 2.4 — Single-turn, vague answer (fail)

```bash
curl -s -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "tc-003",
    "agent": {"name": "test-agent", "version": "1.0"},
    "turns": [
      {
        "turn_index": 1,
        "user_query": "Explain quantum computing in detail.",
        "answer": "Yes."
      }
    ]
  }' | jq '{verdict, final_score}'
```

**Expected (200):**
- `final_score` < 0.50
- `verdict` = `"fail"`

#### Test Case 2.5 — Single-turn, partial answer (review)

```bash
curl -s -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "tc-004",
    "agent": {"name": "test-agent", "version": "1.0"},
    "turns": [
      {
        "turn_index": 1,
        "user_query": "Explain the causes and effects of climate change.",
        "answer": "Climate change is caused by greenhouse gases."
      }
    ]
  }' | jq '{verdict, final_score}'
```

**Expected (200):**
- `final_score` between 0.50 and 0.80
- `verdict` = `"review"`

#### Test Case 2.6 — Turn with context field

```bash
curl -s -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "tc-005",
    "agent": {"name": "rag-agent", "version": "2.0"},
    "turns": [
      {
        "turn_index": 1,
        "user_query": "What did the CEO say about Q3?",
        "answer": "The CEO reported a 15% revenue increase in Q3, exceeding analyst expectations.",
        "context": "Earnings call transcript: CEO stated Q3 revenue grew 15% year-over-year, beating analyst consensus by 3 percentage points."
      }
    ]
  }' | jq '{verdict, final_score}'
```

**Expected (200):**
- Valid response with scores
- `final_score` > 0.70 (answer matches context)

#### Test Case 2.7 — Missing `conversation_id`

```bash
curl -s -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "turns": [{"turn_index": 1, "user_query": "hi", "answer": "hello"}]
  }' | jq .
```

**Expected (400):**
```json
{"error": "conversation_id is required"}
```

#### Test Case 2.8 — Empty turns array

```bash
curl -s -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{"conversation_id": "x", "turns": []}' | jq .
```

**Expected (400):**
```json
{"error": "turns must not be empty"}
```

#### Test Case 2.9 — Invalid JSON body

```bash
curl -s -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{bad json}' | jq .
```

**Expected (400):** error about JSON parsing.

#### Test Case 2.10 — Agent without version

```bash
curl -s -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "tc-010",
    "agent": {"name": "minimal-agent"},
    "turns": [
      {"turn_index": 1, "user_query": "What is 2+2?", "answer": "4"}
    ]
  }' | jq '{conversation_id, agent_name, agent_version, verdict}'
```

**Expected (200):**
- `agent_name` = `"minimal-agent"`
- `agent_version` = `""` (empty string is fine)
- Evaluation completes normally

---

### Query Conversations

#### Test Case 2.11 — List conversations (empty state)

Start a fresh server (in-memory DB), then:

```bash
curl -s http://localhost:18082/api/v1/conversations | jq .
```

**Expected (200):**
```json
{
  "conversations": [],
  "total": 0
}
```

#### Test Case 2.12 — List conversations (after evaluations)

After running test cases 2.2 and 2.3:

```bash
curl -s http://localhost:18082/api/v1/conversations | jq '{total, conversations: [.conversations[] | {conversation_id, verdict, final_score}]}'
```

**Expected (200):**
- `total` >= 2
- Each item has `conversation_id`, `agent_name`, `turn_count`, `final_score`, `verdict`, `created_at`

#### Test Case 2.13 — Get conversation by ID — found

After evaluating `tc-002`:

```bash
curl -s http://localhost:18082/api/v1/conversations/tc-002 | jq .
```

**Expected (200):**
- Full detail response including `turn_results` with per-turn `scores`
- `turn_count` = 3
- `holistic_reason` is a non-empty string

#### Test Case 2.14 — Get conversation by ID — not found

```bash
curl -s http://localhost:18082/api/v1/conversations/does-not-exist
```

**Expected (404):**
```json
{"error": "conversation not found"}
```

---

### Health Metrics

#### Test Case 2.15 — Default window (7d)

```bash
curl -s "http://localhost:18082/api/v1/metrics/health" | jq .
```

**Expected (200):**
```json
{
  "window": "7d",
  "total_evaluations": <N>,
  "avg_confidence": <float>
}
```

#### Test Case 2.16 — All valid window formats

```bash
curl -s "http://localhost:18082/api/v1/metrics/health?window=1d" | jq .window
curl -s "http://localhost:18082/api/v1/metrics/health?window=7d" | jq .window
curl -s "http://localhost:18082/api/v1/metrics/health?window=30d" | jq .window
curl -s "http://localhost:18082/api/v1/metrics/health?window=24h" | jq .window
curl -s "http://localhost:18082/api/v1/metrics/health?window=48h" | jq .window
```

**Expected:** each returns 200 with `window` echoing the requested value.

#### Test Case 2.17 — Invalid window formats

```bash
curl -s "http://localhost:18082/api/v1/metrics/health?window=invalid"
curl -s "http://localhost:18082/api/v1/metrics/health?window=7w"
curl -s "http://localhost:18082/api/v1/metrics/health?window=0d"
curl -s "http://localhost:18082/api/v1/metrics/health?window=-1d"
```

**Expected (400):** `{"error": "invalid window ..."}` for each.

---

### Performance

#### Test Case 2.18 — Concurrent requests

```bash
for i in {1..5}; do
  curl -s -X POST http://localhost:18082/api/v1/conversations/evaluate \
    -H "Content-Type: application/json" \
    -d "{
      \"conversation_id\": \"concurrent-$i\",
      \"agent\": {\"name\": \"perf-test\", \"version\": \"1.0\"},
      \"turns\": [{\"turn_index\": 1, \"user_query\": \"What is $i squared?\", \"answer\": \"$((i*i))\"}]
    }" &
done
wait
echo "All 5 requests completed"
```

**Expected:**
- All 5 complete without error
- Each returns a valid JSON response with `verdict`

#### Test Case 2.19 — Verify all are stored

```bash
curl -s "http://localhost:18082/api/v1/conversations" | jq '.total'
```

**Expected:** total includes all `concurrent-1` through `concurrent-5`.

---

## Part 3 — Batch CLI Tests

### Test Case 3.1 — Basic batch evaluation

```bash
cat > /tmp/conversations.jsonl << 'EOF'
{"conversation_id":"batch-001","agent":{"name":"test-agent","version":"1.0"},"turns":[{"turn_index":1,"user_query":"What is the capital of France?","answer":"The capital of France is Paris."},{"turn_index":2,"user_query":"And Germany?","answer":"The capital of Germany is Berlin."}]}
{"conversation_id":"batch-002","agent":{"name":"test-agent","version":"1.0"},"turns":[{"turn_index":1,"user_query":"What is 2+2?","answer":"4"}]}
{"conversation_id":"batch-003","agent":{"name":"test-agent","version":"1.0"},"turns":[{"turn_index":1,"user_query":"Explain quantum computing.","answer":"Yes."}]}
EOF

./bin/themis-cli evaluate \
  -input /tmp/conversations.jsonl \
  -output /tmp/results.jsonl
```

**Expected:**
- Exit code: 0
- `/tmp/results.jsonl` has exactly 3 lines
- Each line is a valid JSON object

**Verify:**
```bash
wc -l /tmp/results.jsonl   # should be 3
jq -r '.verdict' /tmp/results.jsonl
# expected: pass (or review), pass, fail
```

#### Test Case 3.2 — Verify output structure

```bash
jq '{conversation_id, agent_name, turn_count, turn_avg, holistic_score, final_score, verdict}' /tmp/results.jsonl
```

**Expected:** each result has all 7 fields populated.

#### Test Case 3.3 — Inspect turn-level results

```bash
jq '.turn_results[] | {turn_index, turn_score, judge_count: (.scores | length)}' /tmp/results.jsonl
```

**Expected:** each turn has a `turn_score` and 3 judge scores.

### Test Case 3.4 — Custom worker count

```bash
THEMIS_BATCH_WORKERS=2 ./bin/themis-cli evaluate \
  -input /tmp/conversations.jsonl \
  -output /tmp/results-2w.jsonl
```

**Expected:**
- Exit code: 0
- Same output as test 3.1 (same data, just processed with 2 workers)

```bash
diff <(jq -c 'del(.holistic_reason)' /tmp/results.jsonl | sort) \
     <(jq -c 'del(.holistic_reason)' /tmp/results-2w.jsonl | sort)
# scores may differ slightly due to LLM variability — verdicts should match
```

### Test Case 3.5 — Large dataset

```bash
python3 -c "
import json
for i in range(20):
    print(json.dumps({
        'conversation_id': f'large-{i:03d}',
        'agent': {'name': 'load-test', 'version': '1.0'},
        'turns': [
            {'turn_index': 1, 'user_query': f'What is the capital of country {i}?', 'answer': f'The capital is city {i}.'}
        ]
    }))
" > /tmp/large.jsonl

THEMIS_BATCH_WORKERS=5 ./bin/themis-cli evaluate \
  -input /tmp/large.jsonl \
  -output /tmp/large-results.jsonl
```

**Expected:**
- Exit code: 0
- Exactly 20 result lines
- No duplicate `conversation_id` in output

```bash
wc -l /tmp/large-results.jsonl          # 20
jq -r '.conversation_id' /tmp/large-results.jsonl | sort -u | wc -l  # 20
```

### Test Case 3.6 — Empty input file

```bash
touch /tmp/empty.jsonl
./bin/themis-cli evaluate -input /tmp/empty.jsonl -output /tmp/empty-results.jsonl
echo "exit: $?"
```

**Expected:**
- Exit code: 0
- Output file is empty or does not error

### Test Case 3.7 — Invalid JSON in input (resilience)

```bash
cat > /tmp/mixed.jsonl << 'EOF'
{"conversation_id":"good-001","agent":{"name":"test","version":"1.0"},"turns":[{"turn_index":1,"user_query":"What is Go?","answer":"A programming language."}]}
{this is not json}
{"conversation_id":"good-002","agent":{"name":"test","version":"1.0"},"turns":[{"turn_index":1,"user_query":"What is Rust?","answer":"A systems programming language."}]}
EOF

./bin/themis-cli evaluate \
  -input /tmp/mixed.jsonl \
  -output /tmp/mixed-results.jsonl
echo "exit: $?"
```

**Expected:**
- Exit code: 0
- `/tmp/mixed-results.jsonl` contains 2 lines (bad line is skipped)
- Warning logged for the bad line

### Test Case 3.8 — Missing required flags

```bash
./bin/themis-cli evaluate -input /tmp/conversations.jsonl
echo "exit: $?"
```

**Expected:**
- Exit code: non-zero
- Error message: missing `-output` flag

```bash
./bin/themis-cli evaluate -output /tmp/out.jsonl
echo "exit: $?"
```

**Expected:**
- Exit code: non-zero
- Error message: missing `-input` flag

### Test Case 3.9 — Non-existent input file

```bash
./bin/themis-cli evaluate \
  -input /tmp/does-not-exist.jsonl \
  -output /tmp/out.jsonl
echo "exit: $?"
```

**Expected:** non-zero exit code, file-not-found error.

### Test Case 3.10 — Verdict distribution analysis

```bash
jq -r '.verdict' /tmp/results.jsonl | sort | uniq -c
```

**Expected:** at least one `pass`, at least one `fail` from the mixed-quality dataset.

---

## Part 4 — MCP Tests

### Setup

```bash
# Add Themis to Claude Code
claude mcp add --transport stdio --scope project themis \
  --env OPEN_AI_KEY=sk-proj-... \
  -- /path/to/themis/bin/themis-mcp

# Verify registration
claude mcp list
```

**Expected output from `claude mcp list`:**
```
themis (stdio) - Ready
```

---

### Tool Discovery

#### Test Case 4.1 — List available tools

In a Claude Code session, run:
```
/mcp
```

**Expected:**
```
MCP Servers:
- themis (stdio) - Ready
  Tools:
  - evaluate_conversation: Evaluate a multi-turn conversation...
  - get_conversation: Retrieve a stored conversation evaluation...
```

**Verification:**
- Server status is `Ready` (not `Error` or `Connecting`)
- Both tools are listed with descriptions

---

### evaluate_conversation

#### Test Case 4.2 — Single-turn evaluation

**Prompt in Claude Code:**
```
Use the evaluate_conversation tool to evaluate this:
Conversation ID: mcp-001
Agent: my-agent v1.0
Turn 1: Query="What is the capital of France?" Answer="The capital of France is Paris."
```

**Expected tool call:**
```json
{
  "conversation_id": "mcp-001",
  "agent_name": "my-agent",
  "agent_version": "1.0",
  "turns": [
    {"turn_index": 1, "user_query": "What is the capital of France?", "answer": "The capital of France is Paris."}
  ]
}
```

**Expected result:**
- `final_score` > 0.80
- `verdict` = `"pass"`
- Claude summarizes the result in natural language

#### Test Case 4.3 — Multi-turn evaluation

**Prompt:**
```
Evaluate this 3-turn conversation:
Conversation ID: mcp-002
Agent: assistant v1.0
Turn 1: Q="What is the capital of France?" A="Paris is the capital of France."
Turn 2: Q="What is it known for?" A="Paris is famous for the Eiffel Tower, the Louvre, and world-class cuisine."
Turn 3: Q="What is the population?" A="Paris has approximately 2.1 million people in the city proper."
```

**Expected result:**
- `turn_count` = 3
- `turn_results` has 3 entries with individual turn scores
- `holistic_score` present (conversation flow evaluated)
- `verdict` = `"pass"`
- Claude presents all 3 turn scores and the holistic summary

#### Test Case 4.4 — Low quality answer

**Prompt:**
```
Evaluate:
Conversation ID: mcp-003
Turn 1: Q="Explain quantum computing in detail." A="Yes."
```

**Expected result:**
- `final_score` < 0.30
- `verdict` = `"fail"`
- Claude explains why the answer failed (incomplete, irrelevant)

#### Test Case 4.5 — Hallucination detection

**Prompt:**
```
Evaluate this for quality:
Conversation ID: mcp-004
Turn 1: Q="What is the population of Tokyo?" A="Tokyo has 50 million people and is the largest city in China."
```

**Expected result:**
- Low scores on relevance and coherence (contradictory claims)
- `verdict` = `"fail"`
- Claude flags the factual error in the answer

#### Test Case 4.6 — Missing conversation_id

**Prompt:**
```
Use evaluate_conversation but only provide turns, no conversation_id:
Turn 1: Q="What is Go?" A="A programming language."
```

**Expected behavior:**
- Tool call fails with `"conversation_id is required"`
- Claude asks the user to provide the missing field

#### Test Case 4.7 — Empty turns

**Prompt:**
```
Evaluate conversation mcp-empty with no turns.
```

**Expected behavior:**
- Tool call fails with `"turns must not be empty"`
- Claude explains what is required

---

### get_conversation

#### Test Case 4.8 — Retrieve stored conversation

After evaluating `mcp-002` in test 4.3:

**Prompt:**
```
Use get_conversation to retrieve conversation mcp-002
```

**Expected result:**
- Full conversation record returned
- `turn_count` = 3
- `turn_results` with per-turn scores
- `final_score` matches what was returned by evaluate

#### Test Case 4.9 — Conversation not found

**Prompt:**
```
Get conversation mcp-nonexistent
```

**Expected behavior:**
- Error returned: `"failed to get conversation: ..."`
- Claude explains the conversation was not found

---

### Integration Scenarios

#### Test Case 4.10 — Sequential evaluate then retrieve

**Prompt:**
```
1. Evaluate conversation mcp-seq: Turn 1: Q="What is Go?" A="Go is a statically typed language by Google."
2. Then retrieve conversation mcp-seq and compare the scores.
```

**Expected:**
- `evaluate_conversation` succeeds, returns scores
- `get_conversation` retrieves the same record
- Claude confirms scores match

#### Test Case 4.11 — Batch evaluation in one prompt

**Prompt:**
```
Evaluate these 3 separate conversations:
1. mcp-a: Q="What is 2+2?" A="4"
2. mcp-b: Q="Capital of Spain?" A="Madrid"
3. mcp-c: Q="Who wrote Hamlet?" A="Shakespeare"
Then summarize which had the highest score.
```

**Expected:**
- 3 separate `evaluate_conversation` tool calls
- All return `"pass"` verdicts
- Claude compares and summarizes scores

---

### Docker MCP

#### Test Case 4.12 — Docker-based MCP server

```bash
docker build -t themis-mcp .

claude mcp add --transport stdio --scope project themis-docker \
  --env OPEN_AI_KEY=sk-proj-... \
  -- docker run -i --rm -e OPEN_AI_KEY themis-mcp:latest
```

**Test:** Run test case 4.2 using `themis-docker` server.

**Expected:** Identical results to the binary-based server.

---

## Part 5 — Streaming Tests (Redis Consumer)

### Setup

**Start Redis:**
```bash
redis-server
redis-cli PING  # PONG
```

**Add to `.env`:**
```env
CONVERSATION_STREAMING_ENABLED=true
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_CONVERSATION_STREAM_KEY=eval-conversations
REDIS_CONVERSATION_GROUP=eval-conv-group
REDIS_CONSUMER_NAME=consumer-1
```

**Start the API server with streaming enabled:**
```bash
CONVERSATION_STREAMING_ENABLED=true ./bin/themis-api
```

**Expected startup logs:**
```
INFO Streaming mode enabled - starting conversation Redis consumer
INFO Starting streaming consumer stream_key=eval-conversations consumer_group=eval-conv-group
INFO Starting Themis Server address=:18082
```

---

### Test Case 5.1 — Basic message consumption

```bash
./bin/themis-producer -d '{
  "conversation_id": "stream-001",
  "agent": {"name": "test-agent", "version": "1.0"},
  "turns": [
    {"turn_index": 1, "user_query": "What is the capital of France?", "answer": "The capital of France is Paris."},
    {"turn_index": 2, "user_query": "And Germany?", "answer": "The capital of Germany is Berlin."}
  ]
}'
```

**Expected producer output:**
```
Published successfully!
```

**Expected consumer logs:**
```
INFO starting conversation evaluation conversation_id=stream-001 turn_count=2
INFO conversation evaluation complete final_score=0.90 verdict=pass
```

**Verify message was acknowledged:**
```bash
redis-cli XPENDING eval-conversations eval-conv-group - + 10
# Expected: empty (0 pending — message processed and acked)
```

### Test Case 5.2 — Single-turn message

```bash
./bin/themis-producer -d '{
  "conversation_id": "stream-002",
  "agent": {"name": "test-agent", "version": "1.0"},
  "turns": [
    {"turn_index": 1, "user_query": "What is 2+2?", "answer": "2+2 equals 4."}
  ]
}'
```

**Expected consumer log:**
```
INFO conversation evaluation complete conversation_id=stream-002 verdict=pass
```

### Test Case 5.3 — Low quality answer

```bash
./bin/themis-producer -d '{
  "conversation_id": "stream-003",
  "agent": {"name": "test-agent", "version": "1.0"},
  "turns": [
    {"turn_index": 1, "user_query": "Explain quantum computing in detail.", "answer": "Yes."}
  ]
}'
```

**Expected consumer log:**
```
INFO conversation evaluation complete conversation_id=stream-003 verdict=fail
```

### Test Case 5.4 — Multiple concurrent messages

```bash
for i in {1..10}; do
  ./bin/themis-producer -d "{
    \"conversation_id\": \"concurrent-$i\",
    \"agent\": {\"name\": \"test-agent\", \"version\": \"1.0\"},
    \"turns\": [{\"turn_index\": 1, \"user_query\": \"Test query $i\", \"answer\": \"Test answer $i\"}]
  }"
done
```

**Wait ~60 seconds for processing, then verify:**
```bash
redis-cli XPENDING eval-conversations eval-conv-group - + 20
# Expected: empty (all 10 processed and acked)

redis-cli XINFO GROUPS eval-conversations
# pel-count (pending entries) should be 0
```

### Test Case 5.5 — Invalid JSON payload

```bash
redis-cli XADD eval-conversations '*' payload '{invalid json}'
```

**Expected consumer logs:**
```
ERROR Failed to parse stream message error="invalid character..."
INFO Message acknowledged (parse error)
```

**Verification:**
- Consumer continues running (no crash)
- Message acknowledged (not redelivered)
- API still responds: `curl http://localhost:18082/api/v1/health`

### Test Case 5.6 — API and streaming running concurrently

While the streaming consumer is running and processing messages:

```bash
# Simultaneously: fire API request
curl -s -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "api-while-streaming",
    "agent": {"name": "test"},
    "turns": [{"turn_index": 1, "user_query": "What is AI?", "answer": "AI is artificial intelligence."}]
  }' | jq '{verdict, final_score}'
```

**Expected:**
- HTTP request completes synchronously with valid response
- Streaming consumer continues processing Redis messages
- No interference between the two paths

### Test Case 5.7 — Graceful shutdown

1. Start consumer and send 5 messages
2. Press Ctrl+C while messages are processing

**Expected logs:**
```
INFO Received shutdown signal, finishing current work...
INFO Streaming consumer stopped
INFO Server shutdown complete
```

**Verify:**
```bash
redis-cli XPENDING eval-conversations eval-conv-group - + 10
# Should be 0 — in-flight messages completed before shutdown
```

### Test Case 5.8 — Consumer restart (message redelivery)

1. Start consumer: `CONVERSATION_STREAMING_ENABLED=true ./bin/themis-api`
2. Send one message
3. Kill the consumer abruptly: `kill -9 <pid>`
4. Check pending:
   ```bash
   redis-cli XPENDING eval-conversations eval-conv-group - + 10
   # Should show 1 pending message
   ```
5. Restart consumer
6. After processing, check pending:
   ```bash
   redis-cli XPENDING eval-conversations eval-conv-group - + 10
   # Should be 0 — message redelivered and processed
   ```

### Test Case 5.9 — Horizontal scaling (multiple consumers)

```bash
# Start 3 consumers on different ports
CONVERSATION_STREAMING_ENABLED=true REDIS_CONSUMER_NAME=worker-1 EVAL_AGENT_API_PORT=18082 ./bin/themis-api &
CONVERSATION_STREAMING_ENABLED=true REDIS_CONSUMER_NAME=worker-2 EVAL_AGENT_API_PORT=18083 ./bin/themis-api &
CONVERSATION_STREAMING_ENABLED=true REDIS_CONSUMER_NAME=worker-3 EVAL_AGENT_API_PORT=18084 ./bin/themis-api &

# Send 30 messages
for i in {1..30}; do
  ./bin/themis-producer -d "{
    \"conversation_id\": \"scale-$i\",
    \"agent\": {\"name\": \"scale-test\", \"version\": \"1.0\"},
    \"turns\": [{\"turn_index\": 1, \"user_query\": \"Query $i\", \"answer\": \"Answer $i\"}]
  }"
done
```

**Wait for processing, then verify:**
```bash
redis-cli XINFO CONSUMERS eval-conversations eval-conv-group
# Should show 3 consumers

redis-cli XPENDING eval-conversations eval-conv-group - + 50
# Should be 0 — all 30 messages processed, no duplicates
```

### Test Case 5.10 — Redis connection failure recovery

1. Start consumer with Redis running
2. Stop Redis: `redis-cli SHUTDOWN NOSAVE` (or `brew services stop redis`)
3. Observe consumer logs:
   ```
   ERROR Redis connection lost error="connection refused"
   ERROR Failed to read from stream, retrying...
   ```
4. Verify API still responds:
   ```bash
   curl http://localhost:18082/api/v1/health
   # {"status":"ok",...} — HTTP server unaffected
   ```
5. Restart Redis: `redis-server`
6. Observe consumer resumes processing

---

## Part 6 — Judge Configuration Tests

These test the `enabled` flag in `configs/judges.yaml`.

### Test Case 6.1 — Disable a judge (coherence)

Edit `configs/judges.yaml`:
```yaml
- name: coherence
  enabled: false   # ← change this
```

Restart the server. **Expected startup log:**
```
INFO judge disabled in config, skipping judge=coherence
INFO judge pool built successfully total_judges=2 scope=turn
```

Evaluate a conversation:
```bash
curl -s -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "cfg-001",
    "agent": {"name": "test"},
    "turns": [{"turn_index": 1, "user_query": "What is Go?", "answer": "A programming language."}]
  }' | jq '.turn_results[0].scores | map(.name)'
```

**Expected:** `["relevance", "completeness"]` — no `coherence` entry.

**Verify weight normalization:**
```bash
jq '.turn_results[0].scores | map({name, weight})' response.json
```
**Expected:** `relevance` and `completeness` weights re-normalized to sum to 1.0 (e.g. 0.50 each instead of 0.35 each).

Restore `enabled: true` and restart.

### Test Case 6.2 — Run with only one turn judge

Set relevance and completeness to `enabled: false`, keep only coherence:

```yaml
- name: relevance
  enabled: false
- name: coherence
  enabled: true
- name: completeness
  enabled: false
```

Restart server. **Expected:**
- `total_judges=1 scope=turn`
- Each turn has exactly 1 judge score in the response

Restore config.

### Test Case 6.3 — Disable holistic judge

Set `conversation-flow` to `enabled: false`:

```yaml
- name: conversation-flow
  enabled: false
```

**Expected startup error:**
```
FATAL no enabled conversation-scoped judges found in config
```

The server should refuse to start (conversation judge is required). Restore config.

### Test Case 6.4 — Custom judge weights

Set explicit unequal weights:
```yaml
- name: relevance
  enabled: true
  weight: 0.60
- name: coherence
  enabled: true
  weight: 0.20
- name: completeness
  enabled: true
  weight: 0.20
```

Evaluate a conversation and verify:
```bash
jq '.turn_results[0].scores | map({name, weight})' response.json
```
**Expected:** weights are `relevance=0.60`, `coherence=0.20`, `completeness=0.20`.

Restore to original weights (0.35, 0.30, 0.35).

### Test Case 6.5 — Strict thresholds

```env
VERDICT_PASS_THRESHOLD=0.95
VERDICT_REVIEW_THRESHOLD=0.80
```

Restart and evaluate a good-quality answer. It should now return `"review"` instead of `"pass"` for scores in the 0.80–0.95 range.

Restore thresholds.

### Test Case 6.6 — Holistic weight tuning

```env
CONVERSATION_HOLISTIC_WEIGHT=0.8
```

Evaluate a multi-turn conversation. The `final_score` should weight holistic heavily:
- `final_score ≈ 0.8 × holistic_score + 0.2 × turn_avg`

Verify by comparing `final_score` to the formula manually.

---

## Part 7 — Dashboard Tests

Start the server and open `http://localhost:18082` in a browser.

### Test Case 7.1 — Dashboard loads

- Page title: `Themis - Evaluation Dashboard`
- Sidebar shows: `Conversations` (active) and `Monitoring`
- Stats row shows: `Total Conversations`, `Avg Final Score`, `Pass Rate`
- Default tab is Conversations

### Test Case 7.2 — Conversations list populates

After evaluating some conversations via the API:

1. Click "Refresh" button (or wait for auto-refresh)
2. Conversation cards appear in the list
3. Each card shows: `conversation_id`, agent name, turn count, `final_score`, `verdict` badge

**Verification:**
- Verdict badge color: green=pass, yellow=review, red=fail
- Stats row updates with correct totals

### Test Case 7.3 — Conversation detail view

Click any conversation card.

**Expected:**
- Detail view opens (list view hides)
- Summary card shows `final_score`, `turn_avg`, `holistic_score`, `verdict`
- Holistic reason text is displayed
- Turn list shows all turns

Click a turn.

**Expected:**
- Judge scores expand inline
- Each judge shows name, score percentage, and reason text
- Click again to collapse

### Test Case 7.4 — Back navigation

From detail view, click "← Back to conversations".

**Expected:**
- List view reappears
- URL hash returns to `#conversations`

### Test Case 7.5 — URL deep-linking

Navigate directly to a conversation:
```
http://localhost:18082/#conversations/tc-002
```

**Expected:**
- Conversations tab is active
- Detail view for `tc-002` loads directly

### Test Case 7.6 — Monitoring tab

Click "Monitoring" in the sidebar.

**Expected:**
- Metrics table loads with `Total Evaluations` and `Avg Final Score`
- Window selector defaults to "Last 7 days"

Change window to "Last 1 day":
- Values update to reflect shorter window
- `window` field in response changes to `"1d"`

### Test Case 7.7 — Theme toggle

Click the theme toggle button in the sidebar footer.

**Expected:**
- Theme switches from dark to light (or vice versa)
- Preference persists after page reload (stored in localStorage)

### Test Case 7.8 — Auto-refresh

Enable the auto-refresh toggle in the Conversations tab.

**Expected:**
- Status indicator becomes active
- List refreshes every 10 seconds
- New evaluations submitted via API appear without manual refresh

---

## Part 8 — End-to-End Scenarios

These simulate real usage patterns.

### Scenario 8.1 — Full pipeline walkthrough

```bash
# 1. Start server
./bin/themis-api

# 2. Evaluate a few conversations
curl -s -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{"conversation_id":"e2e-001","agent":{"name":"my-bot","version":"3.2"},"turns":[{"turn_index":1,"user_query":"What is machine learning?","answer":"Machine learning is a branch of AI where systems learn from data to improve performance without being explicitly programmed."},{"turn_index":2,"user_query":"Give me an example.","answer":"A spam filter is a classic example — it learns to identify spam emails from labeled training data."}]}' | jq '{verdict, final_score}'

# 3. Batch evaluate a dataset
./bin/themis-cli evaluate -input /tmp/conversations.jsonl -output /tmp/e2e-results.jsonl

# 4. Verify all stored
curl -s http://localhost:18082/api/v1/conversations | jq '.total'

# 5. Check monitoring
curl -s "http://localhost:18082/api/v1/metrics/health?window=1d" | jq .
```

### Scenario 8.2 — A/B test two agent versions

```bash
cat > /tmp/agent-ab.jsonl << 'EOF'
{"conversation_id":"ab-v1-001","agent":{"name":"agent","version":"v1"},"turns":[{"turn_index":1,"user_query":"What is the capital of Japan?","answer":"Tokyo."}]}
{"conversation_id":"ab-v1-002","agent":{"name":"agent","version":"v1"},"turns":[{"turn_index":1,"user_query":"What is 3x7?","answer":"21."}]}
{"conversation_id":"ab-v2-001","agent":{"name":"agent","version":"v2"},"turns":[{"turn_index":1,"user_query":"What is the capital of Japan?","answer":"Tokyo is the capital of Japan, located on Honshu island."}]}
{"conversation_id":"ab-v2-002","agent":{"name":"agent","version":"v2"},"turns":[{"turn_index":1,"user_query":"What is 3x7?","answer":"3 multiplied by 7 equals 21."}]}
EOF

./bin/themis-cli evaluate -input /tmp/agent-ab.jsonl -output /tmp/ab-results.jsonl

# Compare avg scores by version
jq -s 'group_by(.agent_version) | map({version: .[0].agent_version, avg_score: (map(.final_score) | add/length), pass_rate: (map(select(.verdict=="pass")) | length) / length})' /tmp/ab-results.jsonl
```

**Expected:** v2 has higher avg_score (more complete answers).

### Scenario 8.3 — Stream then query via API

```bash
# 1. Publish via streaming
./bin/themis-producer -d '{"conversation_id":"stream-api-001","agent":{"name":"test","version":"1.0"},"turns":[{"turn_index":1,"user_query":"What is Go?","answer":"Go is a statically typed, compiled language created by Google."}]}'

# 2. Wait for processing (~5s), then query via API
sleep 6
curl -s http://localhost:18082/api/v1/conversations/stream-api-001 | jq '{conversation_id, verdict, final_score}'
```

**Expected:** conversation is stored and retrievable via the API after streaming consumer processes it.

---

## Quick Verification Checklist

Use this as a go/no-go gate before releasing:

| # | Area | Check | Pass? |
|---|------|-------|-------|
| 1 | Unit tests | `go test ./...` — zero failures | |
| 2 | API startup | All 4 judges initialize, server binds on port | |
| 3 | Health | `GET /api/v1/health` returns `{"status":"ok"}` | |
| 4 | Evaluate (good) | `POST /evaluate` returns `verdict=pass`, `final_score>0.8` | |
| 5 | Evaluate (bad) | `POST /evaluate` with vague answer returns `verdict=fail` | |
| 6 | Validation | Missing `conversation_id` returns 400 | |
| 7 | Validation | Empty `turns` returns 400 | |
| 8 | List | `GET /conversations` returns array with `total` | |
| 9 | Get | `GET /conversations/{id}` returns full turn_results | |
| 10 | Not found | `GET /conversations/unknown` returns 404 | |
| 11 | Metrics | `GET /metrics/health?window=7d` returns valid JSON | |
| 12 | Invalid window | `GET /metrics/health?window=7w` returns 400 | |
| 13 | Batch CLI | `evaluate` command produces correct JSONL output | |
| 14 | Batch resilience | Bad JSON line is skipped, good lines processed | |
| 15 | MCP tools | `evaluate_conversation` and `get_conversation` discoverable | |
| 16 | MCP evaluate | Tool returns `final_score`, `verdict`, `turn_results` | |
| 17 | MCP retrieve | `get_conversation` returns stored result | |
| 18 | Streaming | Messages consumed and acknowledged from Redis | |
| 19 | Streaming | Invalid payload is acked (no redelivery loop) | |
| 20 | Dashboard | Conversations tab loads, cards display verdict badges | |
| 21 | Dashboard | Turn detail expands with judge scores inline | |
| 22 | Dashboard | Monitoring tab shows `total_evaluations` | |
| 23 | Judge config | Disabling a judge removes it from response scores | |
| 24 | Weight normalization | Remaining judges re-normalized when one is disabled | |

---

## Troubleshooting

### Server won't start
```bash
# Missing judges.yaml
ls configs/judges.yaml

# Missing LLM key
echo $OPEN_AI_KEY

# Port in use
lsof -i :18082
EVAL_AGENT_API_PORT=18083 ./bin/themis-api
```

### LLM errors during evaluation
```bash
# Test OpenAI key directly
curl https://api.openai.com/v1/models -H "Authorization: Bearer $OPEN_AI_KEY"

# Test AWS Bedrock
aws bedrock list-foundation-models --region us-east-1
```

### Redis streaming not consuming
```bash
redis-cli PING                                          # check Redis is up
redis-cli XINFO GROUPS eval-conversations               # check group exists
redis-cli XPENDING eval-conversations eval-conv-group   # check pending count
```

### MCP server not ready
```bash
# Check binary exists and is executable
ls -la bin/themis-mcp
chmod +x bin/themis-mcp

# Check registration
claude mcp list

# Remove and re-add
claude mcp remove themis
claude mcp add --transport stdio --scope project themis \
  --env OPEN_AI_KEY=$OPEN_AI_KEY \
  -- $(pwd)/bin/themis-mcp
```
