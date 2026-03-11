---
title: Installation
description: Prerequisites and setup instructions for Themis
version: 1.0.0
tags: [installation, setup, prerequisites, getting-started]
related:
  - getting-started/quick-start.md
  - getting-started/configuration.md
---

# Installation

## Prerequisites

### Required
- **Go 1.21+** - For building and running Themis
- **LLM Provider Access** - At least one of:
  - OpenAI Platform API key (recommended - simplest setup)
  - AWS Bedrock access (for Claude models)
  - Azure OpenAI access (for Azure-hosted GPT models)

### Optional
- **Redis** - For streaming mode (`STREAMING_ENABLED=true`)
- **PostgreSQL** - For production persistent storage (SQLite used by default)
- **Docker** - For containerized MCP deployment

## Installation Steps

### 1. Clone Repository

```bash
git clone https://github.com/Terminus-Lab/themis.git
cd themis
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Configure Environment

Copy the example environment file:

```bash
cp .env.example .env
```

Edit `.env` with your credentials (see [Configuration Guide](configuration.md) for details):

**Simplest Setup (OpenAI Platform)**:
```env
OPEN_AI_KEY=sk-proj-...  # Your OpenAI API key
```

**AWS Bedrock (for Claude)**:
```env
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your_key
AWS_SECRET_ACCESS_KEY=your_secret
```

**Azure OpenAI**:
```env
OPEN_AI_KEY=your_azure_key
AZURE_OPENAI_ENDPOINT=https://...openai.azure.com/...
```

### 4. Verify Installation

Run a quick test:

```bash
# Start API server
go run cmd/api/main.go

# In another terminal, test the endpoint
curl http://localhost:18082/
```

You should see the dashboard UI load successfully.

## Database Setup

### SQLite (Default - Recommended for Development)

**Zero configuration required** - SQLite runs in-memory by default:

```env
IN_MEMORY_DB=true  # Default setting
```

Evaluation results are stored during service runtime. No migration needed.

### PostgreSQL (Production)

For persistent storage in production:

```bash
# 1. Set up PostgreSQL
export THEMIS_DB_DATABASE=themis
export THEMIS_DB_USER=themis
export THEMIS_DB_PASSWORD=themis
export THEMIS_DB_HOST=localhost
export THEMIS_DB_PORT=5432
export THEMIS_DB_SSL_MODE=disable
export THEMIS_DB_URL=postgresql://${THEMIS_DB_USER}:${THEMIS_DB_PASSWORD}@${THEMIS_DB_HOST}:${THEMIS_DB_PORT}/${THEMIS_DB_DATABASE}?sslmode=${THEMIS_DB_SSL_MODE}

# 2. Start PostgreSQL (using Docker Compose)
docker compose up --build themis-db -d

# 3. Install migration tool
brew install golang-migrate

# 4. Run migrations
migrate -path ./migrations -database "$THEMIS_DB_URL" up

# 5. Update .env
echo "IN_MEMORY_DB=false" >> .env
echo "THEMIS_DB_URL=$THEMIS_DB_URL" >> .env
```

## Building Binaries

### API Server

```bash
go build -o bin/themis-api cmd/api/main.go
./bin/themis-api
```

### Batch CLI

```bash
go build -o bin/themis-batch cmd/batch/main.go
./bin/themis-batch -input dataset.jsonl -output results.jsonl
```

### MCP Server

```bash
go build -o bin/themis-mcp cmd/mcp/main.go
./bin/themis-mcp
```

## Docker Setup (MCP Mode)

Build Docker image for MCP integration:

```bash
docker build -t themis-mcp .
docker run --env-file .env themis-mcp
```

For Claude Code integration:

```bash
claude mcp add --transport stdio --scope project themis \
  --env AWS_REGION=us-east-1 \
  --env AWS_ACCESS_KEY_ID=your-key \
  --env AWS_SECRET_ACCESS_KEY=your-secret \
  -- docker run -i --rm --env-file .env themis-mcp
```

## Verification

### Test API Endpoint

```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-001",
    "event_type": "agent_response",
    "agent": {"name": "test-agent", "version": "1.0"},
    "interaction": {
      "user_query": "What is 2+2?",
      "answer": "4"
    }
  }'
```

Expected response:
```json
{
  "id": "test-001",
  "confidence": 0.85,
  "verdict": "pass",
  "stages": [...]
}
```

### Access Dashboard

Open browser to `http://localhost:18082` - you should see the evaluation dashboard.

## Troubleshooting

### Port Already in Use

Change the API port in `.env`:

```env
EVAL_AGENT_API_PORT=18083
```

### AWS Credentials Error

Verify your AWS credentials have Bedrock access:

```bash
aws bedrock list-foundation-models --region us-east-1
```

### OpenAI API Key Issues

Test your API key:

```bash
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPEN_AI_KEY"
```

### Judge Configuration Not Found

Ensure `judges.yaml` exists in one of:
1. Path specified by `JUDGES_CONFIG_PATH` env var
2. `./judges.yaml` (next to binary)
3. `./config/judges.yaml`

## Next Steps

- [Quick Start Guide](quick-start.md) - Run your first evaluation
- [Configuration Reference](configuration.md) - Detailed configuration options
- [API Mode](../deployment/api-mode.md) - Deploy HTTP API server
- [CLAUDE.md](../../CLAUDE.md) - Complete project documentation
