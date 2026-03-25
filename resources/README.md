# Themis Resources

Sample datasets for testing Themis conversation evaluation.

## Files

### conversations.jsonl

Three sample multi-turn conversations in the current evaluation format. Use this for basic smoke testing of the API and CLI.

```bash
./bin/themis-cli evaluate -i resources/conversations.jsonl -o /tmp/results.jsonl
```

### annotated_sample.jsonl

Fifteen conversations with `human_label` (pass/review/fail) and `human_score` (0.0–1.0) annotations. When evaluated with the CLI, Themis automatically computes Kendall's τ-b, Cohen's κ (unweighted and weighted), and a confusion matrix.

```bash
./bin/themis-cli evaluate -i resources/annotated_sample.jsonl -o /tmp/results.jsonl
# Correlation report is appended as the last line of results.jsonl

# Or get summary + correlation in one file:
./bin/themis-cli evaluate -i resources/annotated_sample.jsonl -f summary
```

**Distribution:** 5 pass · 5 review · 5 fail

## Input Format

Each line in a JSONL file must be a valid conversation:

```json
{
  "conversation_id": "conv-001",
  "agent": {"name": "my-agent", "version": "1.0"},
  "turns": [
    {"turn_index": 0, "user_query": "What is AI?", "answer": "AI is..."},
    {"turn_index": 1, "user_query": "Can you elaborate?", "answer": "Sure..."}
  ]
}
```

Optional fields for human annotation workflows:

```json
{
  "conversation_id": "conv-001",
  "human_label": "pass",
  "human_score": 0.91,
  "agent": {"name": "my-agent", "version": "1.0"},
  "turns": [...]
}
```
