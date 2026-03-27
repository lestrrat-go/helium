package xsd

import "github.com/lestrrat-go/helium/internal/xsd/value"

// compareDecimal compares two decimal string values using math/big.Rat.
// Returns -1 if a < b, 0 if a == b, 1 if a > b, or -2 on parse error.
func compareDecimal(a, b string) int {
	return value.CompareDecimal(a, b)
}

// checkMinInclusive compares value >= bound using type-aware comparison.
func checkMinInclusive(v, bound, builtinLocal string) bool {
	cmp, ok := value.Compare(v, bound, builtinLocal)
	if !ok {
		return true // can't compare, don't error
	}
	return cmp >= 0
}

// checkMaxInclusive compares value <= bound using type-aware comparison.
func checkMaxInclusive(v, bound, builtinLocal string) bool {
	cmp, ok := value.Compare(v, bound, builtinLocal)
	if !ok {
		return true
	}
	return cmp <= 0
}

// checkMinExclusive compares value > bound using type-aware comparison.
func checkMinExclusive(v, bound, builtinLocal string) bool {
	cmp, ok := value.Compare(v, bound, builtinLocal)
	if !ok {
		return true // can't compare, don't error
	}
	return cmp > 0
}

// checkMaxExclusive compares value < bound using type-aware comparison.
func checkMaxExclusive(v, bound, builtinLocal string) bool {
	cmp, ok := value.Compare(v, bound, builtinLocal)
	if !ok {
		return true
	}
	return cmp < 0
}
