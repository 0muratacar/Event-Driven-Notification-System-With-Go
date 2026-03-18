package delivery

import (
	"testing"
	"time"
)

func TestBackoff(t *testing.T) {
	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
		{10, 5 * time.Minute}, // capped
		{20, 5 * time.Minute}, // capped
	}

	for _, tt := range tests {
		got := Backoff(tt.attempt)
		if got != tt.expected {
			t.Errorf("Backoff(%d) = %v, want %v", tt.attempt, got, tt.expected)
		}
	}
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		attemptCount int
		maxRetries   int
		expected     bool
	}{
		{0, 5, true},
		{4, 5, true},
		{5, 5, false},
		{6, 5, false},
		{0, 0, false},
	}

	for _, tt := range tests {
		got := ShouldRetry(tt.attemptCount, tt.maxRetries)
		if got != tt.expected {
			t.Errorf("ShouldRetry(%d, %d) = %v, want %v", tt.attemptCount, tt.maxRetries, got, tt.expected)
		}
	}
}
