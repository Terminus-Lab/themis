# Judge Development CLI - Final Proposal

**Status:** Proposal
**Date:** 2026-03-12

---

## 1. Purpose

### Problem
Currently, developing judges in Themis requires:
1. **Manual prompt writing** in `configs/judges.yaml` (time-consuming, inconsistent quality)
2. **Trial-and-error iteration** to improve prompts (no systematic approach)
3. **Manual validation** against human annotations (ad-hoc process)

This makes judge development slow, error-prone, and requires deep prompt engineering expertise.

### Solution
Add **author** and **calibrate** commands to Themis CLI to enable:

1. **LLM-assisted prompt generation** from task descriptions (faster, consistent)
2. **Data-driven prompt optimization** using ground truth data (systematic improvement)
3. **Built-in validation workflow** with quality metrics (τ, κ, agreement rate)

### Inspiration
The [llm-judge SDK](https://github.com/povarna/llm-judge) (Python) successfully implements this workflow:
```
author() → calibrate() → run() → validate → re-calibrate
```

We're adapting this pattern to Themis (Go) while maintaining Themis's production-focused architecture.

### Key Insight
**llm-judge** = development tool for creating judges
**Themis** = production system for running judges at scale

By adding author/calibrate to Themis, we get **both** in a single tool.

---

## 2. Architecture

### Current Structure
```
cmd/
├── api/main.go         # HTTP API server
├── batch/main.go       # Batch CLI (flag-based)
├── mcp/main.go         # MCP server
└── producer/main.go    # Test data generator
```

### Proposed Structure
```
cmd/
├── api/main.go         # HTTP API server (unchanged)
├── mcp/main.go         # MCP server (unchanged)
├── producer/main.go    # Test producer (unchanged)
└── themis/             # RENAMED from cmd/batch, enhanced with cobra
    ├── main.go         # Entry point with subcommands
    └── commands/
        ├── run.go      # Batch evaluation (current cmd/batch logic)
        ├── author.go   # NEW: Generate judge prompts
        ├── calibrate.go # NEW: Improve judge prompts
        └── validate.go # NEW: Validate judge accuracy

internal/
├── author/             # NEW: Authoring logic
│   ├── author.go       # LLM prompt generation
│   └── templates.go    # Meta-prompts
├── calibrate/          # NEW: Calibration logic
│   ├── calibrate.go    # Orchestration
│   ├── tuner.go        # LLM single-pass tuning
│   └── analyzer.go     # Error analysis
└── batch/              # Existing (unchanged)
    ├── processor.go
    ├── reader.go
    ├── writer.go
    └── validator.go
```

### Changes
1. **Rename:** `cmd/batch/` → `cmd/themis/`
2. **Add:** Cobra for subcommand routing
3. **Migrate:** Current batch logic to `commands/run.go`
4. **Create:** New `author` and `calibrate` commands
5. **Build:** Single `themis` binary

---

## 3. CLI Mapping

### Build
```bash
# Build single binary
go build -o themis cmd/themis/main.go

# Or use Makefile
make build-themis
```

### Commands

#### `themis run` - Batch Evaluation (Current Functionality)
```bash
themis run \
  --input data.jsonl \
  --output results.jsonl \
  --workers 5 \
  --format jsonl \
  --continue-on-error
```

**Purpose:** Run batch evaluations on dataset
**Same as:** Current `go run cmd/batch/main.go`

**Flags:**
- `--input` - Input JSONL file (required)
- `--output` - Output JSONL file (default: stdout)
- `--workers` - Concurrent workers (default: 5)
- `--format` - Output format: `jsonl` or `summary` (default: `jsonl`)
- `--summary` - Separate summary file path
- `--continue-on-error` - Continue on failures (default: true)
- `--dry-run` - Validate input without evaluating

---

#### `themis author` - Generate Judge Prompt (NEW)
```bash
themis author \
  --judge-name empathy \
  --description "Evaluate empathy in customer support responses" \
  --instructions "Look for acknowledgment of frustration, validating language" \
  --labels "empathetic:1.0,neutral:0.5,dismissive:0.0" \
  --output configs/judges.yaml
```

**Purpose:** Generate judge prompt from task description using LLM

**Flow:**
1. Takes human-provided guidelines (description, instructions, labels)
2. Constructs meta-prompt asking LLM to create evaluation prompt
3. LLM generates structured judge prompt
4. Appends to `configs/judges.yaml` (or outputs to file)

**Flags:**
- `--judge-name` - Judge name (required)
- `--description` - Task description (required)
- `--instructions` - Evaluation instructions (optional)
- `--labels` - Labels in format `label:score,label:score` (required)
- `--requires-context` - Judge needs retrieved context (default: false)
- `--requires-expected-output` - Judge needs ground truth (default: false)
- `--weight` - Judge weight in aggregation (default: 0.15)
- `--model-family` - LLM provider: `anthropic`, `openai`, `openai_platform` (default: `anthropic`)
- `--model-id` - Model ID (default: `claude-3-5-haiku`)
- `--output` - Output file (default: append to `configs/judges.yaml`)
- `--review` - Output to temp file for manual review before adding to config

**Example Output:**
```yaml
# Appended to configs/judges.yaml
- name: empathy
  enabled: true
  weight: 0.15
  description: "Evaluates empathy in customer support responses"
  requires_context: false
  prompt: |
    You are an evaluation judge for empathy assessment.

    Evaluate the customer support response for empathetic language.
    Look for:
    - Acknowledgment of customer frustration or concern
    - Validating language ("I understand", "That must be frustrating")
    - Personal, human tone

    Query: {{.Query}}
    Answer: {{.Answer}}

    Score from 0.0 to 1.0:
    - 1.0 (empathetic): Clear acknowledgment and validation
    - 0.5 (neutral): Professional but lacking empathy
    - 0.0 (dismissive): Robotic, cold, or dismissive tone

    Respond ONLY in raw JSON:
    {"score": <float>, "reason": "<string>"}
  model:
    modelFamily: anthropic
    modelID: us.anthropic.claude-3-5-haiku-20241022-v1:0
    max_tokens: 256
    temperature: 0.0
    retry: true
```

---

#### `themis validate` - Check Judge Accuracy (NEW)
```bash
themis validate \
  --input ground_truth.jsonl \
  --judge empathy \
  --threshold 0.3
```

**Purpose:** Validate judge accuracy against human annotations

**Flow:**
1. Reads JSONL with `human_annotation` field
2. Runs evaluation on all records
3. Computes correlation metrics:
   - Kendall's τ (rank correlation)
   - Cohen's Kappa (agreement)
   - Agreement Rate (exact matches)
4. Outputs validation report
5. Exits with error if τ < threshold

**Flags:**
- `--input` - Ground truth JSONL with `human_annotation` field (required)
- `--judge` - Specific judge to validate (optional, validates all if omitted)
- `--threshold` - Kendall's τ threshold (default: 0.3)
- `--output` - Validation report file (default: `validation-report.json`)

**Input Format:**
```jsonl
{"event_id": "1", "query": "...", "answer": "...", "human_annotation": "pass"}
{"event_id": "2", "query": "...", "answer": "...", "human_annotation": "fail"}
```

**Output:**
```json
{
  "passed": true,
  "total_records": 150,
  "agreement_count": 125,
  "agreement_rate": 0.833,
  "kendalls_tau": 0.42,
  "cohens_kappa": 0.38,
  "threshold": 0.3,
  "interpretation": "Moderate positive correlation"
}
```

---

#### `themis calibrate` - Improve Judge Prompt (NEW)
```bash
themis calibrate \
  --judge-name empathy \
  --ground-truth calibration_data.jsonl \
  --method llm_single_pass \
  --threshold 0.3 \
  --update-config
```

**Purpose:** Improve judge prompt using ground truth data

**Flow:**
1. Loads current prompt from `configs/judges.yaml`
2. Splits ground truth: 80% train, 20% test
3. Runs inference on train set with current prompt
4. Analyzes errors (false positives, false negatives)
5. Constructs meta-prompt with error analysis
6. LLM generates improved prompt
7. Validates improved prompt on test set
8. If test τ ≥ threshold AND improved, updates config
9. Outputs calibration report

**Flags:**
- `--judge-name` - Judge to calibrate (required)
- `--ground-truth` - Ground truth JSONL with `label` field (required)
- `--method` - Tuning method: `llm_single_pass` (default, only option initially)
- `--threshold` - Kendall's τ threshold for success (default: 0.3)
- `--update-config` - Update `configs/judges.yaml` if successful (default: false)
- `--output` - Calibration report file (default: `calibration-report.json`)
- `--train-ratio` - Train/test split ratio (default: 0.8)

**Input Format:**
```jsonl
{"event_id": "1", "query": "...", "answer": "...", "label": "empathetic"}
{"event_id": "2", "query": "...", "answer": "...", "label": "neutral"}
{"event_id": "3", "query": "...", "answer": "...", "label": "dismissive"}
```

**Output:**
```json
{
  "judge_name": "empathy",
  "train_records": 120,
  "test_records": 30,
  "baseline_metrics": {
    "train_accuracy": 0.75,
    "test_kendalls_tau": 0.28,
    "test_cohens_kappa": 0.25,
    "test_agreement_rate": 0.70
  },
  "improved_metrics": {
    "train_accuracy": 0.88,
    "test_kendalls_tau": 0.42,
    "test_cohens_kappa": 0.39,
    "test_agreement_rate": 0.83
  },
  "improvement": {
    "tau_delta": 0.14,
    "kappa_delta": 0.14,
    "agreement_delta": 0.13
  },
  "passed": true,
  "threshold": 0.3,
  "config_updated": true,
  "improved_prompt": "..."
}
```

---

## 4. Complete Workflow

### End-to-End Example

```bash
# Step 1: Author new judge (15 min)
themis author \
  --judge-name tone \
  --description "Evaluate professional tone in responses" \
  --instructions "Flag casual language (hey, lol), require formal address (Mr./Ms.)" \
  --labels "professional:1.0,acceptable:0.7,casual:0.0" \
  --review

# Review generated prompt, edit if needed, then add to config
# Or use --output configs/judges.yaml to append directly

# Step 2: Initial validation with small dataset (1 hour - collect 100 samples)
themis validate \
  --input validation_set.jsonl \
  --judge tone \
  --threshold 0.3

# Output: τ = 0.28 (below threshold)
# Action: Review error cases, adjust guidelines, re-author

# Step 2b: Re-author with improved guidelines
themis author \
  --judge-name tone \
  --description "Evaluate professional tone" \
  --instructions "Professional: Mr./Ms. + last name, formal language. Casual: hey, lol, no worries" \
  --labels "professional:1.0,acceptable:0.7,casual:0.0"

# Step 2c: Re-validate
themis validate --input validation_set.jsonl --judge tone
# Output: τ = 0.42 (passes threshold ✓)

# Step 3: Calibrate with larger dataset (4 hours - collect 800 samples)
themis calibrate \
  --judge-name tone \
  --ground-truth calibration_data.jsonl \
  --method llm_single_pass \
  --threshold 0.3 \
  --update-config

# Output: Test τ = 0.51 (improved from 0.42 ✓)
# Config updated automatically

# Step 4: Deploy to production (5 min)
# Edit configs/judges.yaml to enable judge
# Restart service
go run cmd/api/main.go

# Or streaming mode
EVENTS_STREAMING_ENABLED=true go run cmd/api/main.go

# Step 5: Monitor in production
# Dashboard: http://localhost:18082

# Query results
curl "http://localhost:18082/api/v1/results?agent_name=support-agent&limit=100"

# Step 6: Re-calibrate after 3 months (if τ drops)
# Collect 200 new annotated samples
themis validate --input monthly_validation.jsonl --judge tone
# If τ = 0.27 (dropped), re-calibrate:

themis calibrate \
  --judge-name tone \
  --ground-truth updated_data.jsonl \
  --update-config
```

### Workflow Diagram

```
┌─────────────┐
│   Author    │  Generate prompt from guidelines
│   (15 min)  │  themis author --judge-name X --description "..."
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Validate   │  Check τ ≥ 0.3 with 100 samples
│   (1 hour)  │  themis validate --input validation.jsonl --judge X
└──────┬──────┘
       │
       ├─ τ < 0.3 ──> Review, adjust guidelines, re-author ──┐
       │                                                       │
       └─ τ ≥ 0.3 ──> Continue                               │
              │                                                │
              ▼                                                │
       ┌─────────────┐                                       │
       │  Calibrate  │  Improve prompt with 800 samples     │
       │  (4 hours)  │  themis calibrate --judge-name X     │
       └──────┬──────┘                                       │
              │                                               │
              ├─ Test τ < 0.3 OR not improved ──> Manual review ─┘
              │
              └─ Test τ ≥ 0.3 AND improved ──> Deploy
                     │
                     ▼
              ┌─────────────┐
              │   Deploy    │  Enable in judges.yaml, restart service
              │   (5 min)   │  go run cmd/api/main.go
              └──────┬──────┘
                     │
                     ▼
              ┌─────────────┐
              │   Monitor   │  Track metrics (dashboard, Prometheus)
              │  (ongoing)  │  http://localhost:18082
              └──────┬──────┘
                     │
                     ▼
              ┌─────────────┐
              │Re-calibrate │  If τ drops, collect new data
              │  (3 months) │  themis calibrate --judge-name X
              └─────────────┘
                     │
                     └──────> Back to Validate
```

---

## 5. Implementation Phases

### Phase 1: CLI Infrastructure (Week 1)
- [ ] Rename `cmd/batch/` to `cmd/themis/`
- [ ] Add cobra dependency: `go get github.com/spf13/cobra`
- [ ] Create `cmd/themis/commands/` directory
- [ ] Migrate current logic to `commands/run.go`
- [ ] Update `main.go` to use cobra with subcommands
- [ ] Update Makefile to build as `themis`
- [ ] Test: `themis run` works identically to old `cmd/batch`

**Deliverable:** `themis run` command working

### Phase 2: Author Command (Week 2)
- [ ] Create `internal/author/` package
- [ ] Implement `author.go` (LLM prompt generation logic)
- [ ] Create meta-prompt templates in `templates.go`
- [ ] Create `commands/author.go` with flags
- [ ] Add YAML appending logic (parse, add judge, write)
- [ ] Test with various judge types
- [ ] Document usage

**Deliverable:** `themis author` command working

### Phase 3: Validate Command (Week 2)
- [ ] Create `commands/validate.go`
- [ ] Extract validation logic from `cmd/batch/main.go` (already exists)
- [ ] Add Cohen's Kappa computation to `internal/batch/validator.go`
- [ ] Add Agreement Rate computation
- [ ] Enhance validation report with all metrics
- [ ] Test with sample data

**Deliverable:** `themis validate` command working with τ, κ, agreement rate

### Phase 4: Calibrate Command (Week 3)
- [ ] Create `internal/calibrate/` package
- [ ] Implement `calibrate.go` (orchestration)
- [ ] Implement `tuner.go` (LLM single-pass tuning strategy)
- [ ] Implement `analyzer.go` (error analysis)
- [ ] Create `commands/calibrate.go` with flags
- [ ] Add config update logic (read, replace prompt, write)
- [ ] Test end-to-end calibration loop
- [ ] Document usage

**Deliverable:** `themis calibrate` command working

### Phase 5: Documentation & Release (Week 4)
- [ ] Update all docs to use `themis` CLI
- [ ] Add examples to `docs/examples/`
- [ ] Update `CLAUDE.md` with new commands
- [ ] Create migration guide for existing users
- [ ] Update CI/CD to build `themis` binary
- [ ] Release notes
- [ ] Update GitHub releases

**Deliverable:** Full documentation, release artifacts

---

## 6. Success Criteria

### Before Deployment
- ✅ All commands have `--help` documentation
- ✅ Author generates valid YAML that parses correctly
- ✅ Validate computes τ, κ, agreement rate correctly
- ✅ Calibrate improves test τ on sample dataset
- ✅ All existing batch tests pass with `themis run`
- ✅ End-to-end workflow documented

### After Deployment
- ✅ Users can create new judge in < 30 minutes (author + validate)
- ✅ Calibration improves τ by ≥0.1 on average
- ✅ Documentation includes 3+ complete examples
- ✅ Zero breaking changes to existing API/batch users

---

## 7. Dependencies

### New Go Dependencies
```go
require (
    github.com/spf13/cobra v1.8.0  // CLI framework
)
```

### LLM Provider (Already Supported)
- AWS Bedrock Claude (for author/calibrate LLM calls)
- Azure OpenAI (alternative)
- OpenAI Platform (alternative)

### Data Requirements
For calibration:
- **Initial validation:** 100-200 annotated samples
- **Calibration:** 500-1000 annotated samples
- **Re-calibration:** 200+ new samples every 3 months

---

## 8. Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| LLM-generated prompts low quality | Medium | Human review step, `--review` flag outputs to temp file first |
| Calibration doesn't improve τ | Medium | Document as "best effort", manual prompt editing always available |
| Breaking changes to batch CLI | High | Keep backwards compatibility, `themis run` == old `cmd/batch` |
| Dependency on LLM availability | Medium | Graceful errors, retry logic, document manual fallback |
| Users confused by new commands | Low | Clear documentation, examples, `--help` for each command |

---

## 9. Comparison to Alternatives

### Alternative 1: Use llm-judge SDK Directly
**Pros:** Already implemented, tested
**Cons:** Python (Themis is Go), separate tool (fragmented UX), users need to learn two tools

**Decision:** Integrate into Themis for unified experience

### Alternative 2: Manual Prompt Engineering Only
**Pros:** No new code, simple
**Cons:** Slow, inconsistent quality, requires expertise

**Decision:** Automation significantly improves developer experience

### Alternative 3: External Calibration Service
**Pros:** Separation of concerns
**Cons:** Extra infrastructure, API calls, complexity

**Decision:** CLI integration is simpler for users

---

## 10. Open Questions

### Q1: Should `themis author` require `--labels` or infer from description?
**Proposal:** Require explicit labels initially. Future: Add `--infer-labels` flag to let LLM suggest labels.

### Q2: Should calibration support multiple tuning methods?
**Proposal:** Start with `llm_single_pass` only. Add DSPy integration later if demand exists.

### Q3: Should `themis run` be the default command?
**Proposal:** Yes. If user types `themis --input data.jsonl`, treat as `themis run --input data.jsonl`.

### Q4: How to version judge prompts?
**Proposal:** Store in git. Future: Add `themis history --judge-name X` to show prompt evolution.

### Q5: Should we support batch author (multiple judges at once)?
**Proposal:** Not initially. Add `themis author --batch judges.csv` later if needed.

---

## 11. Timeline

**Week 1 (CLI Infrastructure):** Cobra integration, migrate batch to `run` subcommand
**Week 2 (Author + Validate):** Implement author and validate commands
**Week 3 (Calibrate):** Implement calibration logic
**Week 4 (Docs + Release):** Documentation, testing, release

**Total:** 4 weeks to production-ready CLI

---

## 12. Next Steps

1. **Approve proposal** (this document)
2. **Create GitHub issues** for each phase
3. **Prototype author command** (quick validation)
4. **Begin Phase 1** (CLI infrastructure)
5. **Iterate based on feedback**

---

## Appendix: Command Reference

```bash
# Build
go build -o themis cmd/themis/main.go

# Run batch evaluations
themis run --input data.jsonl --output results.jsonl

# Generate judge prompt
themis author --judge-name X --description "..." --labels "..."

# Validate judge accuracy
themis validate --input ground_truth.jsonl --judge X --threshold 0.3

# Improve judge prompt
themis calibrate --judge-name X --ground-truth data.jsonl --update-config

# Help
themis --help
themis author --help
themis calibrate --help
themis validate --help
themis run --help
```
