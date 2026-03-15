# Validation Metrics Expansion - Specification

**Status:** Planning
**Date:** 2026-03-14

---

## 1. Purpose

### Problem

Currently, validating LLM judges in Themis relies on:
1. **Only Kendall's τ** (rank correlation) - tells if scores correlate, not where judge fails
2. **No categorical agreement metrics** - Cohen's Kappa is industry standard for classification
3. **No diagnostic tools** - can't identify specific error patterns (false positives vs negatives)
4. **Limited stakeholder communication** - τ is hard to explain to non-technical audiences

This makes judge debugging slow, requires statistical expertise, and provides incomplete validation.

### Solution

Expand validation to include **2 additional industry-standard metrics**:

1. **Confusion Matrix** - Visual diagnostic showing exact error breakdown
2. **Cohen's Kappa (κ)** - Categorical agreement accounting for chance (industry standard)

### Key Insight

**3 metrics, 3 purposes:**

1. **Kendall's τ** → Pass/fail decision (τ ≥ 0.3 = deploy, else reject)
2. **Cohen's Kappa** → Industry credibility (report for papers/stakeholders, don't act on)
3. **Confusion Matrix** → Actionable debugging (shows WHERE judge fails, WHAT to fix)

**Simple decision flow:**
```
Check τ → If passed, look at confusion matrix → Fix specific errors
```

### Inspiration

Industry-standard ML evaluation includes both:
- **scikit-learn** (Python): Provides confusion matrix + classification report by default
- **caret** (R): Includes κ, confusion matrix, per-class metrics
- **Research papers**: Require κ for inter-rater agreement studies

**Themis should match this standard** for LLM judge validation.

---

## 2. Architecture

### Current Structure

```
internal/
└── batch/
    ├── processor.go
    ├── reader.go
    ├── writer.go
    └── validator.go       # Only computes Kendall's τ

cmd/batch/main.go          # Validation mode with τ threshold check
```

**Current validation output:**
```json
{
  "passed": true,
  "total_records": 150,
  "kendalls_tau": 0.42,
  "interpretation": "Moderate positive correlation",
  "threshold": 0.3
}
```

### Proposed Structure

```
internal/
├── metrics/               
│   ├── kendalls_tau.go    # MOVED: Extract from batch/validator.go
│   ├── kendalls_tau_test.go
│   ├── confusion_matrix.go
│   ├── confusion_matrix_test.go
│   ├── cohens_kappa.go
│   ├── cohens_kappa_test.go
│   ├── validator.go       # Orchestrates: τ + κ + confusion matrix
│   └── README.md          # Formulas for all 3 metrics
│
└── batch/
    ├── processor.go
    ├── reader.go
    ├── writer.go
    └── validator.go       # ENHANCED: Uses metrics package

cmd/batch/main.go          # Enhanced validation output
```

**Enhanced validation output:**
```json
{
  "passed": true,           // Based on: kendalls_tau ≥ threshold (0.3)
  "total_records": 150,
  "threshold": 0.3,         // Kendall's τ threshold for pass/fail decision

  "correlation_metrics": {
    "kendalls_tau": 0.42,   // PRIMARY: Pass/fail based on this ≥ threshold
    "interpretation": "Moderate positive correlation",
    "passed_threshold": true
  },

  "agreement_metrics": {    // DIAGNOSTIC (report only, don't act on)
    "cohens_kappa": 0.38,
    "interpretation": "Fair agreement"
  },

  "confusion_matrix": {     // DIAGNOSTIC: Shows WHERE judge fails
    "fail": {"fail": 20, "review": 5, "pass": 2},
    "review": {"fail": 3, "review": 15, "pass": 8},
    "pass": {"fail": 1, "review": 6, "pass": 40}
  },

  "per_class_metrics": {    // DIAGNOSTIC: Per-class precision/recall/F1
    "fail": {"precision": 0.833, "recall": 0.741, "f1": 0.785},
    "review": {"precision": 0.577, "recall": 0.577, "f1": 0.577},
    "pass": {"precision": 0.800, "recall": 0.851, "f1": 0.825}
  }
}
```

