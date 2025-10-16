// Package server provides helper utilities for TCP protocol handling.
// JSON encoding/decoding, error responses, message types, and server logic.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/mjmln/word-of-wisdom/internal/pow"
	"github.com/mjmln/word-of-wisdom/internal/quotes"
	"github.com/mjmln/word-of-wisdom/internal/ratelimit"
)

// Message types for client-server communication.
const (
	TypeChallenge = "challenge"
	TypeSolution  = "solution"
	TypeQuote     = "quote"
	TypeError     = "error"
)

// ChallengeMessage is sent from server to client.
type ChallengeMessage struct {
	Type      string         `json:"type"`      // "challenge"
	Challenge *pow.Challenge `json:"challenge"` // PoW Challenge
}

// SolutionMessage is sent from client to server.
type SolutionMessage struct {
	Type  string `json:"type"`  // "solution"
	Nonce string `json:"nonce"` // Nonce value
}

// QuoteMessage is sent from server to client after successful PoW.
type QuoteMessage struct {
	Type  string `json:"type"`  // "quote"
	Quote string `json:"quote"` // The wisdom quote
}

// ErrorMessage is sent from server to client on failure.
type ErrorMessage struct {
	Type  string `json:"type"`  // "error"
	Error string `json:"error"` // Error description
}

// Config holds server configuration.
type Config struct {
	Addr         string
	Bits         int
	ExpiresIn    int
	RateLimit    int  // requests per second per IP (0 = disabled)
	AdaptiveBits bool // enable adaptive difficulty based on attempts
}

// SendJSON encodes and sends a JSON message over the connection.
func SendJSON(conn net.Conn, v interface{}) error {
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(v); err != nil {
		return fmt.Errorf("failed to send JSON: %w", err)
	}
	return nil
}

// ReceiveJSON decodes a JSON message from the connection.
func ReceiveJSON(conn net.Conn, v interface{}) error {
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("failed to receive JSON: %w", err)
	}
	return nil
}

// SendError is a helper to send an error message to the client.
func SendError(conn net.Conn, errMsg string) error {
	return SendJSON(conn, ErrorMessage{
		Type:  TypeError,
		Error: errMsg,
	})
}

// Start starts the TCP server and handles incoming connections.
// Blocks until context is cancelled or listener fails.
func Start(ctx context.Context, cfg Config) error {
	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", cfg.Addr, err)
	}
	defer listener.Close()

	// Initialize rate limiter if enabled
	var limiter *ratelimit.Limiter
	if cfg.RateLimit > 0 {
		limiter = ratelimit.NewLimiter(cfg.RateLimit, cfg.RateLimit*2) // burst = 2x rate
		log.Printf(`{"level":"info","msg":"rate limiter enabled","rate":%d,"burst":%d}`,
			cfg.RateLimit, cfg.RateLimit*2)
	}

	log.Printf(`{"level":"info","msg":"server started","addr":"%s","bits":%d,"expires_in":%d,"rate_limit":%d,"adaptive_bits":%t}`,
		cfg.Addr, cfg.Bits, cfg.ExpiresIn, cfg.RateLimit, cfg.AdaptiveBits)

	// Channel to signal listener to stop
	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		listener.Close()
		close(done)
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				log.Printf(`{"level":"info","msg":"server shutting down"}`)
				return nil
			default:
				log.Printf(`{"level":"error","msg":"accept failed","error":"%v"}`, err)
				continue
			}
		}

		go handleConnection(conn, cfg, limiter)
	}
}

