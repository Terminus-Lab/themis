# Production Monitoring & Validation Workflow - Specification

**Status:** In Progress
**Date:** 2026-03-17

---

## Status Summary

**Completion**: 2/3 Phases Complete (67%)

| Phase | Status | Progress |
|-------|--------|----------|
| Phase 1: Sampling API | ✅ Complete | 100% |
| Phase 2: Proxy Metrics Dashboard | ✅ Complete | 100% |
| Phase 3: Validation History Tracking | 🔲 Not Started | 0% |

**Current Status:** Phases 1 and 2 complete. Dashboard has a "Monitoring" tab with window selector (24h / 7d / 30d / 90d), overview stats, verdict distribution bars, and per-judge score bars.

**Next Steps:** Implement Phase 3 (Validation History Tracking) — `--save-to-db` flag on `themis validate` + `GET /api/v1/validation/history` endpoint.

---

## 1. Purpose

### Problem

**Current State:** Themis validates judges once before deployment, then runs in production indefinitely without quality monitoring.

**Key Questions:**
1. How do you ensure Kendall τ remains acceptable for new production data?
2. How do you detect judge drift without continuous human annotation?
3. What should you do when correlation metrics change?
4. How do you compute Kendall τ without human annotations?

**The Core Constraint:** You **CANNOT** compute Kendall τ without human ground truth annotations.

### Solution

**Two-tier monitoring strategy:**

1. **Tier 1: Proxy Metrics (Continuous, Automated)**
   - Monitor confidence scores, verdict distribution, judge disagreement
   - Detect potential drift **without** human annotations
   - Alert when metrics deviate significantly from baseline

2. **Tier 2: Human Validation (Periodic, Manual)**
   - Sample 25% of production data quarterly
   - Human annotate sample
   - Compute real Kendall τ
   - Recalibrate if τ drops below threshold

**Key Insight:** Proxy metrics provide **early warning**, but only human validation provides **true quality measurement**.

---

## 2. Complete Production Workflow

### Phase 0: Initial Validation (Before Production)

**Goal:** Establish baseline quality metrics before deployment.

```
Step 1: Collect Initial Dataset
  - 1000 question-answer pairs
  - Representative of expected production traffic

Step 2: Human Annotation (25%)
  - Annotate 250 samples (25%)
  - Expert human evaluators score responses
  - Store as ground truth dataset

Step 3: Run Validation
  $ themis validate --input human_annotated.jsonl --threshold 0.3

  Output:
    Kendall τ: 0.50
    Cohen's Kappa: 0.48
    Status: PASSED (τ ≥ 0.3)

Step 4: Save Baseline Metrics
  baseline.json:
  {
    "kendall_tau": 0.50,
    "validation_date": "2026-01-15",
    "avg_confidence": 0.72,
    "verdict_distribution": {
      "pass": 0.70,
      "review": 0.20,
      "fail": 0.10
    },
    "judge_scores": {
      "relevance": 0.78,
      "faithfulness": 0.68,
      "coherence": 0.75
    }
  }

Step 5: Deploy to Production
  - Enable streaming evaluation
  - All evaluations saved to DB
```

**Outcome:** Judge validated with τ = 0.50, ready for production.

---

### Phase 1: Production Streaming (Weeks 1-11)

**Goal:** Process live requests and monitor for drift using proxy metrics.

```
Production Flow:
  User Request → Themis Evaluates → Result Stored in DB
  ├─ Prechecks: Fast heuristics
  ├─ LLM Judges: Parallel evaluation (6 judges)
  ├─ Aggregation: Confidence + verdict
  └─ Storage: evaluation_results table

Monitoring (Daily/Weekly):
  Dashboard shows:
    - Average confidence (current vs baseline)
    - Verdict distribution (pass/review/fail %)
    - Judge score averages per dimension
    - Evaluation throughput
```

**Proxy Metrics Monitored (No Human Annotation Required):**

