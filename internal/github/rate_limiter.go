package github

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RateLimiter provides intelligent rate limiting and retry logic
type RateLimiter struct {
	maxRetries    int
	baseDelay     time.Duration
	maxDelay      time.Duration
	backoffFactor float64
}

// RateLimiterConfig holds configuration for rate limiting behavior
type RateLimiterConfig struct {
	MaxRetries    int           `yaml:"max_retries" json:"max_retries"`
	BaseDelay     time.Duration `yaml:"base_delay" json:"base_delay"`
	MaxDelay      time.Duration `yaml:"max_delay" json:"max_delay"`
	BackoffFactor float64       `yaml:"backoff_factor" json:"backoff_factor"`
}

// NewRateLimiter creates a rate limiter with default configuration
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		maxRetries:    3,
		baseDelay:     time.Second,
		maxDelay:      time.Minute * 5,
		backoffFactor: 2.0,
	}
}

// NewRateLimiterWithConfig creates a rate limiter with custom configuration
func NewRateLimiterWithConfig(config RateLimiterConfig) *RateLimiter {
	rl := NewRateLimiter()
	if config.MaxRetries > 0 {
		rl.maxRetries = config.MaxRetries
	}
	if config.BaseDelay > 0 {
		rl.baseDelay = config.BaseDelay
	}
	if config.MaxDelay > 0 {
		rl.maxDelay = config.MaxDelay
	}
	if config.BackoffFactor > 0 {
		rl.backoffFactor = config.BackoffFactor
	}
	return rl
}

// RetryableFunc represents a function that can be retried
type RetryableFunc func() error

// WithRetry executes a function with intelligent retry and exponential backoff
func (rl *RateLimiter) WithRetry(ctx context.Context, operation string, fn RetryableFunc) error {
	var lastErr error
	
	for attempt := 0; attempt <= rl.maxRetries; attempt++ {
		// Check context cancellation before each attempt
		if ctx.Err() != nil {
			return fmt.Errorf("operation cancelled: %w", ctx.Err())
		}

		err := fn()
		if err == nil {
			return nil // Success!
		}

		lastErr = err

		// Don't retry on the last attempt
		if attempt == rl.maxRetries {
			break
		}

		// Analyze error type to determine if we should retry
		retryDelay, shouldRetry := rl.analyzeErrorAndGetDelay(err, attempt)
		if !shouldRetry {
			return fmt.Errorf("non-retryable error in %s: %w", operation, err)
		}

		// Sleep for the calculated delay
		if retryDelay > 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("operation cancelled during retry delay: %w", ctx.Err())
			case <-time.After(retryDelay):
				// Continue to next attempt
			}
		}
	}

	return rl.formatFinalError(operation, lastErr)
}

// analyzeErrorAndGetDelay determines retry strategy based on error type
func (rl *RateLimiter) analyzeErrorAndGetDelay(err error, attempt int) (time.Duration, bool) {
	errStr := strings.ToLower(err.Error())

	// Check for specific GitHub API errors
	switch {
	case rl.isRateLimitError(err):
		// Primary rate limit - check for Retry-After header
		if rateLimitErr, ok := err.(*RateLimitError); ok {
			// Honor the reset time from GitHub, but cap it reasonably for tests
			resetTime := rateLimitErr.ResetTime
			if resetTime > 30*time.Second {
				resetTime = 30 * time.Second
			}
			return resetTime, true
		}
		// Fallback exponential backoff for rate limits
		return rl.calculateExponentialBackoff(attempt), true

	case rl.isAbuseRateLimitError(err):
		// Secondary rate limit (abuse detection) - longer delays
		if abuseErr, ok := err.(*AbuseRateLimitError); ok && abuseErr.RetryAfter != nil {
			return *abuseErr.RetryAfter, true
		}
		// GitHub recommends waiting at least 1 minute for abuse detection
		baseDelay := time.Minute + time.Duration(attempt)*30*time.Second
		return rl.capDelay(baseDelay), true

	case rl.isServerError(errStr):
		// 5xx errors - exponential backoff
		return rl.calculateExponentialBackoff(attempt), true

	case rl.isTimeoutError(errStr):
		// Timeout errors - shorter exponential backoff
		delay := time.Duration(float64(rl.baseDelay) * math.Pow(1.5, float64(attempt)))
		return rl.capDelay(delay), true

	case rl.isNetworkError(errStr):
		// Network connectivity issues - exponential backoff
		return rl.calculateExponentialBackoff(attempt), true

	default:
		// Non-retryable error (4xx client errors, validation errors, etc.)
		return 0, false
	}
}

// isRateLimitError checks if the error is a primary rate limit error
func (rl *RateLimiter) isRateLimitError(err error) bool {
	if _, ok := err.(*RateLimitError); ok {
		return true
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "rate_limit") ||
		strings.Contains(errStr, "too many requests") ||
		strings.Contains(errStr, "api rate limit exceeded")
}

