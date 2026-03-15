# Ollama Integration - Specification

**Status:** Proposed
**Date:** 2026-03-15

---

## 1. Purpose

### Problem

Currently, Themis requires paid LLM API access (AWS Bedrock, Azure OpenAI, OpenAI Platform) for evaluation. This creates barriers for:
- **Cost-conscious users**: Evaluating 1000s of responses gets expensive
- **Privacy-sensitive applications**: Data must stay on-premises
- **Offline environments**: No internet access for external APIs
- **Experimentation**: Iterating on judge prompts incurs costs

### Solution

Add **Ollama** as a local LLM provider supporting open models:
- Llama 3.1 (8B, 70B, 405B) - Meta's latest
- Llama 3.2 (1B, 3B) - Lightweight edge models
- Mistral (7B, 8x7B) - Strong reasoning
- Qwen2.5 (7B, 14B, 72B) - Multilingual support
- Phi-3 (3.8B) - Microsoft's efficient model

**Key benefits:**
- Zero API costs (run locally or self-hosted)
- Full data privacy (never leaves your infrastructure)
- Works offline (no internet required)
- Fast iteration (no rate limits)

---

## 2. Architecture

### Ollama Overview

**What is Ollama?**
Local LLM runtime with OpenAI-compatible API:
- Runs models via CPU/GPU on Mac, Linux, Windows
- Provides HTTP REST API at `http://localhost:11434`
- Simple installation: `curl -fsSL https://ollama.com/install.sh | sh`
- Model management: `ollama pull llama3.1`, `ollama list`

### Integration Pattern

Themis already supports multi-provider LLM via registry pattern. Ollama follows same approach:

```
internal/llm/
├── anthropic/        # AWS Bedrock Claude
├── openai/           # Azure OpenAI GPT
├── openaiplatform/   # OpenAI Platform API
└── ollama/           # NEW: Local Ollama models
    ├── client.go     # Implements LLMClient interface
    └── config.go     # Ollama-specific config
```

**Interface compatibility:**
Ollama implements existing `LLMClient` interface from `internal/llm/client.go`:
```go
type LLMClient interface {
    InvokeModel(ctx context.Context, prompt string, config ModelConfig) (string, error)
}
```

---

## 3. Implementation

### Step 1: Create Ollama Client Package

**File:** `internal/llm/ollama/client.go`

**Key features:**
- HTTP client to Ollama API endpoint (`http://localhost:11434/api/generate`)
- JSON request/response parsing
- Streaming response handling (Ollama returns NDJSON)
- Error handling for model not found, context length exceeded

**Example request:**
```json
{
  "model": "llama3.1:8b",
  "prompt": "Judge this response...",
  "stream": false,
  "options": {
    "temperature": 0.0,
    "num_predict": 1024
  }
}
```

**Configuration:**
```go
type OllamaConfig struct {
    BaseURL string // Default: http://localhost:11434
    Timeout time.Duration
}
```

### Step 2: Add LLMFamily Constant

**File:** `internal/llm/llm_client_factory.go`

Add new family:
```go
const (
    LLMFamilyAnthropic      = "anthropic"
    LLMFamilyOpenAI         = "openai"
    LLMFamilyOpenAIPlatform = "openai_platform"
    LLMFamilyOllama         = "ollama"  // NEW
)
```

### Step 3: Update Client Registry

**File:** `internal/setup/wiring.go`

Add Ollama case in `createLLMClientRegistry()`:
```go
case llm.LLMFamilyOllama:
    ollamaClient, err := ollama.NewClient(ollama.OllamaConfig{
        BaseURL: config.OllamaBaseURL,
        Timeout: 60 * time.Second,
    })
    if err != nil {
        return nil, err
    }
    registry[modelRef] = ollamaClient
```

### Step 4: Environment Configuration

**File:** `.env`

Add optional Ollama config:
```bash
# Ollama configuration (optional - only if using Ollama models)
OLLAMA_BASE_URL=http://localhost:11434  # Default: localhost
# For remote Ollama: http://192.168.1.100:11434
```

### Step 5: Judge Configuration

**File:** `configs/judges.yaml`

Example judge using Ollama:
```yaml
- name: relevance
  enabled: true
  weight: 0.2
  model:
    modelFamily: ollama           # NEW family
    modelID: llama3.1:8b          # Ollama model tag
    max_tokens: 1024
    temperature: 0.0
    retry: false
  prompt: |
    Evaluate response relevance...
```

**Model naming convention:**
Use Ollama model tags: `llama3.1:8b`, `mistral:7b`, `qwen2.5:14b`

---

## 4. Usage Workflow

### Prerequisites

