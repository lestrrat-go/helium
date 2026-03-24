package xpath3

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"unicode"
	"unicode/utf8"
)

func fnFormatInteger(ctx context.Context, args []Sequence) (Sequence, error) {
	// format-integer($value, $picture [, $lang])
	valSeq := args[0]
	picSeq := args[1]

	// Empty sequence → empty string
	if seqLen(valSeq) == 0 {
		return SingleString(""), nil
	}

	valAtom, err := AtomizeItem(valSeq.Get(0))
	if err != nil {
		return nil, err
	}
	if valAtom.TypeName == TypeUntypedAtomic || valAtom.TypeName == TypeString {
		valAtom, err = CastAtomic(valAtom, TypeInteger)
		if err != nil {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("cannot cast %s to xs:integer", valAtom.TypeName)}
		}
	}
	if !isSubtypeOf(valAtom.TypeName, TypeInteger) {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("format-integer: expected xs:integer, got %s", valAtom.TypeName)}
	}
	n, ok := valAtom.Value.(*big.Int)
	if !ok {
		return nil, fmt.Errorf("xpath3: internal error: expected *big.Int for %s", valAtom.TypeName)
	}

	picAtom, err := AtomizeItem(picSeq.Get(0))
	if err != nil {
		return nil, err
	}
	picture := picAtom.StringVal()

	lang := "en"
	if ec := getFnContext(ctx); ec != nil {
		lang = ec.getDefaultLanguage()
	}
	if len(args) > 2 && seqLen(args[2]) > 0 {
		langAtom, err := AtomizeItem(args[2].Get(0))
		if err != nil {
			return nil, err
		}
		lang = langAtom.StringVal()
	}

	result, err := formatIntegerPicture(n, picture, lang)
	if err != nil {
		return nil, err
	}
	return SingleString(result), nil
}

func formatIntegerPicture(n *big.Int, picture, lang string) (string, error) {
	if picture == "" {
		return "", &XPathError{Code: errCodeFODF1310, Message: "empty picture string"}
	}

	// Split on ';' to get primary token and format modifier
	primary, modifier := splitFormatModifier(picture)
	if primary == "" {
		return "", &XPathError{Code: errCodeFODF1310, Message: "empty primary format token"}
	}
	if modifier == "" && hasImplicitFormatModifier(primary) {
		return "", &XPathError{Code: errCodeFODF1310, Message: fmt.Sprintf("invalid picture: %q", picture)}
	}

	if err := validateModifier(modifier); err != nil {
		return "", err
	}
	ordinal, ordSuffix := parseModifier(modifier)

	// Determine formatting based on primary token
	negative := n.Sign() < 0
	absN := new(big.Int).Abs(n)

	var result string
	var fmtErr error

	switch primary {
	case "A":
		result = formatAlpha(absN, true)
	case "a":
		result = formatAlpha(absN, false)
	case "I":
		result = formatRoman(absN, true)
	case "i":
		result = formatRoman(absN, false)
	case "W":
		result = formatWords(absN, lang, "upper")
		if ordinal {
			result = applyOrdinalWords(result, lang, "upper", ordSuffix)
		}
	case "w":
		result = formatWords(absN, lang, "lower")
		if ordinal {
			result = applyOrdinalWords(result, lang, "lower", ordSuffix)
		}
	case "Ww":
		result = formatWords(absN, lang, "title")
		if ordinal {
			result = applyOrdinalWords(result, lang, "title", ordSuffix)
		}
	default:
		// Check for special single-character format tokens
		if tok := classifyFormatToken(primary); tok != fmtTokenDecimal {
			result, fmtErr = formatSpecialToken(absN, primary, tok)
			if fmtErr != nil {
				return "", fmtErr
			}
			if ordinal {
				result = applyOrdinalDecimal(result, absN, lang, ordSuffix)
			}
		} else {
			result, fmtErr = formatIntegerDecimal(absN, primary)
			if fmtErr != nil {
				return "", fmtErr
			}
			if ordinal {
				result = applyOrdinalDecimal(result, absN, lang, ordSuffix)
			}
		}
	}

	if negative && n.Sign() != 0 {
		result = "-" + result
	}

	return result, nil
}

