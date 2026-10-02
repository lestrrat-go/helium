// Package intconv converts decimal strings to int the same way on every
// platform, whatever the size of int.
package intconv

import (
	"errors"
	"strconv"
)

// Atoi parses s as strconv.Atoi does, except that a value that fits int64 but
// not int saturates at math.MaxInt or math.MinInt instead of failing. Only
// where int is 32 bits can such a value occur, so Atoi gives a number for
// every value an int64 holds on every platform, and callers that compare it
// against sizes or limits behave as they do where int is 64 bits. A value
// beyond int64 still returns the strconv.ErrRange error.
func Atoi(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err == nil {
		return n, nil
	}
	if !errors.Is(err, strconv.ErrRange) {
		return 0, err
	}
	if _, err64 := strconv.ParseInt(s, 10, 64); err64 != nil {
		return 0, err64
	}
	// On a range error strconv.Atoi returns the value saturated at
	// math.MaxInt or math.MinInt.
	return n, nil
}