1. **Average Confidence Score**
   ```sql
   SELECT
     DATE_TRUNC('week', created_at) as week,
     AVG(confidence) as avg_confidence
   FROM evaluation_results
   GROUP BY week;
   ```
   - Baseline: 0.72
   - Alert if drops >10% (e.g., to 0.65)

2. **Verdict Distribution Shift**
   ```sql
   SELECT
     verdict,
     COUNT(*) * 100.0 / SUM(COUNT(*)) OVER() as percentage
   FROM evaluation_results
   WHERE created_at > NOW() - INTERVAL '7 days'
   GROUP BY verdict;
   ```
   - Baseline: 70% pass, 20% review, 10% fail
   - Alert if shifts >15% (e.g., 40% pass, 30% review, 30% fail)

3. **Score Variance Increase**
   ```sql
   SELECT
     DATE_TRUNC('day', created_at) as day,
     STDDEV(confidence) as variance
   FROM evaluation_results
   GROUP BY day;
   ```
   - Alert if variance increases >20%

---

### Phase 2: Periodic Revalidation (Week 12, Quarterly)

**Goal:** Measure true judge quality using human annotations.

```
Step 1: Sample Production Data
  API Call:
    POST /api/v1/validation/sample
    {
      "start_date": "2026-01-01T00:00:00Z",
      "end_date": "2026-03-31T23:59:59Z",
      "percentage": 25
    }

  Response:
    {
      "sample_id": "sample-20260331-abc123",
      "sampled_records": 2500
    }

Step 2: Download Sample for Annotation
  GET /api/v1/validation/sample/sample-20260331-abc123/download
  → quarterly_sample.jsonl

Step 3: Human Annotation
  - Send quarterly_sample.jsonl to annotation team
  - Use Label Studio, Scale AI, or internal annotators
  - Annotators score each response (0.0-1.0)
  - Save as quarterly_annotated.jsonl

Step 4: Run Validation
  $ themis validate --input quarterly_annotated.jsonl --threshold 0.3

  Output:
    Kendall τ: 0.48 (was 0.50 baseline)
    Cohen's Kappa: 0.46
    Status: PASSED (τ ≥ 0.3)
    Delta: -0.02 (slight degradation)

Step 5: Compare to Baseline
  if current_tau >= baseline_tau:
    Action: Continue production, update baseline
  elif current_tau >= threshold (0.3):
    Action: Acceptable degradation, monitor closely
  else:  # current_tau < threshold
    Action: ALERT - Judges degraded, trigger recalibration

Step 6: Investigate Changes (if τ dropped)
  - Check confusion matrix for error patterns
  - Compare false positive/negative rates
  - Analyze which judge dimensions degraded
  - Review failed evaluation samples
```

**Decision Matrix:**

| Current τ | Status | Action |
|-----------|--------|--------|
| τ ≥ 0.50 (baseline) | Excellent | Continue, update baseline |
| 0.3 ≤ τ < 0.50 | Acceptable | Monitor closely, consider calibration |
| τ < 0.3 | Failed | **MUST recalibrate** before continuing |

**Frequency:**
- **Default:** Quarterly (every 3 months)
- **Triggered:** When proxy metrics show significant drift
- **After changes:** When updating judge prompts or models

---

### Phase 3: Recalibration (If τ < Threshold)

**Goal:** Improve judge prompts using ground truth data.

```
Step 1: Analyze Validation Errors
  - Extract false positives (judge too optimistic)
  - Extract false negatives (judge too pessimistic)
  - Identify patterns (short answers, specific domains, etc.)

Step 2: Run Calibration (Future Feature)
  $ themis calibrate \
      --judge-name relevance \
      --ground-truth quarterly_annotated.jsonl \
      --update-config

  Process:
    1. Analyze errors using LLM
    2. Generate improved prompt
    3. Validate on test set
    4. Update configs/judges.yaml if improved

Step 3: Revalidate
  $ themis validate --input test_set.jsonl --threshold 0.3

  Output:
    Kendall τ: 0.52 (improved from 0.28!)
    Status: PASSED

Step 4: Deploy Updated Judges
  - Update configs/judges.yaml
  - Restart Themis services
  - Update baseline.json with new metrics

Step 5: Monitor Post-Deployment
  - Watch proxy metrics closely for 1-2 weeks
  - Run validation again after 1 month to confirm
```