type formatTokenKind int

const (
	fmtTokenDecimal formatTokenKind = iota
	fmtTokenGreekLower
	fmtTokenGreekUpper
	fmtTokenCJK
	fmtTokenCircled
	fmtTokenParenthesized
	fmtTokenFullstopDigit
	fmtTokenUnknown
)

func classifyFormatToken(primary string) formatTokenKind {
	runes := []rune(primary)
	if len(runes) == 0 {
		return fmtTokenDecimal
	}

	// Mixed decimal-digit pictures with ASCII letters fall back to the default
	// decimal token rather than being validated as decimal-digit patterns.
	hasDecimalPattern := false
	hasASCIILetter := false
	for _, r := range runes {
		if r == '#' {
			hasDecimalPattern = true
			continue
		}
		if r >= '0' && r <= '9' {
			hasDecimalPattern = true
			continue
		}
		// Check for Unicode decimal digits (Arabic-Indic, Devanagari, etc.)
		if unicode.IsDigit(r) {
			hasDecimalPattern = true
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			hasASCIILetter = true
		}
	}
	if hasDecimalPattern {
		if hasASCIILetter {
			return fmtTokenUnknown
		}
		return fmtTokenDecimal
	}

	// Single character tokens
	if len(runes) == 1 {
		r := runes[0]
		// Greek lowercase: α (U+03B1) to ω
		if r >= 0x03B1 && r <= 0x03C9 {
			return fmtTokenGreekLower
		}
		// Greek uppercase: Α (U+0391) to Ω
		if r >= 0x0391 && r <= 0x03A9 {
			return fmtTokenGreekUpper
		}
		// CJK ideographic digits: 一 二 三...
		if r == '一' || r == '二' || r == '三' || r == '四' || r == '五' ||
			r == '六' || r == '七' || r == '八' || r == '九' || r == '十' {
			return fmtTokenCJK
		}
		// Circled digits: ① (U+2460) to ⑳ (U+2473)
		if r >= 0x2460 && r <= 0x2473 {
			return fmtTokenCircled
		}
		// Parenthesized digits: ⑴ (U+2474) to ⒇ (U+2487)
		if r >= 0x2474 && r <= 0x2487 {
			return fmtTokenParenthesized
		}
		// Full-stop digits: ⒈ (U+2488) to ⒛ (U+249B)
		if r >= 0x2488 && r <= 0x249B {
			return fmtTokenFullstopDigit
		}
	}

	return fmtTokenUnknown
}

func formatSpecialToken(n *big.Int, primary string, kind formatTokenKind) (string, error) {
	switch kind {
	case fmtTokenGreekLower:
		return formatAlphaRange(n, 'α', 24), nil
	case fmtTokenGreekUpper:
		return formatAlphaRange(n, 'Α', 24), nil
	case fmtTokenCJK:
		return formatCJK(n), nil
	case fmtTokenCircled:
		return formatSequentialSymbol(n, 0x2460, 20), nil
	case fmtTokenParenthesized:
		return formatSequentialSymbol(n, 0x2474, 20), nil
	case fmtTokenFullstopDigit:
		return formatSequentialSymbol(n, 0x2488, 20), nil
	case fmtTokenUnknown:
		// Spec: if the format token is not recognized, use decimal fallback
		return formatIntegerDecimal(n, "1")
	default:
		return formatIntegerDecimal(n, primary)
	}
}

// formatAlphaRange formats using an alphabetic numbering system starting at base.
func formatAlphaRange(n *big.Int, base rune, size int) string {
	if n.Sign() == 0 {
		return "0"
	}
	val := new(big.Int).Set(n)
	one := big.NewInt(1)
	sz := big.NewInt(int64(size))

	var digits []rune
	for val.Sign() > 0 {
		val.Sub(val, one)
		mod := new(big.Int)
		val.DivMod(val, sz, mod)
		digits = append([]rune{base + rune(mod.Int64())}, digits...)
	}
	return string(digits)
}

