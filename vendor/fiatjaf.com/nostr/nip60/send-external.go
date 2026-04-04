package nip60

import (
	"context"
	"fmt"

	"fiatjaf.com/nostr/nip60/client"
	"github.com/elnosh/gonuts/cashu"
	"github.com/elnosh/gonuts/cashu/nuts/nut04"
)

func (w *Wallet) SendExternal(
	ctx context.Context,
	mint string,
	targetAmount uint64,
	opts SendOptions,
) (cashu.Proofs, error) {
	if w.PublishUpdate == nil {
		return nil, fmt.Errorf("can't do write operations: missing PublishUpdate function")
	}

	// get the invoice from target mint
	mintResp, err := client.PostMintQuoteBolt11(ctx, mint, nut04.PostMintQuoteBolt11Request{
		Unit:   cashu.Sat.String(),
		Amount: targetAmount,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate mint quote: %w", err)
	}

	if _, err := w.PayBolt11(ctx, mintResp.Request, PayOptions{
		FromMint: opts.SpecificSourceMint,
	}); err != nil {
		return nil, err
	}

	return redeemMinted(ctx, mint, mintResp.Quote, targetAmount, opts.asSpendingCondition(w.PublicKey))
}
