package nip60

import (
	"context"
	"fmt"
	"slices"

	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/nip60/client"
	"github.com/elnosh/gonuts/cashu"
	"github.com/elnosh/gonuts/cashu/nuts/nut10"
)

// ReceiveOptions contains options for receiving tokens
type ReceiveOptions struct {
	IntoMint                               []string
	IsNutzap                               bool
	AcceptTokensInSourceMintInTheWorseCase bool
}

func (w *Wallet) Receive(
	ctx context.Context,
	proofs cashu.Proofs,
	mint string,
	opts ReceiveOptions,
) error {
	if w.PublishUpdate == nil {
		return fmt.Errorf("can't do write operations: missing PublishUpdate function")
	}

	source, _ := nostr.NormalizeHTTPURL(mint)
	for i, url := range opts.IntoMint {
		var err error
		opts.IntoMint[i], err = nostr.NormalizeHTTPURL(url)
		if err != nil {
			return fmt.Errorf("invalid IntoMint URL '%s'", url)
		}
	}

	swapSettings := swapSettings{}

	for i, proof := range proofs {
		if proof.Secret != "" {
			nut10Secret, err := nut10.DeserializeSecret(proof.Secret)
			if err == nil {
				switch nut10Secret.Kind {
				case nut10.P2PK:
					swapSettings.mustSignOutputs = true

					proofs[i].Witness, err = signInput(w.PrivateKey, proof)
					if err != nil {
						return fmt.Errorf("failed to sign locked proof %d: %w", i, err)
					}
				case nut10.HTLC:
					return fmt.Errorf("HTLC token not supported yet")
				case nut10.AnyoneCanSpend:
					// ok
				}
			}
		}
	}

	sourceKeysets, err := client.GetAllKeysets(ctx, source)
	if err != nil {
		return fmt.Errorf("failed to get %s keysets: %w", source, err)
	}

	// get new proofs
	newProofs, _, err := w.swapProofs(ctx, source, proofs, proofs.Amount(), swapSettings)
	if err != nil {
		return err
	}

	newMint := source // if we don't have to do a lightning swap then new mint will be the same as old mint

	// if we have to swap to our own mint we do it now by getting a bolt11 invoice from our mint
	// and telling the current mint to pay it
	lightningSwap := !slices.Contains(opts.IntoMint, source)
	if lightningSwap {
		for _, targetMint := range opts.IntoMint {
			swappedProofs, err, status := lightningMeltMint(
				ctx,
				newProofs,
				source,
				sourceKeysets,
				targetMint,
			)
			if err != nil {
				switch status {
				case tryAnotherTargetMint:
					continue
				case manualActionRequired:
					return fmt.Errorf("failed to swap (needs manual action): %w", err)
				case nothingCanBeDone:
					return fmt.Errorf("failed to swap (nothing can be done, we probably lost the money): %w", err)
				case storeTokenFromSourceMint:
					if opts.AcceptTokensInSourceMintInTheWorseCase {
						goto saveproofs
					} else {
						return fmt.Errorf("unable to swap out of source mint: %w", err)
					}
				}
			} else {
				// everything went well
				newProofs = swappedProofs
				newMint = targetMint
				goto saveproofs
			}
		}

		// if we got here that means we ran out of our trusted mints to swap to
		if opts.AcceptTokensInSourceMintInTheWorseCase {
			goto saveproofs
		} else {
			return fmt.Errorf("unable to swap in to one of our mints: %w", err)
		}
	}

saveproofs:
	newToken := Token{
		Mint:     newMint,
		Proofs:   newProofs,
		mintedAt: nostr.Now(),
		event:    &nostr.Event{},
	}
	if err := newToken.toEvent(ctx, w.kr, newToken.event); err != nil {
		return fmt.Errorf("failed to make new token: %w", err)
	}

	he := HistoryEntry{
		event: &nostr.Event{},
		TokenReferences: []TokenRef{
			{
				EventID:  newToken.event.ID,
				Created:  true,
				IsNutzap: opts.IsNutzap,
			},
		},
		createdAt: nostr.Now(),
		In:        true,
		Amount:    newToken.Proofs.Amount(),
	}

	w.Lock()
	w.PublishUpdate(*newToken.event, nil, &newToken, nil, false)
	if err := he.toEvent(ctx, w.kr, he.event); err == nil {
		w.PublishUpdate(*he.event, nil, nil, nil, true)
	}
	w.Unlock()

	w.tokensMu.Lock()
	w.Tokens = append(w.Tokens, newToken)
	w.tokensMu.Unlock()

	return nil
}