// formatSequentialSymbol formats using a sequential Unicode symbol range (e.g., circled digits).
func formatSequentialSymbol(n *big.Int, startCodepoint rune, maxVal int) string {
	if !n.IsInt64() || n.Int64() < 1 || n.Int64() > int64(maxVal) {
		// Fall back to decimal for out-of-range values
		return n.String()
	}
	return string(startCodepoint + rune(n.Int64()-1))
}

// CJK ideographic number formatting
var cjkDigits = []string{"〇", "一", "二", "三", "四", "五", "六", "七", "八", "九"}

func formatCJK(n *big.Int) string {
	if n.Sign() == 0 {
		return "〇"
	}
	if !n.IsInt64() {
		return n.String()
	}
	return int64ToCJK(n.Int64())
}

func int64ToCJK(n int64) string {
	if n == 0 {
		return "〇"
	}
	if n < 0 {
		return "負" + int64ToCJK(-n)
	}

	var parts []string

	type cjkUnit struct {
		value int64
		char  string
	}
	units := []cjkUnit{
		{100000000, "億"},
		{10000, "万"},
		{1000, "千"},
		{100, "百"},
		{10, "十"},
	}

	for _, u := range units {
		if n >= u.value {
			q := n / u.value
			if q > 1 {
				parts = append(parts, cjkDigits[q])
			}
			parts = append(parts, u.char)
			n %= u.value
		}
	}

	if n > 0 {
		parts = append(parts, cjkDigits[n])
	}

	return strings.Join(parts, "")
}

func splitFormatModifier(picture string) (string, string) {
	idx := strings.LastIndex(picture, ";")
	if idx < 0 {
		return picture, ""
	}
	return picture[:idx], picture[idx+1:]
}

func hasImplicitFormatModifier(primary string) bool {
	if !containsDecimalPattern(primary) {
		return false
	}
	for i, r := range primary {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return validateModifier(primary[i:]) == nil
		}
	}
	return false
}

func containsDecimalPattern(primary string) bool {
	for _, r := range primary {
		if r == '#' || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func parseModifier(mod string) (bool, string) {
	if mod == "" {
		return false, ""
	}

	ordinal := false
	suffix := ""

	for i := 0; i < len(mod); i++ {
		switch mod[i] {
		case 'o':
			ordinal = true
		case 't', 'c':
			// traditional or cardinal — no special handling
		case '(':
			end := strings.Index(mod[i:], ")")
			if end > 0 {
				suffix = mod[i+1 : i+end]
				i += end
			}
		}
	}

	return ordinal, suffix
}

func validateModifier(mod string) error {
	if mod == "" {
		return nil
	}
	// Valid modifier: [co][t]?  or [co]?[t]  followed by optional (...)
	// Characters allowed: c, o, t, and a single parenthesized group
	i := 0
	for i < len(mod) {
		switch mod[i] {
		case 'c', 'o', 't':
			i++
		case '(':
			end := strings.Index(mod[i:], ")")
			if end < 0 {
				return &XPathError{Code: errCodeFODF1310, Message: fmt.Sprintf("unclosed parenthesis in format modifier: %q", mod)}
			}
			i += end + 1
		default:
			return &XPathError{Code: errCodeFODF1310, Message: fmt.Sprintf("invalid character in format modifier: %q", mod)}
		}
	}
	return nil
}

// formatAlpha formats a number using alphabetic numbering (a=1, b=2, ..., z=26, aa=27, ...).
func formatAlpha(n *big.Int, upper bool) string {
	if n.Sign() == 0 {
		return "0"
	}

	val := new(big.Int).Set(n)
	one := big.NewInt(1)
	twentySix := big.NewInt(26)

	var digits []rune
	base := 'a'
	if upper {
		base = 'A'
	}

	for val.Sign() > 0 {
		val.Sub(val, one)
		mod := new(big.Int)
		val.DivMod(val, twentySix, mod)
		digits = append([]rune{base + rune(mod.Int64())}, digits...)
	}

	return string(digits)
}

// formatRoman formats a number as Roman numerals.
func formatRoman(n *big.Int, upper bool) string {
	if n.Sign() == 0 {
		return "0"
	}

	val := n.Int64()
	if val < 0 {
		val = -val
	}
	if val > 10000 {
		return n.String()
	}

	type romanPair struct {
		value  int64
		symbol string
	}
	table := []romanPair{
		{1000, "m"}, {900, "cm"}, {500, "d"}, {400, "cd"},
		{100, "c"}, {90, "xc"}, {50, "l"}, {40, "xl"},
		{10, "x"}, {9, "ix"}, {5, "v"}, {4, "iv"}, {1, "i"},
	}

	var b strings.Builder
	for _, pair := range table {
		for val >= pair.value {
			b.WriteString(pair.symbol)
			val -= pair.value
		}
	}

	result := b.String()
	if upper {
		result = strings.ToUpper(result)
	}
	return result
}

// formatWords formats a number as words.
func formatWords(n *big.Int, lang, caseStyle string) string {
	if lang != "en" && lang != "" {
		return formatWordsLang(n, lang, caseStyle)
	}
	words := intToEnglishWords(n)
	return applyCase(words, caseStyle)
}

func applyCase(s, caseStyle string) string {
	switch caseStyle {
	case "upper":
		return strings.ToUpper(s)
	case "title":
		return toTitleCase(s)
	default:
		return strings.ToLower(s)
	}
}

func toTitleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			r, size := utf8.DecodeRuneInString(w)
			words[i] = string(unicode.ToUpper(r)) + strings.ToLower(w[size:])
		}
	}
	return strings.Join(words, " ")
}

