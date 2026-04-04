package nip60

import (
	"context"
	"fmt"

	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/nip60/client"
	"github.com/elnosh/gonuts/cashu"
)

func (w *Wallet) SendInternal(ctx context.Context, amount uint64, opts SendOptions) (cashu.Proofs, string, error) {
	if w.PublishUpdate == nil {
		return nil, "", fmt.Errorf("can't do write operations: missing PublishUpdate function")
	}

	w.tokensMu.Lock()
	defer w.tokensMu.Unlock()

	chosen, _, err := w.getProofsForSending(ctx, amount, opts.SpecificSourceMint, nil)
	if err != nil {
		return nil, "", err
	}

	if opts.Hashlock != [32]byte{} {
		if info, err := client.GetMintInfo(ctx, chosen.mint); err != nil || !info.Nuts.Nut14.Supported {
			return nil, chosen.mint, fmt.Errorf("mint doesn't support htlc: %w", err)
		}
	} else if opts.P2PK != nil {
		if info, err := client.GetMintInfo(ctx, chosen.mint); err != nil || !info.Nuts.Nut11.Supported {
			return nil, chosen.mint, fmt.Errorf("mint doesn't support p2pk: %w", err)
		}
	}

	swapSettings := swapSettings{
		spendingCondition: opts.asSpendingCondition(w.PublicKey),
	}

	// get new proofs
	proofsToSend, changeProofs, err := w.swapProofs(ctx, chosen.mint, chosen.proofs, amount, swapSettings)
	if err != nil {
		return nil, chosen.mint, err
	}

	he := HistoryEntry{
		event:           &nostr.Event{},
		TokenReferences: make([]TokenRef, 0, 5),
		createdAt:       nostr.Now(),
		In:              false,
		Amount:          chosen.proofs.Amount() - changeProofs.Amount(),
	}

	if err := w.saveChangeAndDeleteUsedTokens(ctx, chosen.mint, changeProofs, chosen.tokenIndexes, &he); err != nil {
		return nil, chosen.mint, err
	}

	w.Lock()
	if err := he.toEvent(ctx, w.kr, he.event); err == nil {
		w.PublishUpdate(*he.event, nil, nil, nil, true)
	}
	w.Unlock()

	return proofsToSend, chosen.mint, nil
}
