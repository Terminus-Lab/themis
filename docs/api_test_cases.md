# API Test Cases

Comprehensive test scenarios for the eval-agent HTTP API.

**Important:** Expected responses show **representative results**. Actual LLM judge scores may vary slightly (±0.1) due to model variability, but the overall patterns (high/medium/low scores, verdicts, stage counts) should match. Focus on validating behavior patterns rather than exact numeric matches.

## Setup

### Prerequisites

Ensure your `.env` file is configured:

```env
# AWS Bedrock credentials (if using anthropic models)
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your_key
AWS_SECRET_ACCESS_KEY=your_secret

# OpenAI credentials (if using openai models)
OPEN_AI_KEY=sk-...

# API configuration
EVAL_AGENT_API_PORT=18082
EARLY_EXIT_THRESHOLD=0.2
```

Judges are configured in `configs/judges.yaml`. The system dynamically creates LLM clients based on models referenced in the configuration.

### Start the Server

```bash
cd eval-agent
go run cmd/api/main.go
```

Server runs on `http://localhost:18082`

**Expected startup logs:**
```
INFO judge created successfully judge=relevance model_family=anthropic model_id=us.anthropic.claude-3-5-haiku-20241022-v1:0
INFO judge created successfully judge=faithfulness model_family=anthropic model_id=us.anthropic.claude-3-5-haiku-20241022-v1:0
INFO judge created successfully judge=coherence model_family=anthropic model_id=us.anthropic.claude-3-5-haiku-20241022-v1:0
INFO judge created successfully judge=completeness model_family=anthropic model_id=us.anthropic.claude-3-5-haiku-20241022-v1:0
INFO judge created successfully judge=instruction model_family=anthropic model_id=us.anthropic.claude-3-5-haiku-20241022-v1:0
INFO judge pool built successfully total_judges=5
```

## Full Pipeline Evaluation Tests

### Test Case 1: Happy Path - High Quality Answer

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-001",
    "event_type": "agent_response",
    "agent": {"name": "test-agent", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "What is the capital of France?",
      "context": "France is a country in Western Europe. Paris is its capital city and largest metropolis.",
      "answer": "The capital of France is Paris."
    }
  }'
```

**Expected Response (truncated, full response has 8 stages):**
**Expected:**
- Status Code: 200
- `confidence` > 0.8
- `verdict` = "pass"
- **8 stages** (3 prechecks + 5 judges)
- All scores high (answer is correct, relevant, and grounded)
- Response time: ~3-4 seconds (judges run in parallel)

### Test Case 2: Early Exit - Very Short Answer

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-002",
    "event_type": "agent_response",
    "agent": {"name": "test-agent", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "Explain quantum computing in detail",
      "context": "Quantum computing uses quantum mechanics principles...",
      "answer": "Yes."
    }
  }'
```

**Expected:**
- Status Code: 200
- `confidence` < 0.2
- `verdict` = "fail"
- Only 3 stages (prechecks only, early exit triggered)
- No LLM judges called (cost savings)

### Test Case 3: Fail Verdict - Irrelevant Answer

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-003",
    "event_type": "agent_response",
    "agent": {"name": "test-agent", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "What are the main causes of climate change?",
      "context": "Climate change is primarily caused by greenhouse gas emissions from human activities.",
      "answer": "There are various factors that contribute to weather patterns."
    }
  }'
```

**Expected:**
- Status Code: 200
- `confidence` ~0.35
- `verdict` = "fail"
- 8 stages (full pipeline - all judges run)
- Low scores across all judges:
  - **relevance: 0.2** - Answer discusses weather patterns, not climate change causes
  - **faithfulness: 0.2** - Doesn't mention greenhouse gases from context
  - **coherence: 0.3** - Vague but not contradictory
  - **completeness: 0.0** - Completely fails to address the query
  - **instruction: 0.4** - No specific instructions violated
- Total duration: ~3 seconds (5 parallel LLM calls)

### Test Case 4: Hallucination Detection

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-004",
    "event_type": "agent_response",
    "agent": {"name": "test-agent", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "What is the population of Tokyo?",
      "context": "Tokyo is the capital of Japan.",
      "answer": "Tokyo has a population of 50 million people and is the largest city in China."
    }
  }'
```

