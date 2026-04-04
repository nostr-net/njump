package khatru

import (
	"context"
	"errors"
	"fmt"

	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/nip77/negentropy"
	"fiatjaf.com/nostr/nip77/negentropy/storage/vector"
)

type NegentropySession struct {
	neg           *negentropy.Negentropy
	postponeClose func()
}

func (rl *Relay) startNegentropySession(ctx context.Context, filter nostr.Filter) (*vector.Vector, error) {
	if filter.LimitZero {
		return nil, fmt.Errorf("invalid limit 0")
	}

	ctx = SetNegentropy(ctx)

	if nil != rl.OnRequest {
		if reject, msg := rl.OnRequest(ctx, filter); reject {
			return nil, errors.New(nostr.NormalizeOKMessage(msg, "blocked"))
		}
	}

	// fetch events and add them to a negentropy Vector store
	vec := vector.New()
	if nil != rl.QueryStored {
		for event := range rl.QueryStored(ctx, filter) {
			vec.Insert(event.CreatedAt, event.ID)
		}
	}
	vec.Seal()

	return vec, nil
}

var negentropySessionKey = struct{}{}

func IsNegentropySession(ctx context.Context) bool {
	return ctx.Value(negentropySessionKey) != nil
}

func SetNegentropy(ctx context.Context) context.Context {
	return context.WithValue(ctx, negentropySessionKey, struct{}{})
}
