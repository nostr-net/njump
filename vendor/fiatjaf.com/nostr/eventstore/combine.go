package eventstore

import (
	"iter"
	"slices"

	"fiatjaf.com/nostr"
)

// SortedMerge combines two iterators and returns the top limit results aggregated from both.
// limit is implied to be also the maximum number of items each iterator will return.
func SortedMerge(it1, it2 iter.Seq[nostr.Event], limit int) iter.Seq[nostr.Event] {
	if limit < 60 {
		return func(yield func(nostr.Event) bool) {
			acc := make([]nostr.Event, 0, limit*2)
			for evt := range it1 {
				acc = append(acc, evt)
			}
			for evt := range it2 {
				acc = append(acc, evt)
			}
			slices.SortFunc(acc, nostr.CompareEventReverse)
			for i := range min(limit, len(acc)) {
				if !yield(acc[i]) {
					return
				}
			}
		}
	}

	next1, done1 := iter.Pull(it1)
	next2, done2 := iter.Pull(it2)

	return func(yieldInner func(nostr.Event) bool) {
		count := 0
		yield := func(evt nostr.Event) bool {
			shouldContinue := yieldInner(evt)
			count++
			if count >= limit {
				return false
			}
			return shouldContinue
		}

		defer done1()
		defer done2()

		evt1, ok1 := next1()
		evt2, ok2 := next2()

	both:
		if ok1 && ok2 {
			if evt2.CreatedAt > evt1.CreatedAt {
				if !yield(evt2) {
					return
				}
				evt2, ok2 = next2()
				goto both
			} else {
				if !yield(evt1) {
					return
				}
				evt1, ok1 = next1()
				goto both
			}
		}

		if !ok2 {
		only1:
			if ok1 {
				if !yield(evt1) {
					return
				}
				evt1, ok1 = next1()
				goto only1
			}
		}

		if !ok1 {
		only2:
			if ok2 {
				if !yield(evt2) {
					return
				}
				evt2, ok2 = next2()
				goto only2
			}
		}

		return
	}
}
