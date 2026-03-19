# Configuration

Two parts: environment variables (`.env`) and judge definitions (`configs/judges.yaml`).

---

## Environment Variables

### LLM Providers

Configure at least one:

```env
# OpenAI Platform (simplest)
OPEN_AI_KEY=sk-proj-...

# AWS Bedrock (for Claude)
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your_key
AWS_SECRET_ACCESS_KEY=your_secret

# Azure OpenAI
OPEN_AI_KEY=your_azure_key
AZURE_OPENAI_ENDPOINT=https://...openai.azure.com/openai/deployments/...
```

All three can be configured simultaneously — judges reference providers by `modelFamily`.

### Pipeline

```env
EVAL_AGENT_API_PORT=18082                   # API server port (default: 18082)

ENABLE_PRECHECK=true                        # Stage 1 prechecks (default: true)
EARLY_EXIT_THRESHOLD=0.2                    # Exit early if precheck avg < this

PRECHECK_WEIGHT=0.3                         # Stage 1 contribution to final score
LLM_JUDGE_WEIGHT=0.7                        # Stage 2 contribution to final score

VERDICT_PASS_THRESHOLD=0.8                  # confidence > 0.8 → "pass"
VERDICT_REVIEW_THRESHOLD=0.5               # confidence > 0.5 → "review", else "fail"

JUDGE_AGGREGATION_METHOD=weighted_average   # weighted_average | harmonic_mean | median | weighted_product
```

### Database

```env
IN_MEMORY_DB=true                           # SQLite in-memory (default)
THEMIS_DB_URL=postgresql://...              # Required if IN_MEMORY_DB=false
```

### Streaming (optional)

```env
EVENTS_STREAMING_ENABLED=false
REDIS_ADDR=localhost:6379
REDIS_STREAM_KEY=eval-events
REDIS_CONSUMER_GROUP=eval-group
REDIS_CONSUMER_NAME=consumer-1
```

### Batch CLI

```env
THEMIS_BATCH_WORKERS=5                      # Concurrent workers (default: 5)
```

---

## judges.yaml

### Structure

```yaml
judges:
  default_model:
    modelFamily: "openai_platform"  # anthropic | openai | openai_platform
    modelID: gpt-4o-mini
    max_tokens: 200
    temperature: 0.0
    retry: true

  evaluators:
    - name: relevance
      enabled: true
      weight: 0.25
      requires_context: false
      requires_expected_output: false
      model:                            # optional — overrides default_model
        modelFamily: "anthropic"
        modelID: us.anthropic.claude-3-5-sonnet-20241022-v2:0
      prompt: |
        Evaluate if the answer addresses the user's query.

        User Query: {{.UserQuery}}
        Answer: {{.Answer}}
        {{if .Context}}Context: {{.Context}}{{end}}

        Score 0.0–1.0. Respond in JSON:
        {"score": <float>, "reason": "<string>"}
```

### Skip Logic

Judges with `requires_context: true` or `requires_expected_output: true` are skipped automatically if those fields are absent from the request. Existing requests without optional fields continue to work unchanged.

### Prompt Variables

| Variable | Description |
|----------|-------------|
| `{{.UserQuery}}` | User's question |
| `{{.Answer}}` | Agent's response |
| `{{.Context}}` | Retrieved context (RAG) |
| `{{.ExpectedOutput}}` | Ground truth (correctness judge) |

Weights are normalized automatically — they don't need to sum to 1.0.

---

## Configuration Scenarios

### Cost Optimization

```env
ENABLE_PRECHECK=true
EARLY_EXIT_THRESHOLD=0.3
JUDGE_AGGREGATION_METHOD=median
```
```yaml
default_model:
  modelFamily: "openai_platform"
  modelID: gpt-4o-mini
evaluators:
  - name: relevance
    enabled: true
  - name: faithfulness
    enabled: false   # disable expensive judges
  - name: coherence
    enabled: true
  - name: completeness
    enabled: false
  - name: instruction
    enabled: false
```

### Strict Quality Control

```env
ENABLE_PRECHECK=false
JUDGE_AGGREGATION_METHOD=harmonic_mean   # penalizes any low score
VERDICT_PASS_THRESHOLD=0.9
```

One low judge score significantly lowers the final confidence.

### Ground Truth Validation

```env
ENABLE_PRECHECK=false
```
```yaml
evaluators:
  - name: correctness
    enabled: true
    weight: 0.40
    requires_expected_output: true
  - name: relevance
    enabled: true
    weight: 0.30
  - name: coherence
    enabled: true
    weight: 0.30
```

---

## Best Practices

**Validate before deploying judge changes:**
```bash
./bin/themis-cli validate-events -i annotated_sample.jsonl -c 0.3
# Deploy only if Kendall's τ ≥ 0.3
```

**All 4 aggregation methods are computed on every request** and returned in `metrics`:
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
Compare them without re-running evaluations to choose the right method.

**Tune verdict thresholds** based on observed distribution:
- Too many `pass` → increase `VERDICT_PASS_THRESHOLD`
- Too many `fail` → decrease thresholds
- Too many `review` → narrow the review band

---

## Config File Search Order

`judges.yaml` is loaded from the first match:
1. `$JUDGES_CONFIG_PATH` env var
2. `./judges.yaml`
3. `./config/judges.yaml`
4. `./configs/judges.yaml`
