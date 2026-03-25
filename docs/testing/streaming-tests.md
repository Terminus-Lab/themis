---
title: Streaming Mode Test Cases
description: Test scenarios for Themis Redis Stream conversation consumer
version: 0.0.1
tags: [testing, streaming, redis, async, consumer, horizontal-scaling]
related:
  - deployment/api-mode.md
  - testing/api-tests.md
  - testing/batch-tests.md
  - testing/mcp-tests.md
---

# Streaming Mode Test Cases

Test scenarios for Themis running as a Redis Stream consumer for asynchronous conversation evaluation.

## Overview

In unified API + Streaming mode, Themis:
- Runs HTTP API for synchronous requests and metrics
- Connects to a Redis instance
- Joins a consumer group (`eval-conv-group`)
- Consumes `ConversationEvaluationRequest` messages from the `eval-conversations` stream
- Evaluates conversations asynchronously using the two-phase pipeline
- Supports graceful shutdown and message acknowledgment

## Setup

### Prerequisites

**1. Redis Server**

```bash
# Install Redis
brew install redis  # macOS
# OR
apt-get install redis-server  # Linux

# Start Redis
redis-server
```

**2. Environment Configuration**

Add streaming configuration to your `.env`:

```env
# Enable conversation streaming consumer alongside API
CONVERSATION_STREAMING_ENABLED=true

# Redis Stream Configuration
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_CONVERSATION_STREAM_KEY=eval-conversations
REDIS_CONVERSATION_GROUP=eval-conv-group
REDIS_CONSUMER_NAME=consumer-1

# LLM Provider
OPEN_AI_KEY=sk-proj-...

# API Configuration
EVAL_AGENT_API_PORT=18082
```

### Build the Producer

```bash
go build -o bin/themis-producer cmd/producer/main.go
```

### Start the Consumer

```bash
CONVERSATION_STREAMING_ENABLED=true go run cmd/api/main.go
```

**Expected output:**
```
INFO Streaming mode enabled - starting conversation Redis consumer
INFO Starting streaming consumer stream_key=eval-conversations consumer_group=eval-conv-group
INFO Starting Themis Server address=:18082
```

---

## Test Cases

### Test Case 1: Basic Conversation Consumption

**Send a conversation using the CLI producer:**

```bash
./bin/themis-producer -d '{
  "conversation_id": "stream-001",
  "agent": {"name": "test-agent", "version": "1.0"},
  "turns": [
    {"turn_index": 1, "user_query": "What is the capital of France?", "answer": "The capital of France is Paris."},
    {"turn_index": 2, "user_query": "And Germany?", "answer": "The capital of Germany is Berlin."}
  ]
}'
```

**Expected:**
- Producer logs: "Published successfully!"
- Consumer logs:
  ```
  INFO starting conversation evaluation conversation_id=stream-001 turn_count=2
  INFO conversation evaluation complete final_score=0.90 verdict=pass
  ```
- Message acknowledged and removed from pending list

**Verification:**
```bash
# Check pending messages (should be 0 after processing)
redis-cli XPENDING eval-conversations eval-conv-group - + 10
```

### Test Case 2: Single-Turn Conversation

```bash
./bin/themis-producer -d '{
  "conversation_id": "stream-002",
  "agent": {"name": "test-agent", "version": "1.0"},
  "turns": [
    {"turn_index": 1, "user_query": "What is 2+2?", "answer": "2+2 equals 4."}
  ]
}'
```

**Expected logs:**
```
INFO conversation evaluation complete conversation_id=stream-002 final_score=0.92 verdict=pass
```

### Test Case 3: Low Quality Answer

```bash
./bin/themis-producer -d '{
  "conversation_id": "stream-003",
  "agent": {"name": "test-agent", "version": "1.0"},
  "turns": [
    {"turn_index": 1, "user_query": "Explain quantum computing in detail.", "answer": "Yes."}
  ]
}'
```

**Expected logs:**
```
INFO conversation evaluation complete conversation_id=stream-003 final_score=0.15 verdict=fail
```

### Test Case 4: Multiple Concurrent Messages

```bash
for i in {1..10}; do
  ./bin/themis-producer -d "{
    \"conversation_id\": \"concurrent-$i\",
    \"agent\": {\"name\": \"test-agent\", \"version\": \"1.0\"},
    \"turns\": [{\"turn_index\": 1, \"user_query\": \"Test query $i\", \"answer\": \"Test answer $i\"}]
  }"
done
```

**Expected:**
- All 10 messages processed successfully
- No race conditions or errors
- All messages acknowledged

**Verification:**
```bash
redis-cli XINFO GROUPS eval-conversations
redis-cli XPENDING eval-conversations eval-conv-group
```

