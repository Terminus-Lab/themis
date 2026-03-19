# Release Validation Guide

Step-by-step checklist to run before tagging any release. All steps must pass. Do not skip ahead — each gate depends on the previous one being clean.

---

## Prerequisites

```bash
# Clone a fresh copy or start from a clean working tree
git status  # must be clean

# Copy and configure credentials
cp .env.example .env
# Edit .env and fill in your LLM provider credentials (see Configuration section below)

# Ensure Go toolchain is available
go version  # 1.21+
```

---

## Gate 1 — Build All Binaries

Build every binary to catch compilation errors before running anything.

```bash
mkdir -p bin

go build -o bin/themis-api     cmd/api/main.go
go build -o bin/themis-cli     cmd/batch/main.go
go build -o bin/themis-mcp     cmd/mcp/main.go
go build -o bin/themis-producer cmd/producer/main.go
```

**Pass criteria:**
- All 4 binaries exist in `bin/`
- Zero compilation errors or warnings

```bash
ls -lh bin/
# themis-api, themis-cli, themis-mcp, themis-producer — all present
```

---

## Gate 2 — Unit Tests

Run the full test suite. No LLM calls are made here (unit tests use mocks/stubs).

```bash
go test ./... -count=1
```

**Pass criteria:**
- `ok` on every package line
- Zero test failures
- Zero data races (add `-race` for a stricter check)

```bash
# Stricter: with race detector
go test -race ./... -count=1

# With coverage report
go test -cover ./... -count=1
```

> If any package fails, stop here. Do not proceed to integration or API gates.

---

## Gate 3 — Smoke Tests (End-to-End)

Run the automated smoke test script. This exercises build → unit tests → API → CLI in sequence.

```bash
bash scripts/smoke-test.sh
```

