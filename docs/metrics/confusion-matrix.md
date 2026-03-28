---
title: Confusion Matrix — Verdict Classification Analysis
description: Per-class breakdown of where Themis verdicts agree or disagree with human labels
tags: [metrics, validation, confusion-matrix, human-annotation]
---

# Confusion Matrix — Verdict Classification Analysis

## What It Is

The confusion matrix shows, for each human-assigned label, how Themis classified that conversation. It breaks down the overall agreement into per-class detail — letting you see not just *how often* Themis is wrong, but *which direction* it errs.

For Themis's three-class verdict system:

```
                    Themis Prediction
                  fail  review  pass
                ┌──────┬───────┬──────┐
Human   fail    │  TP  │  FN   │  FN  │
Label   review  │  FP  │  TP   │  FP  │
        pass    │  FP  │  FP   │  TP  │
                └──────┴───────┴──────┘
```

Diagonal cells = correct classifications.
Off-diagonal cells = disagreements.

---

## When to Use It

The confusion matrix is the first thing to look at after computing weighted κ. κ gives you a single number; the matrix tells you *where* the errors are.

---

## Input Format

Add a `human_annotation` field at the conversation level (compatible with [stamper](https://github.com/Terminus-Lab/stamper) output):

```json
{"conversation_id":"conv-001","human_annotation":"pass","agent":{"name":"my-agent","version":"1.0"},"turns":[...]}
{"conversation_id":"conv-002","human_annotation":"fail","agent":{"name":"my-agent","version":"1.0"},"turns":[...]}
```

The CLI computes the confusion matrix automatically when annotations are present:

```bash
go run cmd/batch/main.go evaluate -i annotated-dataset.jsonl -f summary
```

Example output:

```json
{
  "correlation_report": {
    "annotated_count": 15,
    "kendall_tau": 0.79,
    "weighted_kappa": 0.93,
    "confusion_matrix": {
      "labels": ["fail", "review", "pass"],
      "matrix": [
        [5, 0, 0],
        [1, 4, 0],
        [0, 0, 5]
      ]
    },
    "disagreements": [
      {
        "conversation_id": "ann-006",
        "human_label": "review",
        "themis_label": "fail",
        "human_score": 0.62,
        "themis_score": 0.3125
      }
    ]
  }
}
```

The `disagreements` array lists every per-conversation mismatch, including the Themis score (and human score when provided), making it easy to audit borderline cases.

---

## Reading the Matrix

### Common Patterns and Fixes

**Themis over-calls `pass` (human says `review` or `fail`, Themis says `pass`):**
- High `FP` in the pass column for fail and review rows
- Themis is too lenient
- Fix: raise `VERDICT_PASS_THRESHOLD` (e.g. 0.8 → 0.85) or raise `VERDICT_REVIEW_THRESHOLD`

**Themis over-calls `fail` (human says `pass`, Themis says `fail`):**
- High value in the fail column for the pass row
- Themis is too strict
- Fix: lower `VERDICT_REVIEW_THRESHOLD` or lower `VERDICT_PASS_THRESHOLD`

**`review` class has poor precision and recall:**
- Most errors involve the review band
- The `review` threshold band is either too wide or misplaced
- Fix: adjust the gap between `VERDICT_REVIEW_THRESHOLD` and `VERDICT_PASS_THRESHOLD`

**`fail`/`pass` agree well but `review` is noisy:**
- Common pattern — the middle class is harder to classify
- Consider widening the review band or treating review as a soft pass/fail based on domain needs

### Per-Class Metrics

| Metric | Meaning |
|--------|---------|
| **Precision** | Of conversations Themis called `pass`, what % were actually `pass`? |
| **Recall** | Of conversations that were actually `pass`, what % did Themis catch? |
| **F1** | Harmonic mean of precision and recall |

For a production evaluation system:
- High **recall on `fail`** is critical — you don't want to miss bad conversations
- High **precision on `pass`** matters if `pass` means "ship this agent"

---

## Severity-Weighted Analysis

Not all errors are equal. A conversation labelled `fail` by humans but predicted `pass` by Themis is more harmful than a `fail` predicted as `review`. Compute a weighted error rate:

```python
# Assign severity weights to off-diagonal cells
# fail→review: weight 1, fail→pass: weight 2
# review→fail: weight 1, review→pass: weight 1
# pass→review: weight 1, pass→fail: weight 2

weight_matrix = [
    [0, 1, 2],   # true: fail
    [1, 0, 1],   # true: review
    [2, 1, 0],   # true: pass
]

total = sum(sum(row) for row in cm)
weighted_errors = sum(cm[i][j] * weight_matrix[i][j]
                      for i in range(3) for j in range(3))
print(f"Weighted error rate: {weighted_errors / total:.3f}")
```

---

## See Also

- [Cohen's Kappa](cohens-kappa.md) — single-number agreement metric
- [Kendall's Tau](kendalls-tau.md) — rank correlation with human scores
- [Interpretation Guide](interpretation-guide.md) — reading Themis scores
- [Configuration](../getting-started/configuration.md) — tuning verdict thresholds
