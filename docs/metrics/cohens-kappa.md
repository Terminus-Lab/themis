# Cohen's Kappa

**Role**: Secondary metric for industry reporting and stakeholder communication.

Measures categorical agreement between two raters, correcting for agreement that could occur by chance alone.

---

## Formula

```
κ = (p_o - p_e) / (1 - p_e)

p_o = observed agreement (proportion of exact matches)
p_e = expected agreement by chance
```

Range: −1 to +1.
- κ = 1: Perfect agreement
- κ = 0: Agreement no better than chance
- κ < 0: Systematic disagreement

---

## Why Not Just Use Accuracy

A judge that always predicts `pass` achieves 85% accuracy on a dataset where 85% of records are `pass`. But κ = 0 — the agreement is entirely explained by the class distribution, not by the judge doing any real evaluation. Kappa exposes this.

---

## Example

10 responses evaluated by human and judge:

| Response | Human | Judge | Match? |
|----------|-------|-------|--------|
| R1 | pass | pass | ✓ |
| R2 | pass | pass | ✓ |
| R3 | pass | review | ✗ |
| R4 | review | review | ✓ |
| R5 | review | pass | ✗ |
| R6 | fail | fail | ✓ |
| R7 | fail | review | ✗ |
| R8 | pass | pass | ✓ |
| R9 | review | review | ✓ |
| R10 | fail | fail | ✓ |

Distribution: Human — 4 pass, 3 review, 3 fail. Judge — 5 pass, 3 review, 2 fail.

**p_o** = 7/10 = 0.70

**p_e** (chance agreement per class):
- pass: (4/10) × (5/10) = 0.20
- review: (3/10) × (3/10) = 0.09
- fail: (3/10) × (2/10) = 0.06
- **p_e = 0.35**

```
κ = (0.70 - 0.35) / (1 - 0.35) = 0.54 → "Moderate agreement"
```

---

## Interpretation Scale (Landis & Koch)

| Range | Meaning |
|-------|---------|
| κ > 0.80 | Almost perfect |
| κ = 0.60–0.80 | Substantial — industry standard |
| κ = 0.40–0.60 | Moderate — acceptable |
| κ = 0.20–0.40 | Fair — needs improvement |
| κ < 0.20 | Poor |

---

## Kappa vs Kendall's τ

| Metric | Measures | Decision role |
|--------|----------|--------------|
| **Kendall's τ** | Rank correlation | **Primary** — pass/fail gate |
| **Cohen's κ** | Categorical agreement | **Secondary** — reporting only |

Kappa does **not** override τ for deployment decisions.

**High κ, low τ** → verdicts are correct but confidence scores don't correlate. Fix: adjust aggregation method or stage weights.

**High τ, low κ** → scores correlate but verdict thresholds are miscalibrated. Fix: adjust `VERDICT_PASS_THRESHOLD` / `VERDICT_REVIEW_THRESHOLD`.

---

## In Themis Validation Output

```bash
./bin/themis-cli validate-events -i human_annotated.jsonl -c 0.3
```

```json
{
  "correlation_metrics": {
    "kendalls_tau": 0.45,
    "interpretation": "Moderate to strong agreement",
    "passed": true
  },
  "agreement_metrics": {
    "cohens_kappa": 0.62,
    "interpretation": "Substantial agreement"
  }
}
```

**Always inspect the confusion matrix alongside kappa** — kappa alone won't show if the judge never uses a particular verdict class.

---

## Next Steps

- [Confusion Matrix](confusion-matrix.md) — per-class error breakdown
- [Interpretation Guide](interpretation-guide.md) — decision framework and failure patterns
