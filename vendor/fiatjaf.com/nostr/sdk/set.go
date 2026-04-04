package sdk

import (
	"context"
	"slices"
	"time"

	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/sdk/cache"
)

// this is similar to list.go and inherits code from that.

type GenericSets[V comparable, I TagItemWithValue[V]] struct {
	PubKey nostr.PubKey  `json:"-"`
	Events []nostr.Event `json:"-"`

	Sets map[string][]I
}

func fetchGenericSets[V comparable, I TagItemWithValue[V]](
	sys *System,
	ctx context.Context,
	pubkey nostr.PubKey,
	actualKind nostr.Kind,
	addressableIndex addressableIndex,
	parseTag func(nostr.Tag) (I, bool),
	cache cache.Cache32[GenericSets[V, I]],
) (fl GenericSets[V, I], fromInternal bool) {
	n := pubkey[7]
	lockIdx := (nostr.Kind(n) + actualKind) % 60
	genericListMutexes[lockIdx].Lock()

	if valueWasJustCached[lockIdx].CompareAndSwap(true, false) {
		// this ensures the cache has had time to commit the values
		// so we don't repeat a fetch immediately after the other
		time.Sleep(time.Millisecond * 10)
	}

	genericListMutexes[lockIdx].Unlock()

	if v, ok := cache.Get(pubkey); ok {
		return v, true
	}

	v := GenericSets[V, I]{PubKey: pubkey}

	events := slices.Collect(
		sys.Store.QueryEvents(nostr.Filter{Kinds: []nostr.Kind{actualKind}, Authors: []nostr.PubKey{pubkey}}, 100),
	)
	if len(events) != 0 {
		// ok, we found something locally
		sets := parseSetsFromEvents(events, parseTag)
		v.Events = events
		v.Sets = sets

		// but if we haven't tried fetching from the network recently we should do it
		lastFetchKey := makeLastFetchKey(actualKind, pubkey)
		lastFetchData, _ := sys.KVStore.Get(lastFetchKey)
		if lastFetchData == nil || nostr.Now()-decodeTimestamp(lastFetchData) > getLocalStoreRefreshDaysForKind(actualKind)*24*60*60 {
			newV := tryFetchSetsFromNetwork(ctx, sys, pubkey, addressableIndex, parseTag)

			// unlike for lists, when fetching sets we will blindly trust whatever we get from the network
			v = *newV
			for _, evt := range newV.Events {
				sys.Store.ReplaceEvent(evt)
			}

			// even if we didn't find anything register this because we tried
			// (and we still have the previous event in our local store)
			sys.KVStore.Set(lastFetchKey, encodeTimestamp(nostr.Now()))
		}

		// and finally save this to cache
		cache.SetWithTTL(pubkey, v, time.Hour*6)
		valueWasJustCached[lockIdx].Store(true)

		return v, true
	}

	if newV := tryFetchSetsFromNetwork(ctx, sys, pubkey, addressableIndex, parseTag); newV != nil {
		v = *newV

		for _, evt := range newV.Events {
			sys.Store.ReplaceEvent(evt)
		}

		// we'll only save this if we got something which means we found at least one event
		lastFetchKey := makeLastFetchKey(actualKind, pubkey)
		sys.KVStore.Set(lastFetchKey, encodeTimestamp(nostr.Now()))
	}

	// save cache even if we didn't get anything
	cache.SetWithTTL(pubkey, v, time.Hour*6)
	valueWasJustCached[lockIdx].Store(true)

	return v, false
}

func tryFetchSetsFromNetwork[V comparable, I TagItemWithValue[V]](
	ctx context.Context,
	sys *System,
	pubkey nostr.PubKey,
	addressableIndex addressableIndex,
	parseTag func(nostr.Tag) (I, bool),
) *GenericSets[V, I] {
	events, err := sys.addressableLoaders[addressableIndex].Load(ctx, pubkey)
	if err != nil {
		return nil
	}

	v := &GenericSets[V, I]{
		PubKey: pubkey,
		Events: events,
		Sets:   parseSetsFromEvents(events, parseTag),
	}
	for _, evt := range events {
		sys.Publisher.Publish(ctx, evt)
	}
	return v
}

func parseSetsFromEvents[V comparable, I TagItemWithValue[V]](
	events []nostr.Event,
	parseTag func(nostr.Tag) (I, bool),
) map[string][]I {
	sets := make(map[string][]I, len(events))
	for _, evt := range events {
		items := make([]I, 0, len(evt.Tags))
		for _, tag := range evt.Tags {
			item, ok := parseTag(tag)
			if ok {
				// check if this already exists before adding
				if slices.IndexFunc(items, func(i I) bool { return i.Value() == item.Value() }) == -1 {
					items = append(items, item)
				}
			}
		}
		sets[evt.Tags.GetD()] = items
	}
	return sets
}
