---
title: MCP Integration Test Cases
description: Test scenarios for Themis MCP server with Claude Code, Claude Desktop, and Cursor
version: 0.0.1
tags: [testing, mcp, claude-code, claude-desktop, cursor, integration]
related:
  - testing/api-tests.md
  - testing/batch-tests.md
  - testing/streaming-tests.md
  - getting-started/installation.md
---

# MCP Integration Test Cases

Test scenarios for Themis MCP server integration with Claude Code, Claude Desktop, and Cursor.

## Overview

The MCP (Model Context Protocol) server enables AI assistants to evaluate conversations directly within development environments.

**Available tools:**
- `evaluate_conversation` — Evaluate a multi-turn conversation
- `get_conversation` — Retrieve a stored conversation evaluation

## Setup

### Build MCP Server

```bash
go build -o bin/themis-mcp cmd/mcp/main.go
```

### Add to Claude Code

```bash
claude mcp add --transport stdio --scope project themis \
  --env OPEN_AI_KEY=sk-proj-... \
  -- /path/to/themis/bin/themis-mcp
```

### Verify Installation

```bash
claude mcp list
# Should show: themis (stdio) - Ready
```

---

## Tool Discovery Tests

### Test Case 1: List Available Tools

**Action:** Start Claude Code session and type `/mcp`

**Expected Output:**
```
MCP Servers:
- themis (stdio) - Ready
  Tools:
  - evaluate_conversation: Evaluate a multi-turn conversation...
  - get_conversation: Retrieve a stored conversation evaluation...
```

**Verification:**
- themis server shows as "Ready"
- Both tools are listed
- Tool descriptions are visible

---

## evaluate_conversation Tool Tests

### Test Case 2: Single-Turn Evaluation

**Prompt in Claude Code:**
```
Use the evaluate_conversation tool to evaluate this:

Conversation ID: test-001
Agent: my-agent v1.0
Turn 1: Query="What is the capital of France?" Answer="The capital of France is Paris."
```

**Expected Tool Call:**
```json
{
  "conversation_id": "test-001",
  "agent": {"name": "my-agent", "version": "1.0"},
  "turns": [
    {"turn_index": 1, "user_query": "What is the capital of France?", "answer": "The capital of France is Paris."}
  ]
}
```

**Expected Response:**
```json
{
  "conversation_id": "test-001",
  "turn_count": 1,
  "turn_avg": 0.93,
  "holistic_score": 0.91,
  "final_score": 0.92,
  "verdict": "pass"
}
```

**Verification:**
- Tool executes without errors
- Returns `final_score`, `turn_avg`, `holistic_score`, and `verdict`
- Claude interprets results and responds naturally

### Test Case 3: Multi-Turn Conversation

**Prompt:**
```
Evaluate this multi-turn conversation:

Conversation ID: conv-france
Agent: assistant v1.0
Turn 1: Query="What is the capital of France?" Answer="Paris is the capital of France."
Turn 2: Query="What is it known for?" Answer="Paris is famous for the Eiffel Tower, Louvre museum, and world-class cuisine."
Turn 3: Query="What is the population?" Answer="Paris has approximately 2.1 million people in the city proper."
```

**Expected Response:**
- `verdict` = "pass"
- `turn_avg` > 0.85 (each turn well-answered)
- `holistic_score` > 0.85 (conversation flows naturally)
- `turn_results` has 3 entries with individual scores

**Verification:**
- All 3 turns evaluated
- Holistic score reflects conversational flow
- Claude summarizes the evaluation result

### Test Case 4: Low Quality Answer

**Prompt:**
```
Evaluate this:

Conversation ID: conv-poor
Turn 1: Query="Explain quantum computing in detail." Answer="Yes."
```

**Expected Response:**
- Low `final_score` (< 0.3)
- `verdict` = "fail"
- Low `turn_score` (vague, single-word answer)

**Verification:**
- Tool detects low-quality answer
- Claude explains why the answer failed

### Test Case 5: Hallucination Detection

**Prompt:**
```
Check if this has hallucinations:

Conversation ID: conv-hallucination
Turn 1: Query="What is the population of Tokyo?" Answer="Tokyo has 50 million people and is the largest city in China."
```

**Expected Response:**
- Low scores on relevance and coherence judges
- `verdict` = "fail"

**Verification:**
- Judges detect contradictory/false claims
- Claude explains the hallucination issue

