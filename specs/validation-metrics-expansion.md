# Validation Metrics Expansion Specification

**Purpose**: Expand judge validation beyond Kendall's τ to include industry-standard metrics for categorical agreement and detailed error analysis.

**Scope**: Batch CLI validation, future `themis validate` command

---

## Status Summary

**Completion**: 0/5 Phases Complete (0%)

| Phase | Status | Progress |
|-------|--------|----------|
| Phase 1: Metrics Package | 🔲 Not Started | 0% |
| Phase 2: Confusion Matrix | 🔲 Not Started | 0% |
| Phase 3: Agreement Metrics | 🔲 Not Started | 0% |
| Phase 4: Integration | 🔲 Not Started | 0% |
| Phase 5: Documentation | 🔲 Not Started | 0% |

**Status**: 🔲 **Planning phase - ready to implement**

**Next Steps**: Begin Phase 1 (Metrics Package structure)

---

## Feature Overview

### Problem
- Current validation uses **only Kendall's τ** (rank correlation)
- No categorical agreement metrics (Cohen's Kappa is industry standard)
- No diagnostic tools to understand WHERE judges fail (confusion matrix)
- Limited insight into error patterns (false positives vs false negatives)

### Solution
Add **4 new validation metrics** to complement Kendall's τ:
1. **Confusion Matrix** - Visual diagnostic (where does judge fail?)
2. **Cohen's Kappa** - Categorical agreement (accounts for chance)
3. **Agreement Rate** - Simple accuracy (% exact matches)
4. **Weighted Cohen's Kappa** - Ordinal agreement (partial credit for near-misses)

### Value
- **Comprehensive Validation**: Both continuous (τ) and categorical (κ) evaluation
- **Actionable Insights**: Confusion matrix reveals systematic biases
- **Industry Standard**: Cohen's Kappa is required for ML evaluation papers
- **Stakeholder Communication**: Agreement rate is easy to explain ("85% agreement")
- **Nuanced Analysis**: Weighted κ distinguishes minor errors (review→pass) from major errors (fail→pass)

### Why These Metrics?

| Metric | What It Measures | When It's Critical |
|--------|------------------|-------------------|
| **Kendall's τ** (existing) | Rank correlation | "Do scores correlate relatively?" |
| **Confusion Matrix** | Per-class error breakdown | "Where exactly does judge fail?" |
| **Cohen's Kappa** | Categorical agreement | "Do exact labels agree (accounting for chance)?" |
| **Agreement Rate** | Raw accuracy | "What % of predictions match exactly?" |
| **Weighted Kappa** | Ordinal agreement | "Are errors proportional to severity?" |

---

## Implementation Phases

### Phase 1: Metrics Package Infrastructure (Week 1, Day 1-2)

**Goal**: Create reusable metrics package with clean interfaces

#### Tasks
- [ ] Create `internal/metrics/` package structure
- [ ] Define core types and interfaces
- [ ] Set up comprehensive test suite
- [ ] Document metric interpretation guidelines

#### File Structure
```
internal/metrics/
├── types.go                    # Core types (ConfusionMatrix, MetricsSummary)
├── confusion_matrix.go         # 3x3 matrix implementation
├── confusion_matrix_test.go
├── cohens_kappa.go             # Standard Cohen's Kappa
├── cohens_kappa_test.go
├── weighted_kappa.go           # Weighted variant for ordinal data
├── weighted_kappa_test.go
├── agreement_rate.go           # Simple accuracy calculation
├── agreement_rate_test.go
├── validator.go                # Orchestrates all metrics computation
├── validator_test.go
└── README.md                   # Metric explanations and formulas
```

#### Core Types
```go
// Label represents a verdict: "fail", "review", or "pass"
type Label string

const (
    LabelFail   Label = "fail"
    LabelReview Label = "review"
    LabelPass   Label = "pass"
)

// ConfusionMatrix represents a 3x3 confusion matrix
type ConfusionMatrix struct {
    Matrix map[Label]map[Label]int  // actual → predicted → count
    Labels []Label                    // ordered labels
}

// AgreementMetrics contains categorical agreement measures
type AgreementMetrics struct {
    CohensKappa      float64            `json:"cohens_kappa"`
    WeightedKappa    float64            `json:"weighted_kappa"`
    AgreementRate    float64            `json:"agreement_rate"`
    TotalSamples     int                `json:"total_samples"`
    PerClassMetrics  map[Label]ClassMetrics `json:"per_class_metrics"`
}

// ClassMetrics contains precision, recall, F1 per class
type ClassMetrics struct {
    Precision float64 `json:"precision"`
    Recall    float64 `json:"recall"`
    F1Score   float64 `json:"f1"`
    Support   int     `json:"support"`  // Number of actual instances
}

// MetricsSummary combines all validation metrics
type MetricsSummary struct {
    CorrelationMetrics struct {
        KendallsTau     float64 `json:"kendalls_tau"`
        Interpretation  string  `json:"interpretation"`
    } `json:"correlation_metrics"`

    AgreementMetrics AgreementMetrics `json:"agreement_metrics"`

    ConfusionMatrix ConfusionMatrix `json:"confusion_matrix"`

    // Binary version (collapse review+fail → fail, keep pass)
    BinaryMetrics struct {
        CohensKappa      float64 `json:"cohens_kappa"`
        AgreementRate    float64 `json:"agreement_rate"`
        ConfusionMatrix  map[string]map[string]int `json:"confusion_matrix"`
    } `json:"binary_metrics"`
}
```

#### Test Requirements
- [ ] Test with perfect agreement (all metrics = 1.0)
- [ ] Test with random agreement (κ ≈ 0, accuracy > 0)
- [ ] Test with complete disagreement (κ < 0)
- [ ] Test with imbalanced classes (90% pass, 5% review, 5% fail)
- [ ] Test edge cases (empty dataset, single class)

**Deliverable**: `internal/metrics/` package with 100% test coverage

---

### Phase 2: Confusion Matrix Implementation (Week 1, Day 3)

**Goal**: Implement confusion matrix as foundation for other metrics

#### Tasks
- [ ] Implement 3x3 confusion matrix builder
- [ ] Add row/column totals calculation
- [ ] Implement per-class precision/recall/F1
- [ ] Add pretty-print formatting for CLI display
- [ ] Create visualization helpers

#### Implementation Details

**Confusion Matrix Structure** (3-class: fail/review/pass):
```
                    Predicted
                fail  review  pass  | Total
Actual  fail     TP     FP     FP   |  27
        review   FN     TP     FP   |  26
        pass     FN     FN     TP   |  47
        --------------------------------
        Total    24     26     50   | 100
```

**Formulas** (per-class):
- **Precision** = TP / (TP + FP) - "Of predicted X, how many were actually X?"
- **Recall** = TP / (TP + FN) - "Of actual X, how many did we predict?"
- **F1** = 2 × (Precision × Recall) / (Precision + Recall) - harmonic mean

**Binary Collapse** (for comparison):
- Merge `fail` + `review` → `fail`
- Keep `pass` as is
- Compute 2x2 confusion matrix

#### Code Structure
```go
// confusion_matrix.go

// Build creates confusion matrix from actual and predicted labels
func Build(actual, predicted []Label) (*ConfusionMatrix, error)

// Add increments count for actual→predicted pair
func (cm *ConfusionMatrix) Add(actual, predicted Label)

// Get returns count for actual→predicted pair
func (cm *ConfusionMatrix) Get(actual, predicted Label) int

// TotalActual returns total count for actual label (row sum)
func (cm *ConfusionMatrix) TotalActual(label Label) int

// TotalPredicted returns total count for predicted label (column sum)
func (cm *ConfusionMatrix) TotalPredicted(label Label) int

// TotalCorrect returns diagonal sum (all correct predictions)
func (cm *ConfusionMatrix) TotalCorrect() int

// ComputeClassMetrics calculates precision/recall/F1 per class
func (cm *ConfusionMatrix) ComputeClassMetrics() map[Label]ClassMetrics

// ToBinary collapses to 2x2 matrix (fail+review → fail, pass → pass)
func (cm *ConfusionMatrix) ToBinary() *ConfusionMatrix

// String returns pretty-printed table for CLI display
func (cm *ConfusionMatrix) String() string
```

#### Test Cases
```go
func TestConfusionMatrix_Perfect(t *testing.T) {
    // All predictions match → precision/recall/F1 = 1.0
}

func TestConfusionMatrix_AllWrong(t *testing.T) {
    // No diagonal entries → precision/recall = 0
}

func TestConfusionMatrix_Imbalanced(t *testing.T) {
    // 90% pass, 10% fail → check class-specific metrics
}

func TestConfusionMatrix_Binary(t *testing.T) {
    // Test fail+review collapse
}

func TestConfusionMatrix_String(t *testing.T) {
    // Test pretty-print formatting
}
```

**Deliverable**: Working confusion matrix with per-class metrics

---

### Phase 3: Agreement Metrics Implementation (Week 1, Day 4-5)

**Goal**: Implement Cohen's Kappa, Weighted Kappa, and Agreement Rate

#### Task 3.1: Agreement Rate (30 minutes)
- [ ] Implement simple accuracy calculation
- [ ] Add tests with various agreement levels

```go
// agreement_rate.go

// Compute calculates agreement rate (simple accuracy)
// Formula: correct / total
func ComputeAgreementRate(actual, predicted []Label) (float64, error) {
    if len(actual) != len(predicted) || len(actual) == 0 {
        return 0, errors.New("invalid input")
    }

    correct := 0
    for i := range actual {
        if actual[i] == predicted[i] {
            correct++
        }
    }

    return float64(correct) / float64(len(actual)), nil
}
```

#### Task 3.2: Cohen's Kappa (2-3 hours)
- [ ] Implement standard Cohen's Kappa formula
- [ ] Handle chance agreement calculation
- [ ] Add interpretation guidelines

**Formula**:
```
κ = (p_o - p_e) / (1 - p_e)

where:
  p_o = observed agreement (diagonal sum / total)
  p_e = expected agreement by chance = Σ(p_actual_i × p_predicted_i)
```

**Interpretation**:
- κ < 0.00: Poor agreement (worse than chance)
- κ = 0.00-0.20: Slight agreement
- κ = 0.21-0.40: Fair agreement
- κ = 0.41-0.60: Moderate agreement
- κ = 0.61-0.80: Substantial agreement
- κ = 0.81-1.00: Almost perfect agreement

```go
// cohens_kappa.go

// Compute calculates Cohen's Kappa from confusion matrix
func ComputeCohensKappa(cm *ConfusionMatrix) (float64, error)

// InterpretKappa returns human-readable interpretation
func InterpretKappa(kappa float64) string
```

#### Task 3.3: Weighted Cohen's Kappa (2-3 hours)
- [ ] Implement weighted variant with distance matrix
- [ ] Define ordinal weights for fail/review/pass
- [ ] Test with various weight schemes

**Weight Matrix** (ordinal distance):
```
        fail  review  pass
fail     0      1      4
review   1      0      1
pass     4      1      0
```

**Rationale**:
- Adjacent categories (fail→review, review→pass): weight = 1 (minor error)
- Skip category (fail→pass): weight = 4 (major error - 2× distance)

**Formula**:
```
κ_w = 1 - (Σ w_ij × O_ij) / (Σ w_ij × E_ij)

where:
  w_ij = weight matrix (distance between categories)
  O_ij = observed counts (confusion matrix)
  E_ij = expected counts by chance
```

```go
// weighted_kappa.go

// WeightScheme defines distance between categories
type WeightScheme map[Label]map[Label]float64

// LinearWeights returns ordinal weights (distance-based)
func LinearWeights() WeightScheme

// QuadraticWeights returns squared-distance weights
func QuadraticWeights() WeightScheme

// ComputeWeightedKappa calculates weighted Cohen's Kappa
func ComputeWeightedKappa(cm *ConfusionMatrix, weights WeightScheme) (float64, error)
```

#### Test Cases
```go
func TestCohensKappa_PerfectAgreement(t *testing.T) {
    // κ = 1.0
}

func TestCohensKappa_ChanceAgreement(t *testing.T) {
    // Random predictions → κ ≈ 0
}

func TestCohensKappa_ImbalancedClasses(t *testing.T) {
    // Test with 90% pass → κ should account for prevalence
}

func TestWeightedKappa_OrdinalErrors(t *testing.T) {
    // Compare κ_w vs standard κ with ordinal mistakes
}
```

**Deliverable**: All agreement metrics implemented and tested

---

### Phase 4: Integration with Validation Pipeline (Week 2, Day 1-2)

**Goal**: Integrate new metrics into existing validation workflow

#### Tasks
- [ ] Update `internal/batch/validator.go` to use new metrics package
- [ ] Enhance `ValidationReport` struct with all metrics
- [ ] Update CLI output formatting
- [ ] Maintain backwards compatibility with existing τ-only output
- [ ] Add metric comparison mode (side-by-side)

#### File Changes

**Update `internal/batch/validator.go`**:
```go
import "github.com/Terminus-Lab/themis/internal/metrics"

// ValidationReport now includes comprehensive metrics
type ValidationReport struct {
    // Existing fields
    Passed           bool    `json:"passed"`
    TotalRecords     int     `json:"total_records"`
    Threshold        float64 `json:"threshold"`

    // Expanded metrics
    MetricsSummary   metrics.MetricsSummary `json:"metrics_summary"`

    // Legacy fields (for backwards compatibility)
    KendallsTau      float64 `json:"kendalls_tau"`
    Interpretation   string  `json:"interpretation"`
}

// Validate computes all metrics
func (v *Validator) Validate(ctx context.Context, data []EvaluationData) (*ValidationReport, error) {
    // 1. Run evaluations
    // 2. Compute Kendall's τ (existing)
    // 3. Build confusion matrix
    // 4. Compute all agreement metrics
    // 5. Generate comprehensive report
}
```

**Update CLI Output** (`cmd/batch/main.go` or future `cmd/themis/commands/validate.go`):
```go
// Pretty-print validation report
func printValidationReport(report *ValidationReport) {
    fmt.Println("=== Validation Report ===")
    fmt.Printf("Total Records: %d\n", report.TotalRecords)
    fmt.Printf("Threshold: %.2f\n", report.Threshold)
    fmt.Printf("Status: %s\n\n", passFailStatus(report.Passed))

    // Correlation Metrics
    fmt.Println("Correlation Metrics:")
    fmt.Printf("  Kendall's τ: %.3f (%s)\n",
        report.MetricsSummary.CorrelationMetrics.KendallsTau,
        report.MetricsSummary.CorrelationMetrics.Interpretation)

    // Agreement Metrics
    fmt.Println("\nAgreement Metrics:")
    fmt.Printf("  Cohen's Kappa:    %.3f\n", report.MetricsSummary.AgreementMetrics.CohensKappa)
    fmt.Printf("  Weighted Kappa:   %.3f\n", report.MetricsSummary.AgreementMetrics.WeightedKappa)
    fmt.Printf("  Agreement Rate:   %.1f%%\n", report.MetricsSummary.AgreementMetrics.AgreementRate * 100)

    // Confusion Matrix
    fmt.Println("\nConfusion Matrix:")
    fmt.Println(report.MetricsSummary.ConfusionMatrix.String())

    // Per-Class Metrics
    fmt.Println("\nPer-Class Performance:")
    printClassMetrics(report.MetricsSummary.AgreementMetrics.PerClassMetrics)
}
```

#### Output Examples

**Terminal Output**:
```
=== Validation Report ===
Total Records: 150
Threshold: 0.30
Status: ✓ PASSED

Correlation Metrics:
  Kendall's τ: 0.420 (Moderate positive correlation)

Agreement Metrics:
  Cohen's Kappa:    0.382
  Weighted Kappa:   0.451
  Agreement Rate:   83.3%

Confusion Matrix:
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
```

**JSON Output** (for programmatic access):
```json
{
  "passed": true,
  "total_records": 150,
  "threshold": 0.3,

  "metrics_summary": {
    "correlation_metrics": {
      "kendalls_tau": 0.420,
      "interpretation": "Moderate positive correlation"
    },

    "agreement_metrics": {
      "cohens_kappa": 0.382,
      "weighted_kappa": 0.451,
      "agreement_rate": 0.833,
      "total_samples": 150,
      "per_class_metrics": {
        "fail": {
          "precision": 0.833,
          "recall": 0.741,
          "f1": 0.785,
          "support": 27
        },
        "review": {
          "precision": 0.577,
          "recall": 0.577,
          "f1": 0.577,
          "support": 26
        },
        "pass": {
          "precision": 0.800,
          "recall": 0.851,
          "f1": 0.825,
          "support": 47
        }
      }
    },

    "confusion_matrix": {
      "matrix": {
        "fail": {"fail": 20, "review": 5, "pass": 2},
        "review": {"fail": 3, "review": 15, "pass": 8},
        "pass": {"fail": 1, "review": 6, "pass": 40}
      },
      "labels": ["fail", "review", "pass"]
    },

    "binary_metrics": {
      "cohens_kappa": 0.521,
      "agreement_rate": 0.873,
      "confusion_matrix": {
        "fail": {"fail": 43, "pass": 10},
        "pass": {"fail": 9, "pass": 88}
      }
    }
  },

  "kendalls_tau": 0.420,
  "interpretation": "Moderate positive correlation"
}
```

#### Backwards Compatibility
- [ ] Ensure existing scripts parsing τ-only output still work
- [ ] Add `--output-format` flag: `legacy`, `full` (default), `json`
- [ ] Legacy format outputs only Kendall's τ for backwards compatibility

**Deliverable**: Integrated metrics pipeline with enhanced validation reports

---

### Phase 5: Documentation & Testing (Week 2, Day 3-5)

**Goal**: Comprehensive documentation and validation with test dataset

#### Task 5.1: Create Test Dataset (Day 3)
- [ ] Generate 150 synthetic samples (50 fail, 50 review, 50 pass)
- [ ] Include edge cases (ambiguous responses, boundary cases)
- [ ] Add ground truth annotations

**Test Dataset Structure** (`resources/validation_test_dataset.jsonl`):
```jsonl
{"event_id": "001", "query": "...", "answer": "...", "human_annotation": "fail"}
{"event_id": "002", "query": "...", "answer": "...", "human_annotation": "review"}
{"event_id": "003", "query": "...", "answer": "...", "human_annotation": "pass"}
...
```

**Distribution Requirements**:
- Balanced classes: 50 fail, 50 review, 50 pass
- Include systematic errors: judge over-predicts "review", under-predicts "fail"
- Include edge cases: borderline fail/review, borderline review/pass
- Include perfect matches: 20% exact agreement

#### Task 5.2: Metric Comparison Analysis (Day 3-4)
- [ ] Run validation with test dataset
- [ ] Compare behavior across different error patterns
- [ ] Document when each metric provides unique insight

**Comparison Scenarios**:

| Scenario | τ | κ | κ_w | Accuracy | Insight |
|----------|---|---|-----|----------|---------|
| Perfect agreement | 1.0 | 1.0 | 1.0 | 100% | All metrics agree |
| Random predictions | 0.0 | 0.0 | 0.0 | 33% | Accuracy misleading |
| Ordinal errors (review↔pass) | 0.5 | 0.3 | 0.5 | 60% | κ_w gives partial credit |
| Major errors (fail↔pass) | -0.2 | -0.1 | -0.3 | 40% | All metrics detect reversal |
| Class imbalance (90% pass) | 0.6 | 0.4 | 0.5 | 85% | κ corrects for prevalence |

#### Task 5.3: Documentation (Day 4-5)
- [ ] Add metric explanations to `docs/metrics/README.md`
- [ ] Update `docs/testing/batch-tests.md` with new validation format
- [ ] Create `docs/metrics/interpretation-guide.md`
- [ ] Add examples to `docs/examples/validation-report-examples.md`

**New Documentation Files**:

1. **`docs/metrics/README.md`**:
   - Formula for each metric
   - When to use which metric
   - Interpretation guidelines
   - Common pitfalls

2. **`docs/metrics/interpretation-guide.md`**:
   - Case studies comparing metrics
   - Decision trees for metric selection
   - Troubleshooting low agreement

3. **`docs/examples/validation-report-examples.md`**:
   - 5 example reports with different patterns
   - Annotated explanations of what each metric reveals
   - Actionable recommendations per scenario

#### Task 5.4: Unit Testing (Day 5)
- [ ] Achieve >95% code coverage for `internal/metrics/`
- [ ] Add integration tests for full validation pipeline
- [ ] Test edge cases (empty data, single class, ties)

**Test Coverage Requirements**:
- All formulas verified against known ground truth
- Edge cases handled gracefully
- Performance benchmarks (should handle 10k samples in <1s)

**Deliverable**: Complete documentation + validated test dataset

---

## Success Criteria

### Technical Requirements
- ✅ All 5 metrics implemented and tested (τ, κ, κ_w, accuracy, confusion matrix)
- ✅ >95% test coverage for `internal/metrics/` package
- ✅ Backwards compatibility maintained (existing τ-only scripts work)
- ✅ Performance: Process 1000 samples in <2 seconds
- ✅ Clean integration with existing validation pipeline

### User Experience
- ✅ CLI output is readable and actionable
- ✅ JSON output is machine-parseable
- ✅ Confusion matrix reveals error patterns clearly
- ✅ Per-class metrics help debug specific issues
- ✅ Binary metrics provide simpler pass/fail view

### Documentation
- ✅ Each metric has formula + interpretation guide
- ✅ Examples show when each metric provides unique insight
- ✅ Test dataset demonstrates metric behavior
- ✅ Integration with future `themis validate` command documented

### Validation
- ✅ Test dataset (150 samples) runs successfully
- ✅ Metrics behave correctly across different error patterns
- ✅ All edge cases handled (imbalanced data, ties, empty results)

---

## Dependencies

### Go Packages (New)
- None - pure Go implementation

### Go Packages (Existing)
- `gonum.org/v1/gonum/stat` - Already used for Kendall's τ

### Test Data
- 150-sample annotated dataset with balanced classes
- Edge case examples (ambiguous labels, boundary cases)

### Documentation Tools
- Existing docs structure (`docs/metrics/`, `docs/examples/`)

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Metric formulas implemented incorrectly | High | Validate against published papers + Python sklearn |
| Confusion matrix visualization unclear | Medium | User testing with CLI output, iterate on format |
| Performance issues with large datasets | Low | Benchmark with 10k samples, optimize if needed |
| Users confused by multiple metrics | Medium | Clear documentation + interpretation guide |
| Breaking changes to existing validation | High | Maintain backwards compatibility, add `--output-format` flag |

---

## Open Questions

### Q1: Should we report both 3-class and binary metrics by default?
**Proposal**: Yes, include both in JSON output. CLI shows 3-class by default, add `--binary` flag to show 2x2 matrix.

### Q2: What weight scheme for weighted kappa?
**Proposal**: Use linear weights (ordinal distance) by default. Add `--weight-scheme` flag: `linear`, `quadratic`.

### Q3: Should confusion matrix show counts or percentages?
**Proposal**: Show counts by default (easier to understand). Add row percentages in parentheses for normalized view.

### Q4: Minimum sample size for reliable metrics?
**Proposal**: Warn if n < 30 samples. Document that κ requires n > 50 for stability.

### Q5: Should we add per-judge metrics (validate each judge individually)?
**Proposal**: Not in this phase. Add `--judge <name>` flag in future iteration to validate specific judge.

---

## Timeline

**Week 1 (Implementation)**:
- Day 1-2: Metrics package infrastructure + confusion matrix
- Day 3: Cohen's Kappa + agreement rate
- Day 4-5: Weighted Kappa + integration

**Week 2 (Testing & Documentation)**:
- Day 1-2: CLI integration + output formatting
- Day 3: Test dataset creation + metric comparison
- Day 4-5: Documentation + user guide

**Total**: 2 weeks to production-ready metrics expansion

---

## Next Steps

1. **Approve this specification** (review and sign off)
2. **Create GitHub issues** for each phase
3. **Begin Phase 1** (Metrics package structure)
4. **Iterate based on test results** (adjust formulas, output format)

---

## Future Enhancements (Post-Launch)

### Phase 6 (Optional): Advanced Metrics
- Spearman's ρ (parametric alternative to Kendall's τ)
- Mean Absolute Error (MAE) for continuous scores
- Per-judge validation (validate each judge independently)