1. **Install Ollama:**
   ```bash
   # Mac/Linux
   curl -fsSL https://ollama.com/install.sh | sh

   # Windows: Download from ollama.com
   ```

2. **Pull model:**
   ```bash
   ollama pull llama3.1:8b
   ollama list  # Verify model downloaded
   ```

3. **Start Ollama server** (usually auto-starts):
   ```bash
   ollama serve  # If not running
   ```

### Configuration

**Option A: All judges use Ollama (zero-cost setup)**
```yaml
# configs/judges.yaml
judges:
  - name: relevance
    model:
      modelFamily: ollama
      modelID: llama3.1:8b

  - name: faithfulness
    model:
      modelFamily: ollama
      modelID: llama3.1:8b

  # ... all 6 judges use Ollama
```

**Option B: Hybrid (local + cloud)**
```yaml
# Mix Ollama (cost-free) with cloud (higher quality)
judges:
  - name: relevance
    model:
      modelFamily: ollama         # Fast, local
      modelID: llama3.1:8b

  - name: correctness
    model:
      modelFamily: anthropic      # Critical, use Claude
      modelID: anthropic.claude-sonnet-4-5
```

### Running Evaluation

```bash
# Same commands, Ollama auto-detected from judges.yaml
go run cmd/api/main.go                    # API server
go run cmd/batch/main.go evaluate ...     # Batch mode
go run cmd/mcp/main.go                    # MCP server
```

**No code changes needed** - existing pipeline auto-routes to Ollama client based on `modelFamily`

---

## 5. Model Recommendations

### Lightweight (Fast, Low Memory)

| Model | Size | Use Case | RAM Required |
|-------|------|----------|--------------|
| `llama3.2:1b` | 1B | Edge devices, quick checks | 2 GB |
| `llama3.2:3b` | 3B | Fast evaluation, high throughput | 4 GB |
| `phi3:3.8b` | 3.8B | Balanced speed/quality | 4 GB |

### Standard (Good Quality)

| Model | Size | Use Case | RAM Required |
|-------|------|----------|--------------|
| `llama3.1:8b` | 8B | **Recommended default** | 8 GB |
| `mistral:7b` | 7B | Strong reasoning | 8 GB |
| `qwen2.5:7b` | 7B | Multilingual support | 8 GB |

### High Quality (Best Results)

| Model | Size | Use Case | RAM Required |
|-------|------|----------|--------------|
| `llama3.1:70b` | 70B | Production-grade quality | 48 GB |
| `qwen2.5:72b` | 72B | Complex evaluations | 48 GB |
| `llama3.1:405b` | 405B | Research, highest quality | 256 GB |

**Recommendation for most users:** `llama3.1:8b` (good balance of speed/quality/memory)

---

## 6. Performance Considerations

### Latency Comparison

| Provider | Typical Latency | Notes |
|----------|----------------|-------|
| Ollama (CPU) | 2-5 sec/judge | MacBook Pro M2, llama3.1:8b |
| Ollama (GPU) | 0.5-1 sec/judge | NVIDIA RTX 4090, llama3.1:8b |
| AWS Bedrock | 1-2 sec/judge | Network latency included |
| OpenAI Platform | 0.5-1.5 sec/judge | Network latency included |

**Note:** Ollama is competitive with cloud APIs on GPU hardware.

### Throughput

**Parallel execution:**
Themis runs 6 judges concurrently. With Ollama on 8-core CPU:
- 6 concurrent requests = ~3-4 sec/evaluation (limited by CPU cores)
- With GPU: ~1 sec/evaluation (much faster)

**Scaling:**
For high throughput, deploy remote Ollama instance:
```bash
# On GPU server
OLLAMA_HOST=0.0.0.0:11434 ollama serve

# In .env
OLLAMA_BASE_URL=http://gpu-server:11434
```

---

## 7. Testing Strategy

### Unit Tests

**File:** `internal/llm/ollama/client_test.go`

Test cases:
1. Successful model invocation
2. Model not found error
3. Connection refused (Ollama not running)
4. Context length exceeded
5. Timeout handling
6. JSON parsing errors

### Integration Tests

1. **Local Ollama test:**
   ```bash
   # Prerequisite: ollama pull llama3.2:1b
   go test ./internal/llm/ollama/... -tags=integration
   ```

2. **Full pipeline test:**
   - Configure one judge with Ollama
   - Run evaluation via API/CLI/MCP
   - Verify results match expected format

### Validation

Compare Ollama vs cloud judges on same dataset:
```bash
# Run with Ollama
JUDGE_CONFIG=judges_ollama.yaml go run cmd/batch/main.go evaluate ...

# Run with Claude
JUDGE_CONFIG=judges_claude.yaml go run cmd/batch/main.go evaluate ...

# Compare Kendall's τ correlation between two judge sets
```

