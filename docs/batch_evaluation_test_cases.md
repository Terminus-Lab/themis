# Batch Evaluation CLI

Process multiple evaluation requests from a JSONL file using a worker pool for concurrent execution.

## Overview

The batch CLI enables offline evaluation of datasets without running the API server. Useful for:
- Testing prompt variations on large datasets
- A/B testing different judge configurations
- Generating evaluation reports for dataset quality assessment
- Research workflows and correlation analysis

## Quick Start

```bash
cd eval-agent
go run cmd/batch/main.go -input dataset.jsonl -output results.jsonl
```

## Command Line Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-input` | string | **required** | Input JSONL file path (or "-" for stdin) |
| `-output` | string | stdout | Output file path |
| `-format` | string | "jsonl" | Output format: "jsonl" or "summary" |
| `-summary` | string | "" | Optional separate summary file |
| `-workers` | int | 5 | Concurrent evaluation workers |
| `-continue-on-error` | bool | true | Continue on evaluation failures |
| `-dry-run` | bool | false | Validate input without evaluating |
| `-validate` | bool | false | Validation mode: compute correlation with human annotations |
| `-correlation-threshold` | float | 0.3 | Kendall's tau threshold for validation |

## Input Format (JSONL)

Each line is a JSON object with the same structure as the API request:

```jsonl
{"event_id":"eval-001","event_type":"agent_response","agent":{"name":"my-agent","type":"rag","version":"1.0"},"interaction":{"user_query":"What is the capital of France?","context":"France is a country in Western Europe. Paris is its capital.","answer":"The capital of France is Paris."}}
{"event_id":"eval-002","event_type":"agent_response","agent":{"name":"my-agent","type":"rag","version":"1.0"},"interaction":{"user_query":"What is AI?","context":"AI stands for Artificial Intelligence.","answer":"AI is the simulation of human intelligence by machines."}}
```

**Optional fields:**
- `expected_output` - For correctness evaluation (ground truth comparison). Only used if correctness judge is enabled in `judges.yaml`.

**Example with expected_output:**
```jsonl
{"event_id":"eval-003","event_type":"agent_response","agent":{"name":"my-agent","type":"rag","version":"1.0"},"interaction":{"user_query":"What is 2+2?","context":"Basic math.","answer":"Four","expected_output":"4"}}
```

## Output Formats

### JSONL Output (Default)

One evaluation result per line, directly pipeable to `jq`:

```jsonl
{"id":"eval-001","stages":[{"name":"length-checker","score":1.0,"reason":"Answer Length is acceptable","duration_ns":12500},{"name":"overlap-checker","score":0.85,"duration_ns":10000},{"name":"format-checker","score":1.0,"duration_ns":8500},{"name":"relevance-judge","score":0.95,"duration_ns":1850000000},{"name":"faithfulness-judge","score":1.0,"duration_ns":1820000000},{"name":"coherence-judge","score":1.0,"duration_ns":1780000000},{"name":"completeness-judge","score":1.0,"duration_ns":1750000000},{"name":"instruction-judge","score":1.0,"duration_ns":1690000000}],"confidence":0.92,"verdict":"pass"}
{"id":"eval-002","stages":[{"name":"length-checker","score":1.0,"duration_ns":14000},{"name":"overlap-checker","score":0.78,"duration_ns":11000},{"name":"format-checker","score":1.0,"duration_ns":9000},{"name":"relevance-judge","score":0.88,"reason":"Relevant answer","duration_ns":1820000000},{"name":"faithfulness-judge","score":0.9,"duration_ns":1780000000},{"name":"coherence-judge","score":1.0,"duration_ns":1750000000},{"name":"completeness-judge","score":0.9,"duration_ns":1720000000},{"name":"instruction-judge","score":1.0,"duration_ns":1680000000}],"confidence":0.85,"verdict":"pass"}
```

