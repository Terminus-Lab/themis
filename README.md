# Themis

AI response evaluation service with MCP and API interfaces. Uses configurable LLM judges to assess response quality across multiple dimensions.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![Documentation](https://img.shields.io/badge/docs-complete-green)](docs/INDEX.md)

**Deploy as**: HTTP API • MCP Server • CLI • Redis Consumer

## Quick Start

### HTTP API

```bash
# Start server
go run cmd/api/main.go

# Evaluate response
curl -X POST http://localhost:18082/api/v1/evaluate \
  -d '{"event_id":"test","agent":{"name":"my-agent"},"interaction":{"user_query":"What is AI?","answer":"Artificial intelligence"}}'

# View dashboard
open http://localhost:18082
```

### MCP Server

```bash
# Build
go build -o bin/themis-mcp cmd/mcp/main.go

# Add to Claude Code
claude mcp add --transport stdio themis -- ./bin/themis-mcp

# Use in conversation
"Evaluate this answer: ..."
```

### CLI Batch

```bash
go run cmd/batch/main.go -input dataset.jsonl -output results.jsonl -workers 10
```

## How It Works

Themis evaluates AI responses using a two-stage pipeline:

**Stage 1 (Prechecks)**: Fast heuristics check length, overlap, and format. Responses scoring below 0.2 exit early, skipping expensive LLM calls.

**Stage 2 (LLM Judges)**: Parallel evaluation across quality dimensions (relevance, faithfulness, coherence, completeness, instruction-following, correctness). Each judge runs concurrently using configurable LLM providers.

### Representative Commands

**Full evaluation**:
```bash
curl -X POST http://localhost:18082/api/v1/evaluate -d '{
  "event_id": "eval-001",
  "agent": {"name": "chatbot", "version": "1.0"},
  "interaction": {
    "user_query": "What is the capital of France?",
    "context": "France is a country in Western Europe. Paris is its capital.",
    "answer": "The capital of France is Paris."
  }
}'
```

**Response**:
```json
{
  "id": "eval-001",
  "confidence": 0.92,
  "verdict": "pass",
  "stages": [
    {"name": "length-checker", "score": 1.0},
    {"name": "relevance-judge", "score": 0.95}
  ]
}
```

**Single judge** (faster):
```bash
curl -X POST http://localhost:18082/api/v1/evaluate/judge/relevance -d '{...}'
```

**Query results**:
```bash
curl "http://localhost:18082/api/v1/results?agent_name=chatbot&verdict=fail"
```

**Batch processing with validation**:
```bash
# Process dataset
go run cmd/batch/main.go -input data.jsonl -output results.jsonl

# Validate judges against human annotations (Kendall's tau)
go run cmd/batch/main.go -input annotated.jsonl -validate -correlation-threshold 0.3
```

## Configuration

Create `.env` with LLM provider credentials:

```env
# OpenAI Platform (simplest)
OPEN_AI_KEY=sk-proj-...

# OR AWS Bedrock
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...

# OR Azure OpenAI
AZURE_OPENAI_ENDPOINT=https://...
OPEN_AI_KEY=...
```

Define judges in `configs/judges.yaml`:

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
        modelID: us.anthropic.claude-3-5-sonnet-20241022-v2:0
      prompt: |
        Evaluate if the answer addresses the user query...
```

Mix different LLM providers in the same pipeline. Each judge can use a different model.

## Key Features

**Integration**
- HTTP API with REST endpoints
- MCP protocol for AI assistants (Claude Code, Claude Desktop, Cursor)
- Redis Streams for async evaluation
- CLI for batch processing
- Web dashboard for visual monitoring

**Evaluation**
- Two-stage pipeline (fast prechecks, then parallel LLM judges)
- Early exit optimization (saves 80% cost on poor responses)
- Six quality dimensions (relevance, faithfulness, coherence, completeness, instruction, correctness)
- Four aggregation methods (weighted_average, harmonic_mean, median, weighted_product)
- Judge validation with Kendall's tau correlation

**Flexibility**
- Multi-provider LLM support (AWS Bedrock, Azure OpenAI, OpenAI Platform)
- YAML-driven judge configuration (edit prompts without code changes)
- Configurable thresholds and weights
- SQLite default, PostgreSQL optional
- Horizontal scaling (multiple Redis consumers)

**Production-Ready**
- Prometheus metrics endpoint
- Structured logging (zerolog)
- Query API for retrieving past results
- Graceful shutdown
- Docker support

## Documentation

| Document | Description |
|----------|-------------|
| [Quick Start](docs/getting-started/quick-start.md) | 5-minute tutorial |
| [Installation](docs/getting-started/installation.md) | Prerequisites and setup |
| [Configuration](docs/getting-started/configuration.md) | Environment variables and judges.yaml |
| [API Mode](docs/deployment/api-mode.md) | HTTP API deployment |
| [MCP Mode](docs/deployment/mcp-mode.md) | AI assistant integration |
| [Batch Mode](docs/deployment/batch-mode.md) | CLI batch processing |
| [Streaming Mode](docs/deployment/streaming-mode.md) | Redis consumer deployment |
| [API Tests](docs/testing/api-tests.md) | HTTP endpoint test cases |
| [Architecture](docs/architecture/) | Pipeline design and judges |
| [Full Index](docs/INDEX.md) | Complete documentation index |

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

- Report bugs or request features via [GitHub Issues](https://github.com/Terminus-Lab/themis/issues)
- Review [Code of Conduct](CODE_OF_CONDUCT.md) before contributing
- Check [Security Policy](SECURITY.md) for vulnerability reporting

## License

[MIT License](LICENSE)
