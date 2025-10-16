#!/bin/bash
# Test rate limiting and adaptive difficulty

set -e

echo "Building binaries..."
go build -o bin/server_test ./cmd/server
go build -o bin/client_test ./cmd/client

echo "Starting server with rate limit (5 req/sec) and adaptive bits..."
WOW_ADDR=:8081 WOW_BITS=18 WOW_RATE_LIMIT=5 WOW_ADAPTIVE_BITS=true WOW_EXPIRES=300 \
  ./bin/server_test &
SERVER_PID=$!

# Wait for server to start
sleep 2

echo "Testing rapid requests (should trigger rate limit and adaptive difficulty)..."
for i in {1..10}; do
  echo "Request #$i:"
  WOW_ADDR=127.0.0.1:8081 ./bin/client_test || true
  sleep 0.1
done

echo "Killing server..."
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null || true

echo "Done! Check logs above for:"
echo "  - 'rate limited' messages after 5-10 requests"
echo "  - 'adaptive difficulty increased' with rising bits"

