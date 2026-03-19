# Changelog

All notable changes to Themis will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.2.0] - 2026-03-19

### Added
- **`/evaluate` Slash Command Skill** (`evaluate.md` at project root): invoke `/evaluate` in any Claude Code or Codex session to evaluate the current conversation using `themis-cli evaluate-conversations`
  - Automatically extracts session turns, writes a temp JSONL, runs the CLI, and reports verdict + confidence inline
  - Supports `/evaluate N` to limit to last N turns
  - Falls back gracefully if binary or `configs/judges.yaml` not found
  - Full CLI command reference included in the skill file
- **`resources/conversations.jsonl`**: example conversation dataset (3 conversations: geography, machine learning, Python debugging) for `evaluate-conversations` testing

### Changed
- **CLI: renamed `evaluate` subcommand to `evaluate-events`** to distinguish single-turn event evaluation from multi-turn conversation evaluation (`evaluate-conversations`)
  - `themis-cli evaluate-events -i events.jsonl -o results.jsonl`
  - Breaking change for existing scripts using `themis-cli evaluate`
- **Dashboard: ChatGPT-style layout** (`static/dashboard.html`)
  - Left sidebar navigation replacing horizontal tabs
  - System sans-serif font replacing monospace
  - ChatGPT color palette: `#171717` sidebar, `#212121` main, `#10a37f` accent
  - SVG icons throughout: T-in-circle lettermark for brand, stroke-based nav icons, favicon
  - Browser tab favicon using inline SVG data URI (respects OS dark/light preference)
  - Console logging on all interactive elements (filters, pagination, tabs, row expand, conversation drill-down, auto-refresh, theme toggle)

### Documentation
- `docs/testing/batch-tests.md`: added Conversations Input Format section, updated all `evaluate` references to `evaluate-events`, added `resources/conversations.jsonl` reference to test cases
- `README.md`: added `/evaluate` slash command section with setup instructions for Claude Code and Codex, binary and `configs/` placement guide
- `README.md`: updated CLI batch processing examples to use `evaluate-events`

## [1.1.0] - 2026-03-18

### Added
- **Conversation Grouping**: Track multi-turn agent interactions across all entry points
  - `conversation_id` field to group related evaluations — **now mandatory** in all requests
  - `agent_name` and `agent_version` fields for agent metadata tracking
  - Database schema migration with `conversation_id TEXT NOT NULL` column and indexes
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
- **Validation Sampling APIs** — two endpoints for human annotation workflows:
  - `POST /api/v1/validation/sample/events/download` — samples individual event evaluations; returns JSONL with `event_id`, `agent`, `interaction` (no scores)
  - `POST /api/v1/validation/sample/conversations/download` — samples whole conversations; returns JSONL where each line is a full conversation with all turns grouped, for conversation-level annotation
  - Both support: `percentage` (1-100, default: 25), `min_size`, `max_size`, date range filters
  - Conversation sampling picks N distinct `conversation_id`s then returns all their turns — never returns partial conversations
- **Enhanced Validation Metrics**: Expanded judge validation beyond Kendall's τ
  - **Cohen's Kappa** - Categorical agreement accounting for chance (industry standard)
  - **Confusion Matrix** - Per-class error breakdown showing exactly where judges fail
  - **Per-class metrics** - Precision, recall, F1 for fail/review/pass classes
  - Validation now returns comprehensive metrics for debugging and reporting
- **Validation Test Datasets**: Production-ready validation examples
  - `resources/validation_test_dataset.jsonl` (150 records) - Successful validation example (τ=0.63, κ=0.91)
  - `resources/validation_failed_dataset.jsonl` (150 records) - Failed validation with adversarial patterns
  - Full interpretation guides with actionable recommendations for each dataset
- **Metrics Documentation**: Comprehensive guides for all validation metrics
  - `docs/metrics/cohens-kappa.md` - Agreement beyond chance explained with examples
  - `docs/metrics/confusion-matrix.md` - Matrix interpretation and common patterns
  - `docs/metrics/kendalls-tau.md` - Rank correlation for LLM judge validation
  - `docs/metrics/interpretation-guide.md` - Decision framework and troubleshooting guide

### Changed
- **`conversation_id` is now mandatory** across API, MCP, and CLI batch input
  - API returns `400 "conversation_id is required"` if omitted
  - MCP tools `evaluate_response` and `evaluate_single_judge` return error if omitted
  - DB schema enforces `NOT NULL` constraint
