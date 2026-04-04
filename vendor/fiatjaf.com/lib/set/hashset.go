package set

import (
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/maps"
)

type HashSet[V constraints.Ordered] struct {
	internal map[V]bool
}

func NewHashSet[V constraints.Ordered](items ...V) Set[V] {
	internal := make(map[V]bool, len(items))
	for _, item := range items {
		internal[item] = true
	}
	return HashSet[V]{internal: internal}
}

func (s HashSet[V]) Add(items ...V) {
	for _, item := range items {
		s.internal[item] = true
	}
}

func (s HashSet[V]) Has(n V) bool {
	_, ok := s.internal[n]
	return ok
}

func (s HashSet[V]) Intersection(other Set[V]) Set[V] {
	inter := HashSet[V]{internal: make(map[V]bool, len(s.internal))}
	for k := range s.internal {
		if other.Has(k) {
			inter.Add(k)
		}
	}
	return inter
}

func (s HashSet[V]) Difference(other Set[V]) Set[V] {
	diff := HashSet[V]{internal: make(map[V]bool, len(s.internal))}
	for k := range s.internal {
		if !other.Has(k) {
			diff.Add(k)
		}
	}
	return diff
}

func (s HashSet[V]) Union(other Set[V]) Set[V] {
	union := HashSet[V]{internal: make(map[V]bool, len(s.internal)+other.Len())}
	for k := range s.internal {
		union.Add(k)
	}
	union.Add(other.Slice()...)
	return union
}

func (s HashSet[V]) Remove(items ...V) {
	for _, item := range items {
		delete(s.internal, item)
	}
}

func (s HashSet[V]) Slice() []V {
	return maps.Keys(s.internal)
}

func (s HashSet[V]) Len() int {
	return len(s.internal)
}
