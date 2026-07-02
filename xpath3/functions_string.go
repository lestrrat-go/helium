package xpath3

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"regexp/syntax"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/dlclark/regexp2"
	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"
)

func init() {
	registerFn("string", 0, 1, fnString)
	registerFn("codepoints-to-string", 1, 1, fnCodepointsToString)
	registerFn("string-to-codepoints", 1, 1, fnStringToCodepoints)
	registerFn("compare", 2, 3, fnCompare)
	registerFn("codepoint-equal", 2, 2, fnCodepointEqual)
	registerFn("concat", 2, -1, fnConcat)
	registerFn("string-join", 1, 2, fnStringJoin)
	registerFn("substring", 2, 3, fnSubstring)
	registerFn("string-length", 0, 1, fnStringLength)
	registerFn("normalize-space", 0, 1, fnNormalizeSpace)
	registerFn("normalize-unicode", 1, 2, fnNormalizeUnicode)
	registerFn("upper-case", 1, 1, fnUpperCase)
	registerFn("lower-case", 1, 1, fnLowerCase)
	registerFn("translate", 3, 3, fnTranslate)
	registerFn("contains", 2, 3, fnContains)
	registerFn("starts-with", 2, 3, fnStartsWith)
	registerFn("ends-with", 2, 3, fnEndsWith)
	registerFn("substring-before", 2, 3, fnSubstringBefore)
	registerFn("substring-after", 2, 3, fnSubstringAfter)
	registerFn("matches", 2, 3, fnMatches)
	registerFn("replace", 3, 4, fnReplace)
	registerFn("tokenize", 1, 3, fnTokenize)
	registerFn("analyze-string", 2, 3, fnAnalyzeString)
	registerFn("contains-token", 2, 3, fnContainsToken)
	registerFn("collation-key", 1, 2, fnCollationKey)
}

func fnString(ctx context.Context, args []Sequence) (Sequence, error) {
	if len(args) == 0 {
		fc := getFnContext(ctx)
		if fc == nil || (fc.contextItem == nil && fc.node == nil) {
			return nil, &XPathError{Code: errCodeXPDY0002, Message: errMsgContextItemAbsent}
		}
		s, ok := fc.contextStringValue()
		if !ok {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "context item has no string value"}
		}
		return SingleString(s), nil
	}
	if seqLen(args[0]) == 0 {
		return SingleString(""), nil
	}
	if seqLen(args[0]) > 1 {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "fn:string requires a single item, got sequence of length > 1"}
	}
	item := args[0].Get(0)
	// fn:string does not accept function items, maps, or arrays
	switch item.(type) {
	case FunctionItem, MapItem, ArrayItem:
		return nil, &XPathError{Code: errCodeFOTY0014, Message: fmt.Sprintf("fn:string: cannot get string value of %T", item)}
	}
	// For node items, fn:string returns the dm:string-value (text content),
	// NOT the typed/atomized value. This preserves lexical forms like "003"
	// even when the node is typed as xs:integer.
	if ni, ok := item.(NodeItem); ok {
		return SingleString(ixpath.StringValue(ni.Node)), nil
	}
	a, err := AtomizeItem(item)
	if err != nil {
		return nil, err
	}
	s, _ := atomicToString(a)
	return SingleString(s), nil
}

func fnCodepointsToString(ctx context.Context, args []Sequence) (Sequence, error) {
	seq := args[0]

	// Check whether XML 1.1 characters are allowed (e.g. XSLT 3.0 context).
	xml11 := false
	if ec := getFnContext(ctx); ec != nil {
		xml11 = ec.allowXML11Chars
	}
	isValid := isValidXMLCodepoint
	if xml11 {
		isValid = isValidXML11Codepoint
	}

	// Fast path: singleton integer (common in unicode-90 where each codepoint
	// is mapped individually via codepoints-to-string(.))
	if seqLen(seq) == 1 {
		cp, err := itemToCodepoint(seq.Get(0))
		if err != nil {
			return nil, err
		}
		if !isValid(cp) {
			return nil, &XPathError{Code: lexicon.ErrFOCH0001, Message: fmt.Sprintf("invalid XML character [x%X]", cp)}
		}
		return SingleString(string(rune(cp))), nil
	}

	var b strings.Builder
	for item := range seqItems(seq) {
		cp, err := itemToCodepoint(item)
		if err != nil {
			return nil, err
		}
		if !isValid(cp) {
			return nil, &XPathError{Code: lexicon.ErrFOCH0001, Message: fmt.Sprintf("invalid XML character [x%X]", cp)}
		}
		b.WriteRune(rune(cp))
	}
	return SingleString(b.String()), nil
}

// itemToCodepoint extracts an integer codepoint from an item, avoiding
// expensive big.Float conversion when the value is already a *big.Int.
func itemToCodepoint(item Item) (int, error) {
	a, err := AtomizeItem(item)
	if err != nil {
		return 0, err
	}
	if a.TypeName == TypeUntypedAtomic {
		a, err = CastAtomic(a, TypeInteger)
		if err != nil {
			return 0, err
		}
	}
	// Fast path: extract int64 directly from *big.Int (avoids big.Float allocation)
	if isIntegerDerived(a.TypeName) {
		if n, ok := a.Value.(*big.Int); ok {
			// A value beyond int64 range cannot be a valid XML codepoint;
			// Int64() would silently wrap mod 2^64, so reject it here.
			if !n.IsInt64() {
				return 0, &XPathError{Code: lexicon.ErrFOCH0001, Message: fmt.Sprintf("invalid XML character [%s]", n.String())}
			}
			v := n.Int64()
			// Validate the codepoint range BEFORE the int conversion: on 32-bit
			// platforms int(v) for an out-of-range v wraps/truncates, which would
			// let a spurious value pass the later isValid check.
			if v < 0 || v > utf8.MaxRune {
				return 0, &XPathError{Code: lexicon.ErrFOCH0001, Message: fmt.Sprintf("invalid XML character [%d]", v)}
			}
			return int(v), nil
		}
	}
	// xs:decimal is stored as *big.Rat. Check integrality exactly here rather
	// than via float64: a high-precision fractional decimal such as
	// 65.000000000000000000000000001 rounds to an integer float64 and would
	// otherwise slip past the fractional check below.
	if r, ok := a.Value.(*big.Rat); ok {
		if !r.IsInt() {
			return 0, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("codepoints-to-string: non-integer codepoint %s", r.RatString())}
		}
		n := r.Num()
		if !n.IsInt64() {
			return 0, &XPathError{Code: lexicon.ErrFOCH0001, Message: fmt.Sprintf("invalid XML character [%s]", n.String())}
		}
		v := n.Int64()
		if v < 0 || v > utf8.MaxRune {
			return 0, &XPathError{Code: lexicon.ErrFOCH0001, Message: fmt.Sprintf("invalid XML character [%d]", v)}
		}
		return int(v), nil
	}
	// Non-integer fallback: a fractional value is not a valid codepoint.
	// Reject it as a type error, but keep accepting integer-valued floats
	// (e.g. 65.0) so arithmetic-derived integer values still work.
	f := a.ToFloat64()
	if math.IsNaN(f) || math.IsInf(f, 0) || f != math.Trunc(f) {
		return 0, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("codepoints-to-string: non-integer codepoint %v", f)}
	}
	// Validate the codepoint range BEFORE the int conversion: converting an
	// out-of-range float64 to int is implementation-defined in Go (it can wrap
	// or overflow, especially on 32-bit platforms), which would let a spurious
	// value slip past the later isValid check.
	if f < 0 || f > utf8.MaxRune {
		return 0, &XPathError{Code: lexicon.ErrFOCH0001, Message: fmt.Sprintf("invalid XML character [%v]", f)}
	}
	return int(f), nil
}

