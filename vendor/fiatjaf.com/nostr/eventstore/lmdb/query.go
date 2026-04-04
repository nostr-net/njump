package lmdb

import (
	"iter"
	"log"
	"math"
	"slices"

	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/eventstore/codec/betterbinary"
	"fiatjaf.com/nostr/eventstore/internal"
	"github.com/PowerDNS/lmdb-go/lmdb"
)

func (b *LMDBBackend) QueryEvents(filter nostr.Filter, maxLimit int) iter.Seq[nostr.Event] {
	return func(yield func(nostr.Event) bool) {
		if filter.IDs != nil {
			// when there are ids we ignore everything else and just fetch the ids
			if err := b.lmdbEnv.View(func(txn *lmdb.Txn) error {
				txn.RawRead = true
				return b.queryByIds(txn, filter.IDs, yield)
			}); err != nil {
				log.Printf("lmdb: unexpected id query error: %s\n", err)
			}
			return
		}

		// ignore search queries
		if filter.Search != "" {
			return
		}

		// max number of events we'll return
		if tlimit := filter.GetTheoreticalLimit(); tlimit == 0 {
			return
		} else if tlimit < maxLimit {
			maxLimit = tlimit
		}

		// do a normal query based on various filters
		if err := b.lmdbEnv.View(func(txn *lmdb.Txn) error {
			txn.RawRead = true
			return b.query(txn, filter, maxLimit, yield)
		}); err != nil {
			log.Printf("lmdb: unexpected query error: %s\n", err)
		}
	}
}

func (b *LMDBBackend) queryByIds(txn *lmdb.Txn, ids []nostr.ID, yield func(nostr.Event) bool) error {
	for _, id := range ids {
		idx, err := txn.Get(b.indexId, id[0:8])
		if err != nil {
			continue
		}

		txn.Get(b.rawEventStore, idx)
		bin, err := txn.Get(b.rawEventStore, idx)
		if err != nil {
			continue
		}

		event := nostr.Event{}
		if err := betterbinary.Unmarshal(bin, &event); err != nil {
			continue
		}

		if !yield(event) {
			return nil
		}
	}

	return nil
}

func (b *LMDBBackend) query(txn *lmdb.Txn, filter nostr.Filter, limit int, yield func(nostr.Event) bool) error {
	queries, extraAuthors, extraKinds, extraTagKey, extraTagValues, since, err := b.prepareQueries(filter)
	if err != nil {
		return err
	}

	iterators := make(iterators, len(queries))
	batchSizePerQuery := internal.BatchSizePerNumberOfQueries(limit, len(queries))

	for q, query := range queries {
		cursor, err := txn.OpenCursor(queries[q].dbi)
		if err != nil {
			return err
		}
		iterators[q] = &iterator{
			query:  query,
			cursor: cursor,
		}

		defer cursor.Close()
		iterators[q].seek(queries[q].startingPoint)

		// initial pull
		iterators[q].pull(batchSizePerQuery, since)
	}

	numberOfIteratorsToPullOnEachRound := max(1, int(math.Ceil(float64(len(iterators))/float64(12))))
	totalEventsEmitted := 0
	tempResults := make([]nostr.Event, 0, batchSizePerQuery*2)

	for len(iterators) > 0 {
		// reset stuff
		tempResults = tempResults[:0]

		// after pulling from all iterators once we now find out what iterators are
		// the ones we should keep pulling from next (i.e. which one's last emitted timestamp is the highest)
		k := min(numberOfIteratorsToPullOnEachRound, len(iterators))
		iterators.quickselect(k)
		threshold := iterators.threshold(k)

		// so we can emit all the events higher than the threshold
		for i := range iterators {
			for t := 0; t < len(iterators[i].timestamps); t++ {
				if iterators[i].timestamps[t] >= threshold {
					idx := iterators[i].idxs[t]

					// discard this regardless of what happens
					iterators[i].timestamps = internal.SwapDelete(iterators[i].timestamps, t)
					iterators[i].idxs = internal.SwapDelete(iterators[i].idxs, t)
					t--

					// fetch actual event
					bin, err := txn.Get(b.rawEventStore, idx)
					if err != nil {
						log.Printf("lmdb: failed to get %x from raw event store: %s (query prefix=%x, index=%s)\n",
							idx, err, iterators[i].query.prefix, b.dbiName(iterators[i].query.dbi))
						continue
					}

					// check it against pubkeys without decoding the entire thing
					if extraAuthors != nil && !slices.Contains(extraAuthors, betterbinary.GetPubKey(bin)) {
						continue
					}

					// check it against kinds without decoding the entire thing
					if extraKinds != nil && !slices.Contains(extraKinds, betterbinary.GetKind(bin)) {
						continue
					}

					// decode the entire thing
					event := nostr.Event{}
					if err := betterbinary.Unmarshal(bin, &event); err != nil {
						log.Printf("lmdb: value read error (id %s) on query prefix %x sp %x dbi %s: %s\n",
							betterbinary.GetID(bin).Hex(), iterators[i].query.prefix, iterators[i].query.startingPoint, b.dbiName(iterators[i].query.dbi), err)
						continue
					}

					// if there is still a tag to be checked, do it now
					if extraTagValues != nil && !event.Tags.ContainsAny(extraTagKey, extraTagValues) {
						continue
					}

					tempResults = append(tempResults, event)
				}
			}
		}

		// emit this stuff in order
		slices.SortFunc(tempResults, nostr.CompareEventReverse)
		for _, evt := range tempResults {
			if !yield(evt) {
				return nil
			}

			totalEventsEmitted++
			if totalEventsEmitted == limit {
				return nil
			}
		}

		// now pull more events
		for i := 0; i < min(len(iterators), numberOfIteratorsToPullOnEachRound); i++ {
			if iterators[i].exhausted {
				if len(iterators[i].idxs) == 0 {
					// eliminating this from the list of iterators
					iterators = internal.SwapDelete(iterators, i)
					i--
				}
				continue
			}

			iterators[i].pull(batchSizePerQuery, since)
		}
	}

	return nil
}
