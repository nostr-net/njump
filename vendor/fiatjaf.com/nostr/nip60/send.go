package nip60

import (
	"context"
	"fmt"
	"slices"

	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/nip60/client"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/elnosh/gonuts/cashu"
	"github.com/elnosh/gonuts/cashu/nuts/nut02"
	"github.com/elnosh/gonuts/cashu/nuts/nut10"
	"github.com/elnosh/gonuts/cashu/nuts/nut11"
)

// SendOptions contains options for sending tokens
type SendOptions struct {
	SpecificSourceMint string
	P2PK               *btcec.PublicKey
	RefundTimelock     nostr.Timestamp
	Hashlock           [32]byte
}

func (opts SendOptions) asSpendingCondition(refund *btcec.PublicKey) *nut10.SpendingCondition {
	if opts.Hashlock != [32]byte{} {
		// when we have an HTLC condition:
		// (it can also include a P2PK and a timelock)
		tags := nut11.P2PKTags{
			NSigs:    1,
			Locktime: 0,
			Sigflag:  nut11.SIGINPUTS,
		}
		if opts.P2PK != nil {
			tags.Pubkeys = []*btcec.PublicKey{opts.P2PK}
		}
		if opts.RefundTimelock != 0 {
			tags.Refund = []*btcec.PublicKey{refund}
			tags.Locktime = int64(opts.RefundTimelock)
		}

		return &nut10.SpendingCondition{
			Kind: nut10.HTLC,
			Data: nostr.HexEncodeToString(opts.Hashlock[:]),
			Tags: nut11.SerializeP2PKTags(tags),
		}
	} else if opts.P2PK != nil {
		// otherwise when it is just a P2PK condition with no hashlock
		// (may also have a timelock)

		tags := nut11.P2PKTags{
			NSigs:    1,
			Locktime: 0,
			Pubkeys:  []*btcec.PublicKey{opts.P2PK},
			Sigflag:  nut11.SIGINPUTS,
		}
		if opts.RefundTimelock != 0 {
			tags.Refund = []*btcec.PublicKey{refund}
			tags.Locktime = int64(opts.RefundTimelock)
		}

		return &nut10.SpendingCondition{
			Kind: nut10.P2PK,
			Data: nostr.HexEncodeToString(opts.P2PK.SerializeCompressed()),
			Tags: nut11.SerializeP2PKTags(tags),
		}
	} else {
		return nil
	}
}

type chosenTokens struct {
	mint         string
	tokens       []Token
	tokenIndexes []int
	proofs       cashu.Proofs
	keysets      []nut02.Keyset
}

func (w *Wallet) saveChangeAndDeleteUsedTokens(
	ctx context.Context,
	mintURL string,
	changeProofs cashu.Proofs,
	usedTokenIndexes []int,
	he *HistoryEntry,
) error {
	// delete spent tokens and save our change
	updatedTokens := make([]Token, 0, len(w.Tokens))

	changeToken := Token{
		mintedAt: nostr.Now(),
		Mint:     mintURL,
		Proofs:   changeProofs,
		Deleted:  make([]nostr.ID, 0, len(usedTokenIndexes)),
		event:    &nostr.Event{},
	}

	for i, token := range w.Tokens {
		if slices.Contains(usedTokenIndexes, i) {
			if token.event != nil {
				token.Deleted = append(token.Deleted, token.event.ID)

				deleteEvent := nostr.Event{
					CreatedAt: nostr.Now(),
					Kind:      5,
					Tags:      nostr.Tags{{"e", token.event.ID.Hex()}, {"k", "7375"}},
				}
				w.kr.SignEvent(ctx, &deleteEvent)

				w.Lock()
				w.PublishUpdate(deleteEvent, &token, nil, nil, false)
				w.Unlock()

				// fill in the history deleted token
				he.TokenReferences = append(he.TokenReferences, TokenRef{
					EventID:  token.event.ID,
					Created:  false,
					IsNutzap: false,
				})
			}
		} else {
			updatedTokens = append(updatedTokens, token)
		}
	}

	if len(changeToken.Proofs) > 0 {
		if err := changeToken.toEvent(ctx, w.kr, changeToken.event); err != nil {
			return fmt.Errorf("failed to make change token: %w", err)
		}
		w.Lock()
		w.PublishUpdate(*changeToken.event, nil, nil, &changeToken, false)
		w.Unlock()

		// we don't have to lock tokensMu here because this function will always be called with that lock already held
		w.Tokens = append(updatedTokens, changeToken)

		// fill in the history created token
		he.TokenReferences = append(he.TokenReferences, TokenRef{
			EventID:  changeToken.event.ID,
			Created:  true,
			IsNutzap: false,
		})
	}

	return nil
}

func (w *Wallet) getProofsForSending(
	ctx context.Context,
	amount uint64,
	fromMint string,
	excludeMints []string,
) (chosenTokens, uint64, error) {
	byMint := make(map[string]chosenTokens)
	for t, token := range w.Tokens {
		if token.reserved {
			continue
		}
		if fromMint != "" && token.Mint != fromMint {
			continue
		}
		if slices.Contains(excludeMints, token.Mint) {
			continue
		}

		part, ok := byMint[token.Mint]
		if !ok {
			keysets, err := client.GetAllKeysets(ctx, token.Mint)
			if err != nil {
				return chosenTokens{}, 0, fmt.Errorf("failed to get %s keysets: %w", token.Mint, err)
			}
			part.keysets = keysets
			part.tokens = make([]Token, 0, 3)
			part.tokenIndexes = make([]int, 0, 3)
			part.proofs = make(cashu.Proofs, 0, 7)
			part.mint = token.Mint
		}

		part.tokens = append(part.tokens, token)
		part.tokenIndexes = append(part.tokenIndexes, t)
		part.proofs = append(part.proofs, token.Proofs...)
		if part.proofs.Amount() >= amount {
			// maybe we found it here
			fee := calculateFee(part.proofs, part.keysets)
			if part.proofs.Amount() >= (amount + fee) {
				// yes, we did
				return part, fee, nil
			}
		}

		byMint[token.Mint] = part
	}

	// if we got here it's because we didn't get enough proofs from the same mint
	return chosenTokens{}, 0, fmt.Errorf("not enough proofs found from the same mint")
}
