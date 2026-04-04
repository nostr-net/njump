package nip61

import (
	"context"
	"fmt"
	"slices"

	"fiatjaf.com/nostr"
	"github.com/elnosh/gonuts/cashu"
)

type Info struct {
	PublicKey nostr.PubKey
	Mints     []string
	Relays    []string
}

func (zi *Info) ToEvent(ctx context.Context, kr nostr.Keyer, evt *nostr.Event) error {
	evt.CreatedAt = nostr.Now()
	evt.Kind = 10019

	evt.Tags = make(nostr.Tags, 0, len(zi.Mints)+len(zi.Relays)+1)
	for _, mint := range zi.Mints {
		url, err := nostr.NormalizeHTTPURL(mint)
		if err != nil {
			return fmt.Errorf("invalid URL '%s': %w", mint, err)
		}

		evt.Tags = append(evt.Tags, nostr.Tag{"mint", url})
	}
	for _, url := range zi.Relays {
		evt.Tags = append(evt.Tags, nostr.Tag{"relay", url})
	}
	if zi.PublicKey != nostr.ZeroPK {
		evt.Tags = append(evt.Tags, nostr.Tag{"pubkey", zi.PublicKey.Hex()})
	}

	if err := kr.SignEvent(ctx, evt); err != nil {
		return err
	}

	return nil
}

func (zi *Info) ParseEvent(evt nostr.Event) error {
	zi.Mints = make([]string, 0)
	for _, tag := range evt.Tags {
		if len(tag) < 2 {
			continue
		}

		switch tag[0] {
		case "mint":
			if len(tag) == 2 || slices.Contains(tag[2:], cashu.Sat.String()) {
				if url, err := nostr.NormalizeHTTPURL(tag[1]); err == nil {
					zi.Mints = append(zi.Mints, url)
				}
			}
		case "relay":
			zi.Relays = append(zi.Relays, tag[1])
		case "pubkey":
			if pk, err := nostr.PubKeyFromHex(tag[1]); err == nil {
				zi.PublicKey = pk
			}
		}
	}

	return nil
}
