package vector

import (
	"bytes"
	"cmp"

	"fiatjaf.com/nostr/nip77/negentropy"
)

func itemCompare(a, b negentropy.Item) int {
	if a.Timestamp == b.Timestamp {
		return bytes.Compare(a.ID[:], b.ID[:])
	}
	return cmp.Compare(a.Timestamp, b.Timestamp)
}

// binary search with custom function
func searchItemWithBound(items []negentropy.Item, bound negentropy.Bound) int {
	n := len(items)
	// Define x[-1] < target and x[n] >= target.
	// Invariant: x[i-1] < target, x[j] >= target.
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1) // avoid overflow when computing h
		// i ≤ h < j
		if items[h].Timestamp < bound.Timestamp ||
			(items[h].Timestamp == bound.Timestamp && bytes.Compare(items[h].ID[:], bound.IDPrefix) == -1) {
			i = h + 1 // preserves x[i-1] < target
		} else {
			j = h // preserves x[j] >= target
		}
	}
	// i == j, x[i-1] < target, and x[j] (= x[i]) >= target  =>  answer is i.
	return i
}