**Expected:**
- Status Code: 200
- `confidence` = 0.393
- `verdict` = "fail"
- 8 stages (full pipeline)
- **Excellent hallucination detection** - All judges identify the errors:
  - **faithfulness: 0.1** - Correctly flags factual errors not in context ("China" not mentioned)
  - **coherence: 0.2** - Identifies logical inconsistency (Tokyo cannot be in both Japan and China)
  - **relevance: 0.2** - Notes answer addresses population but with wrong facts
  - **completeness: 0.0** - Answer is factually wrong
  - **instruction: 0.4** - No specific format instructions, penalizes factual errors
- Total duration: ~2.3 seconds (judges run in parallel)
- System successfully detects:
  - Geographic hallucination (China vs Japan)
  - Population exaggeration (50M vs actual ~14M city, ~37M metro)
  - Contradictory claims

---

## Single Judge Evaluation Tests

### Test Case 5: Relevance Judge

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate/judge/relevance \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-005",
    "event_type": "agent_response",
    "agent": {"name": "test", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "What is machine learning?",
      "context": "ML is a subset of AI.",
      "answer": "Machine learning is a method where computers learn from data without explicit programming."
    }
  }'
```

**Expected:**
- Status Code: 200
- Only 1 stage (relevance-judge)
- Score close to 1.0 for relevant answer

### Test Case 6: Custom Threshold

**Request:**
```bash
curl -X POST "http://localhost:18082/api/v1/evaluate/judge/faithfulness?threshold=0.9" \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-006",
    "event_type": "agent_response",
    "agent": {"name": "test", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "What is the boiling point of water?",
      "context": "Water boils at 100°C at sea level.",
      "answer": "Water boils at 100 degrees Celsius."
    }
  }'
```

**Expected Response:**
- Status Code: 200
- High faithfulness score (grounded in context)
- `verdict` = "pass" (score > 0.9 threshold)

### Test Case 7: Invalid Judge Name

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate/judge/invalid-judge \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-007",
    "event_type": "agent_response",
    "agent": {"name": "test", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "Test",
      "context": "Test",
      "answer": "Test"
    }
  }'
```

**Expected Response:**
- Status Code: 400 or 404
- Error message: "judge not found" or similar

---

## Error Handling Tests

### Test Case 8: Missing Required Fields

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-008",
    "interaction": {
      "user_query": "Test"
    }
  }'
```
---

## Performance Tests

### Test Case 9: Concurrent Requests

**Script:**
```bash
# Send 10 concurrent requests
for i in {1..10}; do
  curl -X POST http://localhost:18082/api/v1/evaluate \
    -H "Content-Type: application/json" \
    -d "{\"event_id\":\"perf-$i\",\"event_type\":\"agent_response\",\"agent\":{\"name\":\"test\",\"type\":\"rag\",\"version\":\"1.0\"},\"interaction\":{\"user_query\":\"Test\",\"context\":\"Test\",\"answer\":\"Test\"}}" &
done
wait
```

**Expected:**
- All 10 requests complete successfully
- Response times < 5 seconds per request
- No race conditions or errors

### Test Case 10: Large Context (10KB)

**Request:**
```bash
# Generate large context
LARGE_CONTEXT=$(python3 -c "print('Context word. ' * 2000)")

curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d "{
    \"event_id\": \"test-010\",
    \"event_type\": \"agent_response\",
    \"agent\": {\"name\": \"test\", \"type\": \"rag\", \"version\": \"1.0\"},
    \"interaction\": {
      \"user_query\": \"Summarize the context\",
      \"context\": \"$LARGE_CONTEXT\",
      \"answer\": \"This is a summary.\"
    }
  }"
```

**Expected:**
- Status Code: 200
- Evaluation completes without timeout
- All judges handle large context

---

## Edge Cases

### Test Case 11: Special Characters in Answer

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-011",
    "event_type": "agent_response",
    "agent": {"name": "test", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "Show me code",
      "context": "Python example",
      "answer": "def hello():\n    print(\"Hello, World!\")\n    return True"
    }
  }'
```

**Expected:**
- Status Code: 200
- Evaluation handles newlines and special characters
- Format checker passes

### Test Case 12: Non-English Text

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-012",
    "event_type": "agent_response",
    "agent": {"name": "test", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "Quelle est la capitale de la France?",
      "context": "La France est un pays en Europe.",
      "answer": "La capitale de la France est Paris."
    }
  }'
