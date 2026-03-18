---
title: CLI Test Cases
description: Test scenarios for Themis CLI
version: 1.0.0
tags: [testing, batch, cli, offline, validation, kendall]
related:
  - testing/api-tests.md
  - testing/mcp-tests.md
  - testing/streaming-tests.md
  - getting-started/configuration.md
---

# CLI Test Cases

Test scenarios for processing multiple evaluation requests from JSONL files using concurrent workers.

## Overview

The batch CLI enables offline evaluation of datasets without running the API server. Useful for:
- Testing prompt variations on large datasets
- A/B testing different judge configurations
- Generating evaluation reports for dataset quality assessment
- Research workflows and correlation analysis

## Setup

### Prerequisites

Ensure your `.env` file is configured:

```env
# LLM Provider (at least one required)
OPEN_AI_KEY=sk-proj-...
# OR
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...

# Pipeline configuration
ENABLE_PRECHECK=true
EARLY_EXIT_THRESHOLD=0.2
```

## Command Line Flags

| Flag | Shorthand | Type | Default | Description |
|------|-----------|------|---------|-------------|
| `--input` | `-i` | string | **required** | Input JSONL file path |
| `--output` | `-o` | string | **required** | Output file path |
| `--format` | `-f` | string | "jsonl" | Output format: "jsonl" or "summary" |
| `--summary` | `-s` | string | "" | Optional separate summary file |
| `--save-to-db` | `-d` | bool | false | Save results to database |

**Environment Variables:**
| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `THEMIS_BATCH_WORKERS` | int | 5 | Number of concurrent evaluation workers |

**Validate Command Flags:**
| Flag | Shorthand | Type | Default | Description |
|------|-----------|------|---------|-------------|
| `--input` | `-i` | string | **required** | Input file path with human annotations |
| `--correlation-threshold` | `-c` | float | 0.3 | Kendall's tau threshold for validation |
| `--save-to-db` | `-d` | bool | false | Save results to database |

## Input Format (JSONL)

Each line is a JSON object with the same structure as the API request:

```jsonl
{"event_id":"eval-001","conversation_id":"conv-batch-001","event_type":"agent_response","agent":{"name":"my-agent","type":"rag","version":"1.0"},"interaction":{"user_query":"What is the capital of France?","context":"France is a country in Western Europe. Paris is its capital.","answer":"The capital of France is Paris."}}
{"event_id":"eval-002","conversation_id":"conv-batch-002","event_type":"agent_response","agent":{"name":"my-agent","type":"rag","version":"1.0"},"interaction":{"user_query":"What is AI?","context":"AI stands for Artificial Intelligence.","answer":"AI is the simulation of human intelligence by machines."}}
```

**Required fields:**
- `conversation_id` - Groups related evaluations into multi-turn conversations. Every evaluation must have a conversation ID.

**Optional fields:**
- `expected_output` - For correctness evaluation (ground truth comparison). Only used if correctness judge is enabled in `judges.yaml`.

**Example with expected_output:**
```jsonl
{"event_id":"eval-003","conversation_id":"conv-batch-003","event_type":"agent_response","agent":{"name":"my-agent","type":"rag","version":"1.0"},"interaction":{"user_query":"What is 2+2?","context":"Basic math.","answer":"Four","expected_output":"4"}}
```

**Example of a multi-turn conversation:**
```jsonl
{"event_id":"turn-001","conversation_id":"conv-abc123","event_type":"agent_response","agent":{"name":"my-agent","type":"rag","version":"1.0"},"interaction":{"user_query":"What is the capital of France?","context":"France is a country in Europe.","answer":"The capital of France is Paris."}}
{"event_id":"turn-002","conversation_id":"conv-abc123","event_type":"agent_response","agent":{"name":"my-agent","type":"rag","version":"1.0"},"interaction":{"user_query":"What is the population?","context":"Paris is the capital of France.","answer":"Paris has approximately 2.2 million inhabitants."}}
{"event_id":"turn-003","conversation_id":"conv-abc123","event_type":"agent_response","agent":{"name":"my-agent","type":"rag","version":"1.0"},"interaction":{"user_query":"Tell me more about its history.","context":"Paris is an ancient city.","answer":"Paris was founded in the 3rd century BC by a Celtic people called the Parisii."}}
```

