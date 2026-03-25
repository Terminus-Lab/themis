---
title: CLI Test Cases
description: Test scenarios for Themis CLI batch evaluation
version: 2.0.0
tags: [testing, batch, cli, offline, conversations]
related:
  - testing/api-tests.md
  - testing/mcp-tests.md
  - testing/streaming-tests.md
  - getting-started/configuration.md
---

# CLI Test Cases

Test scenarios for processing multiple conversation evaluation requests from JSONL files using concurrent workers.

## Overview

The batch CLI enables offline evaluation of conversation datasets without running the API server. Useful for:
- Evaluating large datasets of multi-turn conversations
- A/B testing different judge configurations
- Validating judge accuracy against human-annotated ground truth
- Generating evaluation reports

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

CONVERSATION_HOLISTIC_WEIGHT=0.5
```

## Command Line Flags

**evaluate Command Flags:**
| Flag | Shorthand | Type | Default | Description |
|------|-----------|------|---------|-------------|
| `--input` | `-i` | string | **required** | Input JSONL file path |
| `--output` | `-o` | string | — | Output JSONL file path (required unless `-f summary`) |
| `--format` | `-f` | string | `jsonl` | Output format: `jsonl` or `summary` |
| `--summary` | `-s` | string | — | Optional separate summary JSON file |
| `--save-to-db` | `-d` | bool | `false` | Persist results to database (requires `IN_MEMORY_DB=false`) |

**Environment Variables:**
| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `THEMIS_BATCH_WORKERS` | int | 5 | Number of concurrent evaluation workers |

## Input Format

### Plain Evaluation Input

Each line is a `ConversationEvaluationRequest`:

```json
{"conversation_id":"conv-001","agent":{"name":"my-agent","version":"1.0"},"turns":[{"turn_index":1,"user_query":"What is Go?","answer":"Go is a statically typed programming language."},{"turn_index":2,"user_query":"Who created it?","answer":"Go was created by Google engineers."}]}
{"conversation_id":"conv-002","agent":{"name":"my-agent","version":"1.0"},"turns":[{"turn_index":1,"user_query":"What is Paris?","answer":"Paris is the capital of France."}]}
```

### Annotated Input (Human Ground Truth)

Add `human_label` and/or `human_score` fields to any conversation to enable correlation metrics:

```json
{"conversation_id":"conv-001","human_label":"pass","human_score":0.92,"agent":{"name":"my-agent","version":"1.0"},"turns":[...]}
{"conversation_id":"conv-002","human_label":"review","human_score":0.61,"agent":{"name":"my-agent","version":"1.0"},"turns":[...]}
{"conversation_id":"conv-003","human_label":"fail","human_score":0.20,"agent":{"name":"my-agent","version":"1.0"},"turns":[...]}
```

When annotations are present, the CLI automatically computes:

| Metric | Measures | When useful |
|--------|----------|-------------|
| Kendall's τ | Score rank correlation (continuous) | Is Themis score ordering consistent with human score ordering? |
| Cohen's κ | Label agreement (fail/review/pass) | Overall verdict match rate, chance-corrected |
| Weighted κ | Label agreement, severity-penalized | Distinguishes small errors (review↔pass) from large ones (fail↔pass) — useful for cross-version tracking |
| Confusion matrix | Per-class breakdown | Where exactly are the disagreements? |

A ready-to-use annotated dataset is included at `resources/annotated_sample.jsonl` (15 conversations: 5 pass, 5 review, 5 fail).

## Test Cases

### Test Case 1: Basic Batch Evaluation

**Setup:**
```bash
cat > /tmp/conversations.jsonl << 'EOF'
{"conversation_id":"conv-001","agent":{"name":"test-agent","version":"1.0"},"turns":[{"turn_index":1,"user_query":"What is the capital of France?","answer":"The capital of France is Paris."},{"turn_index":2,"user_query":"And Germany?","answer":"The capital of Germany is Berlin."}]}
{"conversation_id":"conv-002","agent":{"name":"test-agent","version":"1.0"},"turns":[{"turn_index":1,"user_query":"What is 2+2?","answer":"2+2 equals 4."}]}
EOF
```

**Command:**
```bash
go run cmd/batch/main.go evaluate \
  --input /tmp/conversations.jsonl \
  --output /tmp/results.jsonl
