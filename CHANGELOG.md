# Changelog

All notable changes to Themis will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.0] - 2026-03-13

### Added
- **Conversation Grouping**: Track multi-turn agent interactions across all entry points
  - Optional `conversation_id` field to group related evaluations
  - `agent_name` and `agent_version` fields for agent metadata tracking
  - Database schema migration with `conversation_id` column and indexes
  - New API endpoints:
    - `GET /api/v1/conversations` - List all conversations with summary metrics
    - `GET /api/v1/conversations/{id}` - Get conversation turns with detailed evaluations
  - Storage layer methods: `GetConversation()` and `ListConversations()`
  - Conversation summary aggregates: turn count, avg confidence, verdict distribution
- **Dashboard Conversations Tab**: Multi-turn conversation visualization
  - Tab navigation between Results and Conversations views
  - Conversation list with summary cards (turn count, avg confidence, verdict breakdown)
  - Drill-down view for individual conversation turns
  - Clickable turns that navigate to full evaluation in Results tab
  - URL-based navigation with browser back/forward support
  - Real-time stats: total conversations, avg turns, avg confidence
- **MCP Conversation Support**: Added `get_conversation` tool
  - Accepts `conversation_id`, `agent_name`, `agent_version` in evaluation tools
  - Returns full conversation detail with all turns chronologically
  - In-memory storage by default (data persists during MCP session)

### Changed
- **API Response Enhancement**: `GET /api/v1/conversations/{id}` now includes:
  - `avg_confidence` - Average confidence across all turns
  - `agent_name` - Agent name from first turn
  - `agent_version` - Agent version from first turn
- **Type Consolidation**: Unified conversation response types
  - Created `storage.ConversationDetail` as single source of truth
  - MCP and API now share same core types
  - Eliminated duplicate calculations between handlers

### Fixed
- **PostgreSQL Password Bug**: Fixed docker-compose.yml using wrong env var
  - Changed `POSTGRES_PASSWORD=${THEMIS_DB_DATABASE}` to `${THEMIS_DB_PASSWORD}`
- **API 404 Handling**: Non-existent conversations now return 404 with error message
  - Previous behavior incorrectly returned 200 with empty result
  - Updated integration tests to expect proper REST semantics
- **Batch CLI Database Integration**: Added conversation field mapping
  - Fixed `processor.go` to pass `ConversationID`, `AgentName`, `AgentVersion` to context
  - Results now properly saved with conversation metadata
- **Streaming Consumer Normalization**: Fixed normalize() function
  - Added missing `ConversationID`, `AgentName`, `AgentVersion` fields
  - Now matches API handler normalize() behavior

### Documentation
- Updated README.md with MCP conversation tracking examples
- Updated CLAUDE.md with conversation API endpoints and dashboard features
- Added 5 MCP test cases for conversation tracking (Test Cases 24-28)
- Updated `docs/testing/mcp-tests.md` with conversation examples
- Updated `docs/testing/batch-tests.md` with conversation_id examples
- Updated `specs/conversation-grouping.md` to 100% complete (7/7 phases)
- Created `resources/conversation_example.jsonl` with multi-turn examples

### Migration Notes
- **Database Migration Required**: Run migrations before upgrading
  ```bash
  migrate -path ./migrations -database "$THEMIS_DB_URL" up
  ```
- **Backward Compatible**: All conversation fields are optional
  - Existing evaluations without `conversation_id` continue working
  - No breaking changes to existing API endpoints or request formats
- **SQLite Auto-Migration**: In-memory SQLite automatically applies schema
  - No manual migration needed if using `IN_MEMORY_DB=true`

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
