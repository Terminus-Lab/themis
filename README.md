# Themis

AI agent evaluation service using configurable LLM judges and statistical validation.

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://go.dev/)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/Terminus-Lab/themis)


**Build Commands**

```bash
# API Server (HTTP endpoints + web dashboard)
go build -o bin/themis-api cmd/api/main.go

# MCP Server (Claude Code/Desktop integration)
go build -o bin/themis-mcp cmd/mcp/main.go

# CLI (batch processing)
go build -o bin/themis-cli cmd/batch/main.go
```

## Purpose

Themis evaluates AI agent responses through a two-stage pipeline: fast prechecks (heuristics) followed by parallel LLM judge evaluation across six quality dimensions. The framework validates judge accuracy against human annotations using Kendall's τ correlation before production deployment.

**Primary deployment modes:**
- **API Mode**: HTTP server with REST endpoints, web dashboard, and optional Redis streaming for real-time evaluation of agent responses
- **MCP Mode**: Model Context Protocol server that integrates with Claude Code, Claude Desktop, and Cursor for interactive evaluation during development

Both modes share the same core evaluation engine and judge configuration, allowing teams to use API mode for production monitoring and MCP mode for development workflows.

## Security Notice

**Themis v1.0 is designed for internal/trusted network deployment.**
For complete security guidance, see [SECURITY.md](SECURITY.md) and [API Deployment Guide](docs/deployment/api-mode.md).

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

### Download Pre-Built Binaries

Download the latest release from [GitHub Releases](https://github.com/Terminus-Lab/themis/releases/latest).

**Choose your platform:**
- **macOS (Apple Silicon)**: `themis_1.0.0_darwin_arm64.tar.gz` - M1/M2/M3 Macs
- **macOS (Intel)**: `themis_1.0.0_darwin_amd64.tar.gz` - Intel-based Macs
- **Linux (x64)**: `themis_1.0.0_linux_amd64.tar.gz`
- **Linux (ARM)**: `themis_1.0.0_linux_arm64.tar.gz`
- **Windows (x64)**: `themis_1.0.0_windows_amd64.zip`

**Extract and setup:**
```bash
# macOS/Linux
tar -xzf themis_1.0.0_darwin_arm64.tar.gz
cd themis_1.0.0_darwin_arm64

# Windows (PowerShell)
Expand-Archive themis_1.0.0_windows_amd64.zip
cd themis_1.0.0_windows_amd64

# Configure environment
cp .env.example .env
# Edit .env with your LLM provider credentials
```

**⚠️ Important:** Always run binaries from the extracted directory where `configs/judges.yaml` is located.

**What's included:**
- **themis-api**: HTTP REST API server with web dashboard at `http://localhost:18082`
- **themis-mcp**: Model Context Protocol server for Claude Code/Desktop/Cursor integration
- **themis-cli**: CLI for batch processing and validation
- **configs/**: Judge configuration files (judges.yaml)
- **docs/**: Complete documentation
- **.env.example**: Environment template

**Prerequisites**: You'll need LLM provider credentials (AWS Bedrock, Azure OpenAI, or OpenAI Platform). Configure in `.env` file. See [Configuration](#configuration) section below.

For detailed setup instructions, see [Installation Guide](docs/getting-started/installation.md) and [Quick Start Tutorial](docs/getting-started/quick-start.md).

## Web Interface & API

The API server provides both a web dashboard and REST endpoints for evaluation and querying results.

**Start the server:**
```bash
./themis-api
# or with streaming: STREAMING_ENABLED=true ./themis-api
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

**Available Tools:**
- `evaluate_response` - Full pipeline evaluation with all judges
- `evaluate_single_judge` - Fast evaluation with single judge (relevance, faithfulness, etc.)
- `get_conversation` - Retrieve all turns for a conversation ID

**Conversation Tracking:**
All evaluation tools accept optional `conversation_id`, `agent_name`, and `agent_version` fields to group multi-turn interactions. Use `get_conversation` to view all turns and metrics for a conversation.

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

**Usage Examples:**
```
# Basic evaluation
"Evaluate this response: User asked 'What is AI?' and agent answered 'Artificial Intelligence is...'"

# Multi-turn conversation tracking
"Evaluate turn 1 for conversation conv-123: Query='What is Python?' Answer='Programming language'"
"Evaluate turn 2 for conversation conv-123: Query='Is it hard?' Answer='Easy to learn'"
"Show me conversation conv-123"
```

For setup details and advanced usage, see [MCP Test Cases](docs/testing/mcp-tests.md).

## CLI Batch Processing

Process datasets offline with concurrent workers and validate judge accuracy against human annotations.

**Basic evaluation:**
```bash
./bin/themis-cli -input dataset.jsonl -output results.jsonl -workers 10
```

**Validation mode** (Kendall's τ):
```bash
./bin/themis-cli -input annotated.jsonl -validate -correlation-threshold 0.3
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

### ⚠️ Judge Weights Configuration

**Important**: Default judge weights are starting values that **must be tuned** for your specific use case.

**Recommended workflow**:
1. **Collect human annotations** - Have domain experts label 25-50 sample evaluations
2. **Run validation** - Use batch mode with `-validate` to compute Kendall's τ correlation
3. **Adjust weights** - Modify judge weights in `configs/judges.yaml` based on correlation results
4. **Iterate** - Repeat until Kendall's τ ≥ 0.3 (acceptable agreement with human judgment)

See [Batch Test Cases](docs/testing/batch-tests.md) for validation workflow examples.

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
