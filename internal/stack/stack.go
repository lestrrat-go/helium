// Package stack provides the slice-backed LIFO stacks the parser keeps for
// its inputs, open elements, and namespace bindings.
//
// Popping never shrinks a stack's backing array. A parse pushes and pops the
// same levels over and over as it walks down into and back out of nested
// elements, so a stack that reached some depth once reaches it again; keeping
// the capacity lets it do so without reallocating. Popped slots are cleared,
// so they do not keep what they referenced reachable.
package stack

// truncate removes the top n entries of s, or every entry when n exceeds its
// length, and clears the removed slots. The capacity of s is unchanged.
func truncate[T any](s []T, n int) []T {
	if n <= 0 {
		return s
	}
	l := len(s)
	if n > l {
		n = l
	}
	clear(s[l-n:])
	return s[:l-n]
}
