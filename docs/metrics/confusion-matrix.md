# Confusion Matrix for LLM Judge Validation

## The Core Problem

When validating an LLM judge, **Kendall's τ tells you if rankings correlate**, but it doesn't answer critical questions:

- **Where exactly does the judge fail?** Does it over-predict "pass" and miss failures?
- **What kind of errors does it make?** Does it confuse "review" with "pass" or "fail" with "pass"?
- **Is the judge biased?** Does it systematically favor one verdict over others?

**Confusion Matrix solves this** by showing the exact breakdown of correct predictions vs. each type of error.

---

## What is a Confusion Matrix?

A confusion matrix is a table that visualizes the performance of a classification system by comparing **actual labels** (ground truth) vs. **predicted labels**.

### Structure (3-Class: fail/review/pass)

```
                    Predicted
                fail  review  pass  | Total
Actual  fail     20      5      2   |  27
        review    3     15      8   |  26
        pass      1      6     40   |  47
        --------------------------------
        Total    24     26     50   | 100
```

**Reading the matrix:**
- **Rows**: Actual labels (what humans annotated)
- **Columns**: Predicted labels (what LLM judge output)
- **Diagonal (20, 15, 40)**: Correct predictions ✅
- **Off-diagonal**: Errors ❌

**Key insights:**
- **Row totals**: How many samples exist per class in ground truth
- **Column totals**: How many samples the judge predicted per class
- **Cell (i, j)**: How many times actual=i was predicted as j

---

## Detailed Example: Validating Themis Judge

### Scenario: Customer Support Response Evaluation

You have 100 annotated customer support responses with human labels (fail/review/pass). You run Themis judge and compare results.

**Data preparation:**
```jsonl
{"event_id": "1", "query": "...", "answer": "...", "human_annotation": "fail"}
{"event_id": "2", "query": "...", "answer": "...", "human_annotation": "pass"}
{"event_id": "3", "query": "...", "answer": "...", "human_annotation": "review"}
...
{"event_id": "100", "query": "...", "answer": "...", "human_annotation": "pass"}
```

**After running validation:**

| Event ID | Human Label | Judge Verdict | Match? |
|----------|-------------|---------------|--------|
| E1       | fail        | fail          | ✓      |
| E2       | fail        | review        | ✗      |
| E3       | fail        | pass          | ✗      |
| E4       | review      | fail          | ✗      |
| E5       | review      | review        | ✓      |
| E6       | review      | pass          | ✗      |
| E7       | pass        | fail          | ✗      |
| E8       | pass        | review        | ✗      |
| E9       | pass        | pass          | ✓      |
| ...      | ...         | ...           | ...    |

### Building the Confusion Matrix

**Step 1: Count each (actual, predicted) pair**

Go through all 100 records and tally:

```go
// Pseudocode
confusionMatrix := map[string]map[string]int{
    "fail":   {"fail": 0, "review": 0, "pass": 0},
    "review": {"fail": 0, "review": 0, "pass": 0},
    "pass":   {"fail": 0, "review": 0, "pass": 0},
}

for each record:
    actual := record.HumanAnnotation
    predicted := record.JudgeVerdict
    confusionMatrix[actual][predicted]++
```

**Step 2: Result**

```
                    Predicted
                fail  review  pass  | Total
Actual  fail     20      5      2   |  27
        review    3     15      8   |  26
        pass      1      6     40   |  47
        --------------------------------
        Total    24     26     50   | 100
```

### Reading the Matrix

#### **Diagonal = Correct Predictions** ✅

- **20 fail → fail**: Judge correctly identified 20 failures
- **15 review → review**: Judge correctly identified 15 uncertain cases
- **40 pass → pass**: Judge correctly identified 40 good responses

**Total correct: 20 + 15 + 40 = 75 out of 100 (75% accuracy)**

#### **Off-Diagonal = Errors** ❌

**Type 1: False Negatives (FN) - Missing Real Failures**

Look at the "fail" row:
- **5 fail → review**: Judge said "review" but human said "fail"
- **2 fail → pass**: Judge said "pass" but human said "fail" ⚠️ **CRITICAL ERROR**

