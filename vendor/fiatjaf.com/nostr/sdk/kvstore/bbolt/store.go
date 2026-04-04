package bbolt

import (
	"fiatjaf.com/nostr/sdk/kvstore"
	"go.etcd.io/bbolt"
)

var _ kvstore.KVStore = (*Store)(nil)

var (
	defaultBucket = []byte("default")
)

type Store struct {
	db     *bbolt.DB
	bucket []byte
}

func NewStore(path string) (*Store, error) {
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	// Create the default bucket
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(defaultBucket)
		return err
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db, bucket: defaultBucket}, nil
}

func (s *Store) Get(key []byte) ([]byte, error) {
	var val []byte
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(s.bucket)
		if b == nil {
			return nil
		}
		val = b.Get(key)
		if val != nil {
			// Make a copy since bbolt reuses the slice
			valCopy := make([]byte, len(val))
			copy(valCopy, val)
			val = valCopy
		}
		return nil
	})
	return val, err
}

func (s *Store) Set(key []byte, value []byte) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(s.bucket)
		return b.Put(key, value)
	})
}

func (s *Store) Delete(key []byte) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(s.bucket)
		return b.Delete(key)
	})
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Update(key []byte, f func([]byte) ([]byte, error)) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(s.bucket)
		var val []byte
		if v := b.Get(key); v != nil {
			val = make([]byte, len(v))
			copy(val, v)
		}

		newVal, err := f(val)
		if err == kvstore.NoOp {
			return nil
		} else if err != nil {
			return err
		}

		if newVal == nil {
			return b.Delete(key)
		}
		return b.Put(key, newVal)
	})
}
