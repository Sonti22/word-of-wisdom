package pow

import (
	"crypto/sha256"
	"testing"
	"time"
)

// TestLeadingZeroBits verifies the bit counting function.
func TestLeadingZeroBits(t *testing.T) {
	tests := []struct {
		name     string
		hash     []byte
		expected int
	}{
		{
			name:     "all zeros",
			hash:     []byte{0x00, 0x00, 0x00, 0x00},
			expected: 32,
		},
		{
			name:     "no leading zeros",
			hash:     []byte{0xFF, 0xFF, 0xFF, 0xFF},
			expected: 0,
		},
		{
			name:     "4 leading zero bits",
			hash:     []byte{0x0F, 0xFF, 0xFF, 0xFF},
			expected: 4,
		},
		{
			name:     "8 leading zero bits",
			hash:     []byte{0x00, 0xFF, 0xFF, 0xFF},
			expected: 8,
		},
		{
			name:     "16 leading zero bits",
			hash:     []byte{0x00, 0x00, 0xFF, 0xFF},
			expected: 16,
		},
		{
			name:     "1 leading zero bit",
			hash:     []byte{0x7F, 0xFF, 0xFF, 0xFF},
			expected: 1,
		},
		{
			name:     "3 leading zero bits",
			hash:     []byte{0x1F, 0xFF, 0xFF, 0xFF},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := LeadingZeroBits(tt.hash)
			if actual != tt.expected {
				t.Errorf("LeadingZeroBits() = %d, expected %d", actual, tt.expected)
			}
		})
	}
}

// TestGenerateChallenge verifies challenge generation.
func TestGenerateChallenge(t *testing.T) {
	bits := 20
	expiresIn := 300
	resource := "quote"

	challenge, err := Generate(bits, expiresIn, resource)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Verify structure
	if challenge.Ver != "1" {
		t.Errorf("Ver = %s, expected 1", challenge.Ver)
	}
	if challenge.Alg != "sha256" {
		t.Errorf("Alg = %s, expected sha256", challenge.Alg)
	}
	if challenge.Bits != bits {
		t.Errorf("Bits = %d, expected %d", challenge.Bits, bits)
	}
	if challenge.ExpiresIn != expiresIn {
		t.Errorf("ExpiresIn = %d, expected %d", challenge.ExpiresIn, expiresIn)
	}
	if challenge.Resource != resource {
		t.Errorf("Resource = %s, expected %s", challenge.Resource, resource)
	}
	if challenge.Salt == "" {
		t.Error("Salt is empty")
	}
	if len(challenge.Salt) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("Salt length = %d, expected 32", len(challenge.Salt))
	}

	// Verify timestamp is recent (within 1 second)
	now := time.Now().Unix()
	if challenge.Ts < now-1 || challenge.Ts > now+1 {
		t.Errorf("Ts = %d, now = %d, difference too large", challenge.Ts, now)
	}
}

// TestStringForHash verifies the canonical string format.
func TestStringForHash(t *testing.T) {
	challenge := &Challenge{
		Ver:       "1",
		Alg:       "sha256",
		Bits:      20,
		Ts:        1234567890,
		ExpiresIn: 300,
		Resource:  "quote",
		Salt:      "abcdef1234567890",
	}

	expected := "1:sha256:20:1234567890:300:quote:abcdef1234567890"
	actual := challenge.StringForHash()

	if actual != expected {
		t.Errorf("StringForHash() = %s, expected %s", actual, expected)
	}

	// Verify String() uses StringForHash()
	if challenge.String() != expected {
		t.Errorf("String() = %s, expected %s", challenge.String(), expected)
	}
}

// TestSolveAndVerify_Success tests the positive case: solve and verify.
func TestSolveAndVerify_Success(t *testing.T) {
	// Use low difficulty for fast test
	bits := 8
	expiresIn := 300
	resource := "quote"

	challenge, err := Generate(bits, expiresIn, resource)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Solve the challenge
	nonce, err := Solve(challenge, 100000)
	if err != nil {
		t.Fatalf("Solve() failed: %v", err)
	}

	// Verify the solution
	if err := Verify(challenge, nonce, resource); err != nil {
		t.Errorf("Verify() failed: %v", err)
	}

	// Double-check: manually compute hash and verify leading zeros
	input := challenge.StringForHash() + ":" + nonce
	hash := sha256.Sum256([]byte(input))
	actualBits := LeadingZeroBits(hash[:])
	if actualBits < bits {
		t.Errorf("Solution has %d leading zero bits, expected at least %d", actualBits, bits)
	}
}

