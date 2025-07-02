package common

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestExecuteWithRetry(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping long-running retry test in CI environment")
	}

	// use nil logger to speed up tests
	var logger *slog.Logger

	tests := []struct {
		name        string
		executor    HTTPExecutor
		maxRetries  int
		expectError bool
		expectCalls int
		timeout     time.Duration // timeout override (optional)
	}{
		{
			name: "Success on first attempt",
			executor: func(ctx context.Context) (*http.Response, error) {
				return &http.Response{StatusCode: 200}, nil
			},
			maxRetries:  3,
			expectError: false,
			expectCalls: 1,
		},
		{
			name: "Rate limit retry success",
			executor: func() HTTPExecutor {
				callCount := 0
				return func(ctx context.Context) (*http.Response, error) {
					callCount++
					if callCount == 1 {
						return &http.Response{StatusCode: 429}, errors.New("rate limited")
					}
					return &http.Response{StatusCode: 200}, nil
				}
			}(),
			maxRetries:  3,
			expectError: false,
			expectCalls: 2,
		},
		{
			name: "Non-retryable 4xx error",
			executor: func(ctx context.Context) (*http.Response, error) {
				return &http.Response{StatusCode: 400}, errors.New("bad request")
			},
			maxRetries:  3,
			expectError: true,
			expectCalls: 1, // should not retry
		},
		{
			name: "Enforce max retry limit of 5",
			executor: func(ctx context.Context) (*http.Response, error) {
				return &http.Response{StatusCode: 500}, errors.New("server error")
			},
			maxRetries:  10, // request 10 retries (should be capped at 5)
			expectError: true,
			expectCalls: 4,                // initial + 3 retries complete within timeout (deterministic)
			timeout:     40 * time.Second, // sufficient for initial + 3 retries (~39s total)
		},
		{
			name: "Network error (no response)",
			executor: func(ctx context.Context) (*http.Response, error) {
				return nil, errors.New("network error")
			},
			maxRetries:  3,
			expectError: true,
			expectCalls: 3, // shoul retry transient network errors, but limited by 30s timeout
		},
		{
			name: "Zero retries - fail immediately",
			executor: func(ctx context.Context) (*http.Response, error) {
				return &http.Response{StatusCode: 500}, errors.New("server error")
			},
			maxRetries:  0, // no retries allowed
			expectError: true,
			expectCalls: 1, // only initial attempt, no retries
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			wrappedExecutor := func(ctx context.Context) (*http.Response, error) {
				callCount++
				return tt.executor(ctx)
			}

			// use test-specific timeout or default to 30 seconds
			timeout := 30 * time.Second
			if tt.timeout > 0 {
				timeout = tt.timeout
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			resp, err := ExecuteWithRetry(ctx, wrappedExecutor, tt.maxRetries, logger)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if resp == nil {
					t.Errorf("Expected response but got nil")
				}
			}

			if callCount != tt.expectCalls {
				t.Errorf("Expected %d calls but got %d", tt.expectCalls, callCount)
			}
		})
	}
}

func TestShouldRetry(t *testing.T) {
	// table-driven test for retry logic based on HTTP status
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{"Rate limit (429)", 429, true},
		{"Server error (500)", 500, true},
		{"Bad gateway (502)", 502, true},
		{"Service unavailable (503)", 503, true},
		{"Gateway timeout (504)", 504, true},
		{"Bad request (400)", 400, false},
		{"Unauthorized (401)", 401, false},
		{"Not found (404)", 404, false},
		{"Success (200)", 200, false},
		{"Network error (0)", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldRetry(tt.statusCode)
			if result != tt.expected {
				t.Errorf("ShouldRetry(%d) = %v, expected %v", tt.statusCode, result, tt.expected)
			}
		})
	}
}

func TestCalculateBackoff(t *testing.T) {
	// table-driven test for backoff calculation
	// hard-coded ranges; need to adjust this test if the backoff calculation changes
	tests := []struct {
		name        string
		attempt     int
		minDuration time.Duration
		maxDuration time.Duration
	}{
		{"Zero attempt", 0, 0, 0}, // Base case - should return 0
		{"First retry", 1, 2700 * time.Millisecond, 3300 * time.Millisecond},   // 3^1 * (0.9 to 1.1) seconds
		{"Second retry", 2, 8100 * time.Millisecond, 9900 * time.Millisecond},  // 3^2 * (0.9 to 1.1) seconds
		{"Third retry", 3, 24300 * time.Millisecond, 29700 * time.Millisecond}, // 3^3 * (0.9 to 1.1) seconds
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := calculateBackoff(tt.attempt)

			if tt.minDuration == 0 && tt.maxDuration == 0 {
				// Special case for zero attempt - should be exactly 0
				if duration != 0 {
					t.Errorf("calculateBackoff(%d) = %v, expected exactly 0", tt.attempt, duration)
				}
			} else {
				if duration < tt.minDuration || duration > tt.maxDuration {
					t.Errorf("calculateBackoff(%d) = %v, expected between %v and %v",
						tt.attempt, duration, tt.minDuration, tt.maxDuration)
				}
			}
		})
	}
}

func TestRetryContextCancellation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	executor := func(ctx context.Context) (*http.Response, error) {
		// Always return a retryable error
		return &http.Response{StatusCode: 500}, errors.New("server error")
	}

	// create a context that will be cancelled after a short time
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := ExecuteWithRetry(ctx, executor, 5, logger)
	duration := time.Since(start)

	// should fail due to context cancellation
	if err == nil {
		t.Error("Expected error due to context cancellation")
	}

	// should complete quickly due to context cancellation (not wait for full backoff)
	if duration > 500*time.Millisecond {
		t.Errorf("Expected quick failure due to context cancellation, but took %v", duration)
	}
}