This groups 3 evaluations as turns in conversation `conv-abc123`, enabling conversation-level analysis.

## Output Formats

### JSONL Output (Default)

One evaluation result per line (can be processed with `jq`):

```jsonl
{"id":"eval-001","stages":[{"name":"length-checker","score":1.0,"reason":"Answer Length is acceptable","duration_ns":12500},{"name":"overlap-checker","score":0.85,"duration_ns":10000},{"name":"format-checker","score":1.0,"duration_ns":8500},{"name":"relevance-judge","score":0.95,"duration_ns":1850000000}],"confidence":0.92,"verdict":"pass"}
```

### Summary Output

Aggregate statistics in JSON format:

```json
{
  "total": 20,
  "pass_count": 15,
  "fail_count": 3,
  "review_count": 2,
  "avg_confidence": 0.847
}
```

## Test Cases

### Test Case 1: Valid JSONL Input

**Input:** `test-valid.jsonl`
```jsonl
{"event_id":"t1","event_type":"agent_response","agent":{"name":"test","type":"rag","version":"1.0"},"interaction":{"user_query":"What is 2+2?","context":"Math basics","answer":"4"}}
{"event_id":"t2","event_type":"agent_response","agent":{"name":"test","type":"rag","version":"1.0"},"interaction":{"user_query":"Capital of Spain?","context":"Spain is in Europe","answer":"Madrid"}}
```

**Command:**
```bash
./bin/themis-cli evaluate -i test-valid.jsonl -o results.jsonl
```

**Expected Output:**
- Exit code: 0
- `results.jsonl` contains 2 lines (one per evaluation)
- Both records have `verdict` field (`pass`, `review`, or `fail`)
- Logs show: "Input file parsed", "Starting worker pool", "Processing complete"

### Test Case 2: Invalid JSON Lines

**Input:** `test-invalid.jsonl`
```jsonl
{"event_id":"t1","event_type":"agent_response","agent":{"name":"test","type":"rag","version":"1.0"},"interaction":{"user_query":"Valid","context":"","answer":"Valid"}}
{invalid json}
{"event_id":"t3","event_type":"agent_response","agent":{"name":"test","type":"rag","version":"1.0"},"interaction":{"user_query":"Valid","context":"","answer":"Valid"}}
```

**Command:**
```bash
./bin/themis-cli evaluate -i test-invalid.jsonl -o results.jsonl
```

**Expected Output:**
- Exit code: 0 (continues on parse errors)
- Warning log: "Skipping record with parse error" for line 2
- `results.jsonl` contains 2 lines (valid records only)

### Test Case 3: Summary Format

When `-f summary` is used without `-o`, results are printed to stdout. With `-o`, they're written to the specified file.

**Command (stdout):**
```bash
./bin/themis-cli evaluate -i test-valid.jsonl -f summary
```

**Command (file):**
```bash
./bin/themis-cli evaluate -i test-valid.jsonl -f summary -o summary.json
```

**Expected Output:**
```json
{
  "total": 2,
  "pass_count": 2,
  "fail_count": 0,
  "review_count": 0,
  "avg_confidence": 0.91
}
```

### Test Case 4: Graceful Shutdown (SIGINT)

**Command:**
```bash
# Start processing large file
./bin/themis-cli evaluate -i large-dataset.jsonl -o results.jsonl

# Press Ctrl+C after 2 seconds
```

**Expected Behavior:**
- Warning log: "Received interrupt signal, finishing current work..."
- In-flight evaluations complete
- Partial results written to `results.jsonl`
- Files properly closed
- Exit code: 0 or signal exit code