```

**Expected:**
- Status Code: 200
- Evaluation works with non-English text
- Judges provide appropriate scores

---

## Multi-Provider Testing

### Test Case 13: Mixed Provider Evaluation

**Setup:**
Update `configs/judges.yaml` to use different providers:

```yaml
judges:
  default_model:
    modelFamily: "anthropic"
    modelID: us.anthropic.claude-3-5-haiku-20241022-v1:0

  evaluators:
    - name: relevance
      model:
        modelFamily: "anthropic"
        modelID: us.anthropic.claude-3-5-haiku-20241022-v1:0
    - name: faithfulness
      model:
        modelFamily: "openai"
        modelID: gpt-4o-mini
```

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-multi-provider",
    "event_type": "agent_response",
    "agent": {"name": "test-agent", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "What is machine learning?",
      "context": "ML is a subset of AI that enables systems to learn from data.",
      "answer": "Machine learning allows computers to learn patterns from data without explicit programming."
    }
  }'
```

**Expected:**
- Status Code: 200
- Startup logs show both Anthropic and OpenAI clients initialized
- Each judge uses its configured provider
- All judges complete successfully
- Mixed provider evaluation works seamlessly

---

## Correctness Judge Tests (Ground Truth Comparison)

The correctness judge evaluates semantic similarity between an answer and expected output (ground truth). It's disabled by default and requires `expected_output` field in requests.

**Quick Start:** For automated validation of your correctness judge setup, see [VALIDATION_CORRECTNESS.md](VALIDATION_CORRECTNESS.md) - includes a script to test all scenarios automatically.

**Important Notes:**
- These test cases show **expected patterns** - actual scores may vary slightly depending on the LLM's response
- **Enable the correctness judge first** (see Test Case 17) before running tests 18-24
- Scores should be in the expected range (±0.1) even if not exact matches
- The key behaviors to validate:
  - Exact matches score ~1.0
  - Semantic matches (e.g., "4" vs "four") score ~0.8-0.9
  - Wrong answers score ~0.0-0.2
  - Auto-skip works when `expected_output` is missing

### Test Case 14: Enable Correctness Judge

**Setup:**
Enable the correctness judge in `configs/judges.yaml`:

```yaml
judges:
  evaluators:
    # ... other judges ...

    - name: correctness
      enabled: true  # Change from false to true
      description: "Evaluates semantic similarity between answer and expected output"
      requires_context: false
      requires_expected_output: true
      prompt: |
        You are a correctness evaluation judge.
        Compare the provided answer with the expected output (ground truth).
        Score based on semantic equivalence, not exact string match.

        Answer: {{.Answer}}
        Expected Output: {{.ExpectedOutput}}

        Scoring guidelines:
        - 1.0: Semantically identical
        - 0.8-0.9: Mostly correct, minor differences
        - 0.5-0.7: Partially correct
        - 0.2-0.4: Somewhat related but different
        - 0.0-0.1: Completely different

        Respond ONLY in raw JSON with no markdown, no code blocks, no explanation:
        {"score": <float>, "reason": "<string>"}
      model:
        max_tokens: 200
        temperature: 0.0
        retry: true
```

**Restart the API server** after enabling the judge.

### Test Case 15: Single Correctness Judge - Exact Match

**Request:**
```bash
curl -X POST "http://localhost:18082/api/v1/evaluate/judge/correctness?threshold=0.8" \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-correctness-001",
    "event_type": "agent_response",
    "agent": {"name": "test-agent", "type": "qa", "version": "1.0"},
    "interaction": {
      "user_query": "What is the capital of France?",
      "answer": "Paris",
      "expected_output": "Paris"
    }
  }'
```

**Expected:**
- Status Code: 200
- `confidence` = 1.0 (exact match)
- `verdict` = "pass"
- Only 1 stage (correctness judge)
- Response time: ~1-2 seconds

### Test Case 16: Single Correctness Judge - Semantic Match

**Request:**
```bash
curl -X POST "http://localhost:18082/api/v1/evaluate/judge/correctness?threshold=0.8" \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-correctness-002",
    "event_type": "agent_response",
    "agent": {"name": "test-agent", "type": "qa", "version": "1.0"},
    "interaction": {
      "user_query": "What is 2+2?",
      "answer": "The answer is four",
      "expected_output": "4"
    }
  }'
```

