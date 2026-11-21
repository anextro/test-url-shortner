package main

import (
	"sync"
	"time"
)

// RateLimiter is an in-memory per-key fixed-window counter.
type RateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	now    func() time.Time
	counts map[string]*bucket
	lastGC time.Time
}

type bucket struct {
	start time.Time
	n     int
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, now: time.Now, counts: make(map[string]*bucket)}
}

// Allow records one attempt for key and reports whether it is within the limit.
func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	if now.Sub(r.lastGC) > r.window {
		for k, b := range r.counts {
			if now.Sub(b.start) >= r.window {
				delete(r.counts, k)
			}
		}
		r.lastGC = now
	}
	b, ok := r.counts[key]
	if !ok || now.Sub(b.start) >= r.window {
		b = &bucket{start: now}
		r.counts[key] = b
	}
	b.n++
	return b.n <= r.limit
}
