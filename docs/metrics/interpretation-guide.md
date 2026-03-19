# Validation Interpretation Guide

How to read validation results, decide whether to deploy, and diagnose failures.

---

## Decision Framework

```
Step 1 — Kendall's τ (PRIMARY: pass/fail gate)
  τ ≥ 0.3 → proceed to Step 2
  τ < 0.3 → REJECT, do not deploy

Step 2 — Cohen's Kappa (SECONDARY: credibility)
  κ ≥ 0.6 → deploy immediately
  κ = 0.4–0.6 → deploy with monitoring
  κ < 0.4 → investigate confusion matrix before deploying

Step 3 — Confusion Matrix (DIAGNOSTIC: what to fix)
  Used to understand error types, not as a gate
```

| τ | κ | Decision |
|---|---|----------|
| ≥ 0.3 | ≥ 0.6 | ✅ Deploy |
| ≥ 0.3 | 0.4–0.6 | ⚠️ Deploy with monitoring |
| ≥ 0.3 | < 0.4 | ⚠️ Deploy with caution — inspect matrix |
| < 0.3 | any | ❌ Reject |

---

## When Metrics Disagree

### High κ, Low τ (κ=0.75, τ=0.28)

Verdicts are correct but confidence scores don't reflect quality ranking.

**Cause**: Aggregation or weight issues.
**Fix**:
1. Try all 4 `JUDGE_AGGREGATION_METHOD` options
2. Adjust `PRECHECK_WEIGHT` / `LLM_JUDGE_WEIGHT`
3. Consider disabling Stage 1 if it's adding noise

### High τ, Low κ (τ=0.65, κ=0.42)

Scores correlate well but verdict thresholds are miscalibrated.

**Fix**:
```bash
# Check score distribution vs human annotations
jq -r '[.confidence, .human_annotation] | @csv' results.jsonl | sort -t, -k2

# Adjust thresholds
VERDICT_PASS_THRESHOLD=0.75      # lower if too many passes→review
VERDICT_REVIEW_THRESHOLD=0.45    # lower if too many reviews→fail
```

### Both Low (τ=0.22, κ=0.35)

Fundamental judge issue — do not deploy.

**Fix**: Check the confusion matrix for dominant failure pattern (see below), then tune `configs/judges.yaml`.

---

## Failure Patterns

### Style-over-substance bias

**Signs**: τ ≈ 0.2, high fail→pass rate (30–40%). Polite but wrong answers rated as `pass`.

```
fail→pass: 15-20 cases
fail→review: 10-15 cases
```

**Fix**:
- Reduce coherence/instruction weight: 0.15 → 0.08
- Increase correctness/completeness weight: 0.15 → 0.24

---

### Boundary confusion (fail ↔ review)

**Signs**: τ ≈ 0.45, errors go both directions (fail→review AND review→fail). Review F1 < 0.55.

**Fix**:
1. Add explicit review criteria to judge prompts:
   ```
   "review" when the answer is correct but incomplete, or adequate but improvable.
   ```
2. Widen the threshold gap:
   ```env
   VERDICT_PASS_THRESHOLD=0.75
   VERDICT_REVIEW_THRESHOLD=0.50
   ```

---

### Conservative bias (review → fail)

**Signs**: τ ≈ 0.50, short-but-correct answers rejected, review recall < 60%.

**Fix**:
1. Update completeness prompt: accept terse answers that are factually correct
2. Lower review threshold:
   ```env
   VERDICT_REVIEW_THRESHOLD=0.45
   ```

---

### Leniency bias (fail → pass/review)

**Signs**: τ ≈ 0.25, fail→review and fail→pass both high.

**Fix**: Strengthen the correctness judge — increase weight, update prompt to penalize empty or evasive answers.

---

## Per-Class Targets

| Class | Recall target | Priority |
|-------|--------------|----------|
| **fail** | ≥ 75% | **Critical** — must catch bad answers |
| **pass** | ≥ 90% | High — don't block good answers |
| **review** | ≥ 65% | Acceptable — boundary cases are inherently hard |

Fail→pass errors are the most dangerous. Zero is the goal; < 4% is acceptable.

---

## Pre-Deployment Checklist

```
□ Kendall's τ ≥ 0.3
□ Cohen's Kappa ≥ 0.6 (or documented reason for lower)
□ Fail recall ≥ 75%
□ Pass recall ≥ 90%
□ Fail→pass rate < 4%
□ Pass→fail rate < 6%
```

## Post-Deployment Monitoring

```
□ Spot-check random samples in first week
□ Re-validate after any judge configuration change
□ Re-validate quarterly with fresh human annotations
```

---

## Formulas Reference

**Kendall's τ:**
```
τ = (C - D) / sqrt((C + D + T_x) × (C + D + T_y))
C = concordant pairs, D = discordant, T_x/T_y = ties
```

**Cohen's κ:**
```
κ = (p_o - p_e) / (1 - p_e)
p_o = observed agreement, p_e = expected by chance
```

**Per-class metrics:**
```
Precision = TP / (TP + FP)
Recall    = TP / (TP + FN)
F1        = 2 × (P × R) / (P + R)
```

---

## Resources

- Sample datasets: `resources/validation_success_dataset.jsonl`, `resources/validation_failed_dataset.jsonl`
- Judge configuration: `configs/judges.yaml`
- CLI: `./bin/themis-cli validate-events -i dataset.jsonl -c 0.3`
