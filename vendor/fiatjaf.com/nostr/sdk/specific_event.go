package sdk

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/nip19"
	"fiatjaf.com/nostr/sdk/hints"
)

// FetchSpecificEventParameters contains options for fetching specific events.
type FetchSpecificEventParameters struct {
	// WithRelays indicates whether to include relay information in the response
	// (this causes the request to take longer as it will wait for all relays to respond).
	WithRelays bool

	// SkipLocalStore indicates whether to skip checking the local store for the event
	SkipLocalStore bool

	// SaveToLocalStore indicates the result should be saved to local store
	SaveToLocalStore bool
}

// FetchSpecificEventFromInput tries to get a specific event from a NIP-19 code or event ID.
// It supports nevent, naddr, and note NIP-19 codes, as well as raw event IDs.
func (sys *System) FetchSpecificEventFromInput(
	ctx context.Context,
	input string,
	params FetchSpecificEventParameters,
) (event *nostr.Event, successRelays []string, err error) {
	var pointer nostr.Pointer

	prefix, data, err := nip19.Decode(input)
	if err == nil {
		switch prefix {
		case "nevent":
			pointer = data.(nostr.EventPointer)
		case "naddr":
			pointer = data.(nostr.EntityPointer)
		case "note":
			pointer = nostr.EventPointer{ID: data.(nostr.ID)}
		default:
			return nil, nil, fmt.Errorf("invalid code '%s'", input)
		}
	} else {
		if id, err := nostr.IDFromHex(input); err == nil {
			pointer = nostr.EventPointer{ID: id}
		} else {
			return nil, nil, fmt.Errorf("failed to decode '%s': %w", input, err)
		}
	}

	return sys.FetchSpecificEvent(ctx, pointer, params)
}

// FetchSpecificEvent tries to get a specific event using a Pointer (EventPointer or EntityPointer).
// It first checks the local store, then queries relays associated with the event or author,
// and finally falls back to general-purpose relays.
func (sys *System) FetchSpecificEvent(
	ctx context.Context,
	pointer nostr.Pointer,
	params FetchSpecificEventParameters,
) (event *nostr.Event, successRelays []string, err error) {
	// this is for deciding what relays will go on nevent and nprofile later
	priorityRelays := make([]string, 0, 8)

	var filter nostr.Filter
	var author nostr.PubKey
	relays := make([]string, 0, 10)
	fallback := make([]string, 0, 10)
	successRelays = make([]string, 0, 10)

	switch v := pointer.(type) {
	case nostr.EventPointer:
		author = v.Author
		filter.IDs = []nostr.ID{v.ID}
		relays = append(relays, v.Relays...)
		relays = nostr.AppendUnique(relays, sys.FallbackRelays.Next())
		fallback = append(fallback, sys.JustIDRelays.URLs...)
		fallback = nostr.AppendUnique(fallback, sys.FallbackRelays.Next())
		priorityRelays = append(priorityRelays, v.Relays...)
	case nostr.EntityPointer:
		author = v.PublicKey
		filter.Authors = []nostr.PubKey{v.PublicKey}
		filter.Tags = nostr.TagMap{"d": []string{v.Identifier}}
		filter.Kinds = []nostr.Kind{v.Kind}
		relays = append(relays, v.Relays...)
		relays = nostr.AppendUnique(relays, sys.FallbackRelays.Next())
		fallback = append(fallback, sys.FallbackRelays.Next(), sys.FallbackRelays.Next())
		priorityRelays = append(priorityRelays, v.Relays...)
	default:
		return nil, nil, fmt.Errorf("can't call sys.FetchSpecificEvent() with a %v", pointer)
	}

	// try to fetch in our internal eventstore first
	if !params.SkipLocalStore {
		for evt := range sys.Store.QueryEvents(filter, 1) {
			return &evt, nil, nil
		}
	}

	if author != nostr.ZeroPK {
		// fetch relays for author
		authorRelays := sys.FetchOutboxRelays(ctx, author, 3)

		// after that we register these hints as associated with author
		// (we do this after fetching author outbox relays because we are already going to prioritize these hints)
		now := nostr.Now()
		for _, relay := range priorityRelays {
			sys.Hints.Save(author, nostr.NormalizeURL(relay), hints.LastInHint, now)
		}

		// arrange these
		relays = nostr.AppendUnique(relays, authorRelays...)
		priorityRelays = nostr.AppendUnique(priorityRelays, authorRelays...)
	}

	var result *nostr.Event
	fetchProfileOnce := sync.Once{}

attempts:
	for _, attempt := range []struct {
		label          string
		relays         []string
		slowWithRelays bool
	}{
		{
			label:  "fetchspecific",
			relays: relays,
			// set this to true if the caller wants relays, so we won't return immediately
			//   but will instead wait a little while to see if more relays respond
			slowWithRelays: params.WithRelays,
		},
		{
			label:          "fetchspecific",
			relays:         fallback,
			slowWithRelays: false,
		},
	} {
		// actually fetch the event here
		countdown := 6.0
		subManyCtx := ctx

		for ie := range sys.Pool.FetchMany(subManyCtx, attempt.relays, filter, nostr.SubscriptionOptions{
			Label: attempt.label,
		}) {
			fetchProfileOnce.Do(func() {
				go sys.FetchProfileMetadata(ctx, ie.PubKey)
			})

			successRelays = append(successRelays, ie.Relay.URL)
			if result == nil || ie.CreatedAt > result.CreatedAt {
				result = &ie.Event
			}

			if !attempt.slowWithRelays {
				break attempts
			}

			countdown = min(countdown-0.5, 1)
		}
	}

	if result == nil {
		return nil, nil, nil
	}

	// save stuff in cache and in internal store
	if params.SaveToLocalStore {
		sys.Publisher.Publish(ctx, *result)
	}

	// put priority relays first so they get used in nevent and nprofile
	slices.SortFunc(successRelays, func(a, b string) int {
		vpa := slices.Contains(priorityRelays, a)
		vpb := slices.Contains(priorityRelays, b)
		if vpa == vpb {
			return 1
		}
		if vpa && !vpb {
			return 1
		}
		return -1
	})

	return result, successRelays, nil
}
