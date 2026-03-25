# Ollama — Local Model Inference

Ollama lets you run models locally. No API key, no cost, no network latency.
Themis supports it as a first-class provider via the OpenAI-compatible API that Ollama exposes at `localhost:11434/v1`.

---

## 1. Install Ollama

```bash
brew install ollama
```

Or download from [ollama.com](https://ollama.com/download).

---

## 2. Pull a model

```bash
ollama pull qwen2.5:7b
```

Recommended models for evaluation tasks:

| Model | Size | Good for |
|-------|------|----------|
| `qwen2.5:7b` | 4.4 GB | Structured JSON output — most reliable for judges |
| `llama3.1:8b` | 4.7 GB | General purpose |
| `gemma3:12b` | 8.1 GB | Higher quality, needs more RAM |
| `phi3.5:3.8b` | 2.2 GB | Lightweight, fast iteration |

`qwen2.5:7b` is the recommended starting point — it follows JSON formatting instructions reliably, which matters for judge prompts.

---

## 3. Start Ollama

```bash
ollama serve
```

Ollama listens on `http://localhost:11434` by default.

---

## 4. Configure Themis

No environment variable change is needed if Ollama is running locally. The default is already set:

```env
OLLAMA_BASE_URL=http://localhost:11434/v1
```

Point one or more judges at Ollama in `configs/judges.yaml`:

```yaml
judges:
  evaluators:
    - name: relevance
      enabled: true
      scope: turn
      weight: 0.5
      model:
        modelFamily: "ollama"
        modelID: "qwen2.5:7b"
        max_tokens: 200
        temperature: 0.0
        retry: true

    - name: completeness
      enabled: true
      scope: turn
      weight: 0.5
      model:
        modelFamily: "ollama"
        modelID: "qwen2.5:7b"
        max_tokens: 256
        temperature: 0.0
        retry: true

    - name: conversation-flow
      enabled: true
      scope: conversation
      weight: 1.0
      model:
        modelFamily: "ollama"
        modelID: "qwen2.5:7b"
        max_tokens: 300
        temperature: 0.0
        retry: true
```

---

## 5. Run Themis

```bash
go run cmd/api/main.go
```

---

## Troubleshooting

**`connection refused` on startup** — Ollama is not running. Start it with `ollama serve`.

**Judge returns `Error: failed to deserialize LLM response`** — The model didn't return valid JSON. Try `qwen2.5:7b` which handles structured output more reliably, or add explicit JSON instructions to the prompt.

**Slow first response** — The model is loading into memory. Subsequent calls are faster.

**Remote Ollama instance** — Set `OLLAMA_BASE_URL` to point at the remote host:
```env
OLLAMA_BASE_URL=http://192.168.1.100:11434/v1
```