### Test Case 5: Invalid JSON Payload

```bash
redis-cli XADD eval-conversations '*' payload '{invalid json}'
```

**Expected logs:**
```
ERROR Failed to parse stream message error="invalid character..."
INFO Message acknowledged (parse error)
```

**Verification:**
- Message acknowledged (not redelivered)
- Error logged but consumer continues

### Test Case 6: Graceful Shutdown

1. Start consumer
2. Send 5 messages
3. Press Ctrl+C while messages are processing

**Expected behavior:**
```
INFO Received shutdown signal, finishing current work...
INFO Streaming consumer stopped
INFO Server shutdown complete
```

**Verification:**
- In-flight evaluations complete
- No orphaned messages in pending list

### Test Case 7: Consumer Restart (Message Redelivery)

1. Start consumer
2. Send message
3. Kill consumer abruptly (`kill -9`) before acknowledgment
4. Restart consumer

**Expected behavior:**
- Message redelivered on restart
- Message processed and acknowledged

**Verification:**
```bash
# Before restart — 1 pending message
redis-cli XPENDING eval-conversations eval-conv-group - + 10

# After restart and processing — 0 pending
redis-cli XPENDING eval-conversations eval-conv-group
```

### Test Case 8: Horizontal Scaling (Multiple Consumers)

```bash
# Consumer 1
CONVERSATION_STREAMING_ENABLED=true REDIS_CONSUMER_NAME=worker-1 EVAL_AGENT_API_PORT=18082 ./bin/themis-api &

# Consumer 2
CONVERSATION_STREAMING_ENABLED=true REDIS_CONSUMER_NAME=worker-2 EVAL_AGENT_API_PORT=18083 ./bin/themis-api &

# Consumer 3
CONVERSATION_STREAMING_ENABLED=true REDIS_CONSUMER_NAME=worker-3 EVAL_AGENT_API_PORT=18084 ./bin/themis-api &
```

Send 30 messages and verify:
- Messages distributed across all 3 consumers
- No duplicate processing
- All 30 messages eventually acknowledged

```bash
redis-cli XINFO CONSUMERS eval-conversations eval-conv-group
# Should show 3 consumers
```

### Test Case 9: API Available During Streaming

While streaming consumer is running:

```bash
curl -X POST http://localhost:18082/api/v1/conversations/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "api-while-streaming",
    "agent": {"name": "test"},
    "turns": [{"turn_index": 1, "user_query": "What is AI?", "answer": "AI is artificial intelligence."}]
  }'
```

**Expected:**
- HTTP request processed synchronously
- Streaming consumer continues processing Redis messages concurrently
- No interference

### Test Case 10: Redis Connection Failure

1. Start consumer with Redis running
2. Stop Redis server
3. Observe behavior

**Expected logs:**
```
ERROR Redis connection lost error="connection refused"
ERROR Failed to read from stream, retrying...
```

**Expected behavior:**
- Consumer retries connection
- API endpoints still respond (HTTP server unaffected)
- Once Redis restarts, consumer resumes

---

## Monitoring

```bash
# Stream info
redis-cli XINFO STREAM eval-conversations

# Consumer group info
redis-cli XINFO GROUPS eval-conversations

# Consumers in group
redis-cli XINFO CONSUMERS eval-conversations eval-conv-group

# Pending messages
redis-cli XPENDING eval-conversations eval-conv-group - + 10

# Recent messages
redis-cli XREVRANGE eval-conversations + - COUNT 10
```

---

## Troubleshooting

### Messages Not Being Consumed
1. Check consumer is running: `ps aux | grep themis`
2. Check Redis: `redis-cli PING`
3. Check stream key matches: `REDIS_CONVERSATION_STREAM_KEY=eval-conversations`
4. Check consumer group: `redis-cli XINFO GROUPS eval-conversations`

### Messages Stuck in Pending
```bash
redis-cli XPENDING eval-conversations eval-conv-group - + 10

# Claim stuck messages if consumer is dead
redis-cli XCLAIM eval-conversations eval-conv-group consumer-1 3600000 <message-id>
```

### Duplicate Processing
Ensure unique consumer names:
```env
REDIS_CONSUMER_NAME=consumer-${HOSTNAME}-${RANDOM}
```

---

## Next Steps

- [API Test Cases](api-tests.md) - HTTP endpoint testing
- [Batch Test Cases](batch-tests.md) - CLI batch processing tests
- [MCP Test Cases](mcp-tests.md) - MCP integration tests
- [API Mode Deployment](../deployment/api-mode.md) - Unified API + Streaming setup
- [Configuration](../getting-started/configuration.md) - Environment setup
