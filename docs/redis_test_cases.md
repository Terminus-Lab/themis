# Redis Stream Consumer Mode

Advanced usage guide for running eval-agent as a Redis Stream consumer for asynchronous evaluation.

## Overview

In Redis Stream consumer mode, eval-agent:
- Connects to a Redis instance
- Joins a consumer group (`eval-group`)
- Consumes messages from the `eval-events` stream
- Processes evaluation requests asynchronously
- Supports graceful shutdown and acknowledgment

## Use Cases

- **High-throughput evaluation**: Process evaluation requests asynchronously
- **Decoupled architecture**: Separate evaluation from agent response generation
- **Multiple consumers**: Scale horizontally with multiple eval-agent instances
- **Fault tolerance**: Redis Streams provide message persistence and redelivery

---

## Configuration

Add Redis configuration to your `.env`:

```env
# Redis Stream Configuration
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_STREAM_NAME=eval-events
REDIS_CONSUMER_GROUP=eval-group
REDIS_CONSUMER_NAME=eval-consumer-1
```

---

## Running the Consumer

Start the Redis Stream consumer:

```bash
cd eval-agent
go run cmd/streaming/main.go
```

**Expected output:**
```
{"level":"info","message":"Starting eval-agent Redis Stream consumer"}
{"level":"info","stream":"eval-events","group":"eval-group","message":"Consumer started"}
```

The consumer will block and wait for messages on the stream.

---

## Sending Evaluation Requests

### Option 1: CLI Producer (Recommended)

Use the built-in CLI producer:

```bash
go run cmd/producer/main.go -d '{
  "event_id": "evt-001",
  "event_type": "agent_response",
  "agent": {
    "name": "my-agent",
    "type": "rag",
    "version": "1.0.0"
  },
  "interaction": {
    "user_query": "What is the capital of France?",
    "context": "France is a country in Western Europe. Its capital city is Paris.",
    "answer": "The capital of France is Paris."
  }
}'
```

**Flags:**
- `-d <json>`: Inline JSON payload
- `-f <file>`: Read payload from file
- `--redis-addr <addr>`: Redis address (default: localhost:6379)
- `--stream <name>`: Stream name (default: eval-events)

**Example with file:**
```bash
echo '{"event_id":"evt-002",...}' > payload.json
go run cmd/producer/main.go -f payload.json
```

### Option 2: redis-cli

Send messages directly via redis-cli:

```bash
redis-cli XADD eval-events '*' payload '{
  "event_id": "evt-001",
  "event_type": "agent_response",
  "agent": {"name": "my-agent", "type": "rag", "version": "1.0.0"},
  "interaction": {
    "user_query": "What is the capital of France?",
    "context": "France is a country in Western Europe. Its capital city is Paris.",
    "answer": "The capital of France is Paris."
  }
}'
```
