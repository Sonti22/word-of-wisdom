package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/mjmln/word-of-wisdom/internal/pow"
	"github.com/mjmln/word-of-wisdom/internal/server"
)

const (
	testServerAddr = "127.0.0.1:18080" // Use non-standard port for tests
	testBits       = 16                // Low difficulty for fast tests
	testExpiresIn  = 60
)

// TestIntegration_SuccessFlow tests the full happy path:
// client connects → receives challenge → solves PoW → sends solution → receives quote
func TestIntegration_SuccessFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start server in goroutine
	serverCtx, serverCancel := context.WithCancel(ctx)
	defer serverCancel()

	cfg := server.Config{
		Addr:      testServerAddr,
		Bits:      testBits,
		ExpiresIn: testExpiresIn,
	}

	serverErrCh := make(chan error, 1)
	go func() {
		if err := server.Start(serverCtx, cfg); err != nil {
			serverErrCh <- err
		}
	}()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	// Connect as client
	conn, err := net.DialTimeout("tcp", testServerAddr, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(20 * time.Second))

	// Read challenge
	var challengeMsg server.ChallengeMessage
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&challengeMsg); err != nil {
		t.Fatalf("Failed to read challenge: %v", err)
	}

	if challengeMsg.Type != server.TypeChallenge {
		t.Fatalf("Expected challenge message, got: %s", challengeMsg.Type)
	}

	challenge := challengeMsg.Challenge
	t.Logf("Received challenge: bits=%d, salt=%s", challenge.Bits, challenge.Salt)

	// Solve PoW
	nonce, err := pow.Solve(challenge, 10000000)
	if err != nil {
		t.Fatalf("Failed to solve PoW: %v", err)
	}

	t.Logf("Solved PoW: nonce=%s", nonce)

	// Send solution
	solutionMsg := server.SolutionMessage{
		Type:  server.TypeSolution,
		Nonce: nonce,
	}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(solutionMsg); err != nil {
		t.Fatalf("Failed to send solution: %v", err)
	}

	// Read response
	var response map[string]interface{}
	if err := decoder.Decode(&response); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	responseType, ok := response["type"].(string)
	if !ok {
		t.Fatalf("Response missing type field")
	}

	if responseType != server.TypeQuote {
		t.Fatalf("Expected quote, got: %s (%v)", responseType, response)
	}

	quote, ok := response["quote"].(string)
	if !ok || quote == "" {
		t.Fatalf("Invalid or empty quote: %v", response)
	}

	t.Logf("Received quote: %s", quote)

	// Cleanup
	serverCancel()
}

// TestIntegration_InvalidSolution tests that server rejects invalid solution.
func TestIntegration_InvalidSolution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start server
	serverCtx, serverCancel := context.WithCancel(ctx)
	defer serverCancel()

	cfg := server.Config{
		Addr:      testServerAddr,
		Bits:      testBits,
		ExpiresIn: testExpiresIn,
	}

	go func() {
		server.Start(serverCtx, cfg)
	}()

	time.Sleep(500 * time.Millisecond)

	// Connect as client
	conn, err := net.DialTimeout("tcp", testServerAddr, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(10 * time.Second))

	// Read challenge
	var challengeMsg server.ChallengeMessage
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&challengeMsg); err != nil {
		t.Fatalf("Failed to read challenge: %v", err)
	}

	// Send INVALID solution (nonce that doesn't satisfy PoW)
	invalidSolution := server.SolutionMessage{
		Type:  server.TypeSolution,
		Nonce: "invalid-nonce-123",
	}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(invalidSolution); err != nil {
		t.Fatalf("Failed to send solution: %v", err)
	}

	// Expect error response
	var response map[string]interface{}
	if err := decoder.Decode(&response); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	responseType, ok := response["type"].(string)
	if !ok {
		t.Fatalf("Response missing type field")
	}

	if responseType != server.TypeError {
		t.Fatalf("Expected error response, got: %s", responseType)
	}

	errorMsg, ok := response["error"].(string)
	if !ok || errorMsg == "" {
		t.Fatalf("Invalid or empty error message: %v", response)
	}

	if !strings.Contains(errorMsg, "invalid solution") && !strings.Contains(errorMsg, "insufficient PoW") {
		t.Errorf("Unexpected error message: %s", errorMsg)
	}

	t.Logf("Server correctly rejected invalid solution: %s", errorMsg)

	serverCancel()
}

