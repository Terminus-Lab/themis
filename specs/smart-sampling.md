# Smart Sampling — Spec

**Status:** Proposal
**Date:** 2026-03-19
**Scope:** Conversations (events to follow)

---

## Problem

Random 25% sampling has a fundamental bias: most live traffic passes. If 80% of conversations are `pass`, a random sample gives you mostly easy cases — the ones least useful for calibration. You learn nothing new about where the judge is wrong.

---

## What the DB Gives Us

Every conversation record has:

| Field | Type | Useful for |
|---|---|---|
| `verdict` | pass / review / fail | class distribution |
| `confidence` | 0.0 – 1.0 | distance from decision boundary |
| `stage_scores` | JSON array | judge disagreement |
| `turn_count` | int | complexity proxy |
| `agent_name / version` | text | coverage |
| `created_at` | timestamp | temporal coverage |

No extra fields needed — all strategies below use what's already stored.

---

## Sampling Strategies

### 1. Stratified by Verdict *(highest priority)*

Guarantee representation of all three classes regardless of their frequency in live traffic.

```
mode: stratified_verdict
distribution:
  pass:   40%
  review: 30%
  fail:   30%
```

**Why:** A judge that never sees `fail` examples in validation will appear good even when it isn't. Oversampling rare verdicts is the single biggest improvement over random.

---

### 2. Uncertainty Priority

Rank conversations by distance from the two decision thresholds and sample from the uncertain band first.

```
pass threshold:   0.80
review threshold: 0.50

uncertain band:   [0.42 – 0.58]  (near review/fail boundary)
                  [0.72 – 0.88]  (near pass/review boundary)
```

Conversations within these bands have the highest annotation value — the judge is most likely to be wrong here, and human labels here move τ the most.

**Why:** Annotating cases the judge is confident about (confidence 0.95+) rarely improves calibration.

---

### 3. Judge Disagreement

When `stage_scores` shows high variance across judges within a conversation, that conversation is a priority.

```
disagreement = stddev(stage_scores[].score) > 0.25
```

A conversation where `relevance = 0.9` but `faithfulness = 0.2` is more interesting than one where all judges agree at 0.85.

**Why:** Disagreement signals edge cases the aggregation formula may be handling wrong.

---

### 4. Agent / Version Coverage

Ensure every active agent and agent version gets at least N conversations sampled, regardless of traffic volume.

```
min_per_agent_version: 5   # guaranteed floor
remainder: filled by other strategies
```

**Why:** A new agent version with 2% of traffic would get 0–1 samples with pure random or stratified. Coverage sampling guarantees it gets annotated early.

---

### 5. Temporal Spread

Divide the date range into equal windows and sample proportionally from each. Prevents recency bias when traffic spikes.

```
date_range: 30 days → 4 weekly windows, each contributes 25%
```

**Why:** A traffic spike last week would dominate a random sample. Temporal spread detects drift over time and ensures week 1 is as well-represented as week 4.

---

## API Design

### Extended Request Body

```json
POST /api/v1/validation/sample/conversations/download
{
  "start_date": "2026-03-01T00:00:00Z",
  "end_date":   "2026-03-19T00:00:00Z",
  "percentage": 25,
  "min_size": 50,
  "max_size": 500,

  "strategy": "stratified_verdict",

  "stratified_verdict": {
    "pass":   40,
    "review": 30,
    "fail":   30
  },

  "uncertainty": {
    "enabled": true,
    "priority_weight": 0.5
  },

  "disagreement": {
    "enabled": true,
    "stddev_threshold": 0.25
  },

  "coverage": {
    "min_per_agent_version": 5
  }
}
```

`strategy` selects the primary mode. Other fields are optional modifiers applied on top.

**Backward compatible:** omitting `strategy` keeps current random behavior.

---

## Strategy Comparison

| Strategy | Best for | Requires |
|---|---|---|
| `random` (current) | Baseline, quick check | nothing |
| `stratified_verdict` | Calibration, first annotation run | verdict column |
| `uncertainty` | Prompt tuning, boundary cases | confidence column |
| `disagreement` | Debugging aggregation method | stage_scores |
| `coverage` | Multi-agent deployments, new versions | agent_name/version |
| `temporal` | Drift detection | created_at |

---

## Implementation Notes

- All strategies execute as SQL — no post-processing needed
- `stratified_verdict`: `ORDER BY RANDOM() LIMIT N` per verdict bucket
- `uncertainty`: `WHERE confidence BETWEEN 0.42 AND 0.58 OR confidence BETWEEN 0.72 AND 0.88 ORDER BY ABS(confidence - 0.50)`
- `disagreement`: requires parsing `stage_scores` JSON; computed as SQLite `json_each` or PostgreSQL `jsonb_array_elements`; alternatively pre-compute stddev at write time and store as a column
- `coverage`: `GROUP BY agent_name, agent_version` + guaranteed minimum with fallback fill
- Strategies can be combined: coverage guarantees → uncertainty fill → random remainder

---

## Recommended First Run

For a team starting annotation from scratch:

```json
{
  "strategy": "stratified_verdict",
  "stratified_verdict": { "pass": 40, "review": 30, "fail": 30 },
  "coverage": { "min_per_agent_version": 5 },
  "min_size": 50
}
```

This gives balanced class distribution across all agents. After the first validate-conversations run, switch to `uncertainty` to focus on the boundary where the judge is weakest.