// isValidXMLCodepoint returns true if the codepoint is a valid XML character.
// Per XML 1.0 §2.2: #x9 | #xA | #xD | [#x20-#xD7FF] | [#xE000-#xFFFD] | [#x10000-#x10FFFF]
func isValidXMLCodepoint(cp int) bool {
	if cp == 0x9 || cp == 0xA || cp == 0xD {
		return true
	}
	if cp >= 0x20 && cp <= 0xD7FF {
		return true
	}
	if cp >= 0xE000 && cp <= 0xFFFD {
		return true
	}
	if cp >= 0x10000 && cp <= 0x10FFFF {
		return true
	}
	return false
}

// isValidXML11Codepoint extends the XML 1.0 check to also accept XML 1.1
// restricted characters (0x01-0x1F except 0x00). XSLT 3.0 processors need
// these for features like xml-to-json with escaped="1".
func isValidXML11Codepoint(cp int) bool {
	if cp >= 0x1 && cp <= 0xD7FF {
		return true
	}
	if cp >= 0xE000 && cp <= 0xFFFD {
		return true
	}
	if cp >= 0x10000 && cp <= 0x10FFFF {
		return true
	}
	return false
}

func fnStringToCodepoints(ctx context.Context, args []Sequence) (Sequence, error) {
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	if s == "" {
		return validNilSequence, nil
	}
	// Build the codepoint sequence one item per character, charging the op-limit
	// and honoring context cancellation before each append (and capping the
	// length at maxNodes). A huge input string would otherwise materialize an
	// unbounded item sequence in one shot, ignoring both budgets.
	ec := getFnContext(ctx)
	maxNodes := fnMaxNodes(ec)
	var result ItemSlice
	for _, r := range s {
		if err := fnCountOp(ctx, ec); err != nil {
			return nil, err
		}
		if len(result) >= maxNodes {
			return nil, ErrNodeSetLimit
		}
		result = append(result, AtomicValue{TypeName: TypeInteger, Value: big.NewInt(int64(r))})
	}
	return result, nil
}

func fnCompare(ctx context.Context, args []Sequence) (Sequence, error) {
	coll, err := getCollation(ctx, args, 2)
	if err != nil {
		return nil, err
	}
	if seqLen(args[0]) == 0 || seqLen(args[1]) == 0 {
		return validNilSequence, nil
	}
	s1, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	s2, err := coerceArgToString(args[1])
	if err != nil {
		return nil, err
	}
	cmp := coll.compare(s1, s2)
	return SingleInteger(int64(cmp)), nil
}

func fnCodepointEqual(_ context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 || seqLen(args[1]) == 0 {
		return validNilSequence, nil
	}
	s1, err := coerceArgToStringOpt(args[0])
	if err != nil {
		return nil, err
	}
	s2, err := coerceArgToStringOpt(args[1])
	if err != nil {
		return nil, err
	}
	return SingleBoolean(s1 == s2), nil
}

func fnConcat(_ context.Context, args []Sequence) (Sequence, error) {
	var b strings.Builder
	for _, arg := range args {
		s, err := seqToStringErr(arg)
		if err != nil {
			return nil, err
		}
		b.WriteString(s)
	}
	return SingleString(b.String()), nil
}

func fnStringJoin(ctx context.Context, args []Sequence) (Sequence, error) {
	sep := ""
	if len(args) > 1 {
		if getFnContext(ctx).xpath10CompatMode() {
			// The separator's expected type is xs:string, so XPath 1.0 compatibility
			// mode converts it with fn:string applied to its first item.
			sv, err := xpath10CompatStringItem(args[1])
			if err != nil {
				return nil, err
			}
			sep, _ = sv.Value.(string)
		} else {
			var err error
			sep, err = coerceArgToStringRequired(args[1])
			if err != nil {
				return nil, err
			}
		}
	}
	// Atomize the entire sequence (expands list types to multiple items).
	atoms, err := AtomizeSequence(args[0])
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	for i, a := range atoms {
		if i > 0 && sep != "" {
			b.WriteString(sep)
		}
		s, err := atomicToString(a)
		if err != nil {
			return nil, err
		}
		b.WriteString(s)
	}
	return SingleString(b.String()), nil
}

func fnSubstring(_ context.Context, args []Sequence) (Sequence, error) {
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	startPos, err := coerceArgToDoubleRequired(args[1])
	if err != nil {
		return nil, err
	}
	runes := []rune(s)

	// XPath round
	rStart := math.Floor(startPos + 0.5)

	if len(args) == 3 {
		length, err := coerceArgToDoubleRequired(args[2])
		if err != nil {
			return nil, err
		}
		rLength := math.Floor(length + 0.5)
		var b strings.Builder
		for i, r := range runes {
			p := float64(i + 1)
			if p >= rStart && p < rStart+rLength {
				b.WriteRune(r)
			}
		}
		return SingleString(b.String()), nil
	}

	if math.IsNaN(rStart) || math.IsInf(rStart, 1) {
		return SingleString(""), nil
	}
	var b strings.Builder
	for i, r := range runes {
		if float64(i+1) >= rStart {
			b.WriteRune(r)
		}
	}
	return SingleString(b.String()), nil
}

func fnStringLength(ctx context.Context, args []Sequence) (Sequence, error) {
	var s string
	if len(args) == 0 {
		fc := getFnContext(ctx)
		if fc == nil {
			return nil, &XPathError{Code: errCodeXPDY0002, Message: "string-length: context item is absent"}
		}
		var ok bool
		s, ok = fc.contextStringValue()
		if !ok {
			return nil, &XPathError{Code: errCodeXPDY0002, Message: "string-length: context item is absent"}
		}
	} else {
		if seqLen(args[0]) > 1 {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("string-length: expected single item, got sequence of length %d", seqLen(args[0]))}
		}
		var err error
		s, err = seqToStringErr(args[0])
		if err != nil {
			return nil, err
		}
	}
	return SingleInteger(int64(len([]rune(s)))), nil
}

func fnNormalizeSpace(ctx context.Context, args []Sequence) (Sequence, error) {
	var s string
	if len(args) == 0 {
		fc := getFnContext(ctx)
		if fc == nil || (fc.contextItem == nil && fc.node == nil) {
			return nil, &XPathError{Code: errCodeXPDY0002, Message: errMsgContextItemAbsent}
		}
		sv, ok := fc.contextStringValue()
		if !ok {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "context item has no string value"}
		}
		s = sv
	} else {
		var err error
		s, err = coerceArgToString(args[0])
		if err != nil {
			return nil, err
		}
	}
	return SingleString(strings.Join(strings.Fields(s), " ")), nil
}

func fnNormalizeUnicode(_ context.Context, args []Sequence) (Sequence, error) {
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}

	formName := "NFC" // default
	if len(args) > 1 {
		form, err := coerceArgToStringRequired(args[1])
		if err != nil {
			return nil, err
		}
		formName = strings.TrimSpace(strings.ToUpper(form))
		if formName == "" {
			// Empty form string means return input unchanged
			return SingleString(s), nil
		}
	}

	if s == "" {
		return SingleString(""), nil
	}

	var nf norm.Form
	switch formName {
	case "NFC":
		nf = norm.NFC
	case "NFD":
		nf = norm.NFD
	case "NFKC":
		nf = norm.NFKC
	case "NFKD":
		nf = norm.NFKD
	case "FULLY-NORMALIZED":
		// W3C Charmod Normalization: NFC + if the result starts with a
		// composing character, prepend a space. A composing character is
		// one that can be consumed by NFC composition with a preceding
		// starter. We detect this by prepending a known starter and
		// checking whether NFC composition changes the pair.
		result := norm.NFC.String(s)
		if len(result) > 0 {
			r, _ := utf8.DecodeRuneInString(result)
			if isComposingCharacter(r) {
				result = " " + result
			}
		}
		return SingleString(result), nil
	default:
		return nil, &XPathError{Code: errCodeFOCH0003, Message: fmt.Sprintf("unsupported normalization form: %s", formName)}
	}

	return SingleString(nf.String(s)), nil
}

