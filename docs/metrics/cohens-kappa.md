# Cohen's Kappa: Measuring Agreement Beyond Chance

## The Core Problem

When validating an LLM judge, you need to answer: **How much does the judge agree with humans beyond what we'd expect by random chance?**

**Why this matters:**
- Simple accuracy can be misleading (85% accuracy with 85% "pass" rate = always predicting "pass")
- Cohen's Kappa accounts for chance agreement
- Industry standard for inter-rater reliability in research and ML papers
- Enables comparison with other systems and benchmarks

---

## What is Cohen's Kappa?

Cohen's Kappa (κ) measures categorical agreement between two raters, correcting for agreement that could occur by chance alone.

### Formula

```
κ = (p_o - p_e) / (1 - p_e)

Where:
  p_o = Observed agreement (proportion of exact matches)
  p_e = Expected agreement by chance
```

**Range**: -1 to +1
- **κ = 1**: Perfect agreement
- **κ = 0**: Agreement equals chance
- **κ < 0**: Agreement worse than chance (systematic disagreement)

---

## Simple Example: Validating Themis Judge

### Scenario: 10 Customer Support Responses

You have 10 responses evaluated by both humans and Themis judge:

| Response | Human Verdict | Judge Verdict | Match? |
|----------|---------------|---------------|--------|
| R1       | pass          | pass          | ✓      |
| R2       | pass          | pass          | ✓      |
| R3       | pass          | review        | ✗      |
| R4       | review        | review        | ✓      |
| R5       | review        | pass          | ✗      |
| R6       | fail          | fail          | ✓      |
| R7       | fail          | review        | ✗      |
| R8       | pass          | pass          | ✓      |
| R9       | review        | review        | ✓      |
| R10      | fail          | fail          | ✓      |

**Verdicts distribution:**
- Human: 4 pass, 3 review, 3 fail
- Judge: 5 pass, 3 review, 2 fail

---

### Step 1: Calculate Observed Agreement (p_o)

Count exact matches:
- R1, R2, R4, R6, R8, R9, R10 = **7 matches**

```
p_o = 7 / 10 = 0.7 (70% accuracy)
```

---

### Step 2: Calculate Expected Agreement (p_e)

**What if both raters were randomly guessing?** How often would they agree by pure chance?

**For each category, calculate chance agreement:**

#### Pass class:
- Human predicted "pass" 4 times (40% of time)
- Judge predicted "pass" 5 times (50% of time)
- Chance both say "pass" = 0.4 × 0.5 = 0.20

#### Review class:
- Human predicted "review" 3 times (30%)
- Judge predicted "review" 3 times (30%)
- Chance both say "review" = 0.3 × 0.3 = 0.09

#### Fail class:
- Human predicted "fail" 3 times (30%)
- Judge predicted "fail" 2 times (20%)
- Chance both say "fail" = 0.3 × 0.2 = 0.06

**Total expected agreement:**
```
p_e = 0.20 + 0.09 + 0.06 = 0.35 (35%)
```

**Interpretation**: Even if both were randomly guessing (but with same frequency distributions), they'd agree 35% of the time by pure chance.

---

### Step 3: Calculate Cohen's Kappa

```
κ = (p_o - p_e) / (1 - p_e)
κ = (0.70 - 0.35) / (1 - 0.35)
κ = 0.35 / 0.65
κ = 0.538
```

**Result**: κ = 0.54 → **"Moderate agreement"**

---

## Interpretation Scale (Landis & Koch)

| Kappa Range | Interpretation | Meaning |
|-------------|----------------|---------|
| **κ > 0.80** | Almost perfect | Near-human agreement |
| **κ = 0.60-0.80** | Substantial | Strong agreement - industry standard |
| **κ = 0.40-0.60** | Moderate | Acceptable but room for improvement |
| **κ = 0.20-0.40** | Fair | Below standard - needs work |
| **κ < 0.20** | Slight | Poor agreement |
| **κ < 0** | Worse than chance | Systematic disagreement |

