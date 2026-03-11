# Themis

AI agent evaluation service using configurable LLM judges and statistical validation.

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)

**Build Commands**

```bash
# API Server (HTTP endpoints + web dashboard)
go build -o bin/themis-api cmd/api/main.go

# MCP Server (Claude Code/Desktop integration)
go build -o bin/themis-mcp cmd/mcp/main.go

# CLI (batch processing)
go build -o bin/themis-batch cmd/batch/main.go
```

## Purpose

Themis evaluates AI agent responses through a two-stage pipeline: fast prechecks (heuristics) followed by parallel LLM judge evaluation across six quality dimensions. The framework validates judge accuracy against human annotations using Kendall's τ correlation before production deployment.

**Primary deployment modes:**
- **API Mode**: HTTP server with REST endpoints, web dashboard, and optional Redis streaming for real-time evaluation of agent responses
- **MCP Mode**: Model Context Protocol server that integrates with Claude Code, Claude Desktop, and Cursor for interactive evaluation during development

Both modes share the same core evaluation engine and judge configuration, allowing teams to use API mode for production monitoring and MCP mode for development workflows.

## Features

- **Two-Stage Evaluation Pipeline**: Fast prechecks + parallel LLM judges with early exit optimization
- **Statistical Validation**: Kendall's τ correlation against human annotations to ensure judge accuracy
- **Multi-Provider LLM Support**: Mix AWS Bedrock, Azure OpenAI, and OpenAI Platform models in same pipeline
- **YAML-Driven Configuration**: Edit judge prompts and models without code changes
- **Multiple Deployment Options**: HTTP API, MCP server, CLI batch processor, Redis streaming consumer
- **Query & Storage**: SQLite (default) or PostgreSQL with filtering by agent, verdict, timestamp
- **Web Dashboard**: Real-time visualization with dark terminal theme
- **Production-Ready**: Prometheus metrics, structured logging, graceful shutdown, horizontal scaling

## Getting Started

Download pre-built binaries from [GitHub Releases](https://github.com/Terminus-Lab/themis/releases):

**API Server Release** (themis-api):
- HTTP REST API server
- Web dashboard at `http://localhost:18082`
- Optional Redis streaming consumer
- Prometheus metrics endpoint

**MCP Server Release** (themis-mcp):
- Model Context Protocol server
- Integrates with Claude Code, Claude Desktop, Cursor
- Stdio transport for AI assistant tools

**Prerequisites**: Configure LLM provider credentials in `.env` file and edit `configs/judges.yaml` for judge definitions. See [Configuration](#configuration) section below.

For detailed setup instructions, see [Installation Guide](docs/getting-started/installation.md) and [Quick Start Tutorial](docs/getting-started/quick-start.md).

## Web Interface & API

The API server provides both a web dashboard and REST endpoints for evaluation and querying results.

**Start the server:**
```bash
./bin/themis-api
# or with streaming: STREAMING_ENABLED=true ./bin/themis-api
```

**Web Dashboard**: Navigate to `http://localhost:18082` for real-time visualization of evaluation results with filtering, pagination, and detailed inspection.

**API Endpoints:**
- `POST /api/v1/evaluate` - Full pipeline evaluation
- `POST /api/v1/evaluate/judge/{name}` - Single judge evaluation
- `GET /api/v1/results` - Query results with filters (agent_name, verdict, limit, offset)
- `GET /api/v1/results/{event_id}` - Get specific result by ID
- `GET /metrics` - Prometheus metrics

For detailed API examples and test cases, see [API Mode Documentation](docs/deployment/api-mode.md) and [API Test Cases](docs/testing/api-tests.md).

## MCP Integration

The MCP server exposes Themis evaluation as tools for Claude Code, Claude Desktop, and Cursor. Use natural language to evaluate agent responses during development.

**Installation:**
```bash
# Build MCP server
./bin/themis-mcp

# Add to Claude Code
claude mcp add --transport stdio themis -- /absolute/path/to/bin/themis-mcp

# Add to Claude Desktop (edit config.json)
{
  "mcpServers": {
    "themis": {
      "command": "/absolute/path/to/bin/themis-mcp"
    }
  }
}
```

**Usage in conversation:**
```
"Evaluate this response: User asked 'What is AI?' and agent answered 'Artificial Intelligence is...'"
```

For setup details and advanced usage, see [MCP Test Cases](docs/testing/mcp-tests.md).

## CLI Batch Processing

Process datasets offline with concurrent workers and validate judge accuracy against human annotations.

**Basic evaluation:**
```bash
./bin/themis-batch -input dataset.jsonl -output results.jsonl -workers 10
```

**Validation mode** (Kendall's τ):
```bash
./bin/themis-batch -input annotated.jsonl -validate -correlation-threshold 0.3
```

Input format: JSONL with `event_id`, `agent`, `interaction`, and optional `human_score` fields. For detailed examples and validation workflows, see [Batch Test Cases](docs/testing/batch-tests.md).

## Configuration

**Environment Variables** (`.env`):
```env
# LLM Provider (choose one or mix)
OPEN_AI_KEY=sk-proj-...                    # OpenAI Platform
AWS_REGION=us-east-1                       # AWS Bedrock
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...
AZURE_OPENAI_ENDPOINT=https://...          # Azure OpenAI

# Pipeline Settings
ENABLE_PRECHECK=true                       # Stage 1 prechecks
JUDGE_AGGREGATION_METHOD=weighted_average  # Stage 2 aggregation
VERDICT_PASS_THRESHOLD=0.8                 # Pass if confidence > 0.8
VERDICT_REVIEW_THRESHOLD=0.5               # Review if > 0.5, else fail
```

**Judge Configuration** (`configs/judges.yaml`):
```yaml
judges:
  default_model:
    modelFamily: "openai_platform"
    modelID: gpt-4o-mini
  evaluators:
    - name: relevance
      enabled: true
      weight: 0.25
      model:
        modelFamily: "anthropic"
        modelID: claude-3-5-sonnet-20241022
      prompt: "Evaluate if the answer addresses the query..."
```

Each judge can use a different LLM provider and model. For complete configuration reference, see [Configuration Guide](docs/getting-started/configuration.md).

## License

[Apache License 2.0](LICENSE)