// isComposingCharacter returns true if r is a character that could compose
// with a preceding character under NFC. This includes characters with CCC > 0
// and characters that appear as the trailing element of a canonical composition.
// We use norm.NFC.BoundaryBefore: a rune that does NOT start a new boundary
// can compose with a preceding character and is therefore "composing".
func isComposingCharacter(r rune) bool {
	p := norm.NFC.PropertiesString(string(r))
	return !p.BoundaryBefore()
}

// xpathUpperCaser and xpathLowerCaser use golang.org/x/text/cases with
// language.Und for locale-independent full Unicode case mapping (handles
// multi-character expansions like ß→SS, İ→i̇).
var (
	xpathUpperCaser = cases.Upper(language.Und)
	xpathLowerCaser = cases.Lower(language.Und)
)

func fnUpperCase(_ context.Context, args []Sequence) (Sequence, error) {
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	return SingleString(xpathUpperCaser.String(s)), nil
}

func fnLowerCase(_ context.Context, args []Sequence) (Sequence, error) {
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	return SingleString(xpathLowerCaser.String(s)), nil
}

func fnTranslate(_ context.Context, args []Sequence) (Sequence, error) {
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	fromStr, err := coerceArgToStringRequired(args[1])
	if err != nil {
		return nil, err
	}
	toStr, err := coerceArgToStringRequired(args[2])
	if err != nil {
		return nil, err
	}
	from := []rune(fromStr)
	to := []rune(toStr)

	mapping := make(map[rune]rune, len(from))
	remove := make(map[rune]bool)
	for i, r := range from {
		if _, exists := mapping[r]; exists {
			continue
		}
		if remove[r] {
			continue
		}
		if i < len(to) {
			mapping[r] = to[i]
		} else {
			remove[r] = true
		}
	}

	var b strings.Builder
	for _, r := range s {
		if remove[r] {
			continue
		}
		if rep, ok := mapping[r]; ok {
			b.WriteRune(rep)
		} else {
			b.WriteRune(r)
		}
	}
	return SingleString(b.String()), nil
}

// collationStringPair resolves the collation (arg index 2) and coerces the
// first two arguments to strings, in that order. Shared by the collation-aware
// two-string string functions (contains, starts-with, ends-with,
// substring-before, substring-after).
func collationStringPair(ctx context.Context, args []Sequence) (*collationImpl, string, string, error) {
	coll, err := getCollation(ctx, args, 2)
	if err != nil {
		return nil, "", "", err
	}
	s1, err := coerceArgToString(args[0])
	if err != nil {
		return nil, "", "", err
	}
	s2, err := coerceArgToString(args[1])
	if err != nil {
		return nil, "", "", err
	}
	return coll, s1, s2, nil
}

func fnContains(ctx context.Context, args []Sequence) (Sequence, error) {
	coll, s, sub, err := collationStringPair(ctx, args)
	if err != nil {
		return nil, err
	}
	if sub == "" {
		return SingleBoolean(true), nil
	}
	pos, _ := coll.indexOf(s, sub)
	return SingleBoolean(pos >= 0), nil
}

func fnStartsWith(ctx context.Context, args []Sequence) (Sequence, error) {
	coll, s, prefix, err := collationStringPair(ctx, args)
	if err != nil {
		return nil, err
	}
	if prefix == "" {
		return SingleBoolean(true), nil
	}
	ok, _ := coll.hasPrefix(s, prefix)
	return SingleBoolean(ok), nil
}

func fnEndsWith(ctx context.Context, args []Sequence) (Sequence, error) {
	coll, s, suffix, err := collationStringPair(ctx, args)
	if err != nil {
		return nil, err
	}
	if suffix == "" {
		return SingleBoolean(true), nil
	}
	ok, _ := coll.hasSuffix(s, suffix)
	return SingleBoolean(ok), nil
}

func fnSubstringBefore(ctx context.Context, args []Sequence) (Sequence, error) {
	coll, s, sep, err := collationStringPair(ctx, args)
	if err != nil {
		return nil, err
	}
	if sep == "" {
		return SingleString(""), nil
	}
	pos, _ := coll.indexOf(s, sep)
	if pos < 0 {
		return SingleString(""), nil
	}
	return SingleString(s[:pos]), nil
}

func fnSubstringAfter(ctx context.Context, args []Sequence) (Sequence, error) {
	coll, s, sep, err := collationStringPair(ctx, args)
	if err != nil {
		return nil, err
	}
	if sep == "" {
		return SingleString(s), nil
	}
	pos, matchLen := coll.indexOf(s, sep)
	if pos < 0 {
		return SingleString(""), nil
	}
	return SingleString(s[pos+matchLen:]), nil
}

func fnMatches(_ context.Context, args []Sequence) (Sequence, error) {
	// Per XPath spec, xs:string? argument: empty sequence is treated as "".
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	if seqLen(args[1]) == 0 {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "fn:matches pattern must not be empty sequence"}
	}
	pattern, err := coerceArgToStringRequired(args[1])
	if err != nil {
		return nil, err
	}
	flags := ""
	if len(args) > 2 {
		if seqLen(args[2]) == 0 {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "fn:matches flags must not be empty sequence"}
		}
		flags, err = coerceArgToStringRequired(args[2])
		if err != nil {
			return nil, err
		}
	}
	if shouldUseXPathEmptyLineMatcher(pattern, flags) {
		return SingleBoolean(matchesXPathEmptyLine(s)), nil
	}
	re, err := compileXPathRegex(pattern, flags)
	if err != nil {
		return nil, err
	}
	// Fast path: simple \p{Name}/\P{Name} pattern against single-rune input
	if re.isSimple && utf8.RuneCountInString(s) == 1 {
		r, _ := utf8.DecodeRuneInString(s)
		match := unicode.Is(re.unicodeTable, r)
		if re.negated {
			match = !match
		}
		return SingleBoolean(match), nil
	}
	ok, err := re.MatchString(s)
	if err != nil {
		return nil, &XPathError{Code: errCodeFORX0002, Message: fmt.Sprintf("regex match failed: %v", err)}
	}
	return SingleBoolean(ok), nil
}

func fnCollationKey(ctx context.Context, args []Sequence) (Sequence, error) {
	coll, err := getCollation(ctx, args, 1)
	if err != nil {
		return nil, err
	}
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	if coll.key == nil {
		return SingleAtomic(AtomicValue{TypeName: TypeBase64Binary, Value: []byte(s)}), nil
	}
	return SingleAtomic(AtomicValue{TypeName: TypeBase64Binary, Value: coll.key(s)}), nil
}

