package nip61

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"slices"
	"strings"

	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/nip60"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/elnosh/gonuts/cashu"
)

var NutzapsNotAccepted = errors.New("user doesn't accept nutzaps")

type NutzapOptions struct {
	// Optionally specify the event we'll reference in the nutzap,
	// if not specified we'll just send money to the receiver
	EventID nostr.ID

	// string message to include in the nutzap
	Message string

	// We'll send the nutzap to these relays besides any relay found in the kind:10019
	SendToRelays []string

	// Send specifically from this mint
	SpecificSourceMint string
}

func SendNutzap(
	ctx context.Context,
	kr nostr.Keyer,
	w *nip60.Wallet,
	pool *nostr.Pool,
	amount uint64,
	targetUser nostr.PubKey,
	targetUserRelays []string,
	opts NutzapOptions,
) (chan nostr.PublishResult, error) {
	ie := pool.QuerySingle(ctx, targetUserRelays, nostr.Filter{
		Kinds:   []nostr.Kind{10019},
		Authors: []nostr.PubKey{targetUser},
	},
		nostr.SubscriptionOptions{Label: "pre-nutzap"})
	if ie == nil {
		return nil, NutzapsNotAccepted
	}

	info := Info{}
	if err := info.ParseEvent(ie.Event); err != nil {
		return nil, err
	}

	if len(info.Mints) == 0 || info.PublicKey == nostr.ZeroPK {
		return nil, NutzapsNotAccepted
	}

	targetRelays := nostr.AppendUnique(info.Relays, opts.SendToRelays...)
	if len(targetRelays) == 0 {
		return nil, fmt.Errorf("no relays found for sending the nutzap")
	}

	nutzap := nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindNutZap,
		Tags:      make(nostr.Tags, 0, 8),
	}

	nutzap.Tags = append(nutzap.Tags, nostr.Tag{"p", targetUser.Hex()})
	if opts.EventID != nostr.ZeroID {
		nutzap.Tags = append(nutzap.Tags, nostr.Tag{"e", opts.EventID.Hex()})
	}

	p2pk, err := btcec.ParsePubKey(append([]byte{2}, info.PublicKey[:]...))
	if err != nil {
		return nil, fmt.Errorf("invalid p2pk target '%s': %w", info.PublicKey.Hex(), err)
	}

	// check if we have enough tokens in any of these mints
	for mint := range getEligibleTokensWeHave(info.Mints, w.Tokens, amount) {
		if opts.SpecificSourceMint != "" && opts.SpecificSourceMint != mint {
			continue
		}

		proofs, _, err := w.SendInternal(ctx, amount, nip60.SendOptions{
			P2PK:               p2pk,
			SpecificSourceMint: mint,
		})
		if err != nil {
			continue
		}

		// we have succeeded, now we just have to publish the event
		nutzap.Tags = append(nutzap.Tags, nostr.Tag{"u", mint})
		for _, proof := range proofs {
			proofj, _ := json.Marshal(proof)
			nutzap.Tags = append(nutzap.Tags, nostr.Tag{"proof", string(proofj)})
		}

		if err := kr.SignEvent(ctx, &nutzap); err != nil {
			return nil, fmt.Errorf("failed to sign nutzap event %s: %w", nutzap, err)
		}

		return pool.PublishMany(ctx, targetRelays, nutzap), nil
	}

	// we don't have tokens at the desired target mint, so we first have to create some
	for _, mint := range info.Mints {
		proofs, err := w.SendExternal(ctx, mint, amount, nip60.SendOptions{
			P2PK:               p2pk,
			SpecificSourceMint: opts.SpecificSourceMint,
		})
		if err != nil {
			if strings.Contains(err.Error(), "generate mint quote") {
				continue
			}
			return nil, fmt.Errorf("failed to send: %w", err)
		}

		// we have succeeded, now we just have to publish the event
		nutzap.Tags = append(nutzap.Tags, nostr.Tag{"u", mint})
		for _, proof := range proofs {
			proofj, _ := json.Marshal(proof)
			nutzap.Tags = append(nutzap.Tags, nostr.Tag{"proof", string(proofj)})
		}

		if err := kr.SignEvent(ctx, &nutzap); err != nil {
			return nil, fmt.Errorf("failed to sign nutzap event %s: %w", nutzap, err)
		}

		return pool.PublishMany(ctx, targetRelays, nutzap), nil
	}

	return nil, fmt.Errorf("failed to send, we don't have enough money or all mints are down")
}

func getEligibleTokensWeHave(
	theirMints []string,
	ourTokens []nip60.Token,
	targetAmount uint64,
) iter.Seq[string] {
	have := make([]uint64, len(theirMints))

	return func(yield func(string) bool) {
		for _, token := range ourTokens {
			if idx := slices.Index(theirMints, token.Mint); idx != -1 {
				have[idx] += token.Proofs.Amount()

				/*                          hardcoded estimated maximum fee,
				                            unlikely to be more than this */
				if have[idx] > targetAmount*101/100+2 {
					if !yield(token.Mint) {
						break
					}
				}
			}
		}
	}
}

// GetAmountFromNutzap parses and sums all the proofs in a nutzap, returns the amount.
//
// The amount will be in the unit corresponding to the mint keys, which is often "sat" but can be something else.
func GetAmountFromNutzap(evt nostr.Event) uint64 {
	var total uint64
	for _, tag := range evt.Tags {
		if len(tag) >= 2 && tag[0] == "proof" {
			var proof cashu.Proof
			if err := json.Unmarshal([]byte(tag[1]), &proof); err != nil {
				continue
			}
			total += proof.Amount
		}
	}
	return total
}
