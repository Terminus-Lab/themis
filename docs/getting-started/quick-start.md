---
title: Quick Start
description: End-to-end walkthrough to verify Themis before release
version: 1.0.0
tags: [quick-start, tutorial, getting-started, examples]
related:
  - getting-started/installation.md
  - getting-started/configuration.md
  - deployment/api-mode.md
  - testing/api-tests.md
---

# Quick Start

End-to-end walkthrough that covers the full Themis workflow: CLI batch processing, judge validation, API server, dashboard, and sampling.

## Prerequisites

Build the binaries and configure credentials:

```bash
go build -o bin/themis-api cmd/api/main.go
go build -o bin/themis-cli cmd/batch/main.go

cp .env.example .env
# Add your LLM key — simplest option:
echo "OPEN_AI_KEY=sk-proj-YOUR_KEY_HERE" >> .env
```

---

## Step 1 — CLI: Batch Evaluate a Dataset

Run the batch evaluator against the sample dataset to verify the full pipeline works end-to-end from the command line.

```bash
./bin/themis-cli evaluate \
  -i resources/dataset.jsonl \
  -o results.jsonl
```

**Expected output:**
```
INFO Input file parsed records=10
INFO Starting worker pool workers=5
INFO Processing complete total=10 pass=7 review=2 fail=1
```

Inspect the results:

```bash
# Verdict distribution
jq -s 'group_by(.verdict) | map({verdict: .[0].verdict, count: length})' results.jsonl

# Average confidence
jq -s 'map(.confidence) | add/length' results.jsonl
```

**Verify:**
- `results.jsonl` has one line per input record
- Each line has `verdict`, `confidence`, and `stage_scores`
- No errors in logs

---

## Step 2 — CLI: Validate Judge Accuracy

Run the validation command against the annotated dataset to confirm the judges agree with human annotations (Kendall's τ ≥ 0.3 required).

```bash
./bin/themis-cli validate \
  -i resources/validation_success_dataset.jsonl \
  -c 0.3
```

**Expected output:**
```
INFO Starting validation records=150 threshold=0.3
INFO Validation complete kendall_tau=0.63 status=PASSED
INFO LLM judge validated against human annotations
```

**Verify:**
- `status=PASSED`
- `kendall_tau` ≥ 0.3
- Exit code 0

> If this fails, do not proceed to release. Review and tune `configs/judges.yaml`.

---

## Step 3 — API: Start the Server

```bash
./bin/themis-api
```

**Expected startup logs:**
```
INFO judge created successfully judge=relevance
INFO judge created successfully judge=faithfulness
INFO judge created successfully judge=coherence
INFO judge created successfully judge=completeness
INFO judge created successfully judge=instruction
INFO judge pool built successfully total_judges=5
INFO Starting Themis Server address=:18082
```

**Verify health:**
```bash
curl -s http://localhost:18082/api/v1/health | jq .
# {"status":"ok","version":"1.0.0"}
```

---

## Step 4 — API: Seed Evaluation Data

Send a few evaluations to populate the database, including a multi-turn conversation.

**Single evaluation (good answer):**
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

**Failing evaluation (hallucination):**
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

Expected: `verdict: "fail"` (catches hallucination about China).

---

## Step 5 — API: Download a 25% Sample

Sample the evaluation results for human annotation review:

```bash
curl -s -X POST http://localhost:18082/api/v1/validation/sample/download \
  -H "Content-Type: application/json" \
  -d '{
    "start_date": "2020-01-01T00:00:00Z",
    "end_date": "2099-01-01T00:00:00Z",
    "percentage": 25
  }' -o sample.jsonl

echo "Sampled $(wc -l < sample.jsonl) records"
head -1 sample.jsonl | jq '{event_id, verdict, confidence}'
```

**Verify:**
- Status 200
- `Content-Type: application/x-ndjson`
- Each line is valid JSON with `event_id`, `verdict`, `confidence`, `stage_scores`

---

## Step 6 — Dashboard: Verify the UI

Open `http://localhost:18082` in your browser and check each tab:

**Results tab:**
- [ ] Evaluation rows load with verdict badges
- [ ] Filter by `agent_name=my-agent` returns only matching rows
- [ ] Click a row to expand stage scores

**Conversations tab:**
- [ ] `conv-qs-001` appears with 3 turns
- [ ] Click the conversation to see turn-by-turn detail

**Monitoring tab:**
- [ ] Metrics table loads with `total_evaluations`, `avg_confidence`, `avg_judge_disagreement`
- [ ] Switch window to `24h` — values update
- [ ] Disagreement is shown as a decimal (not a percentage)

---

## Step 7 — API: Run a Live Evaluation

Verify the full pipeline returns a correct response:

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
```json
{
  "verdict": "pass",
  "confidence": 0.89,
  "stage_count": 8
}
```

- 8 stages = 3 prechecks + 5 LLM judges
- `confidence` > 0.8 → `verdict: "pass"`

---

## Release Checklist

| Step | What to verify | Pass? |
|------|---------------|-------|
| 1 — CLI evaluate | results.jsonl produced, no errors | |
| 2 — CLI validate | Kendall's τ ≥ 0.3, status=PASSED | |
| 3 — API start | All judges initialized, health returns `ok` | |
| 4 — Seed data | Evaluations stored, conversation created | |
| 5 — Sample download | JSONL response, correct record count | |
| 6 — Dashboard | All 3 tabs load and display correct data | |
| 7 — Live evaluation | 8 stages, pass verdict, correct confidence | |

All steps must pass before tagging a release.

---

## Next Steps

- **[Configuration Guide](configuration.md)** — Tune thresholds and judge weights
- **[API Mode Deployment](../deployment/api-mode.md)** — Production deployment guide
- **[API Tests](../testing/api-tests.md)** — Full API test case reference
- **[Batch Tests](../testing/batch-tests.md)** — CLI and validation test cases
