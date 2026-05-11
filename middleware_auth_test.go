package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestRateLimit(t *testing.T) { // Requires Server to be up and running
	const url = "http://localhost:8080/public/api/healthz"
	cases := []struct {
		name         string
		requestsNum  int
		sleep        time.Duration
		expectTooMny bool
	}{
		{
			name:         "Below burst number",
			requestsNum:  publicRateBurst,
			sleep:        0,
			expectTooMny: false,
		},
		{
			name:         "Over limit frequency",
			requestsNum:  publicRateBurst + 1,
			sleep:        0,
			expectTooMny: true,
		},
		{
			name:         "Under limit frequency",
			requestsNum:  publicRateBurst + 1,
			sleep:        time.Second / time.Duration(publicRateLimit),
			expectTooMny: false,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("Test case %s", c.name), func(t *testing.T) {
			for id := range c.requestsNum {
				time.Sleep(c.sleep)
				resp, err := http.Get(url)
				if err != nil {
					t.Errorf("Request %d: Error %v\n", id, err)
					return
				}
				if c.expectTooMny && id < publicRateBurst && resp.StatusCode != http.StatusTooManyRequests {
					t.Errorf("Request %d: Expected status 429, got %d\n", id, resp.StatusCode)
				}
				resp.Body.Close()
			}
		})
	}
}
