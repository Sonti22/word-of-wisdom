// Package main implements the Word of Wisdom TCP server.
// Server generates PoW challenges, verifies solutions, and returns quotes.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/mjmln/word-of-wisdom/internal/server"
)

func main() {
	// Load configuration from environment variables
	cfg := server.Config{
		Addr:         getEnv("WOW_ADDR", ":8080"),
		Bits:         getEnvInt("WOW_BITS", 22),
		ExpiresIn:    getEnvInt("WOW_EXPIRES", 60),
		RateLimit:    getEnvInt("WOW_RATE_LIMIT", 0),         // 0 = disabled
		AdaptiveBits: getEnvBool("WOW_ADAPTIVE_BITS", false), // false = disabled
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := server.Start(ctx, cfg); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case sig := <-sigChan:
		log.Printf(`{"level":"info","msg":"shutdown signal received","signal":"%s"}`, sig)
		cancel()
		// Give server 5 seconds to gracefully shut down
		time.Sleep(5 * time.Second)
	case err := <-errChan:
		log.Fatalf(`{"level":"fatal","msg":"server error","error":"%v"}`, err)
	}

	log.Printf(`{"level":"info","msg":"server stopped"}`)
}

// getEnv reads an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// getEnvInt reads an integer environment variable or returns a default value.
func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultValue
}

// getEnvBool reads a boolean environment variable or returns a default value.
func getEnvBool(key string, defaultValue bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultValue
}
