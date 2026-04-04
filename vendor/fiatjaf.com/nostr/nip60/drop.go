package nip60

import (
	"context"

	"fiatjaf.com/nostr"
)

// DropToken silently abandons a token
func (w *Wallet) DropToken(
	ctx context.Context,
	tokenID string,
) {
	updatedTokens := make([]Token, 0, len(w.Tokens))

	for _, token := range w.Tokens {
		if token.ID() == tokenID {
			deleteEvent := nostr.Event{
				CreatedAt: nostr.Now(),
				Kind:      5,
				Tags:      nostr.Tags{{"e", token.event.ID.Hex()}, {"k", "7375"}},
			}
			w.kr.SignEvent(ctx, &deleteEvent)

			w.Lock()
			w.PublishUpdate(deleteEvent, &token, nil, nil, false)
			w.Unlock()
		} else {
			updatedTokens = append(updatedTokens, token)
		}
	}

	w.Tokens = updatedTokens
}
