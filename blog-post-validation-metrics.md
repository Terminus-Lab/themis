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

## The Iteration Loop Before Deployment

Static validation on a representative dataset is the starting point, not a certificate. Before deploying, you iterate. After deploying, you watch.

Start with 150–200 samples that cover diverse query types, answer qualities, and edge cases. Annotate a subset manually. Have domain experts assign pass/review/fail verdicts with explicit criteria—what makes a response a failure? Where is the pass/review boundary? Inconsistent annotation criteria produce unreliable correlation signals regardless of judge quality.

Run the judges on the same subset. Compute Kendall's τ, Cohen's Kappa, and the confusion matrix.

The confusion matrix tells you what to tune:

**High fail→pass rate.** Judges are optimistic, they pass bad responses. Sharpen the prompt: add explicit failure criteria, include examples of responses that should fail, raise the pass confidence threshold.

**High pass→fail rate.** Judges are pessimistic. Add explicit pass criteria, lower the fail threshold, clarify the boundary in the prompt.

**Review class scattered across fail and pass.** The borderline category is unclear to the judge. Add boundary examples. Be explicit about what makes a response ambiguous versus clearly good or bad.

Edit the prompts in `judges.yaml`, re-run validation, check whether τ improved and the confusion matrix diagonal strengthened. This loop typically converges in three to five iterations.

The signal to deploy: τ ≥ 0.3, no critical errors (fail→pass cells near zero), and per-class recall on fail ≥ 0.90. The fail recall threshold matters most. Missing actual failures is more dangerous than flagging borderline cases for human review.

---

## Production: What the System Sees

After deployment, the application receives conversation turns from agents—events arriving via API or streaming. Each event is evaluated by the judges and the result (verdict, confidence, stage scores) is written to the database.

Initially, you have no ground truth. You only have judge verdicts.

The first available signal is verdict distribution over time: what fraction of evaluations result in pass, review, fail? If your pre-deployment validation set showed 60% pass / 25% review / 15% fail and early traffic shows similar proportions, the judges are likely working as expected.

Sustained drift in these ratios is the earliest automated indicator that something changed.

---

## Monitoring Verdict Drift

A sustained increase in fail rate over days or weeks could mean agent quality degraded, a new deployment introduced regressions, or a shift in user population exposed edge cases the judges don't handle well. A sustained decrease could mean the agent improved—or that judges became more lenient after a configuration change.

Spikes—sharp increases in fail rate within hours—typically can indicate a deployment event: a new agent version, a prompt change, an infrastructure failure. The pattern is bounded. When the event resolves, the distribution returns to baseline.

Gradual drift is harder to interpret. A 2% per week increase in fail rate over a month is a real signal but an ambiguous one. Query distribution may be shifting toward domains the judges were not validated on. Agent behavior may be slowly changing. Both look identical in the verdict ratio chart.

**What verdict drift does not tell you**: whether the judges or the agent caused the change. This is the core ambiguity. A drop in pass rate has two equally valid explanations: the judges are no longer calibrated to current response patterns, or the agent is producing worse answers. Verdict ratios alone cannot distinguish between these.

---

## The Limits of Cross-Judge Disagreement

The instinct when monitoring is to look at disagreement between judges. If judges diverge on a sample, treat it as a signal that something unusual is happening.

The problem is that judges measure fundamentally different things. A faithfulness judge evaluates whether the answer is grounded in retrieved context. A correctness judge evaluates whether the answer matches an expected output. These dimensions don't move together. High disagreement between them is expected for certain response types—it's not a sign of drift or failure.

Standard deviation across judge scores conflates two phenomena: genuine quality ambiguity (the response is borderline on multiple dimensions) and category mismatch (judges are measuring different things and naturally diverge on some inputs). The judge weights in the configuration already encode the understanding that judges are not equivalent. Treating score variance across unlike judges as a single diagnostic signal ignores that.

Cross-judge disagreement is meaningful within a single quality dimension, if you run multiple relevance judges or multiple faithfulness judges, disagreement between them signals something real. Across unlike judges measuring different axes, it's noise.

---

## When Drift Occurs, You Still Need Humans

