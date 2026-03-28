// Package sequence provides generic sequence types for ordered collections
// with lazy evaluation support.
package sequence

import "iter"

// Interface is an ordered collection of items.
// Implementations include Slice (slice-backed) and Range (lazy computed).
type Interface[T any] interface {
	// Len returns the number of items in the sequence.
	Len() int
	// Get returns the item at index i (0-based).
	Get(i int) T
	// Items returns a lazy iterator over the items.
	Items() iter.Seq[T]
	// Materialize returns the items as a plain slice.
	Materialize() []T
}

// Slice is a sequence backed by a plain slice.
type Slice[T any] []T

func (s Slice[T]) Len() int         { return len(s) }
func (s Slice[T]) Get(i int) T      { return s[i] }
func (s Slice[T]) Materialize() []T { return []T(s) }
func (s Slice[T]) Items() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, item := range s {
			if !yield(item) {
				return
			}
		}
	}
}

// Range is a lazy sequence of length n where item i is produced by at(i).
type Range[T any] struct {
	n  int
	at func(i int) T
}

// NewRange creates a lazy sequence of n items where each item is computed by at(i).
func NewRange[T any](n int, at func(i int) T) *Range[T] {
	return &Range[T]{n: n, at: at}
}

func (r *Range[T]) Len() int    { return r.n }
func (r *Range[T]) Get(i int) T { return r.at(i) }
func (r *Range[T]) Items() iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := range r.n {
			if !yield(r.at(i)) {
				return
			}
		}
	}
}
func (r *Range[T]) Materialize() []T {
	result := make([]T, r.n)
	for i := range r.n {
		result[i] = r.at(i)
	}
	return result
}

// Len returns the length of a sequence, treating nil as empty.
func Len[T any](s Interface[T]) int {
	if s == nil {
		return 0
	}
	return s.Len()
}

// Items returns an iterator over a sequence, treating nil as empty.
func Items[T any](s Interface[T]) iter.Seq[T] {
	if s == nil {
		return func(func(T) bool) {}
	}
	return s.Items()
}

// Materialize returns the items as a slice, treating nil as nil.
func Materialize[T any](s Interface[T]) []T {
	if s == nil {
		return nil
	}
	return s.Materialize()
}

// Clone returns a shallow copy of a sequence as a Slice.
func Clone[T any](s Interface[T]) Interface[T] {
	if s == nil {
		return nil
	}
	return Slice[T](append([]T(nil), s.Materialize()...))
}
