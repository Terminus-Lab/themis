#!/bin/bash
set -e

BASE_URL="http://localhost:18082"

echo "=== 1. Build ==="
go build -o bin/themis-api cmd/api/main.go
go build -o bin/themis-cli cmd/batch/main.go
echo "✓ Build OK"

echo "=== 2. Unit tests ==="
go test ./...
echo "✓ Unit tests OK"

echo "=== 3. Start API server ==="
./bin/themis-api &
SERVER_PID=$!
sleep 3

echo "=== 4. Health check ==="
curl -sf "$BASE_URL/api/v1/health" | jq -e '.status == "ok"' > /dev/null
echo "✓ Health OK"

echo "=== 5. Evaluate conversation ==="
RESULT=$(curl -sf -X POST "$BASE_URL/api/v1/conversations/evaluate" \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "smoke-conv-001",
    "agent": {"name": "smoke-agent", "version": "1.0"},
    "turns": [
      {"turn_index": 0, "user_query": "What is the capital of France?", "answer": "The capital of France is Paris."},
      {"turn_index": 1, "user_query": "What is the population of Paris?", "answer": "Paris has approximately 2.2 million people within city limits."}
    ]
  }')
echo "$RESULT" | jq -e '.verdict' > /dev/null
echo "$RESULT" | jq -e '.final_score' > /dev/null
CONV_ID=$(echo "$RESULT" | jq -r '.conversation_id')
echo "✓ Evaluate OK: $(echo "$RESULT" | jq -r '{verdict, final_score}')"

echo "=== 6. List conversations ==="
LIST=$(curl -sf "$BASE_URL/api/v1/conversations")
echo "$LIST" | jq -e '.total >= 1' > /dev/null
echo "✓ List conversations OK: $(echo "$LIST" | jq '{total}')"

echo "=== 7. Get conversation by ID ==="
DETAIL=$(curl -sf "$BASE_URL/api/v1/conversations/$CONV_ID")
echo "$DETAIL" | jq -e '.conversation_id' > /dev/null
echo "$DETAIL" | jq -e '.turn_results | length >= 1' > /dev/null
echo "✓ Get conversation OK: $(echo "$DETAIL" | jq -r '{conversation_id, verdict, turn_count}')"

echo "=== 8. Health metrics ==="
curl -sf "$BASE_URL/api/v1/metrics/health?window=7d" | jq -e '.total_evaluations >= 1' > /dev/null
echo "✓ Health metrics OK"

echo "=== 9. Stop API server ==="
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null || true
echo "✓ Graceful shutdown OK"

echo "=== 10. CLI evaluate (JSONL output) ==="
./bin/themis-cli evaluate -i resources/conversations.jsonl -o /tmp/smoke-results.jsonl
RESULT_LINES=$(wc -l < /tmp/smoke-results.jsonl | tr -d ' ')
[ "$RESULT_LINES" -ge 1 ] || { echo "✗ CLI evaluate returned 0 lines"; exit 1; }
head -1 /tmp/smoke-results.jsonl | jq -e '.verdict' > /dev/null
echo "✓ CLI evaluate OK: $RESULT_LINES result(s)"

echo "=== 11. CLI evaluate (summary output) ==="
./bin/themis-cli evaluate -i resources/conversations.jsonl -f summary
echo "✓ CLI summary OK"

echo "=== 12. CLI evaluate with human annotations ==="
./bin/themis-cli evaluate -i resources/annotated_sample.jsonl -o /tmp/smoke-annotated.jsonl
LAST_LINE=$(tail -1 /tmp/smoke-annotated.jsonl)
echo "$LAST_LINE" | jq -e '._type == "correlation_report"' > /dev/null
echo "✓ CLI annotation correlation OK: $(echo "$LAST_LINE" | jq -r '{annotated_count, kendall_tau, cohens_kappa}')"

echo ""
echo "All smoke tests passed ✓"
