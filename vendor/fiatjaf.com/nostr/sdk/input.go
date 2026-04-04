package sdk

import (
	"context"

	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/nip05"
	"fiatjaf.com/nostr/nip19"
)

// InputToProfile turns any npub/nprofile/hex/nip05 input into a ProfilePointer (or nil).
func InputToProfile(ctx context.Context, input string) *nostr.ProfilePointer {
	// handle if it is a hex string
	if pk, err := nostr.PubKeyFromHex(input); err == nil {
		return &nostr.ProfilePointer{PublicKey: pk}
	}

	// handle nip19 codes, if that's the case
	prefix, data, err := nip19.Decode(input)
	if err == nil {
		switch prefix {
		case "npub":
			return &nostr.ProfilePointer{PublicKey: data.(nostr.PubKey)}
		case "nprofile":
			pp := data.(nostr.ProfilePointer)
			return &pp
		}
	}

	// handle nip05 ids, if that's the case
	pp, _ := nip05.QueryIdentifier(ctx, input)
	if pp != nil {
		return pp
	}

	return nil
}

// InputToEventPointer turns any note/nevent/hex input into a EventPointer (or nil).
func InputToEventPointer(input string) *nostr.EventPointer {
	// handle if it is a hex string
	if id, err := nostr.IDFromHex(input); err == nil {
		return &nostr.EventPointer{ID: id}
	}

	// handle nip19 codes, if that's the case
	prefix, data, _ := nip19.Decode(input)
	switch prefix {
	case "note":
		return &nostr.EventPointer{ID: data.(nostr.ID)}
	case "nevent":
		if ep, ok := data.(nostr.EventPointer); ok {
			return &ep
		}
	}

	// handle nip05 ids, if that's the case
	return nil
}
