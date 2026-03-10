package icu

import (
	"math"
	"math/big"
	"strings"
)

// DecimalFormat holds the characters used for number formatting per the
// ICU decimal format specification.
type DecimalFormat struct {
	DecimalSeparator  rune
	GroupingSeparator rune
	Percent           rune
	PerMille          rune
	ZeroDigit         rune
	Digit             rune
	PatternSeparator  rune
	ExponentSeparator rune
	Infinity          string
	NaN               string
	MinusSign         rune
}

// DefaultDecimalFormat returns the standard ICU decimal format with default
// characters.
func DefaultDecimalFormat() DecimalFormat {
	return DecimalFormat{
		DecimalSeparator:  '.',
		GroupingSeparator: ',',
		Percent:           '%',
		PerMille:          '\u2030',
		ZeroDigit:         '0',
		Digit:             '#',
		PatternSeparator:  ';',
		ExponentSeparator: 'e',
		Infinity:          "Infinity",
		NaN:               "NaN",
		MinusSign:         '-',
	}
}

// ParsedPicture holds the parsed components of a number format picture string.
type ParsedPicture struct {
	Prefix          string
	Suffix          string
	MinIntDigits    int
	MaxIntDigits    int
	MaxFracDigits   int
	MinFracDigits   int
	GroupingSizes   []int // from right, repeating
	IsPercent       bool
	IsPerMille      bool
	HasDecimalPoint bool
	HasExponent     bool
	MinExpDigits    int
}

// isMandatoryDigit checks if r is in the range [zeroDigit, zeroDigit+9].
func isMandatoryDigit(r, zeroDigit rune) bool {
	return r >= zeroDigit && r <= zeroDigit+9
}

// isPatternDigit checks if r is a digit placeholder (mandatory or optional).
func isPatternDigit(r rune, df DecimalFormat) bool {
	return isMandatoryDigit(r, df.ZeroDigit) || r == df.Digit
}

