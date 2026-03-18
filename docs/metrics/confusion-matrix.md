# Confusion Matrix

**Role**: Diagnostic tool — shows exactly where and how the judge fails.

Kendall's τ tells you *if* rankings correlate. The confusion matrix tells you *what kind of errors* the judge makes.

---

## Structure

```
                    Predicted
                fail  review  pass  | Total
Actual  fail     20      5      2   |  27
        review    3     15      8   |  26
        pass      1      6     40   |  47
        --------------------------------
        Total    24     26     50   | 100
```

- **Diagonal** (20, 15, 40): Correct predictions
- **Off-diagonal**: Errors — the pattern matters

---

## Per-Class Metrics

```
Precision = TP / (TP + FP)   "Of all predicted as X, how many were actually X?"
Recall    = TP / (TP + FN)   "Of all actual X, how many did the judge catch?"
F1        = 2 × (P × R) / (P + R)
```

From the example above:

| Class | Precision | Recall | F1 | Notes |
|-------|-----------|--------|----|-------|
| fail | 0.83 | 0.74 | 0.79 | Catches 74% of real failures |
| review | 0.58 | 0.58 | 0.58 | Weakest — borderline cases are hard |
| pass | 0.80 | 0.85 | 0.83 | Strong pass detection |

---

## Error Severity

Not all errors are equal:

| Error type | Severity | Acceptable rate |
|------------|----------|-----------------|
| fail → pass | **Critical** — bad answer approved | < 4% of fails |
| pass → fail | High — good answer blocked | < 6% of passes |
| fail → review | Medium — surfaces for human review | < 20% of fails |
| review → pass | Medium — too lenient on borderlines | < 16% of reviews |
| review → fail | Low — conservative, safe | < 25% of reviews |
| pass → review | Low — conservative, safe | < 10% of passes |

---

## Common Patterns

### Diagonal dominant ✅ Good judge
```
fail     25   2   0
review    3  20   3
pass      0   2  45
```
Strong diagonal, minimal errors, no systematic bias.

### Upper-right triangle ⚠️ Lenient judge
```
fail     10  10   7
review    2  15   9
pass      0   2  45
```
Many fail→pass and review→pass errors. Bad responses slip through.
**Fix**: Strengthen correctness/completeness judge weights; raise `VERDICT_PASS_THRESHOLD`.

### Lower-left triangle ⚠️ Harsh judge
```
fail     25   2   0
review   10  10   6
pass      8   5  34
```
Many pass→fail and review→fail errors. Good responses rejected.
**Fix**: Update completeness prompt to allow terse-but-correct answers; lower `VERDICT_REVIEW_THRESHOLD`.

### All predictions in one column ❌ Degenerate judge
```
fail      0   0   5
review    0   0  10
pass      0   0  85
```
Judge predicts `pass` for everything. Accuracy is 85% but κ = 0, τ = 0. Useless.

---

## Extracting Error Cases

```bash
# False positives: fail predicted as pass
jq 'select(.human_annotation == "fail" and .judge_verdict == "pass")' \
   validation_results.jsonl > critical_errors.jsonl

# Analyze patterns
jq -r '[.interaction.user_query, .interaction.answer, .confidence] | @csv' \
   critical_errors.jsonl
```

Look for: common topics, response length/tone, confidence scores clustered near verdict thresholds.

---

## Validation Output

```bash
./bin/themis-cli validate-events -i validation_set.jsonl
```

```
Confusion Matrix:
                fail    review  pass    | Total
Actual  fail    20      5       2       | 27
        review  3       15      8       | 26
        pass    1       6       40      | 47
        -----------------------------------
        Total   24      26      50      | 100

Per-Class Performance:
        Precision  Recall    F1      Support
fail    0.833      0.741    0.785      27
review  0.577      0.577    0.577      26
pass    0.800      0.851    0.825      47
```

---

## Next Steps

- [Interpretation Guide](interpretation-guide.md) — decision framework, failure patterns, and threshold tuning