var onesWords = []string{
	"", "one", "two", "three", "four", "five",
	"six", "seven", "eight", "nine", "ten",
	"eleven", "twelve", "thirteen", "fourteen", "fifteen",
	"sixteen", "seventeen", "eighteen", "nineteen",
}

var tensWords = []string{
	"", "", "twenty", "thirty", "forty", "fifty",
	"sixty", "seventy", "eighty", "ninety",
}

func intToEnglishWords(n *big.Int) string {
	if n.Sign() == 0 {
		return "zero"
	}
	if n.Sign() < 0 {
		return "minus " + intToEnglishWords(new(big.Int).Neg(n))
	}
	if n.IsInt64() {
		return int64ToEnglishWords(n.Int64())
	}
	return n.String()
}

func int64ToEnglishWords(n int64) string {
	if n == 0 {
		return "zero"
	}
	if n < 0 {
		return "minus " + int64ToEnglishWords(-n)
	}

	var parts []string

	type unit struct {
		val  int64
		name string
	}
	units := []unit{
		{1000000000000, "trillion"},
		{1000000000, "billion"},
		{1000000, "million"},
		{1000, "thousand"},
	}

	for _, u := range units {
		if n >= u.val {
			parts = append(parts, int64ToEnglishWords(n/u.val)+" "+u.name)
			n %= u.val
		}
	}

	if n >= 100 {
		parts = append(parts, onesWords[n/100]+" hundred")
		n %= 100
	}

	if n > 0 {
		if len(parts) > 0 {
			parts = append(parts, "and")
		}
		if n < 20 {
			parts = append(parts, onesWords[n])
		} else {
			s := tensWords[n/10]
			if n%10 > 0 {
				s += " " + onesWords[n%10]
			}
			parts = append(parts, s)
		}
	}

	return strings.Join(parts, " ")
}

func applyOrdinalWords(words, lang, caseStyle, suffix string) string {
	if lang == "de" {
		return applyOrdinalWordsDe(words, caseStyle, suffix)
	}
	if lang == "fr" {
		return applyOrdinalWordsFr(words, caseStyle, suffix)
	}
	if lang == "it" {
		return applyOrdinalWordsIt(words, caseStyle, suffix)
	}

	result := englishOrdinalWord(words, caseStyle)
	return result
}

func englishOrdinalWord(words, caseStyle string) string {
	parts := strings.Fields(words)
	if len(parts) == 0 {
		return words
	}

	last := strings.ToLower(parts[len(parts)-1])
	ordinal := englishWordToOrdinal(last)

	parts[len(parts)-1] = ordinal
	result := strings.Join(parts, " ")
	return applyCase(result, caseStyle)
}

