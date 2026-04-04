package sdk

import (
	"context"

	"fiatjaf.com/nostr"
	cache_memory "fiatjaf.com/nostr/sdk/cache/memory"
)

type EventRef struct{ nostr.Pointer }

func (e EventRef) Value() string { return e.Pointer.AsTagReference() }

func (sys *System) FetchBookmarkList(ctx context.Context, pubkey nostr.PubKey) GenericList[string, EventRef] {
	sys.bookmarkListCacheOnce.Do(func() {
		if sys.BookmarkListCache == nil {
			sys.BookmarkListCache = cache_memory.New[GenericList[string, EventRef]](1000)
		}
	})

	ml, _ := fetchGenericList(sys, ctx, pubkey, 10003, kind_10003, parseEventRef, sys.BookmarkListCache)
	return ml
}

func (sys *System) FetchPinList(ctx context.Context, pubkey nostr.PubKey) GenericList[string, EventRef] {
	sys.pinListCacheOnce.Do(func() {
		if sys.PinListCache == nil {
			sys.PinListCache = cache_memory.New[GenericList[string, EventRef]](1000)
		}
	})

	ml, _ := fetchGenericList(sys, ctx, pubkey, 10001, kind_10001, parseEventRef, sys.PinListCache)
	return ml
}

func parseEventRef(tag nostr.Tag) (evr EventRef, ok bool) {
	if len(tag) < 2 {
		return evr, false
	}
	switch tag[0] {
	case "e":
		pointer, err := nostr.EventPointerFromTag(tag)
		if err != nil {
			return evr, false
		}
		evr.Pointer = pointer
	case "a":
		pointer, err := nostr.EntityPointerFromTag(tag)
		if err != nil {
			return evr, false
		}
		evr.Pointer = pointer
	default:
		return evr, false
	}

	return evr, false
}