The script validates 10 stages:
1. Binary build
2. Unit tests
3. API server start
4. Health check
5. Evaluate (pass case)
6. Query results
7. Graceful shutdown
8. CLI batch evaluate
9. CLI summary
10. CLI validate-events (Kendall's τ ≥ 0.3)

**Pass criteria:**
- Final line: `All smoke tests passed ✓`
- Exit code 0

> If `smoke-test.sh` fails at any stage, fix the issue before proceeding.

---

## Gate 4 — Judge Configuration Validation

Validate that `configs/judges.yaml` is structurally correct and all enabled judges load without errors.

```bash
# Start the API and check logs for judge initialization
./bin/themis-api 2>&1 | head -30
```

**Pass criteria in logs:**
- `judge pool initialized` (or equivalent) — no missing model entries
- No `ERROR` or `FATAL` lines during startup
- Health endpoint responds immediately after start

```bash
curl -s http://localhost:18082/api/v1/health | jq .
# {"status":"ok","version":"..."}
```

Check each enabled judge appears in the startup log. If a judge fails to initialize due to a missing credential or bad model ID, it will be logged here.

---

## Gate 5 — API Endpoint Validation

Start the API server and test every endpoint. Run these in order.

```bash
./bin/themis-api &
SERVER_PID=$!
sleep 3
```

### 5.1 Health

```bash
curl -s http://localhost:18082/api/v1/health | jq .
```

Expected:
```json
{"status": "ok", "version": "..."}
```

### 5.2 Evaluate — Pass Case

```bash
curl -s -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "rv-001",
    "conversation_id": "conv-rv-001",
    "event_type": "agent_response",
    "agent": {"name": "release-validator", "version": "1.0"},
    "interaction": {
      "user_query": "What is the capital of France?",
      "context": "France is a country in Western Europe. Paris is its capital and largest city.",
      "answer": "The capital of France is Paris."
    }
  }' | jq '{verdict, confidence}'
```

Expected: `verdict: "pass"`, `confidence` > 0.8

### 5.3 Evaluate — Fail Case (Hallucination)

```bash
curl -s -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "rv-002",
    "conversation_id": "conv-rv-002",
    "event_type": "agent_response",
    "agent": {"name": "release-validator", "version": "1.0"},
    "interaction": {
      "user_query": "What is the population of Tokyo?",
      "context": "Tokyo is the capital of Japan.",
      "answer": "Tokyo has 50 million people and is the largest city in China."
    }
  }' | jq '{verdict, confidence}'
```

Expected: `verdict: "fail"`.

### 5.4 Evaluate — With Expected Output (Correctness Judge)

```bash
curl -s -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "rv-003",
    "conversation_id": "conv-rv-003",
    "event_type": "agent_response",
    "agent": {"name": "release-validator", "version": "1.0"},
    "interaction": {
      "user_query": "What is 2 + 2?",
      "context": "Basic arithmetic.",
      "answer": "2 + 2 equals 4.",
      "expected_output": "4"
    }
  }' | jq '{verdict, confidence, stage_scores}'
```

Expected: correctness judge appears in `stage_scores` (not skipped).

### 5.5 Multi-Turn Conversation

```bash
for turn in 1 2 3; do
  curl -s -X POST http://localhost:18082/api/v1/evaluate \
    -H "Content-Type: application/json" \
    -d "{
      \"event_id\": \"rv-conv-$turn\",
      \"conversation_id\": \"conv-rv-mt\",
      \"event_type\": \"agent_response\",
      \"agent\": {\"name\": \"release-validator\", \"version\": \"1.0\"},
      \"interaction\": {
        \"user_query\": \"Tell me about topic $turn\",
        \"context\": \"Context for topic $turn.\",
        \"answer\": \"Here is a detailed answer about topic $turn.\"
      }
    }" | jq '{verdict, confidence}'
done
```

Expected: 3 responses, all non-error.

### 5.6 Single Judge Evaluation

```bash
curl -s -X POST http://localhost:18082/api/v1/evaluate/judge/relevance \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "rv-judge-001",
    "conversation_id": "conv-rv-judge",
    "event_type": "agent_response",
    "agent": {"name": "release-validator", "version": "1.0"},
    "interaction": {
      "user_query": "What is Go?",
      "context": "Go is a statically typed programming language designed at Google.",
      "answer": "Go is a compiled, statically typed language developed by Google."
    }
  }' | jq '{verdict, confidence}'
```

Expected: response with `verdict` and `confidence` from relevance judge only.

### 5.7 Query Results

```bash
# All results
curl -s "http://localhost:18082/api/v1/results?limit=10" | jq '{total, count: (.results | length)}'

# Filter by agent
curl -s "http://localhost:18082/api/v1/results?agent_name=release-validator&limit=10" | jq '.total'

# Filter by verdict
curl -s "http://localhost:18082/api/v1/results?verdict=pass&limit=10" | jq '.total'

# Get specific result by ID
curl -s "http://localhost:18082/api/v1/results/rv-001" | jq '{event_id, verdict}'
```

Expected: results returned with correct filtering, `total` > 0 after seeding above.

### 5.8 Conversations

```bash
# List all conversations
curl -s "http://localhost:18082/api/v1/conversations" | jq '{total, count: (.conversations | length)}'

# Get specific conversation with turns
curl -s "http://localhost:18082/api/v1/conversations/conv-rv-mt" | jq '{conversation_id, turn_count: (.turns | length)}'
```

Expected: `conv-rv-mt` present with 3 turns.

### 5.9 Stop Server

```bash
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null || true
echo "Server stopped"
```

---

## Gate 6 — CLI Validation

### 6.1 Batch Evaluate

```bash
./bin/themis-cli evaluate \
  -i resources/dataset.jsonl \
  -o /tmp/rv-results.jsonl

# Verify output
wc -l /tmp/rv-results.jsonl          # must equal input count
head -1 /tmp/rv-results.jsonl | jq '{verdict, confidence}'   # must have both fields
jq -r '.verdict' /tmp/rv-results.jsonl | sort | uniq -c      # verdict distribution
```

**Pass criteria:**
- Output line count equals input line count
- Every line has `verdict` and `confidence` fields
- No `"verdict": null` entries

### 6.2 Summary Mode

```bash
./bin/themis-cli evaluate \
  -i resources/dataset.jsonl \
  -f summary
```

Expected: verdict distribution table printed to stdout (no output file needed).

### 6.3 Parallel Workers

```bash
THEMIS_BATCH_WORKERS=10 ./bin/themis-cli evaluate \
  -i resources/dataset.jsonl \
  -o /tmp/rv-results-10w.jsonl

# Output should match single-worker output line count
wc -l /tmp/rv-results-10w.jsonl
```

### 6.4 Judge Accuracy Validation (Critical Gate)

```bash
./bin/themis-cli validate-events \
  -i resources/validation_success_dataset.jsonl \
  -c 0.3
```

**Pass criteria:**
- `"status": "PASSED"` in JSON output
- `"kendall_tau"` ≥ 0.3
- Exit code 0

> **If this fails, do not release.** Tune prompts in `configs/judges.yaml` until Kendall's τ ≥ 0.3.

```bash
# Inspect the full validation output
./bin/themis-cli validate-events \
  -i resources/validation_success_dataset.jsonl \
  -c 0.3 | jq '{status, kendall_tau, total_evaluated}'
```

---

## Gate 7 — Aggregation Methods

Verify all 4 aggregation methods produce valid scores and are returned in the `metrics` field.

```bash
for method in weighted_average harmonic_mean median weighted_product; do
  echo "--- $method ---"
  JUDGE_AGGREGATION_METHOD=$method ./bin/themis-api &
  PID=$!
  sleep 3

  curl -s -X POST http://localhost:18082/api/v1/evaluate \
    -H "Content-Type: application/json" \
    -d '{
      "event_id": "rv-agg-001",
      "conversation_id": "conv-rv-agg",
      "event_type": "agent_response",
      "agent": {"name": "agg-test", "version": "1.0"},
      "interaction": {
        "user_query": "What is photosynthesis?",
        "context": "Photosynthesis is the process by which plants convert light into energy.",
        "answer": "Photosynthesis is how plants use sunlight to produce food from carbon dioxide and water."
      }
    }' | jq '.metrics | {stage2_weighted_avg, stage2_harmonic_mean, stage2_median, stage2_weighted_product, aggregation_method}'

  kill $PID
  wait $PID 2>/dev/null || true
done
```

**Pass criteria:**
- All 4 metric fields present and non-null
- `aggregation_method` matches the `JUDGE_AGGREGATION_METHOD` env var used

---

## Gate 8 — Precheck Toggle

Verify `ENABLE_PRECHECK=false` works correctly (LLM-only mode).

```bash
ENABLE_PRECHECK=false ./bin/themis-api &
PID=$!
sleep 3

curl -s -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "rv-nopre-001",
    "conversation_id": "conv-rv-nopre",
    "event_type": "agent_response",
    "agent": {"name": "nopre-test", "version": "1.0"},
    "interaction": {
      "user_query": "What is 2+2?",
      "answer": "4"
    }
  }' | jq '{verdict, confidence, stage_count: (.stage_scores | length)}'

kill $PID
wait $PID 2>/dev/null || true
```

**Pass criteria:**
- Response is valid JSON with `verdict` and `confidence`
- `stage_scores` contains only LLM judge entries (no precheck entries)
- `confidence` equals stage 2 score directly (no stage 1 weighting)

---

## Gate 9 — Dashboard UI

Start the API and open the dashboard manually.

```bash
./bin/themis-api &
PID=$!
sleep 3

# Seed some data first (run Gate 5 evaluate steps above if not already done)

# Then open dashboard
open http://localhost:18082   # macOS
# xdg-open http://localhost:18082  # Linux
```

Checklist:

**Results tab:**
- [ ] Rows load with verdict badges (green/yellow/red)
- [ ] Filter by `agent_name=release-validator` returns only that agent's results
- [ ] Filter by `verdict=pass` returns only passing results
- [ ] Click a row expands to show stage scores and reasons
- [ ] Pagination (Previous/Next) works when more than one page exists
- [ ] Auto-refresh triggers every ~10 seconds (watch network tab)

**Conversations tab:**
- [ ] `conv-rv-mt` appears with 3 turns
- [ ] Click conversation to see per-turn detail
- [ ] Turn rows show individual verdicts
- [ ] Back navigation works (browser back button)

```bash
kill $PID
wait $PID 2>/dev/null || true
```

---

## Gate 10 — Docker Build (MCP)

```bash
docker build -t themis-mcp:release-test .
```

**Pass criteria:**
- Image builds without error
- Image size is reasonable (check with `docker images themis-mcp:release-test`)

```bash
# Quick container smoke test (stdio mode, exits immediately)
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | \
  docker run --rm -i --env-file .env themis-mcp:release-test 2>/dev/null | \
  jq '.result.tools | map(.name)'
```

Expected: `["evaluate_response", "evaluate_single_judge", "get_conversation"]` (or superset).

---

## Gate 11 — Streaming Mode (if Redis available)

Skip this gate if Redis is not configured in your environment.

```bash
# Requires Redis running locally
STREAMING_ENABLED=true REDIS_ADDR=localhost:6379 ./bin/themis-api &
PID=$!
sleep 3

curl -s http://localhost:18082/api/v1/health | jq .
# Must still return {"status":"ok"}

# Produce a test event
./bin/themis-producer &
PROD_PID=$!
sleep 5
kill $PROD_PID

# Check that streamed events appear in results
curl -s "http://localhost:18082/api/v1/results?limit=5" | jq '.total'

kill $PID
wait $PID 2>/dev/null || true
```

**Pass criteria:**
- API health still returns `ok` in unified mode
- Streamed events appear in query results

---

## Gate 12 — goreleaser Dry Run

Validate release configuration without publishing.

```bash
# Requires goreleaser installed: brew install goreleaser
goreleaser check                    # validate .goreleaser.yaml syntax
goreleaser build --snapshot --clean # build all platform binaries locally
```

**Pass criteria:**
- `goreleaser check` exits 0
- `dist/` directory contains binaries for all 6 platform targets

```bash
ls dist/themis*/
# themis-api_linux_amd64, themis-api_linux_arm64
# themis-api_darwin_amd64, themis-api_darwin_arm64
# themis-api_windows_amd64
# (same for themis-cli and themis-mcp)
```

---

## Final Release Checklist

Run through this table. Every row must be checked before tagging.

| # | Gate | Command / Action | Pass Criteria | Pass? |
|---|------|-----------------|---------------|-------|
| 1 | Build | `go build ./cmd/...` | 4 binaries, zero errors | |
| 2 | Unit tests | `go test -race ./...` | All packages `ok` | |
| 3 | Smoke tests | `bash scripts/smoke-test.sh` | `All smoke tests passed ✓` | |
| 4 | Judge init | Check startup logs | No ERROR/FATAL, all judges loaded | |
| 5.1 | Health endpoint | `curl .../health` | `{"status":"ok"}` | |
| 5.2 | Evaluate pass | POST evaluate, good answer | `verdict: "pass"` | |
| 5.3 | Evaluate fail | POST evaluate, hallucination | `verdict: "fail"` | |
| 5.4 | With expected output | POST evaluate + `expected_output` | correctness judge in stage_scores | |
| 5.5 | Multi-turn | 3 turns same conversation_id | 3 valid responses | |
| 5.6 | Single judge | POST evaluate/judge/relevance | verdict from one judge | |
| 5.7 | Query results | GET results with filters | Correct filtering, total > 0 | |
| 5.8 | Conversations | GET conversations + turns | conv-rv-mt with 3 turns | |
| 6.1 | CLI evaluate | `themis-cli evaluate` | Line count matches input | |
| 6.2 | CLI summary | `themis-cli evaluate -f summary` | Verdict distribution printed | |
| 6.3 | CLI workers | `THEMIS_BATCH_WORKERS=10` | Same output count | |
| 6.4 | **Judge accuracy** | `validate-events -c 0.3` | τ ≥ 0.3, `PASSED` | |
| 7 | Aggregation | All 4 `JUDGE_AGGREGATION_METHOD` values | All metrics present per response | |
| 8 | Precheck toggle | `ENABLE_PRECHECK=false` | Valid response, no stage 1 entries | |
| 9 | Dashboard UI | Open `http://localhost:18082` | All tabs load, drill-down works | |
| 10 | Docker build | `docker build` | Image builds, MCP tools listed | |
| 11 | Streaming | `STREAMING_ENABLED=true` (if Redis) | Health ok, streamed events stored | |
| 12 | goreleaser | `goreleaser build --snapshot` | All 6 platform targets built | |

> Gate 6.4 (Judge accuracy, Kendall's τ ≥ 0.3) is the only hard blocker that cannot be bypassed. All other failures indicate bugs that must be fixed before release.

---

## Common Failure Patterns

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| Judge skips unexpectedly | `requires_context: true` but no context sent | Send `context` field in request, or check judge config |
| `confidence: 0` or `verdict: fail` on good answer | LLM key missing or wrong model ID | Check `.env` credentials and `configs/judges.yaml` modelID |
| Kendall's τ < 0.3 | Judge prompts poorly calibrated | Tune prompts in `configs/judges.yaml` and re-run validation |
| Docker build fails | Missing `static/` or `configs/` in image | Check `.goreleaser.yaml` `extra_files` and `Dockerfile` COPY paths |
| `results?agent_name=X` returns 0 | Agent name mismatch in request payload | Verify `agent.name` field in evaluate payload matches filter value |
| Streaming events not appearing | Wrong consumer group or stream key | Check `REDIS_STREAM_KEY` and `REDIS_CONSUMER_GROUP` in `.env` |
