package common

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"log/slog"
	"math"
	mathrand "math/rand"
	"net/http"
	"time"
)

// ShouldRetry determines if an HTTP status code should trigger a retry
// only retries on server errors (5xx) and rate limits (429)
func ShouldRetry(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || // 429 rate limit
		(statusCode >= 500 && statusCode < 600) // 5xx server errors
}

// calculateBackoff calculates the delay for the given retry attempt
// uses exponential backoff: 3^attempt seconds with ±10% jitter
func calculateBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// create local random generator with crypto/rand seed for better entropy
	var seed int64
	if err := binary.Read(rand.Reader, binary.LittleEndian, &seed); err != nil {
		// fallback to time-based seeding if crypto/rand fails
		seed = time.Now().UnixNano()
	}
	rng := mathrand.New(mathrand.NewSource(seed))

	// base delay: 3^attempt seconds
	baseDelay := math.Pow(3, float64(attempt))

	// add/subtract up to 10% jitter (multiplier between 0.9 and 1.1)
	jitter := 0.9 + rng.Float64()*0.2
	delaySeconds := baseDelay * jitter

	return time.Duration(delaySeconds * float64(time.Second))
}

// HTTPExecutor is a function type that executes an HTTP request
type HTTPExecutor func(ctx context.Context) (*http.Response, error)

// ExecuteWithRetry executes an HTTP request with retry logic
// returns the response and error from the final attempt
func ExecuteWithRetry(ctx context.Context, executor HTTPExecutor, maxRetries int, logger *slog.Logger) (*http.Response, error) {
	// Enforce maximum retry limit of 5
	if maxRetries > 5 {
		maxRetries = 5
	}

	var lastResp *http.Response
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// add backoff delay for retry attempts (not for first attempt)
		if attempt > 0 {
			delay := calculateBackoff(attempt)
			if logger != nil {
				logger.Debug("Retrying HTTP request",
					"attempt", attempt,
					"max_retries", maxRetries,
					"delay_seconds", delay.Seconds())
			}

			// use a timer that respects context cancellation
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
				// Continue with retry
			}
		}

		// execute the HTTP request
		resp, err := executor(ctx)
		lastResp = resp
		lastErr = err

		// if context was cancelled, return immediately
		if ctx.Err() != nil {
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			return nil, ctx.Err()
		}

		// determine if we should retry this attempt
		shouldRetryThisAttempt := false

		if err != nil {
			if resp == nil {
				// transient network error (no response received) - retry!
				shouldRetryThisAttempt = true
				if logger != nil {
					logger.Debug("Network error, will retry",
						"attempt", attempt,
						"error", err.Error())
				}
			} else if ShouldRetry(resp.StatusCode) {
				// HTTP error with retryable status code
				shouldRetryThisAttempt = true
				if logger != nil {
					logger.Debug("HTTP request failed with retryable status, will retry",
						"attempt", attempt,
						"status_code", resp.StatusCode,
						"error", err.Error())
				}
			}
		} else if resp != nil && ShouldRetry(resp.StatusCode) {
			// successful request but retryable status code
			shouldRetryThisAttempt = true
			if logger != nil {
				logger.Debug("HTTP request returned retryable status code",
					"attempt", attempt,
					"status_code", resp.StatusCode)
			}
		}

		// if we shouldn't retry or we've exhausted attempts, return
		if !shouldRetryThisAttempt || attempt >= maxRetries {
			// Clean up response body if we're not returning it
			if err != nil && resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			return resp, err
		}

		// clean up response body before retrying
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}

	// return last response/error as fallback (shouldn't be needed)
	return lastResp, lastErr
}
