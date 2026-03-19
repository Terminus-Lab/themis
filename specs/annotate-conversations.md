# annotate-conversations — Spec

**Status:** Proposal
**Date:** 2026-03-19

---

## Purpose

Close the annotation loop for conversation-level drift detection:

```
sample/conversations/download → annotate-conversations → validate-conversations
```

Annotators need a way to label sampled conversations as `pass / review / fail` without opening a JSON file in a text editor. This command provides a terminal-based, single-keypress UX.

---

## Command

```bash
themis-cli annotate-conversations -i sampled.jsonl
themis-cli annotate-conversations -i sampled.jsonl -o my_annotations.jsonl
```

**Flags:**

| Flag | Default | Description |
|---|---|---|
| `-i / --input` | required | JSONL from `sample/conversations/download` |
| `-o / --output` | `{input}_annotated.jsonl` | Annotation output file |

**Output filename rule:** strip `.jsonl` suffix, append `_annotated.jsonl`.
`sampled.jsonl` → `sampled_annotated.jsonl`

---

## Terminal UX

For each conversation, render:

```
─────────────────────────────────────────
Conversation 3 / 47  ·  conv-abc123
─────────────────────────────────────────
Turn 1
  User:  What is Python?
  Agent: Python is a high-level programming language...

Turn 2
  User:  Is it hard to learn?
  Agent: Not at all. It has clean syntax...

─────────────────────────────────────────
[p] pass   [r] review   [f] fail   [s] skip
```

- Single keypress — no Enter needed
- Record written immediately after keypress (safe on Ctrl+C)
- `[s] skip` — conversation is **not written** to output at all

---

## Output Format

Original JSON line + `human_annotation` field appended:

```json
{
  "conversation_id": "conv-abc123",
  "agent": {"name": "support-bot", "version": "2.1"},
  "turns": [...],
  "human_annotation": "pass"
}
```

Skipped conversations produce no output line.

---

## Key Constraints

- **File-only** — no DB reads or writes. DB is for live traffic only.
- **No deduplication** — input files from the sample endpoint never contain `human_annotation`; no need to check for it.
- **Append-safe** — each annotation is flushed immediately. Interrupting mid-session preserves all completed annotations.
- **No LLM calls** — no `setup.Wire()`. Pure file I/O + terminal.

---

## Workflow

```bash
# 1. Sample conversations for annotation
curl -X POST http://localhost:18082/api/v1/validation/sample/conversations/download \
  -d '{"percentage": 25}' -o sampled.jsonl

# 2. Annotate (interactive)
themis-cli annotate-conversations -i sampled.jsonl
# writes: sampled_annotated.jsonl

# 3. Validate judge accuracy
themis-cli validate-conversations -i sampled_annotated.jsonl
```
