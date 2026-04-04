package sdk

import (
	"math/rand"
	"sync"
	"sync/atomic"

	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/eventstore"
	"fiatjaf.com/nostr/eventstore/nullstore"
	"fiatjaf.com/nostr/eventstore/wrappers"
	"fiatjaf.com/nostr/sdk/cache"
	cache_memory "fiatjaf.com/nostr/sdk/cache/memory"
	"fiatjaf.com/nostr/sdk/dataloader"
	"fiatjaf.com/nostr/sdk/hints"
	"fiatjaf.com/nostr/sdk/hints/memoryh"
	"fiatjaf.com/nostr/sdk/kvstore"
	kvstore_memory "fiatjaf.com/nostr/sdk/kvstore/memory"
	"github.com/btcsuite/btcd/btcec/v2"
)

// System represents the core functionality of the SDK, providing access to
// various caches, relays, and dataloaders for efficient Nostr operations.
//
// Usually an application should have a single global instance of this and use
// its internal Pool for all its operations.
//
// Store, KVStore and Hints are databases that should generally be persisted
// for any application that is intended to be executed more than once. By
// default they're set to in-memory stores, but ideally persisteable
// implementations should be given (some alternatives are provided in subpackages).
type System struct {
	KVStore                   kvstore.KVStore
	metadataCacheOnce         sync.Once
	MetadataCache             cache.Cache32[ProfileMetadata]
	relayListCacheOnce        sync.Once
	RelayListCache            cache.Cache32[GenericList[string, Relay]]
	followListCacheOnce       sync.Once
	FollowListCache           cache.Cache32[GenericList[nostr.PubKey, ProfileRef]]
	muteListCacheOnce         sync.Once
	MuteListCache             cache.Cache32[GenericList[nostr.PubKey, ProfileRef]]
	bookmarkListCacheOnce     sync.Once
	BookmarkListCache         cache.Cache32[GenericList[string, EventRef]]
	pinListCacheOnce          sync.Once
	PinListCache              cache.Cache32[GenericList[string, EventRef]]
	blockedRelayListCacheOnce sync.Once
	BlockedRelayListCache     cache.Cache32[GenericList[string, RelayURL]]
	searchRelayListCacheOnce  sync.Once
	SearchRelayListCache      cache.Cache32[GenericList[string, RelayURL]]
	topicListCacheOnce        sync.Once
	TopicListCache            cache.Cache32[GenericList[string, Topic]]
	relaySetsCacheOnce        sync.Once
	RelaySetsCache            cache.Cache32[GenericSets[string, RelayURL]]
	followSetsCacheOnce       sync.Once
	FollowSetsCache           cache.Cache32[GenericSets[nostr.PubKey, ProfileRef]]
	topicSetsCacheOnce        sync.Once
	TopicSetsCache            cache.Cache32[GenericSets[string, Topic]]
	zapProviderCacheOnce      sync.Once
	ZapProviderCache          cache.Cache32[nostr.PubKey]
	mintKeysCacheOnce         sync.Once
	MintKeysCache             cache.Cache32[map[uint64]*btcec.PublicKey]
	nutZapInfoCacheOnce       sync.Once
	NutZapInfoCache           cache.Cache32[NutZapInfo]
	Hints                     hints.HintsDB
	Pool                      *nostr.Pool
	RelayListRelays           *RelayStream
	FollowListRelays          *RelayStream
	MetadataRelays            *RelayStream
	FallbackRelays            *RelayStream
	JustIDRelays              *RelayStream
	UserSearchRelays          *RelayStream
	NoteSearchRelays          *RelayStream
	Store                     eventstore.Store

	Publisher nostr.Publisher

	replaceableLoaders []*dataloader.Loader[nostr.PubKey, nostr.Event]
	addressableLoaders []*dataloader.Loader[nostr.PubKey, []nostr.Event]
}

// SystemModifier is a function that modifies a System instance.
// It's used with NewSystem to configure the system during creation.
type SystemModifier func(sys *System)

// RelayStream provides a rotating list of relay URLs.
// It's used to distribute requests across multiple relays.
type RelayStream struct {
	URLs   []string
	serial atomic.Int32
}

