package cache

import "time"

type Cache32[V any] interface {
	Get(k [32]byte) (v V, ok bool)
	Delete(k [32]byte)
	Set(k [32]byte, v V) bool
	SetWithTTL(k [32]byte, v V, d time.Duration) bool
}
