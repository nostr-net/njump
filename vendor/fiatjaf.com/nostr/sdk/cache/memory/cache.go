package cache_memory

import (
	"encoding/binary"
	"time"

	"github.com/dgraph-io/ristretto/v2"
)

type RistrettoCache[V any] struct {
	Cache *ristretto.Cache[uint64, V]
}

func New[V any](max int64) *RistrettoCache[V] {
	cache, _ := ristretto.NewCache(&ristretto.Config[uint64, V]{
		NumCounters: max * 10,
		MaxCost:     max,
		BufferItems: 64,
		KeyToHash:   func(key uint64) (uint64, uint64) { return key, 0 },
	})
	return &RistrettoCache[V]{Cache: cache}
}

func (s RistrettoCache[V]) Get(k [32]byte) (v V, ok bool) {
	return s.Cache.Get(binary.BigEndian.Uint64(k[32-8:]))
}
func (s RistrettoCache[V]) Delete(k [32]byte) { s.Cache.Del(binary.BigEndian.Uint64(k[32-8:])) }
func (s RistrettoCache[V]) Set(k [32]byte, v V) bool {
	return s.Cache.Set(binary.BigEndian.Uint64(k[32-8:]), v, 1)
}

func (s RistrettoCache[V]) SetWithTTL(k [32]byte, v V, d time.Duration) bool {
	return s.Cache.SetWithTTL(binary.BigEndian.Uint64(k[32-8:]), v, 1, d)
}