// NewRelayStream creates a new RelayStream with the provided URLs.
func NewRelayStream(urls ...string) *RelayStream {
	rs := &RelayStream{URLs: urls}
	rs.serial.Add(rand.Int31n(int32(len(urls))))
	return rs
}

// Next returns the next URL in the rotation.
func (rs *RelayStream) Next() string {
	v := rs.serial.Add(1)
	return rs.URLs[int(v)%len(rs.URLs)]
}

// NewSystem creates a new System with default configuration,
// which can be customized using the provided modifiers.
//
// The list of provided With* modifiers isn't exhaustive and
// most internal fields of System can be modified after the System
// creation -- and in many cases one or another of these will have
// to be modified, so don't be afraid of doing that.
func NewSystem() *System {
	sys := &System{
		KVStore:          kvstore_memory.NewStore(),
		RelayListRelays:  NewRelayStream("wss://indexer.coracle.social", "wss://purplepag.es", "wss://relay.primal.net", "wss://relay.nos.social"),
		FollowListRelays: NewRelayStream("wss://purplepag.es", "wss://antiprimal.net", "wss://relay.damus.io", "wss://relay.nos.social"),
		MetadataRelays:   NewRelayStream("wss://purplepag.es", "wss://antiprimal.net", "wss://relay.damus.io", "wss://relay.nos.social"),
		FallbackRelays: NewRelayStream(
			"wss://offchain.pub",
			"wss://relay.damus.io",
			"wss://nostr.mom",
			"wss://nos.lol",
			"wss://relay.mostr.pub",
			"wss://nostr.land",
			"wss://relay.ditto.pub",
		),
		JustIDRelays: NewRelayStream(
			"wss://cache2.primal.net/v1",
			"wss://relay.nostr.band",
		),
		UserSearchRelays: NewRelayStream(
			"wss://search.nos.today",
			"wss://nostr.wine",
			"wss://relay.nostr.band",
		),
		NoteSearchRelays: NewRelayStream(
			"wss://nostr.wine",
			"wss://relay.nostr.band",
			"wss://search.nos.today",
		),
		Hints: memoryh.NewHintDB(),
	}

	sys.Pool = nostr.NewPool(nostr.PoolOptions{
		AuthorKindQueryMiddleware: sys.TrackQueryAttempts,
		EventMiddleware:           sys.TrackEventHintsAndRelays,
		DuplicateMiddleware:       sys.TrackEventRelaysD,
		PenaltyBox:                true,
	})

	sys.metadataCacheOnce.Do(func() {
		if sys.MetadataCache == nil {
			sys.MetadataCache = cache_memory.New[ProfileMetadata](8000)
		}
	})
	sys.relayListCacheOnce.Do(func() {
		if sys.RelayListCache == nil {
			sys.RelayListCache = cache_memory.New[GenericList[string, Relay]](8000)
		}
	})
	sys.zapProviderCacheOnce.Do(func() {
		if sys.ZapProviderCache == nil {
			sys.ZapProviderCache = cache_memory.New[nostr.PubKey](8000)
		}
	})
	sys.mintKeysCacheOnce.Do(func() {
		if sys.MintKeysCache == nil {
			sys.MintKeysCache = cache_memory.New[map[uint64]*btcec.PublicKey](8000)
		}
	})
	sys.nutZapInfoCacheOnce.Do(func() {
		if sys.NutZapInfoCache == nil {
			sys.NutZapInfoCache = cache_memory.New[NutZapInfo](8000)
		}
	})

	if sys.Store == nil {
		sys.Store = &nullstore.NullStore{}
		sys.Store.Init()
	}
	sys.Publisher = wrappers.DynamicPublisher{GetStore: func() eventstore.Store { return sys.Store }, MaxLimit: 1000}

	sys.initializeReplaceableDataloaders()
	sys.initializeAddressableDataloaders()

	return sys
}

// Close releases resources held by the System.
func (sys *System) Close() {
	if sys.KVStore != nil {
		sys.KVStore.Close()
	}
	if sys.Pool != nil {
		sys.Pool.Close("sdk.System closed")
	}
}
