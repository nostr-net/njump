package sdk

import (
	"context"
	"crypto/sha256"
	"io"
	"net/http"
	"strings"
	"time"

	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/nip60"
	"fiatjaf.com/nostr/nip60/client"
	"fiatjaf.com/nostr/nip61"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/tidwall/gjson"
)

// NutZapInfo represents user nut zap information from kind 10019 events.
// contains both the raw event and parsed info fields.
type NutZapInfo struct {
	PubKey nostr.PubKey `json:"-"` // must always be set otherwise things will break
	Event  *nostr.Event `json:"-"` // may be empty if a nut zap info event wasn't found

	nip61.Info
}

// FetchZapProvider fetches the zap provider public key for a given user from their profile metadata.
// It uses a cache to avoid repeated fetches. If no zap provider is set in the profile, returns an empty PubKey.
func (sys *System) FetchZapProvider(ctx context.Context, pk nostr.PubKey) nostr.PubKey {
	if v, ok := sys.ZapProviderCache.Get(pk); ok {
		return v
	}

	pm := sys.FetchProfileMetadata(ctx, pk)

	if pm.LUD16 != "" {
		parts := strings.Split(pm.LUD16, "@")
		if len(parts) == 2 {
			url := "http://" + parts[1] + "/.well-known/lnurlp/" + parts[0]
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err == nil {
				client := &http.Client{Timeout: 10 * time.Second}
				resp, err := client.Do(req)
				if err == nil {
					defer resp.Body.Close()
					if body, err := io.ReadAll(resp.Body); err == nil {
						gj := gjson.ParseBytes(body)
						if gj.Get("allowsNostr").Type == gjson.True {
							if pk, err := nostr.PubKeyFromHex(gj.Get("nostrPubkey").Str); err == nil {
								sys.ZapProviderCache.SetWithTTL(pk, pk, time.Hour*6)
								return pk
							}
						}
					}
				}
			}
		}
	}

	sys.ZapProviderCache.SetWithTTL(pk, nostr.ZeroPK, time.Hour*2)
	return nostr.ZeroPK
}

// FetchNutZapInfo fetches nut zap info for a given user from the local cache, or from the local store,
// or, failing these, from the target user's defined outbox relays -- then caches the result.
// always returns a NutZapInfo, even if no info was found (in which case only the PubKey field is set).
func (sys *System) FetchNutZapInfo(ctx context.Context, pubkey nostr.PubKey) NutZapInfo {
	if v, ok := sys.NutZapInfoCache.Get(pubkey); ok {
		return v
	}

	for evt := range sys.Store.QueryEvents(nostr.Filter{Kinds: []nostr.Kind{10019}, Authors: []nostr.PubKey{pubkey}}, 1) {
		// ok, we found something locally
		nzi, err := ParseNutZapInfo(evt)
		if err != nil {
			break
		}

		// but if we haven't tried fetching from the network recently we should do it
		lastFetchKey := makeLastFetchKey(10019, pubkey)
		lastFetchData, _ := sys.KVStore.Get(lastFetchKey)
		if lastFetchData == nil || nostr.Now()-decodeTimestamp(lastFetchData) > 7*24*60*60 {
			newNZI := sys.tryFetchNutZapInfoFromNetwork(ctx, pubkey)
			if newNZI != nil && newNZI.Event.CreatedAt > nzi.Event.CreatedAt {
				nzi = *newNZI
			}

			// even if we didn't find anything register this because we tried
			// (and we still have the previous event in our local store)
			sys.KVStore.Set(lastFetchKey, encodeTimestamp(nostr.Now()))
		}

		// and finally save this to cache
		sys.NutZapInfoCache.SetWithTTL(pubkey, nzi, time.Hour*6)

		return nzi
	}

	var nzi NutZapInfo
	nzi.PubKey = pubkey
	if newNZI := sys.tryFetchNutZapInfoFromNetwork(ctx, pubkey); newNZI != nil {
		nzi = *newNZI

		// we'll only save this if we got something which means we found at least one event
		lastFetchKey := makeLastFetchKey(10019, pubkey)
		sys.KVStore.Set(lastFetchKey, encodeTimestamp(nostr.Now()))
	}

	// save cache even if we didn't get anything
	sys.NutZapInfoCache.SetWithTTL(pubkey, nzi, time.Hour*6)

	return nzi
}

func (sys *System) tryFetchNutZapInfoFromNetwork(ctx context.Context, pubkey nostr.PubKey) *NutZapInfo {
	evt, err := sys.replaceableLoaders[kind_10019].Load(ctx, pubkey)
	if err != nil {
		return nil
	}

	nzi, err := ParseNutZapInfo(evt)
	if err != nil {
		return nil
	}

	sys.Publisher.Publish(ctx, evt)
	sys.NutZapInfoCache.SetWithTTL(pubkey, nzi, time.Hour*6)
	return &nzi
}

// ParseNutZapInfo parses a kind 10019 event into a NutZapInfo struct.
// returns an error if the event is not kind 10019 or if parsing fails.
func ParseNutZapInfo(event nostr.Event) (nzi NutZapInfo, err error) {
	err = nzi.Info.ParseEvent(event)

	nzi.PubKey = event.PubKey
	nzi.Event = &event

	return nzi, err
}

// FetchMintKeys fetches the active keyset from the given mint URL and parses the keys.
// It uses a cache to avoid repeated fetches.
func (sys *System) FetchMintKeys(ctx context.Context, mintURL string) (map[uint64]*btcec.PublicKey, error) {
	hash := sha256.Sum256([]byte(mintURL))
	if v, ok := sys.MintKeysCache.Get(hash); ok {
		return v, nil
	}

	keyset, err := client.GetActiveKeyset(ctx, mintURL)
	if err != nil {
		return nil, err
	}

	ksKeys, err := nip60.ParseKeysetKeys(keyset.Keys)
	if err != nil {
		return nil, err
	}

	sys.MintKeysCache.SetWithTTL(hash, ksKeys, time.Hour*6)
	return ksKeys, nil
}