// ParsePicture parses a single picture sub-pattern into its components.
func ParsePicture(pic string, df DecimalFormat) ParsedPicture {
	var pp ParsedPicture
	runes := []rune(pic)

	// Check for exponent separator: split into mantissa and exponent parts
	expSep := df.ExponentSeparator
	if expSep == 0 {
		expSep = 'e'
	}
	expIdx := -1
	for i, r := range runes {
		if r == expSep {
			// Check if next char is a digit placeholder to confirm it's an exponent
			if i+1 < len(runes) && isPatternDigit(runes[i+1], df) {
				expIdx = i
				break
			}
		}
	}

	var mantissaRunes, expRunes []rune
	if expIdx >= 0 {
		mantissaRunes = runes[:expIdx]
		expRunes = runes[expIdx+1:] // skip the 'e'
		pp.HasExponent = true
	} else {
		mantissaRunes = runes
	}

	// Find the start and end of the numeric part in mantissa
	numStart := -1
	numEnd := -1
	for i, r := range mantissaRunes {
		if isPatternDigit(r, df) || r == df.DecimalSeparator || r == df.GroupingSeparator {
			if numStart < 0 {
				numStart = i
			}
			numEnd = i + 1
		} else if r == df.Percent {
			pp.IsPercent = true
			if numStart < 0 {
				continue
			}
			numEnd = i
			break
		} else if r == df.PerMille {
			pp.IsPerMille = true
			if numStart < 0 {
				continue
			}
			numEnd = i
			break
		} else if numStart >= 0 {
			numEnd = i
			break
		}
	}

	if numStart < 0 {
		// No numeric characters found
		return pp
	}

	pp.Prefix = string(mantissaRunes[:numStart])
	// Suffix comes after the exponent part (if any) or after the mantissa numeric part
	if pp.HasExponent {
		// Find end of exponent digits
		expEnd := 0
		for i, r := range expRunes {
			if isPatternDigit(r, df) {
				expEnd = i + 1
			} else {
				break
			}
		}
		pp.MinExpDigits = 0
		for _, r := range expRunes[:expEnd] {
			if isMandatoryDigit(r, df.ZeroDigit) {
				pp.MinExpDigits++
			}
		}
		if pp.MinExpDigits == 0 {
			pp.MinExpDigits = 1
		}
		if expEnd < len(expRunes) {
			pp.Suffix = string(expRunes[expEnd:])
		}
	} else if numEnd < len(mantissaRunes) {
		pp.Suffix = string(mantissaRunes[numEnd:])
	}

	// Parse the numeric part
	numPart := mantissaRunes[numStart:numEnd]
	decPos := -1
	for i, r := range numPart {
		if r == df.DecimalSeparator {
			decPos = i
			pp.HasDecimalPoint = true
			break
		}
	}

	var intPart, fracPart []rune
	if decPos >= 0 {
		intPart = numPart[:decPos]
		fracPart = numPart[decPos+1:]
	} else {
		intPart = numPart
	}

	// Parse integer part: walk left-to-right, collect group sizes
	pp.MinIntDigits = 0
	pp.MaxIntDigits = 0
	// Split integer part by grouping separator
	var groups []int
	currentGroupSize := 0
	for _, r := range intPart {
		if r == df.GroupingSeparator {
			groups = append(groups, currentGroupSize)
			currentGroupSize = 0
		} else {
			if isMandatoryDigit(r, df.ZeroDigit) {
				pp.MinIntDigits++
			}
			pp.MaxIntDigits++
			currentGroupSize++
		}
	}
	groups = append(groups, currentGroupSize) // last group
	// minIntDigits stays 0 if only # digits — allows ".1" instead of "0.1"

	// Convert to grouping sizes (from right)
	// groups[last] is the rightmost (primary) group
	if len(groups) > 1 {
		pp.GroupingSizes = make([]int, len(groups)-1)
		for i := 0; i < len(groups)-1; i++ {
			pp.GroupingSizes[i] = groups[len(groups)-1-i]
		}
	}

	// Parse fractional part
	for _, r := range fracPart {
		if isMandatoryDigit(r, df.ZeroDigit) {
			pp.MinFracDigits++
			pp.MaxFracDigits++
		} else if r == df.Digit {
			pp.MaxFracDigits++
		}
	}

	return pp
}

// FormatNumber formats a numeric value according to the given picture string
// and decimal format. The caller provides pre-extracted numeric properties:
//   - f: the float64 value
//   - isNaN, isPosInf, isNegInf: special value flags
//   - negative: true if the value is negative (or negative zero)
//   - precise: if non-nil, use precise decimal formatting via *big.Rat
//   - picture: the format picture string
//   - df: the decimal format specification
func FormatNumber(f float64, isNaN, isPosInf, isNegInf, negative bool, precise *big.Rat, picture string, df DecimalFormat) (string, error) {
	// Split picture on pattern separator
	parts := strings.Split(picture, string(df.PatternSeparator))
	posPic := parts[0]
	negPic := ""
	if len(parts) > 1 {
		negPic = parts[1]
	}

	// Handle special values
	if isNaN {
		return df.NaN, nil
	}
	if isPosInf {
		pp := ParsePicture(posPic, df)
		return pp.Prefix + df.Infinity + pp.Suffix, nil
	}
	if isNegInf {
		if negPic != "" {
			pp := ParsePicture(negPic, df)
			return pp.Prefix + df.Infinity + pp.Suffix, nil
		}
		pp := ParsePicture(posPic, df)
		return string(df.MinusSign) + pp.Prefix + df.Infinity + pp.Suffix, nil
	}

	if negative {
		f = math.Abs(f)
	}

	var pic string
	if negative && negPic != "" {
		pic = negPic
	} else {
		pic = posPic
	}

	pp := ParsePicture(pic, df)

	// Apply multiplier to float
	if pp.IsPercent {
		f *= 100
	} else if pp.IsPerMille {
		f *= 1000
	}

	// Use big.Rat for precise formatting if provided
	var result string
	if pp.HasExponent {
		if precise != nil {
			result = formatExponentPrecise(precise, pp, df)
		} else {
			result = formatExponentFloat(f, pp, df)
		}
	} else if precise != nil {
		result = FormatDecimalPrecise(precise, pp, df)
	} else {
		result = FormatFloat(f, pp, df)
	}

	prefix := pp.Prefix
	if negative && negPic == "" {
		prefix = string(df.MinusSign) + prefix
	}

	return prefix + result + pp.Suffix, nil
}