func englishWordToOrdinal(word string) string {
	irregulars := map[string]string{
		"one":    "first",
		"two":    "second",
		"three":  "third",
		"four":   "fourth",
		"five":   "fifth",
		"six":    "sixth",
		"seven":  "seventh",
		"eight":  "eighth",
		"nine":   "ninth",
		"ten":    "tenth",
		"eleven": "eleventh",
		"twelve": "twelfth",
	}
	if ord, ok := irregulars[word]; ok {
		return ord
	}
	if strings.HasSuffix(word, "y") {
		return word[:len(word)-1] + "ieth"
	}
	return word + "th"
}

func applyOrdinalDecimal(result string, n *big.Int, lang, suffix string) string {
	// The parenthesized part like (-en) is a language hint for ordinals,
	// NOT a literal suffix. We still use language-appropriate ordinal suffixes.
	_ = suffix // language hint, not literal

	if lang != "en" && lang != "" {
		// For non-English, just append the number — implementation-defined
		return result
	}

	// English ordinal suffixes
	val := new(big.Int).Abs(n)
	mod100 := new(big.Int).Mod(val, big.NewInt(100)).Int64()
	mod10 := new(big.Int).Mod(val, big.NewInt(10)).Int64()

	if mod100 >= 11 && mod100 <= 13 {
		return result + "th"
	}
	switch mod10 {
	case 1:
		return result + "st"
	case 2:
		return result + "nd"
	case 3:
		return result + "rd"
	default:
		return result + "th"
	}
}