**Expected:**
- Status Code: 200
- `confidence` ~0.9 (semantic match despite different format)
- `verdict` = "pass"
- Demonstrates that judge understands "four" = "4"

### Test Case 17 Single Correctness Judge - Wrong Answer

**Request:**
```bash
curl -X POST "http://localhost:18082/api/v1/evaluate/judge/correctness?threshold=0.7" \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-correctness-003",
    "event_type": "agent_response",
    "agent": {"name": "test-agent", "type": "qa", "version": "1.0"},
    "interaction": {
      "user_query": "What is the capital of Italy?",
      "answer": "Milan",
      "expected_output": "Rome"
    }
  }'
```

**Expected:**
- Status Code: 200
- `confidence` ~0.1 (wrong answer)
- `verdict` = "fail"
- Correctly identifies that Milan ≠ Rome

### Test Case 18: Full Pipeline with Correctness - All Judges Pass

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-correctness-pipeline-001",
    "event_type": "agent_response",
    "agent": {"name": "test-agent", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "What is the capital of France?",
      "context": "France is a country in Western Europe. Paris is its capital city.",
      "answer": "The capital of France is Paris.",
      "expected_output": "Paris"
    }
  }'
```

**Expected Response:**
```json
{
  "id": "test-correctness-pipeline-001",
  "stages": [
    {"name": "length-checker", "score": 1.0, "duration_ns": 15000},
    {"name": "overlap-checker", "score": 0.85, "duration_ns": 12000},
    {"name": "format-checker", "score": 1.0, "duration_ns": 10000},
    {"name": "relevance-judge", "score": 0.95, "duration_ns": 1850000000},
    {"name": "faithfulness-judge", "score": 1.0, "duration_ns": 1820000000},
    {"name": "coherence-judge", "score": 1.0, "duration_ns": 1780000000},
    {"name": "completeness-judge", "score": 1.0, "duration_ns": 1750000000},
    {"name": "instruction-judge", "score": 1.0, "duration_ns": 1690000000},
    {"name": "correctness-judge", "score": 1.0, "reason": "Semantically identical to expected output", "duration_ns": 1650000000}
  ],
  "confidence": 0.96,
  "verdict": "pass"
}
```

**Expected:**
- Status Code: 200
- `confidence` > 0.9
- `verdict` = "pass"
- **9 stages** (3 prechecks + 6 judges including correctness)
- All judges score high
- Response time: ~3-4 seconds (judges run in parallel)

### Test Case 18: Full Pipeline with Correctness - Factually Wrong but Well-Formed Answer

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-correctness-pipeline-002",
    "event_type": "agent_response",
    "agent": {"name": "test-agent", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "What is the largest planet in our solar system?",
      "context": "The solar system contains eight planets orbiting the Sun.",
      "answer": "The largest planet in our solar system is Saturn, which is known for its distinctive ring system.",
      "expected_output": "Jupiter"
    }
  }'
```

**Expected Response:**
```json
{
  "id": "test-correctness-pipeline-002",
  "stages": [
    {"name": "length-checker", "score": 1.0, "duration_ns": 14000},
    {"name": "overlap-checker", "score": 0.75, "duration_ns": 11000},
    {"name": "format-checker", "score": 1.0, "duration_ns": 9000},
    {"name": "relevance-judge", "score": 0.9, "duration_ns": 1950000000},
    {"name": "faithfulness-judge", "score": 0.8, "duration_ns": 1880000000},
    {"name": "coherence-judge", "score": 1.0, "duration_ns": 1820000000},
    {"name": "completeness-judge", "score": 1.0, "duration_ns": 1790000000},
    {"name": "instruction-judge", "score": 1.0, "duration_ns": 1750000000},
    {"name": "correctness-judge", "score": 0.1, "reason": "Incorrect - Saturn is not the largest planet. The expected answer is Jupiter.", "duration_ns": 1680000000}
  ],
  "confidence": 0.72,
  "verdict": "review"
}
```

**Expected:**
- Status Code: 200
- `confidence` ~0.72 (only correctness is low, other judges score high)
- `verdict` = "review"
- 9 stages (full pipeline)
- **Key insight**: Other judges score well because the answer is:
  - **Relevant**: Directly addresses "largest planet" question (0.9)
  - **Faithful**: Talks about solar system planets from context (0.8)
  - **Coherent**: Logically structured and well-formed (1.0)
  - **Complete**: Fully answers the question asked (1.0)
  - **Instruction-following**: No format violations (1.0)