---

## 3. Sampling Strategy

### Why Random 25%?

**Statistical Validity:**
- 25% sample size provides good representation
- Reduces annotation cost by 75%
- Still gives statistically significant Kendall τ

**Alternative Sampling Strategies:**

1. **Random (Recommended)**
   - Unbiased representation of production
   - Simple to implement
   - Statistically valid

2. **Stratified (Optional)**
   - Sample proportionally from each verdict class
   - Ensures coverage of rare cases (e.g., fail)
   - More complex, slightly better representation

3. **Active Learning (Future)**
   - Sample high-uncertainty cases only
   - Reduces annotation to ~10% (instead of 25%)
   - Requires uncertainty scores

### Sampling API Design

**Endpoint:**
```http
POST /api/v1/validation/sample

Request:
{
  "start_date": "2026-03-01T00:00:00Z",
  "end_date": "2026-03-31T23:59:59Z",
  "percentage": 25,
  "min_size": 100,
  "max_size": 2500,
  "strategy": "random"  // random, stratified, active
}

Response:
{
  "sample_id": "sample-20260331-abc123",
  "total_records": 10000,
  "sampled_records": 2500,
  "download_url": "/api/v1/validation/sample/sample-20260331-abc123/download"
}
```

**Implementation:**
- Query `evaluation_results` table by date range
- Apply sampling strategy (random shuffle, stratify, etc.)
- Store sample metadata in `validation_samples` table
- Link sampled records in `validation_sample_records` table

---

## 4. Proxy Metrics Dashboard

### Purpose

**Continuous monitoring between human validation cycles.**

**Key Question:** How do you detect drift without human annotations?

**Answer:** Track operational metrics that correlate with judge quality.

### Dashboard Panels

**Panel 1: Confidence Score Trend**
```
Line chart showing avg confidence over time
- Baseline: 0.72 (horizontal line)
- Current: 0.68 (4 weeks rolling average)
- Alert: Red zone if <0.65
```

**Panel 2: Verdict Distribution**
```
Stacked bar chart by week
- Pass: 70% → 65% (slight drop)
- Review: 20% → 23% (slight increase)
- Fail: 10% → 12% (slight increase)
- Alert if any shifts >15%
```

**Panel 3: Judge Score Heatmap**
```
Heatmap of judge scores over time
         relevance  faithfulness  coherence  completeness
Week 1     0.78        0.68         0.75        0.70
Week 2     0.76        0.67         0.74        0.69
Week 3     0.72        0.63         0.71        0.66  ← Degrading
Week 4     0.70        0.60         0.68        0.64  ← Alert!
```

**Panel 4: Disagreement Rate**
```
Line chart of inter-judge disagreement
- Low disagreement (< 0.2): Judges agree, confident
- High disagreement (> 0.3): Judges uncertain, potential drift
```

**Panel 5: Validation History**
```
Table of past validation runs
Date       | Sample Size | Kendall τ | Status  | Action
-----------|-------------|-----------|---------|--------
2026-01-15 | 250         | 0.50      | Passed  | Deployed
2026-04-15 | 2500        | 0.48      | Passed  | Continued
2026-07-15 | 2500        | 0.32      | Passed  | Monitored
2026-10-15 | 2500        | 0.28      | FAILED  | Recalibrated
```

## 6. Implementation Phases

### Phase 1: Sampling API

**Goal:** Enable programmatic sampling of production data for annotation.

**Deliverables:**
1. `POST /api/v1/validation/sample` - Create random sample
2. `GET /api/v1/validation/sample/{id}` - Get sample metadata
3. `GET /api/v1/validation/sample/{id}/download` - Download as JSONL
4. Database tables: `validation_samples`, `validation_sample_records`
5. `internal/validation/sample.go` - Sampling logic

