package main

import (
	"strings"
	"testing"

	"fiatjaf.com/nostr"
)

func TestParseNostrWatchResponseFromStrings(t *testing.T) {
	input := `["wss://relay.damus.io","wss://relay.damus.io"," ws://relay.nos.social " ]`

	relays, err := parseNostrWatchResponse([]byte(input))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if got, want := len(relays), 3; got != want {
		t.Fatalf("expected %d entries, got %d", want, got)
	}
}

func TestParseNostrWatchResponseFromObjectWithArray(t *testing.T) {
	input := `{"relays":["wss://relay.nostr.bg","wss://relay.damus.io"]}`

	relays, err := parseNostrWatchResponse([]byte(input))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if got, want := len(relays), 2; got != want {
		t.Fatalf("expected %d entries, got %d", want, got)
	}
}

func TestParseNostrWatchResponseFromObjectEntries(t *testing.T) {
	input := `[{"url":"wss://relay.damus.io"},{"relay":"wss://relay.nos.social"},{"url":""}]`

	relays, err := parseNostrWatchResponse([]byte(input))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	normalized := normalizeRelayURLs(relays, 10)
	if got, want := len(normalized), 2; got != want {
		t.Fatalf("expected %d normalized entries, got %d", want, got)
	}
}

func TestMergeRelayPools(t *testing.T) {
	base := []string{"wss://relay.damus.io"}
	discovered := []string{"relay.damus.io", "wss://relay.nos.social", "wss://relay.nostr.bg"}

	merged := mergeRelayPools(base, discovered, 2)
	if got, want := len(merged), 2; got != want {
		t.Fatalf("expected %d merged entries, got %d", want, got)
	}
	if merged[0] != "wss://relay.damus.io" || merged[1] != "wss://relay.nos.social" {
		t.Fatalf("unexpected merge result: %v", merged)
	}
}

func TestRelayURLsFromStatusEvent(t *testing.T) {
	evt := nostr.Event{
		Tags: nostr.Tags{
			{"r", "wss://relay.damus.io"},
			{"r", "relay.nostr.band"},
			{"R", "wss://relay.damus.io", "read"},
			{"R", "relay.nostr.band", "!read"},
		},
	}

	got := relayURLsFromStatusEvent(evt)
	if len(got) != 1 {
		t.Fatalf("expected 1 relay after capability filtering, got %d: %v", len(got), got)
	}
	if !strings.Contains(got[0], "relay.damus.io") {
		t.Fatalf("unexpected relay result: %v", got[0])
	}
}

func TestNormalizeRelayURL(t *testing.T) {
	if got, want := normalizeRelayURL(" relay.nostr.band "), "wss://relay.nostr.band"; got != want {
		t.Fatalf("expected normalized relay %q, got %q", want, got)
	}
}
