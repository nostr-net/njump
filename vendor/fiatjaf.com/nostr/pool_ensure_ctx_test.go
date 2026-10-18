package nostr

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Callers blocked on the per-relay lock must abort when their context dies,
// instead of queuing behind an in-flight dial for the same lock bucket.
func TestEnsureRelayAbortsWhenCtxDone(t *testing.T) {
	pool := NewPool(PoolOptions{PenaltyBox: true})
	url := "wss://relay.invalid"
	nm := NormalizeURL(url)

	// hold the lock bucket for this relay name so the EnsureRelay below waits
	unlock, err := namedLock(context.Background(), nm)
	if err != nil {
		t.Fatalf("failed to hold bucket: %v", err)
	}
	defer unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := pool.EnsureRelay(ctx, url)
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if ctx.Err() == nil || !strings.Contains(err.Error(), ctx.Err().Error()) && err != ctx.Err() {
			t.Fatalf("expected context error, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("EnsureRelay did not abort on canceled context")
	}
}

// A failed dial must record the penalty and mark the relay so the next caller
// is rejected immediately instead of serially re-dialing.
func TestEnsureRelayPenaltyBoxEngages(t *testing.T) {
	pool := NewPool(PoolOptions{PenaltyBox: true})
	url := "ws://127.0.0.1:1" // refused immediately
	nm := NormalizeURL(url)

	if _, err := pool.EnsureRelay(context.Background(), url); err == nil {
		t.Fatal("expected dial failure, got nil")
	}

	// marker must exist so the penalty check applies to subsequent calls
	if relay, ok := pool.Relays.Load(nm); !ok || relay != nil {
		t.Fatalf("expected nil marker stored for %s, got ok=%v relay=%v", nm, ok, relay)
	}
	pool.penaltyBoxMu.Lock()
	v := pool.penaltyBox[nm]
	pool.penaltyBoxMu.Unlock()
	if v[1] <= 0 {
		t.Fatalf("expected penalty remaining > 0, got %v", v[1])
	}

	start := time.Now()
	_, err := pool.EnsureRelay(context.Background(), url)
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("penalty-box rejection took %v, expected immediate", elapsed)
	}
	if err == nil || !strings.Contains(err.Error(), "penalty box") {
		t.Fatalf("expected penalty box error, got %v", err)
	}
}
