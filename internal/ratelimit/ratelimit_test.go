package ratelimit

import (
	"net"
	"sync"
	"testing"
	"time"
)

func TestLimiter_Allow_SingleIP(t *testing.T) {
	limiter := NewLimiter(10, 10) // 10 req/sec, burst 10
	ip := net.ParseIP("192.168.1.1")

	// Should allow first 10 requests (burst)
	for i := 0; i < 10; i++ {
		allowed, attempts := limiter.Allow(ip)
		if !allowed {
			t.Errorf("request %d should be allowed", i)
		}
		if attempts != i+1 {
			t.Errorf("expected attempts=%d, got %d", i+1, attempts)
		}
	}

	// 11th request should be blocked (no refill yet)
	allowed, _ := limiter.Allow(ip)
	if allowed {
		t.Error("request 11 should be blocked")
	}
}

func TestLimiter_Allow_Refill(t *testing.T) {
	limiter := NewLimiter(10, 5) // 10 req/sec, burst 5
	ip := net.ParseIP("192.168.1.1")

	// Consume all tokens
	for i := 0; i < 5; i++ {
		limiter.Allow(ip)
	}

	// Should be blocked
	if allowed, _ := limiter.Allow(ip); allowed {
		t.Error("should be rate limited")
	}

	// Wait 200ms (should refill ~2 tokens at 10/sec)
	time.Sleep(200 * time.Millisecond)

	// Should allow 2 requests
	for i := 0; i < 2; i++ {
		allowed, _ := limiter.Allow(ip)
		if !allowed {
			t.Errorf("request after refill %d should be allowed", i)
		}
	}

	// 3rd should be blocked
	if allowed, _ := limiter.Allow(ip); allowed {
		t.Error("request after 2 refills should be blocked")
	}
}

func TestLimiter_Allow_MultipleIPs(t *testing.T) {
	limiter := NewLimiter(5, 5)
	ip1 := net.ParseIP("192.168.1.1")
	ip2 := net.ParseIP("192.168.1.2")

	// Both IPs should have independent buckets
	for i := 0; i < 5; i++ {
		if allowed, _ := limiter.Allow(ip1); !allowed {
			t.Errorf("ip1 request %d should be allowed", i)
		}
		if allowed, _ := limiter.Allow(ip2); !allowed {
			t.Errorf("ip2 request %d should be allowed", i)
		}
	}

	// Both should be blocked now
	if allowed, _ := limiter.Allow(ip1); allowed {
		t.Error("ip1 should be rate limited")
	}
	if allowed, _ := limiter.Allow(ip2); allowed {
		t.Error("ip2 should be rate limited")
	}
}

func TestLimiter_Reset(t *testing.T) {
	limiter := NewLimiter(5, 5)
	ip := net.ParseIP("192.168.1.1")

	// Consume all tokens
	for i := 0; i < 5; i++ {
		limiter.Allow(ip)
	}

	// Should be blocked
	if allowed, _ := limiter.Allow(ip); allowed {
		t.Error("should be rate limited")
	}

	// Reset
	limiter.Reset(ip)

	// Should allow again (new bucket)
	allowed, attempts := limiter.Allow(ip)
	if !allowed {
		t.Error("should be allowed after reset")
	}
	if attempts != 1 {
		t.Errorf("attempts should be 1 after reset, got %d", attempts)
	}
}

func TestLimiter_ConcurrentAccess(t *testing.T) {
	limiter := NewLimiter(100, 100)
	ip := net.ParseIP("192.168.1.1")

	var wg sync.WaitGroup
	allowed := 0
	var mu sync.Mutex

	// 200 concurrent requests
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ok, _ := limiter.Allow(ip); ok {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Should allow ~100 (capacity)
	if allowed < 90 || allowed > 110 {
		t.Errorf("expected ~100 allowed, got %d", allowed)
	}
}

func TestAdaptiveDifficulty(t *testing.T) {
	baseBits := 20

	tests := []struct {
		attempts int
		expected int
	}{
		{1, 20},   // 1st attempt: base
		{2, 20},   // 2nd: base
		{3, 21},   // 3rd: base+1
		{5, 21},   // 5th: base+1
		{6, 22},   // 6th: base+2
		{10, 22},  // 10th: base+2
		{11, 23},  // 11th: base+3
		{20, 23},  // 20th: base+3
		{21, 24},  // 21st: base+4
		{100, 24}, // 100th: base+4 (capped)
	}

	for _, tt := range tests {
		result := AdaptiveDifficulty(baseBits, tt.attempts)
		if result != tt.expected {
			t.Errorf("AdaptiveDifficulty(%d, %d) = %d, want %d",
				baseBits, tt.attempts, result, tt.expected)
		}
	}
}

func TestLimiter_NilIP(t *testing.T) {
	limiter := NewLimiter(10, 10)

	// Should allow (graceful degradation)
	allowed, attempts := limiter.Allow(nil)
	if !allowed {
		t.Error("nil IP should be allowed")
	}
	if attempts != 0 {
		t.Errorf("nil IP attempts should be 0, got %d", attempts)
	}

	// Reset should not panic
	limiter.Reset(nil)
}

func BenchmarkLimiter_Allow(b *testing.B) {
	limiter := NewLimiter(1000, 1000)
	ip := net.ParseIP("192.168.1.1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ip)
	}
}

func BenchmarkLimiter_AllowParallel(b *testing.B) {
	limiter := NewLimiter(10000, 10000)
	ip := net.ParseIP("192.168.1.1")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			limiter.Allow(ip)
		}
	})
}