### Test Case 5: High Concurrency

**Command:**
```bash
THEMIS_BATCH_WORKERS=20 ./bin/themis-cli evaluate -i dataset-100.jsonl -o results.jsonl
```

**Expected Output:**
- All 100 records processed
- Logs show "Starting worker pool" with workers=20
- Processing time < sequential execution time
- All results written correctly

### Test Case 6: Invalid Format Flag

**Command:**
```bash
./bin/themis-cli evaluate -i test.jsonl -f csv
```

**Expected Output:**
- Exit code: 1
- Fatal error: "Invalid format. Supported: jsonl, summary"
- No processing occurs

### Test Case 7: Validation Mode (Human Annotation Correlation)

**Input:** `resources/annotated_sample.jsonl` (20 records with human annotations)

```jsonl
{"event_id":"val-001","event_type":"agent_response","agent":{"name":"test","type":"rag","version":"1.0"},"interaction":{"user_query":"What is the capital of France?","context":"France is a country...","answer":"The capital of France is Paris."},"human_annotation":"pass"}
```

**Command:**
```bash
./bin/themis-cli validate-events -i resources/annotated_sample.jsonl -c 0.3
# or with long flags:
./bin/themis-cli validate-events --input resources/annotated_sample.jsonl --correlation-threshold 0.3
```

**Expected Output (if correlation passes):**
- Exit code: 0
- Validation report with comprehensive metrics:
  - **Kendall's τ** (PRIMARY) - Pass/fail decision
  - **Cohen's Kappa** (REPORT) - Industry standard agreement metric
  - **Confusion Matrix** (DEBUG) - Per-class breakdown
  - **Per-class metrics** - Precision/Recall/F1 for each label
- Status: "PASSED"
- Message: "LLM judge validated against human annotations"

**Console output (JSON structured logging):**
```
INFO Starting validation records=25 threshold=0.3
INFO Evaluation complete duration=12.3s
INFO Validation complete records=25 kendall_tau=0.42 tau_interpretation="Moderate agreement" cohens_kappa=0.38 kappa_interpretation="Fair agreement" threshold=0.3 status=PASSED
INFO LLM judge validated against human annotations
INFO Safe to evaluate full dataset with these judge prompts
```

**Validation summary JSON** (can save with `> validation-report.json`):
```json
{
  "passed": true,
  "total_records": 25,
  "threshold": 0.3,
  "correlation_metrics": {
    "kendalls_tau": 0.42,
    "interpretation": "Moderate agreement",
    "passed_threshold": true
  },
  "agreement_metrics": {
    "cohens_kappa": 0.38,
    "interpretation": "Fair agreement"
  },
  "confusion_matrix": {
    "fail": {"fail": 6, "review": 2, "pass": 1},
    "review": {"fail": 1, "review": 5, "pass": 2},
    "pass": {"fail": 0, "review": 1, "pass": 7}
  },
  "per_class_metrics": {
    "fail": {"precision": 0.857, "recall": 0.667, "f1": 0.750, "support": 9},
    "review": {"precision": 0.625, "recall": 0.625, "f1": 0.625, "support": 8},
    "pass": {"precision": 0.700, "recall": 0.875, "f1": 0.778, "support": 8}
  }
}
```

**Expected Output (if correlation fails):**
- Exit code: 1
- Validation report with:
  - Kendall's τ < 0.3
  - Status: "FAILED"
- Error message about threshold
- Guidance to review judge prompts

**Test with missing annotations:**
```bash
# Create test file with missing human_annotation
echo '{"event_id":"t1","interaction":{"user_query":"Test","answer":"Test"}}' > test-no-annotation.jsonl

./bin/themis-cli validate-events -i test-no-annotation.jsonl
```

**Expected:**
- Exit code: 1
- Error: "validation requires all records to have 'human_annotation' field"
- Lists records missing annotations

---

