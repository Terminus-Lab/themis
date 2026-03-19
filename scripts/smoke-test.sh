#!/bin/bash
set -e

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

echo "=== 4. API health check ==="
curl -sf http://localhost:18082/api/v1/health | jq -e '.status == "ok"' > /dev/null
echo "✓ Health OK"

echo "=== 5. Evaluate (pass case) ==="
RESULT=$(curl -sf -X POST http://localhost:18082/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{"event_id":"smoke-1","conversation_id":"conv-smoke-1","event_type":"agent_response","agent":{"name":"smoke","type":"rag","version":"1"},"interaction":{"user_query":"What is 2+2?","context":"Basic math.","answer":"2 plus 2 equals 4."}}')
echo "$RESULT" | jq -e '.verdict' > /dev/null
echo "✓ Evaluate OK: $(echo $RESULT | jq -r '{verdict,confidence}')"

echo "=== 6. Query results ==="
curl -sf "http://localhost:18082/api/v1/results?limit=5" | jq -e '.total >= 1' > /dev/null
echo "✓ Query results OK"

echo "=== 7. Multi-turn conversation ==="
for turn in 2 3; do
  curl -sf -X POST http://localhost:18082/api/v1/evaluate \
    -H "Content-Type: application/json" \
    -d "{\"event_id\":\"smoke-$turn\",\"conversation_id\":\"conv-smoke-1\",\"event_type\":\"agent_response\",\"agent\":{\"name\":\"smoke\",\"version\":\"1\"},\"interaction\":{\"user_query\":\"Question $turn\",\"answer\":\"Answer $turn with sufficient detail.\"}}" > /dev/null
done
CONV=$(curl -sf "http://localhost:18082/api/v1/conversations/conv-smoke-1")
echo "$CONV" | jq -e '.turn_count == 3' > /dev/null
echo "✓ Conversation turns OK: $(echo $CONV | jq '{conversation_id, turn_count}')"

echo "=== 8. List conversations ==="
curl -sf "http://localhost:18082/api/v1/conversations" | jq -e '.total >= 1' > /dev/null
echo "✓ List conversations OK"

echo "=== 9. Sample download ==="
curl -sf -X POST http://localhost:18082/api/v1/validation/sample/events/download \
  -H "Content-Type: application/json" \
  -d '{"start_date":"2020-01-01T00:00:00Z","end_date":"2099-01-01T00:00:00Z","percentage":100}' \
  -o /tmp/smoke-sample.jsonl
SAMPLE_LINES=$(wc -l < /tmp/smoke-sample.jsonl)
[ "$SAMPLE_LINES" -ge 1 ] || { echo "✗ Sample download returned 0 lines"; exit 1; }
head -1 /tmp/smoke-sample.jsonl | jq -e '.event_id' > /dev/null
echo "✓ Sample download OK: $SAMPLE_LINES records"

echo "=== 10. Stop API server ==="
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null || true
echo "✓ Graceful shutdown OK"

echo "=== 11. CLI evaluate ==="
./bin/themis-cli evaluate -i resources/dataset.jsonl -o /tmp/smoke-results.jsonl
echo "✓ CLI evaluate OK: $(wc -l < /tmp/smoke-results.jsonl) records"

echo "=== 12. CLI summary ==="
./bin/themis-cli evaluate -i resources/dataset.jsonl -f summary
echo "✓ CLI summary OK"

echo "=== 13. CLI evaluate-conversations (summary) ==="
./bin/themis-cli evaluate-conversations \
  -i resources/conversations.jsonl \
  -f summary \
  -o /tmp/smoke-conv-summary.json
cat /tmp/smoke-conv-summary.json | jq -e '.total >= 1' > /dev/null
cat /tmp/smoke-conv-summary.json | jq -e 'has("pass_count") and has("fail_count") and has("review_count") and has("avg_confidence") and has("avg_turn_count")' > /dev/null
echo "✓ CLI evaluate-conversations summary OK: $(cat /tmp/smoke-conv-summary.json | jq '{total, pass_count, fail_count, review_count, avg_confidence}')"

echo "=== 14. CLI validate-events ==="
./bin/themis-cli validate-events -i resources/annotated_sample.jsonl -c 0.3
echo "✓ CLI validate-events OK"

echo ""
echo "All smoke tests passed ✓"