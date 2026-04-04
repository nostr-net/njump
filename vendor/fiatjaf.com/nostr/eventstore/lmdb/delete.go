package lmdb

import (
	"fmt"

	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/eventstore/codec/betterbinary"
	"github.com/PowerDNS/lmdb-go/lmdb"
)

func (b *LMDBBackend) DeleteEvent(id nostr.ID) error {
	return b.lmdbEnv.Update(func(txn *lmdb.Txn) error {
		return b.delete(txn, id)
	})
}

func (b *LMDBBackend) delete(txn *lmdb.Txn, id nostr.ID) error {
	// check if we have this actually
	idx, err := txn.Get(b.indexId, id[0:8])
	if lmdb.IsNotFound(err) {
		// we already do not have this
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get current idx for deleting %x: %w", id[0:8], err)
	}

	// if we do, get it so we can compute the indexes
	bin, err := txn.Get(b.rawEventStore, idx)
	if err != nil {
		return fmt.Errorf("failed to get raw event %x to delete: %w", id, err)
	}

	var evt nostr.Event
	if err := betterbinary.Unmarshal(bin, &evt); err != nil {
		return fmt.Errorf("failed to unmarshal raw event %x to delete: %w", id, err)
	}

	// calculate all index keys we have for this event and delete them
	for k := range b.getIndexKeysForEvent(evt) {
		err := txn.Del(k.dbi, k.key, idx)
		if err != nil {
			return fmt.Errorf("failed to delete index entry %s for %x: %w", b.keyName(k), evt.ID[0:8], err)
		}
	}

	// delete the raw event
	if err := txn.Del(b.rawEventStore, idx, nil); err != nil {
		return fmt.Errorf("failed to delete raw event %x (idx %x): %w", evt.ID[0:8], idx, err)
	}

	return nil
}