**Key: 3-Metric Decision Framework**
```
1. Kendall's τ (PRIMARY - Pass/Fail):
   - τ ≥ 0.3 → PASSED ✅ Judge is reliable, deploy it
   - τ < 0.3 → FAILED ❌ Judge is unreliable, reject or fix

2. Cohen's Kappa (REPORT - Industry Standard):
   - Just report the value (e.g., κ = 0.38 = "Fair agreement")
   - Used for papers, stakeholder communication
   - Does NOT affect pass/fail decision

3. Confusion Matrix (DEBUG - Actionable):
   - Shows WHERE judge fails (which error types)
   - Provides WHAT to fix (specific false positives/negatives)
   - Used to improve judge after it passes τ threshold

Example Decision Flow:
  Step 1: τ = 0.42 → PASSED (≥ 0.3) ✅
  Step 2: Report κ = 0.38 ("Fair agreement")
  Step 3: Check confusion matrix → 7 false negatives (5 fail→review, 2 fail→pass)
  Step 4: Investigate those 7 cases to improve recall
```

### Changes

1. **Create:** `internal/metrics/` package with 3 core metrics:
   - `kendalls_tau.go` (moved from batch/validator.go)
   - `confusion_matrix.go` (new)
   - `cohens_kappa.go` (new)
   - `validator.go` (orchestrator)
2. **Enhance:** `internal/batch/validator.go` to use new metrics package
3. **Maintain:** Backwards compatibility (existing τ-only output preserved)
4. **Add:** CLI pretty-printing for terminal output
5. **Document:** Simple 3-metric decision framework

---

## 3. Metrics Overview

### Why These Metrics?

| Metric | What It Measures | Role | Action |
|--------|------------------|------|--------|
| **Kendall's τ** | Rank correlation | **PRIMARY** | Pass/fail decision (τ ≥ 0.3) |
| **Cohen's Kappa** | Categorical agreement (accounts for chance) | **REPORT** | Industry standard (papers/stakeholders) |
| **Confusion Matrix** | Per-class error breakdown | **DEBUG** | Shows WHERE judge fails + WHAT to fix |

### Example: What Each Metric Reveals

**Scenario:** Judge evaluation results on 100 samples

```
Confusion Matrix:
                Predicted
            fail  review  pass
Actual fail   20      5      2   ← 7 failures missed
       review  3     15      8   ← Borderline cases hard
       pass    1      6     40   ← Strong pass detection
```

**Insights per metric:**

1. **Kendall's τ = 0.45** → **PRIMARY**: "Moderate correlation" → PASSED (≥ 0.3) ✅
2. **Cohen's Kappa = 0.38** → **REPORT**: "Fair agreement" (for stakeholders)
3. **Confusion Matrix** → **DEBUG**: "Judge misses 26% of failures (7/27)"
   - 5 false negatives: fail → review
   - 2 false negatives: fail → pass (critical!)

**Decision:** Judge PASSED (τ ≥ 0.3)

**Action:** Investigate 7 false negative cases to improve recall

---

## 4. Validation Workflow

### Before (Current)

```bash
# Run validation
go run cmd/batch/main.go validate -input human_annotated.jsonl -threshold 0.3

# Output: Only Kendall's τ
{
  "kendalls_tau": 0.42,
  "passed": true
}

# Questions left unanswered:
# - Where does judge fail?
# - What types of errors?
# - How to improve?
```

### After (Enhanced)

```bash
# Run validation (same command)
go run cmd/batch/main.go validate -input human_annotated.jsonl -threshold 0.3

# Terminal output (pretty-printed):
=== Validation Report ===
Total Records: 150
Status: ✓ PASSED

Correlation Metrics (PRIMARY - Pass/Fail):
  Kendall's τ: 0.420 (Moderate positive correlation) → PASSED ✅

Agreement Metrics (REPORT - Industry Standard):
  Cohen's Kappa: 0.382 (Fair agreement)

Confusion Matrix (DEBUG - Actionable Insights):
                Predicted
            fail  review  pass  | Total
Actual fail   20      5      2  |  27
       review  3     15      8  |  26
       pass    1      6     40  |  47
       -------------------------------
       Total  24     26     50  | 100

Per-Class Performance:
            Precision  Recall    F1     Support
  fail        0.833    0.741   0.785     27
  review      0.577    0.577   0.577     26
  pass        0.800    0.851   0.825     47

Interpretation:
  ✓ Strong diagonal (75% accuracy)
  ⚠️ Review class weak (58% F1)
  ✓ Only 2 critical errors (fail→pass)
  → Judge is production-ready

# JSON output (programmatic access):
{
  "passed": true,
  "metrics_summary": { ... },
  "confusion_matrix": { ... }
}
```

