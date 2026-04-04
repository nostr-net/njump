package betterbinary

import (
	"encoding/binary"

	"fiatjaf.com/nostr"
)

func GetKind(evtb []byte) nostr.Kind {
	return nostr.Kind(binary.LittleEndian.Uint16(evtb[1:3]))
}

func GetID(evtb []byte) nostr.ID {
	return nostr.ID(evtb[7:39])
}

func GetPubKey(evtb []byte) nostr.PubKey {
	return nostr.PubKey(evtb[39:71])
}

func GetCreatedAt(evtb []byte) nostr.Timestamp {
	return nostr.Timestamp(binary.LittleEndian.Uint32(evtb[3:7]))
}
