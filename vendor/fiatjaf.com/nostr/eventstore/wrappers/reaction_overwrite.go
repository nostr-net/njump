package wrappers

import (
	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/eventstore"
)

// UniqueReactionsWrapper ensures that only the latest reaction or poll response from an author to an event is kept.
// When a new reaction (kind 7) or poll response (kind 1018) is received, it deletes any previous ones from the same author to the same event.
type UniqueReactionsWrapper struct {
	eventstore.Store
}

func (w *UniqueReactionsWrapper) SaveEvent(event nostr.Event) error {
	// check if it's a reaction or poll response
	if event.Kind == 7 || event.Kind == 1018 {
		// find the referenced event ID
		ref := event.Tags.Find("e")
		if ref != nil {
			// query for existing events of same kind from same author referencing the same event and delete them
			for old := range w.Store.QueryEvents(nostr.Filter{
				Authors: []nostr.PubKey{event.PubKey},
				Kinds:   []nostr.Kind{event.Kind},
				Tags:    nostr.TagMap{"e": []string{ref[1]}},
			}, 1_000) {
				w.Store.DeleteEvent(old.ID)
			}
		}
	}

	// Save the new event
	return w.Store.SaveEvent(event)
}
