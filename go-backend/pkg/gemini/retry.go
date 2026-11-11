package gemini

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var retryAfterRegex = regexp.MustCompile(`(?i)retry in ([\d.]+)s`)

// RetryConfig specifies retry behavior for the Gemini client.
type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

// DefaultRetryConfig returns production-ready retry settings for Gemini free/tier-1 quotas.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 2 * time.Second,
		MaxBackoff:     15 * time.Second,
	}
}

// IsRetryable returns true if an HTTP status or error message is transient.
func IsRetryable(statusCode int, body string) bool {
	if statusCode == http.StatusTooManyRequests || statusCode == http.StatusServiceUnavailable {
		return true
	}
	// 500/503 from Google edge proxies can be transient
	if statusCode >= 500 && statusCode <= 599 {
		return true
	}
	msg := strings.ToLower(body)
	return strings.Contains(msg, "quota") || strings.Contains(msg, "rate limit") || strings.Contains(msg, "resource_exhausted")
}

// ExtractRetryDelay attempts to parse a delay hint from the error response or header.
func ExtractRetryDelay(resp *http.Response, body string) time.Duration {
	if resp != nil {
		if headerVal := resp.Header.Get("Retry-After"); headerVal != "" {
			if seconds, err := strconv.Atoi(headerVal); err == nil && seconds > 0 {
				return time.Duration(seconds) * time.Second
			}
		}
	}

	matches := retryAfterRegex.FindStringSubmatch(body)
	if len(matches) > 1 {
		if secFloat, err := strconv.ParseFloat(matches[1], 64); err == nil && secFloat > 0 {
			return time.Duration(secFloat * float64(time.Second))
		}
	}

	return 0
}

// WithRetry executes a function with exponential backoff and jitter.
func WithRetry[T any](ctx context.Context, cfg RetryConfig, op func() (T, int, string, error)) (T, error) {
	var zero T
	var lastErr error
	backoff := cfg.InitialBackoff

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		// Check context cancellation before attempting
		if err := ctx.Err(); err != nil {
			return zero, fmt.Errorf("operation cancelled: %w", err)
		}

		result, statusCode, body, err := op()
		if err == nil && statusCode >= 200 && statusCode < 300 {
			return result, nil
		}

		// Determine if error is retryable
		if !IsRetryable(statusCode, body) || attempt == cfg.MaxRetries {
			if err != nil {
				return zero, err
			}
			return zero, fmt.Errorf("gemini api error (status %d): %s", statusCode, body)
		}

		lastErr = err
		if lastErr == nil {
			lastErr = fmt.Errorf("gemini status %d: %s", statusCode, body)
		}

		// Calculate delay: use API hint if present, else exponential backoff with jitter
		delay := ExtractRetryDelay(nil, body)
		if delay <= 0 {
			// Add ±25% random jitter to avoid synchronized retry storms
			jitter := time.Duration((rand.Float64()*0.5 - 0.25) * float64(backoff))
			delay = backoff + jitter
			backoff *= 2
			if backoff > cfg.MaxBackoff {
				backoff = cfg.MaxBackoff
			}
		}

		select {
		case <-time.After(delay):
			// Proceed to next retry attempt
		case <-ctx.Done():
			return zero, ctx.Err()
		}
	}

	return zero, lastErr
}