// FormatFloat formats a float64 value using the parsed picture and decimal format.
func FormatFloat(f float64, pp ParsedPicture, df DecimalFormat) string {
	// Round to maxFracDigits
	if pp.MaxFracDigits >= 0 {
		scale := math.Pow(10, float64(pp.MaxFracDigits))
		f = math.Round(f*scale) / scale
	}

	// Split into integer and fractional parts.
	// Use math.Trunc to avoid int64 overflow for large floats.
	trunc := math.Trunc(f)
	frac := f - trunc
	if frac < 0 {
		frac = -frac
	}

	// Format integer part
	bigTrunc, _ := new(big.Float).SetFloat64(trunc).Int(nil)
	intStr := FormatBigInt(bigTrunc, pp.MinIntDigits, pp.GroupingSizes, df)

	if !pp.HasDecimalPoint && pp.MaxFracDigits == 0 {
		if intStr == "" {
			intStr = "0"
		}
		return intStr
	}

	// Format fractional part
	fracStr := FormatFrac(frac, pp.MinFracDigits, pp.MaxFracDigits, df)

	// XPath F&O: when both integer and fractional parts are empty (value rounds
	// to zero with MinIntDigits=0 and MinFracDigits=0), show at least one digit.
	if intStr == "" && fracStr == "" {
		if pp.MaxFracDigits > 0 {
			fracStr = "0"
		} else {
			intStr = "0"
		}
	}

	if fracStr == "" {
		return intStr
	}
	return intStr + string(df.DecimalSeparator) + fracStr
}

// FormatDecimalPrecise formats a *big.Rat value with exact decimal arithmetic.
// The rat value should already have its sign removed (absolute value).
func FormatDecimalPrecise(r *big.Rat, pp ParsedPicture, df DecimalFormat) string {
	r = new(big.Rat).Set(r)

	if pp.IsPercent {
		r.Mul(r, new(big.Rat).SetInt64(100))
	} else if pp.IsPerMille {
		r.Mul(r, new(big.Rat).SetInt64(1000))
	}

	if r.Sign() < 0 {
		r.Neg(r)
	}

	// Round to maxFracDigits
	rounded := RatRoundHalfToEven(r, pp.MaxFracDigits)

	// Get integer and fractional parts
	intPart := new(big.Int).Quo(rounded.Num(), rounded.Denom())
	fracRat := new(big.Rat).Sub(rounded, new(big.Rat).SetInt(intPart))

	intStr := FormatBigInt(intPart, pp.MinIntDigits, pp.GroupingSizes, df)

	if !pp.HasDecimalPoint && pp.MaxFracDigits == 0 {
		if intStr == "" {
			intStr = "0"
		}
		return intStr
	}

	fracStr := FormatRatFrac(fracRat, pp.MinFracDigits, pp.MaxFracDigits, df)

	// XPath F&O: when both integer and fractional parts are empty (value rounds
	// to zero with MinIntDigits=0 and MinFracDigits=0), show at least one digit.
	if intStr == "" && fracStr == "" {
		if pp.MaxFracDigits > 0 {
			fracStr = "0"
		} else {
			intStr = "0"
		}
	}

	if fracStr == "" {
		return intStr
	}
	return intStr + string(df.DecimalSeparator) + fracStr
}

