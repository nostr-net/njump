package main

import (
	"context"
	"net/http/httptest"
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

func TestQueueKeyUsesDomainAndPath(t *testing.T) {
	req := httptest.NewRequest("GET", "http://127.0.0.1/nevent1abc", nil)
	req = req.WithContext(context.WithValue(req.Context(), "domain", "nostr.ae"))

	if got := queueKey(req); got != "nostr.ae:/nevent1abc" {
		t.Fatalf("expected host/path queue key, got %q", got)
	}
}

func TestQueueKeyUsesOEmbedTargetCode(t *testing.T) {
	req := httptest.NewRequest("GET", "http://127.0.0.1/services/oembed?url=https%3A%2F%2Fnostr.at%2Fnevent1qqqqqqqq", nil)
	req = req.WithContext(context.WithValue(req.Context(), "domain", "nostr.at"))

	if got := queueKey(req); got != "nostr.at:oembed:nevent1qqqqqqqq" {
		t.Fatalf("expected oembed-specific queue key, got %q", got)
	}
}
