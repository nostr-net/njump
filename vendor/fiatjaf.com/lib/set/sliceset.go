package set

import (
	"slices"

	"golang.org/x/exp/constraints"
)

type SliceSet[A constraints.Ordered] struct {
	items []A
}

func NewSliceSet[A constraints.Ordered](items ...A) Set[A] {
	slices.Sort(items)
	return &SliceSet[A]{items: items}
}

func (s *SliceSet[A]) Add(items ...A) {
	for _, a := range items {
		idx, exists := slices.BinarySearch(s.items, a)
		if exists {
			return
		}
		s.items = append(s.items, a) // bogus append just to increase the capacity
		copy(s.items[idx+1:], s.items[idx:])
		s.items[idx] = a
	}
}

func (s *SliceSet[A]) Has(item A) bool {
	_, exists := slices.BinarySearch(s.items, item)
	return exists
}

func (s *SliceSet[V]) Intersection(other Set[V]) Set[V] {
	inter := SliceSet[V]{items: make([]V, 0, len(s.items))}
	for _, k := range s.items {
		if other.Has(k) {
			inter.Add(k)
		}
	}
	return &inter
}

func (s *SliceSet[V]) Difference(other Set[V]) Set[V] {
	diff := SliceSet[V]{items: make([]V, 0, len(s.items))}
	for _, k := range s.items {
		if !other.Has(k) {
			diff.Add(k)
		}
	}
	return &diff
}

func (s SliceSet[V]) Union(other Set[V]) Set[V] {
	union := SliceSet[V]{items: make([]V, 0, len(s.items)+other.Len())}
	copy(union.items, s.items)
	union.Add(other.Slice()...)
	return &union
}

func (s *SliceSet[A]) Remove(items ...A) {
	for _, a := range items {
		idx, exists := slices.BinarySearch(s.items, a)
		if !exists {
			continue
		}
		copy(s.items[idx:], s.items[idx+1:])
		s.items = s.items[0 : len(s.items)-1]
	}
}

func (s *SliceSet[A]) Slice() []A {
	return s.items
}

func (s *SliceSet[A]) Len() int {
	return len(s.items)
}