### Phase 7 (Optional): Visualization
- HTML report generation (confusion matrix heatmap)
- Time-series tracking (τ/κ over multiple validation runs)
- Dashboard integration (show metrics in web UI)

### Phase 8 (Optional): Statistical Tests
- Bootstrap confidence intervals for κ
- Statistical significance testing (McNemar's test)
- Sample size recommendations (power analysis)

---

## Appendix: Metric Formulas

### Cohen's Kappa
```
κ = (p_o - p_e) / (1 - p_e)

where:
  p_o = observed agreement = (diagonal sum) / total
  p_e = expected agreement = Σ(p_actual_i × p_predicted_i)

Interpretation:
  < 0.00: Poor (worse than chance)
  0.00-0.20: Slight
  0.21-0.40: Fair
  0.41-0.60: Moderate
  0.61-0.80: Substantial
  0.81-1.00: Almost perfect
```

### Weighted Cohen's Kappa
```
κ_w = 1 - (Σ w_ij × O_ij) / (Σ w_ij × E_ij)

where:
  w_ij = weight matrix (distance between categories)
  O_ij = observed counts
  E_ij = expected counts by chance

Linear weights (ordinal):
  w(i,j) = |i - j|  (distance between categories)
```

### Agreement Rate
```
accuracy = correct / total

where:
  correct = number of exact matches
  total = number of samples
```

### Per-Class Metrics
```
precision_i = TP_i / (TP_i + FP_i)
recall_i = TP_i / (TP_i + FN_i)
F1_i = 2 × (precision_i × recall_i) / (precision_i + recall_i)

where:
  TP_i = true positives for class i (diagonal)
  FP_i = false positives for class i (column sum - diagonal)
  FN_i = false negatives for class i (row sum - diagonal)
```

---

## Appendix: Example Scenarios

### Scenario 1: High Accuracy, Low Kappa
**Situation**: 90% of samples are "pass", judge predicts "pass" for everything.

```
Accuracy: 90% (looks good!)
κ: 0.0 (no better than random)

Conclusion: Judge is exploiting class imbalance, not actually evaluating.
```

### Scenario 2: Moderate Kappa, High Weighted Kappa
**Situation**: Judge makes ordinal errors (review→pass) but no major errors (fail→pass).

```
κ: 0.35 (fair agreement)
κ_w: 0.52 (moderate agreement)

Conclusion: Judge understands quality spectrum but needs calibration on threshold.
```

### Scenario 3: Low Kendall's τ, Moderate Kappa
**Situation**: Judge agrees on labels but assigns different continuous scores.

```
τ: 0.25 (weak correlation)
κ: 0.45 (moderate agreement)

Conclusion: Verdict mapping works, but scoring scale needs adjustment.
```
