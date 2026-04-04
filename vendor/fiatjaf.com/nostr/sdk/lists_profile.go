package sdk

import (
	"context"
	"net/url"
	"strings"

	"fiatjaf.com/nostr"
	cache_memory "fiatjaf.com/nostr/sdk/cache/memory"
)

type ProfileRef struct {
	Pubkey  nostr.PubKey
	Relay   string
	Petname string
}

func (f ProfileRef) Value() nostr.PubKey { return f.Pubkey }

func (sys *System) FetchFollowList(ctx context.Context, pubkey nostr.PubKey) GenericList[nostr.PubKey, ProfileRef] {
	sys.followListCacheOnce.Do(func() {
		if sys.FollowListCache == nil {
			sys.FollowListCache = cache_memory.New[GenericList[nostr.PubKey, ProfileRef]](1000)
		}
	})

	fl, _ := fetchGenericList(sys, ctx, pubkey, 3, kind_3, parseProfileRef, sys.FollowListCache)
	return fl
}

func (sys *System) FetchMuteList(ctx context.Context, pubkey nostr.PubKey) GenericList[nostr.PubKey, ProfileRef] {
	sys.muteListCacheOnce.Do(func() {
		if sys.MuteListCache == nil {
			sys.MuteListCache = cache_memory.New[GenericList[nostr.PubKey, ProfileRef]](1000)
		}
	})

	ml, _ := fetchGenericList(sys, ctx, pubkey, 10000, kind_10000, parseProfileRef, sys.MuteListCache)
	return ml
}

func (sys *System) FetchFollowSets(ctx context.Context, pubkey nostr.PubKey) GenericSets[nostr.PubKey, ProfileRef] {
	sys.followSetsCacheOnce.Do(func() {
		if sys.FollowSetsCache == nil {
			sys.FollowSetsCache = cache_memory.New[GenericSets[nostr.PubKey, ProfileRef]](1000)
		}
	})

	ml, _ := fetchGenericSets(sys, ctx, pubkey, 30000, kind_30000, parseProfileRef, sys.FollowSetsCache)
	return ml
}

func parseProfileRef(tag nostr.Tag) (fw ProfileRef, ok bool) {
	if len(tag) < 2 {
		return fw, false
	}
	if tag[0] != "p" {
		return fw, false
	}

	pubkey, err := nostr.PubKeyFromHex(tag[1])
	if err != nil {
		return fw, false
	}
	fw.Pubkey = pubkey

	if len(tag) > 2 {
		if _, err := url.Parse(tag[2]); err == nil {
			fw.Relay = nostr.NormalizeURL(tag[2])
		}

		if len(tag) > 3 {
			fw.Petname = strings.TrimSpace(tag[3])
		}
	}

	return fw, true
}
