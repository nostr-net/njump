package main

import "testing"

func TestRelayBlacklistByUrlAndHost(t *testing.T) {
	configureRelayBlacklist("wss://blocked.example, blocked.example:8080")

	if !isRelayBlacklisted("wss://blocked.example") {
		t.Fatal("expected direct relay URL to be blacklisted")
	}
	if !isRelayBlacklisted("blocked.example:8080") {
		t.Fatal("expected host-only relay with port to be blacklisted")
	}
	if !isRelayBlacklisted("wss://blocked.example:8080/path") {
		t.Fatal("expected relay host with port and path to be blacklisted")
	}
	if isRelayBlacklisted("wss://allowed.example") {
		t.Fatal("expected non-blacklisted relay to be allowed")
	}
}

func TestFilterRelayList(t *testing.T) {
	configureRelayBlacklist("wss://blocked.example")
	filtered := filterRelayList([]string{
		"wss://allowed.example",
		"wss://blocked.example",
		" relay.example ",
	})

	if got, want := len(filtered), 2; got != want {
		t.Fatalf("expected %d relays after filter, got %d", want, got)
	}

	seen := map[string]struct{}{}
	for _, relay := range filtered {
		seen[relay] = struct{}{}
	}
	if _, ok := seen["wss://blocked.example"]; ok {
		t.Fatal("blacklisted relay should not be included")
	}
}