- **CLI Refactoring**: Migrated batch CLI to Cobra framework with subcommands
  - Renamed binary: `themis-batch` → `themis-cli`
  - New command structure: `themis-cli evaluate`, `themis-cli validate-events`, `themis-cli validate-conversations`
  - Moved worker count to environment variable: `THEMIS_BATCH_WORKERS` (default: 5)
  - Removed `--dry-run` flag (not useful for AI agents)
  - Removed `--continue-on-error` flag (always continues on errors now)
  - Removed `-w/--workers` flag (use `THEMIS_BATCH_WORKERS` env var instead)
  - Removed stdin/stdout support (always uses file paths for clarity)
  - Kept `-f/--format` flag for future format support (e.g., parquet)
  - Kept `-s/--summary` flag for separate summary file generation
  - Auto-generated help text with examples and shell completion support
  - Better error handling with proper exit codes
- **Batch CLI: DB persistence disabled by default** (`themis-cli evaluate`)
  - Results are no longer persisted to DB during batch evaluation by default
  - Use `--save-to-db` / `-d` flag to opt in (requires `IN_MEMORY_DB=false` and `THEMIS_DB_URL`)
  - Validation commands (`themis-cli validate-events`, `themis-cli validate-conversations`) always use in-memory DB — results are never persisted
- **API Response Enhancement**: `GET /api/v1/conversations/{id}` now includes:
  - `avg_confidence` - Average confidence across all turns
  - `agent_name` - Agent name from first turn
  - `agent_version` - Agent version from first turn
- **Type Consolidation**: Unified conversation response types
  - Created `storage.ConversationDetail` as single source of truth
  - MCP and API now share same core types
  - Eliminated duplicate calculations between handlers
- **Validation Output Format**: Enhanced JSON structure with 3-metric framework
  - `correlation_metrics` - Kendall's τ (PRIMARY - pass/fail decision)
  - `agreement_metrics` - Cohen's Kappa (SECONDARY - industry reporting)
  - `confusion_matrix` - Error breakdown (DIAGNOSTIC - debugging)
  - `per_class_metrics` - Precision/recall/F1 per verdict class

### Fixed
- **Azure OpenAI**: Fixed Azure OpenAI provider implementation
- **SQLite In-Memory Connection Pool**: Fixed "no such table: eval_results" error
  - `database/sql` connection pool created separate in-memory databases per connection
  - Fixed by setting `MaxOpenConns(1)` for `:memory:` databases to ensure all operations share one connection
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
- Updated all references from `themis-batch` to `themis-cli` across documentation
  - README.md, CLAUDE.md, installation guide, quick-start guide
  - `docs/testing/batch-tests.md` renamed title and updated all examples
  - Updated `.goreleaser.yaml` release notes
- Updated CLI command examples to use new subcommand structure
  - `themis-cli evaluate -i input.jsonl -o output.jsonl`
  - `themis-cli validate-events -i annotated-events.jsonl`
  - `themis-cli validate-conversations -i annotated-conversations.jsonl`
- Updated `docs/testing/batch-tests.md` with simplified flag reference
  - Removed Test Case 3 (dry-run validation) - no longer supported
  - Updated all examples to use `THEMIS_BATCH_WORKERS` env var
  - Renumbered test cases 3-12 accordingly
- Added `THEMIS_BATCH_WORKERS` environment variable documentation
- Updated README.md with MCP conversation tracking examples
- Updated CLAUDE.md with conversation API endpoints and dashboard features
- Added 5 MCP test cases for conversation tracking (Test Cases 24-28)
- Updated `docs/testing/mcp-tests.md` with conversation examples
- Updated `docs/testing/batch-tests.md` — `conversation_id` moved from optional to required field
- Updated `docs/testing/api-tests.md` — all request examples include `conversation_id`
- Updated `docs/getting-started/quick-start.md` — all curl examples include `conversation_id`
- Updated `specs/conversation-grouping.md` to 100% complete (7/7 phases)
- Created `resources/conversation_example.jsonl` with multi-turn examples

### Migration Notes
- **Database Migration Required**: Run migrations before upgrading
  ```bash
  migrate -path ./migrations -database "$THEMIS_DB_URL" up
  ```
- **Breaking Change**: `conversation_id` is now required in all evaluation requests
  - API: returns `400` if `conversation_id` is missing
  - CLI: batch input records without `conversation_id` will fail validation
  - MCP: `evaluate_response` and `evaluate_single_judge` tools require `conversation_id`
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

[Unreleased]: https://github.com/Terminus-Lab/themis/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/Terminus-Lab/themis/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/Terminus-Lab/themis/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/Terminus-Lab/themis/releases/tag/v1.0.0