// TestIntegration_GarbageData tests that server handles garbage data gracefully.
func TestIntegration_GarbageData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start server
	serverCtx, serverCancel := context.WithCancel(ctx)
	defer serverCancel()

	cfg := server.Config{
		Addr:      testServerAddr,
		Bits:      testBits,
		ExpiresIn: testExpiresIn,
	}

	go func() {
		server.Start(serverCtx, cfg)
	}()

	time.Sleep(500 * time.Millisecond)

	// Connect and send garbage
	conn, err := net.DialTimeout("tcp", testServerAddr, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Read challenge first
	decoder := json.NewDecoder(conn)
	var challengeMsg server.ChallengeMessage
	if err := decoder.Decode(&challengeMsg); err != nil {
		t.Fatalf("Failed to read challenge: %v", err)
	}

	// Send garbage data
	_, err = conn.Write([]byte("THIS IS NOT JSON AT ALL\n"))
	if err != nil {
		t.Fatalf("Failed to write garbage: %v", err)
	}

	// Server should either:
	// 1. Close connection (read returns EOF)
	// 2. Send error message

	var response map[string]interface{}
	err = decoder.Decode(&response)
	if err != nil {
		// Connection closed or read error - this is acceptable
		t.Logf("Server closed connection or returned error: %v", err)
	} else {
		// Server sent error message
		responseType, _ := response["type"].(string)
		if responseType != server.TypeError {
			t.Logf("Server sent non-error response to garbage: %v", response)
		} else {
			t.Logf("Server sent error for garbage data: %v", response["error"])
		}
	}

	serverCancel()
}

// TestIntegration_Timeout tests that server closes connection on timeout.
func TestIntegration_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	// Start server with short deadline
	serverCtx, serverCancel := context.WithCancel(ctx)
	defer serverCancel()

	cfg := server.Config{
		Addr:      testServerAddr,
		Bits:      testBits,
		ExpiresIn: testExpiresIn,
	}

	go func() {
		server.Start(serverCtx, cfg)
	}()

	time.Sleep(500 * time.Millisecond)

	// Connect and do nothing (wait for timeout)
	conn, err := net.DialTimeout("tcp", testServerAddr, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Read challenge
	decoder := json.NewDecoder(conn)
	var challengeMsg server.ChallengeMessage
	if err := decoder.Decode(&challengeMsg); err != nil {
		t.Fatalf("Failed to read challenge: %v", err)
	}

	t.Logf("Received challenge, now waiting for timeout...")

	// Don't send anything, wait for server to timeout (30 seconds)
	// Try to read - should get EOF or timeout
	time.Sleep(31 * time.Second)

	var response map[string]interface{}
	err = decoder.Decode(&response)
	if err == nil {
		t.Errorf("Expected timeout/EOF, but got response: %v", response)
	} else {
		t.Logf("Server correctly closed connection after timeout: %v", err)
	}

	serverCancel()
}

// TestIntegration_CLI tests the full CLI client flow by executing the binary.
func TestIntegration_CLI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Determine binary extension (Windows: .exe, Unix: none)
	binExt := ""
	if os.PathSeparator == '\\' {
		binExt = ".exe"
	}

	serverBin := "../bin/server_test" + binExt
	clientBin := "../bin/client_test" + binExt

	// Build binaries first
	t.Log("Building binaries...")
	buildServer := exec.Command("go", "build", "-o", serverBin, "../cmd/server")
	if out, err := buildServer.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build server: %v\n%s", err, out)
	}

	buildClient := exec.Command("go", "build", "-o", clientBin, "../cmd/client")
	if out, err := buildClient.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build client: %v\n%s", err, out)
	}

	// Start server process
	t.Log("Starting server...")
	serverCmd := exec.Command(serverBin)
	serverCmd.Env = append(os.Environ(),
		"WOW_ADDR="+testServerAddr,
		fmt.Sprintf("WOW_BITS=%d", testBits),
		fmt.Sprintf("WOW_EXPIRES=%d", testExpiresIn),
	)

	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer serverCmd.Process.Kill()

	// Wait for server to start
	time.Sleep(2 * time.Second)

	// Run client
	t.Log("Running client...")
	clientCmd := exec.Command(clientBin)
	clientCmd.Env = append(os.Environ(), "WOW_ADDR="+testServerAddr)

	output, err := clientCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Client failed: %v\nOutput:\n%s", err, output)
	}

	t.Logf("Client output:\n%s", output)

	// Verify output contains quote
	outputStr := string(output)
	if !strings.Contains(outputStr, "Word of Wisdom") {
		t.Errorf("Client output doesn't contain expected quote box")
	}

	if !strings.Contains(outputStr, "PoW solved") {
		t.Errorf("Client output doesn't contain solve confirmation")
	}
}
