package xslt3

import (
	"context"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/internal/sequence"
)

func (ec *execContext) execNumber(ctx context.Context, inst *NumberInst) error {
	node := ec.contextNode

	// XSLT 3.0: select attribute specifies which node to number.
	// Evaluate select before the nil-node check so it works inside xsl:function.
	if inst.Select != nil && inst.Value == nil {
		result, err := ec.evalXPath(nil, inst.Select, node)
		if err != nil {
			return err
		}
		seq := result.Sequence()
		seqLen := 0
		if seq != nil {
			seqLen = sequence.Len(seq)
		}
		if seqLen == 0 {
			// XTTE1000: select evaluates to empty sequence
			return dynamicError(errCodeXTTE1000,
				"xsl:number select expression returned empty sequence")
		}
		if seqLen > 1 {
			// XTTE1000: select must return exactly one node
			return dynamicError(errCodeXTTE1000,
				"xsl:number select expression returned more than one item")
		}
		if ni, ok := seq.Get(0).(xpath3.NodeItem); ok {
			node = ni.Node
		} else {
			// XTTE0990: select result is not a node
			return dynamicError(errCodeXTTE0990,
				"xsl:number select expression did not return a node")
		}
	}

	// XTTE0990: context item must be a node when value is absent
	if inst.Value == nil && node == nil {
		return dynamicError(errCodeXTTE0990,
			"xsl:number requires a node context when value is not specified")
	}

	var bigNums []*big.Int

	if inst.Value != nil {
		// value attribute: evaluate expression and use result directly
		result, err := ec.evalXPath(nil, inst.Value, node)
		if err != nil {
			return err
		}
		seq := result.Sequence()
		for item := range sequence.Items(seq) {
			av, err := xpath3.AtomizeItem(item)
			if err != nil {
				// XTDE0980: value is not numeric
				return dynamicError(errCodeXTDE0980,
					"xsl:number value is not numeric")
			}
			bi, err := atomicToBigInt(av)
			if err != nil {
				return err
			}
			bigNums = append(bigNums, bi)
		}
	} else {
		var nums []int
		switch inst.Level {
		case "single":
			nums = ec.numberSingle(inst, node)
		case "multiple":
			nums = ec.numberMultiple(inst, node)
		case "any":
			nums = ec.numberAny(inst, node)
		default:
			nums = ec.numberSingle(inst, node)
		}
		for _, n := range nums {
			bigNums = append(bigNums, big.NewInt(int64(n)))
		}
	}

	// Apply start-at offset (XSLT 3.0): default is 1, so start-at="0" subtracts 1
	if inst.StartAt != nil {
		saStr, err := inst.StartAt.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		// start-at can be a space-separated list of integers, one per level
		saParts := strings.Fields(saStr)
		for i, n := range bigNums {
			offset := 0
			if i < len(saParts) {
				offset, _ = strconv.Atoi(saParts[i])
			} else if len(saParts) > 0 {
				offset, _ = strconv.Atoi(saParts[len(saParts)-1])
			}
			// start-at shifts numbering: number = number + startAt - 1
			bigNums[i] = new(big.Int).Add(n, big.NewInt(int64(offset-1)))
		}
	}

	// Format the number list
	format := "1"
	if inst.Format != nil {
		var err error
		format, err = inst.Format.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
	}

	groupSep := ""
	if inst.GroupingSeparator != nil {
		var err error
		groupSep, err = inst.GroupingSeparator.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
	}
	groupSize := 0
	if inst.GroupingSize != nil {
		gsStr, err := inst.GroupingSize.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		groupSize, _ = strconv.Atoi(gsStr)
	}

	lang := ""
	if inst.Lang != nil {
		var err error
		lang, err = inst.Lang.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		if lang != "" && !langRe.MatchString(lang) {
			return dynamicError(errCodeXTDE0030,
				"xsl:number lang attribute is not a valid language code: %q", lang)
		}
	}

	ordinal := ""
	if inst.Ordinal != nil {
		var err error
		ordinal, err = inst.Ordinal.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
	}

	formatted := formatBigNumberList(bigNums, format, groupSep, groupSize, lang, ordinal)
	text, err := ec.resultDoc.CreateText([]byte(formatted))
	if err != nil {
		return err
	}
	return ec.addNode(text)
}

