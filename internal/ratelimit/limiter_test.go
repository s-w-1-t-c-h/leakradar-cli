package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestBucket_AllowsBurstThenThrottles(t *testing.T) {
	b := NewBucket(10) // 10/s, burst 10
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < 10; i++ {
		if err := b.Wait(ctx); err != nil {
			t.Fatalf("Wait: %v", err)
		}
	}
	burstElapsed := time.Since(start)
	if burstElapsed > 200*time.Millisecond {
		t.Fatalf("burst of 10 took %s, expected near-instant", burstElapsed)
	}

	// The 11th call should have to wait for a refill (~100ms at 10/s).
	waitStart := time.Now()
	if err := b.Wait(ctx); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	waited := time.Since(waitStart)
	if waited < 50*time.Millisecond {
		t.Fatalf("expected throttling after burst, only waited %s", waited)
	}
}

func TestBucket_RespectsContextCancellation(t *testing.T) {
	b := NewBucket(1)
	ctx := context.Background()
	if err := b.Wait(ctx); err != nil {
		t.Fatalf("Wait: %v", err)
	}

	cctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := b.Wait(cctx); err == nil {
		t.Fatal("expected context deadline error, got nil")
	}
}
