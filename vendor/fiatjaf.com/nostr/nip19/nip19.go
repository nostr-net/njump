package nip19

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"fiatjaf.com/nostr"
	"github.com/btcsuite/btcd/btcutil/bech32"
)

func Decode(bech32string string) (prefix string, value any, err error) {
	prefix, bits5, err := bech32.DecodeNoLimit(bech32string)
	if err != nil {
		return "", nil, err
	}

	data, err := bech32.ConvertBits(bits5, 5, 8, false)
	if err != nil {
		return prefix, nil, fmt.Errorf("failed to translate data into 8 bits: %s", err.Error())
	}

	switch prefix {
	case "nsec":
		if len(data) != 32 {
			return prefix, nil, fmt.Errorf("nsec should be 32 bytes (%d)", len(data))
		}
		return prefix, nostr.SecretKey(data[0:32]), nil
	case "note":
		if len(data) != 32 {
			return prefix, nil, fmt.Errorf("note should be 32 bytes (%d)", len(data))
		}
		return prefix, nostr.EventPointer{
			ID: nostr.ID(data[0:32]),
		}, nil
	case "npub":
		if len(data) != 32 {
			return prefix, nil, fmt.Errorf("npub should be 32 bytes (%d)", len(data))
		}
		return prefix, nostr.PubKey(data[0:32]), nil
	case "nprofile":
		var result nostr.ProfilePointer
		curr := 0
		for {
			t, v := readTLVEntry(data[curr:])
			if v == nil {
				// end here
				if result.PublicKey == nostr.ZeroPK {
					return prefix, result, fmt.Errorf("no pubkey found for nprofile")
				}

				return prefix, result, nil
			}

			switch t {
			case TLVDefault:
				if len(v) != 32 {
					return prefix, nil, fmt.Errorf("pubkey should be 32 bytes (%d)", len(v))
				}
				result.PublicKey = nostr.PubKey(v)
			case TLVRelay:
				result.Relays = append(result.Relays, string(v))
			default:
				// ignore
			}

			curr = curr + 2 + len(v)
		}
	case "nevent":
		var result nostr.EventPointer
		curr := 0
		for {
			t, v := readTLVEntry(data[curr:])
			if v == nil {
				// end here
				if result.ID == nostr.ZeroID {
					return prefix, result, fmt.Errorf("no id found for nevent")
				}

				return prefix, result, nil
			}

			switch t {
			case TLVDefault:
				if len(v) != 32 {
					return prefix, nil, fmt.Errorf("id should be 32 bytes (%d)", len(v))
				}
				result.ID = nostr.ID(v)
			case TLVRelay:
				result.Relays = append(result.Relays, string(v))
			case TLVAuthor:
				if len(v) != 32 {
					return prefix, nil, fmt.Errorf("author should be 32 bytes (%d)", len(v))
				}
				result.Author = nostr.PubKey(v)
			case TLVKind:
				if len(v) != 4 {
					return prefix, nil, fmt.Errorf("invalid uint32 value for integer (%v)", v)
				}
				result.Kind = nostr.Kind(binary.BigEndian.Uint32(v))
			default:
				// ignore
			}

			curr = curr + 2 + len(v)
		}
	case "naddr":
		var result nostr.EntityPointer
		var hasIdentifier bool
		curr := 0
		for {
			t, v := readTLVEntry(data[curr:])
			if v == nil {
				// end here
				if result.Kind == 0 || !hasIdentifier || result.PublicKey == nostr.ZeroPK {
					return prefix, result, fmt.Errorf("incomplete naddr")
				}

				return prefix, result, nil
			}

			switch t {
			case TLVDefault:
				result.Identifier = string(v)
				hasIdentifier = true
			case TLVRelay:
				result.Relays = append(result.Relays, string(v))
			case TLVAuthor:
				if len(v) != 32 {
					return prefix, nil, fmt.Errorf("author should be 32 bytes (%d)", len(v))
				}
				result.PublicKey = nostr.PubKey(v)
			case TLVKind:
				result.Kind = nostr.Kind(binary.BigEndian.Uint32(v))
			default:
				// ignore
			}

			curr = curr + 2 + len(v)
		}
	}

	return prefix, data, fmt.Errorf("unknown tag %s", prefix)
}

func EncodeNsec(sk [32]byte) string {
	bits5, _ := bech32.ConvertBits(sk[:], 8, 5, true)
	nsec, _ := bech32.Encode("nsec", bits5)
	return nsec
}

func EncodeNpub(pk nostr.PubKey) string {
	bits5, _ := bech32.ConvertBits(pk[:], 8, 5, true)
	npub, _ := bech32.Encode("npub", bits5)
	return npub
}

func EncodeNprofile(pk nostr.PubKey, relays []string) string {
	buf := &bytes.Buffer{}
	writeTLVEntry(buf, TLVDefault, pk[:])

	for _, url := range relays {
		writeTLVEntry(buf, TLVRelay, []byte(url))
	}

	bits5, _ := bech32.ConvertBits(buf.Bytes(), 8, 5, true)

	nprofile, _ := bech32.Encode("nprofile", bits5)
	return nprofile
}

func EncodeNevent(id nostr.ID, relays []string, author nostr.PubKey) string {
	buf := &bytes.Buffer{}
	writeTLVEntry(buf, TLVDefault, id[:])

	for _, url := range relays {
		writeTLVEntry(buf, TLVRelay, []byte(url))
	}

	if author != nostr.ZeroPK {
		writeTLVEntry(buf, TLVAuthor, author[:])
	}

	bits5, _ := bech32.ConvertBits(buf.Bytes(), 8, 5, true)
	nevent, _ := bech32.Encode("nevent", bits5)
	return nevent
}

func EncodeNaddr(pk nostr.PubKey, kind nostr.Kind, identifier string, relays []string) string {
	buf := &bytes.Buffer{}

	writeTLVEntry(buf, TLVDefault, []byte(identifier))

	for _, url := range relays {
		writeTLVEntry(buf, TLVRelay, []byte(url))
	}

	writeTLVEntry(buf, TLVAuthor, pk[:])

	kindBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(kindBytes, uint32(kind))
	writeTLVEntry(buf, TLVKind, kindBytes)

	bits5, _ := bech32.ConvertBits(buf.Bytes(), 8, 5, true)
	naddr, _ := bech32.Encode("naddr", bits5)
	return naddr
}
