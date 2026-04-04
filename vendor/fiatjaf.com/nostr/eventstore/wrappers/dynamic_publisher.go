package wrappers

import (
	"context"
	"fmt"
	"iter"

	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/eventstore"
)

var _ nostr.Publisher = DynamicPublisher{}

type DynamicPublisher struct {
	GetStore func() eventstore.Store
	MaxLimit int
}

func (w DynamicPublisher) QueryEvents(filter nostr.Filter) iter.Seq[nostr.Event] {
	return w.GetStore().QueryEvents(filter, w.MaxLimit)
}

func (w DynamicPublisher) Publish(ctx context.Context, evt nostr.Event) error {
	if evt.Kind.IsEphemeral() {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if evt.Kind.IsRegular() {
		if err := w.GetStore().SaveEvent(evt); err != nil && err != eventstore.ErrDupEvent {
			return fmt.Errorf("failed to save: %w", err)
		} else {
			return err
		}
	}

	return w.GetStore().ReplaceEvent(evt)
}
