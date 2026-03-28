package icu

import (
	"fmt"
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
	RepeatGrouping  bool
	FracGroupSizes  []int // from left
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

func parseGroupedPart(part []rune, sep rune, df DecimalFormat) ([]int, int, int, error) {
	if len(part) == 0 {
		return nil, 0, 0, nil
	}

	var groups []int
	currentGroupSize := 0
	minDigits := 0
	maxDigits := 0
	prevSep := false
	for i, r := range part {
		switch {
		case r == sep:
			if i == len(part)-1 || prevSep {
				return nil, 0, 0, fmt.Errorf("invalid grouping separator placement")
			}
			groups = append(groups, currentGroupSize)
			currentGroupSize = 0
			prevSep = true
		case isPatternDigit(r, df):
			if isMandatoryDigit(r, df.ZeroDigit) {
				minDigits++
			}
			maxDigits++
			currentGroupSize++
			prevSep = false
		default:
			return nil, 0, 0, fmt.Errorf("invalid picture character %q", r)
		}
	}
	groups = append(groups, currentGroupSize)
	return groups, minDigits, maxDigits, nil
}

func shouldRepeatGrouping(groups []int) bool {
	if len(groups) < 2 {
		return false
	}
	repeated := groups[1]
	for i := 2; i < len(groups); i++ {
		if groups[i] != repeated {
			return false
		}
	}
	return groups[0] <= repeated
}

func hasInvalidLiteralChars(part []rune, df DecimalFormat) bool {
	for _, r := range part {
		if isPatternDigit(r, df) ||
			r == df.DecimalSeparator ||
			r == df.GroupingSeparator ||
			r == df.PatternSeparator {
			return true
		}
	}
	return false
}

// ParsePicture parses a single picture sub-pattern into its components.
func ParsePicture(pic string, df DecimalFormat) (ParsedPicture, error) {
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
		return pp, nil
	}

	// Validate percent/per-mille: at most one of either, and not both
	percentCount := 0
	perMilleCount := 0
	for _, r := range mantissaRunes {
		switch r {
		case df.Percent:
			percentCount++
		case df.PerMille:
			perMilleCount++
		}
	}
	if percentCount > 1 {
		return ParsedPicture{}, fmt.Errorf("picture contains more than one percent sign")
	}
	if perMilleCount > 1 {
		return ParsedPicture{}, fmt.Errorf("picture contains more than one per-mille sign")
	}
	if percentCount > 0 && perMilleCount > 0 {
		return ParsedPicture{}, fmt.Errorf("picture contains both percent and per-mille signs")
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
			if hasInvalidLiteralChars(expRunes[expEnd:], df) {
				return ParsedPicture{}, fmt.Errorf("invalid active character in picture suffix")
			}
			for _, r := range expRunes[expEnd:] {
				if r == df.Percent || r == df.PerMille {
					return ParsedPicture{}, fmt.Errorf("percent and per-mille are not allowed in exponent pictures")
				}
			}
		}
	} else if numEnd < len(mantissaRunes) {
		pp.Suffix = string(mantissaRunes[numEnd:])
		if hasInvalidLiteralChars(mantissaRunes[numEnd:], df) {
			return ParsedPicture{}, fmt.Errorf("invalid active character in picture suffix")
		}
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
	groups, minIntDigits, maxIntDigits, err := parseGroupedPart(intPart, df.GroupingSeparator, df)
	if err != nil {
		return ParsedPicture{}, err
	}
	pp.MinIntDigits = minIntDigits
	pp.MaxIntDigits = maxIntDigits
	pp.RepeatGrouping = shouldRepeatGrouping(groups)
	// minIntDigits stays 0 if only # digits — allows ".1" instead of "0.1"

	// Convert to grouping sizes (from right)
	// groups[last] is the rightmost (primary) group
	if len(groups) > 1 {
		pp.GroupingSizes = make([]int, len(groups)-1)
		for i := range len(groups) - 1 {
			pp.GroupingSizes[i] = groups[len(groups)-1-i]
		}
	}

	// Parse fractional part
	fracGroups, minFracDigits, maxFracDigits, err := parseGroupedPart(fracPart, df.GroupingSeparator, df)
	if err != nil {
		return ParsedPicture{}, err
	}
	if len(fracPart) > 0 && fracPart[0] == df.GroupingSeparator {
		return ParsedPicture{}, fmt.Errorf("invalid grouping separator placement")
	}
	seenOptionalFracDigit := false
	for _, r := range fracPart {
		switch {
		case r == df.GroupingSeparator:
			continue
		case r == df.Digit:
			seenOptionalFracDigit = true
		case isMandatoryDigit(r, df.ZeroDigit):
			if seenOptionalFracDigit {
				return ParsedPicture{}, fmt.Errorf("mandatory fractional digit after optional digit")
			}
		}
	}
	pp.MinFracDigits = minFracDigits
	pp.MaxFracDigits = maxFracDigits
	if len(fracGroups) > 1 {
		pp.FracGroupSizes = fracGroups
	}
	if pp.MaxIntDigits+pp.MaxFracDigits == 0 {
		return ParsedPicture{}, fmt.Errorf("picture requires at least one digit")
	}

	// Validate: in integer part, mandatory digits must not precede optional digits
	seenMandatory := false
	for _, r := range intPart {
		if r == df.GroupingSeparator {
			continue
		}
		if isMandatoryDigit(r, df.ZeroDigit) {
			seenMandatory = true
		} else if r == df.Digit && seenMandatory {
			return ParsedPicture{}, fmt.Errorf("optional digit after mandatory digit in integer part")
		}
	}

	return pp, nil
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
	if len(parts) > 2 {
		return "", fmt.Errorf("picture %q contains more than one pattern separator", picture)
	}
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
		pp, ppErr := ParsePicture(posPic, df)
		if ppErr != nil {
			return "", ppErr
		}
		return pp.Prefix + df.Infinity + pp.Suffix, nil
	}
	if isNegInf {
		if negPic != "" {
			pp, ppErr := ParsePicture(negPic, df)
			if ppErr != nil {
				return "", ppErr
			}
			return pp.Prefix + df.Infinity + pp.Suffix, nil
		}
		pp, ppErr := ParsePicture(posPic, df)
		if ppErr != nil {
			return "", ppErr
		}
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

	pp, err := ParsePicture(pic, df)
	if err != nil {
		return "", err
	}

	// Apply multiplier to float
	if pp.IsPercent {
		f *= 100
	} else if pp.IsPerMille {
		f *= 1000
	}
	if math.IsInf(f, 0) {
		if negative && negPic == "" {
			return string(df.MinusSign) + pp.Prefix + df.Infinity + pp.Suffix, nil
		}
		return pp.Prefix + df.Infinity + pp.Suffix, nil
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
	intStr := FormatBigInt(bigTrunc, pp.MinIntDigits, pp.GroupingSizes, pp.RepeatGrouping, df)

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
	fracStr = applyFractionGrouping(fracStr, pp.FracGroupSizes, df)
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

	intStr := FormatBigInt(intPart, pp.MinIntDigits, pp.GroupingSizes, pp.RepeatGrouping, df)

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
	fracStr = applyFractionGrouping(fracStr, pp.FracGroupSizes, df)
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
		effMaxFrac := max(pp.MaxFracDigits, pp.MaxIntDigits)
		r.MaxFracDigits = effMaxFrac
		if effMaxFrac > 0 {
			r.HasDecimalPoint = true
		}
		// Show leading zero when pattern had integer digit placeholders
		if pp.MaxIntDigits > 0 {
			r.MinIntDigits = 1
		} else if pp.MaxFracDigits > 0 && r.MinFracDigits == 0 {
			// With pictures like ".#e0", rounding can carry into the integer part;
			// keep one fractional digit in that case.
			r.MinFracDigits = 1
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
	s.WriteString(localizeDigits(digits, df.ZeroDigit))
	return s.String()
}

// FormatBigInt formats an integer with minimum digits and grouping separators.
func FormatBigInt(n *big.Int, minDigits int, groupingSizes []int, repeatGrouping bool, df DecimalFormat) string {
	s := n.String()
	if minDigits == 0 && s == "0" {
		s = ""
	}
	// Pad to minDigits
	for len(s) < minDigits {
		s = "0" + s
	}

	if len(groupingSizes) == 0 || len(s) == 0 {
		return localizeDigits(s, df.ZeroDigit)
	}

	// Insert grouping separators from right
	// groupingSizes[0] is the primary (rightmost) group size.
	var result []rune
	runes := []rune(s)
	groupIdx := 0
	count := 0
	groupSize := groupingSizes[0]

	for i := len(runes) - 1; i >= 0; i-- {
		if groupSize > 0 && count > 0 && count == groupSize {
			result = append(result, df.GroupingSeparator)
			count = 0
			if groupIdx+1 < len(groupingSizes) {
				groupIdx++
				groupSize = groupingSizes[groupIdx]
			} else if !repeatGrouping {
				groupSize = 0
			}
		}
		result = append(result, runes[i])
		count++
	}

	// Reverse
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return localizeDigits(string(result), df.ZeroDigit)
}

// FormatFrac formats the fractional part of a float64.
func FormatFrac(frac float64, minDigits, maxDigits int, df DecimalFormat) string {
	if maxDigits == 0 {
		return ""
	}

	// Build fractional digits
	var digits []byte
	remaining := frac
	for range maxDigits {
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
	return localizeDigits(string(digits[:end]), df.ZeroDigit)
}

// FormatRatFrac formats the fractional part of a *big.Rat.
func FormatRatFrac(frac *big.Rat, minDigits, maxDigits int, df DecimalFormat) string {
	if maxDigits == 0 {
		return ""
	}

	var digits []byte
	ten := big.NewInt(10)
	rem := new(big.Rat).Set(frac)

	for range maxDigits {
		rem.Mul(rem, new(big.Rat).SetInt(ten))
		intPart := new(big.Int).Quo(rem.Num(), rem.Denom())
		digits = append(digits, byte('0'+intPart.Int64()))
		rem.Sub(rem, new(big.Rat).SetInt(intPart))
	}

	end := len(digits)
	for end > minDigits && digits[end-1] == '0' {
		end--
	}
	return localizeDigits(string(digits[:end]), df.ZeroDigit)
}

func localizeDigits(s string, zeroDigit rune) string {
	if s == "" || zeroDigit == '0' {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(zeroDigit + (r - '0'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func applyFractionGrouping(frac string, groups []int, df DecimalFormat) string {
	if len(groups) <= 1 || len(frac) == 0 {
		return frac
	}

	runes := []rune(frac)
	var b strings.Builder
	pos := 0
	for i, groupSize := range groups {
		if pos >= len(runes) {
			break
		}
		end := min(pos+groupSize, len(runes))
		b.WriteString(string(runes[pos:end]))
		pos = end
		if pos < len(runes) && i < len(groups)-1 {
			b.WriteRune(df.GroupingSeparator)
		}
	}
	if pos < len(runes) {
		b.WriteString(string(runes[pos:]))
	}
	return b.String()
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
	// If d != 1 after removing all 2s and 5s, the decimal is non-terminating
	if d.Cmp(big.NewInt(1)) != 0 {
		return math.MaxInt
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