**Response structure** (same as API):
- `id`: Event ID from input
- `stages`: Array of all checker and judge results
  - **8 stages** when correctness judge disabled (3 prechecks + 5 judges)
  - **9 stages** when correctness judge enabled (3 prechecks + 6 judges)
- `confidence`: Aggregated confidence score (0.0-1.0)
- `verdict`: Final verdict ("pass", "review", or "fail")

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

## Usage Examples

### Basic Batch Evaluation

```bash
go run cmd/batch/main.go \
  -input test-dataset.jsonl \
  -output results.jsonl
```

### High Concurrency for Large Datasets

```bash
go run cmd/batch/main.go \
  -input large-dataset.jsonl \
  -workers 10 \
  -output results.jsonl
```

### Summary Report Only

```bash
go run cmd/batch/main.go \
  -input dataset.jsonl \
  -format summary \
  -output summary.json
```

### Combined: Results + Summary

```bash
go run cmd/batch/main.go \
  -input resources/dataset.jsonl \
  -output resources/results.jsonl \
  -summary resources/summary.json
```

### Pipeline from stdin

```bash
cat dataset.jsonl | go run cmd/batch/main.go -input - | jq 'select(.verdict=="fail")'
```

### Dry Run Validation

```bash
go run cmd/batch/main.go \
  -input dataset.jsonl \
  -dry-run
```

### Validation Mode (Human Annotation Correlation)

Validate LLM judge accuracy against human annotations by computing Kendall's correlation.

**Requirements:**
- Input file must have `human_annotation` field for each record
- Valid values: `"pass"`, `"review"`, `"fail"`

**Example:**
```bash
go run cmd/batch/main.go \
  -input annotated_sample.jsonl \
  -validate \
  -correlation-threshold 0.3
```

**Input record with human annotation:**
```jsonl
{
  "event_id":"val-001",
  "event_type":"agent_response",
  "agent":{"name":"test","type":"rag","version":"1.0"},
  "interaction":{
    "user_query":"What is the capital of France?",
    "context":"France is a country...",
    "answer":"The capital of France is Paris."
  },
  "human_annotation":"pass"
}
```

**Output (JSON to stdout):**
```json
{
  "total_records": 20,
  "agreement_count": 15,
  "agreement_rate": 0.75,
  "kendall_tau": 0.42,
  "threshold": 0.3,
  "passed": true,
  "confusion_matrix": {
    "pass_pass": 7,
    "pass_review": 1,
    "pass_fail": 0,
    "review_pass": 1,
    "review_review": 5,
    "review_fail": 1,
    "fail_pass": 0,
    "fail_review": 0,
    "fail_fail": 5
  },
  "interpretation": "Moderate agreement"
}
```

**Logs (to stderr):**
```
INFO Validation mode enabled
INFO Evaluating 20 records with human annotations...
INFO Evaluation complete duration=15.2s
INFO Computing Kendall's correlation...
INFO Validation complete records=20 agreement=15 agreement_rate=0.75 kendall_tau=0.42 threshold=0.3 status="PASSED" interpretation="Moderate agreement"
INFO Validation summary written file=validation-summary.json
INFO LLM judge validated against human annotations
INFO Safe to evaluate full dataset with these judge prompts
```

**If correlation is below threshold:**
```
ERROR Validation failed: Kendall's tau below threshold tau=0.18 threshold=0.3
ERROR Review configs/judges.yaml prompts and re-run validation
```

**Output format:**

The validation result is output as JSON to **stdout** (for piping to tools like `jq`), and also saved to `validation-summary.json`:

```bash
# Pipe to jq
go run cmd/batch/main.go -input annotated.jsonl -validate | jq '.kendall_tau'

# Save to file
go run cmd/batch/main.go -input annotated.jsonl -validate > my-validation.json
```

The JSON structure:
```json
{
  "total_records": 20,
  "agreement_count": 15,
  "agreement_rate": 0.75,
  "kendall_tau": 0.42,
  "threshold": 0.3,
  "passed": true,
  "confusion_matrix": {
    "pass_pass": 7,
    "pass_review": 1,
    ...
  },
  "interpretation": "Moderate agreement"
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
go run cmd/batch/main.go -input test-valid.jsonl -output results.jsonl
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
go run cmd/batch/main.go -input test-invalid.jsonl -output results.jsonl
```