// formatIntegerDecimal formats a number using decimal digits with optional
// grouping separators and zero-padding based on the picture string.
func formatIntegerDecimal(n *big.Int, picture string) (string, error) {
	runes := []rune(picture)

	// Detect the zero digit from the picture
	zeroDigit := rune('0')
	for _, r := range runes {
		if r != '#' && unicode.IsDigit(r) {
			zeroDigit = unicodeDigitZero(r)
			break
		}
	}

	// Parse the picture into digit characters and grouping separators
	type picElement struct {
		isDigit bool
		isMand  bool // mandatory (0) vs optional (#)
		sep     rune // grouping separator character (if !isDigit)
	}

	var elements []picElement
	for _, r := range runes {
		if r == '#' {
			elements = append(elements, picElement{isDigit: true, isMand: false})
		} else if isDecimalDigitInRange(r, zeroDigit) {
			elements = append(elements, picElement{isDigit: true, isMand: true})
		} else {
			elements = append(elements, picElement{isDigit: false, sep: r})
		}
	}

	// Extract group sizes and separator characters from right to left
	// Walk the elements to build groups
	var groups []int
	var groupSeps []rune // separator character for each group boundary
	currentSize := 0
	minDigits := 0

	// First, count total digits and check structure
	var digitElements []picElement
	for _, e := range elements {
		if e.isDigit {
			digitElements = append(digitElements, e)
			if e.isMand {
				minDigits++
			}
		}
	}

	// Walk elements to build groups (left to right)
	currentSize = 0
	for _, e := range elements {
		if e.isDigit {
			currentSize++
		} else {
			// This is a separator
			if currentSize == 0 && len(groups) == 0 {
				// Leading separator — error
				return "", &XPathError{Code: errCodeFODF1310, Message: fmt.Sprintf("invalid picture: %q", picture)}
			}
			groups = append(groups, currentSize)
			groupSeps = append(groupSeps, e.sep)
			currentSize = 0
		}
	}
	groups = append(groups, currentSize) // last group

	// Validate: no empty groups (adjacent separators or trailing separator)
	for _, g := range groups {
		if g == 0 {
			return "", &XPathError{Code: errCodeFODF1310, Message: fmt.Sprintf("invalid picture: %q", picture)}
		}
	}

	// Validate: # cannot appear after 0 (in left-to-right order)
	seenMandatory := false
	for _, e := range digitElements {
		if e.isMand {
			seenMandatory = true
		} else if seenMandatory {
			return "", &XPathError{Code: errCodeFODF1310, Message: fmt.Sprintf("invalid picture: '#' after '0' in %q", picture)}
		}
	}

	if minDigits == 0 {
		minDigits = 1
	}

	// Format the number
	s := n.String()
	for len(s) < minDigits {
		s = "0" + s
	}

	// Apply grouping separators
	numSeps := len(groups) - 1
	if numSeps > 0 {
		sRunes := []rune(s)

		// Determine if grouping repeats:
		// - 1 separator: always repeat at rightmost group interval
		// - 2+ separators: repeat only if ALL groups (including leftmost) are the same size
		shouldRepeat := false
		interval := groups[len(groups)-1] // rightmost group size
		if numSeps == 1 {
			shouldRepeat = true
		} else {
			shouldRepeat = true
			for _, g := range groups {
				if g != interval {
					shouldRepeat = false
					break
				}
			}
		}

		if shouldRepeat {
			sep := groupSeps[numSeps-1] // use rightmost separator character
			var result []rune
			for i := len(sRunes) - 1; i >= 0; i-- {
				digitPos := len(sRunes) - 1 - i
				if digitPos > 0 && digitPos%interval == 0 {
					result = append(result, sep)
				}
				result = append(result, sRunes[i])
			}
			for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
				result[i], result[j] = result[j], result[i]
			}
			s = string(result)
		} else {
			// Explicit positions only — build positions from right
			var sepPositions []int
			var sepChars []rune
			pos := 0
			for i := len(groups) - 1; i >= 1; i-- {
				pos += groups[i]
				sepPositions = append(sepPositions, pos)
				sepChars = append(sepChars, groupSeps[i-1])
			}
			sepAt := make(map[int]rune)
			for i, p := range sepPositions {
				sepAt[p] = sepChars[i]
			}
			var result []rune
			for i := len(sRunes) - 1; i >= 0; i-- {
				digitPos := len(sRunes) - 1 - i
				if sep, ok := sepAt[digitPos]; ok {
					result = append(result, sep)
				}
				result = append(result, sRunes[i])
			}
			for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
				result[i], result[j] = result[j], result[i]
			}
			s = string(result)
		}
	}

	// Translate to target digit set if non-ASCII
	if zeroDigit != '0' {
		s = translateDigits(s, zeroDigit)
	}

	return s, nil
}

// unicodeDigitZero returns the zero digit for the same numeric block as r.
func unicodeDigitZero(r rune) rune {
	if r >= '0' && r <= '9' {
		return '0'
	}
	// Unicode decimal digit blocks are consecutive groups of 10.
	// Walk backwards from r to find the zero.
	for z := r; z >= r-9 && z >= 0; z-- {
		if !unicode.IsDigit(z) {
			return z + 1
		}
	}
	return r - 9 // Fallback: assume it's a digit 9
}

func isDecimalDigitInRange(r rune, zero rune) bool {
	return r >= zero && r <= zero+9
}

func translateDigits(s string, zeroDigit rune) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(zeroDigit + (r - '0'))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Language-specific word formatting

func formatWordsLang(n *big.Int, lang, caseStyle string) string {
	switch lang {
	case "de":
		return applyCase(intToGermanWords(n), caseStyle)
	case "fr":
		return applyCase(intToFrenchWords(n), caseStyle)
	case "it":
		return applyCase(intToItalianWords(n), caseStyle)
	default:
		return applyCase(intToEnglishWords(n), caseStyle)
	}
}

func intToGermanWords(n *big.Int) string {
	if n.Sign() == 0 {
		return "null"
	}
	if !n.IsInt64() {
		return n.String()
	}
	return int64ToGermanWords(n.Int64())
}