// numberNodeMatches tests if a node matches the count pattern.
// If no count pattern, matches nodes with the same type and name as the context node.
func (ec *execContext) numberNodeMatches(inst *NumberInst, target helium.Node, contextNode helium.Node) bool {
	if inst.Count != nil {
		// Special case: count="." matches any non-document node
		if inst.Count.source == "." && target.Type() != helium.DocumentNode {
			return true
		}
		// Per XSLT 3.0 section 13.4.1, current() within a count pattern
		// refers to the node being tested against the pattern (the candidate),
		// so that predicates like [@bar = current()/@bar] evaluate relative
		// to each candidate node during the numbering walk.
		savedCurrent := ec.currentNode
		ec.currentNode = target
		result := inst.Count.matchPattern(ec, target)
		ec.currentNode = savedCurrent
		return result
	}
	// Default: same node type and expanded name
	if contextNode == nil {
		return false
	}
	if target.Type() != contextNode.Type() {
		return false
	}
	if target.Type() == helium.ElementNode {
		te := target.(*helium.Element)
		ce := contextNode.(*helium.Element)
		return te.LocalName() == ce.LocalName() && te.URI() == ce.URI()
	}
	return target.Name() == contextNode.Name()
}

// numberFromMatches tests if a node matches the from pattern.
func (ec *execContext) numberFromMatches(inst *NumberInst, node helium.Node) bool {
	if inst.From == nil {
		return false
	}
	// Same current() semantics as numberNodeMatches: current() refers to
	// the candidate node being tested, not the xsl:number context node.
	savedCurrent := ec.currentNode
	ec.currentNode = node
	result := inst.From.matchPattern(ec, node)
	ec.currentNode = savedCurrent
	return result
}

// numberSingle implements level="single": find the first ancestor-or-self that
// matches the count pattern, then count preceding siblings that match.
func (ec *execContext) numberSingle(inst *NumberInst, node helium.Node) []int {
	// Find the first ancestor-or-self that matches count
	target := ec.numberFindAncestorOrSelf(inst, node)
	if target == nil {
		return nil
	}

	// Count preceding siblings that match count pattern
	count := 1
	for sib := target.PrevSibling(); sib != nil; sib = sib.PrevSibling() {
		if ec.numberNodeMatches(inst, sib, node) {
			count++
		}
	}
	return []int{count}
}

// numberMultiple implements level="multiple": find all ancestors-or-self that match
// count (stopping at from), and for each count preceding siblings.
func (ec *execContext) numberMultiple(inst *NumberInst, node helium.Node) []int {
	var ancestors []helium.Node
	for n := node; n != nil; n = n.Parent() {
		if ec.numberFromMatches(inst, n) {
			// Include the from node itself if it matches count
			if ec.numberNodeMatches(inst, n, node) {
				ancestors = append(ancestors, n)
			}
			break
		}
		if ec.numberNodeMatches(inst, n, node) {
			ancestors = append(ancestors, n)
		}
		if n.Type() == helium.DocumentNode {
			break
		}
	}

	// Reverse to get document order (outermost first)
	for i, j := 0, len(ancestors)-1; i < j; i, j = i+1, j-1 {
		ancestors[i], ancestors[j] = ancestors[j], ancestors[i]
	}

	nums := make([]int, len(ancestors))
	for i, anc := range ancestors {
		count := 1
		for sib := anc.PrevSibling(); sib != nil; sib = sib.PrevSibling() {
			if ec.numberNodeMatches(inst, sib, node) {
				count++
			}
		}
		nums[i] = count
	}
	return nums
}

