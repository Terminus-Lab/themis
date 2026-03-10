# Eval Agent Resources

This directory contains test datasets and example payloads for testing the eval-agent.

## Files

### dataset.jsonl

A comprehensive test dataset with 20 evaluation requests in JSONL format (one JSON object per line). Each request includes:
- Various query types (technical, conceptual, troubleshooting, etc.)
- Different answer qualities (perfect, partial, irrelevant)
- Edge cases (vague queries, follow-ups, multi-part questions)

**Use with batch CLI:**
```bash
cd eval-agent
go run cmd/batch/main.go -input resources/dataset.jsonl -output results.jsonl
```

**Sample records:**
- High quality answers (expected: pass)
- Partial/incomplete answers (expected: review)
- Irrelevant/hallucinated answers (expected: fail)
- Edge cases (greetings, follow-ups, ambiguous queries)

**Total:** 20 test cases

### annotated_sample.jsonl

A validation dataset with 20 evaluation requests that include human annotations. Used for testing validation mode and computing Kendall's correlation between LLM judges and human judgment.

**Use with validation mode:**
```bash
cd eval-agent
go run cmd/batch/main.go \
  -input resources/annotated_sample.jsonl \
  -validate \
  -correlation-threshold 0.3
```

**Features:**
- Each record includes `human_annotation` field: "pass", "review", or "fail"
- Designed to test correlation analysis
- Mix of perfect, moderate, and poor answers
- Expected Kendall's τ > 0.5 (moderate to strong agreement)

**Sample record:**
```json
{
  "event_id": "val-001",
  "interaction": {
    "user_query": "What is the capital of France?",
    "context": "France is a country...",
    "answer": "The capital of France is Paris."
  },
  "human_annotation": "pass"
}
```

**Distribution:**
- Pass: 8 records
- Review: 6 records
- Fail: 6 records

### payloads/

Directory containing individual JSON payload examples for testing the HTTP API:

- `pass.json` - Example request that should return `pass` verdict
- `review.json` - Example request that should return `review` verdict
- `fail-early.json` - Example that triggers early exit (precheck failure)
- `README.md` - Documentation for payloads

**Use with API:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d @resources/payloads/pass.json
```

## Creating Your Own Test Dataset

To create a custom JSONL dataset:

```bash
# Each line is a separate JSON object
cat > my-dataset.jsonl << 'EOF'
{"event_id":"test-1","event_type":"agent_response","agent":{"name":"my-agent","type":"rag","version":"1.0"},"interaction":{"user_query":"What is AI?","context":"AI stands for Artificial Intelligence.","answer":"AI is the simulation of human intelligence by machines."}}
{"event_id":"test-2","event_type":"agent_response","agent":{"name":"my-agent","type":"rag","version":"1.0"},"interaction":{"user_query":"What is ML?","context":"ML is machine learning.","answer":"ML is a subset of AI that learns from data."}}
EOF

# Test it
go run cmd/batch/main.go -input my-dataset.jsonl -dry-run
```

## Dataset Statistics

**Current dataset.jsonl breakdown:**
- Perfect relevance (0.8-1.0): 10 test cases
- Medium relevance (0.4-0.7): 5 test cases
- Low/No relevance (0.0-0.4): 5 test cases

**Query types covered:**
- Technical how-to questions
- Conceptual explanations
- Troubleshooting scenarios
- Multi-part questions
- Follow-up questions with context
- Yes/No questions
- Comparison questions
- Greetings and clarifications