### Debugging Example

**Problem:** Judge has low recall for "fail" class (74%)

**Step 1: Check confusion matrix**
```
Actual fail: 20 (correct) + 5 (→review) + 2 (→pass) = 27 total
Missing: 7 failures (26%)
```

**Step 2: Extract false negatives**
```bash
jq 'select(.human_annotation == "fail" and .judge_verdict != "fail")' \
   validation_results.jsonl > false_negatives.jsonl
```

**Step 3: Analyze patterns**
- Are they short responses? → Adjust length checker
- Are they polite but wrong? → Emphasize correctness in prompt
- Are they edge cases? → Add examples to judge config

**Step 4: Re-validate**
```bash
# After adjusting judge prompt
go run cmd/batch/main.go validate -input human_annotated.jsonl

# Check improvement:
# - Recall(fail) increased from 74% → 85%
# - Kendall's τ increased from 0.42 → 0.51
```

---

## 5. Implementation Phases

### Phase 1: Metrics Package Infrastructure (Week 1, Day 1-2)

**Goal:** Create reusable metrics package with clean interfaces

**Tasks:**
- [X] Create `internal/metrics/` package structure
- [X] Define core types: `Label`, `ConfusionMatrix`, `ClassMetrics`, `MetricsSummary`
- [ ] Set up test framework with edge cases
- [ ] Document metric formulas in README

**Deliverable:** `internal/metrics/` package skeleton with 100% test coverage

---

### Phase 2: Confusion Matrix Implementation (Week 1, Day 3)

**Goal:** Implement confusion matrix as foundation for other metrics

**Tasks:**
- [X] Implement `Build()` - creates 3x3 matrix from actual/predicted labels
- [X] Implement `ComputeClassMetrics()` - calculates precision/recall/F1
- [ ] Implement `String()` - pretty-prints matrix for CLI
- [ ] Implement `ToBinary()` - collapses to 2x2 (fail+review vs pass)
- [X] Add comprehensive tests

**Deliverable:** Working confusion matrix with per-class metrics

**Key Methods:**
```go
Build(actual, predicted []Label) (*ConfusionMatrix, error)
ComputeClassMetrics() map[Label]ClassMetrics
ToBinary() *ConfusionMatrix
String() string
```

---

### Phase 3: Cohen's Kappa Implementation (Week 1, Day 4)

**Goal:** Implement Cohen's Kappa (industry standard categorical agreement metric)

**Tasks:**
- [ ] Implement `ComputeCohensKappa(cm *ConfusionMatrix)` - computes κ from confusion matrix
- [ ] Implement `InterpretKappa(kappa float64)` - returns human-readable interpretation
- [ ] Add comprehensive tests (perfect agreement, random, imbalanced)

**Formula:**
```
κ = (p_o - p_e) / (1 - p_e)

where:
  p_o = observed agreement (diagonal sum / total)
  p_e = expected agreement by chance = Σ(p_actual_i × p_predicted_i)
```

**Interpretation Scale:**
- κ < 0.00: Poor (worse than chance)
- κ = 0.00-0.20: Slight agreement
- κ = 0.21-0.40: Fair agreement
- κ = 0.41-0.60: Moderate agreement
- κ = 0.61-0.80: Substantial agreement
- κ = 0.81-1.00: Almost perfect agreement

**Key Methods:**
```go
ComputeCohensKappa(cm *ConfusionMatrix) (float64, error)
InterpretKappa(kappa float64) string
```

**Deliverable:** Cohen's Kappa implemented and tested

---

### Phase 4: Integration with Validation Pipeline (Week 1, Day 5 + Week 2, Day 1)

**Goal:** Integrate new metrics into existing validation workflow

**Tasks:**
- [ ] Update `internal/batch/validator.go` to use metrics package
- [ ] Enhance `ValidationReport` struct with comprehensive metrics
- [ ] Add CLI pretty-printing for terminal output
- [ ] Maintain backwards compatibility (legacy τ-only output)
- [ ] Add `--output-format` flag: `legacy`, `full`, `json`

