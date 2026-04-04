package sdk

import (
	"context"

	"fiatjaf.com/nostr"
	cache_memory "fiatjaf.com/nostr/sdk/cache/memory"
)

type Relay struct {
	URL    string
	Inbox  bool
	Outbox bool
}

func (r Relay) Value() string { return r.URL }

type RelayURL string

func (r RelayURL) Value() string { return string(r) }

func (sys *System) FetchRelayList(ctx context.Context, pubkey nostr.PubKey) GenericList[string, Relay] {
	ml, _ := fetchGenericList(sys, ctx, pubkey, 10002, kind_10002, parseRelayFromKind10002, sys.RelayListCache)
	return ml
}

func (sys *System) FetchBlockedRelayList(ctx context.Context, pubkey nostr.PubKey) GenericList[string, RelayURL] {
	sys.blockedRelayListCacheOnce.Do(func() {
		if sys.BlockedRelayListCache == nil {
			sys.BlockedRelayListCache = cache_memory.New[GenericList[string, RelayURL]](1000)
		}
	})

	ml, _ := fetchGenericList(sys, ctx, pubkey, 10006, kind_10006, parseRelayURL, sys.BlockedRelayListCache)
	return ml
}

func (sys *System) FetchSearchRelayList(ctx context.Context, pubkey nostr.PubKey) GenericList[string, RelayURL] {
	sys.searchRelayListCacheOnce.Do(func() {
		if sys.SearchRelayListCache == nil {
			sys.SearchRelayListCache = cache_memory.New[GenericList[string, RelayURL]](1000)
		}
	})

	ml, _ := fetchGenericList(sys, ctx, pubkey, 10007, kind_10007, parseRelayURL, sys.SearchRelayListCache)
	return ml
}

func (sys *System) FetchRelaySets(ctx context.Context, pubkey nostr.PubKey) GenericSets[string, RelayURL] {
	sys.relaySetsCacheOnce.Do(func() {
		if sys.RelaySetsCache == nil {
			sys.RelaySetsCache = cache_memory.New[GenericSets[string, RelayURL]](1000)
		}
	})

	ml, _ := fetchGenericSets(sys, ctx, pubkey, 30002, kind_30002, parseRelayURL, sys.RelaySetsCache)
	return ml
}

func parseRelayFromKind10002(tag nostr.Tag) (rl Relay, ok bool) {
	if len(tag) < 2 {
		return rl, false
	}

	if u := tag[1]; u != "" && tag[0] == "r" {
		if !nostr.IsValidRelayURL(u) {
			return rl, false
		}
		u := nostr.NormalizeURL(u)

		relay := Relay{
			URL: u,
		}

		if len(tag) == 2 {
			relay.Inbox = true
			relay.Outbox = true
		} else if tag[2] == "write" {
			relay.Outbox = true
		} else if tag[2] == "read" {
			relay.Inbox = true
		}

		return relay, true
	}

	return rl, false
}

func parseRelayURL(tag nostr.Tag) (rl RelayURL, ok bool) {
	if len(tag) < 2 {
		return rl, false
	}

	if u := tag[1]; u != "" && tag[0] == "relay" {
		if !nostr.IsValidRelayURL(u) {
			return rl, false
		}
		u := nostr.NormalizeURL(u)
		return RelayURL(u), true
	}

	return rl, false
}
