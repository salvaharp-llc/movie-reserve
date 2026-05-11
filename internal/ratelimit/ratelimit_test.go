package ratelimit

import (
	"fmt"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestAddGet(t *testing.T) {
	const interval = 5 * time.Second
	cases := []struct {
		ip    string
		limit rate.Limit
		burst int
	}{
		{
			ip:    "69.69.69.69",
			limit: rate.Limit(0),
			burst: 1,
		},
		{
			ip:    "16.16.16.16",
			limit: rate.Limit(1),
			burst: 2,
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("Test case %v", i), func(t *testing.T) {
			ipLimiter := NewIpLimiter(interval)
			ipLimiter.Add(c.ip, c.limit, c.burst)
			limiter, ok := ipLimiter.Get(c.ip)
			if !ok {
				t.Errorf("expected to find ip")
				return
			}
			if limiter.Limit() != c.limit || limiter.Burst() != c.burst {
				t.Errorf("mismatch in limiter values for ip")
				return
			}
		})
	}
}

func TestReapLoop(t *testing.T) {
	const baseTime = 5 * time.Millisecond
	const waitTime = baseTime + 5*time.Millisecond
	ipLimiter := NewIpLimiter(baseTime)
	ipLimiter.Add("69.69.69.69", rate.Limit(0), 1)

	_, ok := ipLimiter.Get("69.69.69.69")
	if !ok {
		t.Errorf("expected to find ip")
		return
	}

	time.Sleep(waitTime)

	_, ok = ipLimiter.Get("69.69.69.69")
	if ok {
		t.Errorf("expected to not find ip")
		return
	}
}
