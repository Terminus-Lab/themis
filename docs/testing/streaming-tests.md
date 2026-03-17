---
title: Streaming Mode Test Cases
description: Test scenarios for Themis Redis Stream consumer
version: 1.0.0
tags: [testing, streaming, redis, async, consumer, horizontal-scaling]
related:
  - deployment/api-mode.md
  - testing/api-tests.md
  - testing/batch-tests.md
  - testing/mcp-tests.md
---

# Streaming Mode Test Cases

Test scenarios for Themis running as a Redis Stream consumer for asynchronous evaluation.

## Overview

In unified API + Streaming mode, Themis:
- Runs HTTP API for synchronous requests and metrics
- Connects to a Redis instance
- Joins a consumer group (`eval-group`)
- Consumes messages from the `eval-events` stream
- Processes evaluation requests asynchronously
- Supports graceful shutdown and acknowledgment

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
# Enable streaming consumer alongside API
STREAMING_ENABLED=true

# Redis Stream Configuration
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_STREAM_KEY=eval-events
REDIS_CONSUMER_GROUP=eval-group
REDIS_CONSUMER_NAME=consumer-1

# LLM Provider
OPEN_AI_KEY=sk-proj-...
# OR AWS Bedrock credentials

# API Configuration
EVAL_AGENT_API_PORT=18082
```

### Build the Producer

```bash
go build -o bin/themis-producer cmd/producer/main.go
```

### Start the Consumer

```bash
STREAMING_ENABLED=true ./bin/themis-api
```

**Expected output:**
```
INFO Streaming mode enabled - starting Redis consumer
INFO Starting streaming consumer stream_key=eval-events consumer_group=eval-group consumer_name=consumer-1
INFO Starting Themis Server address=:18082 streaming_enabled=true
```

## Test Cases

### Test Case 1: Basic Message Consumption

**Send message using CLI producer:**

```bash
./bin/themis-producer -d '{
  "event_id": "test-001",
  "event_type": "agent_response",
  "agent": {
    "name": "test-agent",
    "version": "1.0"
  },
  "interaction": {
    "user_query": "What is the capital of France?",
    "context": "France is a country in Western Europe. Paris is its capital.",
    "answer": "The capital of France is Paris."
  }
}'
```

**Expected:**
- Producer logs: "Message sent successfully"
- Consumer logs:
  ```
  INFO Processing message from stream message_id=<id> event_id=test-001
  INFO Evaluation complete event_id=test-001 verdict=pass confidence=0.92
  INFO Message acknowledged
  ```
- Message acknowledged and removed from pending list

**Verification:**
```bash
# Check pending messages (should be empty)
redis-cli XPENDING eval-events eval-group - + 10
```

### Test Case 2: High Quality Answer

**Send message:**

```bash
./bin/themis-producer -d '{
  "event_id": "test-002",
  "event_type": "agent_response",
  "agent": {"name": "test-agent", "version": "1.0"},
  "interaction": {
    "user_query": "What is 2+2?",
    "answer": "The answer is 4."
  }
}'
```

**Expected logs:**
```
INFO Processing message event_id=test-002
INFO Evaluation complete verdict=pass confidence=0.95
```

**Verification:** Check result in database (if persistence enabled)

### Test Case 3: Poor Quality Answer (Early Exit)

**Send message:**

```bash
./bin/themis-producer -d '{
  "event_id": "test-003",
  "event_type": "agent_response",
  "agent": {"name": "test-agent", "version": "1.0"},
  "interaction": {
    "user_query": "Explain quantum computing in detail",
    "answer": "Yes."
  }
}'
```

**Expected logs:**
```
INFO Processing message event_id=test-003
INFO Early exit triggered confidence=0.15
INFO Evaluation complete verdict=fail confidence=0.15 duration=<fast>
```

**Verification:**
- Fast processing (< 1 second)
- No LLM judge calls (check logs for judge execution)

### Test Case 4: Hallucination Detection

**Send message:**

```bash
./bin/themis-producer -d '{
  "event_id": "test-004",
  "event_type": "agent_response",
  "agent": {"name": "test-agent", "version": "1.0"},
  "interaction": {
    "user_query": "What is the population of Tokyo?",
    "context": "Tokyo is the capital of Japan.",
    "answer": "Tokyo has 50 million people and is the largest city in China."
  }
}'
```

**Expected logs:**
```
INFO Processing message event_id=test-004
INFO Evaluation complete verdict=fail confidence=0.39
INFO Stages: faithfulness=0.1 coherence=0.2
```

**Verification:** Low faithfulness and coherence scores

### Test Case 5: Multiple Concurrent Messages

**Send 10 messages:**

```bash
for i in {1..10}; do
  ./bin/themis-producer -d "{
    \"event_id\": \"concurrent-$i\",
    \"event_type\": \"agent_response\",
    \"agent\": {\"name\": \"test-agent\", \"version\": \"1.0\"},
    \"interaction\": {
      \"user_query\": \"Test query $i\",
      \"answer\": \"Test answer $i\"
    }
  }"
