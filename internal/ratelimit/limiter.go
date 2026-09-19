// Package ratelimit provides a small client-side token bucket so the CLI
// self-throttles instead of hammering the LeakRadar API into 429s during
// batch runs.
package ratelimit

import (
	"context"
	"sync"
	"time"
)

// Bucket is a simple token-bucket limiter safe for concurrent use.
type Bucket struct {
	mu         sync.Mutex
	tokens     float64
	max        float64
	refillRate float64 // tokens per second
	last       time.Time
}

// NewBucket creates a bucket that allows ratePerSecond sustained requests,
// with a burst capacity equal to ratePerSecond (i.e. up to 1s of headroom).
func NewBucket(ratePerSecond float64) *Bucket {
	if ratePerSecond <= 0 {
		ratePerSecond = 1
	}
	return &Bucket{
		tokens:     ratePerSecond,
		max:        ratePerSecond,
		refillRate: ratePerSecond,
		last:       time.Now(),
	}
}

// Wait blocks until a token is available or ctx is cancelled.
func (b *Bucket) Wait(ctx context.Context) error {
	for {
		wait, ok := b.reserve()
		if ok {
			return nil
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (b *Bucket) reserve() (time.Duration, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.last = now
	b.tokens += elapsed * b.refillRate
	if b.tokens > b.max {
		b.tokens = b.max
	}

	if b.tokens >= 1 {
		b.tokens--
		return 0, true
	}

	missing := 1 - b.tokens
	wait := time.Duration(missing/b.refillRate*1000) * time.Millisecond
	return wait, false
}
