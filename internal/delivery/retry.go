package delivery

import (
	"math"
	"time"
)

// Backoff calculates exponential backoff with jitter.
// Base delay: 1s, multiplied by 2^attempt, capped at 5 minutes.
func Backoff(attempt int) time.Duration {
	base := float64(time.Second)
	delay := base * math.Pow(2, float64(attempt))

	maxDelay := float64(5 * time.Minute)
	if delay > maxDelay {
		delay = maxDelay
	}

	return time.Duration(delay)
}

// RetryAt returns the absolute time when a retry should be attempted.
func RetryAt(attempt int) time.Time {
	return time.Now().Add(Backoff(attempt))
}

// ShouldRetry determines if a notification should be retried based on attempt count and max retries.
func ShouldRetry(attemptCount, maxRetries int) bool {
	return attemptCount < maxRetries
}