done
```

**Expected:**
- All 10 messages processed successfully
- No race conditions or errors
- Messages processed in order received
- All messages acknowledged

**Verification:**
```bash
# Check consumer group info
redis-cli XINFO GROUPS eval-events

# Check pending count (should be 0)
redis-cli XPENDING eval-events eval-group
```

### Test Case 6: Invalid JSON Payload

**Send invalid message:**

```bash
redis-cli XADD eval-events '*' payload '{invalid json}'
```

**Expected logs:**
```
ERROR Failed to parse message payload error="invalid character 'i' looking for beginning of object key string"
INFO Message acknowledged (parse error)
```

**Verification:**
- Message acknowledged (not redelivered)
- Error logged but consumer continues

### Test Case 7: Missing Required Fields

**Send incomplete message:**

```bash
./bin/themis-producer -d '{
  "event_id": "test-incomplete",
  "interaction": {
    "user_query": "Test"
  }
}'
```

**Expected logs:**
```
ERROR Validation error: missing required field: answer
INFO Message acknowledged (validation error)
```

**Verification:**
- Message acknowledged
- Validation error logged
- Consumer continues processing

### Test Case 8: Graceful Shutdown

**Action:**
1. Start consumer
2. Send 5 messages
3. Press Ctrl+C while messages are processing

**Expected behavior:**
```
INFO Received shutdown signal, finishing current work...
INFO Completing in-flight evaluations...
INFO All in-flight evaluations completed
INFO Streaming consumer stopped
INFO Server shutdown complete
```

**Verification:**
- In-flight evaluations complete
- Messages acknowledged
- No orphaned messages in pending list

### Test Case 9: Consumer Restart (Message Redelivery)

**Action:**
1. Start consumer
2. Send message
3. Kill consumer abruptly (kill -9) before acknowledgment
4. Restart consumer

**Expected behavior:**
- Message redelivered on restart
- Message processed successfully
- Message acknowledged

**Verification:**
```bash
# Check pending messages before restart
redis-cli XPENDING eval-events eval-group - + 10

# Should show 1 pending message
# After restart and processing, pending count should be 0
```

### Test Case 10: Horizontal Scaling (Multiple Consumers)

**Setup:**

Start 3 consumer instances:

```bash
# Consumer 1
STREAMING_ENABLED=true REDIS_CONSUMER_NAME=worker-1 EVAL_AGENT_API_PORT=18082 ./themis-api &

# Consumer 2
STREAMING_ENABLED=true REDIS_CONSUMER_NAME=worker-2 EVAL_AGENT_API_PORT=18083 ./themis-api &

# Consumer 3
STREAMING_ENABLED=true REDIS_CONSUMER_NAME=worker-3 EVAL_AGENT_API_PORT=18084 ./themis-api &
```

**Send 30 messages:**

```bash
for i in {1..30}; do
  ./bin/themis-producer -d "{
    \"event_id\": \"scale-$i\",
    \"event_type\": \"agent_response\",
    \"agent\": {\"name\": \"test-agent\", \"version\": \"1.0\"},
    \"interaction\": {
      \"user_query\": \"Test query $i\",
      \"answer\": \"Test answer $i\"
    }
  }"
