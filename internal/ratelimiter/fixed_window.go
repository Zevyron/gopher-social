package ratelimiter

import (
	"sync"
	"time"
)

type clientBucket struct {
	count 		int
	expiresAt	time.Time
}

type FixedWindowRateLimiter struct {
	mu		sync.Mutex
	clients	map[string]*clientBucket
	limit 	int
	window 	time.Duration
}

func NewFixedWindowLimiter(limit int, window time.Duration) *FixedWindowRateLimiter {
	rl := &FixedWindowRateLimiter{
		clients: make(map[string]*clientBucket),
		limit: limit,
		window: window,
	}

	go rl.cleanupLoop()

	return rl
}

func (rl *FixedWindowRateLimiter) Allow(ip string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.clients[ip]

	//	First request or window has expired then reset bucket
	if !exists || now.After(bucket.expiresAt) {
		rl.clients[ip] = &clientBucket{
			count: 1,
			expiresAt: now.Add(rl.window),
		}

		return true, 0
	}

	// Window is active and under the limit
	if bucket.count < rl.limit {
		bucket.count++
		return true, 0
	}

	return false, time.Until(bucket.expiresAt)
}

func (rl *FixedWindowRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C{
		rl.mu.Lock()
		now := time.Now()
		for ip, bucket := range rl.clients {
			if now.After(bucket.expiresAt) {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}