// numberAny implements level="any": count all matching nodes in document order
// that precede (or are) the context node, going back to the nearest from match.
// The from node itself is included in the count if it matches count.
func (ec *execContext) numberAny(inst *NumberInst, node helium.Node) []int {
	count := 0
	cur := node
	for cur != nil {
		if ec.numberNodeMatches(inst, cur, node) {
			count++
		}
		if ec.numberFromMatches(inst, cur) {
			break
		}
		cur = ec.prevInDocOrder(cur)
	}
	if count == 0 {
		return nil
	}
	return []int{count}
}

// prevInDocOrder returns the previous node in document order.
func (ec *execContext) prevInDocOrder(node helium.Node) helium.Node {
	// Previous sibling's deepest last descendant
	if prev := node.PrevSibling(); prev != nil {
		return ec.lastDescendant(prev)
	}
	// Otherwise, parent (including document node)
	parent := node.Parent()
	if parent == nil {
		return nil
	}
	return parent
}

// lastDescendant returns the deepest last descendant of node (or node itself if leaf).
func (ec *execContext) lastDescendant(node helium.Node) helium.Node {
	if node.Type() == helium.ElementNode {
		elem := node.(*helium.Element)
		if last := elem.LastChild(); last != nil {
			return ec.lastDescendant(last)
		}
	}
	return node
}

// numberFindAncestorOrSelf finds the first ancestor-or-self that matches
// the count pattern.
func (ec *execContext) numberFindAncestorOrSelf(inst *NumberInst, node helium.Node) helium.Node {
	for n := node; n != nil; n = n.Parent() {
		if ec.numberNodeMatches(inst, n, node) {
			return n
		}
		// Stop at document node to avoid walking above the tree
		if n.Type() == helium.DocumentNode {
			return nil
		}
	}
	return nil
}

// atomicToBigInt converts an atomic value to *big.Int for xsl:number formatting.
// Integer types are used directly; decimals are rounded; doubles/floats are rounded
// then converted. Returns an error for NaN, Inf, or negative values.
func atomicToBigInt(av xpath3.AtomicValue) (*big.Int, error) {
	switch v := av.Value.(type) {
	case *big.Int:
		// Integer type — use directly without float64 conversion
		if v.Sign() < 0 {
			return nil, dynamicError(errCodeXTDE0980,
				"xsl:number value is not a non-negative integer: %v", v)
		}
		return new(big.Int).Set(v), nil
	case *big.Rat:
		// Decimal type — round half-away-from-zero (fn:round semantics per XSLT spec)
		// Truncate toward zero to get integer part
		intPart := new(big.Int).Quo(v.Num(), v.Denom())
		// Compute remainder: v - intPart
		rem := new(big.Rat).Sub(v, new(big.Rat).SetInt(intPart))
		absRem := new(big.Rat).Abs(rem)
		half := new(big.Rat).SetFrac64(1, 2)
		if absRem.Cmp(half) >= 0 {
			// Round away from zero
			if v.Sign() >= 0 {
				intPart.Add(intPart, big.NewInt(1))
			} else {
				intPart.Sub(intPart, big.NewInt(1))
			}
		}
		if intPart.Sign() < 0 {
			return nil, dynamicError(errCodeXTDE0980,
				"xsl:number value is not a non-negative integer: %v", v)
		}
		return intPart, nil
	default:
		// Double/float — cast to double first
		dv, err := xpath3.CastAtomic(av, xpath3.TypeDouble)
		if err != nil {
			return nil, dynamicError(errCodeXTDE0980,
				"xsl:number value %q cannot be converted to a number", av.StringVal())
		}
		f := math.Round(dv.DoubleVal())
		if math.IsNaN(f) || math.IsInf(f, 0) || f < 0 {
			return nil, dynamicError(errCodeXTDE0980,
				"xsl:number value is not a non-negative integer: %v", dv.DoubleVal())
		}
		// Use strconv to get the scientific-notation string, then parse
		// with big.Float at high precision. This avoids the float64
		// mantissa noise that SetFloat64 would preserve (e.g. 1e100
		// as float64 is not exactly 10^100).
		s := strconv.FormatFloat(f, 'e', -1, 64)
		bf, _, err := new(big.Float).SetPrec(512).Parse(s, 10)
		if err != nil {
			return nil, dynamicError(errCodeXTDE0980,
				"xsl:number value %q cannot be converted to integer", s)
		}
		bi, _ := bf.Int(nil)
		return bi, nil
	}
}

