package sync

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// rateLimitedClient wraps an *http.Client with a minimum request interval
// and Retry-After backoff on 429/503 responses. It implements both
// http.RoundTripper and go-atlassian's common.HTTPClient.
type rateLimitedClient struct {
	client      *http.Client
	minInterval time.Duration
	mu          sync.Mutex
	lastReq     time.Time
}

func newRateLimitedClient(client *http.Client, reqsPerSec float64) *rateLimitedClient {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	var interval time.Duration
	if reqsPerSec > 0 {
		interval = time.Duration(float64(time.Second) / reqsPerSec)
	}
	return &rateLimitedClient{
		client:      client,
		minInterval: interval,
	}
}

func (c *rateLimitedClient) RoundTrip(req *http.Request) (*http.Response, error) {
	return c.Do(req)
}

func (c *rateLimitedClient) HTTPClient() *http.Client {
	return &http.Client{
		Timeout:   c.client.Timeout,
		Transport: c,
	}
}

func (c *rateLimitedClient) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	const maxRetries = 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if c.minInterval > 0 {
			c.mu.Lock()
			wait := c.minInterval - time.Since(c.lastReq)
			if wait > 0 {
				c.mu.Unlock()
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(wait):
				}
				c.mu.Lock()
			}
			c.lastReq = time.Now()
			c.mu.Unlock()
		}

		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.Request == nil {
			resp.Request = req
		}
		if (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable) && attempt < maxRetries {
			backoff := parseRetryAfter(resp.Header.Get("Retry-After"))
			_ = resp.Body.Close()
			if req.GetBody != nil {
				newBody, bodyErr := req.GetBody()
				if bodyErr == nil {
					req.Body = newBody
				}
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			continue
		}
		return resp, nil
	}
	return nil, context.Canceled
}

func parseRetryAfter(val string) time.Duration {
	if secs, err := strconv.Atoi(val); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return time.Second
}
