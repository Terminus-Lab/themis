# Quick Start

End-to-end walkthrough covering CLI batch evaluation, judge validation, API server, dashboard, and sampling. Use this as a release verification checklist.

## Prerequisites

```bash
go build -o bin/themis-api cmd/api/main.go
go build -o bin/themis-cli cmd/batch/main.go

cp .env.example .env
echo "OPEN_AI_KEY=sk-proj-YOUR_KEY_HERE" >> .env
```

---

## Step 1 — CLI: Batch Evaluate

```bash
./bin/themis-cli evaluate \
  -i resources/dataset.jsonl \
  -o results.jsonl
```

**Verify:**
- `results.jsonl` has one line per input record
- Each line has `verdict`, `confidence`, and `stage_scores`
- No errors in logs

```bash
# Verdict distribution
jq -s 'group_by(.verdict) | map({verdict: .[0].verdict, count: length})' results.jsonl
```

---

## Step 2 — CLI: Validate Judge Accuracy

```bash
./bin/themis-cli validate \
  -i resources/validation_success_dataset.jsonl \
  -c 0.3
```

**Verify:**
- `status=PASSED`
- `kendall_tau` ≥ 0.3
- Exit code 0

> If this fails, do not proceed to release. Review and tune `configs/judges.yaml`.

---

## Step 3 — API: Start Server

```bash
./bin/themis-api
```

**Verify health:**
```bash
curl -s http://localhost:18082/api/v1/health | jq .
# {"status":"ok","version":"1.0.0"}
```

---

## Step 4 — API: Seed Evaluation Data

**Good answer:**
```bash
curl -s -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "qs-001",
    "event_type": "agent_response",
    "agent": {"name": "my-agent", "version": "1.0"},
    "interaction": {
      "user_query": "What is the capital of France?",
      "context": "France is a country in Western Europe. Paris is its capital.",
      "answer": "The capital of France is Paris."
    }
  }' | jq '{verdict, confidence}'
```

**Multi-turn conversation:**
```bash
for turn in 1 2 3; do
  curl -s -X POST http://localhost:18082/api/v1/evaluate \
    -H "Content-Type: application/json" \
    -d "{
      \"event_id\": \"conv-turn-$turn\",
      \"conversation_id\": \"conv-qs-001\",
      \"event_type\": \"agent_response\",
      \"agent\": {\"name\": \"my-agent\", \"version\": \"1.0\"},
      \"interaction\": {
        \"user_query\": \"Question $turn\",
        \"context\": \"Some context.\",
        \"answer\": \"Answer $turn with relevant information.\"
      }
    }" | jq '{verdict, confidence}'
done
```

**Hallucination (should fail):**
```bash
curl -s -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "qs-002",
    "event_type": "agent_response",
    "agent": {"name": "my-agent", "version": "1.0"},
    "interaction": {
      "user_query": "What is the population of Tokyo?",
      "context": "Tokyo is the capital of Japan.",
      "answer": "Tokyo has 50 million people and is the largest city in China."
    }
  }' | jq '{verdict, confidence}'
```

Expected: `verdict: "fail"`.

---

## Step 5 — API: Download a 25% Sample

```bash
curl -s -X POST http://localhost:18082/api/v1/validation/sample/download \
  -H "Content-Type: application/json" \
  -d '{
    "start_date": "2020-01-01T00:00:00Z",
    "end_date": "2099-01-01T00:00:00Z",
    "percentage": 25
  }' -o sample.jsonl

echo "Sampled $(wc -l < sample.jsonl) records"
head -1 sample.jsonl | jq .
```

**Verify:**
- Status 200
- `Content-Type: application/x-ndjson`
- Each line contains `event_id`, `agent`, `interaction` fields — no evaluation scores (by design, to avoid biasing human annotators)

---

## Step 6 — Dashboard: Verify the UI

Open `http://localhost:18082` and check:

**Results tab:**
- [ ] Rows load with verdict badges
- [ ] Filter by `agent_name=my-agent` works
- [ ] Click a row to expand stage scores

**Conversations tab:**
- [ ] `conv-qs-001` appears with 3 turns
- [ ] Click conversation to see turn detail

**Monitoring tab:**
- [ ] Metrics table loads (`total_evaluations`, `avg_confidence`, `avg_judge_disagreement`)
- [ ] Window switcher (7d / 24h) updates values

---

## Step 7 — API: Live Evaluation

```bash
curl -s -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "qs-live-001",
    "event_type": "agent_response",
    "agent": {"name": "my-agent", "version": "1.0"},
    "interaction": {
      "user_query": "Explain what machine learning is.",
      "context": "Machine learning is a subset of AI that enables systems to learn from data.",
      "answer": "Machine learning allows computers to improve their performance on tasks by learning patterns from data, without being explicitly programmed for each scenario."
    }
  }' | jq '{verdict, confidence, stage_count: (.stage_scores | length)}'
```

**Expected:**
- `verdict: "pass"`
- `confidence` > 0.8
- `stage_count` = 8 (3 prechecks + 5 LLM judges)

---

## Release Checklist

| Step | What to verify | Pass? |
|------|----------------|-------|
| 1 — CLI evaluate | `results.jsonl` produced, no errors | |
| 2 — CLI validate | Kendall's τ ≥ 0.3, `status=PASSED` | |
| 3 — API start | All judges initialized, health returns `ok` | |
| 4 — Seed data | Evaluations stored, conversation created | |
| 5 — Sample download | JSONL response, interaction-only fields | |
| 6 — Dashboard | All 3 tabs load and display correct data | |
| 7 — Live evaluation | 8 stages, pass verdict | |

All steps must pass before tagging a release.

---

## Next Steps

- [Configuration](configuration.md) — Tune thresholds and judge weights
- [API Mode](../deployment/api-mode.md) — Production deployment
- [API Tests](../testing/api-tests.md) — Full API test reference
- [Batch Tests](../testing/batch-tests.md) — CLI and validation test cases