```

**Expected:**
- Exit code: 0
- `/tmp/results.jsonl` contains 2 lines
- Each line is a valid JSON `ConversationEvaluationResult`
- Results include `conversation_id`, `turn_avg`, `holistic_score`, `final_score`, `verdict`

**Sample output line:**
```json
{
  "conversation_id": "conv-001",
  "agent_name": "test-agent",
  "agent_version": "1.0",
  "turn_count": 2,
  "turn_avg": 0.91,
  "holistic_score": 0.88,
  "final_score": 0.895,
  "verdict": "pass",
  "turn_results": [...]
}
```

### Test Case 2: Summary Format (stdout)

Print a human-readable summary to stdout without writing a JSONL file:

```bash
go run cmd/batch/main.go evaluate \
  -i /tmp/conversations.jsonl \
  -f summary
```

**Expected:**
- Exit code: 0
- Structured summary printed to stdout
- No `-o` flag required

### Test Case 3: JSONL Results + Separate Summary File

```bash
go run cmd/batch/main.go evaluate \
  -i /tmp/conversations.jsonl \
  -o /tmp/results.jsonl \
  -s /tmp/summary.json
```

**Expected:**
- `/tmp/results.jsonl` — one JSON result per line
- `/tmp/summary.json` — aggregated summary object

### Test Case 4: Annotated Dataset — Correlation Metrics

Uses the bundled annotated sample to measure how well Themis agrees with human labels:

```bash
go run cmd/batch/main.go evaluate \
  -i resources/annotated_sample.jsonl \
  -f summary
```

**Expected:**
- Exit code: 0
- JSON summary printed to stdout, e.g.:

```json
{
  "total": 15,
  "verdict_counts": {
    "fail": 5,
    "pass": 5,
    "review": 5
  },
  "correlation_report": {
    "annotated_count": 15,
    "kendall_tau": 0.72,
    "cohens_kappa": 0.65,
    "weighted_kappa": 0.71,
    "confusion_matrix": {
      "labels": ["fail", "review", "pass"],
      "matrix": [
        [4, 1, 0],
        [0, 4, 1],
        [0, 1, 4]
      ]
    }
  }
}
```

To save both JSONL results and a separate summary in one pass:

```bash
go run cmd/batch/main.go evaluate \
  -i resources/annotated_sample.jsonl \
  -o /tmp/annotated_results.jsonl \
  -s /tmp/annotated_summary.json
```

The final line of the JSONL output (`-o`) will contain the correlation report:

```json
{"_type":"correlation_report","annotated_count":15,"kendall_tau":0.72,"cohens_kappa":0.65,"weighted_kappa":0.71,"confusion_matrix":{...}}
```

### Test Case 5: Custom Worker Count

```bash
THEMIS_BATCH_WORKERS=10 go run cmd/batch/main.go evaluate \
  --input /tmp/conversations.jsonl \
  --output /tmp/results.jsonl
```

**Expected:**
- Same output as Test Case 1
- Faster processing for large datasets (10 workers instead of 5)

### Test Case 6: Empty Input File

**Setup:**
```bash
touch /tmp/empty.jsonl
```

**Command:**
```bash
go run cmd/batch/main.go evaluate \
  --input /tmp/empty.jsonl \
  --output /tmp/results.jsonl
```

**Expected:**
- Exit code: 0
- `/tmp/results.jsonl` is empty or contains 0 results
- No error output

### Test Case 7: Invalid JSON in Input

**Setup:**
```bash
cat > /tmp/bad.jsonl << 'EOF'
{"conversation_id":"conv-ok","agent":{"name":"test","version":"1.0"},"turns":[{"turn_index":1,"user_query":"Q","answer":"A"}]}
{invalid json line}
{"conversation_id":"conv-ok-2","agent":{"name":"test","version":"1.0"},"turns":[{"turn_index":1,"user_query":"Q","answer":"A"}]}
EOF
```

**Command:**
```bash
go run cmd/batch/main.go evaluate \
  --input /tmp/bad.jsonl \
  --output /tmp/results.jsonl
```

**Expected:**
- Exit code: 0
- `/tmp/results.jsonl` contains 2 valid results (bad line is skipped with a warning log)

### Test Case 8: Missing Required Flags

```bash
go run cmd/batch/main.go evaluate --input /tmp/conversations.jsonl
```

**Expected:**
- Exit code: non-zero
- Error: `required flag "output" not set` (omitting `-o` is only valid with `-f summary`)

---

## Next Steps

- [API Test Cases](api-tests.md) - HTTP endpoint testing
- [MCP Test Cases](mcp-tests.md) - Claude Code integration
- [Configuration](../getting-started/configuration.md) - Tune judges and thresholds