func int64ToGermanWords(n int64) string {
	if n == 0 {
		return "null"
	}
	ones := []string{"", "eins", "zwei", "drei", "vier", "fünf", "sechs", "sieben", "acht", "neun", "zehn",
		"elf", "zwölf", "dreizehn", "vierzehn", "fünfzehn", "sechzehn", "siebzehn", "achtzehn", "neunzehn"}
	tens := []string{"", "", "zwanzig", "dreißig", "vierzig", "fünfzig", "sechzig", "siebzig", "achtzig", "neunzig"}

	if n < 20 {
		return ones[n]
	}
	if n < 100 {
		if n%10 == 0 {
			return tens[n/10]
		}
		unit := ones[n%10]
		if n%10 == 1 {
			unit = "ein"
		}
		return unit + "und" + tens[n/10]
	}
	return fmt.Sprintf("%d", n)
}

func applyOrdinalWordsDe(words, caseStyle, suffix string) string {
	base := germanOrdinalWord(strings.ToLower(words))
	if suffix == "" || suffix == "%spellout-ordinal" {
		return applyCase(base, caseStyle)
	}
	return applyCase(germanOrdinalWithSuffix(base, suffix), caseStyle)
}

func germanOrdinalWord(words string) string {
	irregulars := map[string]string{
		"null":   "nullte",
		"ein":    "erste",
		"eins":   "erste",
		"drei":   "dritte",
		"sieben": "siebte",
		"acht":   "achte",
	}
	if ordinal, ok := irregulars[words]; ok {
		return ordinal
	}

	if n, ok := germanWordToNumber(words); ok && n < 20 {
		return words + "te"
	}
	return words + "ste"
}

func germanWordToNumber(word string) (int64, bool) {
	units := map[string]int64{
		"null": 0, "ein": 1, "eins": 1, "zwei": 2, "drei": 3, "vier": 4,
		"fünf": 5, "sechs": 6, "sieben": 7, "acht": 8, "neun": 9, "zehn": 10,
		"elf": 11, "zwölf": 12, "dreizehn": 13, "vierzehn": 14, "fünfzehn": 15,
		"sechzehn": 16, "siebzehn": 17, "achtzehn": 18, "neunzehn": 19,
	}
	if n, ok := units[word]; ok {
		return n, true
	}

	tens := []struct {
		word string
		val  int64
	}{
		{"zwanzig", 20},
		{"dreißig", 30},
		{"vierzig", 40},
		{"fünfzig", 50},
		{"sechzig", 60},
		{"siebzig", 70},
		{"achtzig", 80},
		{"neunzig", 90},
	}
	for _, ten := range tens {
		if word == ten.word {
			return ten.val, true
		}
		if !strings.HasSuffix(word, ten.word) {
			continue
		}
		prefix := strings.TrimSuffix(word, ten.word)
		prefix = strings.TrimSuffix(prefix, "und")
		if prefix == "" {
			return ten.val, true
		}
		if unit, ok := units[prefix]; ok && unit > 0 && unit < 10 {
			return ten.val + unit, true
		}
	}

	return 0, false
}

func germanOrdinalWithSuffix(base, suffix string) string {
	sfx := strings.TrimPrefix(suffix, "-")
	if sfx == "" {
		return base
	}
	base = strings.TrimSuffix(base, "e")
	return base + sfx
}

func intToFrenchWords(n *big.Int) string {
	if n.Sign() == 0 {
		return "zéro"
	}
	if !n.IsInt64() {
		return n.String()
	}
	return int64ToFrenchWords(n.Int64())
}

func int64ToFrenchWords(n int64) string {
	if n == 0 {
		return "zéro"
	}
	ones := []string{"", "un", "deux", "trois", "quatre", "cinq", "six", "sept", "huit", "neuf", "dix",
		"onze", "douze", "treize", "quatorze", "quinze", "seize", "dix-sept", "dix-huit", "dix-neuf"}
	tens := []string{"", "", "vingt", "trente", "quarante", "cinquante", "soixante", "soixante", "quatre-vingt", "quatre-vingt"}

	if n < 20 {
		return ones[n]
	}
	if n < 100 {
		t := n / 10
		u := n % 10
		if t == 7 || t == 9 {
			return tens[t] + "-" + ones[u+10]
		}
		if u == 0 {
			if t == 8 {
				return "quatre-vingts"
			}
			return tens[t]
		}
		if u == 1 && t != 8 {
			return tens[t] + " et un"
		}
		return tens[t] + "-" + ones[u]
	}
	return fmt.Sprintf("%d", n)
}