**Our example**: κ = 0.54 → Moderate agreement

---

## Why Kappa is Better Than Simple Accuracy

### Example 1: The "Always Pass" Judge (Class Imbalance)

**Dataset**: 85 pass, 10 review, 5 fail (85% pass rate)

| Response | Human | Judge | Match? |
|----------|-------|-------|--------|
| R1-85    | pass  | pass  | ✓ (85) |
| R86-95   | review | pass | ✗ (10) |
| R96-100  | fail  | pass  | ✗ (5)  |

**Simple Accuracy**: 85/100 = 85%
**Observed agreement (p_o)**: 0.85

**Expected agreement (p_e)**:
- Human "pass" = 85%, Judge "pass" = 100%
- p_e = (0.85 × 1.0) + (0.10 × 0) + (0.05 × 0) = 0.85

**Cohen's Kappa**:
```
κ = (0.85 - 0.85) / (1 - 0.85)
κ = 0 / 0.15
κ = 0.0
```

**Interpretation:**
- **Accuracy says**: 85% - looks good!
- **Kappa says**: 0.0 - agreement is purely by chance, judge is useless

**Why?** The judge always predicts "pass" and benefits from class imbalance. Kappa reveals this exploit.

---

### Example 2: Balanced but Random Judge

**Dataset**: 33 pass, 34 review, 33 fail (perfectly balanced)

**Judge predicts randomly** with same distribution:

| Verdict | Human Count | Judge Count | Matches (by chance) |
|---------|-------------|-------------|---------------------|
| pass    | 33          | 33          | ~11                 |
| review  | 34          | 34          | ~11                 |
| fail    | 33          | 33          | ~11                 |

**Total matches by chance**: ~33 out of 100

**Simple Accuracy**: 33% (terrible)
**Cohen's Kappa**: ~0.0 (confirms it's pure chance)

**Interpretation:** Both metrics agree - judge is useless.

---

## Detailed Calculation from Confusion Matrix

### Confusion Matrix Example

```
                    Predicted
                fail  review  pass  | Total
Actual  fail     20      5      2   |  27
        review    3     15      8   |  26
        pass      1      6     40   |  47
        --------------------------------
        Total    24     26     50   | 100
```

### Step 1: Calculate p_o (Observed Agreement)

Diagonal cells = exact matches:
```
p_o = (20 + 15 + 40) / 100 = 75 / 100 = 0.75
```

### Step 2: Calculate p_e (Expected Agreement)

**For each class:**

#### Fail:
- Human "fail" proportion: 27/100 = 0.27
- Judge "fail" proportion: 24/100 = 0.24
- Chance agreement: 0.27 × 0.24 = 0.0648

#### Review:
- Human "review" proportion: 26/100 = 0.26
- Judge "review" proportion: 26/100 = 0.26
- Chance agreement: 0.26 × 0.26 = 0.0676

#### Pass:
- Human "pass" proportion: 47/100 = 0.47
- Judge "pass" proportion: 50/100 = 0.50
- Chance agreement: 0.47 × 0.50 = 0.2350

**Total expected agreement:**
```
p_e = 0.0648 + 0.0676 + 0.2350 = 0.3674
```

### Step 3: Calculate Kappa

```
κ = (0.75 - 0.3674) / (1 - 0.3674)
κ = 0.3826 / 0.6326
κ = 0.605
```

**Result**: κ = 0.605 → **"Substantial agreement"**

---

## Kappa vs Accuracy: Real-World Example

### Scenario: Medical Diagnosis Validation

**Context**: Testing AI diagnostic system against doctor diagnoses on 1000 cases

#### System A (Naive)
```
Accuracy: 90%
Kappa: 0.10

Why? Disease prevalence is 10%. System predicts "healthy" for everyone.
- Matches 90% (all healthy cases)
- But fails all sick patients
- Agreement is mostly by chance (exploiting class imbalance)
```

