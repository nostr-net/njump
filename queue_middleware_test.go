package main

import (
	"testing"
	"time"
)

func TestQueueAcquireTimeoutDurationFromConfig(t *testing.T) {
	s.QueueAcquireTimeoutMs = 2500
	if got, want := queueAcquireTimeoutDuration(), 2500*time.Millisecond; got != want {
		t.Fatalf("expected timeout %s, got %s", want, got)
	}
}

func TestQueueAcquireTimeoutDurationDefault(t *testing.T) {
	s.QueueAcquireTimeoutMs = 0
	if got, want := queueAcquireTimeoutDuration(), 6*time.Second; got != want {
		t.Fatalf("expected default timeout %s, got %s", want, got)
	}
}
