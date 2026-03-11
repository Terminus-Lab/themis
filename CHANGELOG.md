# Changelog

All notable changes to Themis will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-03-11

### Added
- Two-stage evaluation pipeline (prechecks + LLM judges)
- CI/CD workflows (test, lint, release)
- GoReleaser configuration for binary releases
- CHANGELOG.md for tracking releases
- Six quality dimensions: relevance, faithfulness, coherence, completeness, instruction, correctness
- Multi-provider LLM support (AWS Bedrock, Azure OpenAI, OpenAI Platform)
- HTTP API with REST endpoints
  - `POST /api/v1/evaluate` - Full evaluation pipeline
  - `POST /api/v1/evaluate/judge/{name}` - Single judge evaluation
  - `GET /api/v1/results` - Query evaluation results with filters
  - `GET /api/v1/results/{event_id}` - Get single result by ID
  - `GET /` - Web dashboard
- MCP server for Claude Code/Desktop integration
- Redis Streams consumer for async evaluation
- CLI batch processing with worker pools
- Kendall's τ validation against human annotations
- Query API with filtering and pagination
- Web dashboard for result visualization
- Four aggregation methods: weighted_average, harmonic_mean, median, weighted_product
- Early exit optimization (saves 80% LLM costs on poor responses)
- SQLite (in-memory) and PostgreSQL storage support
- YAML-driven judge configuration
- Docker support
- Comprehensive documentation
  - Getting started guides
  - Configuration reference
  - Deployment guides
  - Test case documentation
  - Security policy
  - Contributing guidelines

### Features

**Evaluation Pipeline**
- Fast heuristic prechecks (length, overlap, format)
- Parallel LLM judge execution with configurable timeout
- Automatic judge skip logic for missing required fields
- Configurable verdict thresholds (pass/review/fail)
- Multiple aggregation methods computed per request
- Confidence score calculation with stage weighting

**Integration**
- HTTP API with CORS support
- MCP protocol for AI assistants
- Redis Streams for async processing
- Unified API+Streaming mode
- Horizontal scaling with multiple consumers

**Flexibility**
- Mix different LLM providers in same pipeline
- Each judge can use different model
- YAML-driven prompts (no code changes needed)
- Configurable weights and thresholds
- Optional prechecks stage
- Multiple storage backends

**Production-Ready**
- Structured logging (zerolog)
- Graceful shutdown
- Query API for historical results
- Auto-refresh dashboard
- Error handling and retry logic
- Test coverage: 84-100% on core packages

### Fixed
- Early exit evaluations now stored in database (were previously skipped in storage)
- License inconsistency resolved (Apache 2.0 everywhere)
- **CRITICAL**: Enabled CGO in release builds to support SQLite (fixes "go-sqlite3 requires cgo" fatal error)
- Removed internal themis-producer from release binaries (not needed by end users)
- Added warning that binaries must run from extracted directory for configs/judges.yaml discovery

[Unreleased]: https://github.com/Terminus-Lab/themis/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/Terminus-Lab/themis/releases/tag/v1.0.0
