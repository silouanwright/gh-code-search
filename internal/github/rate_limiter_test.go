package github

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestNewRateLimiter tests the creation of a new rate limiter with defaults
func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter()

	if rl.maxRetries != 3 {
		t.Errorf("Expected maxRetries to be 3, got %d", rl.maxRetries)
	}
	if rl.baseDelay != time.Second {
		t.Errorf("Expected baseDelay to be 1s, got %v", rl.baseDelay)
	}
	if rl.maxDelay != time.Minute*5 {
		t.Errorf("Expected maxDelay to be 5m, got %v", rl.maxDelay)
	}
	if rl.backoffFactor != 2.0 {
		t.Errorf("Expected backoffFactor to be 2.0, got %f", rl.backoffFactor)
	}
}

// TestNewRateLimiterWithConfig tests custom configuration
func TestNewRateLimiterWithConfig(t *testing.T) {
	config := RateLimiterConfig{
		MaxRetries:    5,
		BaseDelay:     2 * time.Second,
		MaxDelay:      10 * time.Minute,
		BackoffFactor: 1.5,
	}

	rl := NewRateLimiterWithConfig(config)

	if rl.maxRetries != 5 {
		t.Errorf("Expected maxRetries to be 5, got %d", rl.maxRetries)
	}
	if rl.baseDelay != 2*time.Second {
		t.Errorf("Expected baseDelay to be 2s, got %v", rl.baseDelay)
	}
	if rl.maxDelay != 10*time.Minute {
		t.Errorf("Expected maxDelay to be 10m, got %v", rl.maxDelay)
	}
	if rl.backoffFactor != 1.5 {
		t.Errorf("Expected backoffFactor to be 1.5, got %f", rl.backoffFactor)
	}
}

// TestWithRetrySuccess tests successful execution without retries
func TestWithRetrySuccess(t *testing.T) {
	rl := NewRateLimiter()
	ctx := context.Background()

	callCount := 0
	err := rl.WithRetry(ctx, "test operation", func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected function to be called once, got %d", callCount)
	}
}