// handleConnection processes a single client connection.
// Flow: send challenge → read solution → verify → send quote/error.
func handleConnection(conn net.Conn, cfg Config, limiter *ratelimit.Limiter) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()

	// Extract IP for rate limiting
	var clientIP net.IP
	if tcpAddr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		clientIP = tcpAddr.IP
	}

	// Check rate limit
	var attempts int
	if limiter != nil && clientIP != nil {
		allowed, attemptCount := limiter.Allow(clientIP)
		attempts = attemptCount
		if !allowed {
			log.Printf(`{"level":"warn","msg":"rate limited","remote":"%s","attempts":%d}`,
				remoteAddr, attempts)
			SendError(conn, "rate limit exceeded, please try again later")
			return
		}
	}

	log.Printf(`{"level":"info","msg":"connection accepted","remote":"%s","attempts":%d}`,
		remoteAddr, attempts)

	// Set deadline for entire operation (30 seconds)
	deadline := time.Now().Add(30 * time.Second)
	if err := conn.SetDeadline(deadline); err != nil {
		log.Printf(`{"level":"error","msg":"failed to set deadline","remote":"%s","error":"%v"}`, remoteAddr, err)
		return
	}

	// Step 1: Calculate adaptive difficulty
	bits := cfg.Bits
	if cfg.AdaptiveBits && attempts > 0 {
		bits = ratelimit.AdaptiveDifficulty(cfg.Bits, attempts)
		if bits > cfg.Bits {
			log.Printf(`{"level":"info","msg":"adaptive difficulty increased","remote":"%s","base_bits":%d,"new_bits":%d,"attempts":%d}`,
				remoteAddr, cfg.Bits, bits, attempts)
		}
	}

	// Step 2: Generate challenge
	challenge, err := pow.Generate(bits, cfg.ExpiresIn, "quote")
	if err != nil {
		log.Printf(`{"level":"error","msg":"failed to generate challenge","remote":"%s","error":"%v"}`, remoteAddr, err)
		SendError(conn, "internal server error")
		return
	}

	// Step 3: Send challenge to client
	challengeMsg := ChallengeMessage{
		Type:      TypeChallenge,
		Challenge: challenge,
	}
	if err := SendJSON(conn, challengeMsg); err != nil {
		log.Printf(`{"level":"error","msg":"failed to send challenge","remote":"%s","error":"%v"}`, remoteAddr, err)
		return
	}

	log.Printf(`{"level":"debug","msg":"challenge sent","remote":"%s","bits":%d,"salt":"%s","attempts":%d}`,
		remoteAddr, challenge.Bits, challenge.Salt, attempts)

	// Step 3: Read solution from client
	var solutionMsg SolutionMessage
	if err := ReceiveJSON(conn, &solutionMsg); err != nil {
		log.Printf(`{"level":"error","msg":"failed to read solution","remote":"%s","error":"%v"}`, remoteAddr, err)
		SendError(conn, "invalid solution format")
		return
	}

	if solutionMsg.Type != TypeSolution {
		log.Printf(`{"level":"error","msg":"invalid message type","remote":"%s","type":"%s"}`, remoteAddr, solutionMsg.Type)
		SendError(conn, "expected solution message")
		return
	}

	// Step 4: Verify PoW solution
	if err := pow.Verify(challenge, solutionMsg.Nonce, "quote"); err != nil {
		log.Printf(`{"level":"warn","msg":"invalid solution","remote":"%s","error":"%v"}`, remoteAddr, err)
		SendError(conn, fmt.Sprintf("invalid solution: %v", err))
		return
	}

	log.Printf(`{"level":"info","msg":"solution verified","remote":"%s","nonce":"%s"}`, remoteAddr, solutionMsg.Nonce)

	// Reset rate limit for this IP after successful PoW (optional)
	if limiter != nil && clientIP != nil {
		limiter.Reset(clientIP)
	}

	// Step 5: Get random quote
	quote, err := quotes.Random()
	if err != nil {
		log.Printf(`{"level":"error","msg":"failed to get quote","remote":"%s","error":"%v"}`, remoteAddr, err)
		SendError(conn, "failed to generate quote")
		return
	}

	// Step 6: Send quote to client
	quoteMsg := QuoteMessage{
		Type:  TypeQuote,
		Quote: quote,
	}
	if err := SendJSON(conn, quoteMsg); err != nil {
		log.Printf(`{"level":"error","msg":"failed to send quote","remote":"%s","error":"%v"}`, remoteAddr, err)
		return
	}

	log.Printf(`{"level":"info","msg":"quote sent","remote":"%s"}`, remoteAddr)
}
