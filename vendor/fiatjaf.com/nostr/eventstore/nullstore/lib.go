package nullstore

import (
	"iter"

	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/eventstore"
)

var _ eventstore.Store = NullStore{}

type NullStore struct{}

func (b NullStore) Init() error {
	return nil
}

func (b NullStore) Close() {}

func (b NullStore) DeleteEvent(id nostr.ID) error {
	return nil
}

func (b NullStore) QueryEvents(filter nostr.Filter, maxLimit int) iter.Seq[nostr.Event] {
	return func(yield func(nostr.Event) bool) {}
}

func (b NullStore) SaveEvent(evt nostr.Event) error {
	return nil
}

func (b NullStore) ReplaceEvent(evt nostr.Event) error {
	return nil
}

func (b NullStore) CountEvents(filter nostr.Filter) (uint32, error) {
	return 0, nil
}