func isAlphanumeric(r rune) bool {
	if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
		return true
	}
	return r > 127 && (unicode.IsLetter(r) || unicode.IsNumber(r))
}

func formatSingleNumber(num int, token string, groupSep string, groupSize int, lang string, ordinal string) string {
	switch token {
	case "a":
		return toLowerAlpha(num)
	case "A":
		return toUpperAlpha(num)
	case "i":
		return strings.ToLower(toRoman(num))
	case "I":
		return toRoman(num)
	case "w":
		return numberToWordsLang(num, "lower", lang, ordinal)
	case "W":
		return numberToWordsLang(num, "upper", lang, ordinal)
	case "Ww":
		return numberToWordsLang(num, "title", lang, ordinal)
	default:
		runes := []rune(token)
		firstRune := runes[0]

		// Non-ASCII digit: use as base of a decimal numbering system
		// (e.g., ٠ = U+0660 for Arabic-Indic digits)
		if firstRune > 127 && unicode.IsDigit(firstRune) {
			s := formatWithDigitSystem(num, digitZeroOf(firstRune), len(runes))
			if groupSep != "" && groupSize > 0 {
				s = applyGroupingSeparator(s, groupSep, groupSize)
			}
			return s
		}
		// Non-ASCII number (not a digit): ordinal numbering from that codepoint
		// (e.g., ① = U+2460 for circled digits)
		if firstRune > 127 && unicode.IsNumber(firstRune) {
			return formatWithOrdinalSystem(num, firstRune)
		}
		// Non-ASCII letter: alphabetic numbering from that codepoint
		// (e.g., α = U+03B1 for Greek lowercase: α, β, γ, ..., ω, αα, αβ, ...)
		if firstRune > 127 && unicode.IsLetter(firstRune) {
			return formatWithAlphaSystem(num, firstRune)
		}

		// ASCII numeric format: determine minimum width from token (e.g., "001" = width 3)
		minWidth := len(token)
		s := strconv.Itoa(num)
		for len(s) < minWidth {
			s = "0" + s
		}
		if groupSep != "" && groupSize > 0 {
			s = applyGroupingSeparator(s, groupSep, groupSize)
		}
		// Append ordinal suffix for numeric format with ordinal="yes"
		if ordinal != "" {
			s += numericOrdinalSuffix(num, lang)
		}
		return s
	}
}

// numericOrdinalSuffix returns the suffix for a numeric ordinal (e.g., "st", "nd", "rd", "th").
func numericOrdinalSuffix(num int, lang string) string {
	// Default to English ordinal suffixes
	n := num % 100
	if n >= 11 && n <= 13 {
		return "th"
	}
	switch num % 10 {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	default:
		return "th"
	}
}

// knownAlphabetLengths maps the start character of well-known alphabets to their
// standard length, preventing over-counting into diacritical variants.
// langRe matches xs:language values per XML Schema (BCP 47 language tags).
var langRe = regexp.MustCompile(`^[a-zA-Z]{1,8}(-[a-zA-Z0-9]{1,8})*$`)

var knownAlphabetLengths = map[rune]int{
	'α': 25, // Greek lowercase: α(U+03B1) through ω(U+03C9)
	'Α': 25, // Greek uppercase: Α(U+0391) through Ω(U+03A9)
	'а': 32, // Cyrillic lowercase: а(U+0430) through я(U+044F)
	'А': 32, // Cyrillic uppercase: А(U+0410) through Я(U+042F)
}