func fnReplace(_ context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return SingleString(""), nil // input is xs:string? — empty yields ""
	}
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	if seqLen(args[1]) == 0 {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "fn:replace pattern must not be empty sequence"}
	}
	pattern, err := coerceArgToStringRequired(args[1])
	if err != nil {
		return nil, err
	}
	if seqLen(args[2]) == 0 {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "fn:replace replacement must not be empty sequence"}
	}
	replacement, err := coerceArgToStringRequired(args[2])
	if err != nil {
		return nil, err
	}
	flags := ""
	if len(args) > 3 {
		if seqLen(args[3]) == 0 {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "fn:replace flags must not be empty sequence"}
		}
		flags, err = coerceArgToStringRequired(args[3])
		if err != nil {
			return nil, err
		}
	}

	isLiteral := strings.Contains(flags, "q")

	re, err := compileXPathRegex(pattern, flags)
	if err != nil {
		return nil, err
	}

	// Fast path: simple \p{Name}/\P{Name} with empty replacement — filter runes directly
	if re.isSimple && replacement == "" {
		var b strings.Builder
		for _, r := range s {
			match := unicode.Is(re.unicodeTable, r)
			if re.negated {
				match = !match
			}
			if !match {
				b.WriteRune(r)
			}
		}
		return SingleString(b.String()), nil
	}

	// XPath spec: error if pattern matches empty string
	matchesEmpty, err := re.MatchString("")
	if err != nil {
		return nil, &XPathError{Code: errCodeFORX0002, Message: fmt.Sprintf("regex match failed: %v", err)}
	}
	if matchesEmpty {
		return nil, &XPathError{Code: errCodeFORX0003, Message: "replacement pattern matches zero-length string"}
	}

	var goRepl string
	if isLiteral {
		// With q flag, replacement is literal — escape Go's special chars
		goRepl = strings.ReplaceAll(replacement, "$", "$$")
	} else {
		// Validate and translate XPath replacement string to Go syntax.
		goRepl, err = translateXPathReplacement(replacement, re.NumSubexp())
		if err != nil {
			return nil, err
		}
	}

	replaced, err := re.ReplaceAllString(s, goRepl)
	if err != nil {
		return nil, &XPathError{Code: errCodeFORX0002, Message: fmt.Sprintf("regex replace failed: %v", err)}
	}
	return SingleString(replaced), nil
}

// translateXPathReplacement converts an XPath replacement string to Go regexp syntax.
// XPath uses: $N for backrefs, \$ for literal $, \\ for literal \.
// Go uses:    $N for backrefs, $$ for literal $.
// numGroups is the number of capture groups in the pattern — $N consumes the
// maximum number of digits that form a valid group reference (≤ numGroups).
func translateXPathReplacement(repl string, numGroups int) (string, error) {
	var b strings.Builder
	for i := 0; i < len(repl); i++ {
		ch := repl[i]
		switch ch {
		case '\\':
			if i+1 >= len(repl) {
				return "", &XPathError{Code: errCodeFORX0004, Message: "invalid replacement string: trailing backslash"}
			}
			next := repl[i+1]
			switch next {
			case '\\':
				b.WriteByte('\\')
				i++
			case '$':
				b.WriteString("$$") // Go's literal $
				i++
			default:
				return "", &XPathError{Code: errCodeFORX0004, Message: fmt.Sprintf("invalid replacement string: \\%c", next)}
			}
		case '$':
			if i+1 >= len(repl) || repl[i+1] < '0' || repl[i+1] > '9' {
				return "", &XPathError{Code: errCodeFORX0004, Message: "invalid replacement string: $ not followed by digit"}
			}
			// Collect digits after $, but only consume as many as form a
			// valid group number (≤ numGroups). Remaining digits are literal.
			i++
			start := i
			num := 0
			validEnd := start // end of the longest valid group number
			for i < len(repl) && repl[i] >= '0' && repl[i] <= '9' {
				num = num*10 + int(repl[i]-'0')
				i++
				if num > 0 && num <= numGroups {
					validEnd = i
				}
			}
			// $0 always refers to the whole match.
			if validEnd == start && i > start && repl[start] == '0' {
				validEnd = start + 1
			}
			if validEnd == start {
				// No valid group number found — group number exceeds numGroups.
				// Per XPath spec, references to non-existent groups are replaced
				// with empty string. Write nothing for the group reference.
			} else {
				// Write the valid group reference
				b.WriteString("${")
				b.WriteString(repl[start:validEnd])
				b.WriteByte('}')
				// Write remaining digits as literal text
				for k := validEnd; k < i; k++ {
					b.WriteByte(repl[k])
				}
			}
			i-- // outer loop will i++
		default:
			b.WriteByte(ch)
		}
	}
	return b.String(), nil
}

func fnTokenize(_ context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return validNilSequence, nil // input is xs:string? — empty yields empty
	}
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	if s == "" {
		return validNilSequence, nil
	}

	// 1-arg form: normalize XML whitespace (#x20, #x9, #xA, #xD), then split
	if len(args) == 1 {
		tokens := splitXMLWhitespace(s)
		result := make(ItemSlice, len(tokens))
		for i, t := range tokens {
			result[i] = AtomicValue{TypeName: TypeString, Value: t}
		}
		return result, nil
	}

	if seqLen(args[1]) == 0 {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "fn:tokenize pattern must not be empty sequence"}
	}
	pattern, err := coerceArgToStringRequired(args[1])
	if err != nil {
		return nil, err
	}
	flags := ""
	if len(args) > 2 {
		if seqLen(args[2]) == 0 {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "fn:tokenize flags must not be empty sequence"}
		}
		flags, err = coerceArgToStringRequired(args[2])
		if err != nil {
			return nil, err
		}
	}
	re, err := compileXPathRegex(pattern, flags)
	if err != nil {
		return nil, err
	}

	// XPath spec: error if pattern matches zero-length string
	matchesEmpty, err := re.MatchString("")
	if err != nil {
		return nil, &XPathError{Code: errCodeFORX0002, Message: fmt.Sprintf("regex match failed: %v", err)}
	}
	if matchesEmpty {
		return nil, &XPathError{Code: errCodeFORX0003, Message: "tokenize pattern matches zero-length string"}
	}

	parts, err := re.Split(s, -1)
	if err != nil {
		return nil, &XPathError{Code: errCodeFORX0002, Message: fmt.Sprintf("regex split failed: %v", err)}
	}
	result := make(ItemSlice, len(parts))
	for i, p := range parts {
		result[i] = AtomicValue{TypeName: TypeString, Value: p}
	}
	return result, nil
}

// splitXMLWhitespace splits s on XML whitespace (#x20, #x9, #xA, #xD),
// stripping leading/trailing whitespace and collapsing runs. Unlike
// strings.Fields, it does NOT treat Unicode whitespace (e.g. \u00A0) as
// separators.
func splitXMLWhitespace(s string) []string {
	var tokens []string
	start := -1
	for i, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if start >= 0 {
				tokens = append(tokens, s[start:i])
				start = -1
			}
		} else {
			if start < 0 {
				start = i
			}
		}
	}
	if start >= 0 {
		tokens = append(tokens, s[start:])
	}
	return tokens
}

func shouldUseXPathEmptyLineMatcher(pattern, flags string) bool {
	if strings.ContainsRune(flags, 'q') || !strings.ContainsRune(flags, 'm') {
		return false
	}
	if strings.ContainsRune(flags, 'x') {
		pattern = stripFreeSpacing(pattern)
	}
	return pattern == "^$"
}

func matchesXPathEmptyLine(s string) bool {
	return s == "" || strings.HasPrefix(s, "\n") || strings.Contains(s, "\n\n")
}

