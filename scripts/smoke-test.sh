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

echo "=== 7. Stop API server ==="
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null || true
echo "✓ Graceful shutdown OK"

echo "=== 8. CLI evaluate ==="
./bin/themis-cli evaluate -i resources/dataset.jsonl -o /tmp/smoke-results.jsonl
echo "✓ CLI evaluate OK: $(wc -l < /tmp/smoke-results.jsonl) records"

echo "=== 9. CLI summary ==="
./bin/themis-cli evaluate -i resources/dataset.jsonl -f summary
echo "✓ CLI summary OK"

echo "=== 10. CLI validate ==="
./bin/themis-cli validate -i resources/annotated_sample.jsonl -c 0.3
echo "✓ CLI validate OK"

echo ""
echo "All smoke tests passed ✓"