// formatWithAlphaSystem formats using alphabetic numbering (like a-z but with
// non-ASCII letters). Wraps at the end of the alphabet:
// α=1, β=2, ..., ω=25, αα=26, αβ=27, etc.
func formatWithAlphaSystem(num int, start rune) string {
	if num <= 0 {
		return strconv.Itoa(num)
	}

	// Use known alphabet length if available, otherwise detect
	seqLen, ok := knownAlphabetLengths[start]
	if !ok {
		seqLen = 0
		for r := start; unicode.IsLetter(r); r++ {
			seqLen++
			if seqLen > 100 {
				break // safety cap
			}
		}
	}
	if seqLen == 0 {
		return strconv.Itoa(num)
	}

	// Convert to base-N alphabetic representation (1-based, like a=1, b=2, ..., z=26, aa=27)
	var result []rune
	n := num
	for n > 0 {
		n-- // convert to 0-based
		result = append([]rune{start + rune(n%seqLen)}, result...)
		n /= seqLen
	}
	return string(result)
}

// formatBigNumberList formats a list of *big.Int values according to an XSLT
// format string, supporting all format tokens (decimal, alphabetic, roman,
// non-ASCII digit systems) with prefix, suffix, separators, and grouping.
func formatBigNumberList(bigNums []*big.Int, format string, groupSep string, groupSize int, lang string, ordinal string) string {
	// Parse format string into tokens (same logic as formatNumberList)
	type fmtToken struct {
		format    string
		separator string
	}

	runes := []rune(format)
	var prefix, suffix string
	var tokens []fmtToken

	i := 0
	for i < len(runes) && !isAlphanumeric(runes[i]) {
		i++
	}
	prefix = string(runes[:i])

	for i < len(runes) {
		start := i
		for i < len(runes) && isAlphanumeric(runes[i]) {
			i++
		}
		if start == i {
			break
		}
		fmtStr := string(runes[start:i])

		sepStart := i
		for i < len(runes) && !isAlphanumeric(runes[i]) {
			i++
		}
		sep := string(runes[sepStart:i])

		if i >= len(runes) {
			tokens = append(tokens, fmtToken{format: fmtStr})
			suffix = sep
		} else {
			tokens = append(tokens, fmtToken{format: fmtStr, separator: sep})
		}
	}

	if len(tokens) == 0 {
		tokens = []fmtToken{{format: "1"}}
		suffix = prefix
	}

	defaultSep := "."
	if len(tokens) > 1 {
		defaultSep = tokens[len(tokens)-2].separator
		if defaultSep == "" {
			defaultSep = "."
		}
	}

	var buf strings.Builder
	buf.WriteString(prefix)
	for idx, num := range bigNums {
		if idx > 0 {
			if idx < len(tokens) && tokens[idx-1].separator != "" {
				buf.WriteString(tokens[idx-1].separator)
			} else {
				buf.WriteString(defaultSep)
			}
		}
		tokIdx := idx
		if tokIdx >= len(tokens) {
			tokIdx = len(tokens) - 1
		}
		// If the number fits in an int, use the standard formatter for full
		// support of roman numerals, alphabetic, words, etc.
		if num.IsInt64() {
			n := int(num.Int64())
			if int64(n) == num.Int64() {
				buf.WriteString(formatSingleNumber(n, tokens[tokIdx].format, groupSep, groupSize, lang, ordinal))
				continue
			}
		}
		// Large number: only decimal digit formatting is meaningful
		buf.WriteString(formatBigSingleNumber(num, tokens[tokIdx].format, groupSep, groupSize))
	}
	buf.WriteString(suffix)
	return buf.String()
}

// formatBigSingleNumber formats a *big.Int using a decimal format token.
// Supports ASCII digits with grouping and non-ASCII digit systems.
func formatBigSingleNumber(num *big.Int, token string, groupSep string, groupSize int) string {
	runes := []rune(token)
	firstRune := runes[0]

	// Non-ASCII digit system (e.g., Arabic-Indic ٠)
	if firstRune > 127 && unicode.IsDigit(firstRune) {
		s := formatBigWithDigitSystem(num, digitZeroOf(firstRune), len(runes))
		if groupSep != "" && groupSize > 0 {
			s = applyGroupingSeparator(s, groupSep, groupSize)
		}
		return s
	}

	// Default: decimal with optional grouping and min-width
	minWidth := len(token)
	s := num.String()
	for len(s) < minWidth {
		s = "0" + s
	}
	if groupSep != "" && groupSize > 0 {
		s = applyGroupingSeparator(s, groupSep, groupSize)
	}
	return s
}