func fnAnalyzeString(ctx context.Context, args []Sequence) (Sequence, error) {
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	pattern, err := coerceArgToStringRequired(args[1])
	if err != nil {
		return nil, err
	}
	flags := ""
	if len(args) > 2 {
		flags, err = coerceArgToString(args[2])
		if err != nil {
			return nil, err
		}
	}
	re, err := compileXPathRegex(pattern, flags)
	if err != nil {
		return nil, err
	}
	matchesEmpty, err := re.MatchString("")
	if err != nil {
		return nil, &XPathError{Code: errCodeFORX0002, Message: fmt.Sprintf("regex match failed: %v", err)}
	}
	if matchesEmpty {
		return nil, &XPathError{Code: errCodeFORX0003, Message: "analyze-string pattern matches zero-length string"}
	}

	doc := helium.NewDefaultDocument()
	root, err := createAnalyzeStringElement(doc, "analyze-string-result")
	if err != nil {
		return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("analyze-string: failed to build result: %v", err)}
	}
	if err := root.DeclareNamespace("fn", NSFn); err != nil {
		return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("analyze-string: failed to build result: %v", err)}
	}
	if err := doc.SetDocumentElement(root); err != nil {
		return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("analyze-string: failed to build result: %v", err)}
	}

	// Stream the matches one at a time instead of materializing every match up
	// front. An input with a huge number of matches would otherwise allocate the
	// full match slice (an O(matches) up-front cost) and build the whole result
	// tree before the op budget or a cancellation could intervene. When an op
	// budget is in force, cap the enumeration at remaining+1 matches so the
	// callback observes the one match that exhausts the budget and rejects
	// without producing the rest; an unbounded budget streams uncapped.
	ec := getFnContext(ctx)
	findLimit := -1
	if remaining, bounded := fnRemainingOps(ec); bounded {
		if n := remaining + 1; n > 0 {
			findLimit = n
		}
	}
	pos := 0
	var buildErr error
	buildFail := func(err error) bool {
		buildErr = &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("analyze-string: failed to build result: %v", err)}
		return false
	}
	iterErr := re.eachStringSubmatchIndex(s, findLimit, func(m []int) bool {
		// Charge the op-limit and honor context cancellation BEFORE building any
		// result nodes for this match: an over-budget or cancelled run stops here
		// instead of emitting an element first, and the full match slice is never
		// materialized up front.
		if err := fnCountOp(ctx, ec); err != nil {
			buildErr = err
			return false
		}
		start, end := m[0], m[1]
		if start > pos {
			if err := appendAnalyzeStringTextElement(doc, root, "non-match", s[pos:start]); err != nil {
				return buildFail(err)
			}
		}
		matchElem, err := createAnalyzeStringElement(doc, "match")
		if err != nil {
			return buildFail(err)
		}
		// Check for groups
		if len(m) > 2 {
			groupPos := start
			for g := 1; g < len(m)/2; g++ {
				gs, ge := m[2*g], m[2*g+1]
				if gs < 0 {
					continue
				}
				if gs > groupPos {
					if err := matchElem.AppendText([]byte(s[groupPos:gs])); err != nil {
						return buildFail(err)
					}
				}
				groupElem, err := createAnalyzeStringElement(doc, "group")
				if err != nil {
					return buildFail(err)
				}
				_ = groupElem.SetLiteralAttribute("nr", fmt.Sprintf("%d", g))
				if err := groupElem.AppendText([]byte(s[gs:ge])); err != nil {
					return buildFail(err)
				}
				if err := matchElem.AddChild(groupElem); err != nil {
					return buildFail(err)
				}
				groupPos = ge
			}
			if groupPos < end {
				if err := matchElem.AppendText([]byte(s[groupPos:end])); err != nil {
					return buildFail(err)
				}
			}
		} else {
			if err := matchElem.AppendText([]byte(s[start:end])); err != nil {
				return buildFail(err)
			}
		}
		if err := root.AddChild(matchElem); err != nil {
			return buildFail(err)
		}
		pos = end
		return true
	})
	if buildErr != nil {
		return nil, buildErr
	}
	if iterErr != nil {
		// A leading-context pattern that cannot stream is bounded to xpath3's
		// full-context allocation ceiling; surface that resource condition as-is
		// so errors.Is(err, ErrRegexMatchLimit) keeps working. Any other engine
		// error maps to FORX0002 like the former FindAllStringSubmatchIndex path.
		if errors.Is(iterErr, ErrRegexMatchLimit) {
			return nil, iterErr
		}
		return nil, &XPathError{Code: errCodeFORX0002, Message: fmt.Sprintf("regex match failed: %v", iterErr)}
	}
	if pos < len(s) {
		if err := appendAnalyzeStringTextElement(doc, root, "non-match", s[pos:]); err != nil {
			return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("analyze-string: failed to build result: %v", err)}
		}
	}

	return ItemSlice{NodeItem{Node: root}}, nil
}

func createAnalyzeStringElement(doc *helium.Document, localName string) (*helium.Element, error) {
	elem := doc.CreateElement(localName)
	if err := elem.SetActiveNamespace("fn", NSFn); err != nil {
		return nil, err
	}
	return elem, nil
}

func appendAnalyzeStringTextElement(doc *helium.Document, parent *helium.Element, localName, text string) error {
	elem, err := createAnalyzeStringElement(doc, localName)
	if err != nil {
		return err
	}
	if text != "" {
		if err := elem.AppendText([]byte(text)); err != nil {
			return err
		}
	}
	return parent.AddChild(elem)
}

// compileXPathRegex compiles an XPath regex pattern with flags.
// Maps XPath flags (i,m,s,x) to Go regexp equivalents.
// Translates XPath/XML Schema regex features to Go-compatible patterns.
type compiledXPathRegex struct {
	std          *regexp.Regexp
	backtrack    *regexp2.Regexp
	numGroups    int
	unicodeTable *unicode.RangeTable // non-nil for simple \p{Name} or \P{Name} patterns
	negated      bool                // true when the simple pattern is \P{...}
	isSimple     bool                // true when unicodeTable is usable for single-rune fast paths
	// stdNeedsFullContext is true when the std (RE2) pattern contains a
	// leading-context zero-width assertion (^, \A, \b, \B, or multi-line ^)
	// whose truth depends on input to the LEFT of the current scan position.
	// Such patterns cannot be streamed by advancing an offset over s[pos:]
	// (slicing makes those assertions see a false "start of input"), so
	// eachStdSubmatchIndex matches them against the whole string via a bounded
	// FindAllStringSubmatchIndex instead. The pattern stays on Go's RE2 engine
	// (no backtracking-ReDoS regression), and the caller-supplied limit bounds
	// the up-front materialization to the resource cap rather than the match
	// count.
	stdNeedsFullContext bool
}

// patternNeedsFullContext reports whether the RE2 pattern contains a
// leading-context zero-width assertion. Trailing assertions ($, \z, multi-line
// $) are unaffected by left-truncation — the slice end coincides with the
// string end — so they do NOT require the full-context fallback.
func patternNeedsFullContext(pattern string) bool {
	parsed, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		// Be conservative: if we cannot analyze it, assume it needs full
		// context so correctness never depends on the analysis succeeding.
		return true
	}
	return regexpHasLeadingContextAssertion(parsed)
}

func regexpHasLeadingContextAssertion(re *syntax.Regexp) bool {
	switch re.Op {
	case syntax.OpBeginText, syntax.OpBeginLine, syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		return true
	}
	return slices.ContainsFunc(re.Sub, regexpHasLeadingContextAssertion)
}

type xpathRegexCacheKey struct {
	pattern string
	flags   string
}

var compiledXPathRegexCache = newRegexLRUCache(xpathRegexCacheCapacity)

