package retry

import (
	"errors"
	"time"
)

// Policy defines the retry policy.
type Policy struct {
	MaxRetries  int
	Backoff     BackoffStrategy
	RetryableFn func(error) bool
}

// DefaultPolicy returns a default retry policy with exponential backoff.
func DefaultPolicy() Policy {
	return Policy{
		MaxRetries:  3,
		Backoff:     DefaultExponentialBackoff(),
		RetryableFn: DefaultRetryableFn,
	}
}

// NewPolicy creates a new retry policy with the given parameters.
func NewPolicy(maxRetries int, delays []time.Duration) Policy {
	backoff := NewListBackoff(delays)
	return Policy{
		MaxRetries:  maxRetries,
		Backoff:     backoff,
		RetryableFn: DefaultRetryableFn,
	}
}

// GetDelay returns the delay for the given attempt number.
func (p Policy) GetDelay(attempt int) time.Duration {
	if p.Backoff == nil {
		return 0
	}
	return p.Backoff.Delay(attempt)
}

// IsRetryable checks if the error is retryable.
func (p Policy) IsRetryable(err error) bool {
	if p.RetryableFn == nil {
		return true
	}
	return p.RetryableFn(err)
}

// DefaultRetryableFn is the default function to check if an error is retryable.
// By default, all errors are retryable.
func DefaultRetryableFn(err error) bool {
	if err == nil {
		return false
	}

	// Non-retryable errors
	if errors.Is(err, contextCanceled) {
		return false
	}
	if errors.Is(err, contextDeadlineExceeded) {
		return false
	}

	return true
}

// Sentinel errors for retry logic.
var (
	contextCanceled         = errors.New("context canceled")
	contextDeadlineExceeded = errors.New("context deadline exceeded")
)