**Impact:** 7 out of 27 failures (26%) were missed. Bad responses slipped through!

**Type 2: False Positives (FP) - False Alarms**

Look at the "pass" column:
- **2 fail → pass**: Covered above (FN for fail class)
- **8 review → pass**: Judge was optimistic, passed borderline cases
- **Total FP for "pass"**: 2 + 8 = 10 false positives

Look at "fail" column:
- **3 review → fail**: Judge too strict on borderline cases
- **1 pass → fail**: Judge failed a good response ⚠️ **BAD UX**

**Impact:** 4 out of 24 predicted failures (17%) were false alarms. Users get frustrated!

**Type 3: Review Confusion**

- **3 review → fail**: Judge too harsh
- **8 review → pass**: Judge too lenient

Review cases are the hardest - only 58% accuracy (15/26) for borderline responses.

---

## Per-Class Metrics: Precision, Recall, F1

The confusion matrix enables calculation of **precision, recall, and F1** per class.

### Formulas

For each class (fail, review, pass):

```
Precision = TP / (TP + FP)
  "Of all predictions of this class, how many were correct?"

Recall = TP / (TP + FN)
  "Of all actual instances of this class, how many did we detect?"

F1 = 2 × (Precision × Recall) / (Precision + Recall)
  "Harmonic mean of precision and recall"
```

### Calculating for "fail" Class

**From the matrix:**
```
                    Predicted
                fail  review  pass
Actual  fail     20      5      2    ← TP=20, FN=5+2=7
        review    3     ...    ...   ↓
        pass      1     ...    ...   FP = 3+1=4
                 ---
        Total    24
```

- **True Positives (TP)**: Cell [fail, fail] = 20
- **False Positives (FP)**: Column "fail" - TP = 24 - 20 = 4
  - Includes: 3 review→fail + 1 pass→fail
- **False Negatives (FN)**: Row "fail" - TP = 27 - 20 = 7
  - Includes: 5 fail→review + 2 fail→pass

**Calculations:**
```
Precision(fail) = 20 / (20 + 4) = 20 / 24 = 0.833 (83.3%)
  → "When judge says 'fail', it's correct 83% of the time"

Recall(fail) = 20 / (20 + 7) = 20 / 27 = 0.741 (74.1%)
  → "Judge detects 74% of actual failures"

F1(fail) = 2 × (0.833 × 0.741) / (0.833 + 0.741) = 0.785
```

### Calculating for All Classes

| Class  | TP | FP | FN | Precision | Recall | F1    | Support |
|--------|----|----|----|-----------| -------|-------|---------|
| fail   | 20 | 4  | 7  | 0.833     | 0.741  | 0.785 | 27      |
| review | 15 | 11 | 11 | 0.577     | 0.577  | 0.577 | 26      |
| pass   | 40 | 10 | 7  | 0.800     | 0.851  | 0.825 | 47      |

**Interpretation:**

- **fail class**: Good precision (83%), decent recall (74%)
  - Judge is conservative - when it says "fail", it's usually right
  - But it misses 26% of failures (7/27)

- **review class**: Weak performance (58%)
  - Judge struggles with borderline cases
  - Many review cases misclassified as pass (8) or fail (3)

- **pass class**: Strong performance (80%+ on both)
  - Judge reliably identifies good responses
  - 85% recall means it catches most good responses

---

## Confusion Matrix vs Kendall's Tau: What Each Reveals

### Example 1: High Accuracy, Low Kendall's τ

```
                    Predicted
                fail  review  pass  | Total
Actual  fail     25      2      0   |  27
        review    0     24      2   |  26
        pass      0      0     47   |  47
        --------------------------------
        Total    25     26     49   | 100

Accuracy: 96% (25+24+47)/100
Kendall's τ: 0.45 (moderate)
```

**What confusion matrix reveals:**
- Judge is excellent at exact classification (96%)
- Almost no major errors (fail↔pass)
- Minor errors only in adjacent categories

**Why τ is moderate:**
- Many ties in verdict distribution
- Kendall's τ is conservative with ordinal data
- τ measures rank correlation, not exact matches

**Conclusion:** Judge is production-ready despite moderate τ