// DefaultRegexMatchTimeout bounds how long the backtracking regex engine
// will spend on a single match before returning a timeout error. Patterns
// using features outside RE2 (backreferences, character-class subtraction,
// large quantifiers) are compiled with [regexp2], a backtracking engine
// that is vulnerable to catastrophic-backtracking ReDoS on adversary-
// supplied inputs. The default is a defense-in-depth ceiling for those
// patterns; pure RE2-compatible patterns are unaffected.
//
// regexp2's fastclock has ~100ms granularity, which can fire spurious
// timeouts on loaded shared CI runners when the configured budget is
// only a few hundred milliseconds. The default of 5s gives generous
// headroom while still terminating a pathological ReDoS attempt long
// before it exhausts a request budget.
//
// Set to 0 to disable the timeout entirely. Mutating this value affects
// only subsequently-compiled regexes; already-cached compilations keep
// the timeout in effect at the time they were compiled.
var DefaultRegexMatchTimeout = 5 * time.Second

// isRegexMatchTimeout reports whether err is regexp2's wall-clock match-timeout
// error. regexp2 returns an unexported, untyped fmt error for timeouts, so the
// only stable discriminator is its message text.
func isRegexMatchTimeout(err error) bool {
	return err != nil && strings.Contains(err.Error(), "match timeout")
}

// withSpuriousTimeoutRetry runs fn, retrying when it reports a regexp2 match
// timeout that cannot be genuine.
//
// regexp2 enforces MatchTimeout via a shared "fastclock" background goroutine
// that stamps the current time into an atomic every ~100ms. On a heavily
// loaded host that goroutine can be starved for seconds and then jump the
// clock forward in a single step; a match that has barely started then sees
// its deadline already crossed and fails with a "match timeout" almost
// immediately. (This is the residual race left after regexp2 v1.12.0's fix,
// which only refreshes the stale clock when its updater is not running.)
//
// A genuine timeout only fires after roughly `budget` of wall-clock time has
// elapsed, whereas a spurious one fires well before that. So when fn reports a
// timeout but less than half the budget (budget/2) has actually elapsed, the
// result is treated as bogus and we retry — by then the clock goroutine is
// live and the match completes normally. Timeouts that fire at or beyond
// budget/2 (genuine ReDoS) are propagated unchanged, preserving the
// DoS-protection contract.
func withSpuriousTimeoutRetry[T any](budget time.Duration, fn func() (T, error)) (T, error) {
	const maxAttempts = 3
	var v T
	var err error
	for range maxAttempts {
		start := time.Now()
		v, err = fn()
		if !isRegexMatchTimeout(err) {
			return v, err
		}
		// A real timeout burns ~budget of wall time before firing; treat
		// anything well short of that as a spurious fastclock jump and retry.
		if budget <= 0 || time.Since(start) >= budget/2 {
			return v, err
		}
	}
	return v, err
}

func (r *compiledXPathRegex) MatchString(s string) (bool, error) {
	if r.backtrack != nil {
		return withSpuriousTimeoutRetry(r.backtrack.MatchTimeout, func() (bool, error) {
			return r.backtrack.MatchString(s)
		})
	}
	return r.std.MatchString(s), nil
}

func (r *compiledXPathRegex) ReplaceAllString(s, replacement string) (string, error) {
	if r.backtrack != nil {
		return withSpuriousTimeoutRetry(r.backtrack.MatchTimeout, func() (string, error) {
			return r.backtrack.Replace(s, replacement, -1, -1)
		})
	}
	return r.std.ReplaceAllString(s, replacement), nil
}

// findStringMatch and findNextMatch wrap regexp2's match iteration with the
// same spurious-timeout retry applied to MatchString, so multi-match callers
// (Split, FindAllStringSubmatchIndex) are equally resilient to fastclock jumps.
func (r *compiledXPathRegex) findStringMatch(s string) (*regexp2.Match, error) {
	return withSpuriousTimeoutRetry(r.backtrack.MatchTimeout, func() (*regexp2.Match, error) {
		return r.backtrack.FindStringMatch(s)
	})
}

func (r *compiledXPathRegex) findNextMatch(m *regexp2.Match) (*regexp2.Match, error) {
	return withSpuriousTimeoutRetry(r.backtrack.MatchTimeout, func() (*regexp2.Match, error) {
		return r.backtrack.FindNextMatch(m)
	})
}

func (r *compiledXPathRegex) Split(s string, n int) ([]string, error) {
	if r.backtrack == nil {
		return r.std.Split(s, n), nil
	}

	offsets := runeByteOffsets(s)
	var parts []string
	last := 0
	count := 0
	match, err := r.findStringMatch(s)
	if err != nil {
		return nil, err
	}
	for match != nil {
		if n > 0 && count >= n-1 {
			break
		}
		start := offsets[match.Index]
		end := offsets[match.Index+match.Length]
		parts = append(parts, s[last:start])
		last = end
		count++
		match, err = r.findNextMatch(match)
		if err != nil {
			return nil, err
		}
	}
	parts = append(parts, s[last:])
	return parts, nil
}

