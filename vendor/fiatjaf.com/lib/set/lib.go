package set

import "golang.org/x/exp/constraints"

type Set[A constraints.Ordered] interface {
	Add(item ...A)
	Remove(item ...A)
	Has(item A) bool
	Intersection(other Set[A]) Set[A]
	Difference(other Set[A]) Set[A]
	Union(other Set[A]) Set[A]
	Slice() []A
	Len() int
}

var (
	_ Set[string] = (*SliceSet[string])(nil)
	_ Set[string] = (*HashSet[string])(nil)
	_ Set[int]    = (*SliceSet[int])(nil)
	_ Set[int]    = (*HashSet[int])(nil)
)
