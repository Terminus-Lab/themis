---
title: Score Interpretation Guide
description: How to read and act on Themis evaluation scores
tags: [metrics, scores, verdict, interpretation]
---

# Score Interpretation Guide

## Score Structure

Every conversation evaluation produces four scores:

| Field | What it measures | Range |
|-------|-----------------|-------|
| `turn_avg` | Average per-turn quality (relevance, coherence, completeness) | 0.0 – 1.0 |
| `holistic_score` | Conversation-level flow and context-awareness across all turns | 0.0 – 1.0 |
| `final_score` | Weighted combination of the two | 0.0 – 1.0 |
| `verdict` | Classification derived from `final_score` | pass / review / fail |

**Formula:**

```
final_score = α × holistic_score + (1 - α) × turn_avg
```

where `α = CONVERSATION_HOLISTIC_WEIGHT` (default 0.5).

---

## Verdicts

| Verdict | Condition | Meaning |
|---------|-----------|---------|
| `pass` | `final_score > VERDICT_PASS_THRESHOLD` (default 0.8) | Acceptable quality |
| `review` | `final_score > VERDICT_REVIEW_THRESHOLD` (default 0.5) | Needs human review |
| `fail` | `final_score ≤ VERDICT_REVIEW_THRESHOLD` | Poor quality |

### Tuning thresholds

If your distribution skews too heavily in one direction:

- **Too many `pass`** → raise `VERDICT_PASS_THRESHOLD` (e.g. 0.85 or 0.90)
- **Too many `fail`** → lower `VERDICT_REVIEW_THRESHOLD` (e.g. 0.40)
- **Too many `review`** → narrow the band: raise pass threshold and raise review threshold together

---

## Per-Turn Scores

Each turn has a `turn_score` (weighted average of individual judge scores) and a `scores` array:

```json
{
  "turn_index": 1,
  "turn_score": 0.87,
  "scores": [
    {"name": "relevance",    "score": 0.92, "weight": 0.35, "reason": "..."},
    {"name": "coherence",    "score": 0.85, "weight": 0.30, "reason": "..."},
    {"name": "completeness", "score": 0.83, "weight": 0.35, "reason": "..."}
  ]
}
```

`turn_score = sum(score × weight)` across all enabled judges.

### Reading judge reasons

The `reason` field is the LLM judge's natural-language explanation for its score. This is the most actionable output — it tells you *why* a score is low, not just that it is.

---

## Holistic Score

The `holistic_score` captures what per-turn scores cannot: whether the agent builds on context from earlier turns, maintains consistency, and guides the user naturally through the conversation.

- A conversation can have high `turn_avg` (each individual answer is fine in isolation) but low `holistic_score` (the agent ignores what was said two turns ago).
- The `holistic_reason` field explains the holistic assessment in plain text.

---

## Score Variability

LLM judges are non-deterministic. Expect ±0.05 variability between runs on the same input. To reduce noise:
- Set `temperature: 0.0` in `judges.yaml` (already the default)
- Use `retry: true` to recover from transient failures
- For dataset-level analysis, individual score variance averages out across many evaluations

---

## Interpreting Dataset Results (Batch Output)

After running `themis-cli evaluate`, the JSONL output can be analyzed:

```bash
# Verdict distribution
jq -r '.verdict' results.jsonl | sort | uniq -c

# Average final score
jq -s 'map(.final_score) | add/length' results.jsonl

# Conversations below threshold
jq 'select(.final_score < 0.6) | {conversation_id, final_score, verdict}' results.jsonl

# Per-agent average score
jq -s 'group_by(.agent_name) | map({agent: .[0].agent_name, avg: (map(.final_score) | add/length)})' results.jsonl

# Lowest scoring turns across all conversations
jq '.turn_results[] | {conversation_id: .conversation_id, turn_index, turn_score} | select(.turn_score < 0.5)' results.jsonl
```

---

## See Also

- [Kendall's Tau](kendalls-tau.md) — rank correlation with human judgment
- [Cohen's Kappa](cohens-kappa.md) — verdict agreement with human labels
- [Confusion Matrix](confusion-matrix.md) — verdict classification analysis
- [Configuration](../getting-started/configuration.md) — tuning thresholds and weights