### Example 2: Moderate Accuracy, Negative Kendall's τ

```
                    Predicted
                fail  review  pass  | Total
Actual  fail      2      5     20   |  27
        review    8     10      8   |  26
        pass     14     11     22   |  47
        --------------------------------
        Total    24     26     50   | 100

Accuracy: 34% (2+10+22)/100
Kendall's τ: -0.15 (negative correlation!)
```

**What confusion matrix reveals:**
- **Critical issue**: 20 failures predicted as pass (top-right cell)
- Judge is inverted - worse than random
- Heavy bias toward "pass" verdict (50% of predictions)

**Why τ is negative:**
- Rankings are reversed (bad responses ranked high)
- Systematic anti-correlation with human judgment

**Conclusion:** Judge is broken, do not use

### Example 3: Class Imbalance Problem

```
                    Predicted
                fail  review  pass  | Total
Actual  fail      0      0      5   |   5
        review    0      0     10   |  10
        pass      0      0     85   |  85
        --------------------------------
        Total     0      0    100   | 100

Accuracy: 85% (85/100)
Kendall's τ: 0.0 (no correlation)
Cohen's κ: 0.0 (no agreement)
```

**What confusion matrix reveals:**
- Judge **always predicts "pass"** (right column is 100)
- It's exploiting class imbalance (85% of data is pass)
- 85% accuracy is misleading
- Judge is not evaluating at all!

**Why accuracy is misleading:**
- Baseline accuracy (always predict majority class) = 85%
- Judge adds zero value over naive baseline

**Conclusion:** Judge is useless despite high accuracy

---

## Common Patterns and What They Mean

### Pattern 1: **Diagonal Dominant** ✅ Good Judge

```
                    Predicted
                fail  review  pass
Actual  fail     25      2      0
        review    3     20      3
        pass      0      2     45
```

- **Strong diagonal**: Most predictions are correct
- **Minimal off-diagonal**: Few errors
- **Symmetric errors**: No systematic bias
- **Verdict:** Judge is reliable

### Pattern 2: **Upper Triangle** ⚠️ Lenient Judge

```
                    Predicted
                fail  review  pass
Actual  fail     10     10      7
        review    2     15      9
        pass      0      2     45
```

- **Heavy upper-right**: Many fail→pass, review→pass errors
- Judge is **too optimistic**
- Bad responses slip through
- **Risk:** Safety issues, poor user experience

### Pattern 3: **Lower Triangle** ⚠️ Harsh Judge

```
                    Predicted
                fail  review  pass
Actual  fail     25      2      0
        review   10     10      6
        pass      8      5     34
```

- **Heavy lower-left**: Many pass→fail, review→fail errors
- Judge is **too strict**
- Good responses rejected
- **Risk:** User frustration, false alarms

### Pattern 4: **Review Column Heavy** ⚠️ Uncertain Judge

```
                    Predicted
                fail  review  pass
Actual  fail      5     20      2
        review    3     18      5
        pass      1     25     21
```

- Judge over-uses "review" verdict (63% of predictions)
- Lacks confidence to commit to fail/pass
- **Risk:** Too many human review cases, defeats automation purpose

### Pattern 5: **Random Scatter** ❌ Broken Judge

```
                    Predicted
                fail  review  pass
Actual  fail      9      8     10
        review    8      9      9
        pass     10      8     29
```

- No clear pattern
- Weak diagonal
- Errors spread uniformly
- **Verdict:** Judge is no better than random guessing

---

## Themis Implementation Details

### Data Format

**Input: Human-annotated JSONL**

```jsonl
{"event_id": "1", "query": "How do I reset password?", "answer": "Click forgot password", "human_annotation": "pass"}
{"event_id": "2", "query": "Refund policy?", "answer": "No refunds", "human_annotation": "fail"}
{"event_id": "3", "query": "Shipping time?", "answer": "3-5 days usually", "human_annotation": "review"}
```

### Building Confusion Matrix in Go

