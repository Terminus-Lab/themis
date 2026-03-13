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

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-input` | string | **required** | Input JSONL file path |
| `-output` | string | **required** | Output file path |
| `-format` | string | "jsonl" | Output format: "jsonl" or "summary" |
| `-summary` | string | "" | Optional separate summary file |

**Environment Variables:**
| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `THEMIS_BATCH_WORKERS` | int | 5 | Number of concurrent evaluation workers |

**Validate Command Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-input` | string | **required** | Input file path with human annotations |
| `-correlation-threshold` | float | 0.3 | Kendall's tau threshold for validation |

## Input Format (JSONL)

Each line is a JSON object with the same structure as the API request:

```jsonl
{"event_id":"eval-001","event_type":"agent_response","agent":{"name":"my-agent","type":"rag","version":"1.0"},"interaction":{"user_query":"What is the capital of France?","context":"France is a country in Western Europe. Paris is its capital.","answer":"The capital of France is Paris."}}
{"event_id":"eval-002","event_type":"agent_response","agent":{"name":"my-agent","type":"rag","version":"1.0"},"interaction":{"user_query":"What is AI?","context":"AI stands for Artificial Intelligence.","answer":"AI is the simulation of human intelligence by machines."}}
```

**Optional fields:**
- `conversation_id` - For grouping related evaluations into multi-turn conversations. Enables conversation-level analysis and metrics.
- `expected_output` - For correctness evaluation (ground truth comparison). Only used if correctness judge is enabled in `judges.yaml`.

**Example with expected_output:**
```jsonl
{"event_id":"eval-003","event_type":"agent_response","agent":{"name":"my-agent","type":"rag","version":"1.0"},"interaction":{"user_query":"What is 2+2?","context":"Basic math.","answer":"Four","expected_output":"4"}}
```

**Example with conversation_id (multi-turn conversation):**
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
./themis-cli evaluate -input test-valid.jsonl -output results.jsonl
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
./themis-cli evaluate -input test-invalid.jsonl -output results.jsonl
```

**Expected Output:**
- Exit code: 0 (continues on parse errors)
- Warning log: "Skipping record with parse error" for line 2
- `results.jsonl` contains 2 lines (valid records only)

### Test Case 3: Summary Format

**Command:**
```bash
./themis-cli evaluate -input test-valid.jsonl -format summary
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
./themis-cli evaluate -input large-dataset.jsonl -output results.jsonl

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
THEMIS_BATCH_WORKERS=20 ./themis-cli evaluate -input dataset-100.jsonl -output results.jsonl
```

**Expected Output:**
- All 100 records processed
- Logs show "Starting worker pool" with workers=20
- Processing time < sequential execution time
- All results written correctly

### Test Case 6: Invalid Format Flag

**Command:**
```bash
./themis-cli evaluate -input test.jsonl -format csv
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
./themis-cli validate \
  -input resources/annotated_sample.jsonl \
  -correlation-threshold 0.3
```

**Expected Output (if correlation passes):**
- Exit code: 0
- Validation report printed with:
  - Kendall's τ ≥ 0.3
  - Agreement rate
  - Confusion matrix
  - Status: "PASSED"
- `validation-summary.json` file created
- Message: "LLM judge validated against human annotations"

**Console output:**
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
└──────────────────────────────────────────┘

✅ LLM judge validated against human annotations
→ Safe to evaluate full dataset with these judge prompts
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

./themis-cli evaluate -input test-no-annotation.jsonl -validate
```

**Expected:**
- Exit code: 1
- Error: "Validation mode requires all records to have 'human_annotation' field"
- Lists records missing annotations

### Test Case 8: Combined Results + Summary

**Command:**
```bash
./themis-cli \
  -input resources/dataset.jsonl \
  -output resources/results.jsonl \
  -summary resources/summary.json
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
time THEMIS_BATCH_WORKERS=10 ./themis-cli \
  -input dataset-1000.jsonl \
  -output results-1000.jsonl
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
./themis-cli evaluate -input conversation-dataset.jsonl -output results.jsonl
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
time THEMIS_BATCH_WORKERS=1 ./themis-cli evaluate -input dataset-100.jsonl -output /dev/null

# 5 workers (default)
time ./themis-cli evaluate -input dataset-100.jsonl -output /dev/null

# 20 workers (high concurrency)
time THEMIS_BATCH_WORKERS=20 ./themis-cli evaluate -input dataset-100.jsonl -output /dev/null
```

**Expected:**
- Performance improves with more workers (up to a point)
- Diminishing returns after ~10-15 workers (LLM API rate limits)
- No errors or race conditions

## Integration with Analysis Tools

### Filter Failed Evaluations with jq

```bash
# Run evaluation first
./themis-cli evaluate -input dataset.jsonl -output results.jsonl

# Then analyze results
jq 'select(.verdict=="fail")' results.jsonl
```

### Calculate Average Confidence with jq

```bash
# Run evaluation first
./themis-cli evaluate -input dataset.jsonl -output results.jsonl

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

**Solution:** Ensure you specify `-input` flag with a valid file path.

```bash
./themis-cli evaluate -input dataset.jsonl
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
THEMIS_BATCH_WORKERS=3 ./themis-cli evaluate -input dataset.jsonl -output results.jsonl
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
