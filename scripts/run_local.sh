#!/bin/bash
# Local integration test: start server and run client
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Word of Wisdom - Local Integration Test ===${NC}"

# Configuration
SERVER_ADDR="${WOW_ADDR:-:8080}"
SERVER_BITS="${WOW_BITS:-20}"
SERVER_EXPIRES="${WOW_EXPIRES:-60}"
CLIENT_ADDR="${SERVER_ADDR#:}"  # Remove leading : if present
if [ "$CLIENT_ADDR" = "$SERVER_ADDR" ]; then
    CLIENT_ADDR="127.0.0.1${SERVER_ADDR}"
fi

# Build binaries
echo -e "${YELLOW}Building binaries...${NC}"
go build -o bin/server ./cmd/server
go build -o bin/client ./cmd/client

# Start server in background
echo -e "${YELLOW}Starting server (addr=${SERVER_ADDR}, bits=${SERVER_BITS}, expires=${SERVER_EXPIRES})...${NC}"
WOW_ADDR="$SERVER_ADDR" WOW_BITS="$SERVER_BITS" WOW_EXPIRES="$SERVER_EXPIRES" ./bin/server > server.log 2>&1 &
SERVER_PID=$!

# Wait for server to start
echo -e "${YELLOW}Waiting for server to start...${NC}"
sleep 2

# Check if server is running
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo -e "${RED}Server failed to start. Check server.log${NC}"
    cat server.log
    exit 1
fi

echo -e "${GREEN}Server started (PID: $SERVER_PID)${NC}"

# Run client
echo -e "${YELLOW}Running client...${NC}"
export WOW_ADDR="$CLIENT_ADDR"
if ./bin/client; then
    echo -e "${GREEN}✓ Client succeeded${NC}"
    EXIT_CODE=0
else
    echo -e "${RED}✗ Client failed${NC}"
    EXIT_CODE=1
fi

# Cleanup: stop server
echo -e "${YELLOW}Stopping server...${NC}"
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

echo -e "${GREEN}=== Test completed ===${NC}"
echo "Server log:"
cat server.log

exit $EXIT_CODE