func (r *compiledXPathRegex) FindAllStringSubmatchIndex(s string, n int) ([][]int, error) {
	if r.backtrack == nil {
		return r.std.FindAllStringSubmatchIndex(s, n), nil
	}

	offsets := runeByteOffsets(s)
	var result [][]int
	match, err := r.findStringMatch(s)
	if err != nil {
		return nil, err
	}
	for match != nil {
		groups := match.Groups()
		entry := make([]int, 0, len(groups)*2)
		for _, group := range groups {
			if len(group.Captures) == 0 {
				entry = append(entry, -1, -1)
				continue
			}
			start := offsets[group.Index]
			end := offsets[group.Index+group.Length]
			entry = append(entry, start, end)
		}
		result = append(result, entry)
		if n > 0 && len(result) >= n {
			break
		}
		match, err = r.findNextMatch(match)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// eachStringSubmatchIndex streams successive matches of the regex in s one at a
// time, calling fn with the per-match (start,end) index pairs (the same layout
// as one FindAllStringSubmatchIndex entry). Iteration stops early when fn
// returns false. limit caps the maximum number of matches ever produced (a
// non-positive limit means no cap); it bounds the up-front materialization for
// the one path that cannot stream incrementally (leading-context patterns),
// keeping live memory proportional to the cap rather than the input's match
// count. The streaming paths never accumulate matches at all.
func (r *compiledXPathRegex) eachStringSubmatchIndex(s string, limit int, fn func([]int) bool) error {
	// Normalize the public contract: a non-positive limit means "uncapped".
	// Internally that is represented as -1, which is also the "all matches"
	// sentinel understood by the full-context FindAllStringSubmatchIndex path.
	if limit <= 0 {
		limit = -1
	}
	if r.backtrack == nil {
		return r.eachStdSubmatchIndex(s, limit, fn)
	}
	return r.eachBacktrackSubmatchIndex(s, limit, fn)
}

// maxFullContextIndexCells bounds the TOTAL number of index ints the single
// FindAllStringSubmatchIndex pass may allocate for a leading-context RE2
// pattern that cannot be streamed (see eachStdSubmatchIndex). Such a pattern
// must materialize its matches in one shot, and each match record holds
// 2*(NumSubexp()+1) ints — so bounding the match COUNT alone lets a
// high-capture pattern (e.g. `^()()()...` with the `m` flag) over a large
// input allocate far beyond the intended ceiling before the cap fires.
// Bounding the total index cells instead keeps the worst-case allocation fixed
// regardless of capture count: the per-pattern match cap is derived from this
// cell budget, and an input producing more cells than this is rejected with
// [ErrRegexMatchLimit] rather than silently truncated or allowed to allocate
// proportionally to the input.
const maxFullContextIndexCells = 1 << 20 // ~1M index ints

// eachStdSubmatchIndex ports the standard library's regexp.allMatches loop
// (its successive-match + empty-match advancement rules) so that the streamed
// match sequence is identical to std.FindAllStringSubmatchIndex, while
// delivering one match at a time instead of accumulating them all.
//
// The streaming loop advances an offset and matches against s[pos:]; that is
// exact for patterns without a leading-context assertion. Patterns that DO have
// one (^, \A, \b, \B) would see a spurious "start of input" at every slice
// boundary, so they are matched against the WHOLE string by Go's RE2 engine via
// FindAllStringSubmatchIndex. That call is the only one that accumulates, so it
// is bounded to the maxFullContextIndexCells budget rather than to the caller's
// (possibly byte-budget-sized) limit; an input that exceeds that ceiling is
// rejected with [ErrRegexMatchLimit] instead of allocating one match per input
// position. RE2 stays linear, so a valid backtracking-shaped pattern like
// ^(a+)+b never runs on a backtracking engine.
func (r *compiledXPathRegex) eachStdSubmatchIndex(s string, limit int, fn func([]int) bool) error {
	if r.stdNeedsFullContext {
		return r.eachStdFullContext(s, limit, fn)
	}

	end := len(s)
	pos := 0
	prevMatchEnd := -1
	produced := 0
	for pos <= end {
		loc := r.std.FindStringSubmatchIndex(s[pos:])
		if loc == nil {
			break
		}
		// FindStringSubmatchIndex reports indices relative to s[pos:]; shift
		// them back to absolute positions (leaving the -1 "unmatched group"
		// sentinels intact).
		for i := range loc {
			if loc[i] >= 0 {
				loc[i] += pos
			}
		}
		matchStart, matchEnd := loc[0], loc[1]
		accept := true
		if matchEnd == pos {
			// Empty match at the current scan position. Mirror allMatches:
			// reject an empty match immediately following a previous match,
			// then advance past one rune to avoid looping forever.
			if matchStart == prevMatchEnd {
				accept = false
			}
			_, width := utf8.DecodeRuneInString(s[pos:])
			if width > 0 {
				pos += width
			} else {
				pos = end + 1
			}
		} else {
			pos = matchEnd
		}
		prevMatchEnd = matchEnd
		if accept {
			if !fn(loc) {
				return nil
			}
			produced++
			if limit > 0 && produced >= limit {
				return nil
			}
		}
	}
	return nil
}

// eachStdFullContext handles the leading-context branch of eachStdSubmatchIndex:
// a pattern whose zero-width assertions depend on input to the left of the scan
// position cannot be streamed by slicing, so its matches are produced by a
// single FindAllStringSubmatchIndex pass over the whole string. To keep that
// one accumulating pass from amplifying a bounded input into a match record per
// position, the pass is capped at a per-pattern match ceiling derived from the
// maxFullContextIndexCells budget — independently of the caller's limit (which
// may be derived from a large byte budget). When the caller's own limit is the
// smaller bound, it governs and the caller observes the overflow itself; when
// this function's ceiling is the binding bound and it is exceeded, the input is
// rejected with [ErrRegexMatchLimit] rather than silently truncated. Each
// surviving match is handed to fn one at a time, so a caller checking a
// cancelled context inside fn observes it between matches.
func (r *compiledXPathRegex) eachStdFullContext(s string, limit int, fn func([]int) bool) error {
	// Each FindAllStringSubmatchIndex match record holds 2*(NumSubexp()+1) ints,
	// so derive the match ceiling from the total cell budget instead of bounding
	// the match count directly. This keeps the worst-case allocation fixed
	// regardless of capture count: a high-capture pattern gets a proportionally
	// smaller match cap (clamped to at least one so a pattern whose single record
	// already exceeds the budget can still produce one match before tripping it).
	cellsPerMatch := 2 * (r.std.NumSubexp() + 1)
	ceiling := max(maxFullContextIndexCells/cellsPerMatch, 1)
	// A smaller caller limit takes precedence and lets the caller enforce its own
	// budget (in which case we honor the public limit contract exactly — produce
	// at most `limit`).
	internalBound := true
	if limit > 0 && limit <= ceiling {
		ceiling = limit
		internalBound = false
	}
	// When our own ceiling is binding, request one extra so "exactly at the
	// ceiling" is distinguishable from "over the ceiling"; when the caller's
	// limit is binding, request exactly that many. Either way the allocation is
	// bounded to (ceiling+1)*cellsPerMatch ints — at most ~maxFullContextIndexCells
	// plus one record, never proportional to the input match count.
	n := ceiling
	if internalBound {
		n = ceiling + 1
	}
	matches := r.std.FindAllStringSubmatchIndex(s, n)
	if internalBound && len(matches) > ceiling {
		return ErrRegexMatchLimit
	}
	for _, m := range matches {
		if !fn(m) {
			return nil
		}
	}
	return nil
}

// eachBacktrackSubmatchIndex is the regexp2 (backtracking-engine) counterpart
// of eachStdSubmatchIndex for patterns that genuinely require regexp2
// (backreferences, character-class subtraction, large quantifiers). It streams
// via the engine's native FindNextMatch semantics (which resume against the
// original string, preserving leading-context assertions), converting regexp2's
// rune indices to byte offsets and stopping early when fn returns false. Because
// matches are produced incrementally and never accumulated, a caller can enforce
// a budget or honor a cancelled context DURING enumeration. No empty-match dedup
// is applied — these patterns follow regexp2's own iteration. A positive limit
// stops iteration after that many matches; a non-positive limit means uncapped.
func (r *compiledXPathRegex) eachBacktrackSubmatchIndex(s string, limit int, fn func([]int) bool) error {
	re := r.backtrack
	offsets := runeByteOffsets(s)
	match, err := withSpuriousTimeoutRetry(re.MatchTimeout, func() (*regexp2.Match, error) {
		return re.FindStringMatch(s)
	})
	if err != nil {
		return err
	}
	produced := 0
	for match != nil {
		groups := match.Groups()
		entry := make([]int, 0, len(groups)*2)
		for _, group := range groups {
			if len(group.Captures) == 0 {
				entry = append(entry, -1, -1)
				continue
			}
			start := offsets[group.Index]
			end := offsets[group.Index+group.Length]
			entry = append(entry, start, end)
		}
		if !fn(entry) {
			return nil
		}
		produced++
		if limit > 0 && produced >= limit {
			return nil
		}
		cur := match
		match, err = withSpuriousTimeoutRetry(re.MatchTimeout, func() (*regexp2.Match, error) {
			return re.FindNextMatch(cur)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *compiledXPathRegex) NumSubexp() int {
	if r.backtrack != nil {
		return r.numGroups
	}
	return r.std.NumSubexp()
}

func runeByteOffsets(s string) []int {
	offsets := make([]int, 0, utf8.RuneCountInString(s)+1)
	for i := range s {
		offsets = append(offsets, i)
	}
	offsets = append(offsets, len(s))
	return offsets
}

// resolveUnicodeProperty maps a Unicode property name to a *unicode.RangeTable.
// It checks unicode.Categories, unicode.Scripts, and the unicodeBlocks map.
// Returns nil if the name is not recognized.
func resolveUnicodeProperty(name string) *unicode.RangeTable {
	if rt, ok := unicode.Categories[name]; ok {
		return rt
	}
	if rt, ok := unicode.Scripts[name]; ok {
		return rt
	}
	return nil
}

// detectSimpleUnicodePattern checks whether pattern (before flag processing)
// is exactly \p{Name} or \P{Name} with no flags other than possibly empty.
// Returns (table, negated, true) when the pattern is simple.
func detectSimpleUnicodePattern(pattern, flags string) (*unicode.RangeTable, bool, bool) {
	// Only patterns with no flags (or empty flags) qualify
	if flags != "" {
		return nil, false, false
	}
	runes := []rune(pattern)
	if len(runes) < 5 {
		return nil, false, false
	}
	if runes[0] != '\\' {
		return nil, false, false
	}
	neg := false
	switch runes[1] {
	case 'p':
		// ok
	case 'P':
		neg = true
	default:
		return nil, false, false
	}
	if runes[2] != '{' || runes[len(runes)-1] != '}' {
		return nil, false, false
	}
	name := string(runes[3 : len(runes)-1])
	rt := resolveUnicodeProperty(name)
	if rt == nil {
		return nil, false, false
	}
	return rt, neg, true
}

func compileXPathRegex(pattern, flags string) (*compiledXPathRegex, error) {
	cacheKey := xpathRegexCacheKey{pattern: pattern, flags: flags}
	if cached, ok := compiledXPathRegexCache.Load(cacheKey); ok {
		return cached, nil
	}

	// Detect simple \p{Name} / \P{Name} patterns for single-rune fast paths
	simpleTable, simpleNegated, simpleOk := detectSimpleUnicodePattern(pattern, flags)

	// Check for 'q' flag early to skip validation for literal patterns
	hasQ := strings.ContainsRune(flags, 'q')
	if !hasQ && strings.ContainsRune(flags, 'x') {
		pattern = stripFreeSpacing(pattern)
	}
	hasBackrefs := !hasQ && hasXPathBackrefs(pattern)
	hasSubtraction := !hasQ && hasXPathCharClassSubtraction(pattern)
	hasLargeQuantifier := !hasQ && hasLargeXPathQuantifier(pattern)

	if !hasQ {
		// Reject Perl-specific constructs first
		if err := rejectPerlSpecific(pattern); err != nil {
			return nil, err
		}

		// Validate XPath-specific regex restrictions before compilation
		if err := validateXPathRegex(pattern, hasBackrefs); err != nil {
			return nil, err
		}
	}

	var prefix strings.Builder
	prefix.WriteString("(?")
	dotAll := false
	literal := false
	ignoreCase := false
	var re2Opts regexp2.RegexOptions = regexp2.RE2
	for _, f := range flags {
		switch f {
		case 'i':
			ignoreCase = true
			prefix.WriteRune('i')
			re2Opts |= regexp2.IgnoreCase
		case 'm':
			prefix.WriteRune('m')
			re2Opts |= regexp2.Multiline
		case 's':
			// Handled by translateXPathRegex dotAll parameter;
			// do not add Go's (?s) since we expand '.' ourselves.
			dotAll = true
		case 'x':
			// Free-spacing normalization was applied before validation.
		case 'q':
			// Literal mode: quote the entire pattern, skip regex translation
			literal = true
		default:
			return nil, &XPathError{Code: errCodeFORX0001, Message: fmt.Sprintf("invalid regex flag: %c", f)}
		}
	}

	if literal {
		pattern = regexp.QuoteMeta(pattern)
	} else {
		if hasBackrefs {
			pattern = normalizeXPathBackrefs(pattern)
		}
		// Translate XPath/XML Schema regex features to Go-compatible patterns
		translated, err := translateXPathRegex(pattern, dotAll, ignoreCase)
		if err != nil {
			return nil, err
		}
		pattern = translated
	}

	if prefix.Len() > 2 {
		prefix.WriteRune(')')
		pattern = prefix.String() + pattern
	}
	if hasBackrefs || hasSubtraction || hasLargeQuantifier {
		re, err := regexp2.Compile(pattern, re2Opts)
		if err != nil {
			return nil, &XPathError{Code: errCodeFORX0002, Message: fmt.Sprintf("invalid regular expression: %s", err)}
		}
		if t := DefaultRegexMatchTimeout; t > 0 {
			re.MatchTimeout = t
		}
		compiled := &compiledXPathRegex{
			backtrack:    re,
			numGroups:    len(re.GetGroupNumbers()) - 1,
			unicodeTable: simpleTable,
			negated:      simpleNegated,
			isSimple:     simpleOk,
		}
		actual, _ := compiledXPathRegexCache.LoadOrStore(cacheKey, compiled)
		return actual, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, &XPathError{Code: errCodeFORX0002, Message: fmt.Sprintf("invalid regular expression: %s", err)}
	}
	compiled := &compiledXPathRegex{
		std:                 re,
		unicodeTable:        simpleTable,
		negated:             simpleNegated,
		isSimple:            simpleOk,
		stdNeedsFullContext: patternNeedsFullContext(pattern),
	}
	actual, _ := compiledXPathRegexCache.LoadOrStore(cacheKey, compiled)
	return actual, nil
}

// stripFreeSpacing removes unescaped whitespace from a regex pattern (x flag).
func stripFreeSpacing(pattern string) string {
	var b strings.Builder
	runes := []rune(pattern)
	inCharClass := 0
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\\' && i+1 < len(runes) {
			next := i + 1
			if inCharClass == 0 && unicode.IsSpace(runes[next]) {
				for next < len(runes) && unicode.IsSpace(runes[next]) {
					next++
				}
				if next >= len(runes) {
					b.WriteRune(r)
					break
				}
			}
			b.WriteRune(r)
			i = next
			b.WriteRune(runes[i])
			continue
		}
		switch r {
		case '[':
			inCharClass++
		case ']':
			if inCharClass > 0 {
				inCharClass--
			}
		}
		if inCharClass == 0 && unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// fnContainsToken implements fn:contains-token($input, $token [, $collation])
// Returns true if any string in $input, after tokenizing on whitespace,
// matches $token (compared case-insensitively if collation is default).
func fnContainsToken(ctx context.Context, args []Sequence) (Sequence, error) {
	coll, err := getCollation(ctx, args, 2)
	if err != nil {
		return nil, err
	}
	token, err := coerceArgToStringRequired(args[1])
	if err != nil {
		return nil, err
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return SingleBoolean(false), nil
	}
	for item := range seqItems(args[0]) {
		a, err := AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		s, _ := atomicToString(a)
		for _, tok := range splitXMLWhitespace(s) {
			if coll.compare(tok, token) == 0 {
				return SingleBoolean(true), nil
			}
		}
	}
	return SingleBoolean(false), nil
}

// getCollation resolves the collation from function arguments.
// If a collation argument is provided, it resolves it using the base URI from the eval context.
// Otherwise returns the default codepoint collation.
func getCollation(ctx context.Context, args []Sequence, collationArgIdx int) (*collationImpl, error) {
	baseURI := ""
	if fc := getFnContext(ctx); fc != nil {
		baseURI = fc.baseURI
		if collationArgIdx >= len(args) || seqLen(args[collationArgIdx]) == 0 {
			if fc.defaultCollation != "" {
				return resolveCollation(fc.defaultCollation, baseURI)
			}
			return codepointCollation, nil
		}
	} else if collationArgIdx >= len(args) || seqLen(args[collationArgIdx]) == 0 {
		return codepointCollation, nil
	}
	uri, err := coerceArgToString(args[collationArgIdx])
	if err != nil {
		return nil, err
	}
	return resolveCollation(uri, baseURI)
}
