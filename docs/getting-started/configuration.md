---
title: Configuration
description: Complete configuration reference for Themis
version: 1.0.0
tags: [configuration, environment-variables, settings, judges-yaml]
related:
  - getting-started/installation.md
  - architecture/judges.md
  - architecture/aggregation.md
---

# Configuration

Themis configuration consists of two parts:
1. **Environment variables** (`.env` file) - Service settings and credentials
2. **Judges configuration** (`configs/judges.yaml`) - Judge definitions and prompts

## Environment Variables

### LLM Provider Credentials

Configure at least one LLM provider:

#### OpenAI Platform (Recommended - Simplest)

```env
OPEN_AI_KEY=sk-proj-...  # Standard OpenAI API key
```

#### AWS Bedrock (for Claude models)

```env
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your_key
AWS_SECRET_ACCESS_KEY=your_secret
```

#### Azure OpenAI (for Azure-hosted GPT)

```env
OPEN_AI_KEY=your_azure_openai_api_key
AZURE_OPENAI_ENDPOINT=https://...openai.azure.com/openai/deployments/...
```

**Note**: You can use multiple providers simultaneously by configuring all credentials.

### Service Configuration

```env
# API Server
EVAL_AGENT_API_PORT=18082                   # HTTP server port (default: 18082)

# Pipeline Behavior
ENABLE_PRECHECK=true                        # Enable Stage 1 prechecks (default: true)
EARLY_EXIT_THRESHOLD=0.2                    # Precheck early exit threshold (default: 0.2)

# Aggregation Weights
PRECHECK_WEIGHT=0.3                         # Stage 1 weight in final score (default: 0.3)
LLM_JUDGE_WEIGHT=0.7                        # Stage 2 weight in final score (default: 0.7)

# Verdict Thresholds
VERDICT_PASS_THRESHOLD=0.8                  # Confidence > this → "pass" (default: 0.8)
VERDICT_REVIEW_THRESHOLD=0.5                # Confidence > this → "review" (default: 0.5)
                                            # Confidence ≤ this → "fail"

# Stage 2 Aggregation
JUDGE_AGGREGATION_METHOD=weighted_average   # Options: weighted_average, harmonic_mean, median, weighted_product
```

### Database Configuration

```env
# SQLite (Default - Development)
IN_MEMORY_DB=true                           # Use in-memory SQLite (default: true)

# PostgreSQL (Production)
IN_MEMORY_DB=false                          # Disable in-memory mode
THEMIS_DB_URL=postgresql://user:pass@localhost:5432/themis?sslmode=disable
```

### Streaming Configuration

```env
# Redis Stream Consumer (optional)
STREAMING_ENABLED=false                     # Enable Redis consumer (default: false)
REDIS_ADDR=localhost:6379                   # Redis server address
REDIS_PASSWORD=                             # Redis password (optional)
REDIS_STREAM_KEY=eval-events                # Stream key name
REDIS_CONSUMER_GROUP=eval-group             # Consumer group name
REDIS_CONSUMER_NAME=consumer-1              # Unique consumer ID (for scaling)
```

### Judge Configuration Path

```env
# Judge Config Location (optional)
JUDGES_CONFIG_PATH=/path/to/judges.yaml     # Explicit path to judges.yaml
```

**Default search order** (if `JUDGES_CONFIG_PATH` not set):
1. `./judges.yaml` (next to binary)
2. `./config/judges.yaml`
3. `./configs/judges.yaml`

## Judges Configuration (judges.yaml)

### File Structure