// formatBigWithDigitSystem formats a *big.Int using a non-ASCII decimal digit
// system (e.g., Arabic-Indic digits starting at U+0660).
func formatBigWithDigitSystem(num *big.Int, zero rune, minWidth int) string {
	if num.Sign() < 0 {
		return "-" + formatBigWithDigitSystem(new(big.Int).Neg(num), zero, minWidth)
	}
	if num.Sign() == 0 {
		s := string(zero)
		for len([]rune(s)) < minWidth {
			s = string(zero) + s
		}
		return s
	}
	ten := big.NewInt(10)
	var result []rune
	n := new(big.Int).Set(num)
	for n.Sign() > 0 {
		mod := new(big.Int)
		n.DivMod(n, ten, mod)
		result = append([]rune{zero + rune(mod.Int64())}, result...)
	}
	for len(result) < minWidth {
		result = append([]rune{zero}, result...)
	}
	return string(result)
}

// digitZeroOf returns the zero digit for the Unicode digit block containing r.
func digitZeroOf(r rune) rune {
	// Unicode digit blocks are groups of 10 consecutive codepoints.
	// The zero digit is always at an offset of (r - digit_value) from r.
	// For standard digit blocks, digit_value = r % 10 when aligned at 0.
	// Use unicode.Digit to get the numeric value.
	for d := rune(0); d <= 9; d++ {
		if r-d >= 0 && unicode.IsDigit(r-d) {
			// Verify this is actually the zero of the block
			candidate := r - d
			if !unicode.IsDigit(candidate-1) || candidate == 0 {
				return candidate
			}
		}
	}
	// Fallback: assume digit value is r mod 10 offset
	return r - (r % 10)
}

// formatWithDigitSystem formats a number using a decimal digit system
// starting at the given zero codepoint.
func formatWithDigitSystem(num int, zero rune, minWidth int) string {
	if num < 0 {
		return "-" + formatWithDigitSystem(-num, zero, minWidth)
	}
	if num == 0 {
		s := string(zero)
		for len([]rune(s)) < minWidth {
			s = string(zero) + s
		}
		return s
	}
	var runes []rune
	n := num
	for n > 0 {
		runes = append([]rune{zero + rune(n%10)}, runes...)
		n /= 10
	}
	for len(runes) < minWidth {
		runes = append([]rune{zero}, runes...)
	}
	return string(runes)
}

// ordinalSystem describes a Unicode numbering system with potentially
// non-contiguous ranges and a special zero character.
type ordinalSystem struct {
	oneChar rune   // the character representing 1
	zero    rune   // the character representing 0 (0 if none)
	ranges  []rune // pairs of (first, last) codepoints for contiguous ranges starting at 1
}

