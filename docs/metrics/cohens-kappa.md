---
title: Cohen's Kappa — Verdict Agreement
description: Measuring agreement between Themis verdicts and human labels
tags: [metrics, validation, cohens-kappa, agreement, human-annotation]
---

# Cohen's Kappa — Verdict Agreement

## What It Is

Cohen's κ (kappa) measures the agreement between Themis verdict labels (`pass`/`review`/`fail`) and human-assigned labels, **correcting for chance agreement**.

Raw accuracy (percentage match) is misleading — if 80% of conversations are `pass`, a system that always predicts `pass` achieves 80% accuracy without learning anything. κ corrects for this.

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

Use κ when your dataset has human-assigned verdict labels (not scores). It directly measures whether Themis's `pass`/`review`/`fail` classification matches human expert judgment on the same conversations.

> **Status:** The batch CLI does not yet compute κ automatically. The field `human_label` is not yet part of `ConversationEvaluationRequest`. When implemented, it will be added as an optional field and the CLI will output a kappa report when human labels are present.

---

## Input Format

Planned input format (conversation level):

```json
{"conversation_id":"conv-001","human_label":"pass","agent":{"name":"my-agent","version":"1.0"},"turns":[...]}
{"conversation_id":"conv-002","human_label":"fail","agent":{"name":"my-agent","version":"1.0"},"turns":[...]}
{"conversation_id":"conv-003","human_label":"review","agent":{"name":"my-agent","version":"1.0"},"turns":[...]}
```

`human_label` must be one of: `"pass"`, `"review"`, `"fail"`.

---

## Computing Manually (Current Approach)

**Step 1 — Run batch evaluation:**
```bash
./bin/themis-cli evaluate -input annotated-dataset.jsonl -output results.jsonl
```

**Step 2 — Compute κ:**
```python
import json
from sklearn.metrics import cohen_kappa_score

# Load
themis = {r['conversation_id']: r['verdict']
          for r in (json.loads(l) for l in open('results.jsonl'))}
human  = {r['conversation_id']: r['human_label']
          for r in (json.loads(l) for l in open('annotated-dataset.jsonl'))
          if 'human_label' in r}

ids = sorted(set(themis) & set(human))
t = [themis[i] for i in ids]
h = [human[i]  for i in ids]

kappa = cohen_kappa_score(h, t, labels=['pass', 'review', 'fail'])
print(f"Cohen's κ = {kappa:.3f}  (n = {len(ids)})")
```

---

## Weighted Kappa

For ordered labels (fail < review < pass), a disagreement of `fail` vs `pass` is worse than `fail` vs `review`. Use **linear weighted kappa** to penalize larger disagreements more:

```python
kappa_weighted = cohen_kappa_score(h, t,
                                   labels=['fail', 'review', 'pass'],
                                   weights='linear')
print(f"Weighted κ = {kappa_weighted:.3f}")
```

Weighted kappa is usually more informative than unweighted for a 3-class ordered classification.

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