### Test Case 7.1: Validation with Passing Dataset (Production-Ready Judge)

**Input:** `resources/validation_test_dataset.jsonl` (150 records - 50 fail, 50 pass, 50 review)

This dataset tests a well-calibrated judge with clear category boundaries and strong correctness enforcement.

**Command:**
```bash
go run cmd/batch/main.go validate \
  -i resources/validation_test_dataset.jsonl \
  -c 0.3
```

**Expected Output:**
- Exit code: 0
- Status: ✓ PASSED
- **Kendall's τ**: 0.632 (Moderate to strong agreement)
- **Cohen's Kappa**: 0.910 (Almost perfect)
- **Overall Accuracy**: 94% (141/150 correct)

**Key Metrics:**
```json
{
  "passed": true,
  "total_records": 150,
  "correlation_metrics": {
    "kendalls_tau": 0.6315883668903803,
    "interpretation": "Moderate to strong agreement",
    "passed_threshold": true
  },
  "agreement_metrics": {
    "cohens_kappa": 0.9099999999999999,
    "interpretation": "Almost perfect"
  },
  "confusion_matrix": {
    "fail": {"fail": 49, "pass": 0, "review": 1},
    "pass": {"fail": 0, "pass": 50, "review": 0},
    "review": {"fail": 0, "pass": 8, "review": 42}
  },
  "per_class_metrics": {
    "fail": {"precision": 1.0, "recall": 0.98, "f1": 0.99, "support": 50},
    "pass": {"precision": 0.86, "recall": 1.0, "f1": 0.93, "support": 50},
    "review": {"precision": 0.98, "recall": 0.84, "f1": 0.90, "support": 50}
  }
}
```

**Interpretation:**
- ✅ Judge exceeds threshold by 2.1x (0.63 vs 0.3)
- ✅ Near-perfect categorical agreement (κ = 0.91)
- ✅ Zero critical errors (no fail→pass)
- ✅ Strong performance across all classes
- ✅ **Recommendation**: Deploy to production

**Full analysis:** See [resources/validation_test_dataset_interpretation.md](../../resources/validation_test_dataset_interpretation.md)

**Use case:** Baseline validation to confirm judge configuration is production-ready.

---

### Test Case 7.2: Validation with Failing Dataset (Judge Issue Detection)

**Input:** `resources/validation_failed_dataset.jsonl` (150 records - designed to expose common judge failure modes)

This dataset contains:
- **Fail cases (1-30)**: Verbose but empty answers (tests style-over-substance bias)
- **Fail cases (31-50)**: Confidently wrong answers (tests correctness detection)
- **Pass cases (51-100)**: Correct comprehensive answers
- **Review cases (101-150)**: Terse but correct answers (tests brevity penalty)

**Command:**
```bash
go run cmd/batch/main.go validate \
  -i resources/validation_failed_dataset.jsonl \
  -c 0.3
```

**Expected Output (with poorly calibrated judge):**
- Exit code: 1
- Status: ✗ FAILED
- **Kendall's τ**: < 0.3 (Below threshold)
- **Cohen's Kappa**: < 0.6 (Below industry standard)

**Hypothetical Failed Results:**
```json
{
  "passed": false,
  "total_records": 150,
  "correlation_metrics": {
    "kendalls_tau": 0.22,
    "interpretation": "Weak agreement",
    "passed_threshold": false
  },
  "agreement_metrics": {
    "cohens_kappa": 0.35,
    "interpretation": "Fair agreement"
  }
}
```

**Common Failure Patterns Detected:**
1. **Style-over-substance**: Judge promotes polite but empty answers (fail→pass)
2. **Weak correctness**: Judge doesn't catch factually wrong answers (fail→review)
3. **Penalizes brevity**: Judge rejects short but correct answers (review→fail)

**Full failure analysis and fixes:** See [resources/validation_failed_dataset_interpretation.md](../../resources/validation_failed_dataset_interpretation.md)