// exponentScalingFactor computes the number of integer digits in the mantissa
// for scientific notation formatting, per the XPath F&O spec:
//   - MinIntDigits > 0: scaling factor = MinIntDigits
//   - MinIntDigits = 0, MaxIntDigits >= 2: scaling factor = MaxIntDigits
//   - Otherwise: scaling factor = 0 (mantissa in [0.1, 1))
func exponentScalingFactor(pp ParsedPicture) int {
	if pp.MinIntDigits > 0 {
		return pp.MinIntDigits
	}
	if pp.MaxIntDigits >= 2 {
		return pp.MaxIntDigits
	}
	return 0
}

// adjustMantissaPP adjusts the ParsedPicture for mantissa formatting in
// scientific notation. When scaling factor m=0, unused integer digit slots
// effectively become optional fractional digits.
func adjustMantissaPP(pp ParsedPicture, m int) ParsedPicture {
	r := pp
	r.HasExponent = false
	r.IsPercent = false
	r.IsPerMille = false

	if m == 0 {
		// Integer # slots become optional fractional digits
		effMaxFrac := pp.MaxFracDigits
		if pp.MaxIntDigits > effMaxFrac {
			effMaxFrac = pp.MaxIntDigits
		}
		r.MaxFracDigits = effMaxFrac
		if effMaxFrac > 0 {
			r.HasDecimalPoint = true
		}
		// Show leading zero when pattern had integer digit placeholders
		if pp.MaxIntDigits > 0 {
			r.MinIntDigits = 1
		}
	}

	return r
}

// formatExponentFloat formats a float64 in scientific notation.
func formatExponentFloat(f float64, pp ParsedPicture, df DecimalFormat) string {
	if f == 0 {
		return formatExponentZero(pp, df)
	}

	m := exponentScalingFactor(pp)

	// Compute exponent: normalize so mantissa has m integer digits
	exp := 0
	abs := math.Abs(f)
	if m == 0 {
		// Normalize so mantissa is in [0.1, 1) — first significant digit
		// is the first fractional digit
		for abs >= 1 {
			abs /= 10
			exp++
		}
		for abs > 0 && abs < 0.1 {
			abs *= 10
			exp--
		}
	} else {
		upperBound := math.Pow(10, float64(m))
		lowerBound := math.Pow(10, float64(m-1))
		for abs >= upperBound {
			abs /= 10
			exp++
		}
		for abs < lowerBound {
			abs *= 10
			exp--
		}
	}

	mantissaPP := adjustMantissaPP(pp, m)
	mantissaStr := FormatFloat(abs, mantissaPP, df)

	// Strip trailing decimal separator when there are no fractional digits
	mantissaStr = strings.TrimSuffix(mantissaStr, string(df.DecimalSeparator))

	// Format the exponent
	expStr := formatExponentDigits(exp, pp.MinExpDigits, df)

	expSep := df.ExponentSeparator
	if expSep == 0 {
		expSep = 'e'
	}

	return mantissaStr + string(expSep) + expStr
}

// formatExponentPrecise formats a *big.Rat in scientific notation.
func formatExponentPrecise(r *big.Rat, pp ParsedPicture, df DecimalFormat) string {
	if r.Sign() == 0 {
		return formatExponentZero(pp, df)
	}

	if pp.IsPercent {
		r = new(big.Rat).Mul(r, new(big.Rat).SetInt64(100))
	} else if pp.IsPerMille {
		r = new(big.Rat).Mul(r, new(big.Rat).SetInt64(1000))
	}

	m := exponentScalingFactor(pp)

	// Compute exponent by normalizing the value
	exp := 0
	abs := new(big.Rat).Abs(r)
	ten := new(big.Rat).SetInt64(10)
	one := new(big.Rat).SetInt64(1)

	tenth := new(big.Rat).SetFrac64(1, 10)

	if m == 0 {
		// Normalize so mantissa is in [0.1, 1)
		for abs.Cmp(one) >= 0 {
			abs.Quo(abs, ten)
			exp++
		}
		for abs.Sign() > 0 && abs.Cmp(tenth) < 0 {
			abs.Mul(abs, ten)
			exp--
		}
	} else {
		threshold := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(m)), nil))
		lowerBound := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(m-1)), nil))
		if m == 1 {
			lowerBound = one
		}
		for abs.Cmp(lowerBound) < 0 {
			abs.Mul(abs, ten)
			exp--
		}
		for abs.Cmp(threshold) >= 0 {
			abs.Quo(abs, ten)
			exp++
		}
	}

	// Format the mantissa
	mantissaPP := adjustMantissaPP(pp, m)
	mantissaStr := FormatDecimalPrecise(abs, mantissaPP, df)

	// Strip trailing decimal separator when there are no fractional digits
	mantissaStr = strings.TrimSuffix(mantissaStr, string(df.DecimalSeparator))

	// Format the exponent
	expStr := formatExponentDigits(exp, pp.MinExpDigits, df)

	expSep := df.ExponentSeparator
	if expSep == 0 {
		expSep = 'e'
	}

	return mantissaStr + string(expSep) + expStr
}

