package nip19

import (
	"fmt"

	"fiatjaf.com/nostr"
)

func EncodePointer(pointer nostr.Pointer) string {
	switch v := pointer.(type) {
	case nostr.ProfilePointer:
		if v.Relays == nil {
			return EncodeNpub(v.PublicKey)
		} else {
			return EncodeNprofile(v.PublicKey, v.Relays)
		}
	case nostr.EventPointer:
		return EncodeNevent(v.ID, v.Relays, v.Author)
	case nostr.EntityPointer:
		return EncodeNaddr(v.PublicKey, v.Kind, v.Identifier, v.Relays)
	}
	return ""
}

func ToPointer(code string) (nostr.Pointer, error) {
	prefix, data, err := Decode(code)
	if err != nil {
		return nil, err
	}

	switch prefix {
	case "npub":
		return nostr.ProfilePointer{PublicKey: data.(nostr.PubKey)}, nil
	case "nprofile":
		return data.(nostr.ProfilePointer), nil
	case "nevent":
		return data.(nostr.EventPointer), nil
	case "note":
		return nostr.EventPointer{ID: data.(nostr.ID)}, nil
	case "naddr":
		return data.(nostr.EntityPointer), nil
	default:
		return nil, fmt.Errorf("unexpected prefix '%s' to '%s'", prefix, code)
	}
}
