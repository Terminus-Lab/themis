# Validation Mode Specification

**Purpose**: Validate LLM judge accuracy against human annotations using Kendall's correlation.

**Scope**: Batch CLI only (not API/streaming)

---

## Feature Overview

### Problem
- LLM judges give scores, but how do we know they're accurate?
- Need to validate against human judgment before trusting in production

### Solution
- Collect human annotations on a sample (e.g., 25 records)
- Run LLM judges on same sample
- Compare rankings using Kendall's tau (τ)
- Gate: τ < 0.3 → reject, τ ≥ 0.3 → accept

### Value
- **Confidence**: Know your judges are validated
- **Quality**: Catch bad judge prompts before production
- **Iterative**: Test prompt improvements quickly

---

## Data Format

### Input JSONL (with human annotations)

```jsonl
{
  "event_id": "eval-001",
  "event_type": "agent_response",
  "agent": {"name": "my-agent", "type": "rag", "version": "1.0"},
  "interaction": {
    "user_query": "What is the capital of France?",
    "context": "France is a country in Europe...",
    "answer": "The capital of France is Paris."
  },
  "human_annotation": "pass"
}
```

**Required**:
- `human_annotation` field must be present
- Valid values: `"pass"`, `"review"`, `"fail"` (matches verdict values)

**Note**: Records without `human_annotation` are skipped in validation mode

---

## CLI Interface

### New Flags

```
-validate
    Enable validation mode (compute correlation with human annotations)
    Default: false

-correlation-threshold float
    Minimum Kendall's tau to pass validation
    Default: 0.3
    Range: 0.0 to 1.0
```

### Usage

**Basic validation**:
```bash
go run cmd/batch/main.go \
  -input annotated_sample.jsonl \
  -validate
```

**Custom threshold**:
```bash
go run cmd/batch/main.go \
  -input annotated_sample.jsonl \
  -validate \
  -correlation-threshold 0.5
```

**With output** (optional):
```bash
go run cmd/batch/main.go \
  -input annotated_sample.jsonl \
  -validate \
  -output validation_results.jsonl
```

---

## Behavior

### Validation Mode Flow

1. **Input Validation**
   - Check all records have `human_annotation` field
   - Exit with error if any missing

2. **LLM Evaluation**
   - Run full pipeline on all records
   - Get `verdict` for each (pass/review/fail)

3. **Correlation Computation**
   - Convert labels to ranks:
     - `pass` = 2
     - `review` = 1
     - `fail` = 0
   - Compute Kendall's tau between human ranks and LLM ranks

4. **Gate Decision**
   - If τ < threshold: Exit with error, suggest prompt improvements
   - If τ ≥ threshold: Success, print validation report

### Output Modes

**Console output** (always):
```
INFO Validation mode enabled
INFO Evaluating 25 records with human annotations...
INFO Evaluation complete duration=12.3s
INFO Computing Kendall's correlation...

┌──────────────────────────────────────────┐
│ VALIDATION RESULTS                       │
├──────────────────────────────────────────┤
│ Records evaluated: 25                    │
│ Agreement:         19 / 25 (76%)        │
│ Kendall's τ:       0.42                 │
│ Threshold:         0.3                   │
│ Status:            ✅ PASSED             │
│ Interpretation:    Moderate agreement    │
├──────────────────────────────────────────┤
│ Confusion Matrix:                        │
│                Human                     │
│          pass  review  fail              │
│   pass    12      2     0                │
│   review   1      3     1    LLM         │
│   fail     0      2     4                │
└──────────────────────────────────────────┘

✅ LLM judge validated against human annotations
→ Safe to evaluate full dataset with these judge prompts
```

**File output** (if `-output` specified):
```jsonl
{"event_id":"1","human_annotation":"pass","llm_verdict":"pass","confidence":0.92,"agreement":true}
{"event_id":"2","human_annotation":"fail","llm_verdict":"fail","confidence":0.15,"agreement":true}
{"event_id":"3","human_annotation":"pass","llm_verdict":"review","confidence":0.67,"agreement":false}
```

Plus summary file: `{output}.validation-summary.json`
```json
{
  "total_records": 25,
  "agreement_count": 19,
  "agreement_rate": 0.76,
  "kendall_tau": 0.42,
  "threshold": 0.3,
  "passed": true,
  "confusion_matrix": {
    "pass_pass": 12,
    "pass_review": 2,
    "pass_fail": 0,
    "review_pass": 1,
    "review_review": 3,
    "review_fail": 1,
    "fail_pass": 0,
    "fail_review": 2,
    "fail_fail": 4
  },
  "interpretation": "Moderate agreement with human annotations"
}
```

### Error Cases

**Missing human annotations**:
```
ERROR Validation mode requires all records to have 'human_annotation' field
ERROR Record at line 15 is missing human_annotation
ERROR Found 3 records without annotations
→ Fix input file and re-run
```

**Correlation below threshold**:
```
ERROR Kendall's τ (0.18) below threshold (0.3)
ERROR LLM judge does not match human judgment sufficiently

→ Next steps:
  1. Review configs/judges.yaml prompts
  2. Common issues:
     - Judge prompt too vague
     - Missing evaluation criteria
     - Wrong scoring rubric
  3. Improve prompts and re-run validation
  4. Consider collecting more diverse annotations
```