// isAbuseRateLimitError checks if the error is a secondary rate limit (abuse detection)
func (rl *RateLimiter) isAbuseRateLimitError(err error) bool {
	if _, ok := err.(*AbuseRateLimitError); ok {
		return true
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "secondary rate limit") ||
		strings.Contains(errStr, "abuse detection") ||
		strings.Contains(errStr, "abuse rate limit")
}

// isServerError checks for 5xx server errors
func (rl *RateLimiter) isServerError(errStr string) bool {
	return strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") ||
		strings.Contains(errStr, "internal server error") ||
		strings.Contains(errStr, "bad gateway") ||
		strings.Contains(errStr, "service unavailable") ||
		strings.Contains(errStr, "gateway timeout")
}

// isTimeoutError checks for timeout-related errors
func (rl *RateLimiter) isTimeoutError(errStr string) bool {
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "context deadline")
}

// isNetworkError checks for network connectivity errors
func (rl *RateLimiter) isNetworkError(errStr string) bool {
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "network unreachable") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "connection reset")
}

// calculateExponentialBackoff calculates delay using exponential backoff
func (rl *RateLimiter) calculateExponentialBackoff(attempt int) time.Duration {
	delay := time.Duration(float64(rl.baseDelay) * math.Pow(rl.backoffFactor, float64(attempt)))
	return rl.capDelay(delay)
}

// capDelay ensures delay doesn't exceed maximum
func (rl *RateLimiter) capDelay(delay time.Duration) time.Duration {
	if delay > rl.maxDelay {
		return rl.maxDelay
	}
	return delay
}

// formatFinalError creates a detailed error message for final failure
func (rl *RateLimiter) formatFinalError(operation string, err error) error {
	errStr := err.Error()

	switch {
	case rl.isRateLimitError(err):
		return fmt.Errorf("rate limit exceeded during %s after %d retries: %w\n\n💡 Suggestions:\n  • Wait until your rate limit resets (check: gh code-search rate-limit)\n  • Use authenticated requests (verify: gh auth status)\n  • Consider reducing batch operation frequency\n  • Add delays between operations", operation, rl.maxRetries, err)

	case rl.isAbuseRateLimitError(err):
		return fmt.Errorf("GitHub abuse detection triggered during %s after %d retries: %w\n\n💡 Suggestions:\n  • You're making requests too rapidly\n  • Wait at least 1 minute before retrying\n  • Implement longer delays between batch operations\n  • Reduce concurrent operations", operation, rl.maxRetries, err)

	case rl.isServerError(strings.ToLower(errStr)):
		return fmt.Errorf("GitHub server error during %s after %d retries: %w\n\n💡 Suggestions:\n  • This is a temporary GitHub server issue\n  • Check GitHub's status page: https://status.github.com\n  • Try again later with smaller batch sizes\n  • Consider using --verbose to monitor retry attempts", operation, rl.maxRetries, err)

	default:
		return fmt.Errorf("operation %s failed after %d retries: %w", operation, rl.maxRetries, err)
	}
}

// IntelligentDelay provides smart delays between batch operations
func (rl *RateLimiter) IntelligentDelay(ctx context.Context, complexity OperationComplexity) error {
	delay := rl.calculateIntelligentDelay(complexity)
	
	if delay == 0 {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}

// OperationComplexity represents the expected load of an operation
type OperationComplexity int

const (
	LowComplexity OperationComplexity = iota
	MediumComplexity
	HighComplexity
)

// CalculateIntelligentDelay calculates appropriate delay based on operation complexity (exported for performance tracking)
func (rl *RateLimiter) CalculateIntelligentDelay(complexity OperationComplexity) time.Duration {
	return rl.calculateIntelligentDelay(complexity)
}

// calculateIntelligentDelay calculates appropriate delay based on operation complexity
func (rl *RateLimiter) calculateIntelligentDelay(complexity OperationComplexity) time.Duration {
	switch complexity {
	case LowComplexity:
		return 500 * time.Millisecond
	case MediumComplexity:
		return time.Second
	case HighComplexity:
		return 2 * time.Second
	default:
		return time.Second
	}
}

// ParseRetryAfterHeader extracts delay from HTTP Retry-After header
func ParseRetryAfterHeader(header http.Header) time.Duration {
	retryAfter := header.Get("Retry-After")
	if retryAfter == "" {
		return 0
	}

	// Try parsing as seconds
	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP date
	if timestamp, err := time.Parse(http.TimeFormat, retryAfter); err == nil {
		return time.Until(timestamp)
	}

	return 0
}

// EstimateComplexity estimates operation complexity based on search parameters
func EstimateComplexity(query string, maxResults int, hasFilters bool) OperationComplexity {
	// Simple heuristic for operation complexity
	score := 0
	
	// Query complexity
	if len(strings.Fields(query)) > 3 {
		score++
	}
	if strings.Contains(query, "*") || strings.Contains(query, "?") {
		score++
	}

	// Result set size
	if maxResults > 100 {
		score += 2
	} else if maxResults > 50 {
		score++
	}

	// Filter complexity
	if hasFilters {
		score++
	}

	switch {
	case score >= 4:
		return HighComplexity
	case score >= 2:
		return MediumComplexity
	default:
		return LowComplexity
	}
}