Regardless of which automated signal triggered the alert—verdict ratio, confidence distribution, cross-judge disagreement—the resolution path is the same: human evaluation.

Sample from the drift period. Fetch a subset weighted toward the verdict categories that drifted. Have annotators label it. Compute τ and the confusion matrix against the new annotations.

If τ dropped significantly compared to your pre-deployment baseline, the judges drifted. Check per-class metrics against the original validation run to identify which judge failed. Fix the prompts, re-validate, redeploy the configuration.

If τ is stable, the judges are working correctly. The agent changed. Cross-reference with deployment logs. The performance shift is real and the judges are surfacing it accurately.

**Query-answer similarity** adds one more signal here. Compute embedding similarity between production queries and your original validation set queries. If similarity is low—production queries have drifted far from the distribution the judges were validated on—the judges may not generalize to the new inputs. This doesn't mean the judge is wrong; it means validation coverage was insufficient. Add new annotations for the divergent query types and re-validate.

The practical value of similarity: it tells you whether you're looking at a judge calibration problem or an out-of-distribution problem. Different root cause, different fix.

---

## The Annotation Bottleneck

Every monitoring path leads back to human annotation. Verdict drift requires annotation to distinguish judge failure from agent failure. Out-of-distribution queries require annotation to expand validation coverage. Prompt tuning requires annotation to measure improvement.

Human annotation is expensive and slow. It doesn't scale with traffic volume. This is the fundamental constraint in automated evaluation systems. You can defer the annotation cost, but you cannot eliminate it. The practical question is how to make each annotation round as targeted as possible: label the specific cases that resolve the current ambiguity, not a random sample.

---

## Can You Get Annotation Without Humans?

Worth asking directly.

**Implicit conversation signals.** In multi-turn conversations, user behavior carries evaluation signal. Did the user ask the same question again after receiving an answer? Did they rephrase? Did they explicitly correct the agent? These patterns suggest the previous answer was inadequate without requiring an explicit label. The signal is noisy—users rephrase for reasons unrelated to answer quality—but it accumulates continuously with traffic.

**Explicit in-band feedback.** Add a lightweight feedback step to the conversation: after an agent response, prompt the user with a simple question—"Did this answer your question?" The binary response becomes an annotation. Over time, this produces a ground-truth signal without separate annotation sessions.

Neither approach replaces human annotation fully. Implicit signals require heuristics that may not generalize across agent types. Explicit feedback has selection bias—users who respond differ systematically from users who don't.

But they change the economics. Instead of periodic annotation sprints triggered by drift alerts, you accumulate signal continuously. The labeled dataset grows with traffic. When drift occurs, you have more data to work with before deciding whether to escalate to a full annotation round.

The tradeoff is instrumentation cost and user experience friction. An explicit feedback prompt adds a step to every conversation. Implicit signal extraction requires parsing conversation structure and maintaining heuristics over time. For high-volume systems where annotation costs compound quickly, collecting feedback in-band reduces dependency on scheduled human review cycles.

---

## Continuous Validation Is an Operational System

Static validation—run once before deployment—answers one question: are judges calibrated on this dataset?

Continuous validation—drift monitoring, periodic re-annotation, similarity tracking, feedback collection—answers a harder question: do judges remain calibrated as the world changes?

The implementation gap between these two is significant. Static validation is a CI check that runs in minutes. Continuous validation is an operational system: instrumented conversation flows, scheduled annotation jobs, correlation tracking over time, alerting on drift thresholds.

Building toward continuous validation doesn't mean discarding static validation. The initial validation run is the first data point in an ongoing measurement process, not a certification that expires at deployment. The three metrics—Kendall's τ, Cohen's Kappa, confusion matrix—remain the measurement tools at every stage. What changes is the frequency of measurement and the source of annotation data.

I don't see a fully automated solution. On every resolution path—judge drift, agent degradation, out-of-distribution inputs—you eventually reach the same bottleneck: a human looking at sampled responses and making judgments. The goal of the monitoring system is to make that work as targeted and infrequent as possible.

Build first. Measure comprehensively. Monitor continuously.

---

*Validation metrics and monitoring implementation available in Themis: github.com/Terminus-Lab/themis*