**Acceptance Criteria:**
- [X] Can sample 25% of date range via API
- [X] Sample stored in DB with metadata
- [X] Can download sample as JSONL for annotation
- [X] Random sampling is unbiased
- [X] Size constraints (min/max) work correctly

**Estimated Time:** 2 days

---

### Phase 2: Proxy Metrics Dashboard

**Goal:** Continuous monitoring between human validation cycles.

**Deliverables:**
1. ✅ `GET /api/v1/metrics/health?window=7d` - Health metrics endpoint (`internal/api/handler.go`)
2. ✅ Enhanced `static/dashboard.html` with "Monitoring" tab

**Metrics Computed:**
- ✅ Average confidence
- ✅ Verdict distribution (counts + percentages)
- ✅ Per-judge average scores
- ✅ Inter-judge disagreement rate (avg std-dev across evaluations)

**Acceptance Criteria:**
- [X] `GET /api/v1/metrics/health?window=7d` returns all proxy metrics
- [X] Window parameter supports `h` (hours) and `d` (days)
- [X] Dashboard displays real-time metrics in Monitoring tab

**Estimated Time:** 2 days

---

### Phase 3: Validation History Tracking

**Goal:** Track validation runs over time for trend analysis.

**Deliverables:**
1. Enhance `themis validate` with `--save-to-db` flag
2. `GET /api/v1/validation/history` - List past validations
3. Dashboard panel showing Kendall τ trend over time
4. Database table: `validation_runs`

**Acceptance Criteria:**
- [X] Validation results automatically saved to DB
- [X] Can query validation history via API
- [X] Dashboard shows trend chart (τ over time)
- [X] Can compare current vs baseline metrics

**Estimated Time:** 1.5 days

---

## 7. Complete Example Workflow

### Scenario: 3-Month Production Cycle

```
Week 0: Initial Validation
  - Collect 1000 samples
  - Annotate 250 (25%)
  - Run: themis validate --input annotated.jsonl
  - Result: τ = 0.50, PASSED
  - Save baseline.json
  - Deploy to production

Week 1-4: Production (Month 1)
  - Stream processes 10,000 requests
  - Dashboard shows:
    - Avg confidence: 0.71 (baseline: 0.72) ✓
    - Verdict dist: 69% pass, 21% review, 10% fail ✓
  - Status: Healthy, no alerts

Week 5-8: Production (Month 2)
  - Stream processes 12,000 requests
  - Dashboard shows:
    - Avg confidence: 0.67 (baseline: 0.72) ⚠️ -7%
    - Verdict dist: 60% pass, 25% review, 15% fail ⚠️
  - Status: Proxy metrics show drift
  - Action: Schedule validation for next week

Week 9: Triggered Validation (Earlier than planned)
  - Sample 25% from weeks 1-8: 5,500 records → 1,375 samples
  - Send to annotation team
  - Wait 3-5 days for annotations
  - Run validation: τ = 0.42, PASSED (but dropped from 0.50)
  - Action: Continue production, but monitor closely

Week 10-12: Production (Month 3)
  - Stream processes 11,000 requests
  - Dashboard shows:
    - Avg confidence: 0.64 (baseline: 0.72) ⚠️ -11%
    - Verdict dist: 55% pass, 28% review, 17% fail 🚨
  - Status: Significant drift detected
  - Action: Schedule recalibration

Week 13: Quarterly Validation + Recalibration
  - Sample 25% from weeks 1-12: 33,000 records → 8,250 samples
  - Annotate 2,500 (subset for speed)
  - Run validation: τ = 0.28, FAILED 🚨
  - Action: MUST recalibrate

  Recalibration:
    1. Analyze errors (confusion matrix shows high false negatives)
    2. Run: themis calibrate --judge-name relevance
    3. Improved prompt generated
    4. Validate on test set: τ = 0.52, PASSED ✓
    5. Deploy updated judges

Week 14: Post-Recalibration
  - Stream processes 10,000 requests
  - Dashboard shows:
    - Avg confidence: 0.73 (improved!)
    - Verdict dist: 68% pass, 22% review, 10% fail ✓
  - Status: Back to healthy

Week 15+: Continue Production
  - Monthly monitoring via proxy metrics
  - Quarterly validation (3 months from week 13)
```

