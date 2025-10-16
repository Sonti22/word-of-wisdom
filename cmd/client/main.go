// Package main implements the Word of Wisdom TCP client.
// Client connects to server, receives challenge, solves PoW, and gets a quote.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/mjmln/word-of-wisdom/internal/pow"
	"github.com/mjmln/word-of-wisdom/internal/server"
)

func main() {
	serverAddr := getEnv("SERVER_ADDR", getEnv("WOW_ADDR", "127.0.0.1:8080"))

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if err := run(ctx, serverAddr); err != nil {
		log.Fatalf(`{"level":"fatal","msg":"client failed","error":"%v"}`, err)
	}
}

// getEnv reads an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// run executes the main client logic:
// 1. Connect to server
// 2. Receive challenge
// 3. Solve PoW (parallel)
// 4. Send solution
// 5. Receive quote
func run(ctx context.Context, serverAddr string) error {
	log.Printf(`{"level":"info","msg":"connecting to server","addr":"%s"}`, serverAddr)

	conn, err := net.DialTimeout("tcp", serverAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	log.Printf(`{"level":"info","msg":"connected to server"}`)

	// Set deadline for entire operation
	deadline := time.Now().Add(120 * time.Second)
	if err := conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("failed to set deadline: %w", err)
	}

	// Step 1: Read challenge from server
	var challengeMsg server.ChallengeMessage
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&challengeMsg); err != nil {
		return fmt.Errorf("failed to read challenge: %w", err)
	}

	if challengeMsg.Type != server.TypeChallenge {
		return fmt.Errorf("unexpected message type: %s", challengeMsg.Type)
	}

	challenge := challengeMsg.Challenge
	log.Printf(`{"level":"info","msg":"received challenge","bits":%d,"expires_in":%d}`,
		challenge.Bits, challenge.ExpiresIn)

	// Step 2: Solve PoW (parallel bruteforce)
	solveStart := time.Now()
	nonce, err := solveParallel(ctx, challenge)
	if err != nil {
		return fmt.Errorf("failed to solve PoW: %w", err)
	}
	solveDuration := time.Since(solveStart)

	log.Printf(`{"level":"info","msg":"PoW solved","nonce":"%s","duration_ms":%d}`,
		nonce, solveDuration.Milliseconds())

	// Step 3: Send solution to server
	solutionMsg := server.SolutionMessage{
		Type:  server.TypeSolution,
		Nonce: nonce,
	}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(solutionMsg); err != nil {
		return fmt.Errorf("failed to send solution: %w", err)
	}

	log.Printf(`{"level":"info","msg":"solution sent"}`)

	// Step 4: Read response (quote or error)
	var response map[string]interface{}
	if err := decoder.Decode(&response); err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	responseType, ok := response["type"].(string)
	if !ok {
		return fmt.Errorf("invalid response format: missing type")
	}

	if responseType == server.TypeError {
		errorMsg, _ := response["error"].(string)
		return fmt.Errorf("server error: %s", errorMsg)
	}

	if responseType != server.TypeQuote {
		return fmt.Errorf("unexpected response type: %s", responseType)
	}

	quote, ok := response["quote"].(string)
	if !ok {
		return fmt.Errorf("invalid quote format")
	}

	// Step 5: Print quote to user
	fmt.Printf("\n╔════════════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  Word of Wisdom                                                    ║\n")
	fmt.Printf("╠════════════════════════════════════════════════════════════════════╣\n")
	fmt.Printf("║                                                                    ║\n")
	fmt.Printf("║  %s\n", wrapText(quote, 66))
	fmt.Printf("║                                                                    ║\n")
	fmt.Printf("╠════════════════════════════════════════════════════════════════════╣\n")
	fmt.Printf("║  PoW solved in: %s (bits: %d)                          \n",
		solveDuration.Round(time.Millisecond), challenge.Bits)
	fmt.Printf("╚════════════════════════════════════════════════════════════════════╝\n\n")

	log.Printf(`{"level":"info","msg":"quote received"}`)

	return nil
}

// solveParallel solves the PoW challenge using parallel goroutines.
// Uses all available CPU cores for maximum performance.
func solveParallel(ctx context.Context, challenge *pow.Challenge) (string, error) {
	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}

	log.Printf(`{"level":"debug","msg":"starting parallel solver","workers":%d,"bits":%d}`,
		numWorkers, challenge.Bits)

	// Context for cancellation when solution is found
	solveCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Channel for solution
	solutionChan := make(chan string, 1)
	var wg sync.WaitGroup

	challengeStr := challenge.StringForHash()

	// Start worker goroutines
	for workerID := 0; workerID < numWorkers; workerID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Each worker starts from different offset
			start := id
			step := numWorkers

			for nonce := start; ; nonce += step {
				// Check if context is cancelled
				select {
				case <-solveCtx.Done():
					return
				default:
				}

				// Try this nonce
				nonceStr := strconv.Itoa(nonce)
				input := challengeStr + ":" + nonceStr
				hash := sha256.Sum256([]byte(input))

				// Check leading zero bits
				if pow.LeadingZeroBits(hash[:]) >= challenge.Bits {
					// Solution found! Try to send it
					select {
					case solutionChan <- nonceStr:
						cancel() // Stop other workers
					default:
						// Another worker already found solution
					}
					return
				}

				// Check every 10000 iterations to avoid blocking on context check
				if nonce%10000 == 0 {
					select {
					case <-solveCtx.Done():
						return
					default:
					}
				}
			}
		}(workerID)
	}

	// Wait for solution or timeout
	select {
	case nonce := <-solutionChan:
		wg.Wait() // Wait for all workers to stop
		return nonce, nil
	case <-ctx.Done():
		cancel()
		wg.Wait()
		return "", fmt.Errorf("timeout or cancelled while solving PoW")
	}
}

// wrapText wraps text to fit within a given width (simple implementation).
func wrapText(text string, width int) string {
	if len(text) <= width {
		// Pad with spaces to fill the width
		return fmt.Sprintf("%-*s ║", width, text)
	}

	// Simple truncation for now (proper word wrap would be more complex)
	return text[:width-3] + "... ║"
}
