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
| `--output` | `-o` | string | **required** | Output file path |

**Environment Variables:**
| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `THEMIS_BATCH_WORKERS` | int | 5 | Number of concurrent evaluation workers |

## Input Format

Each line in the input JSONL file is a `ConversationEvaluationRequest`:

```json
{"conversation_id":"conv-001","agent":{"name":"my-agent","version":"1.0"},"turns":[{"turn_index":1,"user_query":"What is Go?","answer":"Go is a statically typed programming language."},{"turn_index":2,"user_query":"Who created it?","answer":"Go was created by Google engineers."}]}
{"conversation_id":"conv-002","agent":{"name":"my-agent","version":"1.0"},"turns":[{"turn_index":1,"user_query":"What is Paris?","answer":"Paris is the capital of France."}]}
```

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
  -input /tmp/conversations.jsonl \
  -output /tmp/results.jsonl
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

### Test Case 2: Custom Worker Count

```bash
THEMIS_BATCH_WORKERS=10 go run cmd/batch/main.go evaluate \
  -input /tmp/conversations.jsonl \
  -output /tmp/results.jsonl
```

**Expected:**
- Same output as Test Case 1
- Faster processing for large datasets (10 workers instead of 5)

### Test Case 3: Empty Input File

**Setup:**
```bash
touch /tmp/empty.jsonl
```

**Command:**
```bash
go run cmd/batch/main.go evaluate \
  -input /tmp/empty.jsonl \
  -output /tmp/results.jsonl
```

**Expected:**
- Exit code: 0
- `/tmp/results.jsonl` is empty or contains 0 results
- No error output

### Test Case 4: Invalid JSON in Input

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
  -input /tmp/bad.jsonl \
  -output /tmp/results.jsonl
```

**Expected:**
- Exit code: 0
- `/tmp/results.jsonl` contains 2 valid results (bad line is skipped with a warning log)

### Test Case 5: Missing Required Flags

```bash
go run cmd/batch/main.go evaluate -input /tmp/conversations.jsonl
```

**Expected:**
- Exit code: non-zero
- Error: missing required `-output` flag

---

## Next Steps

- [API Test Cases](api-tests.md) - HTTP endpoint testing
- [MCP Test Cases](mcp-tests.md) - Claude Code integration
- [Configuration](../getting-started/configuration.md) - Tune judges and thresholds