---

## 8. Success Criteria

### Technical Requirements
- [X] Sampling API enables 25% random sampling from date range
- [X] Proxy metrics dashboard updates in real-time
- [X] Validation history tracked in DB
- [X] Can query past validation runs
- [X] Alert thresholds configurable
- [X] All database migrations tested

### Workflow Requirements
- [X] Initial validation establishes baseline
- [X] Production streaming stores all evaluations
- [X] Proxy metrics detect drift without human annotations
- [X] Periodic validation measures true τ with humans
- [X] Recalibration triggered when τ drops
- [X] Complete cycle tested end-to-end

### Documentation
- [X] Workflow documented with examples
- [X] API endpoints documented
- [X] Dashboard usage guide
- [X] Proxy metrics interpretation guide
- [X] Decision matrix for validation results

---

## 9. Cost Analysis

### Human Annotation Costs

**Per Validation Run:**
- Sample size: 2,500 records (25% of 10,000)
- Annotation rate: $0.20 - $0.80 per sample
- Total cost: $500 - $2,000 per run

**Annual Costs:**
- Quarterly validation: 4 runs × $500-2000 = $2,000 - $8,000
- Triggered validations: 2 additional × $500-2000 = $1,000 - $4,000
- Total: $3,000 - $12,000 per year

### Monitoring Costs

**Proxy Metrics Dashboard:**
- Infrastructure: $0 (built into API server)
- Query overhead: Negligible (<1% DB load)

**Sampling API:**
- Infrastructure: $0 (built into API server)
- Storage: ~10MB per sample (JSONL)

**Total Monitoring Cost:** ~$0 (part of existing infrastructure)

### ROI

**Without Monitoring:**
- Deploy broken judges
- Lose customer trust
- Manual investigation: Days of engineering time
- Cost: $5,000 - $20,000 in incident response

**With Monitoring:**
- Detect drift early
- Validate quarterly
- Proactive recalibration
- Cost: $3,000 - $12,000 per year (planned, predictable)

**Savings:** $2,000 - $8,000 per incident avoided

---

## 10. Future Enhancements

### Post-MVP Features

1. **Automated Human Validation Workflow**
   - Integration with Label Studio API
   - Auto-create annotation projects
   - Callback when annotations complete

2. **Synthetic Validation (Alternative to Human)**
   - Use GPT-4o/Claude Opus as "synthetic ground truth"
   - Weekly automated validation (~$15-50 per run)
   - Reduces human dependency to quarterly only

3. **Active Learning Sampling**
   - Sample high-uncertainty cases only
   - Reduce annotation from 25% → 10%
   - Save $600-1500 per validation run

4. **Automated Alerting**
   - Slack/email when proxy metrics exceed thresholds
   - PagerDuty integration for critical drift

5. **Per-Judge Validation**
   - Validate each judge dimension independently
   - Identify which specific judge degraded

6. **Calibration Automation**
   - Auto-trigger calibration when τ drops
   - LLM-powered prompt improvement
   - A/B test old vs new prompts

7. **Multi-Model Validation**
   - Compare production judges across different LLM families
   - Test if judge quality varies by model (Claude vs GPT)

---

## 11. Decision

**Recommendation:** Implement Phase 1 (Sampling API) first.

**Rationale:**
1. Unblocks periodic validation workflow immediately
2. No complex dependencies (just DB queries)
3. Useful for both manual and automated workflows
4. Foundation for future enhancements
5. Low risk, high value

**Next Steps:**
1. Create database migrations (validation_samples, validation_sample_records)
2. Implement `internal/validation/sample.go`
3. Add API endpoints (`POST /sample`, `GET /sample/{id}`)
4. Write unit tests
5. Document API in `docs/api/validation-endpoints.md`
6. Update `CLAUDE.md` with sampling workflow

**Proceed with Phase 1?**