```go
// internal/metrics/confusion_matrix.go

type Label string

const (
    LabelFail   Label = "fail"
    LabelReview Label = "review"
    LabelPass   Label = "pass"
)

type ConfusionMatrix struct {
    Matrix map[Label]map[Label]int // actual -> predicted -> count
    Labels []Label                   // ordered: [fail, review, pass]
}

// Build creates confusion matrix from parallel slices
func Build(actual, predicted []Label) (*ConfusionMatrix, error) {
    if len(actual) != len(predicted) {
        return nil, fmt.Errorf("mismatched lengths")
    }

    cm := &ConfusionMatrix{
        Matrix: make(map[Label]map[Label]int),
        Labels: []Label{LabelFail, LabelReview, LabelPass},
    }

    // Initialize nested map
    for _, actualLabel := range cm.Labels {
        cm.Matrix[actualLabel] = make(map[Label]int)
        for _, predictedLabel := range cm.Labels {
            cm.Matrix[actualLabel][predictedLabel] = 0
        }
    }

    // Count pairs
    for i := range actual {
        cm.Matrix[actual[i]][predicted[i]]++
    }

    return cm, nil
}

// Get returns count for actual->predicted pair
func (cm *ConfusionMatrix) Get(actual, predicted Label) int {
    return cm.Matrix[actual][predicted]
}

// TotalActual returns row sum (support for this class)
func (cm *ConfusionMatrix) TotalActual(label Label) int {
    sum := 0
    for _, predicted := range cm.Labels {
        sum += cm.Matrix[label][predicted]
    }
    return sum
}

// TotalPredicted returns column sum
func (cm *ConfusionMatrix) TotalPredicted(label Label) int {
    sum := 0
    for _, actual := range cm.Labels {
        sum += cm.Matrix[actual][label]
    }
    return sum
}

// TotalCorrect returns diagonal sum
func (cm *ConfusionMatrix) TotalCorrect() int {
    sum := 0
    for _, label := range cm.Labels {
        sum += cm.Matrix[label][label]
    }
    return sum
}

// TotalSamples returns total count
func (cm *ConfusionMatrix) TotalSamples() int {
    sum := 0
    for _, actual := range cm.Labels {
        sum += cm.TotalActual(actual)
    }
    return sum
}
```

### Computing Per-Class Metrics

```go
// ClassMetrics holds precision, recall, F1 for one class
type ClassMetrics struct {
    Precision float64 `json:"precision"`
    Recall    float64 `json:"recall"`
    F1Score   float64 `json:"f1"`
    Support   int     `json:"support"` // number of actual instances
}

// ComputeClassMetrics calculates precision/recall/F1 per class
func (cm *ConfusionMatrix) ComputeClassMetrics() map[Label]ClassMetrics {
    metrics := make(map[Label]ClassMetrics)

    for _, label := range cm.Labels {
        // True Positives: diagonal cell
        tp := cm.Get(label, label)

        // False Positives: column sum - TP
        fp := cm.TotalPredicted(label) - tp

        // False Negatives: row sum - TP
        fn := cm.TotalActual(label) - tp

        // Precision = TP / (TP + FP)
        var precision float64
        if tp+fp > 0 {
            precision = float64(tp) / float64(tp+fp)
        }

        // Recall = TP / (TP + FN)
        var recall float64
        if tp+fn > 0 {
            recall = float64(tp) / float64(tp+fn)
        }

        // F1 = 2 * (P * R) / (P + R)
        var f1 float64
        if precision+recall > 0 {
            f1 = 2 * (precision * recall) / (precision + recall)
        }

        metrics[label] = ClassMetrics{
            Precision: precision,
            Recall:    recall,
            F1Score:   f1,
            Support:   cm.TotalActual(label),
        }
    }

    return metrics
}
```

### Pretty Printing for CLI

