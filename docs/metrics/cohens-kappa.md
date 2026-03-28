---
title: Weighted Kappa — Verdict Agreement
description: Measuring agreement between Themis verdicts and human labels using severity-weighted kappa
tags: [metrics, validation, cohens-kappa, weighted-kappa, agreement, human-annotation]
---

# Weighted Kappa — Verdict Agreement

## What It Is

Weighted κ (kappa) measures the agreement between Themis verdict labels (`pass`/`review`/`fail`) and human-assigned labels, **correcting for chance agreement** and **penalizing larger mismatches more**.

Raw accuracy (percentage match) is misleading — if 80% of conversations are `pass`, a system that always predicts `pass` achieves 80% accuracy without learning anything. κ corrects for this. The weighted variant goes further: predicting `fail` when the human said `pass` (2 steps apart) is penalized more than predicting `review` when the human said `pass` (1 step apart).

- κ = **1.0** — perfect agreement
- κ = **0.0** — agreement no better than chance
- κ < **0.0** — worse than chance (systematic disagreement)

| κ | Interpretation |
|---|----------------|
| 0.80 – 1.00 | Almost perfect |
| 0.60 – 0.80 | Substantial |
| 0.40 – 0.60 | Moderate |
| 0.20 – 0.40 | Fair |
| < 0.20 | Slight or poor |

A production-ready judge setup should achieve κ > 0.60.

---

## When to Use It

Use weighted κ when your dataset has human-assigned verdict labels (not scores). It directly measures whether Themis's `pass`/`review`/`fail` classification matches human expert judgment on the same conversations, with appropriate penalties for severity of disagreement.

---

## Input Format

```json
{"conversation_id":"conv-001","human_annotation":"pass","agent":{"name":"my-agent","version":"1.0"},"turns":[...]}
{"conversation_id":"conv-002","human_annotation":"fail","agent":{"name":"my-agent","version":"1.0"},"turns":[...]}
{"conversation_id":"conv-003","human_annotation":"review","agent":{"name":"my-agent","version":"1.0"},"turns":[...]}
```

`human_annotation` must be one of: `"pass"`, `"review"`, `"fail"`.

The CLI computes weighted κ automatically when annotations are present:

```bash
go run cmd/batch/main.go evaluate -i resources/annotated_sample.jsonl -f summary
```

---

## Why Weighted (Not Unweighted)

Themis uses **linear weighted kappa** exclusively. The verdict labels have a natural ordering (`fail < review < pass`), so unweighted kappa — which treats all mismatches equally — discards useful information. Weighted kappa reflects that misclassifying `fail` as `pass` is a worse error than misclassifying it as `review`.

---

## Diagnosing Low Kappa

Low κ is usually caused by one of:

| Symptom | Diagnosis | Fix |
|---------|-----------|-----|
| Themis marks `review` where humans say `pass` | Pass threshold too high | Lower `VERDICT_PASS_THRESHOLD` |
| Themis marks `pass` where humans say `fail` | Judges too lenient | Raise `VERDICT_PASS_THRESHOLD` or `VERDICT_REVIEW_THRESHOLD` |
| `review` class rarely agrees | Middle band too wide or too narrow | Tune both thresholds together |
| Agreement on `pass`/`fail` but not `review` | `review` band is poorly positioned | Adjust `VERDICT_REVIEW_THRESHOLD` |

Use the [Confusion Matrix](confusion-matrix.md) alongside κ to pinpoint exactly which label pairs are disagreeing.

---

## Inter-Annotator Kappa

κ can also be computed between two human annotators before involving Themis at all. This tells you how consistent your human labels are — the ceiling for what Themis can achieve.

If two human annotators agree at κ = 0.75, you cannot expect Themis to exceed 0.75 on that dataset. This is the **inter-annotator ceiling**.

```python
# Replace themis labels with a second annotator's labels
kappa_human = cohen_kappa_score(annotator_1, annotator_2,
                                labels=['pass', 'review', 'fail'])
print(f"Inter-annotator κ = {kappa_human:.3f}")
```

---

## See Also

- [Kendall's Tau](kendalls-tau.md) — rank correlation with human scores
- [Confusion Matrix](confusion-matrix.md) — per-class breakdown of agreement
- [Interpretation Guide](interpretation-guide.md) — reading Themis scores
- [Configuration](../getting-started/configuration.md) — tuning verdict thresholds