```yaml
judges:
  # Default model for all judges (can be overridden per judge)
  default_model:
    modelFamily: "openai_platform"  # anthropic | openai | openai_platform
    modelID: gpt-4o-mini            # Model identifier
    max_tokens: 200                 # Token limit for response
    temperature: 0.0                # Temperature (0.0 = deterministic)
    retry: true                     # Retry on failure

  # Individual judge definitions
  evaluators:
    - name: relevance
      enabled: true
      description: "Evaluates if answer addresses the user query"
      weight: 0.25                  # Contribution to weighted average
      requires_context: false       # Skip if context missing
      requires_expected_output: false  # Skip if expected_output missing

      # Optional: Override model for this judge
      model:
        modelFamily: "anthropic"
        modelID: us.anthropic.claude-3-5-sonnet-20241022-v2:0
        max_tokens: 200
        temperature: 0.0
        retry: true

      # Evaluation prompt (Go template syntax)
      prompt: |
        You are an evaluation judge. Assess how well the answer addresses the user's query.

        User Query: {{.UserQuery}}
        Answer: {{.Answer}}
        {{if .Context}}Context: {{.Context}}{{end}}

        Score from 0.0 (irrelevant) to 1.0 (highly relevant).

        Respond in JSON:
        {"score": <float>, "reason": "<string>"}

    - name: faithfulness
      enabled: true
      description: "Evaluates if answer is grounded in provided context"
      weight: 0.25
      requires_context: true        # Auto-skip if no context provided
      prompt: |
        Evaluate if the answer is factually grounded in the context.
        Penalize hallucinations and unsupported claims.

        Context: {{.Context}}
        Answer: {{.Answer}}

        Score: 0.0 (hallucination) to 1.0 (fully grounded)

        Respond in JSON:
        {"score": <float>, "reason": "<string>"}

    - name: coherence
      enabled: true
      description: "Evaluates logical consistency"
      weight: 0.15
      prompt: |
        Evaluate logical consistency and internal coherence of the answer.

        Answer: {{.Answer}}

        Score: 0.0 (incoherent) to 1.0 (perfectly coherent)

        Respond in JSON:
        {"score": <float>, "reason": "<string>"}

    - name: completeness
      enabled: true
      description: "Evaluates if answer fully addresses the query"
      weight: 0.15
      prompt: |
        Evaluate if the answer completely addresses all aspects of the query.

        User Query: {{.UserQuery}}
        Answer: {{.Answer}}

        Score: 0.0 (incomplete) to 1.0 (comprehensive)

        Respond in JSON:
        {"score": <float>, "reason": "<string>"}

    - name: instruction
      enabled: true
      description: "Evaluates adherence to instructions"
      weight: 0.10
      prompt: |
        Check if the answer follows any specific instructions in the query.

        User Query: {{.UserQuery}}
        Answer: {{.Answer}}

        Score: 0.0 (violates instructions) to 1.0 (follows perfectly)

        Respond in JSON:
        {"score": <float>, "reason": "<string>"}

    - name: correctness
      enabled: false                # Disabled by default
      description: "Evaluates semantic similarity with expected output"
      weight: 0.10
      requires_expected_output: true  # Auto-skip if not provided
      prompt: |
        Compare the answer with the expected output (ground truth).
        Score based on semantic equivalence, not exact string match.

        Answer: {{.Answer}}
        Expected Output: {{.ExpectedOutput}}

        Guidelines:
        - 1.0: Semantically identical
        - 0.8-0.9: Mostly correct, minor differences
        - 0.5-0.7: Partially correct
        - 0.2-0.4: Somewhat related but different
        - 0.0-0.1: Completely different

        Respond in JSON:
        {"score": <float>, "reason": "<string>"}
```

### Configuration Principles

#### 1. Multi-Provider Support

Each judge can use a different LLM provider:

```yaml
evaluators:
  - name: relevance
    model:
      modelFamily: "anthropic"
      modelID: us.anthropic.claude-3-5-sonnet-20241022-v2:0

  - name: faithfulness
    model:
      modelFamily: "openai_platform"
      modelID: gpt-4o-mini

  - name: coherence
    model:
      modelFamily: "openai"  # Azure OpenAI
      modelID: gpt-4o
```

#### 2. Skip Logic

Judges with `requires_context: true` or `requires_expected_output: true` automatically skip if required field is missing:

```yaml
- name: faithfulness
  requires_context: true  # Skips if request has no context field

- name: correctness
  requires_expected_output: true  # Skips if request has no expected_output
```

This maintains backwards compatibility - existing requests work unchanged.

#### 3. Weighted Scoring

Judge weights control contribution to Stage 2 score (weighted_average method):

```yaml
- name: relevance
  weight: 0.25  # 25% of Stage 2 score

- name: faithfulness
  weight: 0.25  # 25% of Stage 2 score

- name: coherence
  weight: 0.15  # 15% of Stage 2 score
```

