package main

import (
	"context"
	"runtime/debug"
	"time"

	"fiatjaf.com/nostr"
)

// pruneIdleRelayConnections periodically closes relay connections that have no
// active subscriptions. The nostr-sdk relay pool grows without bound as the
// outbox model discovers relays for each unique pubkey. Each idle websocket
// carries ~200-800KB of buffers and compression state (CompressionContextTakeover).
//
// Each NewRelay starts a goroutine waiting on <-ctx.Done() (derived from the
// pool context). EnsureRelay overwrites disconnected entries without closing
// the old relay, so the old goroutine leaks. The pruner must call Close() to
// cancel the relay's context before removing it from the map.
func pruneIdleRelayConnections(ctx context.Context) {
	const interval = 2 * time.Minute

	// Relays that should never be pruned (configured in relay config).
	keep := make(map[string]struct{})
	for _, url := range sys.FallbackRelays.URLs {
		keep[url] = struct{}{}
	}
	for _, url := range sys.MetadataRelays.URLs {
		keep[url] = struct{}{}
	}
	for _, url := range sys.RelayListRelays.URLs {
		keep[url] = struct{}{}
	}
	for _, url := range sys.FollowListRelays.URLs {
		keep[url] = struct{}{}
	}
	for _, url := range sys.JustIDRelays.URLs {
		keep[url] = struct{}{}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}

		var total, pruned int
		sys.Pool.Relays.Range(func(url string, relay *nostr.Relay) bool {
			total++

			if _, ok := keep[url]; ok {
				return true
			}

			if relay == nil || !relay.IsConnected() {
				if relay != nil {
					relay.Close()
				}
				sys.Pool.Relays.Delete(url)
				pruned++
				return true
			}

			hasActiveSubs := false
			relay.Subscriptions.Range(func(_ int64, sub *nostr.Subscription) bool {
				hasActiveSubs = true
				return false
			})

			if !hasActiveSubs {
				relay.Close()
				sys.Pool.Relays.Delete(url)
				pruned++
			}

			return true
		})

		if pruned > 0 {
			log.Info().Int("total", total).Int("pruned", pruned).Int("remaining", total-pruned).Msg("pruned idle relay connections")
			debug.FreeOSMemory()
		}
	}
}