#### System B (Trained)
```
Accuracy: 85%
Kappa: 0.72

Why? System makes real diagnoses, catches 80% of sick patients.
- Lower accuracy due to false positives
- But Kappa is high - agreement beyond chance
- System adds value over baseline
```

**Conclusion**: System B is better despite lower accuracy!

---

## Kappa Limitations

### 1. Paradox: Low Kappa Despite High Agreement

**Scenario**: Judge is very accurate but conservative

```
                    Predicted
                fail  review  pass  | Total
Actual  fail     48      2      0   |  50
        review    0      0      0   |   0
        pass      0      2     48   |  50
        --------------------------------
        Total    48      4     48   | 100
```

**Observed agreement**: p_o = (48 + 0 + 48) / 100 = 96%
**Expected agreement**: p_e = (0.5 × 0.48) + (0 × 0.04) + (0.5 × 0.48) = 0.48

```
κ = (0.96 - 0.48) / (1 - 0.48) = 0.48 / 0.52 = 0.92
```

**Result**: κ = 0.92 (almost perfect) ✓

**But if "review" class exists but never predicted:**

```
                    Predicted
                fail  review  pass  | Total
Actual  fail     48      0      2   |  50
        review    2      0      2   |   4
        pass      0      0     46   |  46
        --------------------------------
        Total    50      0     50   | 100
```

**Observed agreement**: p_o = (48 + 0 + 46) / 100 = 94%
**Expected agreement**: p_e = (0.5 × 0.5) + (0.04 × 0) + (0.46 × 0.5) = 0.48

```
κ = (0.94 - 0.48) / (1 - 0.48) = 0.46 / 0.52 = 0.88
```

**Issue**: High accuracy (94%) and high Kappa (0.88), but judge never uses "review" category.

**Lesson**: Always inspect confusion matrix alongside Kappa to detect missing categories.

---

### 2. Sensitivity to Prevalence

Kappa can vary with class distribution even with same error rate:

**Dataset A** (balanced):
- 33% fail, 33% review, 34% pass
- 10% error rate across all classes
- κ ≈ 0.85

**Dataset B** (imbalanced):
- 5% fail, 10% review, 85% pass
- 10% error rate across all classes
- κ ≈ 0.60

**Why?** Expected agreement (p_e) is higher with imbalanced classes, reducing Kappa denominator.

**Lesson**: Compare Kappa only on similar class distributions.

---

## Cohen's Kappa in Themis Validation

### When Kappa is Used

**Role**: Secondary metric for reporting industry-standard agreement

**Decision hierarchy:**
1. **Kendall's τ ≥ 0.3?** → PRIMARY pass/fail decision
2. **Cohen's Kappa** → Report for credibility (papers, stakeholders)
3. **Confusion Matrix** → Diagnostic tool for debugging

### Typical Validation Output

```bash
themis validate --input human_annotated.jsonl --threshold 0.3

=== Validation Report ===
Total Records: 100
Status: ✓ PASSED

Correlation Metrics (PRIMARY):
  Kendall's τ: 0.452 (Moderate to strong agreement) → PASSED ✅

Agreement Metrics (REPORT):
  Cohen's Kappa: 0.605 (Substantial agreement) ✓

Confusion Matrix (DEBUG):
  [confusion matrix display]

Interpretation:
  ✓ Judge validated (τ ≥ 0.3)
  ✓ Substantial agreement with humans (κ = 0.6)
  ✓ Safe to deploy to production
```

---

## Kappa vs Kendall's τ: When to Use Which

| Metric | Measures | Use Case | Threshold |
|--------|----------|----------|-----------|
| **Kendall's τ** | Rank correlation | Primary validation (pass/fail) | τ ≥ 0.3 |
| **Cohen's Kappa** | Categorical agreement | Industry reporting | κ ≥ 0.6 |
| **Confusion Matrix** | Per-class errors | Debugging and improvement | N/A |

### Example: High Kappa, Low Tau

```
κ = 0.75 (substantial)
τ = 0.28 (weak)

Confusion Matrix:
                    Predicted
                fail  review  pass  | Total
Actual  fail     25      2      0   |  27
        review    0     24      2   |  26
        pass      0      0     47   |  47
```

