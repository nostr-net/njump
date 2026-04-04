# eventstore

A collection of reusable database connectors, wrappers and schemas that store Nostr events and expose a simple Go interface:

```go
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
```

## Available Implementations

- **bleve**: Full-text search and indexing using the Bleve search library
- **boltdb**: Embedded key-value database using BoltDB
- **lmdb**: High-performance embedded database using LMDB
- **mmm**: Custom memory-mapped storage with advanced indexing
- **nullstore**: No-op store for testing and development
- **slicestore**: Simple in-memory slice-based store

## Command-line Tool

There is an [`eventstore` command-line tool](cmd/eventstore) that can be used to query these databases directly.
