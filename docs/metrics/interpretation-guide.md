# Validation Metrics Interpretation Guide

**Purpose**: How to interpret validation results and diagnose judge performance issues.

**Audience**: Developers, ML engineers, and stakeholders evaluating LLM judge quality.

---

## Table of Contents

1. [Decision Framework](#decision-framework)
2. [Metric Interpretation Scales](#metric-interpretation-scales)
3. [When Metrics Disagree](#when-metrics-disagree)
4. [Common Failure Patterns](#common-failure-patterns)
5. [Confusion Matrix Diagnostics](#confusion-matrix-diagnostics)
6. [Troubleshooting Flowchart](#troubleshooting-flowchart)
7. [Quick Reference Checklist](#quick-reference-checklist)
8. [Case Studies](#case-studies)

---

## Decision Framework

### Three-Step Validation Process

```
┌──────────────────────────────────────────────────────────┐
│ Step 1: Check Kendall's τ (PRIMARY - Pass/Fail Decision) │
├──────────────────────────────────────────────────────────┤
│ τ ≥ 0.3? → PASS (judges are reliable)                   │
│ τ < 0.3? → FAIL (judges are unreliable)                 │
└──────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────┐
│ Step 2: Check Cohen's Kappa (SECONDARY - Credibility)   │
├──────────────────────────────────────────────────────────┤
│ κ ≥ 0.6? → Excellent (industry standard met)            │
│ κ = 0.4-0.6? → Acceptable (report with caveats)         │
│ κ < 0.4? → Below standard (investigate issues)          │
└──────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────┐
│ Step 3: Analyze Confusion Matrix (DIAGNOSTIC)           │
├──────────────────────────────────────────────────────────┤
│ • Identify error types (fail→pass, review→fail, etc.)   │
│ • Calculate error rates per class                        │
│ • Prioritize fixes by criticality × frequency           │
└──────────────────────────────────────────────────────────┘
```

### Decision Rules

| Kendall's τ | Cohen's Kappa | Decision | Next Steps |
|-------------|---------------|----------|------------|
| ≥ 0.3 | ≥ 0.6 | ✅ **Deploy to production** | Monitor first 100 evaluations |
| ≥ 0.3 | 0.4-0.6 | ⚠️ **Deploy with monitoring** | Plan improvements, track metrics |
| ≥ 0.3 | < 0.4 | ⚠️ **Deploy with caution** | Investigate confusion matrix |
| < 0.3 | Any | ❌ **REJECT - Do not deploy** | Fix judges, re-validate |

**Key Principle**: Kendall's τ is the **gatekeeper** (pass/fail), Kappa is the **reporter** (how good), Confusion Matrix is the **debugger** (what to fix).

---

## Metric Interpretation Scales

### Kendall's τ (Rank Correlation)

Measures whether judge's confidence scores correlate with human quality rankings.

| Range | Interpretation | Action Required |
|-------|----------------|-----------------|
| **τ ≥ 0.6** | Strong agreement | ✅ Deploy immediately - Excellent performance |
| **τ = 0.4-0.6** | Moderate agreement | ✅ Deploy with monitoring - Good performance |
| **τ = 0.3-0.4** | Weak but acceptable | ⚠️ Deploy, plan improvements - Minimum acceptable |
| **τ < 0.3** | Inadequate | ❌ **REJECT** - Fix required before deployment |

**Threshold Justification**: τ = 0.3 represents "weak positive correlation" - the minimum needed to trust judge rankings. Below this, judges cannot reliably distinguish quality levels.

**Real-World Context**: Human-LLM ranking correlations rarely exceed τ = 0.7-0.8. A score of 0.6+ is excellent.

---

### Cohen's Kappa (Categorical Agreement)

Measures categorical agreement accounting for chance.

| Range | Interpretation | Landis & Koch Scale | Practical Meaning |
|-------|----------------|---------------------|-------------------|
| **κ > 0.8** | Almost perfect | Excellent | Near-human agreement |
| **κ = 0.6-0.8** | Substantial | Good | Industry standard met |
| **κ = 0.4-0.6** | Moderate | Acceptable | Usable but room for improvement |
| **κ = 0.2-0.4** | Fair | Below standard | Needs improvement |
| **κ < 0.2** | Slight | Poor | Fundamentally broken |

**Note**: Kappa is a **reporting metric** - use it for papers, stakeholder communication, and industry comparisons. It does NOT override Kendall's τ for pass/fail decisions.

---

### Per-Class Metrics (Precision/Recall/F1)

| Metric | Formula | Meaning | Target |
|--------|---------|---------|--------|
| **Precision** | TP / (TP + FP) | Of predicted class, how many correct? | ≥ 0.75 |
| **Recall** | TP / (TP + FN) | Of actual class, how many caught? | ≥ 0.75 |
| **F1 Score** | 2 × (Precision × Recall) / (Precision + Recall) | Harmonic mean of precision/recall | ≥ 0.75 |

**Class-Specific Targets**:

| Class | Precision Target | Recall Target | F1 Target | Priority |
|-------|------------------|---------------|-----------|----------|
| **fail** | ≥ 0.80 | ≥ 0.75 | ≥ 0.77 | **CRITICAL** - Must catch bad answers |
| **pass** | ≥ 0.85 | ≥ 0.90 | ≥ 0.87 | Important - Don't reject good answers |
| **review** | ≥ 0.65 | ≥ 0.65 | ≥ 0.65 | Acceptable - Boundary cases are hard |

---

## When Metrics Disagree

### Scenario A: High Kappa, Low Tau

**Example**: κ = 0.75, τ = 0.28

**Meaning**: Judge gets categories right (pass/fail) but confidence scores don't reflect quality ranking.

**Symptoms**:
- Verdicts are mostly correct
- But confidence scores are random/inverted
- Judge says "pass" when it should, but confidence varies wildly

**Diagnosis**: Verdict logic is OK, but judge scoring is broken.

**Root Causes**:
- Aggregation method issues (try different methods: weighted_average vs harmonic_mean)
- Stage 1/Stage 2 weight imbalance
- Individual judge scores are noisy

**Fixes**:
1. Check `JUDGE_AGGREGATION_METHOD` - try all 4 methods
2. Adjust `PRECHECK_WEIGHT` and `LLM_JUDGE_WEIGHT` in `.env`
3. Review individual judge score distributions
4. Consider disabling Stage 1 prechecks if causing noise

**Priority**: Medium - Judge works but confidence scores are misleading

---

### Scenario B: High Tau, Low Kappa

**Example**: κ = 0.42, τ = 0.65

**Meaning**: Judge ranks quality well but picks wrong categories.

**Symptoms**:
- Confidence scores correlate with quality
- But verdict thresholds are miscalibrated
- Example: Score 0.78 should be "pass" but verdict is "review"

**Diagnosis**: Good score correlation, but verdict thresholds are wrong.

**Root Causes**:
- `VERDICT_PASS_THRESHOLD` too high/low
- `VERDICT_REVIEW_THRESHOLD` too high/low
- Thresholds don't match score distribution

**Fixes**:
1. Analyze score distribution per human annotation:
   ```bash
   jq -r '[.confidence, .human_annotation] | @csv' results.jsonl | sort -t, -k2
   ```
2. Adjust thresholds:
   ```bash
   # If too many passes → reviews
   VERDICT_PASS_THRESHOLD=0.75  # Lower (was 0.8)

   # If too many reviews → fails
   VERDICT_REVIEW_THRESHOLD=0.45  # Lower (was 0.5)
   ```
3. Re-validate after threshold changes

**Priority**: Medium - Easy fix, high impact

---

### Scenario C: Both Low

**Example**: κ = 0.35, τ = 0.22

**Meaning**: Judge is fundamentally broken.

**Symptoms**:
- Categories wrong
- Scores don't correlate
- High error rates across all classes

**Diagnosis**: Systematic issues requiring major reconfiguration.

**Root Causes**:
- Judge weights prioritize wrong dimensions (style over correctness)
- Judge prompts are poorly calibrated
- Missing or disabled critical judges (e.g., correctness)

**Fixes**: See [Case Study: Failed Validation](../../resources/validation_failed_dataset_interpretation.md)

**Priority**: P0 - CRITICAL - Do not deploy

---

### Scenario D: Both High

**Example**: κ = 0.91, τ = 0.63

**Meaning**: Judge working excellently.

**Symptoms**:
- Categories correct
- Scores correlate with quality
- Low error rates

**Diagnosis**: Production-ready.

**Next Steps**: See [Case Study: Successful Validation](../../resources/validation_success_dataset_interpretation.md)

**Priority**: Deploy immediately

---

## Common Failure Patterns

### Pattern Recognition Table

| Pattern Name | τ | κ | Confusion Matrix Signature | Root Cause | Fix |
|--------------|---|---|---------------------------|------------|-----|
| **Style-over-substance** | Low (0.2) | Low (0.3) | High fail→pass errors | Coherence weighted too high | Increase correctness weight, reduce coherence |
| **Boundary confusion** | Medium (0.45) | Medium (0.55) | Bidirectional fail↔review | Unclear review criteria | Define review threshold explicitly |
| **Conservative bias** | Medium (0.50) | Medium (0.60) | High review→fail errors | Completeness too strict | Allow terse but correct answers |
| **Leniency bias** | Low (0.25) | Low (0.35) | High fail→review/pass | Correctness too lenient | Strengthen correctness judge |
| **Perfect pass, poor fail** | Medium (0.48) | High (0.72) | pass: 100%, fail: <60% | Asymmetric calibration | Strengthen fail detection, add examples |
| **Penalizes brevity** | Medium (0.52) | Medium (0.58) | review→fail when answers short | Completeness requires elaboration | Update completeness prompt |

---

### Pattern 1: Style-Over-Substance Bias

**Symptoms**:
- Polite but empty answers rated as "pass"
- Verbose but wrong answers rated higher than terse correct ones
- Coherence/instruction judges dominate scoring

**Example**:
```
Query: "What is the capital of France?"
Answer: "Thank you for asking! France has a rich history of governance..."
Human: fail
Judge: pass (0.82 confidence)
```

**Diagnosis**:
```
Confusion Matrix:
  fail→pass: 15-20 cases (30-40%)
  fail→review: 10-15 cases (20-30%)
```

**Fix**:
1. Reduce coherence weight: 0.15 → 0.08
2. Reduce instruction weight: 0.15 → 0.08
3. Increase correctness weight: 0.15 → 0.24
4. Increase completeness weight: 0.15 → 0.24

---

### Pattern 2: Boundary Confusion (Fail ↔ Review)

**Symptoms**:
- Errors go both directions: fail→review AND review→fail
- Review class has lowest F1 score
- Inconsistent scoring near verdict thresholds

**Example**:
```
Query: "What is HTTP?"
Answer 1: "Protocol for websites" → fail (human: review)
Answer 2: "Something about internet" → review (human: fail)
```

**Diagnosis**:
```
Confusion Matrix:
  fail→review: 10-15 cases
  review→fail: 10-15 cases
  Review F1: 0.50-0.60
```

**Fix**:
1. Add explicit review criteria to all judge prompts:
   ```
   Review verdict criteria:
   - Correct but oversimplified
   - Correct but incomplete
   - Adequate but could be improved
   ```
2. Add review examples to `configs/judges.yaml`
3. Widen threshold gap:
   ```bash
   VERDICT_PASS_THRESHOLD=0.75   # Was: 0.8
   VERDICT_REVIEW_THRESHOLD=0.50 # Was: 0.5 (maintain gap)
   ```

---

### Pattern 3: Conservative Bias (Review → Fail)

**Symptoms**:
- Short but correct answers rejected
- High precision but low recall on review class
- Completeness judge penalizes brevity

**Example**:
```
Query: "What is DNS?"
Answer: "Domain Name System" (context already explained DNS)
Human: review (terse but correct)
Judge: fail (0.38 confidence - too brief)
```

**Diagnosis**:
```
Confusion Matrix:
  review→fail: 15-20 cases (30-40% of reviews)
  Review recall: <60%
```

**Fix**:
1. Update completeness judge prompt:
   ```
   Short answers can be complete if they:
   1. Directly address the question
   2. Are factually correct
   3. Provide key information
   ```
2. Lower review threshold:
   ```bash
   VERDICT_REVIEW_THRESHOLD=0.45  # Was: 0.5 (more inclusive)
   ```

---

## Confusion Matrix Diagnostics

### Step-by-Step Analysis

#### Step 1: Check Diagonal Strength

**Strong Diagonal** (✅ Good):
```
             Predicted
         fail  pass  review
fail      40     5      5     ← 80% recall
pass       2    45      3     ← 90% recall
review     3     5     42     ← 84% recall
```
- All classes ≥ 75% recall → Judge is fundamentally sound

**Weak Diagonal** (❌ Poor):
```
             Predicted
         fail  pass  review
fail      25    15     10     ← 50% recall
pass       8    35      7     ← 70% recall
review    12     8     30     ← 60% recall
```
- Any class < 60% recall → Systematic issues

---

#### Step 2: Identify Critical Errors

**Error Severity Ranking**:

1. **CRITICAL**: fail → pass (promotes bad answers)
   - **Impact**: Bad responses approved without review
   - **Priority**: P0 - Fix immediately
   - **Acceptable**: 0-2 cases (< 4% of fails)

2. **HIGH**: pass → fail (rejects good answers)
   - **Impact**: Good responses needlessly blocked
   - **Priority**: P1 - Fix soon
   - **Acceptable**: 0-3 cases (< 6% of passes)

3. **MEDIUM**: fail → review, review → pass
   - **Impact**: Errors surface for human review
   - **Priority**: P2 - Monitor
   - **Acceptable**: < 20% error rate

4. **LOW**: review → fail, pass → review
   - **Impact**: Conservative errors (safe)
   - **Priority**: P3 - Low priority
   - **Acceptable**: < 25% error rate

---

#### Step 3: Calculate Error Rates

```bash
# Extract error counts from confusion matrix
fail_to_pass=$(jq '.confusion_matrix.fail.pass' results.json)
fail_total=$(jq '.per_class_metrics.fail.support' results.json)
error_rate=$(echo "scale=2; $fail_to_pass / $fail_total * 100" | bc)

echo "Fail→Pass error rate: ${error_rate}%"
```

**Target Error Rates**:
- fail→pass: < 4%
- pass→fail: < 6%
- fail→review: < 20%
- review→pass: < 16%
- review→fail: < 25%
- pass→review: < 10%

---

#### Step 4: Extract Error Cases

```bash
# Extract all false positives (fail→pass)
jq 'select(.human_annotation == "fail" and .judge_verdict == "pass")' \
   validation_results.jsonl > false_positives.jsonl

# Analyze common patterns
jq -r '[.interaction.user_query, .interaction.answer, .confidence] | @csv' \
   false_positives.jsonl | sort -t, -k3 -rn
```

**Look for patterns**:
- Are errors concentrated in specific topics?
- Do errors have common length/tone characteristics?
- Are confidence scores clustered near verdict threshold?

---

## Troubleshooting Flowchart

```
                    ┌─────────────────────┐
                    │ Validation Result   │
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │ Kendall's τ ≥ 0.3?  │
                    └──────────┬──────────┘
                               │
              ┌────────────────┴────────────────┐
              │ YES                             │ NO
              ▼                                 ▼
    ┌──────────────────┐           ┌──────────────────────┐
    │ ✅ PASSED        │           │ ❌ FAILED            │
    └────────┬─────────┘           └─────────┬────────────┘
             │                               │
    ┌────────▼─────────┐         ┌──────────▼────────────┐
    │ Check Cohen's κ  │         │ Check Confusion Matrix│
    └────────┬─────────┘         └─────────┬─────────────┘
             │                             │
    ┌────────▼─────────┐         ┌────────▼──────────────┐
    │ κ ≥ 0.6?         │         │ High fail→pass?       │
    └────────┬─────────┘         │  (>10% of fails)      │
             │                   └─────────┬─────────────┘
      ┌──────┴──────┐                     │ YES
      │ YES    NO   │                     ▼
      ▼      ▼      │           ┌──────────────────────┐
   Deploy  Monitor  │           │ Style-over-substance │
                    │           │ • Reduce coherence   │
                    │           │ • Increase correct   │
                    │           └──────────────────────┘
                    │
                    │           ┌──────────────────────┐
                    │           │ High fail→review?    │
                    │           │  (>15% of fails)     │
                    │           └─────────┬────────────┘
                    │                     │ YES
                    │                     ▼
                    │           ┌──────────────────────┐
                    │           │ Weak correctness     │
                    │           │ • Strengthen correct │
                    │           │ • Update prompts     │
                    │           └──────────────────────┘
                    │
                    │           ┌──────────────────────┐
                    │           │ High review→fail?    │
                    │           │  (>20% of reviews)   │
                    │           └─────────┬────────────┘
                    │                     │ YES
                    │                     ▼
                    │           ┌──────────────────────┐
                    │           │ Penalizes brevity    │
                    │           │ • Update completeness│
                    │           │ • Lower threshold    │
                    └───────────┴──────────────────────┘
```

---

## Quick Reference Checklist

### Pre-Deployment Validation

```
□ Kendall's τ ≥ 0.3? (PRIMARY - pass/fail decision)
□ Cohen's Kappa ≥ 0.6? (SECONDARY - industry credibility)
□ Fail recall ≥ 75%? (CRITICAL - catches bad answers)
□ Pass recall ≥ 90%? (Important - doesn't reject good answers)
□ Review F1 ≥ 0.65? (Acceptable - handles boundaries)
□ Zero fail→pass errors? (CRITICAL - no false promotions)
□ Fail→pass rate < 4%? (If not zero, must be minimal)
□ Pass→fail rate < 6%? (Don't reject too many good answers)
```

### Post-Deployment Monitoring

```
□ Monitor first 100 production evaluations
□ Spot-check random samples daily (first week)
□ Track metric drift over time
□ Re-validate after judge configuration changes
□ Re-validate quarterly with fresh human annotations
```

---

## Case Studies

### Successful Validation Example

See detailed analysis: [resources/validation_success_dataset_interpretation.md](../../resources/validation_success_dataset_interpretation.md)

**Key Metrics**:
- Kendall's τ: 0.632
- Cohen's Kappa: 0.910
- Overall Accuracy: 94%

**Takeaways**:
- High Kappa despite moderate τ (expected for categorical vs continuous)
- Near-perfect pass detection (100% recall)
- Strong fail detection (98% recall)
- Good review handling (84% recall)

---

### Failed Validation Example

See detailed analysis: [resources/validation_failed_dataset_interpretation.md](../../resources/validation_failed_dataset_interpretation.md)

**Key Metrics**:
- Kendall's τ: 0.22 (FAILED)
- Cohen's Kappa: 0.35
- Overall Accuracy: 57%

**Takeaways**:
- Style-over-substance bias (30% fail→pass)
- Weak correctness enforcement (30% fail→review)
- Penalizes brevity (40% review→fail)

**Fixes Applied**:
- Rebalanced judge weights (correctness: 0.15 → 0.24)
- Updated judge prompts (penalize empty answers)
- Adjusted verdict thresholds

---

## Metric Formulas Reference

### Kendall's Tau (τ)

```
τ = (C - D) / sqrt((C + D + T_x) × (C + D + T_y))

Where:
  C = Concordant pairs (both raters agree on ranking)
  D = Discordant pairs (raters disagree on ranking)
  T_x = Ties in first rater's rankings
  T_y = Ties in second rater's rankings
```

**Range**: -1 to +1
- +1 = Perfect agreement
- 0 = Random agreement
- -1 = Perfect disagreement

---

### Cohen's Kappa (κ)

```
κ = (p_o - p_e) / (1 - p_e)

Where:
  p_o = Observed agreement (proportion of exact matches)
  p_e = Expected agreement by chance
```

**Range**: -1 to +1
- +1 = Perfect agreement
- 0 = Agreement by chance
- <0 = Worse than chance

---

### Confusion Matrix Metrics

```
Precision = TP / (TP + FP)   # Of predicted class, how many correct?
Recall    = TP / (TP + FN)   # Of actual class, how many caught?
F1        = 2 × (P × R) / (P + R)  # Harmonic mean

Where:
  TP = True Positives
  FP = False Positives
  FN = False Negatives
```

---

## Additional Resources

- **Implementation**: `internal/metrics/` package
- **Configuration**: `configs/judges.yaml` for judge weights and prompts
- **Testing**: `resources/validation_test_dataset.jsonl` (passing example)
- **Testing**: `resources/validation_failed_dataset.jsonl` (failing example)
- **CLI Tool**: `go run cmd/batch/main.go validate -input dataset.jsonl`

---

## Version History

- **v1.0** (2026-03-16): Initial release with Kendall's τ, Cohen's Kappa, and Confusion Matrix
- Phase 5.2 completion: Metrics comparison analysis and interpretation framework