func formatExponentZero(pp ParsedPicture, df DecimalFormat) string {
	mantissaPP := adjustMantissaPP(pp, exponentScalingFactor(pp))
	mantissaStr := FormatFloat(0, mantissaPP, df)

	// Strip trailing decimal separator
	mantissaStr = strings.TrimSuffix(mantissaStr, string(df.DecimalSeparator))

	expStr := formatExponentDigits(0, pp.MinExpDigits, df)

	expSep := df.ExponentSeparator
	if expSep == 0 {
		expSep = 'e'
	}

	return mantissaStr + string(expSep) + expStr
}

func formatExponentDigits(exp, minDigits int, df DecimalFormat) string {
	neg := exp < 0
	if neg {
		exp = -exp
	}
	s := strings.Builder{}
	if neg {
		s.WriteRune(df.MinusSign)
	}
	digits := big.NewInt(int64(exp)).String()
	for len(digits) < minDigits {
		digits = "0" + digits
	}
	s.WriteString(digits)
	return s.String()
}

// FormatBigInt formats an integer with minimum digits and grouping separators.
func FormatBigInt(n *big.Int, minDigits int, groupingSizes []int, df DecimalFormat) string {
	s := n.String()
	if minDigits == 0 && s == "0" {
		s = ""
	}
	// Pad to minDigits
	for len(s) < minDigits {
		s = "0" + s
	}

	if len(groupingSizes) == 0 || len(s) == 0 {
		return s
	}

	// Insert grouping separators from right
	// groupingSizes[0] is the primary (rightmost) group size
	// groupingSizes[last] repeats for all subsequent groups
	var result []rune
	runes := []rune(s)
	groupIdx := 0
	count := 0
	groupSize := groupingSizes[0]

	for i := len(runes) - 1; i >= 0; i-- {
		if count > 0 && count == groupSize {
			result = append(result, df.GroupingSeparator)
			count = 0
			if groupIdx+1 < len(groupingSizes) {
				groupIdx++
			}
			groupSize = groupingSizes[groupIdx]
		}
		result = append(result, runes[i])
		count++
	}

	// Reverse
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return string(result)
}

// FormatFrac formats the fractional part of a float64.
func FormatFrac(frac float64, minDigits, maxDigits int, df DecimalFormat) string {
	_ = df
	if maxDigits == 0 {
		return ""
	}

	// Build fractional digits
	var digits []byte
	remaining := frac
	for i := 0; i < maxDigits; i++ {
		remaining *= 10
		d := int(remaining)
		digits = append(digits, byte('0'+d))
		remaining -= float64(d)
	}

	// Trim trailing zeros beyond minDigits
	end := len(digits)
	for end > minDigits && digits[end-1] == '0' {
		end--
	}
	return string(digits[:end])
}

