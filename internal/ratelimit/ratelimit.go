package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type ipLimiterEntry struct {
	createdAt time.Time
	limiter   *rate.Limiter
}

type IpLimiter struct {
	ipMap map[string]ipLimiterEntry
	mu    sync.RWMutex
}

func NewIpLimiter(interval time.Duration) *IpLimiter {
	ipLimiter := IpLimiter{
		ipMap: make(map[string]ipLimiterEntry),
		mu:    sync.RWMutex{},
	}
	go ipLimiter.reapLoop(interval)
	return &ipLimiter
}

func (il *IpLimiter) Add(key string, limit rate.Limit, burst int) *rate.Limiter {
	il.mu.Lock()
	defer il.mu.Unlock()

	limiter := rate.NewLimiter(limit, burst)
	il.ipMap[key] = ipLimiterEntry{
		createdAt: time.Now(),
		limiter:   limiter,
	}

	return limiter
}

func (il *IpLimiter) Get(key string) (*rate.Limiter, bool) {
	il.mu.RLock()
	defer il.mu.RUnlock()

	entry, exists := il.ipMap[key]

	return entry.limiter, exists
}

func (il *IpLimiter) reapLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)

	for range ticker.C {
		il.mu.Lock()
		for k, entry := range il.ipMap {
			if time.Since(entry.createdAt) > interval {
				delete(il.ipMap, k)
			}
		}
		il.mu.Unlock()
	}
}