Weights are normalized automatically (don't need to sum to 1.0).

#### 4. Prompt Templates

Prompts use Go template syntax with available fields:
- `{{.UserQuery}}` - User's question
- `{{.Answer}}` - Agent's response
- `{{.Context}}` - Retrieved context (RAG)
- `{{.ExpectedOutput}}` - Ground truth (correctness judge)

Conditional rendering:
```
{{if .Context}}Context: {{.Context}}{{end}}
```

## Configuration Scenarios

### Scenario 1: Cost Optimization (Fast & Cheap)

```env
# .env
ENABLE_PRECHECK=true
EARLY_EXIT_THRESHOLD=0.3  # More aggressive early exit
JUDGE_AGGREGATION_METHOD=median  # Fast, no weights needed
```

```yaml
# judges.yaml
judges:
  default_model:
    modelFamily: "openai_platform"
    modelID: gpt-4o-mini  # Cheap model

  evaluators:
    - name: relevance
      enabled: true
    - name: faithfulness
      enabled: false  # Disable expensive judges
    - name: coherence
      enabled: true
    - name: completeness
      enabled: false
    - name: instruction
      enabled: false
```

### Scenario 2: High-Quality Evaluation (Best Accuracy)

```env
# .env
ENABLE_PRECHECK=false  # Skip prechecks, always use LLM
JUDGE_AGGREGATION_METHOD=weighted_average
VERDICT_PASS_THRESHOLD=0.9  # Strict passing criteria
```

```yaml
# judges.yaml
judges:
  default_model:
    modelFamily: "anthropic"
    modelID: us.anthropic.claude-3-5-sonnet-20241022-v2:0  # Best model

  evaluators:
    # All judges enabled with high-quality prompts
    - name: relevance
      enabled: true
      weight: 0.20
    - name: faithfulness
      enabled: true
      weight: 0.20
    # ... all judges enabled
```

### Scenario 3: Strict Quality Control

```env
# .env
JUDGE_AGGREGATION_METHOD=harmonic_mean  # Penalizes low scores
VERDICT_PASS_THRESHOLD=0.85
VERDICT_REVIEW_THRESHOLD=0.6
```

One low judge score significantly lowers final confidence.

### Scenario 4: Ground Truth Validation

```env
# .env
ENABLE_PRECHECK=false  # Always check correctness
```

```yaml
# judges.yaml
evaluators:
  - name: correctness
    enabled: true  # Enable correctness judge
    weight: 0.40   # High weight for accuracy
  - name: relevance
    enabled: true
    weight: 0.30
  - name: coherence
    enabled: true
    weight: 0.30
```

### Scenario 5: Multi-Provider Redundancy

Use multiple models for same evaluation dimension:

```yaml
evaluators:
  - name: relevance-claude
    description: "Relevance check with Claude"
    model:
      modelFamily: "anthropic"
      modelID: us.anthropic.claude-3-5-haiku-20241022-v1:0
    weight: 0.15
    prompt: |
      Evaluate relevance...

  - name: relevance-gpt
    description: "Relevance check with GPT"
    model:
      modelFamily: "openai_platform"
      modelID: gpt-4o-mini
    weight: 0.15
    prompt: |
      Evaluate relevance...
```

Both contribute to final score, providing redundancy and consensus.

## Configuration Best Practices

### 1. Start Simple, Iterate

Begin with:
- OpenAI Platform (easiest setup)
- Default thresholds
- All judges enabled
- `weighted_average` aggregation

Tune based on results.

### 2. Validate Judge Changes

Before deploying prompt changes:

```bash
# Collect human annotations (25+ samples)
# Run validation
go run cmd/batch/main.go \
  -input annotated_sample.jsonl \
  -validate \
  -correlation-threshold 0.3

# Deploy only if Kendall's τ ≥ 0.3
```

See [Validation Guide](../guides/validation.md) for details.

### 3. Experiment with Aggregation Methods

All 4 methods are computed and returned in `metrics`:

```json
{
  "metrics": {
    "stage2_weighted_avg": 0.85,
    "stage2_harmonic_mean": 0.82,
    "stage2_median": 0.87,
    "stage2_weighted_product": 0.79
  }
}
```

Compare methods without re-running evaluations.

### 4. Use Environment-Specific Configs

```bash
# Development
cp .env.dev .env
# Relaxed thresholds, fast models

# Staging
cp .env.staging .env
# Balanced settings

# Production
cp .env.prod .env
# Strict thresholds, best models, PostgreSQL
```

### 5. Monitor and Adjust Thresholds

Track verdict distribution in production:
- Too many `pass`: Increase `VERDICT_PASS_THRESHOLD`
- Too many `fail`: Decrease thresholds
- Too many `review`: Adjust `VERDICT_REVIEW_THRESHOLD` gap

## Configuration Locations

### Priority Order

1. **Environment variables** - Highest priority
2. **`.env` file** - Loaded automatically
3. **Default values** - Built-in defaults

### Judge Config Search Path

1. `$JUDGES_CONFIG_PATH` env var
2. `./judges.yaml`
3. `./config/judges.yaml`
4. `./configs/judges.yaml`

**Recommendation**: Use `configs/judges.yaml` (checked into git) and set `JUDGES_CONFIG_PATH` for overrides.

## Next Steps

- [Quick Start](quick-start.md) - Run your first evaluation
- [Pipeline Architecture](../architecture/pipeline.md) - Understand evaluation flow
- [Aggregation Methods](../architecture/aggregation.md) - Deep dive into scoring
- [Adding Judges](../guides/adding-judges.md) - Create custom judges