```go
// String returns formatted table for terminal display
func (cm *ConfusionMatrix) String() string {
    var sb strings.Builder

    // Header
    sb.WriteString("                    Predicted\n")
    sb.WriteString("                ")
    for _, label := range cm.Labels {
        sb.WriteString(fmt.Sprintf("%-8s", label))
    }
    sb.WriteString("| Total\n")

    // Rows
    for i, actualLabel := range cm.Labels {
        if i == 0 {
            sb.WriteString("Actual  ")
        } else {
            sb.WriteString("        ")
        }
        sb.WriteString(fmt.Sprintf("%-8s", actualLabel))

        // Cells
        for _, predictedLabel := range cm.Labels {
            count := cm.Get(actualLabel, predictedLabel)
            sb.WriteString(fmt.Sprintf("%-8d", count))
        }

        // Row total
        rowTotal := cm.TotalActual(actualLabel)
        sb.WriteString(fmt.Sprintf("| %d\n", rowTotal))
    }

    // Separator
    sb.WriteString("        ")
    sb.WriteString(strings.Repeat("-", 8*len(cm.Labels)+8))
    sb.WriteString("\n")

    // Column totals
    sb.WriteString("        Total   ")
    for _, label := range cm.Labels {
        colTotal := cm.TotalPredicted(label)
        sb.WriteString(fmt.Sprintf("%-8d", colTotal))
    }
    total := cm.TotalSamples()
    sb.WriteString(fmt.Sprintf("| %d\n", total))

    return sb.String()
}
```

### CLI Output Example

```bash
themis validate --input validation_set.jsonl

=== Validation Report ===
Total Records: 100
Status: ✓ PASSED

Confusion Matrix:
                    Predicted
                fail    review  pass    | Total
Actual  fail    20      5       2       | 27
        review  3       15      8       | 26
        pass    1       6       40      | 47
        ------------------------------------------------
        Total   24      26      50      | 100

Per-Class Performance:
                Precision  Recall    F1      Support
  fail          0.833      0.741    0.785      27
  review        0.577      0.577    0.577      26
  pass          0.800      0.851    0.825      47

Interpretation:
  ✓ Strong diagonal (75% accuracy)
  ⚠️ Review class weak (58% F1) - judge struggles with borderline cases
  ✓ Critical errors minimal (only 2 fail→pass)
  ✓ Judge is production-ready
```

---

## Using Confusion Matrix for Debugging

### Scenario 1: **High False Negatives for "fail"**

**Observation:**
```
                    Predicted
                fail  review  pass
Actual  fail     10     10      7
```

Out of 27 failures, judge only caught 10 (37% recall).

**Root Cause Analysis:**
1. Check the 17 missed failures (10 fail→review + 7 fail→pass)
2. Look for patterns in missed cases:
   - Are they short responses? → Adjust length precheck
   - Are they polite but wrong? → Emphasize correctness in prompt
   - Are they ambiguous edge cases? → Add examples to judge prompt

**Action:**
```bash
# Extract false negatives for manual review
jq 'select(.human_annotation == "fail" and .judge_verdict != "fail")' \
   validation_results.jsonl > false_negatives.jsonl

# Analyze patterns
cat false_negatives.jsonl | jq -r '.answer' | head -10
```

### Scenario 2: **High False Positives for "pass"**

**Observation:**
```
                    Predicted
                pass
Actual  fail      7
        review   15
                ---
        FP:      22
```

22 out of 50 "pass" predictions were wrong (44% false positive rate).

**Root Cause Analysis:**
1. Judge is too lenient (optimistic bias)
2. Check the 22 false positives:
   - Are they borderline cases? → Tighten pass threshold
   - Do they lack key information? → Add completeness weight
   - Are they technically correct but unhelpful? → Emphasize helpfulness

**Action:**
```yaml
# Adjust aggregator config
VERDICT_PASS_THRESHOLD=0.85  # Increase from 0.80 (more strict)
```

### Scenario 3: **"review" Class Overused**

**Observation:**
```
                    Predicted
                fail  review  pass
        Total    10     60     30
```

Judge predicts "review" 60% of the time (model is uncertain).

**Root Cause:**
- Judge confidence is low across the board
- Thresholds might be too narrow (0.5-0.8 review range too wide)

**Action:**
```yaml
# Widen pass threshold, narrow review range
VERDICT_PASS_THRESHOLD=0.75      # More cases pass
VERDICT_REVIEW_THRESHOLD=0.35    # Fewer cases stuck in review
```

---

## Binary Confusion Matrix (Simplified View)

For simpler analysis, collapse to **binary classification** (fail vs pass):

### Collapsing Strategy

