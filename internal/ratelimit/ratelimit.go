// Package ratelimit provides IP-based rate limiting using token bucket algorithm.
package ratelimit

import (
	"net"
	"sync"
	"time"
)

// Limiter manages rate limits for multiple IPs using token bucket algorithm.
type Limiter struct {
	mu       sync.RWMutex
	buckets  map[string]*bucket
	rate     int           // tokens per second
	capacity int           // max tokens
	cleanup  time.Duration // cleanup interval for old entries
}

// bucket represents a token bucket for a single IP.
type bucket struct {
	tokens    float64
	lastCheck time.Time
	attempts  int // total attempts (for adaptive difficulty)
}

// NewLimiter creates a new rate limiter.
// rate: tokens per second (e.g., 10 = 10 requests/sec)
// capacity: max tokens in bucket (burst size)
func NewLimiter(rate, capacity int) *Limiter {
	l := &Limiter{
		buckets:  make(map[string]*bucket),
		rate:     rate,
		capacity: capacity,
		cleanup:  5 * time.Minute,
	}
	go l.cleanupLoop()
	return l
}

// Allow checks if the IP is allowed to make a request.
// Returns (allowed bool, attempts int).
func (l *Limiter) Allow(ip net.IP) (bool, int) {
	if ip == nil {
		return true, 0 // allow if IP is nil (shouldn't happen)
	}

	key := ip.String()
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, exists := l.buckets[key]
	if !exists {
		b = &bucket{
			tokens:    float64(l.capacity),
			lastCheck: now,
			attempts:  0,
		}
		l.buckets[key] = b
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(b.lastCheck).Seconds()
	b.tokens += elapsed * float64(l.rate)
	if b.tokens > float64(l.capacity) {
		b.tokens = float64(l.capacity)
	}
	b.lastCheck = now

	// Check if token available
	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		b.attempts++
		return true, b.attempts
	}

	// Rate limited
	b.attempts++
	return false, b.attempts
}

// Reset clears rate limit state for an IP (e.g., after successful PoW).
func (l *Limiter) Reset(ip net.IP) {
	if ip == nil {
		return
	}

	key := ip.String()
	l.mu.Lock()
	delete(l.buckets, key)
	l.mu.Unlock()
}

// cleanupLoop periodically removes old entries.
func (l *Limiter) cleanupLoop() {
	ticker := time.NewTicker(l.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for key, b := range l.buckets {
			// Remove entries older than 2x cleanup interval
			if now.Sub(b.lastCheck) > 2*l.cleanup {
				delete(l.buckets, key)
			}
		}
		l.mu.Unlock()
	}
}

// AdaptiveDifficulty calculates adaptive PoW difficulty based on attempts.
// baseBits: default difficulty (e.g., 20)
// attempts: number of attempts from this IP
// Returns adjusted difficulty (caps at baseBits + 4).
func AdaptiveDifficulty(baseBits, attempts int) int {
	// Increase difficulty for repeated attempts:
	// 1-2 attempts: baseBits
	// 3-5 attempts: baseBits + 1
	// 6-10 attempts: baseBits + 2
	// 11-20 attempts: baseBits + 3
	// 21+ attempts: baseBits + 4
	switch {
	case attempts <= 2:
		return baseBits
	case attempts <= 5:
		return baseBits + 1
	case attempts <= 10:
		return baseBits + 2
	case attempts <= 20:
		return baseBits + 3
	default:
		return baseBits + 4
	}
}
