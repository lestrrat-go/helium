package number

import (
	"math"
	"strconv"
	"strings"
)

// ToString converts a float64 to its XPath string representation,
// matching libxml2's xmlXPathFormatNumber behavior.
//
// libxml2 uses three formatting branches:
//  1. Integers within int32 range: decimal integer format (%d)
//  2. |value| >= 1e9 or |value| < 1e-5: scientific notation with trailing zero stripping
//  3. Otherwise: fixed notation with trailing zero stripping
//
// Both scientific and fixed branches use DBL_DIG (15) significant digits.
func ToString(f float64) string {
	if math.IsNaN(f) {
		return "NaN"
	}
	if math.IsInf(f, 1) {
		return "Infinity"
	}
	if math.IsInf(f, -1) {
		return "-Infinity"
	}
	if f == 0 {
		return "0"
	}

	// Match libxml2: integers within int32 range use %d format.
	if f > math.MinInt32 && f < math.MaxInt32 && f == math.Trunc(f) {
		return strconv.Itoa(int(f))
	}

	abs := math.Abs(f)

	const (
		upperDouble = 1e9
		lowerDouble = 1e-5
		dblDig      = 15
	)

	var s string
	if abs >= upperDouble || abs < lowerDouble {
		// Scientific notation matching libxml2's %*.*e branch.
		s = strconv.FormatFloat(f, 'e', dblDig-1, 64)
	} else {
		// Fixed notation matching libxml2's %0.*f branch.
		intPlace := 1 + int(math.Log10(abs))
		fracPlace := (dblDig - 1) - intPlace
		if fracPlace < 0 {
			fracPlace = 0
		}
		s = strconv.FormatFloat(f, 'f', fracPlace, 64)
	}

	return trimTrailingZeros(s)
}

// trimTrailingZeros strips trailing zeros after the decimal point
// in a formatted number string. Handles both fixed and scientific notation.
// Matches libxml2's post-format zero stripping logic.
func trimTrailingZeros(s string) string {
	eIdx := strings.IndexByte(s, 'e')

	var mantissa, exponent string
	if eIdx >= 0 {
		mantissa = s[:eIdx]
		exponent = s[eIdx:]
	} else {
		mantissa = s
	}

	dotIdx := strings.IndexByte(mantissa, '.')
	if dotIdx < 0 {
		return s
	}

	end := len(mantissa)
	for end > dotIdx+1 && mantissa[end-1] == '0' {
		end--
	}
	if mantissa[end-1] == '.' {
		end--
	}

	return mantissa[:end] + exponent
}
