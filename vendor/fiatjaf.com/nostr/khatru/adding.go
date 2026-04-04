package khatru

import (
	"context"
	"errors"
	"fmt"

	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/eventstore"
	"fiatjaf.com/nostr/nip40"
)

// AddEvent sends an event through then normal add pipeline, as if it was received from a websocket.
func (rl *Relay) AddEvent(ctx context.Context, evt nostr.Event) (skipBroadcast bool, writeError error) {
	if evt.Kind.IsEphemeral() {
		return false, rl.handleEphemeral(ctx, evt)
	} else {
		return rl.handleNormal(ctx, evt)
	}
}

func (rl *Relay) handleNormal(ctx context.Context, evt nostr.Event) (skipBroadcast bool, writeError error) {
	if nil != rl.OnEvent {
		if reject, msg := rl.OnEvent(ctx, evt); reject {
			if msg == "" {
				return true, errors.New("blocked: no reason")
			} else {
				return true, errors.New(nostr.NormalizeOKMessage(msg, "blocked"))
			}
		}
	}

	// will store
	// regular kinds are just saved directly
	if evt.Kind.IsRegular() {
		if nil != rl.StoreEvent {
			if err := rl.StoreEvent(ctx, evt); err != nil {
				switch err {
				case eventstore.ErrDupEvent:
					return true, nil
				default:
					return false, fmt.Errorf("%s", nostr.NormalizeOKMessage(err.Error(), "error"))
				}
			}
		}
	} else {
		// otherwise it's a replaceable
		if nil != rl.ReplaceEvent {
			if err := rl.ReplaceEvent(ctx, evt); err != nil {
				switch err {
				case eventstore.ErrDupEvent:
					return true, nil
				default:
					return false, fmt.Errorf("%s", nostr.NormalizeOKMessage(err.Error(), "error"))
				}
			}
		}
	}

	if nil != rl.OnEventSaved {
		rl.OnEventSaved(ctx, evt)
	}

	// track event expiration if applicable
	if rl.expirationManager != nil {
		rl.expirationManager.trackEvent(evt.ID, nip40.GetExpiration(evt.Tags))
	}

	return false, nil
}