// TestWithRetryRateLimitError tests retry behavior with rate limit errors
func TestWithRetryRateLimitError(t *testing.T) {
	rl := NewRateLimiter()
	rl.maxRetries = 2
	rl.baseDelay = 10 * time.Millisecond // Speed up test

	ctx := context.Background()

	callCount := 0
	rateLimitErr := &RateLimitError{
		Message:   "Rate limit exceeded",
		ResetTime: 50 * time.Millisecond,
		Limit:     5000,
		Remaining: 0,
	}

	err := rl.WithRetry(ctx, "test operation", func() error {
		callCount++
		if callCount < 3 {
			return rateLimitErr
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected success after retries, got %v", err)
	}
	if callCount != 3 {
		t.Errorf("Expected function to be called 3 times, got %d", callCount)
	}
}

// TestWithRetryAbuseRateLimitError tests abuse detection handling
func TestWithRetryAbuseRateLimitError(t *testing.T) {
	rl := NewRateLimiter()
	rl.maxRetries = 2
	rl.baseDelay = 10 * time.Millisecond

	ctx := context.Background()

	retryAfter := 50 * time.Millisecond
	abuseErr := &AbuseRateLimitError{
		Message:    "Abuse detection triggered",
		RetryAfter: &retryAfter,
	}

	callCount := 0
	err := rl.WithRetry(ctx, "test operation", func() error {
		callCount++
		if callCount < 2 {
			return abuseErr
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected success after retries, got %v", err)
	}
	if callCount != 2 {
		t.Errorf("Expected function to be called 2 times, got %d", callCount)
	}
}

// TestWithRetryServerError tests 5xx error handling
func TestWithRetryServerError(t *testing.T) {
	rl := NewRateLimiter()
	rl.maxRetries = 2
	rl.baseDelay = 10 * time.Millisecond

	ctx := context.Background()

	callCount := 0
	serverErr := errors.New("500 Internal Server Error")

	err := rl.WithRetry(ctx, "test operation", func() error {
		callCount++
		if callCount < 2 {
			return serverErr
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected success after retries, got %v", err)
	}
	if callCount != 2 {
		t.Errorf("Expected function to be called 2 times, got %d", callCount)
	}
}

// TestWithRetryNonRetryableError tests non-retryable errors
func TestWithRetryNonRetryableError(t *testing.T) {
	rl := NewRateLimiter()
	ctx := context.Background()

	callCount := 0
	authErr := &AuthenticationError{Message: "Authentication failed"}

	err := rl.WithRetry(ctx, "test operation", func() error {
		callCount++
		return authErr
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if callCount != 1 {
		t.Errorf("Expected function to be called once, got %d", callCount)
	}
	if !strings.Contains(err.Error(), "non-retryable error") {
		t.Errorf("Expected non-retryable error message, got %v", err)
	}
}

// TestWithRetryMaxRetriesExceeded tests behavior when max retries are exceeded
func TestWithRetryMaxRetriesExceeded(t *testing.T) {
	rl := NewRateLimiter()
	rl.maxRetries = 2
	rl.baseDelay = 5 * time.Millisecond

	ctx := context.Background()

	callCount := 0
	rateLimitErr := &RateLimitError{Message: "Rate limit exceeded"}

	err := rl.WithRetry(ctx, "test operation", func() error {
		callCount++
		return rateLimitErr
	})

	if err == nil {
		t.Error("Expected error after max retries, got nil")
	}
	if callCount != 3 { // Initial call + 2 retries
		t.Errorf("Expected function to be called 3 times, got %d", callCount)
	}
	if !strings.Contains(err.Error(), "after 2 retries") {
		t.Errorf("Expected retry count in error message, got %v", err)
	}
}

// TestWithRetryCancellation tests context cancellation
func TestWithRetryCancellation(t *testing.T) {
	rl := NewRateLimiter()
	rl.baseDelay = 200 * time.Millisecond
	rl.maxRetries = 1 // Reduce retries to make test faster

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	callCount := 0
	err := rl.WithRetry(ctx, "test operation", func() error {
		callCount++
		// Return rate limit error with short reset time
		return &RateLimitError{
			Message:   "Rate limit exceeded",
			ResetTime: 300 * time.Millisecond, // Longer than context timeout
		}
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}

	// The error should indicate cancellation during retry delay
	if !strings.Contains(err.Error(), "cancelled") && !strings.Contains(err.Error(), "context deadline") {
		// If we get a rate limit error, it means the context wasn't properly checked
		t.Logf("Got error: %v", err)
		t.Error("Expected cancellation-related error")
	}
}

// TestAnalyzeErrorAndGetDelay tests error classification and delay calculation
func TestAnalyzeErrorAndGetDelay(t *testing.T) {
	rl := NewRateLimiter()

	tests := []struct {
		name        string
		err         error
		attempt     int
		expectRetry bool
		expectDelay bool
	}{
		{
			name:        "rate limit error",
			err:         &RateLimitError{Message: "Rate limit exceeded", ResetTime: time.Minute},
			attempt:     0,
			expectRetry: true,
			expectDelay: true,
		},
		{
			name:        "abuse rate limit error",
			err:         &AbuseRateLimitError{Message: "Abuse detection"},
			attempt:     0,
			expectRetry: true,
			expectDelay: true,
		},
		{
			name:        "server error 500",
			err:         errors.New("500 Internal Server Error"),
			attempt:     0,
			expectRetry: true,
			expectDelay: true,
		},
		{
			name:        "timeout error",
			err:         errors.New("context deadline exceeded"),
			attempt:     0,
			expectRetry: true,
			expectDelay: true,
		},
		{
			name:        "authentication error",
			err:         &AuthenticationError{Message: "Auth failed"},
			attempt:     0,
			expectRetry: false,
			expectDelay: false,
		},
		{
			name:        "validation error",
			err:         &ValidationError{Message: "Invalid query"},
			attempt:     0,
			expectRetry: false,
			expectDelay: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay, shouldRetry := rl.analyzeErrorAndGetDelay(tt.err, tt.attempt)

			if shouldRetry != tt.expectRetry {
				t.Errorf("Expected shouldRetry=%v, got %v", tt.expectRetry, shouldRetry)
			}

			if tt.expectDelay && delay == 0 {
				t.Error("Expected delay > 0, got 0")
			}

			if !tt.expectDelay && delay > 0 {
				t.Errorf("Expected no delay, got %v", delay)
			}
		})
	}
}

// TestExponentialBackoff tests exponential backoff calculation
func TestExponentialBackoff(t *testing.T) {
	rl := NewRateLimiter()
	rl.baseDelay = 100 * time.Millisecond
	rl.backoffFactor = 2.0
	rl.maxDelay = 1 * time.Second

	// Test exponential increase
	delay0 := rl.calculateExponentialBackoff(0)
	delay1 := rl.calculateExponentialBackoff(1)
	delay2 := rl.calculateExponentialBackoff(2)

	if delay0 != 100*time.Millisecond {
		t.Errorf("Expected delay0 to be 100ms, got %v", delay0)
	}

	if delay1 != 200*time.Millisecond {
		t.Errorf("Expected delay1 to be 200ms, got %v", delay1)
	}

	if delay2 != 400*time.Millisecond {
		t.Errorf("Expected delay2 to be 400ms, got %v", delay2)
	}

	// Test max delay cap
	delay10 := rl.calculateExponentialBackoff(10)
	if delay10 != 1*time.Second {
		t.Errorf("Expected delay to be capped at 1s, got %v", delay10)
	}
}

// TestIntelligentDelay tests complexity-based delays
func TestIntelligentDelay(t *testing.T) {
	rl := NewRateLimiter()
	ctx := context.Background()

	tests := []struct {
		complexity    OperationComplexity
		expectedDelay time.Duration
	}{
		{LowComplexity, 500 * time.Millisecond},
		{MediumComplexity, time.Second},
		{HighComplexity, 2 * time.Second},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.complexity)), func(t *testing.T) {
			start := time.Now()
			err := rl.IntelligentDelay(ctx, tt.complexity)
			elapsed := time.Since(start)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Allow 50ms tolerance for timing
			tolerance := 50 * time.Millisecond
			if elapsed < tt.expectedDelay-tolerance || elapsed > tt.expectedDelay+tolerance {
				t.Errorf("Expected delay ~%v, got %v", tt.expectedDelay, elapsed)
			}
		})
	}
}

// TestEstimateComplexity tests operation complexity estimation
func TestEstimateComplexity(t *testing.T) {
	tests := []struct {
		name               string
		query              string
		maxResults         int
		hasFilters         bool
		expectedComplexity OperationComplexity
	}{
		{
			name:               "simple query",
			query:              "config",
			maxResults:         10,
			hasFilters:         false,
			expectedComplexity: LowComplexity,
		},
		{
			name:               "complex query with wildcards",
			query:              "config AND typescript OR *.json",
			maxResults:         50,
			hasFilters:         true,
			expectedComplexity: MediumComplexity,
		},
		{
			name:               "high complexity query",
			query:              "complex query with many terms AND wildcards *",
			maxResults:         200,
			hasFilters:         true,
			expectedComplexity: HighComplexity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			complexity := EstimateComplexity(tt.query, tt.maxResults, tt.hasFilters)
			if complexity != tt.expectedComplexity {
				t.Errorf("Expected complexity %v, got %v", tt.expectedComplexity, complexity)
			}
		})
	}
}

// TestParseRetryAfterHeader tests HTTP Retry-After header parsing
func TestParseRetryAfterHeader(t *testing.T) {
	tests := []struct {
		name     string
		header   map[string][]string
		expected time.Duration
	}{
		{
			name:     "seconds format",
			header:   map[string][]string{"Retry-After": {"120"}},
			expected: 120 * time.Second,
		},
		{
			name:     "no header",
			header:   map[string][]string{},
			expected: 0,
		},
		{
			name:     "invalid format",
			header:   map[string][]string{"Retry-After": {"invalid"}},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := ParseRetryAfterHeader(tt.header)
			if duration != tt.expected {
				t.Errorf("Expected duration %v, got %v", tt.expected, duration)
			}
		})
	}
}

// BenchmarkWithRetrySuccess benchmarks successful operations
func BenchmarkWithRetrySuccess(b *testing.B) {
	rl := NewRateLimiter()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rl.WithRetry(ctx, "bench operation", func() error {
			return nil
		})
	}
}

// BenchmarkExponentialBackoff benchmarks backoff calculation
func BenchmarkExponentialBackoff(b *testing.B) {
	rl := NewRateLimiter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.calculateExponentialBackoff(i % 10)
	}
}