**What's happening:**
- Categories are correct (high Kappa)
- But confidence scores don't correlate (low τ)
- Verdict logic OK, but scoring is broken

**Action**: Fix aggregation method or stage weights, don't change judges.

---

### Example: High Tau, Low Kappa

```
τ = 0.65 (moderate to strong)
κ = 0.42 (moderate)

Confusion Matrix:
                    Predicted
                fail  review  pass  | Total
Actual  fail     20      5      2   |  27
        review    8     10      8   |  26
        pass      2      5     40   |  47
```

**What's happening:**
- Scores correlate well (high τ)
- But many category mismatches (moderate Kappa)
- Especially review class scattered

**Action**: Adjust verdict thresholds (VERDICT_PASS_THRESHOLD, VERDICT_REVIEW_THRESHOLD).

---

## Practical Recommendations

### 1. Always Report Both Metrics

```json
{
  "kendalls_tau": 0.45,
  "tau_interpretation": "Moderate agreement",
  "tau_passed": true,

  "cohens_kappa": 0.62,
  "kappa_interpretation": "Substantial agreement"
}
```

### 2. Use Kappa for Stakeholder Communication

**For technical audience**: "Kendall's τ = 0.45, judge is validated"
**For non-technical audience**: "Cohen's Kappa = 0.62, substantial agreement with human experts"

### 3. Don't Over-Rely on Kappa Alone

**Good validation:**
- τ ≥ 0.3 ✓
- κ ≥ 0.6 ✓
- Confusion matrix shows few critical errors (fail→pass) ✓

**Misleading validation:**
- τ < 0.3 ✗
- κ = 0.75 ✓ (looks good!)
- Confusion matrix reveals all predictions are "pass" (exploiting imbalance)

### 4. Inspect Confusion Matrix

Kappa alone doesn't show:
- Which classes are problematic
- What types of errors occur
- Whether judge exploits class imbalance

**Always check**: diagonal strength, error distribution, class balance

---

## Key Takeaways

1. **Kappa accounts for chance** - unlike simple accuracy
2. **Range: -1 to +1** - with 0 = random agreement
3. **Industry standard** - use for papers and reporting
4. **Secondary to Kendall's τ** - in Themis validation workflow
5. **Not sufficient alone** - can be misleading without confusion matrix
6. **Sensitive to class imbalance** - expected agreement varies with prevalence
7. **Interpretation scale** - κ > 0.6 is substantial, > 0.8 is almost perfect
8. **Use with τ and confusion matrix** - three metrics give complete picture
9. **Good for communication** - easier to explain than τ to stakeholders
10. **Watch for paradoxes** - low Kappa despite high accuracy can happen

---

## Formula Summary

```
Cohen's Kappa:
  κ = (p_o - p_e) / (1 - p_e)

Where:
  p_o = Observed agreement = diagonal_sum / total_samples

  p_e = Expected agreement = Σ (P(human=i) × P(judge=i))
      = Σ (row_total_i / N) × (col_total_i / N)

Interpretation (Landis & Koch):
  κ > 0.80 → Almost perfect
  κ = 0.60-0.80 → Substantial
  κ = 0.40-0.60 → Moderate
  κ = 0.20-0.40 → Fair
  κ < 0.20 → Slight
  κ < 0 → Worse than chance
```

---

## Next Steps

1. **Integrate Kappa calculation** into validation pipeline
2. **Report alongside Kendall's τ** in CLI output
3. **Document in validation reports** for industry credibility
4. **Use for benchmarking** against other evaluation systems
5. **Include in research papers** when publishing judge methodology
6. **Educate stakeholders** on interpretation scale
7. **Monitor over time** to detect judge degradation
8. **Compare across judge versions** to measure improvements

Cohen's Kappa provides the industry-standard metric for demonstrating your LLM judge reliability to external audiences, while Kendall's τ remains your primary internal validation gate.