// knownOrdinalSystems maps the "1" character to its ordinal system definition.
var knownOrdinalSystems = map[rune]ordinalSystem{
	// Circled digits: ①-⑳, ㉑-㉟, ㊱-㊿
	0x2460: {oneChar: 0x2460, zero: 0x24EA, ranges: []rune{0x2460, 0x2473, 0x3251, 0x325F, 0x32B1, 0x32BF}},
	// Parenthesized digits: ⑴-⒇ (no special zero)
	0x2474: {oneChar: 0x2474, zero: 0, ranges: []rune{0x2474, 0x2487}},
	// Full-stop digits: ⒈-⒛, zero: 🄀 (U+1F100)
	0x2488: {oneChar: 0x2488, zero: 0x1F100, ranges: []rune{0x2488, 0x249B}},
	// Double circled digits: ⓵-⓾ (no special zero)
	0x24F5: {oneChar: 0x24F5, zero: 0, ranges: []rune{0x24F5, 0x24FE}},
	// Dingbat negative circled: ❶-❿, ⓫-⓴
	0x2776: {oneChar: 0x2776, zero: 0x24FF, ranges: []rune{0x2776, 0x277F, 0x24EB, 0x24F4}},
	// Dingbat negative circled sans-serif: ➊-➓
	0x278A: {oneChar: 0x278A, zero: 0x1F10C, ranges: []rune{0x278A, 0x2793}},
	// Dingbat negative circled sans-serif (alt, starting from ➀): ➀-➉
	0x2780: {oneChar: 0x2780, zero: 0x1F10B, ranges: []rune{0x2780, 0x2789}},
	// Parenthesized ideograph: ㈠-㈩
	0x3220: {oneChar: 0x3220, zero: 0, ranges: []rune{0x3220, 0x3229}},
	// Circled ideograph: ㊀-㊉
	0x3280: {oneChar: 0x3280, zero: 0, ranges: []rune{0x3280, 0x3289}},
	// Aegean numbers: 𐄇-𐄐 (1-10)
	0x10107: {oneChar: 0x10107, zero: 0, ranges: []rune{0x10107, 0x10110}},
	// Coptic Epact numbers: 𐋡-𐋪 (1-10)
	0x102E1: {oneChar: 0x102E1, zero: 0, ranges: []rune{0x102E1, 0x102EA}},
	// Rumi numerals: 𐹠-𐹩 (1-10)
	0x10E60: {oneChar: 0x10E60, zero: 0, ranges: []rune{0x10E60, 0x10E69}},
	// Brahmi number signs: 𑁒-𑁛 (1-10)
	0x11052: {oneChar: 0x11052, zero: 0, ranges: []rune{0x11052, 0x1105B}},
	// Sinhala archaic numbers: 𑇡-𑇪 (1-10)
	0x111E1: {oneChar: 0x111E1, zero: 0, ranges: []rune{0x111E1, 0x111EA}},
	// Counting rod unit digits: 𝍠-𝍨 (1-9)
	0x1D360: {oneChar: 0x1D360, zero: 0, ranges: []rune{0x1D360, 0x1D368}},
	// Mende Kikakui digits: 𞣇-𞣏 (1-9)
	0x1E8C7: {oneChar: 0x1E8C7, zero: 0, ranges: []rune{0x1E8C7, 0x1E8CF}},
	// Digit comma: 🄂-🄊 (1-9), zero: 🄁
	0x1F102: {oneChar: 0x1F102, zero: 0x1F101, ranges: []rune{0x1F102, 0x1F10A}},
}

// formatWithOrdinalSystem formats using a known ordinal numbering system.
// Falls back to decimal when the number exceeds the system's range.
func formatWithOrdinalSystem(num int, start rune) string {
	if num < 0 {
		return strconv.Itoa(num)
	}

	// Look up the system by the start character (which represents 1)
	sys, ok := knownOrdinalSystems[start]
	if !ok {
		// Unknown system: detect range by finding the block boundaries.
		// First check if start-1 is also a numbering character (the zero).
		hasZero := false
		prev := start - 1
		if prev > 0 && (unicode.IsNumber(prev) || unicode.IsLetter(prev)) {
			hasZero = true
		}
		// Count consecutive same-category characters from start, capped at 10.
		rangeLen := ordinalRangeLength(start)
		// If we have a zero predecessor, the system is (zero, 1..N).
		// The range from start to the end of the system is one less than
		// the total block size starting at zero.
		if hasZero {
			totalFromZero := ordinalRangeLength(prev)
			rangeFromStart := totalFromZero - 1
			if rangeFromStart < rangeLen {
				rangeLen = rangeFromStart
			}
		}

		if num == 0 {
			if hasZero {
				return string(prev)
			}
			return strconv.Itoa(0)
		}
		if num <= rangeLen {
			return string(start + rune(num-1))
		}
		return strconv.Itoa(num)
	}

	if num == 0 {
		if sys.zero != 0 {
			return string(sys.zero)
		}
		return strconv.Itoa(0)
	}

	// Map num to the correct codepoint across potentially non-contiguous ranges
	pos := num // 1-based position in the sequence
	for i := 0; i+1 < len(sys.ranges); i += 2 {
		rangeStart := sys.ranges[i]
		rangeEnd := sys.ranges[i+1]
		rangeLen := int(rangeEnd - rangeStart + 1)
		if pos <= rangeLen {
			return string(rangeStart + rune(pos-1))
		}
		pos -= rangeLen
	}

	// Number exceeds ordinal system range: fall back to decimal
	return strconv.Itoa(num)
}

