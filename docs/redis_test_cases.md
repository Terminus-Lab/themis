# Redis Stream Consumer Mode

Advanced usage guide for running Themis with Redis Stream consumer for asynchronous evaluation.

## Overview

In unified API + Streaming mode, Themis:
- Runs HTTP API for synchronous requests and metrics
- Connects to a Redis instance
- Joins a consumer group (`eval-group`)
- Consumes messages from the `eval-events` stream
- Processes evaluation requests asynchronously
- Supports graceful shutdown and acknowledgment

## Use Cases

- **High-throughput evaluation**: Process evaluation requests asynchronously
- **Decoupled architecture**: Separate evaluation from agent response generation
- **Multiple consumers**: Scale horizontally with multiple Themis instances
- **Fault tolerance**: Redis Streams provide message persistence and redelivery
- **Observability**: Prometheus `/metrics` endpoint exposes streaming metrics

---

## Configuration

Add streaming configuration to your `.env`:

```env
# Enable streaming consumer alongside API
STREAMING_ENABLED=true

# Redis Stream Configuration
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_STREAM_KEY=eval-events
REDIS_CONSUMER_GROUP=eval-group
REDIS_CONSUMER_NAME=consumer-1  # Unique per instance
```

---

## Running the Consumer

Start Themis in unified API + Streaming mode:

```bash
STREAMING_ENABLED=true go run cmd/api/main.go
```

**Expected output:**
```
{"level":"info","message":"Streaming mode enabled - starting Redis consumer"}
{"level":"info","stream_key":"eval-events","consumer_group":"eval-group","consumer_name":"consumer-1","message":"Starting streaming consumer"}
{"level":"info","address":":18082","streaming_enabled":true,"message":"Starting Themis Server"}
```

The service will:
- Start HTTP API on port 18082
- Start streaming consumer in background
- Expose `/metrics` endpoint for Prometheus

**API is available while streaming runs:**
```bash
# Health check
curl http://localhost:18082/api/v1/health

# Manual evaluation
curl -X POST http://localhost:18082/api/v1/evaluate -d '{...}'

# Prometheus metrics
curl http://localhost:18082/metrics
```

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
