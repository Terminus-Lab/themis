# Validation Metrics Expansion - Specification

**Status:** In Progress
**Date:** 2026-03-14
**Updated:** 2026-03-15

---

## Status Summary

**Completion**: 3/5 Phases Complete (60%)

| Phase | Status | Progress |
|-------|--------|----------|
| Phase 1: Metrics Package Infrastructure | ✅ Complete | 100% |
| Phase 2: Confusion Matrix Implementation | ✅ Complete | 100% |
| Phase 3: Cohen's Kappa + Kendall's Tau Refactor | ✅ Complete | 100% |
| Phase 4: Integration with Validation Pipeline | 🔲 Not Started | 0% |
| Phase 5: Documentation & Testing | 🔲 Not Started | 0% |

**Current Status:** ✅ Phases 1-3 complete. Ready for Phase 4 (Integration)

**Next Steps:** Integrate new metrics into validation pipeline

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

### Phase 1: Metrics Package Infrastructure ✅ COMPLETE

**Goal:** Create reusable metrics package with clean interfaces

**Tasks:**
- [X] Create `internal/metrics/` package structure
- [X] Define core types: `Label`, `ConfusionMatrix`, `ClassMetrics`
- [X] Document metric formulas in README

**Deliverable:** ✅ Package structure with types and documentation

---

### Phase 2: Confusion Matrix Implementation ✅ COMPLETE

**Goal:** Implement confusion matrix as foundation for other metrics

**Tasks:**
- [X] Implement `Build()` - creates 3x3 matrix from actual/predicted labels
- [X] Implement `Get()`, `TotalActual()`, `TotalPredict()`, `TotalCorrect()`, `TotalSample()`
- [X] Implement `ComputeClassMetrics()` - calculates precision/recall/F1
- [X] Implement `ToBinary()` - collapses to 2x2 matrix
- [X] Add comprehensive tests (16 tests, 100% coverage)
- [X] Document confusion matrix concepts in README

**Deliverable:** ✅ Full confusion matrix implementation with tests and documentation

---

### Phase 3: Cohen's Kappa + Kendall's Tau Refactor ✅ COMPLETE

**Goal:** Implement Cohen's Kappa and refactor Kendall's tau into metrics package

**Tasks:**
- [X] Implement `ComputeCohensKappa(cm *ConfusionMatrix)` - computes κ from confusion matrix
- [X] Implement `InterpretKappa(kappa float64)` - returns human-readable interpretation
- [X] Add comprehensive tests (perfect agreement, random, imbalanced)
- [X] Refactor Kendall's tau from `internal/batch/validator.go` into `internal/metrics/kendalls_tau.go`
- [X] Implement `ComputeKendallsTau(humanAnnotations, llmPredictions []Label)` in metrics package
- [X] Implement `InterpretTau(tau float64)` in metrics package
- [X] Add tests for Kendall's tau in metrics package
- [X] Update `internal/batch/validator.go` to use metrics package functions

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
ComputeKendallsTau(humanAnnotations, llmPredictions []Label) (float64, error)
InterpretTau(tau float64) string
```

**Deliverable:** ✅ All 3 core metrics (Kendall's τ, Confusion Matrix, Cohen's Kappa) implemented in metrics package with 95.1% test coverage

---

### Phase 4: Integration with Validation Pipeline 

**Goal:** Integrate new metrics into existing validation workflow

**Tasks:**
- [X] Update `internal/batch/validator.go` to use metrics package
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

### Phase 5: Documentation & Testing

**Goal:** Comprehensive documentation and validation with test dataset

**Task 5.1: Create Test Dataset**
- [ ] Generate 150 synthetic samples (50 fail, 50 review, 50 pass)
- [ ] Include edge cases (ambiguous, boundary, perfect matches)
- [ ] Add to `resources/validation_test_dataset.jsonl`

**Task 5.2: Metric Comparison Analysis**
- [ ] Run validation with test dataset
- [ ] Compare metrics across different error patterns
- [ ] Document when each metric provides unique insight

**Task 5.3: Documentation**
- [ ] Create `docs/metrics/README.md` - formulas and interpretations
- [ ] Create `docs/metrics/interpretation-guide.md` - case studies
- [ ] Update `docs/testing/batch-tests.md` - new validation examples
- [ ] Create `docs/examples/validation-report-examples.md` - 5 scenarios

**Task 5.4: Unit Testing**
- [ ] Achieve >95% code coverage for `internal/metrics/`
- [ ] Add integration tests for full validation pipeline
- [ ] Performance benchmarks (10k samples in <1s)

**Deliverable:** Complete documentation + validated test dataset

---

## 6. Success Criteria

### Technical Requirements
- All 3 core metrics in `internal/metrics/` implemented and tested:
  - `kendalls_tau.go` (moved from batch/validator.go)
  - `confusion_matrix.go`
  - `cohens_kappa.go`
  - `validator.go` (orchestrator)
- >95% test coverage for `internal/metrics/` package
- Backwards compatibility maintained (existing scripts work unchanged)
- Performance: Process 1000 samples in <2 seconds
- Clean integration with existing validation pipeline
- Simple 3-metric decision framework documented

### User Experience
- CLI output is readable and actionable
- JSON output is machine-parseable
- Confusion matrix reveals error patterns clearly
- Per-class metrics help debug specific issues
- Metric interpretation is well-documented

### Documentation
- Each metric has formula + interpretation guide
- Examples show when each metric provides unique insight
- Test dataset (150 samples) demonstrates metric behavior
- Integration with future `themis validate` command documented

## 7. Future Enhancements

### Phase 6: Additional Metrics
- **Weighted Cohen's Kappa** - Partial credit for ordinal errors (fail→review vs fail→pass)
- **Agreement Rate** - Simple accuracy (% exact matches)
- **Spearman's ρ** - Parametric alternative to Kendall's τ
- **Mean Absolute Error (MAE)** - For continuous scores
- **Per-judge validation** - Validate each judge independently

**Note:** Intentionally excluded to keep simple (3 metrics only)