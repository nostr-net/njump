package khatru

import (
	"context"
	"errors"

	"fiatjaf.com/nostr"
)

func (rl *Relay) handleEphemeral(ctx context.Context, evt nostr.Event) error {
	if nil != rl.OnEvent {
		if reject, msg := rl.OnEvent(ctx, evt); reject {
			if msg == "" {
				return errors.New("blocked: no reason")
			} else {
				return errors.New(nostr.NormalizeOKMessage(msg, "blocked"))
			}
		}
	}

	if nil != rl.OnEphemeralEvent {
		rl.OnEphemeralEvent(ctx, evt)
	}

	return nil
}
