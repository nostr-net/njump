package vector

import (
	"iter"
	"slices"

	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/nip77/negentropy"
	"fiatjaf.com/nostr/nip77/negentropy/storage"
)

type Vector struct {
	items  []negentropy.Item
	sealed bool

	acc storage.Accumulator
}

func New() *Vector {
	return &Vector{
		items: make([]negentropy.Item, 0, 30),
	}
}

func (v *Vector) Insert(createdAt nostr.Timestamp, id nostr.ID) {
	item := negentropy.Item{Timestamp: createdAt, ID: id}
	v.items = append(v.items, item)
}

func (v *Vector) Size() int { return len(v.items) }

func (v *Vector) Seal() {
	if v.sealed {
		panic("trying to seal an already sealed vector")
	}
	v.sealed = true
	slices.SortFunc(v.items, itemCompare)
}

func (v *Vector) GetBound(idx int) negentropy.Bound {
	if idx < len(v.items) {
		return negentropy.Bound{Timestamp: v.items[idx].Timestamp, IDPrefix: v.items[idx].ID[:]}
	}
	return negentropy.InfiniteBound
}

func (v *Vector) Range(begin, end int) iter.Seq2[int, negentropy.Item] {
	return func(yield func(int, negentropy.Item) bool) {
		for i := begin; i < end; i++ {
			if !yield(i, v.items[i]) {
				break
			}
		}
	}
}

func (v *Vector) FindLowerBound(begin, end int, bound negentropy.Bound) int {
	idx := searchItemWithBound(v.items[begin:end], bound)
	return begin + idx
}

func (v *Vector) Fingerprint(begin, end int) [negentropy.FingerprintSize]byte {
	v.acc.Reset()

	for _, item := range v.Range(begin, end) {
		v.acc.AddBytes(item.ID[:])
	}

	return v.acc.GetFingerprint(end - begin)
}