**Use case:**
- Test judge robustness against adversarial cases
- Identify systematic biases before production deployment
- Validate judge improvements after configuration changes

---

**Metric Interpretation Resources:**

For comprehensive guidance on interpreting validation results:
- **Kendall's τ**: [docs/metrics/kendalls-tau.md](../metrics/kendalls-tau.md)
- **Cohen's Kappa**: [docs/metrics/cohens-kappa.md](../metrics/cohens-kappa.md)
- **Confusion Matrix**: [docs/metrics/confusion-matrix.md](../metrics/confusion-matrix.md)
- **Interpretation Guide**: [docs/metrics/interpretation-guide.md](../metrics/interpretation-guide.md) - Decision framework and troubleshooting

### Test Case 8: Combined Results + Summary

**Command:**
```bash
./bin/themis-cli evaluate \
  -i resources/dataset.jsonl \
  -o resources/results.jsonl \
  -s resources/summary.json
```

**Expected:**
- `results.jsonl` contains detailed evaluation results
- `summary.json` contains aggregate statistics
- Both files created successfully

## Performance Tests

### Test Case 9: Large Dataset Performance

**Input:** 1000 records

**Command:**
```bash
time THEMIS_BATCH_WORKERS=10 ./bin/themis-cli evaluate \
  -i dataset-1000.jsonl \
  -o results-1000.jsonl
```

**Expected:**
- **Throughput**: ~5-10 evaluations/second with 10 workers
- **Memory**: Loads all records into memory (suitable for datasets up to ~10K records)
- **Cost**: Each evaluation = 1 precheck + 5 LLM calls (unless early exit)
- Processing time significantly faster than sequential

### Test Case 10: Conversation Tracking

**Input:** `conversation-dataset.jsonl`
```jsonl
{"event_id":"conv-a-turn-1","conversation_id":"conv-a","event_type":"agent_response","agent":{"name":"test","type":"rag","version":"1.0"},"interaction":{"user_query":"What is AI?","context":"","answer":"AI stands for Artificial Intelligence."}}
{"event_id":"conv-a-turn-2","conversation_id":"conv-a","event_type":"agent_response","agent":{"name":"test","type":"rag","version":"1.0"},"interaction":{"user_query":"How does it work?","context":"","answer":"It uses machine learning algorithms."}}
{"event_id":"conv-b-turn-1","conversation_id":"conv-b","event_type":"agent_response","agent":{"name":"test","type":"rag","version":"1.0"},"interaction":{"user_query":"What is Python?","context":"","answer":"Python is a programming language."}}
```

**Command:**
```bash
./bin/themis-cli evaluate -i conversation-dataset.jsonl -o results.jsonl
```

**Expected Output:**
- All 3 evaluations include `conversation_id` in stored results
- Evaluations can be queried by conversation via API: `GET /api/v1/conversations/conv-a`
- Enables conversation-level metrics (average confidence per conversation, verdict distribution across turns)

**Analyzing with jq:**
```bash
# Group results by conversation_id
jq -s 'group_by(.conversation_id) | map({conversation: .[0].conversation_id, turns: length, avg_confidence: (map(.confidence) | add / length)})' results.jsonl

# Expected output:
# [
#   {"conversation": "conv-a", "turns": 2, "avg_confidence": 0.85},
#   {"conversation": "conv-b", "turns": 1, "avg_confidence": 0.92}
# ]
```

### Test Case 11: Worker Pool Scaling

Test with different worker counts:

```bash
# 1 worker (sequential)
time THEMIS_BATCH_WORKERS=1 ./bin/themis-cli evaluate -i dataset-100.jsonl -o /dev/null

# 5 workers (default)
time ./bin/themis-cli evaluate -i dataset-100.jsonl -o /dev/null

# 20 workers (high concurrency)
time THEMIS_BATCH_WORKERS=20 ./bin/themis-cli evaluate -i dataset-100.jsonl -o /dev/null
```

