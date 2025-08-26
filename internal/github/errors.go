package github

import "time"

// Custom error types for better error handling

// RateLimitError represents a GitHub API rate limit error
type RateLimitError struct {
	Message   string
	ResetTime time.Duration
	Limit     int
	Remaining int
}

func (e *RateLimitError) Error() string {
	return e.Message
}

// AbuseRateLimitError represents a GitHub API abuse detection error
type AbuseRateLimitError struct {
	Message    string
	RetryAfter *time.Duration
}

func (e *AbuseRateLimitError) Error() string {
	return e.Message
}

// AuthenticationError represents an authentication failure
type AuthenticationError struct {
	Message string
}

func (e *AuthenticationError) Error() string {
	return e.Message
}

// AuthorizationError represents an authorization failure
type AuthorizationError struct {
	Message string
}

func (e *AuthorizationError) Error() string {
	return e.Message
}

// NotFoundError represents a resource not found error
type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string {
	return e.Message
}

// ValidationError represents a query validation error
type ValidationError struct {
	Message string
	Errors  []string
}

func (e *ValidationError) Error() string {
	if len(e.Errors) > 0 {
		return e.Message + ": " + e.Errors[0]
	}
	return e.Message
}

// APIError represents a generic API error
type APIError struct {
	Message string
}

func (e *APIError) Error() string {
	return e.Message
}
