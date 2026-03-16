# Metrics Package

Validation metrics for evaluating LLM judge accuracy against human annotations.

**3 Core Metrics:**
1. **Kendall's τ (PRIMARY)** - Pass/fail decision (τ ≥ 0.3 = deploy)
2. **Confusion Matrix (DEBUG)** - Shows WHERE judge fails (actionable insights)
3. **Cohen's Kappa (REPORT)** - Industry standard for stakeholder communication

---

## Kendall's Tau (τ)

### Why It's Required

**Kendall's τ answers:** "Do the judge's scores correlate with human rankings?"

**Primary purpose:** Pass/fail decision for judge deployment (τ ≥ 0.3 = deploy)

Kendall's tau measures rank correlation between human annotations and LLM verdicts:
- τ = 1.0: Perfect agreement (all pairs ranked same way)
- τ = 0.0: No correlation (random guessing)
- τ = -1.0: Perfect disagreement (all pairs ranked opposite)

---

### Formula

```
τ = (concordant - discordant) / total_pairs

where:
  concordant = pairs where both humans and judge rank in same direction
  discordant = pairs where humans and judge rank in opposite directions
  total_pairs = n × (n - 1) / 2
```

**Ranking:**
- fail = 0 (worst)
- review = 1 (middle)
- pass = 2 (best)

**Example:**
```
Human:  [fail, review, pass]  →  [0, 1, 2]
Judge:  [fail, review, pass]  →  [0, 1, 2]

All 3 pairs concordant:
  (0,1): human 0<1, judge 0<1 ✓
  (0,2): human 0<2, judge 0<2 ✓
  (1,2): human 1<2, judge 1<2 ✓

τ = (3 - 0) / 3 = 1.0 (perfect)
```

---

### Interpretation Scale

| τ Value | Interpretation | Action |
|---------|----------------|--------|
| ≥ 0.7 | Strong agreement | Deploy confidently |
| 0.5 - 0.69 | Moderate to strong agreement | Deploy with monitoring |
| 0.3 - 0.49 | Moderate agreement | Deploy (minimum threshold) |
| 0.1 - 0.29 | Weak agreement | Reject, improve judge |
| < 0.1 | Very weak or no agreement | Reject, major issues |

**Decision threshold:** τ ≥ 0.3 is minimum for production deployment

---

### When to Use Tau

**Use Kendall's τ as PRIMARY metric:**
- Pass/fail decision for judge deployment
- Validates that judge understands quality ordering
- Works with ordinal labels (fail < review < pass)
- Robust to ties and small sample sizes

**Limitations:**
- Doesn't show WHERE judge fails (use confusion matrix)
- Doesn't account for class imbalance (use Cohen's Kappa for reporting)
- Only measures rank correlation, not exact agreement

**Decision flow:**
1. Compute τ on validation set
2. If τ ≥ 0.3 → PASSED, judge is deployable
3. If τ < 0.3 → FAILED, investigate with confusion matrix

---

## Confusion Matrix

### Why It's Required

**Kendall's τ answers:** "Do scores correlate overall?"
**Confusion matrix answers:** "WHERE exactly does the judge fail?"

A confusion matrix shows the breakdown of every prediction vs. actual label:

```
                Predicted
            fail  review  pass
Actual fail   20      5      2    ← Judge missed 7 failures
       review  3     15      8    ← Borderline cases hard
       pass    1      6     40    ← Strong pass detection
```

**Use case:** When τ = 0.42 (moderate), the confusion matrix reveals:
- 2 critical errors (fail→pass): Bad responses slip through
- 5 minor errors (fail→review): Conservative but safe
- Need to improve failure detection, but judge is deployable

---

## Why Count False Positives (FP) and False Negatives (FN)?

**False Negatives (FN):** Actual failures the judge missed
- **Impact:** Bad responses slip into production
- **Example:** Judge says "pass" but human says "fail" → Safety risk

**False Positives (FP):** Judge incorrectly flags good responses
- **Impact:** Users get frustrated by false alarms
- **Example:** Judge says "fail" but human says "pass" → Bad UX

**Trade-off:** You can't optimize both simultaneously. Choose based on risk tolerance:
- Safety-critical systems: Minimize FN (catch all failures, accept some false alarms)
- User-facing systems: Balance FN and FP (avoid annoying users)

---

## Precision, Recall, F1 Score

### Precision
```
Precision = TP / (TP + FP)
```

**Meaning:** "When judge says X, how often is it correct?"

**Example:**
- Judge predicts "fail" 10 times
- 8 are actually failures, 2 are false alarms
- Precision = 8/10 = 0.8 (80% accurate when predicting fail)

**High precision = Few false alarms**

---

### Recall
```
Recall = TP / (TP + FN)
```

**Meaning:** "Of all actual X, how many did judge detect?"

