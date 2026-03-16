# Build First, Measure Comprehensively

*This is a continuation of [AI Agent Evaluation: Build First, Decide Later](https://vladpovarna.substack.com/p/ai-agent-evaluation-build-first-decide)*

## The Score Interpretation Problem

You've built judges, tuned prompts, collected 150 human annotations, and computed Kendall's τ. The result: 0.4.

Is that good? Should you deploy? The number alone tells you correlation exists, but not what your judges actually do wrong.

This is the interpretation gap: validation metrics produce numbers, but numbers without context don't guide action.

## What a Single Correlation Score Hides

Kendall's τ measures rank correlation, whether judges order responses the same way humans do. A τ of 0.4 means moderate agreement. But it doesn't tell you which classes your judge misclassifies (failures as passes? passes as failures?), whether errors are evenly distributed or concentrated in one category, if your judge exploits class imbalance (predicting "pass" for everything), or what proportion of critical errors (fail→pass) occur.

A judge with τ = 0.4 could be safe but imprecise, or dangerously biased with acceptable overall correlation.

## Three Metrics, Three Perspectives

After running evaluation on some datasets, I quickly realized that one metric wasn't enough. After some research, I've added two more: Cohen's Kappa and confusion matrix analysis.

**Kendall's τ** measures rank correlation across all samples. It answers: do judges rank quality the same way humans do?

**Cohen's Kappa** measures categorical agreement beyond chance. It answers: how much better is the judge than random guessing with the same class distribution?

**Confusion Matrix** shows exact error patterns. It answers: where specifically does the judge fail?

Together they provide actionable diagnosis. τ validates correlation. Kappa quantifies agreement accounting for imbalance. The matrix reveals what to fix.

## A Validation Example

Here's output from 150 annotated samples:

```json
{
  "passed": true,
  "total_records": 150,
  "threshold": 0.3,
  "correlation_metrics": {
    "kendalls_tau": 0.63,
    "interpretation": "Moderate to strong agreement",
    "passed_threshold": true
  },
  "agreement_metrics": {
    "cohens_kappa": 0.91,
    "interpretation": "Almost perfect"
  },
  "confusion_matrix": {
    "fail": {"fail": 49, "pass": 0, "review": 1},
    "pass": {"fail": 0, "pass": 50, "review": 0},
    "review": {"fail": 0, "pass": 8, "review": 42}
  },
  "per_class_metrics": {
    "fail": {"precision": 1.0, "recall": 0.98, "f1": 0.99, "support": 50},
    "pass": {"precision": 0.86, "recall": 1.0, "f1": 0.93, "support": 50} ,
    "review": {"precision": 0.98, "recall": 0.84, "f1": 0.90, "support": 50}
  }
}
```

## Why Per-Class Metrics Matter

The overall accuracy of 94% looks great, but it hides class-specific performance. Per-class metrics reveal whether your judge handles all verdict types equally well or struggles with specific categories.

**Precision** answers: when the judge predicts this class, how often is it correct? High precision means few false alarms. For the fail class, precision of 1.0 means every response the judge marked as "fail" was actually a failure, no false rejections of good responses.

**Recall** answers: of all actual instances of this class, how many did the judge catch? High recall means few misses. For the fail class, recall of 0.98 means the judge caught 98% of actual failures, missing only 2%.

**F1 score** balances precision and recall into a single metric. It's the harmonic mean, so both must be high for a good F1 score.

**Support** shows how many samples of each class exist in the validation set. Here it's balanced (50 each), which makes metrics directly comparable.

Why calculate these for each class separately? Different error types have different costs. Missing a failure (low fail recall) is more dangerous than being overly conservative (low fail precision). Perfect pass detection (recall 1.0) ensures you never reject good responses. Weaker review performance (recall 0.84) is acceptable because review is inherently ambiguous.

**Which metric should you optimize for each class?** All three matter, but as a rule, start with these priorities:

**Fail class: Optimize recall first.** Missing actual failures (low recall) means bad responses get approved, users receive wrong information, silent failure. False alarms (low precision) mean good responses get flagged for human review, safe failure mode. Better to be conservative. Target: recall ≥ 0.95, then worry about precision.

**Pass class: Optimize recall, then precision.** High recall means you don't reject good responses. High precision means you don't pass things that need review. Both matter for user experience, but rejecting good responses is more frustrating than extra review cases. Target: recall ≥ 0.90, precision ≥ 0.85.

**Review class: Balance both with F1.** Review is inherently ambiguous. Borderline cases can reasonably go either way. F1 score captures the balance between catching review cases and not over-using the review verdict. Target: F1 ≥ 0.75.

In safety-critical evaluation, recall on the fail class is the most important single metric. Catch the bad responses first. Tune precision later to reduce human review load.

**What this shows:**

The confusion matrix diagonal is strong, most predictions match human annotations. Zero critical errors: no failures classified as passes, no passes classified as failures. The judge caught 98% of actual failures (recall = 0.98) and never incorrectly flagged bad responses as good. Review class handling is weaker (84% recall). Eight review cases were predicted as pass, borderline cases where the judge was optimistic. But these errors surface for human evaluation rather than silently approving garbage.

Kappa at 0.91 confirms the judge agrees far beyond chance. Even with balanced class distribution (50/50/50), the judge achieves 94% accuracy where random guessing would yield 33%.

Kendall's τ at 0.63 shows moderate correlation. Why not higher? The metric is conservative with ordinal data and many ties. Categorical agreement (kappa) can exceed rank correlation (tau) when exact matches are high but confidence scores vary.

**What this doesn't show:**

The validation set is balanced (50/50/50). Real traffic might be 85% pass, 10% review, 5% fail. Judge behavior under class imbalance is unknown. These metrics don't explain *why* errors occur. The eight review→pass cases need manual inspection to identify patterns: are they edge cases? Formatting issues? Domain-specific ambiguity? Correlation and agreement measure past performance on static data. Judge behavior can drift as response patterns change. Continuous validation is required.

## Why Not More Metrics?

These three metrics (Kendall's τ, Cohen's Kappa, confusion matrix) aren't the only validation options. Here are others I considered:

| Metric | What It Measures | Why I Didn't Add It |
|--------|------------------|---------------------|
| **Spearman's ρ** | Rank correlation (parametric) | Similar to Kendall's τ but assumes more about distribution. Kendall's τ is more robust for ordinal data with ties. |
| **Pearson's r** | Linear correlation | Assumes linear relationship between scores. Evaluation scores rarely have this property. |
| **Matthews Correlation** | Balanced binary classification | Only works for binary decisions. I have three classes (fail/review/pass). |
| **ROC AUC** | Trade-off between true/false positive rates | Useful when thresholds are tunable post-deployment. My verdict thresholds are fixed in configuration. |
| **Expected Calibration Error** | Score probability calibration | Measures whether confidence scores match actual probabilities. Valuable but adds complexity. |

I wanted to keep validation simple. Three metrics provide complementary perspectives: correlation (τ), agreement beyond chance (Kappa), and error patterns (confusion matrix). Adding more would make interpretation harder without adding significantly more insight for my use case.

## Why This Matters

Validation is not a pass/fail check. It's diagnosis.

A single metric—τ, kappa, or accuracy—can pass validation while judges fail in practice. The eight review→pass errors in the example might seem minor at 16% error rate. But if those eight cases represent a systematic pattern (short answers, specific topics, formatting edge cases), the judge will fail consistently on that pattern in production.

The combination of rank correlation, categorical agreement, and error breakdown reveals whether judges are safe to deploy and where to focus improvement effort.

Validation without interpretation is just numbers. Interpretation without multiple perspectives is guessing.

## Moving Forward

Validation metrics answer three questions: Does the judge correlate with human judgment? (Kendall's τ), Does it agree beyond random chance? (Cohen's Kappa), and Where exactly does it fail? (Confusion Matrix).

When validation fails, the confusion matrix shows what to fix: rebalance judge weights, adjust verdict thresholds, add training examples for weak classes, or revise prompts to handle edge cases.

When validation passes, the metrics tell you what to monitor: error rates per class, confidence score distributions near thresholds, performance on minority classes.

Build first. Measure comprehensively. Decide based on error patterns, not single numbers.

---

*Validation metrics implementation available in Themis at github.com/Terminus-Lab/themis*