// ordinalRangeLength returns how many consecutive characters starting at r
// belong to the same Unicode category (Number or Letter). This determines
// the range of an ordinal numbering system. For unknown systems, the range
// is capped at 10 to avoid accidentally including characters from adjacent
// but unrelated numbering systems.
func ordinalRangeLength(r rune) int {
	isNum := unicode.IsNumber(r)
	count := 0
	for c := r; ; c++ {
		if isNum {
			if !unicode.IsNumber(c) {
				break
			}
		} else {
			if !unicode.IsLetter(c) {
				break
			}
		}
		count++
		if count >= 10 {
			break // cap unknown systems at 10
		}
	}
	return count
}

// numberToWords converts a number to English words.
func numberToWords(n int, upper bool) string {
	if n == 0 {
		if upper {
			return "ZERO"
		}
		return "zero"
	}
	var ones = []string{"", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine",
		"ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen", "eighteen", "nineteen"}
	var tens = []string{"", "", "twenty", "thirty", "forty", "fifty", "sixty", "seventy", "eighty", "ninety"}

	var words func(int) string
	words = func(n int) string {
		if n < 0 {
			return "minus " + words(-n)
		}
		if n < 20 {
			return ones[n]
		}
		if n < 100 {
			w := tens[n/10]
			if n%10 != 0 {
				w += " " + ones[n%10]
			}
			return w
		}
		if n < 1000 {
			w := ones[n/100] + " hundred"
			if n%100 != 0 {
				w += " and " + words(n%100)
			}
			return w
		}
		if n < 1000000 {
			w := words(n/1000) + " thousand"
			if n%1000 != 0 {
				w += " " + words(n%1000)
			}
			return w
		}
		if n < 1000000000 {
			w := words(n/1000000) + " million"
			if n%1000000 != 0 {
				w += " " + words(n%1000000)
			}
			return w
		}
		if n < 1000000000000 {
			w := words(n/1000000000) + " billion"
			if n%1000000000 != 0 {
				w += " " + words(n%1000000000)
			}
			return w
		}
		return strconv.Itoa(n)
	}
	result := words(n)
	if upper {
		return strings.ToUpper(result)
	}
	return result
}

func applyGroupingSeparator(s string, sep string, size int) string {
	// Insert separator from right to left every 'size' digits
	if size <= 0 || sep == "" {
		return s
	}
	runes := []rune(s)
	var result []rune
	for i, j := len(runes)-1, 0; i >= 0; i, j = i-1, j+1 {
		if j > 0 && j%size == 0 {
			// Prepend separator (reversed, will be re-reversed)
			sepRunes := []rune(sep)
			for k := len(sepRunes) - 1; k >= 0; k-- {
				result = append(result, sepRunes[k])
			}
		}
		result = append(result, runes[i])
	}
	// Reverse
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return string(result)
}

func toLowerAlpha(n int) string {
	if n <= 0 {
		return strconv.Itoa(n)
	}
	var buf []byte
	for n > 0 {
		n--
		buf = append([]byte{byte('a' + n%26)}, buf...)
		n /= 26
	}
	return string(buf)
}

func toUpperAlpha(n int) string {
	return strings.ToUpper(toLowerAlpha(n))
}

func toRoman(n int) string {
	if n <= 0 || n >= 4000 {
		return strconv.Itoa(n)
	}
	vals := []struct {
		val int
		sym string
	}{
		{1000, "M"}, {900, "CM"}, {500, "D"}, {400, "CD"},
		{100, "C"}, {90, "XC"}, {50, "L"}, {40, "XL"},
		{10, "X"}, {9, "IX"}, {5, "V"}, {4, "IV"}, {1, "I"},
	}
	var buf strings.Builder
	for _, v := range vals {
		for n >= v.val {
			buf.WriteString(v.sym)
			n -= v.val
		}
	}
	return buf.String()
}