**Deliverable:** Enhanced validation pipeline with full metrics reporting

**Backwards Compatibility:**
```bash
# Legacy format (for existing scripts)
go run cmd/batch/main.go validate -input data.jsonl -output-format legacy
# Output: Only Kendall's τ (existing behavior)

# Full format (default)
go run cmd/batch/main.go validate -input data.jsonl
# Output: All metrics with pretty-printing

# JSON format (programmatic)
go run cmd/batch/main.go validate -input data.jsonl -output-format json
# Output: Machine-parseable JSON
```

---

### Phase 5: Documentation & Testing (Week 2, Day 2-5)

**Goal:** Comprehensive documentation and validation with test dataset

**Task 5.1: Create Test Dataset** (Day 3)
- [ ] Generate 150 synthetic samples (50 fail, 50 review, 50 pass)
- [ ] Include edge cases (ambiguous, boundary, perfect matches)
- [ ] Add to `resources/validation_test_dataset.jsonl`

**Task 5.2: Metric Comparison Analysis** (Day 3-4)
- [ ] Run validation with test dataset
- [ ] Compare metrics across different error patterns
- [ ] Document when each metric provides unique insight

**Task 5.3: Documentation** (Day 4-5)
- [ ] Create `docs/metrics/README.md` - formulas and interpretations
- [ ] Create `docs/metrics/interpretation-guide.md` - case studies
- [ ] Update `docs/testing/batch-tests.md` - new validation examples
- [ ] Create `docs/examples/validation-report-examples.md` - 5 scenarios

**Task 5.4: Unit Testing** (Day 5)
- [ ] Achieve >95% code coverage for `internal/metrics/`
- [ ] Add integration tests for full validation pipeline
- [ ] Performance benchmarks (10k samples in <1s)

**Deliverable:** Complete documentation + validated test dataset

---

## 6. Success Criteria

### Technical Requirements
- ✅ All 3 core metrics in `internal/metrics/` implemented and tested:
  - `kendalls_tau.go` (moved from batch/validator.go)
  - `confusion_matrix.go`
  - `cohens_kappa.go`
  - `validator.go` (orchestrator)
- ✅ >95% test coverage for `internal/metrics/` package
- ✅ Backwards compatibility maintained (existing scripts work unchanged)
- ✅ Performance: Process 1000 samples in <2 seconds
- ✅ Clean integration with existing validation pipeline
- ✅ Simple 3-metric decision framework documented

### User Experience
- ✅ CLI output is readable and actionable
- ✅ JSON output is machine-parseable
- ✅ Confusion matrix reveals error patterns clearly
- ✅ Per-class metrics help debug specific issues
- ✅ Metric interpretation is well-documented

### Documentation
- ✅ Each metric has formula + interpretation guide
- ✅ Examples show when each metric provides unique insight
- ✅ Test dataset (150 samples) demonstrates metric behavior
- ✅ Integration with future `themis validate` command documented

---

## 7. Dependencies

### Go Packages (New)
- **None** - Pure Go implementation (no external dependencies)

### Go Packages (Existing)
- `gonum.org/v1/gonum/stat` - Already used for Kendall's τ

### Test Data
- 150-sample annotated dataset with balanced classes
- Edge case examples (ambiguous labels, boundary cases)

### Documentation Tools
- Existing docs structure (`docs/metrics/`, `docs/examples/`)

---

## 8. Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Metric formulas implemented incorrectly | High | Validate against published papers + Python scikit-learn |
| Confusion matrix visualization unclear | Medium | User testing with CLI output, iterate on format |
| Performance issues with large datasets | Low | Benchmark with 10k samples, optimize if needed |
| Users confused by multiple metrics | Medium | Clear documentation + interpretation guide |
| Breaking changes to existing validation | High | Maintain backwards compatibility, `--output-format legacy` |

---

## 9. Comparison to Alternatives

### Alternative 1: Use Python scikit-learn for Validation
**Pros:** Battle-tested, comprehensive metrics
**Cons:** Requires Python, fragmented toolchain, users need both Go and Python

**Decision:** Implement in Go for unified experience

