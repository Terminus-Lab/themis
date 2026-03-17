# Installation

## Prerequisites

**Required — at least one LLM provider:**
- OpenAI Platform API key (simplest setup)
- AWS Bedrock access (for Claude models)
- Azure OpenAI (for Azure-hosted GPT)

**Optional:**
- Redis — for streaming mode (`STREAMING_ENABLED=true`)
- PostgreSQL — for persistent storage (SQLite is used by default)
- Docker — for containerized MCP deployment
- Go 1.24+ — only needed if building from source

---

## Method 1: Pre-Built Binaries (Recommended)

Download the latest release from [GitHub Releases](https://github.com/Terminus-Lab/themis/releases/latest) and extract it.

```bash
# macOS (Apple Silicon)
tar -xzf themis_*_darwin_arm64.tar.gz && cd themis_*_darwin_arm64

# macOS (Intel)
tar -xzf themis_*_darwin_amd64.tar.gz && cd themis_*_darwin_amd64

# Linux (x64)
tar -xzf themis_*_linux_amd64.tar.gz && cd themis_*_linux_amd64

# Linux (ARM)
tar -xzf themis_*_linux_arm64.tar.gz && cd themis_*_linux_arm64

# Windows (PowerShell)
Expand-Archive themis_*_windows_amd64.zip && cd themis_*_windows_amd64
```

**⚠️ Always run binaries from the extracted directory** — `configs/judges.yaml` must be present alongside the binary.

Configure credentials:
```bash
cp .env.example .env
# Edit .env — minimum required: one LLM provider key
```

---

## Method 2: Build from Source

```bash
git clone https://github.com/Terminus-Lab/themis.git
cd themis
go mod download

go build -o bin/themis-api cmd/api/main.go
go build -o bin/themis-mcp cmd/mcp/main.go
go build -o bin/themis-cli cmd/batch/main.go

cp .env.example .env
```

---

## Database Setup

### SQLite (Default)

Zero configuration. Set in `.env`:
```env
IN_MEMORY_DB=true  # default
```

Data is held in memory for the duration of the service run.

### PostgreSQL (Production)

```bash
export THEMIS_DB_URL=postgresql://user:pass@localhost:5432/themis?sslmode=disable

# Start PostgreSQL
docker compose up --build themis-db -d

# Run migrations
brew install golang-migrate
migrate -path ./migrations -database "$THEMIS_DB_URL" up

# Update .env
echo "IN_MEMORY_DB=false" >> .env
echo "THEMIS_DB_URL=$THEMIS_DB_URL" >> .env
```

---

## Docker (MCP Mode)

```bash
docker build -t themis-mcp .
docker run --env-file .env themis-mcp

# Add to Claude Code
claude mcp add --transport stdio --scope project themis \
  -- docker run -i --rm --env-file .env themis-mcp
```

---

## Verify Installation

```bash
# Start the API server
./bin/themis-api

# In another terminal
curl -s http://localhost:18082/api/v1/health | jq .
# {"status":"ok","version":"1.0.0"}
```

---

## Troubleshooting

**Port in use:**
```env
EVAL_AGENT_API_PORT=18083
```

**AWS credentials error:**
```bash
aws bedrock list-foundation-models --region us-east-1
```

**OpenAI key issues:**
```bash
curl https://api.openai.com/v1/models -H "Authorization: Bearer $OPEN_AI_KEY"
```

**judges.yaml not found** — ensure the file exists at one of:
1. Path in `JUDGES_CONFIG_PATH` env var
2. `./configs/judges.yaml` (relative to binary)

---

## Next Steps

- [Quick Start](quick-start.md) — Run your first evaluation
- [Configuration](configuration.md) — Tune judges and thresholds
