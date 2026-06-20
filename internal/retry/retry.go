package retry

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// RetryableFunc is a function that can be retried.
type RetryableFunc func(context.Context) error

// Do executes a retryable function with the given policy.
func Do(ctx context.Context, fn RetryableFunc, policy Policy, log *logrus.Entry) error {
	var lastErr error

	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := policy.GetDelay(attempt)
			log.WithFields(logrus.Fields{
				"attempt": attempt,
				"max":     policy.MaxRetries,
				"delay":   delay.String(),
			}).Debug("retrying operation")

			select {
			case <-ctx.Done():
				return fmt.Errorf("operation cancelled after %d attempts: %w", attempt, ctx.Err())
			case <-time.After(delay):
			}
		}

		err := fn(ctx)
		if err == nil {
			if attempt > 0 {
				log.WithFields(logrus.Fields{
					"attempts": attempt,
				}).Info("operation succeeded after retries")
			}
			return nil
		}

		lastErr = err

		if !policy.IsRetryable(err) {
			log.WithFields(logrus.Fields{
				"attempt": attempt + 1,
				"error":   err.Error(),
			}).Warn("operation failed with non-retryable error")
			return fmt.Errorf("non-retryable error: %w", err)
		}

		log.WithFields(logrus.Fields{
			"attempt": attempt + 1,
			"max":     policy.MaxRetries,
			"error":   err.Error(),
		}).Warn("operation failed, will retry")
	}

	return fmt.Errorf("operation failed after %d attempts: %w", policy.MaxRetries, lastErr)
}

// DoWithData executes a retryable function that returns data.
func DoWithData[T any](ctx context.Context, fn func(context.Context) (T, error), policy Policy, log *logrus.Entry) (T, error) {
	var lastErr error
	var result T

	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := policy.GetDelay(attempt)
			log.WithFields(logrus.Fields{
				"attempt": attempt,
				"max":     policy.MaxRetries,
				"delay":   delay.String(),
			}).Debug("retrying operation")

			select {
			case <-ctx.Done():
				return result, fmt.Errorf("operation cancelled after %d attempts: %w", attempt, ctx.Err())
			case <-time.After(delay):
			}
		}

		result, err := fn(ctx)
		if err == nil {
			if attempt > 0 {
				log.WithFields(logrus.Fields{
					"attempts": attempt,
				}).Info("operation succeeded after retries")
			}
			return result, nil
		}

		lastErr = err

		if !policy.IsRetryable(err) {
			log.WithFields(logrus.Fields{
				"attempt": attempt + 1,
				"error":   err.Error(),
			}).Warn("operation failed with non-retryable error")
			return result, fmt.Errorf("non-retryable error: %w", err)
		}

		log.WithFields(logrus.Fields{
			"attempt": attempt + 1,
			"max":     policy.MaxRetries,
			"error":   err.Error(),
		}).Warn("operation failed, will retry")
	}

	return result, fmt.Errorf("operation failed after %d attempts: %w", policy.MaxRetries, lastErr)
}
