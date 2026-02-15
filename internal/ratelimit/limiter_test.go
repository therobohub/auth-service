package ratelimit

import (
	"sync"
	"testing"
	"time"
)

func TestLimiter_Allow(t *testing.T) {
	t.Run("single request allowed", func(t *testing.T) {
		limiter := NewLimiter(1.0, 1)
		if !limiter.Allow("test/repo") {
			t.Error("expected first request to be allowed")
		}
	})

	t.Run("burst limit", func(t *testing.T) {
		limiter := NewLimiter(1.0, 3)
		repo := "test/repo"

		// First 3 requests should be allowed (burst)
		for i := 0; i < 3; i++ {
			if !limiter.Allow(repo) {
				t.Errorf("expected request %d to be allowed", i+1)
			}
		}

		// 4th request should be denied
		if limiter.Allow(repo) {
			t.Error("expected 4th request to be denied")
		}
	})

	t.Run("rate refill", func(t *testing.T) {
		limiter := NewLimiter(10.0, 1) // 10 requests per second
		repo := "test/repo"

		// Use up the burst
		if !limiter.Allow(repo) {
			t.Error("expected first request to be allowed")
		}

		// Next request should be denied immediately
		if limiter.Allow(repo) {
			t.Error("expected second request to be denied immediately")
		}

		// Wait for token refill (100ms for 10 RPS = 1 token)
		time.Sleep(150 * time.Millisecond)

		// Now should be allowed again
		if !limiter.Allow(repo) {
			t.Error("expected request after refill to be allowed")
		}
	})

	t.Run("per-repository isolation", func(t *testing.T) {
		limiter := NewLimiter(1.0, 1)
		
		repo1 := "test/repo1"
		repo2 := "test/repo2"

		// Both repos should be allowed independently
		if !limiter.Allow(repo1) {
			t.Error("expected repo1 first request to be allowed")
		}

		if !limiter.Allow(repo2) {
			t.Error("expected repo2 first request to be allowed")
		}

		// Both should now be rate limited
		if limiter.Allow(repo1) {
			t.Error("expected repo1 second request to be denied")
		}

		if limiter.Allow(repo2) {
			t.Error("expected repo2 second request to be denied")
		}
	})
}

func TestLimiter_Concurrent(t *testing.T) {
	limiter := NewLimiter(10.0, 10)
	repo := "test/repo"

	var wg sync.WaitGroup
	allowed := 0
	var mu sync.Mutex

	// Launch 20 concurrent requests
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if limiter.Allow(repo) {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// With burst of 10, exactly 10 should be allowed
	if allowed != 10 {
		t.Errorf("expected 10 allowed requests, got %d", allowed)
	}
}

func TestLimiter_Reset(t *testing.T) {
	limiter := NewLimiter(1.0, 1)
	
	limiter.Allow("test/repo1")
	limiter.Allow("test/repo2")

	if count := limiter.GetLimiterCount(); count != 2 {
		t.Errorf("expected 2 limiters, got %d", count)
	}

	limiter.Reset()

	if count := limiter.GetLimiterCount(); count != 0 {
		t.Errorf("expected 0 limiters after reset, got %d", count)
	}

	// Should be able to use after reset
	if !limiter.Allow("test/repo1") {
		t.Error("expected request to be allowed after reset")
	}
}

func TestLimiter_GetLimiterCount(t *testing.T) {
	limiter := NewLimiter(1.0, 1)

	if count := limiter.GetLimiterCount(); count != 0 {
		t.Errorf("expected 0 limiters initially, got %d", count)
	}

	limiter.Allow("test/repo1")
	limiter.Allow("test/repo2")
	limiter.Allow("test/repo1") // Same repo, should not create new limiter

	if count := limiter.GetLimiterCount(); count != 2 {
		t.Errorf("expected 2 limiters, got %d", count)
	}
}

func TestLimiter_HighRPS(t *testing.T) {
	limiter := NewLimiter(100.0, 10)
	repo := "test/repo"

	// Use up burst
	for i := 0; i < 10; i++ {
		if !limiter.Allow(repo) {
			t.Errorf("expected burst request %d to be allowed", i+1)
		}
	}

	// Wait a bit for refill (10ms should give us ~1 token at 100 RPS)
	time.Sleep(20 * time.Millisecond)

	// Should be allowed again
	if !limiter.Allow(repo) {
		t.Error("expected request after refill to be allowed")
	}
}