### Alternative 2: Only Add Confusion Matrix (Skip Kappa)
**Pros:** Simpler implementation
**Cons:** Cohen's Kappa is industry standard for inter-rater agreement

**Decision:** Include κ for research/publication requirements

### Alternative 3: External Metrics Service
**Pros:** Separation of concerns
**Cons:** Extra infrastructure, API calls, complexity

**Decision:** Inline metrics package is simpler

---

## 10. Open Questions

### Q1: Should we report both 3-class and binary metrics by default?
**Proposal:** Yes, include both in JSON output. CLI shows 3-class by default, add `--binary` flag for 2x2 view.

### Q2: What weight scheme for weighted kappa?
**Proposal:** Use linear weights (ordinal distance) by default. Add `--weight-scheme` flag: `linear`, `quadratic`.

### Q3: Should confusion matrix show counts or percentages?
**Proposal:** Show counts by default (easier to understand). Add row percentages in parentheses for normalized view.

### Q4: Minimum sample size for reliable metrics?
**Proposal:** Warn if n < 30 samples. Document that κ requires n > 50 for stability.

### Q5: Should we add per-judge metrics (validate each judge individually)?
**Proposal:** Not in this phase. Add `--judge <name>` flag in future iteration.

---

## 11. Timeline

**Week 1 (Implementation):**
- Day 1-2: Metrics package infrastructure (types, tests, README)
- Day 3: Confusion matrix implementation + per-class metrics
- Day 4: Cohen's Kappa implementation
- Day 5: Integration with validation pipeline (part 1)

**Week 2 (Integration & Documentation):**
- Day 1: Integration with validation pipeline (part 2) + CLI output formatting
- Day 2: Test dataset creation (150 samples)
- Day 3-4: Metric comparison analysis + interpretation guide
- Day 5: Final documentation + examples

**Total:** 2 weeks to production-ready 3-metric validation (simplified from original 5-metric proposal)

---

## 12. Next Steps

1. **Approve this specification** (review and sign off)
2. **Create GitHub issues** for each phase
3. **Begin Phase 1** (Metrics package structure)
4. **Iterate based on test results** (adjust formulas, output format)
5. **Release with v0.x.0** (breaking change to validation output format)

---

## 13. Future Enhancements (Post-Launch)

### Phase 6: Additional Metrics (If User Demand)
- **Weighted Cohen's Kappa** - Partial credit for ordinal errors (fail→review vs fail→pass)
- **Agreement Rate** - Simple accuracy (% exact matches)
- **Spearman's ρ** - Parametric alternative to Kendall's τ
- **Mean Absolute Error (MAE)** - For continuous scores
- **Per-judge validation** - Validate each judge independently

**Note:** Intentionally excluded from initial release to keep simple (3 metrics only)

### Phase 7: Visualization (Optional)
- HTML report generation (confusion matrix heatmap)
- Time-series tracking (τ/κ over multiple validation runs)
- Dashboard integration (show metrics in web UI)

