package negentropy

import (
	"fmt"
	"iter"

	"fiatjaf.com/nostr"
)

const FingerprintSize = 16

type Mode uint8

const (
	SkipMode        Mode = 0
	FingerprintMode Mode = 1
	IdListMode      Mode = 2
)

func (v Mode) String() string {
	switch v {
	case SkipMode:
		return "SKIP"
	case FingerprintMode:
		return "FINGERPRINT"
	case IdListMode:
		return "IDLIST"
	default:
		return "<UNKNOWN-ERROR>"
	}
}

type Item struct {
	Timestamp nostr.Timestamp
	ID        nostr.ID
}

func (i Item) String() string { return fmt.Sprintf("Item<%d:%x>", i.Timestamp, i.ID[:]) }

type Bound struct {
	Timestamp nostr.Timestamp
	IDPrefix  []byte
}

func (b Bound) String() string {
	if b.Timestamp == InfiniteBound.Timestamp {
		return "Bound<infinite>"
	}
	return fmt.Sprintf("Bound<%d:%x>", b.Timestamp, b.IDPrefix)
}

type Storage interface {
	Size() int
	Range(begin, end int) iter.Seq2[int, Item]
	FindLowerBound(begin, end int, value Bound) int
	GetBound(idx int) Bound
	Fingerprint(begin, end int) [FingerprintSize]byte
}