**Expected:**
- Performance improves with more workers (up to a point)
- Diminishing returns after ~10-15 workers (LLM API rate limits)
- No errors or race conditions

## Integration with Analysis Tools

### Filter Failed Evaluations with jq

```bash
# Run evaluation first
./bin/themis-cli evaluate -i dataset.jsonl -o results.jsonl

# Then analyze results
jq 'select(.verdict=="fail")' results.jsonl
```

### Calculate Average Confidence with jq

```bash
# Run evaluation first
./bin/themis-cli evaluate -i dataset.jsonl -o results.jsonl

# Calculate average confidence
jq -s 'map(.confidence) | add/length' results.jsonl
```

### Import to pandas (Python)

```python
import pandas as pd

# Read JSONL output
df = pd.read_json('results.jsonl', lines=True)
print(df.describe())

# Filter by verdict
fails = df[df['verdict'] == 'fail']
print(fails[['id', 'confidence', 'verdict']])
```

## Troubleshooting

### Issue: "required flag -input not provided"

**Solution:** Ensure you specify `-i` (or `--input`) flag with a valid file path.

```bash
./bin/themis-cli evaluate -i dataset.jsonl -o results.jsonl
```

### Issue: "Failed to open input file"

**Solution:** Check file exists and has read permissions.

```bash
ls -la dataset.jsonl
chmod 644 dataset.jsonl
```

### Issue: Worker Pool Processes 0 Records

**Solution:** Check for parse errors in input JSONL. Check the logs during evaluation - invalid records will show warnings but evaluation continues.

### Issue: High Memory Usage

**Solution:** Dataset is loaded into memory. For very large datasets (>100K), consider splitting into smaller batches.

```bash
split -l 10000 large-dataset.jsonl batch-
# Process each batch separately
```

### Issue: LLM API Rate Limits

**Solution:** Reduce worker count to stay within rate limits.

```bash
THEMIS_BATCH_WORKERS=3 ./bin/themis-cli evaluate -i dataset.jsonl -o results.jsonl
```

## Validation Mode Details

### Kendall's Tau Correlation

Validation mode computes Kendall's tau (τ) correlation between LLM verdicts and human annotations:

- **τ ≥ 0.3**: Judge prompts validated (proceed to production)
- **τ < 0.3**: Judge prompts need improvement

### Confusion Matrix

Shows agreement between human and LLM verdicts:

```
                Human
          pass  review  fail
   pass    12      2     0
   review   1      3     1    LLM
   fail     0      2     4
```

### Workflow

1. **Collect human annotations** - Sample 25+ records from your dataset
2. **Add `human_annotation` field** - "pass", "review", or "fail"
3. **Run validation** - Compute correlation with LLM judges
4. **Iterate prompts** - If τ < 0.3, improve judge prompts in `judges.yaml`
5. **Deploy** - Once τ ≥ 0.3, safe to evaluate full dataset

See [CLAUDE.md](../../CLAUDE.md) for complete validation details.

## Performance Benchmarks

**Test environment**: MacBook Pro M1, 10 workers

| Dataset Size | Processing Time | Throughput |
|--------------|-----------------|------------|
| 10 records   | ~5 seconds      | 2 eval/s   |
| 100 records  | ~45 seconds     | 2.2 eval/s |
| 1000 records | ~7 minutes      | 2.4 eval/s |

**Notes**:
- Early exit significantly improves throughput for low-quality responses
- Performance limited by LLM API latency (not CPU)
- Memory usage: ~1MB per 1000 records

## Next Steps

- [API Test Cases](api-tests.md) - HTTP endpoint testing
- [MCP Test Cases](mcp-tests.md) - Claude Code integration testing
- [Streaming Test Cases](streaming-tests.md) - Redis consumer testing
- [Configuration](../getting-started/configuration.md) - Tune pipeline settings
- [CLAUDE.md](../../CLAUDE.md) - Complete documentation including validation
