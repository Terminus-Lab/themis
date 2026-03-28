---
title: Kendall's Tau — Judge Rank Correlation
description: How to validate that Themis rankings correlate with human judgment
tags: [metrics, validation, kendalls-tau, correlation, human-annotation]
---

# Kendall's Tau — Judge Rank Correlation

## What It Is

Kendall's τ (tau) measures how well Themis's ranking of conversations agrees with human-assigned rankings. It answers: **does Themis rank conversations the same way a human expert would?**

- τ = **1.0** — perfect agreement (Themis and humans rank identically)
- τ = **0.0** — no correlation (random agreement)
- τ = **-1.0** — perfect inversion (Themis ranks the best conversations as worst)

A well-calibrated judge setup should achieve τ > 0.7. τ > 0.85 is excellent.

---

## When to Use It

Run Kendall's τ when:
- Setting up Themis for the first time on a new domain — validate the judges before trusting verdicts in production
- Changing judge configuration (adding/removing judges, changing weights or models)
- Investigating suspiciously high or low verdict rates from the batch CLI

This is an **offline, dataset-level metric** — not part of the per-conversation pipeline.

---

## Input Format

To compute τ, your dataset needs a `human_score` (float, 0.0–1.0) at the conversation level. Supports 1–5 scale scores too — the CLI normalizes them automatically.

```json
{"conversation_id":"conv-001","human_score":0.90,"agent":{"name":"my-agent","version":"1.0"},"turns":[...]}
{"conversation_id":"conv-002","human_score":0.45,"agent":{"name":"my-agent","version":"1.0"},"turns":[...]}
{"conversation_id":"conv-003","human_score":0.72,"agent":{"name":"my-agent","version":"1.0"},"turns":[...]}
```

The CLI computes τ automatically when `human_score` fields are present:

```bash
go run cmd/batch/main.go evaluate -i annotated-dataset.jsonl -f summary
```

---

## Interpreting the Output

| τ | Interpretation |
|---|----------------|
| ≥ 0.85 | Excellent — judges are well-calibrated for this domain |
| 0.70 – 0.85 | Good — minor miscalibration, acceptable for most use cases |
| 0.50 – 0.70 | Moderate — review judge weights and prompts |
| < 0.50 | Poor — judges may not be appropriate for this domain |

---

## Improving a Low τ

**Low τ usually means one of:**

1. **Wrong judge weights** — the dimension humans care about most (e.g. completeness) has too little weight. Adjust `weight` in `judges.yaml`.

2. **Wrong model** — the LLM used as judge doesn't understand the domain well. Try a larger model or a domain-specific one.

3. **Prompt mismatch** — judge prompts evaluate the wrong thing for your domain. Rewrite prompts to match what your human annotators were assessing.

4. **Domain shift** — judges calibrated for general Q&A don't transfer to code review or medical Q&A without tuning.

**Iterative process:**
```
annotate sample → run batch → compute τ → adjust config → repeat
```

---

## Dataset Size

τ is unreliable on small samples. Minimum recommended sizes:

| Purpose | Minimum conversations |
|---------|----------------------|
| Sanity check | 30 |
| Configuration tuning | 100 |
| Production validation | 300+ |

The p-value from `scipy.stats.kendalltau` tells you whether the correlation is statistically significant. At n=30, τ > 0.3 is typically significant (p < 0.05).

---

## See Also

- [Cohen's Kappa](cohens-kappa.md) — verdict-level agreement (pass/review/fail vs. human labels)
- [Confusion Matrix](confusion-matrix.md) — where verdicts disagree with human labels
- [Interpretation Guide](interpretation-guide.md) — reading Themis scores
- [Configuration](../getting-started/configuration.md) — tuning judge weights
