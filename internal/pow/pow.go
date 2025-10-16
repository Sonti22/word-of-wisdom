// Package pow implements Proof-of-Work logic using hashcash algorithm.
// Challenge generation, solving, and verification.
package pow

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

// Challenge represents a PoW challenge sent by the server.
type Challenge struct {
	Ver       string `json:"ver"`        // Protocol version (e.g., "1")
	Alg       string `json:"alg"`        // Algorithm (e.g., "sha256")
	Bits      int    `json:"bits"`       // Difficulty (leading zero bits required)
	Ts        int64  `json:"ts"`         // Timestamp (Unix seconds)
	ExpiresIn int    `json:"expires_in"` // TTL in seconds
	Resource  string `json:"resource"`   // Resource identifier (e.g., "quote")
	Salt      string `json:"salt"`       // Random salt for uniqueness
}

// Solution represents a PoW solution sent by the client.
type Solution struct {
	Nonce string `json:"nonce"` // Nonce that satisfies the challenge
}

// Generate creates a new PoW challenge with the given difficulty and TTL.
// Salt is generated using crypto/rand.
func Generate(bits int, expiresIn int, resource string) (*Challenge, error) {
	salt, err := generateSalt(16) // 16 bytes = 128 bits
	if err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	return &Challenge{
		Ver:       "1",
		Alg:       "sha256",
		Bits:      bits,
		Ts:        time.Now().Unix(),
		ExpiresIn: expiresIn,
		Resource:  resource,
		Salt:      salt,
	}, nil
}

// generateSalt creates a random hex-encoded salt of the given byte length.
func generateSalt(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// String returns the challenge string used for hashing:
// "ver:alg:bits:ts:expires_in:resource:salt"
func (c *Challenge) String() string {
	return c.StringForHash()
}

// StringForHash returns the canonical challenge string for hashing.
// Format: "ver:alg:bits:ts:expires_in:resource:salt"
func (c *Challenge) StringForHash() string {
	return fmt.Sprintf("%s:%s:%d:%d:%d:%s:%s",
		c.Ver, c.Alg, c.Bits, c.Ts, c.ExpiresIn, c.Resource, c.Salt)
}

// Solve attempts to find a nonce that satisfies the challenge.
// Returns the nonce or an error if no solution is found within maxIterations.
func Solve(c *Challenge, maxIterations int) (string, error) {
	challengeStr := c.StringForHash()

	for i := 0; i < maxIterations; i++ {
		nonce := strconv.Itoa(i)
		input := challengeStr + ":" + nonce
		hash := sha256.Sum256([]byte(input))

		if LeadingZeroBits(hash[:]) >= c.Bits {
			return nonce, nil
		}
	}

	return "", fmt.Errorf("no solution found within %d iterations", maxIterations)
}

// Verify checks if the given solution (nonce) satisfies the challenge.
// Validates: resource match, timestamp window, PoW difficulty.
func Verify(c *Challenge, solution string, resource string) error {
	// 1. Check resource match
	if c.Resource != resource {
		return fmt.Errorf("resource mismatch: expected %s, got %s", resource, c.Resource)
	}

	// 2. Check timestamp window: now must be in [ts, ts+expires_in]
	now := time.Now().Unix()
	if now < c.Ts {
		return fmt.Errorf("challenge not yet valid: ts=%d, now=%d", c.Ts, now)
	}
	if now > c.Ts+int64(c.ExpiresIn) {
		return fmt.Errorf("challenge expired: ts=%d, now=%d, expires_in=%d", c.Ts, now, c.ExpiresIn)
	}

	// 3. Verify PoW: hash(challenge:nonce) has >= bits leading zero bits
	challengeStr := c.StringForHash()
	input := challengeStr + ":" + solution
	hash := sha256.Sum256([]byte(input))

	actualBits := LeadingZeroBits(hash[:])
	if actualBits < c.Bits {
		return fmt.Errorf("insufficient PoW: expected %d leading zero bits, got %d", c.Bits, actualBits)
	}

	return nil
}

// LeadingZeroBits counts the number of leading zero bits in a byte slice.
// Works directly with bytes for accuracy and performance.
func LeadingZeroBits(hash []byte) int {
	count := 0
	for _, b := range hash {
		if b == 0 {
			// All 8 bits are zero
			count += 8
		} else {
			// Count leading zeros in this byte
			for i := 7; i >= 0; i-- {
				if (b & (1 << i)) == 0 {
					count++
				} else {
					return count
				}
			}
			return count
		}
	}
	return count
}
