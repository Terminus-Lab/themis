# Conversation Grouping Specification

**Purpose**: Group related evaluations into conversations to track multi-turn agent interactions.

**Scope**: API, Batch CLI, MCP, Streaming (all entry points)

---

## Status Summary

**Completion**: 6.67/7 Phases Complete (95%)

| Phase | Status | Progress |
|-------|--------|----------|
| Phase 1: Schema Migration | ✅ Complete | 100% |
| Phase 2: Storage Layer | ✅ Complete | 100% |
| Phase 3: API Layer | ✅ Complete | 100% |
| Phase 4: Entry Points | ⚠️ Partial | 67% (2/3 complete) |
| Phase 5: Dashboard | ✅ Complete | 100% |
| Phase 6: Documentation | ✅ Complete | 100% |
| Phase 7: Performance Opt | ⏸️ On Hold | N/A (trigger-based) |

**Next Steps**:
1. ~~Fix streaming consumer normalize() function~~ ✅ **Done**
2. ~~Add dashboard UI for conversations~~ ✅ **Done**
3. Update MCP adapter for conversation_id (optional)

---

## Feature Overview

### Problem
- Current system evaluates individual agent responses in isolation
- No way to track quality across multi-turn conversations
- Cannot analyze conversation-level patterns (e.g., degradation over turns)

### Solution
- Add optional `conversation_id` field to group related evaluations
- Compute conversation-level aggregates on read (no write complexity)
- Simple SQL GROUP BY queries - works with SQLite and PostgreSQL

### Value
- **Multi-turn Analysis**: Track agent quality across conversation sessions
- **Conversation Metrics**: Average confidence, verdict distribution per conversation
- **Debugging**: View all turns in a conversation for root cause analysis
- **Simple Design**: Single column, no complex write-time aggregation

---

## Implementation Phases

### Phase 1: Schema Migration
- [X] Add `conversation_id` column to evaluations table
- [X] Create indexes for conversation queries
- [X] Test migration on SQLite and PostgreSQL

### Phase 2: Storage Layer
- [X] Add `ConversationID` field to `Evaluation` struct
- [X] Implement `GetConversation()` method
- [X] Implement `ListConversations()` method
- [X] Add unit tests for conversation queries

### Phase 3: API Layer
- [X] Update `/api/v1/evaluate` handler to accept `conversation_id`
- [X] Add `GET /api/v1/conversations/{id}` endpoint
- [X] Add `GET /api/v1/conversations` endpoint
- [X] Add API integration tests

### Phase 4: Entry Points
- [X] Update batch CLI to support conversation_id in JSONL
  - Already supported via `models.EvaluationRequest`
  - Added test: `TestReader_WithConversationID()`
  - Added example file: `resources/conversation_example.jsonl`
  - Updated documentation in `docs/testing/batch-tests.md`
- [ ] Update MCP adapter to accept conversation_id
  - TODO: Update MCP request handling
- [X] Update streaming consumer to parse conversation_id
  - Fixed `normalize()` in `internal/stream/redis/consumer.go` to include ConversationID, AgentName, AgentVersion
  - Added comprehensive tests: `TestNormalize()` with 3 test cases + `TestNormalize_CreatedAtTimestamp()`
  - Now matches API handler normalize() function

### Phase 5: Dashboard
- [X] Add "Conversations" tab to dashboard
  - Tab navigation between Results and Conversations views
  - Load Conversations button with status indicator
- [X] Show conversation list with summary cards
  - Conversation ID, turn count, agent info
  - Average confidence across turns
  - Verdict distribution (pass/review/fail counts)
  - First/last turn timestamps
- [X] Drill-down view for conversation turns
  - Click conversation card to view all turns
  - Sequential display of query/answer pairs per turn
  - Individual turn verdicts and confidence scores
  - Back button to return to conversations list
- [X] Stats dashboard for conversations
  - Total conversations count
  - Average turns per conversation
  - Average confidence across all conversations

### Phase 6: Documentation
- [X] Update documentation. Update the existing examples
  - Updated `docs/testing/batch-tests.md` with conversation_id examples
  - Updated `resources/README.md` with conversation tracking guide
  - Created `CONVERSATION_BATCH_SUPPORT.md` with complete usage guide
  - Added 5 integration tests for conversation endpoints
- [X] Create a new batch of data multiple events correlated to a single conversation.
  - Created `resources/conversation_example.jsonl` with 10 records
  - 3 conversations: 4 turns (Paris), 3 turns (ML), 3 turns (Quantum)

### Phase 7: Performance Optimization (If Needed)
**Trigger**: Only implement if conversation query latency p95 >300ms in production.

**Option A: Application-Level Cache (Recommended)**
- [ ] Add in-memory LRU cache for conversation summaries
- [ ] Implement 5-minute TTL per conversation
- [ ] Add cache hit/miss metrics
- [ ] Monitor cache effectiveness (target >80% hit rate)
