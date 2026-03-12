# Conversation Grouping Specification

**Purpose**: Group related evaluations into conversations to track multi-turn agent interactions.

**Scope**: API, Batch CLI, MCP, Streaming (all entry points)

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
- [ ] Add unit tests for conversation queries

### Phase 3: API Layer
- [ ] Update `/api/v1/evaluate` handler to accept `conversation_id`
- [ ] Add `GET /api/v1/conversations/{id}` endpoint
- [ ] Add `GET /api/v1/conversations` endpoint
- [ ] Add API integration tests

### Phase 4: Entry Points
- [ ] Update batch CLI to support conversation_id in JSONL
- [ ] Update MCP adapter to accept conversation_id
- [ ] Update streaming consumer to parse conversation_id

### Phase 5: Dashboard (Optional)
- [ ] Add "Conversations" tab to dashboard
- [ ] Show conversation list with filters
- [ ] Drill-down view for conversation turns

### Phase 6: Documentation
- [] Update documentation. Update the existing examples
- [] Create a new batch of data multiple events correlated to a single conversation.

### Phase 7: Performance Optimization (If Needed)
**Trigger**: Only implement if conversation query latency p95 >300ms in production.

**Option A: Application-Level Cache (Recommended)**
- [ ] Add in-memory LRU cache for conversation summaries
- [ ] Implement 5-minute TTL per conversation
- [ ] Add cache hit/miss metrics
- [ ] Monitor cache effectiveness (target >80% hit rate)
