# Changelog

All notable changes to Themis will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.0.1] - 2026-03-25

### Added
- **Two-phase conversation evaluation pipeline**
  - Phase A: per-turn judges (`relevance`, `coherence`, `completeness`) run concurrently per turn
  - Phase B: holistic `conversation-flow` judge evaluates the full conversation
  - Final score: `α × holistic_score + (1-α) × turn_avg` (α = `CONVERSATION_HOLISTIC_WEIGHT`, default 0.5)
- **Multi-provider LLM support** via registry pattern: AWS Bedrock (`anthropic`), Azure OpenAI (`openai`), OpenAI Platform (`openai_platform`)
- **YAML-driven judge configuration** (`configs/judges.yaml`) — add or modify judges without code changes
- **Four entry points** sharing the same core evaluation logic:
  - `cmd/api` — REST API with go-restful; endpoints for evaluate, list/get conversations, health metrics, dashboard UI
  - `cmd/batch` — CLI batch processor (`themis-cli`) with JSONL input/output and concurrent worker pool
  - `cmd/mcp` — Model Context Protocol server for Claude Code / Claude Desktop / Cursor integration
  - `cmd/producer` — Redis stream producer for test data generation
- **Conversation streaming** (`CONVERSATION_STREAMING_ENABLED=true`): Redis Streams consumer runs in the same process as the API
- **Batch CLI** (`themis-cli`) with Cobra subcommands:
  - `evaluate` — process JSONL conversation datasets with configurable worker count (`THEMIS_BATCH_WORKERS`)
  - `validate` — stub for upcoming judge accuracy validation against human annotations
  - `version` — print CLI version
  - Flags: `-i/--input`, `-o/--output`, `-f/--format` (`jsonl` or `summary`), `-s/--summary`, `-d/--save-to-db`
- **Human annotation correlation** — when input records include `human_label` and/or `human_score`, the CLI automatically computes and appends a correlation report:
  - Kendall's τ-b (score rank correlation)
  - Cohen's κ (label agreement, chance-corrected)
  - Weighted κ (ordinal severity — penalises fail↔pass more than adjacent-class errors)
  - 3×3 confusion matrix (human rows, Themis columns)
- **Annotated sample dataset** (`resources/annotated_sample.jsonl`): 15 conversations (5 pass / 5 review / 5 fail) for smoke-testing correlation metrics
- **Storage backends**: SQLite in-memory (default, zero config) and PostgreSQL (production)
- **Dashboard UI** (`static/dashboard.html`) served at `GET /`
- **Structured logging** via zerolog; graceful shutdown on SIGINT/SIGTERM
- **Docker support** for MCP server deployment

[Unreleased]: https://github.com/Terminus-Lab/themis/compare/v0.0.1...HEAD
[0.0.1]: https://github.com/Terminus-Lab/themis/releases/tag/v0.0.1
