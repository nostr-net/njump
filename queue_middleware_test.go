package main

import (
	"context"
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

func TestInitQueueBuckets(t *testing.T) {
	initQueueBuckets(2)
	if len(buckets) != 52 {
		t.Fatalf("expected 52 buckets, got %d", len(buckets))
	}
	if err := buckets[13].Acquire(context.Background(), 2); err != nil {
		t.Fatalf("expected to acquire all capacity, got: %v", err)
	}
	buckets[13].Release(2)
}