// TestVerify_ExpiredChallenge verifies that expired challenges are rejected.
func TestVerify_ExpiredChallenge(t *testing.T) {
	// Create an old challenge (expired)
	challenge := &Challenge{
		Ver:       "1",
		Alg:       "sha256",
		Bits:      8,
		Ts:        time.Now().Unix() - 400, // 400 seconds ago
		ExpiresIn: 300,                     // expires after 300 seconds
		Resource:  "quote",
		Salt:      "testsalt",
	}

	// Find a valid nonce (for testing expiration logic)
	nonce, err := Solve(challenge, 100000)
	if err != nil {
		t.Fatalf("Solve() failed: %v", err)
	}

	// Verify should fail due to expiration
	err = Verify(challenge, nonce, "quote")
	if err == nil {
		t.Error("Verify() should fail for expired challenge")
	}
	if err != nil && err.Error() != "challenge expired: ts="+challenge.String() {
		// Check that error message mentions expiration
		if err.Error()[:18] != "challenge expired:" {
			t.Errorf("Expected expiration error, got: %v", err)
		}
	}
}

// TestVerify_ResourceMismatch verifies that resource mismatch is rejected.
func TestVerify_ResourceMismatch(t *testing.T) {
	bits := 8
	expiresIn := 300
	resource := "quote"

	challenge, err := Generate(bits, expiresIn, resource)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	nonce, err := Solve(challenge, 100000)
	if err != nil {
		t.Fatalf("Solve() failed: %v", err)
	}

	// Verify with wrong resource
	err = Verify(challenge, nonce, "different-resource")
	if err == nil {
		t.Error("Verify() should fail for resource mismatch")
	}
	if err != nil && err.Error()[:18] != "resource mismatch:" {
		t.Errorf("Expected resource mismatch error, got: %v", err)
	}
}

// TestVerify_InsufficientWork verifies that insufficient PoW is rejected.
func TestVerify_InsufficientWork(t *testing.T) {
	bits := 20
	expiresIn := 300
	resource := "quote"

	challenge, err := Generate(bits, expiresIn, resource)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Use a nonce that doesn't satisfy the difficulty
	invalidNonce := "0"

	err = Verify(challenge, invalidNonce, resource)
	if err == nil {
		t.Error("Verify() should fail for insufficient PoW")
	}
	if err != nil && err.Error()[:17] != "insufficient PoW:" {
		t.Errorf("Expected insufficient PoW error, got: %v", err)
	}
}

// TestVerify_InvalidNonce verifies that random invalid nonce is rejected.
func TestVerify_InvalidNonce(t *testing.T) {
	bits := 16
	expiresIn := 300
	resource := "quote"

	challenge, err := Generate(bits, expiresIn, resource)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Use a completely invalid nonce
	invalidNonce := "this-will-never-work"

	err = Verify(challenge, invalidNonce, resource)
	if err == nil {
		t.Error("Verify() should fail for invalid nonce")
	}
}

// TestSolve_NoSolutionFound verifies that Solve returns error if max iterations reached.
func TestSolve_NoSolutionFound(t *testing.T) {
	bits := 30 // Very high difficulty
	expiresIn := 300
	resource := "quote"

	challenge, err := Generate(bits, expiresIn, resource)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Try to solve with very low iteration limit
	_, err = Solve(challenge, 10)
	if err == nil {
		t.Error("Solve() should fail with low iteration limit and high difficulty")
	}
	if err != nil && err.Error()[:19] != "no solution found w" {
		t.Errorf("Expected 'no solution found' error, got: %v", err)
	}
}

// TestVerify_FutureTimestamp verifies that challenges with future timestamp are rejected.
func TestVerify_FutureTimestamp(t *testing.T) {
	// Create a challenge with future timestamp
	challenge := &Challenge{
		Ver:       "1",
		Alg:       "sha256",
		Bits:      8,
		Ts:        time.Now().Unix() + 100, // 100 seconds in the future
		ExpiresIn: 300,
		Resource:  "quote",
		Salt:      "testsalt",
	}

	// Find a valid nonce (just for testing)
	nonce, err := Solve(challenge, 100000)
	if err != nil {
		t.Fatalf("Solve() failed: %v", err)
	}

	// Verify should fail due to future timestamp
	err = Verify(challenge, nonce, "quote")
	if err == nil {
		t.Error("Verify() should fail for future timestamp")
	}
	if err != nil && err.Error()[:24] != "challenge not yet valid:" {
		t.Errorf("Expected 'not yet valid' error, got: %v", err)
	}
}

// BenchmarkSolve benchmarks the solving process for different difficulties.
func BenchmarkSolve_8bits(b *testing.B) {
	challenge, _ := Generate(8, 300, "quote")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Solve(challenge, 100000)
	}
}

func BenchmarkSolve_16bits(b *testing.B) {
	challenge, _ := Generate(16, 300, "quote")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Solve(challenge, 10000000)
	}
}

// BenchmarkLeadingZeroBits benchmarks the bit counting function.
func BenchmarkLeadingZeroBits(b *testing.B) {
	hash := sha256.Sum256([]byte("test"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LeadingZeroBits(hash[:])
	}
}
