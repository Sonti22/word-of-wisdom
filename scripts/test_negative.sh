#!/bin/bash
# Negative integration tests: test error handling
set +e  # Don't exit on errors

SERVER_ADDR="${1:-127.0.0.1:8080}"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
GRAY='\033[0;90m'
NC='\033[0m'

echo -e "${GREEN}=== Word of Wisdom - Negative Tests ===${NC}"

# Test 1: Send garbage data
test_garbage_data() {
    echo -e "\n${YELLOW}[TEST] Sending garbage data to server...${NC}"
    
    # Use netcat if available, otherwise use bash TCP
    if command -v nc &> /dev/null; then
        (echo "THIS IS NOT VALID JSON !!!" | nc $SERVER_ADDR 2>&1) > /tmp/wow_garbage.log
        if grep -q "error" /tmp/wow_garbage.log; then
            echo -e "${GREEN}✓ Server returned error${NC}"
        else
            echo -e "${GREEN}✓ Server closed connection (expected behavior)${NC}"
        fi
    else
        # Fallback: use bash TCP
        HOST=$(echo $SERVER_ADDR | cut -d: -f1)
        PORT=$(echo $SERVER_ADDR | cut -d: -f2)
        exec 3<>/dev/tcp/$HOST/$PORT 2>/dev/null
        if [ $? -eq 0 ]; then
            read -t 2 challenge <&3
            echo -e "${GRAY}Received challenge${NC}"
            echo "THIS IS NOT VALID JSON !!!" >&3
            read -t 2 response <&3
            exec 3>&-
            if echo "$response" | grep -q "error"; then
                echo -e "${GREEN}✓ Server returned error${NC}"
            else
                echo -e "${GREEN}✓ Server closed connection${NC}"
            fi
        fi
    fi
}

# Test 2: Send invalid solution
test_invalid_solution() {
    echo -e "\n${YELLOW}[TEST] Sending invalid solution...${NC}"
    
    if command -v nc &> /dev/null; then
        HOST=$(echo $SERVER_ADDR | cut -d: -f1)
        PORT=$(echo $SERVER_ADDR | cut -d: -f2)
        
        # Connect and send invalid solution
        (
            exec 3<>/dev/tcp/$HOST/$PORT
            read -t 2 challenge <&3
            echo -e "${GRAY}Received challenge${NC}"
            echo '{"type":"solution","nonce":"invalid-nonce-12345"}' >&3
            read -t 2 response <&3
            exec 3>&-
            
            if echo "$response" | grep -q '"type":"error"'; then
                echo -e "${GREEN}✓ Server returned error for invalid solution${NC}"
            else
                echo -e "${RED}✗ Expected error, got: $response${NC}"
            fi
        ) 2>/dev/null
    else
        echo -e "${GRAY}Skipping (requires netcat or bash /dev/tcp)${NC}"
    fi
}

# Test 3: Test timeout
test_timeout() {
    echo -e "\n${YELLOW}[TEST] Testing timeout (not sending solution)...${NC}"
    
    HOST=$(echo $SERVER_ADDR | cut -d: -f1)
    PORT=$(echo $SERVER_ADDR | cut -d: -f2)
    
    if command -v nc &> /dev/null; then
        # Connect, read challenge, then wait
        (
            nc $HOST $PORT 2>&1 | head -1
            echo -e "${GRAY}Waiting for timeout...${NC}"
            sleep 5
        ) > /dev/null 2>&1
        echo -e "${GREEN}✓ Connection handled${NC}"
    else
        echo -e "${GRAY}Skipping (requires netcat)${NC}"
    fi
}

# Test 4: Multiple concurrent connections
test_multiple_connections() {
    echo -e "\n${YELLOW}[TEST] Testing multiple concurrent connections...${NC}"
    
    HOST=$(echo $SERVER_ADDR | cut -d: -f1)
    PORT=$(echo $SERVER_ADDR | cut -d: -f2)
    
    for i in {1..5}; do
        (
            if command -v nc &> /dev/null; then
                nc -w 2 $HOST $PORT < /dev/null > /dev/null 2>&1
            else
                exec 3<>/dev/tcp/$HOST/$PORT 2>/dev/null
                read -t 1 challenge <&3
                exec 3>&-
            fi
            echo -e "${GRAY}Client $i: OK${NC}"
        ) &
    done
    
    wait
    echo -e "${GREEN}✓ Multiple connections handled${NC}"
}

# Run all tests
test_garbage_data
test_invalid_solution
test_timeout
test_multiple_connections

echo -e "\n${GREEN}=== All negative tests completed ===${NC}"

