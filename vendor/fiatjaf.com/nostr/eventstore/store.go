package eventstore

import (
	"iter"

	"fiatjaf.com/nostr"
)

// Store is a persistence layer for nostr events handled by a relay.
type Store interface {
	// Init is called at the very beginning by [Server.Start], after [Relay.Init],
	// allowing a storage to initialize its internal resources.
	Init() error

	// Close must be called after you're done using the store, to free up resources and so on.
	Close()

	// QueryEvents returns events that match the filter
	QueryEvents(filter nostr.Filter, maxLimit int) iter.Seq[nostr.Event]

	// DeleteEvent deletes an event atomically by ID
	DeleteEvent(nostr.ID) error

	// SaveEvent just saves an event, no side-effects.
	SaveEvent(nostr.Event) error

	// ReplaceEvent atomically replaces a replaceable or addressable event.
	// Conceptually it is like a Query->Delete->Save, but streamlined.
	ReplaceEvent(nostr.Event) error

	// CountEvents counts all events that match a given filter
	CountEvents(nostr.Filter) (uint32, error)
}