**Example:**
- 12 actual failures in dataset
- Judge catches 8, misses 4
- Recall = 8/12 = 0.67 (caught 67% of failures)

**High recall = Few misses**

---

### F1 Score
```
F1 = 2 × (Precision × Recall) / (Precision + Recall)
```

**Meaning:** Harmonic mean of precision and recall (balanced metric)

**Why it matters:**
- Precision alone is misleading (judge that never predicts "fail" has 0 FP but useless)
- Recall alone is misleading (judge that always predicts "fail" catches everything but unusable)
- F1 balances both: good F1 means judge is both accurate AND complete

**Example:**
- Precision = 0.8, Recall = 0.67
- F1 = 0.73 (reasonable balance)

---

## When to Use Each Metric

| Metric | What to Check | Action If Low |
|--------|---------------|---------------|
| **Precision** | False alarm rate | Too many good responses flagged → Tighten fail threshold |
| **Recall** | Miss rate | Too many failures slip through → Review false negatives, improve prompt |
| **F1** | Overall balance | Judge is either too strict or too lenient → Re-calibrate |

---

## Example: Debugging with Confusion Matrix

**Problem:** Judge has low recall for "fail" class (0.60)

**Step 1:** Check confusion matrix
```
Actual fail: 10 (detected) + 5 (→review) + 2 (→pass) = 17 total
Missing: 7 failures (41%)
```

**Step 2:** Extract false negatives
```bash
jq 'select(.human_annotation == "fail" and .judge_verdict != "fail")' results.jsonl
```

**Step 3:** Find patterns
- Are they short responses? → Adjust length checker
- Are they polite but wrong? → Emphasize correctness in prompt
- Are they edge cases? → Add examples to judge config

**Step 4:** Re-validate after fix
- Recall improved from 0.60 → 0.75 ✅

---

## Binary Confusion Matrix

Sometimes 3 classes (fail/review/pass) are too granular. Collapse to 2:

**Merge fail+review → "negative" (needs work)**
**Keep pass → "positive" (good quality)**

```
                Predicted
            negative  positive
Actual negative   43        10     ← 81% caught
       positive    7        40     ← 85% correct
```

**When to use:** Simpler metrics, easier stakeholder communication

---

## Cohen's Kappa

### Why It's Required

**Problem with accuracy:** A judge that always predicts "pass" gets 90% accuracy if 90% of data is "pass" (useless!)

**Cohen's Kappa corrects for chance agreement:**
- Accuracy = "How often do we agree?"
- Kappa = "How often do we agree, **minus expected agreement by random guessing**?"

**Use case:** Validating judges for research papers, stakeholder reporting

---

### Formula

```
κ = (observed_agreement - expected_agreement) / (1 - expected_agreement)

Where:
  observed_agreement = (diagonal sum) / total
  expected_agreement = Σ(p_actual_i × p_predicted_i)
```

**Intuition:**
- If judge agrees with humans more than random guessing → κ > 0
- If judge agrees exactly as much as random guessing → κ = 0
- If judge disagrees more than random guessing → κ < 0

---

### Interpretation Scale

| κ Value | Interpretation | Meaning |
|---------|----------------|---------|
| < 0.00 | Poor | Worse than random |
| 0.00-0.20 | Slight | Barely better than random |
| 0.21-0.40 | Fair | Some agreement, not great |
| 0.41-0.60 | Moderate | Reasonable agreement |
| 0.61-0.80 | Substantial | Strong agreement |
| 0.81-1.00 | Almost perfect | Excellent agreement |

**Industry standard:** κ ≥ 0.40 is acceptable, κ ≥ 0.60 is good

---

### When to Use Kappa

**Use Kappa when:**
- Data has class imbalance (e.g., 90% pass, 5% review, 5% fail)
- Writing research papers (required for inter-rater agreement)
- Communicating with stakeholders who know ML standards
- Comparing judges across different datasets

**Don't rely on Kappa alone:**
- Always check confusion matrix (Kappa won't tell you where judge fails)
- Always check Kendall's τ first (that's the pass/fail decision)
- Kappa is **report-only** metric, not decision metric

---

### Example: Why Kappa Matters

**Scenario 1: Imbalanced dataset (90% pass)**

Judge always predicts "pass":
- Accuracy = 90% (looks good!)
- Cohen's Kappa = 0.0 (no better than random)
- **Verdict:** Judge is useless

**Scenario 2: Balanced errors**

Judge agrees 75% of time on balanced dataset:
- Accuracy = 75%
- Cohen's Kappa = 0.62 (substantial agreement)
- **Verdict:** Judge is reliable

**Key insight:** Kappa tells you if judge is actually evaluating or just exploiting class distribution

---

## Testing

```bash
go test ./internal/metrics/...        # Run tests
go test -cover ./internal/metrics/... # With coverage
```

Current coverage: **95.1%** (24 tests passing)
