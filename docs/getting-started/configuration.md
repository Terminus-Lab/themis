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
EVAL_AGENT_API_PORT=18082           # API server port (default: 18082)

CONVERSATION_HOLISTIC_WEIGHT=0.5    # α: weight for holistic score (0.0–1.0)
                                    # final_score = α × holistic_score + (1-α) × turn_avg

VERDICT_PASS_THRESHOLD=0.8          # final_score > 0.8 → "pass"
VERDICT_REVIEW_THRESHOLD=0.5        # final_score > 0.5 → "review", else "fail"
```

### Database

```env
IN_MEMORY_DB=true                   # SQLite in-memory (default)
THEMIS_DB_URL=postgresql://...      # Required if IN_MEMORY_DB=false
```

### Streaming (optional)

```env
CONVERSATION_STREAMING_ENABLED=false
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_CONVERSATION_STREAM_KEY=eval-conversations
REDIS_CONVERSATION_GROUP=eval-conv-group
REDIS_CONSUMER_NAME=consumer-1
```

### Batch CLI

```env
THEMIS_BATCH_WORKERS=5              # Concurrent workers (default: 5)
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
      scope: "turn"           # "turn" = per-turn judge, "conversation" = holistic judge
      weight: 0.35
      model:                  # optional — overrides default_model
        modelFamily: "openai_platform"
        modelID: gpt-4o-mini
      prompt: |
        Evaluate if the answer addresses the user's query.

        User Query: {{.Query}}
        Answer: {{.Answer}}

        Score 0.0–1.0. Respond in JSON:
        {"score": <float>, "reason": "<string>"}

    - name: conversation-flow
      enabled: true
      scope: "conversation"   # holistic judge — sees all turns
      weight: 1.0
      prompt: |
        Evaluate the overall flow of this conversation.
        ...
```

### Judge Scopes

| Scope | Description | Phase |
|-------|-------------|-------|
| `turn` | Evaluated per turn (relevance, coherence, completeness) | Phase A |
| `conversation` | Evaluated once for the full conversation (conversation-flow) | Phase B |

### Prompt Variables

| Variable | Available In | Description |
|----------|-------------|-------------|
| `{{.Query}}` | turn judges | User's question for this turn |
| `{{.Answer}}` | turn judges | Agent's response for this turn |
| `{{.Context}}` | turn judges | Retrieved context (optional, for RAG) |
| `{{.Turns}}` | conversation judges | All turns in the conversation |
| `{{.ConversationID}}` | conversation judges | Conversation identifier |

### Default Judges

Four judges are configured out of the box:

| Judge | Scope | Weight | Purpose |
|-------|-------|--------|---------|
| `relevance` | turn | 0.35 | Is the answer relevant to the query? |
| `coherence` | turn | 0.30 | Is the answer coherent and well-formed? |
| `completeness` | turn | 0.35 | Does the answer fully address the query? |
| `conversation-flow` | conversation | 1.0 | Does the conversation flow naturally? |

Turn judge weights are normalized automatically within their scope.

---

## Configuration Scenarios

### Focus on Turn Quality

```yaml
evaluators:
  - name: relevance
    enabled: true
    scope: "turn"
    weight: 0.50
  - name: completeness
    enabled: true
    scope: "turn"
    weight: 0.50
  - name: coherence
    enabled: false   # skip coherence
  - name: conversation-flow
    enabled: true
    scope: "conversation"
    weight: 1.0
```

```env
CONVERSATION_HOLISTIC_WEIGHT=0.3   # Weight turns more heavily
```

### Focus on Conversation Quality

```env
CONVERSATION_HOLISTIC_WEIGHT=0.7   # Weight holistic score more heavily
```

### Strict Quality Control

```env
VERDICT_PASS_THRESHOLD=0.9
VERDICT_REVIEW_THRESHOLD=0.7
```

---

## Best Practices

**Tune the holistic weight** based on what matters more:
- High `CONVERSATION_HOLISTIC_WEIGHT` (0.7+): Overall conversation quality matters most
- Low `CONVERSATION_HOLISTIC_WEIGHT` (0.3-): Per-turn accuracy matters most
- Default (0.5): Balanced evaluation

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