**Expected Output:**
- Exit code: 0 (continues on parse errors)
- Warning log: "Skipping record with parse error" for line 2
- `results.jsonl` contains 2 lines (valid records only)

### Test Case 3: Dry Run Validation

**Input:** `test-mixed.jsonl` (contains 1 valid + 1 invalid)

**Command:**
```bash
go run cmd/batch/main.go -input test-mixed.jsonl -dry-run
```

**Expected Output:**
- Exit code: 1 (fails on validation errors)
- Error log: "Validation error" for invalid line
- Final log: "Validation failed" with error count
- No evaluations performed (no AWS calls)

### Test Case 4: Summary Format

**Command:**
```bash
go run cmd/batch/main.go -input test-valid.jsonl -format summary
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

### Test Case 5: Graceful Shutdown (SIGINT)

**Command:**
```bash
# Start processing large file
go run cmd/batch/main.go -input large-dataset.jsonl -output results.jsonl

# Press Ctrl+C after 2 seconds
```

**Expected Behavior:**
- Warning log: "Received interrupt signal, finishing current work..."
- In-flight evaluations complete
- Partial results written to `results.jsonl`
- Files properly closed
- Exit code: 0 or signal exit code

### Test Case 6: High Concurrency

**Command:**
```bash
go run cmd/batch/main.go -input dataset-100.jsonl -workers 20 -output results.jsonl
```

**Expected Output:**
- All 100 records processed
- Logs show "Starting worker pool" with workers=20
- Processing time < sequential execution time
- All results written correctly

### Test Case 7: Invalid Format Flag

**Command:**
```bash
go run cmd/batch/main.go -input test.jsonl -format csv
```

**Expected Output:**
- Exit code: 1
- Fatal error: "Invalid format. Supported: jsonl, summary"
- No processing occurs

### Test Case 8: Validation Mode (Human Annotation Correlation)

**Input:** `resources/annotated_sample.jsonl` (20 records with human annotations)

**Command:**
```bash
go run cmd/batch/main.go \
  -input resources/annotated_sample.jsonl \
  -validate \
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

go run cmd/batch/main.go -input test-no-annotation.jsonl -validate
```

**Expected:**
- Exit code: 1
- Error: "Validation mode requires all records to have 'human_annotation' field"
- Lists records missing annotations

## Performance

- **Throughput:** ~5-10 evaluations/second with 5 workers (depends on LLM latency)
- **Memory:** Loads all records into memory before processing (suitable for datasets up to ~10K records)
- **Cost:** Each evaluation = 1 precheck + 5 LLM calls (unless early exit)

## Troubleshooting

### "required flag -input not provided"
Ensure you specify `-input` flag with a valid file path.

### "Failed to open input file"
Check file exists and has read permissions.

### Worker pool processes 0 records
Check for parse errors in input JSONL. Use `-dry-run` to validate.

### High memory usage
Dataset is loaded into memory. For very large datasets (>100K), consider splitting into smaller batches.

## Integration with Analysis Tools

### Filter failed evaluations with jq
```bash
go run cmd/batch/main.go -input dataset.jsonl | jq 'select(.verdict=="fail")'
```

### Calculate average confidence with jq
```bash
go run cmd/batch/main.go -input dataset.jsonl | jq -s 'map(.confidence) | add/length'
```

### Import to pandas (Python)
```python
import pandas as pd

# Read JSONL output
df = pd.read_json('results.jsonl', lines=True)
print(df.describe())
```

## Future Enhancements

- [ ] CSV output format with dynamic columns
- [ ] Progress bar / live progress tracking
- [ ] Resume from checkpoint for large datasets
- [ ] Streaming output (write results as they complete)
- [ ] Per-judge statistics in summary