**Option 1: Merge fail+review → negative class**
```
fail + review → "negative" (needs work)
pass → "positive" (good quality)
```

**Option 2: Merge review+pass → positive class**
```
fail → "negative" (definite failure)
review + pass → "positive" (acceptable or better)
```

### Example: Option 1 (fail+review vs pass)

**Original 3x3:**
```
                    Predicted
                fail  review  pass  | Total
Actual  fail     20      5      2   |  27
        review    3     15      8   |  26
        pass      1      6     40   |  47
```

**Collapsed 2x2:**
```
                    Predicted
                negative  positive  | Total
Actual  negative    43        10    |  53
        positive     7        40    |  47
        -------------------------------
        Total       50        50    | 100
```

**Metrics:**
```
Accuracy: (43+40)/100 = 83%
Precision(positive): 40/(40+10) = 80%
Recall(positive): 40/(40+7) = 85%
```

**Benefit:** Simpler interpretation, often higher agreement metrics.

---

## Confusion Matrix + Kendall's Tau: Complete Picture

Use **both metrics together** for comprehensive validation:

| Metric | What It Measures | When It's Critical |
|--------|------------------|--------------------|
| **Kendall's τ** | Rank correlation | "Do scores correlate overall?" |
| **Confusion Matrix** | Exact classification errors | "Where exactly does judge fail?" |
| **Cohen's Kappa** | Categorical agreement | "How much better than random?" |

### Example: Complementary Insights

**Validation results:**
```
Kendall's τ: 0.45 (moderate)
Accuracy: 75%
Cohen's κ: 0.62 (substantial)

Confusion Matrix shows:
  - Only 2 critical errors (fail→pass)
  - Review class weak (58% F1)
  - Pass class strong (85% recall)
```

**Interpretation:**
- τ is moderate due to many ties (review class ambiguity)
- κ is substantial (accounting for chance agreement)
- Confusion matrix reveals judge is safe (few fail→pass errors)
- Actionable: Improve review class detection, but judge is production-ready

---

## Complete Validation Workflow

```
1. Run Evaluation on Annotated Dataset
   ↓
   themis validate --input human_annotated.jsonl
   ↓
2. Check Kendall's τ ≥ 0.3 (rank correlation)
   ↓
3. Analyze Confusion Matrix
   ├─ Diagonal strong? → Good judge
   ├─ Upper triangle heavy? → Too lenient
   ├─ Lower triangle heavy? → Too harsh
   └─ Scattered? → Broken judge
   ↓
4. Check Per-Class Metrics
   ├─ Precision(fail) low? → Too many false alarms
   ├─ Recall(fail) low? → Missing real failures ⚠️
   └─ F1(review) low? → Struggles with borderline cases
   ↓
5. Extract Error Cases for Analysis
   ↓
   jq 'select(.human_annotation != .judge_verdict)' results.jsonl
   ↓
6. Iterate: Adjust prompts, thresholds, weights
   ↓
7. Re-validate until metrics acceptable
```

---

## Key Takeaways

1. **Confusion Matrix is diagnostic** - shows exactly where judge fails
2. **Kendall's τ is validation** - confirms overall correlation
3. **Use both together** - τ for threshold check, matrix for debugging
4. **Watch the diagonal** - strong diagonal = reliable judge
5. **Check critical errors** - fail→pass errors are most dangerous
6. **Per-class metrics matter** - weak review class is common and acceptable
7. **Binary view simplifies** - use for stakeholder communication
8. **Pattern recognition helps** - upper triangle = lenient, lower = harsh
9. **Iterate with insights** - confusion matrix guides prompt improvements
10. **Production readiness** - few critical errors + τ ≥ 0.3 = deploy

---

## Next Steps

1. **Implement confusion matrix builder** in `internal/metrics/`
2. **Add per-class metrics calculation** (precision, recall, F1)
3. **Integrate with validation pipeline** in `internal/batch/validator.go`
4. **Add pretty-printing** for CLI output
5. **Test with sample dataset** (150 annotations)
6. **Document common patterns** and their interpretations
7. **Use for judge tuning** - identify systematic errors and fix prompts

The confusion matrix will become your primary debugging tool for understanding and improving LLM judge behavior.