---

## Kendall's Tau Computation

### Algorithm

```go
func ComputeKendallTau(pairs []AnnotationPair) float64 {
    // pairs: [{human: "pass", llm: "pass"}, ...]

    // 1. Convert to ranks
    ranks := make([]RankPair, len(pairs))
    for i, p := range pairs {
        ranks[i] = RankPair{
            human: verdictToRank(p.human),
            llm:   verdictToRank(p.llm),
        }
    }

    // 2. Count concordant and discordant pairs
    concordant := 0
    discordant := 0

    for i := 0; i < len(ranks); i++ {
        for j := i + 1; j < len(ranks); j++ {
            humanDiff := ranks[i].human - ranks[j].human
            llmDiff := ranks[i].llm - ranks[j].llm

            if humanDiff*llmDiff > 0 {
                concordant++  // Same direction
            } else if humanDiff*llmDiff < 0 {
                discordant++  // Opposite direction
            }
            // If diff == 0, it's a tie, don't count
        }
    }

    // 3. Compute tau
    totalPairs := len(ranks) * (len(ranks) - 1) / 2
    tau := float64(concordant-discordant) / float64(totalPairs)

    return tau
}

func verdictToRank(verdict string) int {
    switch verdict {
    case "pass":
        return 2
    case "review":
        return 1
    case "fail":
        return 0
    default:
        return -1  // invalid
    }
}
```

### Example Calculation

```
5 records:

ID | Human   | LLM     | Human Rank | LLM Rank
---|---------|---------|------------|----------
1  | pass    | pass    | 2          | 2
2  | pass    | review  | 2          | 1
3  | review  | review  | 1          | 1
4  | fail    | fail    | 0          | 0
5  | fail    | review  | 0          | 1

Compare all pairs:
(1,2): human 2>2 (tie), llm 2>1 → tie (don't count)
(1,3): human 2>1, llm 2>1 → concordant ✓
(1,4): human 2>0, llm 2>0 → concordant ✓
(1,5): human 2>0, llm 2>1 → concordant ✓
(2,3): human 2>1, llm 1>1 (tie) → tie (don't count)
(2,4): human 2>0, llm 1>0 → concordant ✓
(2,5): human 2>0, llm 1>1 (tie) → tie (don't count)
(3,4): human 1>0, llm 1>0 → concordant ✓
(3,5): human 1>0, llm 1>1 (tie) → tie (don't count)
(4,5): human 0>0 (tie), llm 0<1 → tie (don't count)

Concordant: 5
Discordant: 0
Total pairs: 10

τ = (5 - 0) / 10 = 0.5 ✅
```

---

## Implementation Files

### New Files

**internal/batch/validator.go**
- `type AnnotationPair struct { human, llm string }`
- `func ComputeKendallTau(pairs []AnnotationPair) float64`
- `func GenerateConfusionMatrix(pairs []AnnotationPair) map[string]int`
- `func InterpretTau(tau float64) string`

**internal/batch/validator_test.go**
- Test cases for Kendall computation
- Test confusion matrix generation
- Test interpretation logic

### Modified Files

**models/evaluation.go**
- Add `HumanAnnotation *string` to `EvaluationRequest`

**cmd/batch/main.go**
- Add `-validate` and `-correlation-threshold` flags
- Add validation mode logic
- Call validator functions
- Print validation report
- Exit with error if tau < threshold

---

## Acceptance Criteria

- [ ] `-validate` flag enables validation mode
- [ ] Records without `human_annotation` are rejected in validation mode
- [ ] Kendall's tau is computed correctly
- [ ] Confusion matrix is displayed
- [ ] Exit code 1 if tau < threshold
- [ ] Exit code 0 if tau ≥ threshold
- [ ] Validation summary written to file (if `-output` specified)
- [ ] Clear error messages for common issues
- [ ] Unit tests for Kendall computation
- [ ] Documentation in batch_evaluation_test_cases.md

---

## Testing

### Test Case 1: Perfect Agreement

**Input**: 10 records where `human_annotation` == `llm_verdict`

**Expected**: τ = 1.0, all pass

### Test Case 2: Random Agreement

**Input**: 20 records with random human/LLM labels

**Expected**: τ ≈ 0.0 to 0.3

### Test Case 3: Below Threshold

**Input**: 15 records with τ = 0.18

**Expected**: Error exit, suggest prompt improvements

### Test Case 4: Missing Annotations

**Input**: 10 records, 3 missing `human_annotation`

**Expected**: Error exit, list missing records

---

## Future Enhancements (Out of Scope)

1. **Semantic Similarity Check** (Step 5 from llm_as_judge.md)
   - Embed queries
   - Compare sample vs full dataset
   - Gate if similarity < 0.8

2. **Per-Judge Correlation**
   - Compute τ for each judge individually
   - Identify which judges need improvement

3. **Iterative Mode**
   - Quick re-run after prompt changes
   - Track τ over iterations

4. **Sample Generation**
   - Auto-sample 25% from full dataset
   - Export for human annotation

---

## Effort Estimate

**Complexity**: Medium

**Effort**: 4-6 hours

**Breakdown**:
- Kendall computation: 1.5h
- CLI integration: 1h
- Validation logic: 1h
- Testing: 1h
- Documentation: 0.5h

**Dependencies**: None (all batch-local changes)
