package limiter

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestValidation(t *testing.T) {
	if _, err := NewTokenBucket(0, 10); err == nil {
		t.Fatal("expected error for zero capacity")
	}
	if _, err := NewTokenBucket(10, -1); err == nil {
		t.Fatal("expected error for negative refill rate")
	}
}

func TestBurstAndRefill(t *testing.T) {
	tb, err := NewTokenBucket(5, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	now := time.Now()
	tb.now = func() time.Time { return now }
	tb.lastRefill = now

	for i := 0; i < 5; i++ {
		if !tb.Allow() {
			t.Fatalf("expected token %d to be allowed", i+1)
		}
	}
	if tb.Allow() {
		t.Fatal("expected bucket to be exhausted")
	}

	now = now.Add(300 * time.Millisecond)
	if !tb.AllowN(3) {
		t.Fatal("expected 3 tokens after 300ms at 10 tokens/sec")
	}
}

func TestConcurrentRaceSafety(t *testing.T) {
	tb, err := NewTokenBucket(50, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var allowed atomic.Int64
	var wg sync.WaitGroup
	for g := 0; g < 100; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tb.Allow() {
				allowed.Add(1)
			}
			_ = tb.Available()
		}()
	}
	wg.Wait()
	if allowed.Load() == 0 || allowed.Load() > 55 {
		t.Fatalf("unexpected allowed count: %d", allowed.Load())
	}
}
