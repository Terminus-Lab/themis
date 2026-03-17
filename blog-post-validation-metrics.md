# Build First, Measure Comprehensively

*This is a continuation of [AI Agent Evaluation: Build First, Decide Later](https://vladpovarna.substack.com/p/ai-agent-evaluation-build-first-decide)*

## The Score Interpretation Problem

You've built judges, tuned prompts, collected 150 human annotations, and computed Kendall's τ. The result: 0.4.

Is that good? Should you deploy? The number alone tells you correlation exists, but not what your judges actually do wrong.

This is the interpretation gap: validation metrics produce numbers, but numbers without context don't guide action.

## Three Metrics, Three Perspectives

A single metric can pass validation while judges fail in practice. Kendall's τ measures rank correlation—whether judges order responses the same way humans do. A τ of 0.4 means moderate agreement, but it doesn't tell you which classes the judge misclassifies, whether errors are concentrated in one category, or what proportion of critical errors (fail→pass) occur. A judge with τ = 0.4 could be safe but imprecise, or dangerously biased with acceptable overall correlation.

Three metrics together provide what one cannot:

**Kendall's τ** answers: do judges rank quality the same way humans do? It's robust to ordinal data with ties—the natural structure of pass/review/fail verdicts.

**Cohen's Kappa** answers: how much better is the judge than random guessing with the same class distribution? It catches judges that exploit class imbalance (always predicting "pass" on an 85% pass dataset achieves 85% accuracy; Kappa exposes this as zero real agreement).

**Confusion Matrix** answers: where specifically does the judge fail? It's the diagnostic layer—τ and Kappa tell you something is wrong; the matrix tells you what.

Together: τ validates correlation. Kappa quantifies agreement beyond chance. The matrix reveals what to fix.

## A Validation Example

Here's output from 150 annotated samples, balanced 50 per class:

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
    "fail":   {"fail": 49, "pass": 0, "review": 1},
    "pass":   {"fail": 0,  "pass": 50, "review": 0},
    "review": {"fail": 0,  "pass": 8,  "review": 42}
  },
  "per_class_metrics": {
    "fail":   {"precision": 1.00, "recall": 0.98, "f1": 0.99, "support": 50},
    "pass":   {"precision": 0.86, "recall": 1.00, "f1": 0.93, "support": 50},
    "review": {"precision": 0.98, "recall": 0.84, "f1": 0.90, "support": 50}
  }
}
```

The diagonal is strong. Zero critical errors: no failures classified as passes, no passes classified as failures. The judge caught 98% of actual failures and never incorrectly cleared a bad response. Review class handling is weaker—eight review cases were classified as pass, borderline responses where the judge was optimistic. These surface for human review rather than silently approving bad answers.

Kappa at 0.91 confirms agreement far beyond chance. With balanced classes, random guessing would yield 33% accuracy; the judge achieves 94%. τ at 0.63 is lower than Kappa—this is expected. Categorical agreement can exceed rank correlation when exact matches are high but confidence scores near the thresholds vary, producing ties that τ penalizes conservatively.

**What this doesn't show:** The validation set is balanced (50/50/50). Real traffic might be 85% pass, 10% review, 5% fail. Judge behavior under class imbalance is unknown. These metrics don't explain *why* errors occur—the eight review→pass cases need manual inspection. Are they short answers? Domain-specific ambiguity? Formatting edge cases? If those eight cases share a pattern, the judge will fail consistently once real traffic arrives.

## Per-Class Metrics and What to Optimize

Different error types have different costs. Per-class metrics make those costs explicit.

**Precision** answers: when the judge predicts this class, how often is it correct? For the fail class, precision 1.0 means every response labeled "fail" was actually a failure—no good responses incorrectly rejected.

**Recall** answers: of all actual instances of this class, how many did the judge catch? For the fail class, recall 0.98 means the judge caught 98% of actual failures, missing only 2%.

**F1** is the harmonic mean of both. It's the right summary when you can't sacrifice one for the other.

The priorities differ by class:

**Fail class: optimize recall first.** Missing a failure means a bad response gets approved—silent harm. A false alarm means a good response gets flagged for human review—safe, recoverable. Be conservative. Target recall ≥ 0.95, then tune precision to reduce review load.

**Pass class: optimize recall, then precision.** Rejecting good responses frustrates users and erodes trust in the agent faster than extra review cases. Target recall ≥ 0.90, precision ≥ 0.85.

**Review class: balance with F1.** Review is inherently ambiguous. Borderline cases can reasonably go either way. F1 ≥ 0.75 is a reasonable target; higher is not always achievable without forcing the judge to commit on genuinely uncertain inputs.

## Why Not More Metrics?

Several standard metrics were candidates:

| Metric | Why Not |
|--------|---------|
| **Spearman's ρ** | Similar to Kendall's τ but less robust for ordinal data with ties. Kendall's τ is the right choice here. |
| **Pearson's r** | Assumes linear relationships. Evaluation scores rarely have this property. |
| **Matthews Correlation** | Designed for binary classification; multi-class extensions exist but are less interpretable than the combination already chosen. |
| **ROC AUC** | Most useful when thresholds are tunable post-deployment. Verdict thresholds here are fixed in configuration. |
| **Expected Calibration Error** | Measures whether confidence scores match actual probabilities. Valuable but adds interpretation complexity without changing the core deployment decision. |

Three metrics provide complementary perspectives without layering interpretation complexity. The goal is actionable diagnosis, not exhaustive measurement.

---

## Getting Judges Production-Ready

Static validation is the starting point, not a certificate. Before deploying, you iterate.

Start with 150–200 samples covering diverse query types, answer qualities, and edge cases. Annotate manually with explicit criteria—what makes a response a failure? Where is the pass/review boundary? Inconsistent annotation criteria produce unreliable correlation signals regardless of judge quality.

Run the judges on the same subset. Compute τ, Kappa, and the confusion matrix. The confusion matrix tells you what to tune:

**High fail→pass rate.** Judges are optimistic. Sharpen the prompt: add explicit failure criteria, include examples that should fail, raise the pass confidence threshold.

**High pass→fail rate.** Judges are pessimistic. Add explicit pass criteria, lower the fail threshold.

**Review class scattered.** The borderline category is unclear. Add boundary examples and be explicit about what makes a response ambiguous versus clearly good or bad.

Edit prompts in `judges.yaml`, re-run validation, check whether τ improved and the confusion matrix diagonal strengthened. This loop typically converges in three to five iterations.

Deploy when: τ ≥ 0.3, fail→pass cells near zero, fail recall ≥ 0.90.

---

## Production: Monitoring Verdict Distribution

After deployment, the application receives conversation turns—events arriving via API or streaming. Each is evaluated by the judges and written to the database.

Initially, you have no ground truth. You only have judge verdicts.

The first available signal is verdict distribution over time: what fraction of evaluations result in pass, review, fail? If your pre-deployment validation set showed 60% pass / 25% review / 15% fail and early production traffic shows similar proportions, the judges are likely working as expected. Sustained drift in these ratios is the earliest automated indicator that something changed.

A sustained increase in fail rate over days or weeks could mean agent quality degraded, a deployment introduced regressions, or a shift in user population exposed edge cases the judges don't handle well. Spikes—sharp increases within hours—typically indicate a deployment event and resolve when the event does. Gradual drift is harder to read: query distribution may be shifting, or agent behavior may be slowly changing. Both look identical in the verdict ratio chart.

**What verdict drift does not tell you:** whether the judges or the agent caused the change. A drop in pass rate has two equally valid explanations: the judges are no longer calibrated to current response patterns, or the agent is producing worse answers. Verdict ratios alone cannot distinguish between these.

## The Limits of Cross-Judge Disagreement

The instinct when monitoring is to look at disagreement between judges—if judges diverge on a sample, treat it as a signal.

The problem is that judges measure fundamentally different things. A faithfulness judge evaluates whether the answer is grounded in retrieved context. A correctness judge evaluates whether the answer matches an expected output. These dimensions don't move together. High disagreement between them is expected for certain response types—it's not a sign of drift.

Standard deviation across judge scores conflates two phenomena: genuine quality ambiguity (the response is borderline on multiple dimensions) and category mismatch (judges are measuring different things and naturally diverge on some inputs). The weights in `judges.yaml` already encode the understanding that judges are not equivalent. Treating score variance across unlike judges as a diagnostic signal ignores that.

Cross-judge disagreement is meaningful within a single quality dimension—multiple relevance judges or multiple faithfulness judges disagreeing signals something real. Across unlike judges measuring different axes, it's noise.

## When Drift Occurs: The Resolution Path

Regardless of which automated signal triggered the alert—verdict ratio, confidence distribution—the resolution path is the same: human evaluation.

Sample from the drift period. Weight the sample toward the verdict categories that drifted. Have annotators label it. Compute τ and the confusion matrix against the new annotations.

If τ dropped significantly compared to your pre-deployment baseline, the judges drifted. Check per-class metrics against the original validation run to identify which judge failed. Fix the prompts, re-validate, redeploy the configuration.

If τ is stable, the judges are working correctly. The agent changed. Cross-reference with deployment logs. The performance shift is real and the judges are surfacing it accurately.

**Query-answer similarity** adds one diagnostic signal here. Compute embedding similarity between production queries and your original validation set queries. If similarity is low, production inputs have drifted far from the distribution the judges were validated on. The judges may not generalize. The fix is not recalibration—it's expanding validation coverage with annotations for the new query types.

This distinction matters: judge calibration failure and out-of-distribution failure look similar in verdict ratios but require different responses.

## The Annotation Bottleneck

Every monitoring path leads back to human annotation. Verdict drift requires annotation to distinguish judge failure from agent failure. Out-of-distribution inputs require annotation to expand validation coverage. Prompt tuning requires annotation to measure improvement.

Human annotation is expensive and slow. It doesn't scale with traffic volume. This is the fundamental constraint. You can defer the cost, but you cannot eliminate it. The practical question is how to make each annotation round targeted: label the cases that resolve the current ambiguity, not a random sample.

## Can You Get Annotation Without Humans?

Worth asking directly.

**Implicit conversation signals.** In multi-turn conversations, user behavior carries evaluation signal. Did the user ask the same question again? Did they rephrase? Did they explicitly correct the agent? These patterns suggest the previous answer was inadequate without requiring an explicit label. The signal is noisy—users rephrase for reasons unrelated to answer quality—but it accumulates continuously with traffic.

**Explicit in-band feedback.** After an agent response, prompt the user: "Did this answer your question?" The binary response becomes an annotation. Over time, this produces a ground-truth signal without separate annotation sessions.

Neither replaces human annotation fully. Implicit signals require heuristics that may not generalize across agent types. Explicit feedback has selection bias—users who respond differ systematically from users who don't.

But they change the economics. Instead of periodic annotation sprints triggered by drift alerts, you accumulate signal continuously. The labeled dataset grows with traffic. When drift occurs, you have more labeled data to work with before escalating to a full annotation round. The tradeoff is instrumentation cost and conversation friction—an extra step after every agent response. For high-volume systems where annotation costs compound quickly, the exchange is worth considering.

## Continuous Validation Is an Operational System

Static validation—run once before deployment—answers one question: are judges calibrated on this dataset?

Continuous validation—drift monitoring, periodic re-annotation, similarity tracking, feedback collection—answers a harder question: do judges remain calibrated as the world changes?

The implementation gap is significant. Static validation is a CI check. Continuous validation is an operational system: instrumented conversation flows, scheduled annotation jobs, correlation tracking over time, alerting on drift thresholds.

The initial validation run is the first data point in an ongoing measurement process, not a certification that expires at deployment. The three metrics—Kendall's τ, Cohen's Kappa, confusion matrix—remain the tools at every stage. What changes is the frequency of measurement and the source of annotation data.

I don't have a fully automated solution to offer here, and I'm skeptical of anyone who claims they do. On every resolution path—judge drift, agent degradation, out-of-distribution inputs—you eventually reach the same bottleneck: a human looking at sampled responses and making judgments. The tooling I've built around Themis is an attempt to make that work as targeted and infrequent as possible, not to eliminate it.

That's the honest constraint. Build the system knowing it requires occasional human input, design the feedback loops to minimize how often you need it, and don't confuse low annotation frequency with full automation.

Build first. Measure comprehensively. Monitor continuously.

---

*Validation metrics and monitoring implementation available in Themis: github.com/Terminus-Lab/themis*
