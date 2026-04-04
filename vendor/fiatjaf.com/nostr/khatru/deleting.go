package khatru

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"fiatjaf.com/nostr"
	"golang.org/x/sync/errgroup"
)

var (
	ErrNothingToDelete = errors.New("blocked: nothing to delete")
	ErrNotAuthor       = errors.New("blocked: you are not the author of this event")
)

// event deletion -- nip09
func (rl *Relay) handleDeleteRequest(ctx context.Context, evt nostr.Event) error {
	if nil == rl.QueryStored || nil == rl.DeleteEvent {
		// if we don't have a way to query or to delete that means we won't delete anything
		return ErrNothingToDelete
	}

	haveDeletedSomething := false
	for _, tag := range evt.Tags {
		if len(tag) >= 2 {
			var f nostr.Filter

			switch tag[0] {
			case "e":
				id, err := nostr.IDFromHex(tag[1])
				if err != nil {
					return fmt.Errorf("invalid 'e' tag '%s': %w", tag[1], err)
				}
				f = nostr.Filter{IDs: []nostr.ID{id}}
			case "a":
				spl := strings.SplitN(tag[1], ":", 3)
				if len(spl) != 3 {
					continue
				}
				kind, err := strconv.Atoi(spl[0])
				if err != nil {
					continue
				}
				author, err := nostr.PubKeyFromHex(spl[1])
				if err != nil {
					continue
				}

				identifier := spl[2]
				f = nostr.Filter{
					Kinds:   []nostr.Kind{nostr.Kind(kind)},
					Authors: []nostr.PubKey{author},
					Tags:    nostr.TagMap{"d": []string{identifier}},
					Until:   evt.CreatedAt,
				}
			default:
				continue
			}

			ctx := context.WithValue(ctx, internalCallKey, struct{}{})

			errg, ctx := errgroup.WithContext(ctx)
			for target := range rl.QueryStored(ctx, f) {
				// got the event, now check if the user can delete it
				if target.PubKey == evt.PubKey {
					// delete it
					errg.Go(func() error {
						if err := rl.DeleteEvent(ctx, target.ID); err != nil {
							return err
						}

						// if it was tracked to be expired that is not needed anymore
						if rl.expirationManager != nil {
							rl.expirationManager.removeEvent(target.ID)
						}

						haveDeletedSomething = true
						if rl.OnEventDeleted != nil {
							rl.OnEventDeleted(ctx, target)
						}
						return nil
					})
				} else {
					// fail and stop here
					return ErrNotAuthor
				}

				// don't try to query this same event again
				break
			}

			if err := errg.Wait(); err != nil {
				return err
			}
		}
	}

	if haveDeletedSomething {
		return nil
	}

	return ErrNothingToDelete
}