### Phase 8: Statistical Tests (Optional)
- Bootstrap confidence intervals for κ
- Statistical significance testing (McNemar's test)
- Sample size recommendations (power analysis)

---

## Appendix A: Metric Formulas

### Cohen's Kappa
```
κ = (p_o - p_e) / (1 - p_e)

where:
  p_o = observed agreement = (diagonal sum) / total
  p_e = expected agreement = Σ(p_actual_i × p_predicted_i)

Interpretation:
  < 0.00: Poor (worse than chance)
  0.00-0.20: Slight agreement
  0.21-0.40: Fair agreement
  0.41-0.60: Moderate agreement
  0.61-0.80: Substantial agreement
  0.81-1.00: Almost perfect agreement
```

### Per-Class Metrics (Derived from Confusion Matrix)
```
precision_i = TP_i / (TP_i + FP_i)
  "Of predicted class i, how many were correct?"

recall_i = TP_i / (TP_i + FN_i)
  "Of actual class i, how many did we detect?"

F1_i = 2 × (precision_i × recall_i) / (precision_i + recall_i)
  "Harmonic mean of precision and recall"

where:
  TP_i = true positives for class i (diagonal cell)
  FP_i = false positives (column sum - diagonal)
  FN_i = false negatives (row sum - diagonal)
```

---

## Appendix B: Example Scenarios

### Scenario 1: High Accuracy, Low Kappa (Class Imbalance)
**Situation:** 90% of samples are "pass", judge predicts "pass" for everything.

```
Accuracy: 90% (looks good!)
Cohen's Kappa: 0.0 (no better than random guessing)

Conclusion: Judge is exploiting class imbalance, not actually evaluating.
Action: Check confusion matrix - likely predicting majority class only.
```

### Scenario 2: Moderate Kappa with Few Critical Errors
**Situation:** Judge has moderate kappa but confusion matrix shows most errors are minor (review↔pass).

```
Cohen's Kappa: 0.35 (fair agreement)
Confusion matrix: Only 2 fail→pass errors (critical), but many review→pass (8)

Conclusion: Judge is safe (few critical errors) but over-predicts "review".
Action: Consider adjusting VERDICT_PASS_THRESHOLD slightly lower to reduce review cases.
```

### Scenario 3: Low Kendall's τ, Moderate Kappa (Scaling Issue)
**Situation:** Judge agrees on categorical labels but assigns different continuous scores.

```
Kendall's τ: 0.25 (weak correlation)
Cohen's Kappa: 0.45 (moderate agreement)

Conclusion: Verdict mapping works, but scoring scale needs adjustment.
Action: Judge's continuous scores (0-1) map correctly to verdicts, no changes needed.
```

### Scenario 4: High Recall, Low Precision for "fail" (False Alarms)
**Situation:** Judge detects all failures but flags many false positives.

```
Recall(fail): 0.95 (excellent - catches failures)
Precision(fail): 0.45 (poor - many false alarms)

Confusion matrix shows:
  - 1 pass → fail (false positive)
  - 8 review → fail (false positives)

Conclusion: Judge is too strict, over-predicting failures.
Action: Increase VERDICT_REVIEW_THRESHOLD to reduce false "fail" predictions.
```

### Scenario 5: Weak Review Class (Expected)
**Situation:** Review class has low F1 score (common pattern).

```
F1(fail): 0.78 (good)
F1(review): 0.55 (weak)
F1(pass): 0.82 (good)

Confusion matrix shows:
  - Many review → pass (8)
  - Many review → fail (3)

Conclusion: Borderline cases are hard to classify (expected behavior).
Action: This is acceptable - review cases are inherently ambiguous. Monitor over time.
```

---

## Appendix C: CLI Output Examples

### Compact Format (Terminal)
```
=== Validation Report ===
Total: 150 | Status: ✓ PASSED

Metrics:
  τ=0.420  κ=0.382  κ_w=0.451  acc=83.3%

Confusion Matrix (fail/review/pass):
     f   r   p
f | 20   5   2  (27)
r |  3  15   8  (26)
p |  1   6  40  (47)

Key Insights:
  ✓ Strong performance (75% accuracy)
  ⚠️ Review class weak (F1=0.58)
  ✓ Only 2 critical errors (fail→pass)
```

### Detailed Format (Full Report)
```
=== Validation Report ===
Total Records: 150
Threshold: 0.30
Status: ✓ PASSED

Correlation Metrics (PRIMARY - Pass/Fail):
  Kendall's τ: 0.420 (Moderate positive correlation) → PASSED ✅

Agreement Metrics (REPORT - Industry Standard):
  Cohen's Kappa: 0.382 (Fair agreement)

Confusion Matrix (DEBUG - Actionable Insights):
                    Predicted
                fail  review  pass  | Total
Actual  fail     20      5      2   |  27
        review    3     15      8   |  26
        pass      1      6     40   |  47
        --------------------------------
        Total    24     26     50   | 100

Per-Class Performance:
                Precision  Recall    F1      Support
  fail          0.833      0.741    0.785      27
  review        0.577      0.577    0.577      26
  pass          0.800      0.851    0.825      47

Binary Classification (fail+review vs pass):
  Cohen's Kappa:    0.521
  Agreement Rate:   87.3%

Interpretation:
  ✓ Judge is production-ready (τ ≥ 0.3, κ ≥ 0.3)
  ✓ Strong diagonal (75% accuracy)
  ⚠️ Review class weak (58% F1) - expected for borderline cases
  ✓ Minimal critical errors (only 2 fail→pass)
  → Recommendation: Deploy, monitor review class over time
```