func applyOrdinalWordsFr(words, caseStyle, _ string) string {
	lower := strings.ToLower(words)
	if lower == "un" {
		return applyCase("premier", caseStyle)
	}
	base := strings.TrimSuffix(lower, "e")
	return applyCase(base+"ième", caseStyle)
}

func intToItalianWords(n *big.Int) string {
	if n.Sign() == 0 {
		return "zero"
	}
	if !n.IsInt64() {
		return n.String()
	}
	return int64ToItalianWords(n.Int64())
}

func int64ToItalianWords(n int64) string {
	if n == 0 {
		return "zero"
	}
	ones := []string{"", "uno", "due", "tre", "quattro", "cinque", "sei", "sette", "otto", "nove", "dieci",
		"undici", "dodici", "tredici", "quattordici", "quindici", "sedici", "diciassette", "diciotto", "diciannove"}
	tens := []string{"", "", "venti", "trenta", "quaranta", "cinquanta", "sessanta", "settanta", "ottanta", "novanta"}

	if n < 20 {
		return ones[n]
	}
	if n < 100 {
		t := n / 10
		u := n % 10
		base := tens[t]
		if u == 0 {
			return base
		}
		if u == 1 || u == 8 {
			base = base[:len(base)-1]
		}
		return base + ones[u]
	}
	return fmt.Sprintf("%d", n)
}

func applyOrdinalWordsIt(words, caseStyle, suffix string) string {
	lower := strings.ToLower(words)
	switch suffix {
	case "", "%spellout-ordinal", "%spellout-ordinal-masculine", "-o":
		return applyCase(italianOrdinalWord(lower, false), caseStyle)
	case "%spellout-ordinal-feminine", "-a":
		return applyCase(italianOrdinalWord(lower, true), caseStyle)
	default:
		n := italianWordToNumber(lower)
		if n == 0 {
			return applyCase(italianOrdinalWord(lower, false), caseStyle)
		}
		return applyCase(italianOrdinalWithSuffix(n, suffix), caseStyle)
	}
}

func italianOrdinalWord(words string, feminine bool) string {
	n := italianWordToNumber(words)
	if n != 0 {
		if feminine {
			if s, ok := map[int64]string{
				1: "prima", 2: "seconda", 3: "terza", 4: "quarta", 5: "quinta",
				6: "sesta", 7: "settima", 8: "ottava", 9: "nona", 10: "decima",
			}[n]; ok {
				return s
			}
		} else {
			if s, ok := map[int64]string{
				1: "primo", 2: "secondo", 3: "terzo", 4: "quarto", 5: "quinto",
				6: "sesto", 7: "settimo", 8: "ottavo", 9: "nono", 10: "decimo",
			}[n]; ok {
				return s
			}
		}
	}

	stem := strings.TrimRight(words, "aeiou")
	if stem == "" {
		stem = words
	}
	if feminine {
		return stem + "esima"
	}
	return stem + "esimo"
}

func italianWordToNumber(word string) int64 {
	italianNumbers := map[string]int64{
		"uno": 1, "due": 2, "tre": 3, "quattro": 4, "cinque": 5,
		"sei": 6, "sette": 7, "otto": 8, "nove": 9, "dieci": 10,
	}
	if n, ok := italianNumbers[word]; ok {
		return n
	}
	return 0
}

func italianOrdinalWithSuffix(n int64, suffix string) string {
	// Suffix starts with '-' for gender marker, e.g., "-o" for masculine, "-a" for feminine
	sfx := strings.TrimPrefix(suffix, "-")
	stems := map[int64]string{
		1: "prim", 2: "second", 3: "terz", 4: "quart", 5: "quint",
		6: "sest", 7: "settim", 8: "ottav", 9: "non", 10: "decim",
	}
	if stem, ok := stems[n]; ok {
		return stem + sfx
	}
	return fmt.Sprintf("%d°", n)
}
