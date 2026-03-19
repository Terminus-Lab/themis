Evaluate the current conversation using Themis CLI.

Extract every user/assistant exchange from this session as conversation turns, write them to a temporary JSONL file, run `themis-cli evaluate-conversations`, display the results, and then delete the temp file.

## Step 1 — locate the CLI

Use `THEMIS_CLI` if set, otherwise fall back to `themis-cli` on PATH:

```bash
THEMIS_CLI="${THEMIS_CLI:-themis-cli}"
```

If the binary is not found, stop and tell the user:

> `themis-cli` not found. Set the `THEMIS_CLI` environment variable to the full path of the binary, or add it to your PATH. Download from https://github.com/Terminus-Lab/themis/releases

## Step 2 — locate the judges config

`themis-cli` must be run from (or pointed at) a directory containing `configs/judges.yaml`.

Use `THEMIS_DIR` if set, otherwise try these in order:
1. `$HOME/.themis`
2. The directory containing the `themis-cli` binary

If `configs/judges.yaml` is not found in any location, stop and tell the user:

> `configs/judges.yaml` not found. Set `THEMIS_DIR` to the directory that contains the `configs/` folder. If you installed from a release archive, run from the extracted directory.

## Step 3 — build the conversation payload

Reconstruct the session turns from the current conversation. Each user/assistant exchange is one turn.

- `turn_index` starts at 0
- `user_query` is the user's message
- `answer` is your (Claude's or the assistant's) response
- Omit any turn where either side is empty
- If `$ARGUMENTS` contains a number (e.g. `5`), include only the last N turns

Build one JSON object:

```json
{
  "conversation_id": "claude-session-<unix-timestamp>",
  "agent": {"name": "claude-code", "version": "1.0"},
  "turns": [
    {"turn_index": 0, "user_query": "...", "answer": "..."},
    {"turn_index": 1, "user_query": "...", "answer": "..."}
  ]
}
```

## Step 4 — run evaluation

Write the JSON object as a single line to a temp file, run the CLI, then delete the temp file:

```bash
TMPFILE=$(mktemp /tmp/themis-eval-XXXXXX.jsonl)
echo '<payload>' > "$TMPFILE"

cd "$THEMIS_DIR" && \
  "$THEMIS_CLI" evaluate-conversations -i "$TMPFILE" -f summary

rm "$TMPFILE"
```

## Step 5 — display results

Show a clear summary:

- **Verdict**: `pass` / `review` / `fail`
- **Confidence**: score between 0.0 and 1.0
- **Turn count**: how many turns were evaluated
- A 2–3 sentence plain-English interpretation of what the scores mean for this conversation

---

## CLI command reference

| Command | What it does |
|---|---|
| `evaluate-events` | Evaluate individual single-turn events from a JSONL file |
| `evaluate-conversations` | Evaluate full multi-turn conversations from a JSONL file |
| `validate-events` | Validate judge accuracy against human annotations (Kendall's τ) |
| `validate-conversations` | Conversation-level judge validation *(coming soon)* |

### evaluate-events

Input: JSONL where each line has `event_id`, `agent`, and `interaction` fields.

```bash
themis-cli evaluate-events -i events.jsonl -o results.jsonl
THEMIS_BATCH_WORKERS=10 themis-cli evaluate-events -i events.jsonl -o results.jsonl
themis-cli evaluate-events -i events.jsonl -f summary
themis-cli evaluate-events -i events.jsonl -o results.jsonl -s summary.json
```

Flags:

| Flag | Default | Description |
|---|---|---|
| `-i / --input` | required | Input JSONL file |
| `-o / --output` | required unless `-f summary` | Output JSONL file |
| `-f / --format` | `jsonl` | `jsonl` or `summary` |
| `-s / --summary` | — | Write aggregate stats to a separate file |
| `-d / --save-to-db` | false | Persist results to database |

### evaluate-conversations

Input: JSONL where each line has `conversation_id`, `agent`, and `turns[]` fields.

```bash
themis-cli evaluate-conversations -i conversations.jsonl -o results.jsonl
themis-cli evaluate-conversations -i conversations.jsonl -f summary
themis-cli evaluate-conversations -i conversations.jsonl -o results.jsonl -s summary.json
```

Same flags as `evaluate-events`.

### validate-events

Input: JSONL with `human_annotation` (`pass` / `review` / `fail`) on each record.

```bash
themis-cli validate-events -i annotated.jsonl
themis-cli validate-events -i annotated.jsonl -c 0.5
```

Flags:

| Flag | Default | Description |
|---|---|---|
| `-i / --input` | required | JSONL with `human_annotation` per record |
| `-c / --correlation-threshold` | `0.3` | Minimum Kendall's τ to pass |
| `-d / --save-to-db` | false | Persist results to database |

### Output formats

**JSONL** (default) — one result per line, pipe to `jq`:

```bash
jq 'select(.verdict=="fail")' results.jsonl
jq -s 'map(.confidence) | add/length' results.jsonl
```

**Summary** — aggregate stats printed as JSON:

```json
{
  "total": 10,
  "pass_count": 7,
  "fail_count": 1,
  "review_count": 2,
  "avg_confidence": 0.83,
  "avg_turn_count": 3.1
}
```

### Verdict thresholds (set in `.env`)

| Verdict | Condition |
|---|---|
| `pass` | confidence > `VERDICT_PASS_THRESHOLD` (default 0.8) |
| `review` | confidence > `VERDICT_REVIEW_THRESHOLD` (default 0.5) |
| `fail` | confidence ≤ review threshold |