**Expected:** τ > 0.7 indicates Ollama judges perform similarly to cloud

---

## 8. Documentation Updates

### Files to Update

1. **README.md** - Add Ollama to supported providers
2. **CLAUDE.md** - Update LLM provider list
3. **docs/getting-started/installation.md** - Add Ollama setup steps
4. **docs/getting-started/configuration.md** - Document Ollama env vars
5. **configs/judges.yaml** - Add example Ollama judge

### New Documentation

Create `docs/providers/ollama.md`:
- Installation instructions
- Model selection guide
- Performance tuning tips
- Troubleshooting (model not found, OOM errors)

---

## 9. Migration Path

### For Existing Users

**Zero-downtime migration:**
1. Install Ollama, pull model
2. Update ONE judge in `judges.yaml` to use Ollama
3. Test with sample evaluation
4. Gradually migrate remaining judges
5. Remove cloud credentials from `.env` when fully migrated

**Cost savings example:**
- 10,000 evaluations/month × 6 judges = 60,000 LLM calls
- AWS Bedrock Claude Haiku: $0.00025/1K tokens × ~500 tokens = $7.50
- Ollama: $0 (free after hardware)
- **Monthly savings:** $7.50+ (scales with volume)

---

## 10. Success Criteria

### Technical Requirements

- [ ] Ollama client implements `LLMClient` interface
- [ ] Supports all Ollama model tags (llama3.1, mistral, qwen, etc.)
- [ ] Error handling for model not found, timeout, OOM
- [ ] Unit tests with 90%+ coverage
- [ ] Integration test with local Ollama instance

### User Experience

- [ ] Works with existing pipeline (no code changes)
- [ ] Clear error messages when Ollama not running
- [ ] Documentation covers installation → first evaluation in <10 minutes
- [ ] Performance comparable to cloud APIs (with GPU)

### Validation

- [ ] Ollama judges achieve τ ≥ 0.5 correlation with human annotations
- [ ] Results consistent with cloud provider judges (τ ≥ 0.7 correlation)

---

## 11. Future Enhancements

### Phase 2 Features

1. **Model auto-download:** If model not found, auto `ollama pull`
2. **Multi-instance load balancing:** Round-robin across multiple Ollama servers
3. **GPU optimization:** Detect GPU availability, suggest best model size
4. **Prompt caching:** Cache system prompts for faster inference
5. **Quantization support:** Use 4-bit/8-bit quantized models (faster, less memory)

### Advanced Use Cases

1. **Fine-tuned models:** Support custom Ollama models trained on evaluation data
2. **Embedding models:** Use Ollama for semantic similarity checks (nomic-embed-text)
3. **Vision models:** Future support for LLaVA (multimodal evaluation)

---

## 12. Open Questions

1. **Default model:** Should we ship with recommended Ollama model in judges.yaml?
2. **Fallback behavior:** If Ollama fails, fallback to cloud provider?
3. **Model validation:** Verify model exists before starting evaluation?
4. **Streaming:** Should we support streaming responses for faster TTFT?

---

## Appendix: Example Implementation

### Minimal Ollama Client (Proof of Concept)

```go
package ollama

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type Client struct {
    baseURL    string
    httpClient *http.Client
}

type generateRequest struct {
    Model   string                 `json:"model"`
    Prompt  string                 `json:"prompt"`
    Stream  bool                   `json:"stream"`
    Options map[string]interface{} `json:"options,omitempty"`
}

type generateResponse struct {
    Response string `json:"response"`
    Done     bool   `json:"done"`
}

func (c *Client) InvokeModel(ctx context.Context, prompt string, config ModelConfig) (string, error) {
    req := generateRequest{
        Model:  config.ModelID,
        Prompt: prompt,
        Stream: false,
        Options: map[string]interface{}{
            "temperature":  config.Temperature,
            "num_predict":  config.MaxTokens,
        },
    }

    body, _ := json.Marshal(req)
    httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewReader(body))
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return "", fmt.Errorf("ollama request failed: %w", err)
    }
    defer resp.Body.Close()

    var result generateResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", fmt.Errorf("failed to parse response: %w", err)
    }

    return result.Response, nil
}
```

**~50 lines of code** for full Ollama integration!

---

## Timeline Estimate

- **Phase 1 (Core):** 1-2 days
  - Ollama client implementation
  - Registry integration
  - Basic testing

- **Phase 2 (Polish):** 1 day
  - Documentation
  - Integration tests
  - Example configurations

**Total:** 2-3 days for production-ready Ollama support
