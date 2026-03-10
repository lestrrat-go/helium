package xpath3

import (
	"math"
	"math/big"
	"strings"
)

type decimalFormat struct {
	decimalSeparator  rune
	groupingSeparator rune
	percent           rune
	perMille          rune
	zeroDigit         rune
	digit             rune
	patternSeparator  rune
	infinity          string
	nan               string
	minusSign         rune
}

func defaultDecimalFormat() decimalFormat {
	return decimalFormat{
		decimalSeparator:  '.',
		groupingSeparator: ',',
		percent:           '%',
		perMille:          '\u2030',
		zeroDigit:         '0',
		digit:             '#',
		patternSeparator:  ';',
		infinity:          "Infinity",
		nan:               "NaN",
		minusSign:         '-',
	}
}

type parsedPicture struct {
	prefix            string
	suffix            string
	minIntDigits      int
	maxFracDigits     int
	minFracDigits     int
	groupingSizes     []int // from right, repeating
	isPercent         bool
	isPerMille        bool
	hasDecimalPoint   bool
}

func parsePicture(pic string, df decimalFormat) parsedPicture {
	var pp parsedPicture
	runes := []rune(pic)

	// Find the start and end of the numeric part
	numStart := -1
	numEnd := -1
	for i, r := range runes {
		if r == df.zeroDigit || r == df.digit || r == df.decimalSeparator || r == df.groupingSeparator {
			if numStart < 0 {
				numStart = i
			}
			numEnd = i + 1
		} else if r == df.percent {
			pp.isPercent = true
			if numStart < 0 {
				continue
			}
			numEnd = i
			break
		} else if r == df.perMille {
			pp.isPerMille = true
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

	pp.prefix = string(runes[:numStart])
	if numEnd < len(runes) {
		pp.suffix = string(runes[numEnd:])
	}

	// Parse the numeric part
	numPart := runes[numStart:numEnd]
	decPos := -1
	for i, r := range numPart {
		if r == df.decimalSeparator {
			decPos = i
			pp.hasDecimalPoint = true
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
	pp.minIntDigits = 0
	// Split integer part by grouping separator
	var groups []int
	currentGroupSize := 0
	for _, r := range intPart {
		if r == df.groupingSeparator {
			groups = append(groups, currentGroupSize)
			currentGroupSize = 0
		} else {
			if r == df.zeroDigit {
				pp.minIntDigits++
			}
			currentGroupSize++
		}
	}
	groups = append(groups, currentGroupSize) // last group
	// minIntDigits stays 0 if only # digits — allows ".1" instead of "0.1"

	// Convert to grouping sizes (from right)
	// groups[last] is the rightmost (primary) group
	if len(groups) > 1 {
		pp.groupingSizes = make([]int, len(groups)-1)
		for i := 0; i < len(groups)-1; i++ {
			pp.groupingSizes[i] = groups[len(groups)-1-i]
		}
	}

	// Parse fractional part
	for _, r := range fracPart {
		if r == df.zeroDigit {
			pp.minFracDigits++
			pp.maxFracDigits++
		} else if r == df.digit {
			pp.maxFracDigits++
		}
	}

	return pp
}

func formatNumber(a AtomicValue, picture string, df decimalFormat) (string, error) {
	// Split picture on pattern separator
	parts := strings.Split(picture, string(df.patternSeparator))
	posPic := parts[0]
	negPic := ""
	if len(parts) > 1 {
		negPic = parts[1]
	}

	f := a.ToFloat64()

	// Handle special values
	if math.IsNaN(f) {
		return df.nan, nil
	}
	if math.IsInf(f, 1) {
		pp := parsePicture(posPic, df)
		return pp.prefix + df.infinity + pp.suffix, nil
	}
	if math.IsInf(f, -1) {
		if negPic != "" {
			pp := parsePicture(negPic, df)
			return pp.prefix + df.infinity + pp.suffix, nil
		}
		pp := parsePicture(posPic, df)
		return string(df.minusSign) + pp.prefix + df.infinity + pp.suffix, nil
	}

	negative := false
	if f < 0 || (f == 0 && math.Signbit(f)) {
		negative = true
		f = math.Abs(f)
	}

	var pic string
	if negative && negPic != "" {
		pic = negPic
	} else {
		pic = posPic
	}

	pp := parsePicture(pic, df)

	// Apply multiplier
	if pp.isPercent {
		f *= 100
	} else if pp.isPerMille {
		f *= 1000
	}

	// Use big.Rat for precise formatting if the source is decimal/integer
	var result string
	if isIntegerDerived(a.TypeName) || a.TypeName == TypeDecimal {
		result = formatDecimalPrecise(a, pp, df)
	} else {
		result = formatFloat(f, pp, df)
	}

	prefix := pp.prefix
	if negative && negPic == "" {
		prefix = string(df.minusSign) + prefix
	}

	return prefix + result + pp.suffix, nil
}

func formatFloat(f float64, pp parsedPicture, df decimalFormat) string {
	// Round to maxFracDigits
	if pp.maxFracDigits >= 0 {
		scale := math.Pow(10, float64(pp.maxFracDigits))
		f = math.Round(f*scale) / scale
	}

	// Split into integer and fractional parts
	intVal := int64(f)
	frac := f - float64(intVal)
	if frac < 0 {
		frac = -frac
	}

	// Format integer part
	intStr := formatBigInt(new(big.Int).SetInt64(intVal), pp.minIntDigits, pp.groupingSizes, df)

	if !pp.hasDecimalPoint && pp.maxFracDigits == 0 {
		return intStr
	}

	// Format fractional part
	fracStr := formatFrac(frac, pp.minFracDigits, pp.maxFracDigits, df)
	if fracStr == "" && !pp.hasDecimalPoint {
		return intStr
	}
	return intStr + string(df.decimalSeparator) + fracStr
}

func formatDecimalPrecise(a AtomicValue, pp parsedPicture, df decimalFormat) string {
	var r *big.Rat
	if isIntegerDerived(a.TypeName) {
		r = new(big.Rat).SetInt(a.BigInt())
	} else {
		r = new(big.Rat).Set(a.BigRat())
	}

	if pp.isPercent {
		r.Mul(r, new(big.Rat).SetInt64(100))
	} else if pp.isPerMille {
		r.Mul(r, new(big.Rat).SetInt64(1000))
	}

	if r.Sign() < 0 {
		r.Neg(r)
	}

	// Round to maxFracDigits
	rounded := ratRoundHalfToEven(r, pp.maxFracDigits)

	// Get integer and fractional parts
	intPart := new(big.Int).Quo(rounded.Num(), rounded.Denom())
	fracRat := new(big.Rat).Sub(rounded, new(big.Rat).SetInt(intPart))

	intStr := formatBigInt(intPart, pp.minIntDigits, pp.groupingSizes, df)

	if !pp.hasDecimalPoint && pp.maxFracDigits == 0 {
		return intStr
	}

	fracStr := formatRatFrac(fracRat, pp.minFracDigits, pp.maxFracDigits, df)
	if fracStr == "" && !pp.hasDecimalPoint {
		return intStr
	}
	return intStr + string(df.decimalSeparator) + fracStr
}

func formatBigInt(n *big.Int, minDigits int, groupingSizes []int, df decimalFormat) string {
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
			result = append(result, df.groupingSeparator)
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

func formatFrac(frac float64, minDigits, maxDigits int, df decimalFormat) string {
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

func formatRatFrac(frac *big.Rat, minDigits, maxDigits int, df decimalFormat) string {
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
