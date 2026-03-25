# Quick Start

End-to-end walkthrough: CLI batch evaluation, API server, dashboard, and streaming.

## Prerequisites

```bash
go build -o bin/themis-api cmd/api/main.go
go build -o bin/themis-cli cmd/batch/main.go

cp .env.example .env
echo "OPEN_AI_KEY=sk-proj-YOUR_KEY_HERE" >> .env
```

---

## Step 1 — CLI: Batch Evaluate Conversations

Use the annotated sample dataset (15 conversations spanning pass/review/fail quality tiers):

```bash
./bin/themis-cli evaluate \
  -input resources/annotated_sample.jsonl \
  -output /tmp/results.jsonl
```

**Verify:**
- `/tmp/results.jsonl` has one line per conversation
- Each line has `final_score`, `verdict`, `turn_avg`, `holistic_score`
- The summary includes a `correlation_report` comparing Themis verdicts against the `human_label` annotations

```bash
# Verdict distribution
jq -s 'group_by(.verdict) | map({verdict: .[0].verdict, count: length})' /tmp/results.jsonl
```

---

## Step 2 — API: Start Server

```bash
./bin/themis-api
```

**Verify health:**
```bash
curl -s http://localhost:18082/api/v1/health | jq .
# {"status":"ok"}
```

---

## Step 3 — API: Evaluate a Conversation

**Single-turn (good answer):**
```bash
curl -s -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "qs-001",
    "agent": {"name": "my-agent", "version": "1.0"},
    "turns": [
      {
        "turn_index": 1,
        "user_query": "What is the capital of France?",
        "answer": "The capital of France is Paris."
      }
    ]
  }' | jq '{verdict, final_score, turn_avg, holistic_score}'
```

**Multi-turn conversation:**
```bash
curl -s -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "qs-002",
    "agent": {"name": "my-agent", "version": "1.0"},
    "turns": [
      {"turn_index": 1, "user_query": "What is the capital of France?", "answer": "Paris."},
      {"turn_index": 2, "user_query": "What is it known for?", "answer": "Paris is known for the Eiffel Tower, art, and cuisine."},
      {"turn_index": 3, "user_query": "What is the population?", "answer": "Paris has about 2.1 million people in the city proper."}
    ]
  }' | jq '{verdict, final_score, turn_avg, holistic_score}'
```

**Low quality answer (should fail):**
```bash
curl -s -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "qs-003",
    "agent": {"name": "my-agent", "version": "1.0"},
    "turns": [
      {"turn_index": 1, "user_query": "Explain quantum computing.", "answer": "Yes."}
    ]
  }' | jq '{verdict, final_score}'
```

Expected: `verdict: "fail"`.

---

## Step 4 — API: Query Results

```bash
# List all conversations
curl -s http://localhost:18082/api/v1/conversations | jq .

# Get specific conversation details
curl -s http://localhost:18082/api/v1/conversations/qs-002 | jq .

# Health metrics
curl -s "http://localhost:18082/api/v1/metrics/health?window=7d" | jq .
```

---

## Step 5 — Dashboard: Verify the UI

Open `http://localhost:18082` and check:

- [ ] Conversation list loads with verdict badges and scores
- [ ] Click a conversation to see per-turn scores
- [ ] Monitoring tab shows `total_evaluations` and `avg_confidence`
- [ ] Window switcher (7d / 1d) updates values

---

## Release Checklist

| Step | What to verify | Pass? |
|------|----------------|-------|
| 1 — CLI evaluate | `results.jsonl` produced, no errors | |
| 2 — API start | All judges initialized, health returns `ok` | |
| 3 — Evaluate | Conversations stored, verdicts correct | |
| 4 — Query | List and get endpoints return correct data | |
| 5 — Dashboard | Conversations and metrics display correctly | |

---

## Next Steps

- [Configuration](configuration.md) — Tune thresholds and judge weights
- [API Mode](../deployment/api-mode.md) — Production deployment
- [API Tests](../testing/api-tests.md) — Full API test reference
- [Batch Tests](../testing/batch-tests.md) — CLI test cases