// FormatRatFrac formats the fractional part of a *big.Rat.
func FormatRatFrac(frac *big.Rat, minDigits, maxDigits int, df DecimalFormat) string {
	_ = df
	if maxDigits == 0 {
		return ""
	}

	var digits []byte
	ten := big.NewInt(10)
	rem := new(big.Rat).Set(frac)

	for i := 0; i < maxDigits; i++ {
		rem.Mul(rem, new(big.Rat).SetInt(ten))
		intPart := new(big.Int).Quo(rem.Num(), rem.Denom())
		digits = append(digits, byte('0'+intPart.Int64()))
		rem.Sub(rem, new(big.Rat).SetInt(intPart))
	}

	end := len(digits)
	for end > minDigits && digits[end-1] == '0' {
		end--
	}
	return string(digits[:end])
}

// RatRoundHalfToEven rounds a *big.Rat to the given precision using
// half-to-even rounding.
func RatRoundHalfToEven(r *big.Rat, precision int) *big.Rat {
	if precision < 0 {
		// Guard against absurdly large negative precision
		if -precision > 1000 {
			return new(big.Rat) // rounds to zero
		}
		// Round to 10^(-precision) — convert to integer, round, convert back
		scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-precision)), nil)
		scaleRat := new(big.Rat).SetInt(scale)
		divided := new(big.Rat).Quo(r, scaleRat)
		rounded := RatRoundHalfToEvenInt(divided)
		return new(big.Rat).Mul(new(big.Rat).SetInt(rounded), scaleRat)
	}
	// If precision is very large and already exceeds the denominator's decimal
	// digits, the value is already exact — return as-is. This avoids computing
	// astronomically large powers of 10 (e.g. 10^4294967296).
	if precision > 1000 || RatDecimalDigits(r) <= precision {
		return new(big.Rat).Set(r)
	}
	// Multiply by 10^precision, round to integer, divide back
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(precision)), nil)
	scaleRat := new(big.Rat).SetInt(scale)
	shifted := new(big.Rat).Mul(r, scaleRat)
	rounded := RatRoundHalfToEvenInt(shifted)
	return new(big.Rat).SetFrac(rounded, new(big.Int).Set(scale))
}

// RatDecimalDigits returns the number of decimal digits needed to represent
// the fractional part of r exactly. Returns 0 for integers.
func RatDecimalDigits(r *big.Rat) int {
	if r.IsInt() {
		return 0
	}
	// Factor out 2s and 5s from denominator; count max
	d := new(big.Int).Set(r.Denom())
	twos, fives := 0, 0
	for d.Bit(0) == 0 {
		d.Rsh(d, 1)
		twos++
	}
	five := big.NewInt(5)
	mod := new(big.Int)
	for {
		d.QuoRem(d, five, mod)
		if mod.Sign() != 0 {
			break
		}
		fives++
	}
	if twos > fives {
		return twos
	}
	return fives
}

// RatRoundHalfToEvenInt rounds a *big.Rat to the nearest integer using
// half-to-even rounding.
func RatRoundHalfToEvenInt(r *big.Rat) *big.Int {
	if r.IsInt() {
		return new(big.Int).Set(r.Num())
	}
	// Get integer part (truncated toward zero)
	intPart := new(big.Int).Quo(r.Num(), r.Denom())
	// Fractional remainder
	rem := new(big.Rat).Sub(r, new(big.Rat).SetInt(intPart))
	rem.Abs(rem)

	half := new(big.Rat).SetFrac64(1, 2)
	cmp := rem.Cmp(half)

	if cmp < 0 {
		// Closer to floor
		return intPart
	}
	if cmp > 0 {
		// Closer to ceil
		if r.Sign() > 0 {
			return intPart.Add(intPart, big.NewInt(1))
		}
		return intPart.Sub(intPart, big.NewInt(1))
	}
	// Exactly half — round to even
	if new(big.Int).And(intPart, big.NewInt(1)).Sign() == 0 {
		return intPart // already even
	}
	if r.Sign() > 0 {
		return intPart.Add(intPart, big.NewInt(1))
	}
	return intPart.Sub(intPart, big.NewInt(1))
}
