package quotes

import (
	"testing"
)

// TestRandom_ReturnsNonEmpty verifies that Random() returns a non-empty string.
func TestRandom_ReturnsNonEmpty(t *testing.T) {
	quote, err := Random()
	if err != nil {
		t.Fatalf("Random() failed: %v", err)
	}

	if quote == "" {
		t.Error("Random() returned empty string")
	}
}

// TestRandom_ReturnsQuoteFromPool verifies that returned quote is from the pool.
func TestRandom_ReturnsQuoteFromPool(t *testing.T) {
	quote, err := Random()
	if err != nil {
		t.Fatalf("Random() failed: %v", err)
	}

	// Check that quote exists in pool
	found := false
	for _, q := range pool {
		if q == quote {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Random() returned quote not in pool: %s", quote)
	}
}

// TestRandom_MultipleCalls verifies that function doesn't panic on repeated calls.
func TestRandom_MultipleCalls(t *testing.T) {
	iterations := 100

	for i := 0; i < iterations; i++ {
		quote, err := Random()
		if err != nil {
			t.Fatalf("Random() failed on iteration %d: %v", i, err)
		}
		if quote == "" {
			t.Errorf("Random() returned empty string on iteration %d", i)
		}
	}
}

// TestRandom_Distribution verifies that different quotes are returned over time.
// This is a statistical test - it checks that we get at least 50% of unique quotes
// over 1000 iterations (with 10 quotes, probability of getting all 10 is very high).
func TestRandom_Distribution(t *testing.T) {
	iterations := 1000
	seen := make(map[string]int)

	for i := 0; i < iterations; i++ {
		quote, err := Random()
		if err != nil {
			t.Fatalf("Random() failed: %v", err)
		}
		seen[quote]++
	}

	// We should see at least half of the quotes in the pool
	minUnique := len(pool) / 2
	if len(seen) < minUnique {
		t.Errorf("Random() distribution too narrow: got %d unique quotes, expected at least %d",
			len(seen), minUnique)
	}

	// Log distribution for manual inspection (if needed)
	t.Logf("Distribution over %d iterations:", iterations)
	for quote, count := range seen {
		percentage := float64(count) / float64(iterations) * 100
		// Truncate long quotes for logging
		displayQuote := quote
		if len(displayQuote) > 50 {
			displayQuote = displayQuote[:47] + "..."
		}
		t.Logf("  %5.2f%% - %s", percentage, displayQuote)
	}
}

// TestRandom_AllQuotesEventually verifies that all quotes can be returned.
// This test has a very small probability of failure due to randomness,
// but with 10000 iterations and 10 quotes, it should always pass.
func TestRandom_AllQuotesEventually(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping statistical test in short mode")
	}

	iterations := 10000
	seen := make(map[string]bool)

	for i := 0; i < iterations; i++ {
		quote, err := Random()
		if err != nil {
			t.Fatalf("Random() failed: %v", err)
		}
		seen[quote] = true

		// Early exit if we've seen all quotes
		if len(seen) == len(pool) {
			t.Logf("All quotes seen after %d iterations", i+1)
			return
		}
	}

	if len(seen) < len(pool) {
		missing := []string{}
		for _, q := range pool {
			if !seen[q] {
				missing = append(missing, q)
			}
		}
		t.Errorf("Not all quotes returned after %d iterations. Missing %d quotes: %v",
			iterations, len(missing), missing)
	}
}

// TestPoolNotEmpty verifies that the quote pool is not empty.
func TestPoolNotEmpty(t *testing.T) {
	if len(pool) == 0 {
		t.Error("Quote pool is empty")
	}

	if len(pool) < 5 {
		t.Errorf("Quote pool too small: got %d quotes, expected at least 5", len(pool))
	}
}

// TestPoolQuotesValid verifies that all quotes in the pool are non-empty.
func TestPoolQuotesValid(t *testing.T) {
	for i, quote := range pool {
		if quote == "" {
			t.Errorf("Quote at index %d is empty", i)
		}
		if len(quote) < 10 {
			t.Errorf("Quote at index %d is too short: %q", i, quote)
		}
	}
}

// BenchmarkRandom benchmarks the Random() function.
func BenchmarkRandom(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := Random()
		if err != nil {
			b.Fatalf("Random() failed: %v", err)
		}
	}
}

// TestRandom_ConcurrentSafe verifies that Random() is safe to call concurrently.
func TestRandom_ConcurrentSafe(t *testing.T) {
	// Run multiple goroutines calling Random() simultaneously
	goroutines := 100
	iterations := 10

	done := make(chan bool, goroutines)
	errors := make(chan error, goroutines*iterations)

	for g := 0; g < goroutines; g++ {
		go func() {
			for i := 0; i < iterations; i++ {
				_, err := Random()
				if err != nil {
					errors <- err
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for g := 0; g < goroutines; g++ {
		<-done
	}
	close(errors)

	// Check if any errors occurred
	errorCount := 0
	for err := range errors {
		t.Errorf("Concurrent call failed: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Total concurrent errors: %d", errorCount)
	}
}
