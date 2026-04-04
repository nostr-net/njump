package nip77

import (
	"context"
	"fmt"
	"sync"

	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/nip77/negentropy"
	"fiatjaf.com/nostr/nip77/negentropy/storage/vector"
)

type Direction struct {
	From  nostr.Querier
	To    nostr.Publisher
	Items chan nostr.ID
}

func NegentropySync(
	ctx context.Context,

	relayUrl string,
	filter nostr.Filter,

	// where our local events will be read from.
	// if it is nil the sync will be unidirectional: download-only.
	source nostr.Querier,

	// where new events received from the relay will be written to.
	// if it is nil the sync will be unidirectional: upload-only.
	// it can also be a nostr.QuerierPublisher in case source isn't provided
	// and you need a download-only sync that respects local data.
	target nostr.Publisher,

	// handle ids received on each direction, usually called with Sync() so the corresponding events are
	// fetched from the source and published to the target
	handle func(ctx context.Context, directions Direction),
) error {
	id := "nl-tmp" // for now we can't have more than one subscription in the same connection

	vec := vector.New()
	neg := negentropy.New(vec, 60_000, source != nil, target != nil)

	// connect to relay
	var err error
	errch := make(chan error)
	var relay *nostr.Relay
	relay, err = nostr.RelayConnect(ctx, relayUrl, nostr.RelayOptions{
		CustomHandler: func(data string) {
			envelope := ParseNegMessage(data)
			if envelope == nil {
				return
			}
			switch env := envelope.(type) {
			case *OpenEnvelope, *CloseEnvelope:
				errch <- fmt.Errorf("unexpected %s received from relay", env.Label())
				return
			case *ErrorEnvelope:
				errch <- fmt.Errorf("relay returned a %s: %s", env.Label(), env.Reason)
				return
			case *MessageEnvelope:
				nextmsg, err := neg.Reconcile(env.Message)
				if err != nil {
					errch <- fmt.Errorf("failed to reconcile: %w", err)
					return
				}

				if nextmsg != "" {
					msgb, _ := MessageEnvelope{id, nextmsg}.MarshalJSON()
					relay.Write(msgb)
				}
			}
		},
	})
	if err != nil {
		return err
	}

	// fill our local vector
	var usedSource nostr.Querier
	if source != nil {
		for evt := range source.QueryEvents(filter) {
			vec.Insert(evt.CreatedAt, evt.ID)
		}
		usedSource = source
	}
	if target != nil {
		if targetSource, ok := target.(nostr.Querier); ok && targetSource != usedSource {
			for evt := range targetSource.QueryEvents(filter) {
				vec.Insert(evt.CreatedAt, evt.ID)
			}
		}
	}
	vec.Seal()

	// kickstart the process
	msg := neg.Start()
	open, _ := OpenEnvelope{id, filter, msg}.MarshalJSON()
	err = relay.WriteWithError(open)
	if err != nil {
		return fmt.Errorf("failed to write to relay: %w", err)
	}

	defer func() {
		clse, _ := CloseEnvelope{id}.MarshalJSON()
		relay.Write(clse)
	}()

	wg := sync.WaitGroup{}

	// handle emitted events from either direction
	if source != nil {
		wg.Go(func() {
			handle(ctx, Direction{
				From:  source,
				To:    relay,
				Items: neg.Haves,
			})
		})
	}
	if target != nil {
		wg.Go(func() {
			handle(ctx, Direction{
				From:  relay,
				To:    target,
				Items: neg.HaveNots,
			})
		})
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		wg.Wait()
		select {
		case errch <- nil:
		case <-ctx.Done():
		}
	}()

	select {
	case err = <-errch:
		if err != nil {
			return err
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func SyncEventsFromIDs(ctx context.Context, dir Direction) {
	// this is only necessary because relays are too ratelimiting
	batch := make([]nostr.ID, 0, 50)

	seen := make(map[nostr.ID]struct{})
	for item := range dir.Items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}

		batch = append(batch, item)
		if len(batch) == 50 {
			for evt := range dir.From.QueryEvents(nostr.Filter{IDs: batch}) {
				dir.To.Publish(ctx, evt)
			}
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		for evt := range dir.From.QueryEvents(nostr.Filter{IDs: batch}) {
			dir.To.Publish(ctx, evt)
		}
	}
}