done
```

**Expected:**
- Messages distributed across 3 consumers
- No duplicate processing
- All 30 messages processed successfully
- Logs show different consumers processing different messages

**Verification:**
```bash
# Check consumer group consumers
redis-cli XINFO CONSUMERS eval-events eval-group

# Should show 3 consumers with pending=0
```

### Test Case 11: API Endpoint Available During Streaming

**Action:**
While streaming consumer is running, test API endpoint:

```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "api-test",
    "agent": {"name": "test-agent"},
    "interaction": {
      "user_query": "What is AI?",
      "answer": "AI is artificial intelligence."
    }
  }'
```

**Expected:**
- API request processed successfully
- Returns evaluation result
- Streaming consumer continues processing Redis messages
- No interference between API and streaming

### Test Case 12: Metrics Endpoint

**Action:**

```bash
curl http://localhost:18082/metrics
```

**Expected metrics:**
```
# HELP themis_evaluations_total Total number of evaluations
# TYPE themis_evaluations_total counter
themis_evaluations_total{source="stream",verdict="pass"} 15
themis_evaluations_total{source="stream",verdict="fail"} 3
themis_evaluations_total{source="api",verdict="pass"} 2

# HELP themis_evaluation_duration_seconds Evaluation duration
# TYPE themis_evaluation_duration_seconds histogram
themis_evaluation_duration_seconds_bucket{source="stream",le="1"} 5
themis_evaluation_duration_seconds_bucket{source="stream",le="5"} 18

# HELP themis_stream_messages_processed Messages processed from stream
# TYPE themis_stream_messages_processed counter
themis_stream_messages_processed{status="success"} 18
themis_stream_messages_processed{status="error"} 2
```

**Verification:**
- Metrics separated by source (stream vs api)
- Counters incrementing correctly
- Histogram buckets populated

### Test Case 13: Redis Connection Failure

**Action:**
1. Start consumer with Redis running
2. Stop Redis server
3. Observe behavior

**Expected logs:**
```
ERROR Redis connection lost error="connection refused"
ERROR Failed to read from stream, retrying... retry_attempt=1
ERROR Failed to read from stream, retrying... retry_attempt=2
...
```

**Expected behavior:**
- Consumer retries connection
- API endpoints still respond (HTTP server unaffected)
- Once Redis reconnects, consumer resumes

**Verification:**
```bash
# Restart Redis
redis-server &

# Consumer logs should show:
INFO Redis connection restored
INFO Resuming stream processing
```

### Test Case 14: Large Payload

**Send message with large context:**

```bash
LARGE_CONTEXT=$(python3 -c "print('Context word. ' * 2000)")

./bin/themis-producer -d "{
  \"event_id\": \"large-payload\",
  \"event_type\": \"agent_response\",
  \"agent\": {\"name\": \"test-agent\", \"version\": \"1.0\"},
  \"interaction\": {
    \"user_query\": \"Summarize\",
    \"context\": \"$LARGE_CONTEXT\",
    \"answer\": \"This is a summary.\"
  }
}"
```

**Expected:**
- Message processed successfully
- No timeout or truncation
- Evaluation completes (may take longer)

### Test Case 15: Consumer Crash Recovery

**Action:**
1. Send message
2. Kill consumer with SIGKILL during processing (before acknowledgment)
3. Restart consumer

**Expected:**
- Message in pending list after crash
- Message claimed and reprocessed on restart
- Message eventually acknowledged

**Verification:**
```bash
# After crash, check pending
redis-cli XPENDING eval-events eval-group - + 10

# Should show message with consumer-1, idle time increasing
# After restart, pending count returns to 0
```

## Performance Tests

### Test Case 16: Throughput Test

**Send 1000 messages:**

```bash
time for i in {1..1000}; do
  ./bin/themis-producer -d "{
    \"event_id\": \"perf-$i\",
    \"event_type\": \"agent_response\",
    \"agent\": {\"name\": \"test-agent\", \"version\": \"1.0\"},
    \"interaction\": {
      \"user_query\": \"Test $i\",
      \"answer\": \"Answer $i\"
    }
  }" &

  if [ $((i % 10)) -eq 0 ]; then
    wait
  fi
