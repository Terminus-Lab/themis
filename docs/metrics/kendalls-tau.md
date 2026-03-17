# Kendall's τ

**Role**: Primary pass/fail gate for judge validation.

Measures whether the LLM judge's verdict rankings correlate with human rankings. If the judge consistently ranks responses the same way humans do, it can be trusted at scale.

---

## How It Works

Kendall's τ compares every pair of records and checks whether both raters (human and judge) agree on which one is better:

- **Concordant pair**: Both agree on ranking → +1
- **Discordant pair**: They disagree → -1
- **Tied pair**: One or both have equal scores → ignored

```
τ = (concordant - discordant) / total_pairs
```

Range: −1 to +1. Higher = stronger agreement.

---

## Themis Verdict Ranking

Themis converts verdicts to ranks before computing τ:
- `pass` = 2
- `review` = 1
- `fail` = 0

### Example (5 records)

| Event | Human | Judge | Human Rank | Judge Rank |
|-------|-------|-------|-----------|-----------|
| E1 | pass | pass | 2 | 2 |
| E2 | pass | review | 2 | 1 |
| E3 | review | review | 1 | 1 |
| E4 | fail | fail | 0 | 0 |
| E5 | fail | review | 0 | 1 |

Total pairs = 10. Results: 5 concordant, 0 discordant, 5 ties.

```
τ = (5 - 0) / 10 = 0.5
```

τ = 0.5 is "moderate to strong" — **passes** the default threshold of 0.3.

Note: τ < 1.0 even with zero discordant pairs because ties reduce the denominator. Perfect τ = 1.0 requires every verdict to match exactly.

---

## Interpretation

| Range | Meaning | Action |
|-------|---------|--------|
| τ ≥ 0.6 | Strong | Deploy immediately |
| τ = 0.4–0.6 | Moderate | Deploy with monitoring |
| τ = 0.3–0.4 | Weak but acceptable | Deploy, plan improvements |
| τ < 0.3 | Inadequate | **Reject — fix judges first** |

**Default threshold**: τ ≥ 0.3. Represents the minimum positive correlation needed to trust the judge's rankings. In practice, human–LLM agreement rarely exceeds τ = 0.7–0.8, so 0.6+ is excellent.

---

## Usage

```bash
./bin/themis-cli validate -i human_annotated.jsonl -c 0.3
```

Output includes:
```json
{
  "kendall_tau": 0.5,
  "interpretation": "Moderate to strong agreement",
  "passed": true,
  "threshold": 0.3
}
```

---

## Why τ Over Accuracy

Simple accuracy is misleading when classes are imbalanced (e.g., 80% `pass`). A judge that always predicts `pass` achieves 80% accuracy but τ ≈ 0 — it's not actually ranking quality, just exploiting the distribution.

Kendall's τ is scale-invariant and handles ties, making it robust for the ordinal verdict system Themis uses.

---

## Next Steps

- [Cohen's Kappa](cohens-kappa.md) — secondary metric for stakeholder reporting
- [Interpretation Guide](interpretation-guide.md) — decision framework and failure pattern diagnosis
