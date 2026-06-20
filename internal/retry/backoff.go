package retry

import (
	"math"
	"math/rand"
	"time"
)

// BackoffStrategy defines the interface for backoff calculations.
type BackoffStrategy interface {
	Delay(attempt int) time.Duration
}

// FixedBackoff returns a fixed delay between retries.
type FixedBackoff struct {
	FixedDelay time.Duration
}

// Delay returns the fixed delay.
func (b *FixedBackoff) Delay(attempt int) time.Duration {
	return b.FixedDelay
}

// IncrementalBackoff increases delay by a fixed increment each attempt.
type IncrementalBackoff struct {
	InitialDelay time.Duration
	Increment    time.Duration
}

// Delay returns the incremental delay.
func (b *IncrementalBackoff) Delay(attempt int) time.Duration {
	return b.InitialDelay + time.Duration(attempt-1)*b.Increment
}

// ExponentialBackoff implements exponential backoff with jitter.
type ExponentialBackoff struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       float64
}

// DefaultExponentialBackoff creates a default exponential backoff strategy.
func DefaultExponentialBackoff() *ExponentialBackoff {
	return &ExponentialBackoff{
		InitialDelay: 1 * time.Minute,
		MaxDelay:     30 * time.Minute,
		Multiplier:   2.0,
		Jitter:       0.1,
	}
}

// Delay calculates the backoff delay for the given attempt number.
func (b *ExponentialBackoff) Delay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// Calculate exponential delay
	delay := float64(b.InitialDelay) * math.Pow(b.Multiplier, float64(attempt-1))

	// Apply jitter
	if b.Jitter > 0 {
		jitter := delay * b.Jitter * (rand.Float64()*2 - 1)
		delay += jitter
	}

	// Cap at max delay
	if delay > float64(b.MaxDelay) {
		delay = float64(b.MaxDelay)
	}

	// Ensure non-negative
	if delay < 0 {
		delay = 0
	}

	return time.Duration(delay)
}

// ListBackoff uses a predefined list of delays.
type ListBackoff struct {
	Delays []time.Duration
}

// NewListBackoff creates a ListBackoff from a slice of durations.
func NewListBackoff(delays []time.Duration) *ListBackoff {
	return &ListBackoff{Delays: delays}
}

// Delay returns the delay for the given attempt from the list.
func (b *ListBackoff) Delay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	idx := attempt - 1
	if idx >= len(b.Delays) {
		// Use the last delay if we've exhausted the list
		return b.Delays[len(b.Delays)-1]
	}

	return b.Delays[idx]
}