---

## get_conversation Tool Tests

### Test Case 6: Retrieve Stored Conversation

After evaluating `conv-france` in Test Case 3:

**Prompt:**
```
Use get_conversation to retrieve conversation conv-france
```

**Expected Response:**
```json
{
  "conversation_id": "conv-france",
  "agent_name": "assistant",
  "agent_version": "1.0",
  "turn_count": 3,
  "final_score": 0.91,
  "verdict": "pass",
  "turn_results": [...]
}
```

**Verification:**
- Returns full conversation with per-turn details
- Agent metadata preserved

### Test Case 7: Conversation Not Found

**Prompt:**
```
Get conversation conv-nonexistent
```

**Expected Response:**
- Error: conversation not found
- Claude explains no conversation with that ID exists

---

## Error Handling Tests

### Test Case 8: Missing Required Fields

**Prompt:**
```
Use evaluate_conversation but only provide turns, no conversation_id
```

**Expected Behavior:**
- Tool call fails with validation error
- Claude asks for the missing `conversation_id`

### Test Case 9: Empty Turns

**Prompt:**
```
Evaluate conversation conv-empty with no turns
```

**Expected Behavior:**
- Tool call fails with "turns must not be empty"
- Claude explains what's required

---

## Integration Tests

### Test Case 10: Sequential Evaluate and Retrieve

**Conversation:**
```
1. Evaluate this: conversation_id=conv-seq, turn 1: Q="What is Go?" A="Go is a statically typed language created by Google."
2. Now retrieve conversation conv-seq
```

**Verification:**
- Evaluate succeeds and returns scores
- Get retrieves the same conversation with consistent data

### Test Case 11: Multiple Conversations

**Prompt:**
```
Evaluate these 3 conversations separately:
1. conv-a: Q="What is 2+2?" A="4"
2. conv-b: Q="Capital of Spain?" A="Madrid"
3. conv-c: Q="Who wrote Hamlet?" A="Shakespeare"
```

**Expected Behavior:**
- Claude makes 3 separate tool calls
- All evaluations complete
- Claude summarizes and compares results

---

## Platform Tests

### Test Case 12: Docker MCP Server

**Setup:**
```bash
docker build -t themis-mcp .

claude mcp add --transport stdio --scope project themis \
  --env OPEN_AI_KEY=sk-proj-... \
  -- docker run -i --rm -e OPEN_AI_KEY themis-mcp:latest
```

**Test:** Use `evaluate_conversation` tool

**Expected:**
- Tool works identically to binary version
- Docker container starts/stops correctly

### Test Case 13: Cursor Integration

**Setup:** Add to Cursor MCP config:
```json
{
  "mcpServers": {
    "themis": {
      "command": "/path/to/themis/bin/themis-mcp",
      "env": {"OPEN_AI_KEY": "sk-proj-..."}
    }
  }
}
```

**Expected:** Tool appears and works identically to Claude Code

### Test Case 14: Claude Desktop Integration

**Setup:** Add to `claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "themis": {
      "command": "/path/to/themis/bin/themis-mcp",
      "env": {"OPEN_AI_KEY": "sk-proj-..."}
    }
  }
}
```

---

## Persistence Notes

### Test Case 15: In-Memory Persistence

```
Session 1:
1. Evaluate conversation conv-session-test (2 turns)
2. get_conversation conv-session-test → returns result ✓

3. Restart MCP server
4. get_conversation conv-session-test → "not found" ✓ (expected — in-memory DB)
```

**Note:** For persistent storage across restarts, set `IN_MEMORY_DB=false` and provide `THEMIS_DB_URL`.

---

## Quick Verification Checklist

- [ ] MCP server shows "Ready" in `/mcp` list
- [ ] Both tools (`evaluate_conversation`, `get_conversation`) are discoverable
- [ ] Single-turn evaluation works end-to-end
- [ ] Multi-turn evaluation returns `turn_results` for each turn
- [ ] `get_conversation` retrieves stored results
- [ ] Error messages are clear and helpful
- [ ] Works across Claude Code, Desktop, and Cursor

---

## Next Steps

- [API Test Cases](api-tests.md) - HTTP endpoint testing
- [Batch Test Cases](batch-tests.md) - CLI batch processing tests
- [Streaming Test Cases](streaming-tests.md) - Redis consumer tests
- [Configuration](../getting-started/configuration.md) - Environment setup