done
wait
```

**Expected:**
- All 1000 messages processed
- No message loss
- Average throughput: 5-10 messages/second (depends on LLM latency)

**Verification:**
```bash
# Check stream length
redis-cli XLEN eval-events

# Check pending count
redis-cli XPENDING eval-events eval-group
```

### Test Case 17: Latency Test

**Send message and measure end-to-end latency:**

```bash
START=$(date +%s%N)
./bin/themis-producer -d '{
  "event_id": "latency-test",
  "event_type": "agent_response",
  "agent": {"name": "test-agent", "version": "1.0"},
  "interaction": {
    "user_query": "Test",
    "answer": "Test answer"
  }
}'

# Wait for processing
sleep 5

# Check logs for completion timestamp
END=$(date +%s%N)
LATENCY=$(((END - START) / 1000000))  # Convert to ms
echo "Latency: ${LATENCY}ms"
```

**Expected latency:**
- High quality answer (full pipeline): 3-5 seconds
- Poor quality answer (early exit): < 1 second

## Monitoring and Debugging

### Test Case 18: Redis Stream Info

**Check stream information:**

```bash
# Stream info
redis-cli XINFO STREAM eval-events

# Consumer group info
redis-cli XINFO GROUPS eval-events

# Consumers in group
redis-cli XINFO CONSUMERS eval-events eval-group

# Pending messages
redis-cli XPENDING eval-events eval-group - + 10
```

**Expected:**
- Stream exists with correct name
- Consumer group created
- Active consumers listed
- Pending count = 0 (all messages processed)

### Test Case 19: Manual Stream Inspection

**Manually inspect stream messages:**

```bash
# Read last 10 messages
redis-cli XREVRANGE eval-events + - COUNT 10

# Read specific message
redis-cli XRANGE eval-events <message-id> <message-id>
```

**Verification:**
- Messages have correct structure
- Payload field contains JSON
- Message IDs are sequential

### Test Case 20: Consumer Lag Monitoring

**Monitor consumer lag:**

```bash
# Check last delivered ID vs stream length
redis-cli XINFO CONSUMERS eval-events eval-group

# Check pending messages
redis-cli XPENDING eval-events eval-group
```

**Expected:**
- Low lag (< 10 messages behind)
- Pending count low or zero
- Idle time low for active consumers

## Troubleshooting

### Issue: Messages Not Being Consumed

**Debug steps:**
1. Check consumer is running: `ps aux | grep themis`
2. Check Redis connection: `redis-cli PING`
3. Check consumer group: `redis-cli XINFO GROUPS eval-events`
4. Check stream has messages: `redis-cli XLEN eval-events`

### Issue: Messages Stuck in Pending

**Solution:**
```bash
# Check pending messages
redis-cli XPENDING eval-events eval-group - + 10

# Claim stuck messages (if consumer dead)
redis-cli XCLAIM eval-events eval-group consumer-1 3600000 <message-id>
```

### Issue: Duplicate Processing

**Possible causes:**
- Multiple consumers with same `REDIS_CONSUMER_NAME`
- Consumer crashed before acknowledgment

**Solution:** Ensure unique consumer names:
```env
REDIS_CONSUMER_NAME=consumer-${HOSTNAME}-${RANDOM}
```

## Next Steps

- [API Test Cases](api-tests.md) - HTTP endpoint testing
- [Batch Test Cases](batch-tests.md) - CLI batch processing tests
- [MCP Test Cases](mcp-tests.md) - MCP integration tests
- [API Mode Deployment](../deployment/api-mode.md) - Unified API + Streaming setup
- [Configuration](../getting-started/configuration.md) - Environment setup
- [CLAUDE.md](../../CLAUDE.md) - Complete streaming, scaling, and monitoring guide