- **Correctness judge: 0.1** - Only judge that catches Saturn ≠ Jupiter
- Demonstrates that correctness is orthogonal to quality - you can have a well-formed, coherent answer that's factually wrong

### Test Case 19: Correctness Judge Auto-Skip (No Expected Output)

**Request:**
```bash
curl -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test-correctness-skip",
    "event_type": "agent_response",
    "agent": {"name": "test-agent", "type": "rag", "version": "1.0"},
    "interaction": {
      "user_query": "What is machine learning?",
      "context": "ML is a subset of AI.",
      "answer": "Machine learning is a method where computers learn from data."
    }
  }'
```

**Expected Response:**
```json
{
  "id": "test-correctness-skip",
  "stages": [
    {"name": "length-checker", "score": 1.0, "duration_ns": 15000},
    {"name": "overlap-checker", "score": 0.75, "duration_ns": 12000},
    {"name": "format-checker", "score": 1.0, "duration_ns": 10000},
    {"name": "relevance-judge", "score": 0.95, "duration_ns": 1850000000},
    {"name": "faithfulness-judge", "score": 0.9, "duration_ns": 1820000000},
    {"name": "coherence-judge", "score": 1.0, "duration_ns": 1780000000},
    {"name": "completeness-judge", "score": 0.9, "duration_ns": 1750000000},
    {"name": "instruction-judge", "score": 1.0, "duration_ns": 1690000000}
  ],
  "confidence": 0.89,
  "verdict": "pass"
}
```

**Expected:**
- Status Code: 200
- **8 stages** (3 prechecks + 5 judges - correctness judge auto-skipped)
- No correctness stage in results (because `expected_output` not provided)
- No errors or warnings
- Demonstrates backwards compatibility - requests without `expected_output` work unchanged

**Server logs:**
```
DEBUG skipping judge - expected_output not provided judge=correctness
DEBUG all judges completed judgeCount=5
```

---

## Batch Evaluation with Correctness

### Test Case 20: Batch Correctness Evaluation

**Setup:**
Create a test file `test_correctness_batch.jsonl`:

```jsonl
{"event_id": "batch-001", "event_type": "agent_response", "agent": {"name": "test"}, "interaction": {"user_query": "Capital of France?", "answer": "Paris", "expected_output": "Paris"}}
{"event_id": "batch-002", "event_type": "agent_response", "agent": {"name": "test"}, "interaction": {"user_query": "2+2?", "answer": "4", "expected_output": "Four"}}
{"event_id": "batch-003", "event_type": "agent_response", "agent": {"name": "test"}, "interaction": {"user_query": "Capital of Italy?", "answer": "Milan", "expected_output": "Rome"}}
{"event_id": "batch-004", "event_type": "agent_response", "agent": {"name": "test"}, "interaction": {"user_query": "What is Go?", "answer": "A programming language", "expected_output": "Go is a programming language created by Google"}}
```

**Run batch evaluation:**
```bash
go run cmd/batch/main.go \
  -input test_correctness_batch.jsonl \
  -output correctness_results.jsonl \
  -workers 2
```

**Expected output (showing full pipeline):**

Each record will have 9 stages (3 prechecks + 6 judges). Example for batch-001:
```json
{
  "id": "batch-001",
  "stages": [
    {"name": "length-checker", "score": 1.0},
    {"name": "overlap-checker", "score": 0.9},
    {"name": "format-checker", "score": 1.0},
    {"name": "relevance-judge", "score": 0.95},
    {"name": "faithfulness-judge", "score": 0.9},
    {"name": "coherence-judge", "score": 1.0},
    {"name": "completeness-judge", "score": 1.0},
    {"name": "instruction-judge", "score": 1.0},
    {"name": "correctness-judge", "score": 1.0, "reason": "Exact match"}
  ],
  "confidence": 0.96,
  "verdict": "pass"
}
```

**Note:** Full pipeline runs (all judges + prechecks). For brevity, examples below show only correctness-judge scores.

**Console summary:**
```
INFO Starting batch evaluation input=test_correctness_batch.jsonl output=correctness_results.jsonl workers=2
INFO Worker pool finished
INFO Batch evaluation complete duration=8.234s total_records=4 success=4 errors=0
```