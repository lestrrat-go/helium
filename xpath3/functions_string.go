package xpath3

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/dlclark/regexp2"
	"github.com/lestrrat-go/helium"
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
			return nil, &XPathError{Code: errCodeXPDY0002, Message: "context item is absent"}
		}
		s, ok := fc.contextStringValue()
		if !ok {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "context item has no string value"}
		}
		return SingleString(s), nil
	}
	if len(args[0]) == 0 {
		return SingleString(""), nil
	}
	if len(args[0]) > 1 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "fn:string requires a single item, got sequence of length > 1"}
	}
	item := args[0][0]
	// fn:string does not accept function items, maps, or arrays
	switch item.(type) {
	case FunctionItem, MapItem, ArrayItem:
		return nil, &XPathError{Code: errCodeFOTY0014, Message: fmt.Sprintf("fn:string: cannot get string value of %T", item)}
	}
	a, err := AtomizeItem(item)
	if err != nil {
		return nil, err
	}
	s, _ := atomicToString(a)
	return SingleString(s), nil
}

func fnCodepointsToString(_ context.Context, args []Sequence) (Sequence, error) {
	seq := args[0]

	// Fast path: singleton integer (common in unicode-90 where each codepoint
	// is mapped individually via codepoints-to-string(.))
	if len(seq) == 1 {
		cp, err := itemToCodepoint(seq[0])
		if err != nil {
			return nil, err
		}
		if !isValidXMLCodepoint(cp) {
			return nil, &XPathError{Code: "FOCH0001", Message: fmt.Sprintf("invalid XML character [x%X]", cp)}
		}
		return SingleString(string(rune(cp))), nil
	}

	var b strings.Builder
	for _, item := range seq {
		cp, err := itemToCodepoint(item)
		if err != nil {
			return nil, err
		}
		if !isValidXMLCodepoint(cp) {
			return nil, &XPathError{Code: "FOCH0001", Message: fmt.Sprintf("invalid XML character [x%X]", cp)}
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
			return int(n.Int64()), nil
		}
	}
	return int(a.ToFloat64()), nil
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

func fnStringToCodepoints(_ context.Context, args []Sequence) (Sequence, error) {
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	if s == "" {
		return nil, nil
	}
	runes := []rune(s)
	result := make(Sequence, len(runes))
	for i, r := range runes {
		result[i] = AtomicValue{TypeName: TypeInteger, Value: big.NewInt(int64(r))}
	}
	return result, nil
}

func fnCompare(ctx context.Context, args []Sequence) (Sequence, error) {
	coll, err := getCollation(ctx, args, 2)
	if err != nil {
		return nil, err
	}
	if len(args[0]) == 0 || len(args[1]) == 0 {
		return nil, nil
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
	if len(args[0]) == 0 || len(args[1]) == 0 {
		return nil, nil
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

func fnStringJoin(_ context.Context, args []Sequence) (Sequence, error) {
	sep := ""
	if len(args) > 1 {
		var err error
		sep, err = coerceArgToStringRequired(args[1])
		if err != nil {
			return nil, err
		}
	}
	var b strings.Builder
	for i, item := range args[0] {
		if i > 0 && sep != "" {
			b.WriteString(sep)
		}
		a, err := AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		s, _ := atomicToString(a)
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
		if len(args[0]) > 1 {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("string-length: expected single item, got sequence of length %d", len(args[0]))}
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
			return nil, &XPathError{Code: errCodeXPDY0002, Message: "context item is absent"}
		}
		sv, ok := fc.contextStringValue()
		if !ok {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "context item has no string value"}
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

func fnContains(ctx context.Context, args []Sequence) (Sequence, error) {
	coll, err := getCollation(ctx, args, 2)
	if err != nil {
		return nil, err
	}
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	sub, err := coerceArgToString(args[1])
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
	coll, err := getCollation(ctx, args, 2)
	if err != nil {
		return nil, err
	}
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	prefix, err := coerceArgToString(args[1])
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
	coll, err := getCollation(ctx, args, 2)
	if err != nil {
		return nil, err
	}
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	suffix, err := coerceArgToString(args[1])
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
	coll, err := getCollation(ctx, args, 2)
	if err != nil {
		return nil, err
	}
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	sep, err := coerceArgToString(args[1])
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
	coll, err := getCollation(ctx, args, 2)
	if err != nil {
		return nil, err
	}
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	sep, err := coerceArgToString(args[1])
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
	if len(args[0]) == 0 {
		return SingleBoolean(false), nil // input is xs:string? — empty yields false
	}
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	if len(args[1]) == 0 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "fn:matches pattern must not be empty sequence"}
	}
	pattern, err := coerceArgToStringRequired(args[1])
	if err != nil {
		return nil, err
	}
	flags := ""
	if len(args) > 2 {
		if len(args[2]) == 0 {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "fn:matches flags must not be empty sequence"}
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
	if len(args[0]) == 0 {
		return SingleString(""), nil // input is xs:string? — empty yields ""
	}
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	if len(args[1]) == 0 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "fn:replace pattern must not be empty sequence"}
	}
	pattern, err := coerceArgToStringRequired(args[1])
	if err != nil {
		return nil, err
	}
	if len(args[2]) == 0 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "fn:replace replacement must not be empty sequence"}
	}
	replacement, err := coerceArgToStringRequired(args[2])
	if err != nil {
		return nil, err
	}
	flags := ""
	if len(args) > 3 {
		if len(args[3]) == 0 {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "fn:replace flags must not be empty sequence"}
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
			if validEnd == start {
				// No valid group number found — $0 or group exceeds numGroups
				// Per XPath spec, this is still valid syntax but refers to
				// a non-existent group, which Go replaces with empty string.
				// Write the full collected number.
				b.WriteString("${")
				b.WriteString(repl[start:i])
				b.WriteByte('}')
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
	if len(args[0]) == 0 {
		return nil, nil // input is xs:string? — empty yields empty
	}
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	if s == "" {
		return nil, nil
	}

	// 1-arg form: normalize XML whitespace (#x20, #x9, #xA, #xD), then split
	if len(args) == 1 {
		tokens := splitXMLWhitespace(s)
		result := make(Sequence, len(tokens))
		for i, t := range tokens {
			result[i] = AtomicValue{TypeName: TypeString, Value: t}
		}
		return result, nil
	}

	if len(args[1]) == 0 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "fn:tokenize pattern must not be empty sequence"}
	}
	pattern, err := coerceArgToStringRequired(args[1])
	if err != nil {
		return nil, err
	}
	flags := ""
	if len(args) > 2 {
		if len(args[2]) == 0 {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "fn:tokenize flags must not be empty sequence"}
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
	result := make(Sequence, len(parts))
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

func fnAnalyzeString(_ context.Context, args []Sequence) (Sequence, error) {
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

	pos := 0
	matches, err := re.FindAllStringSubmatchIndex(s, -1)
	if err != nil {
		return nil, &XPathError{Code: errCodeFORX0002, Message: fmt.Sprintf("regex match failed: %v", err)}
	}
	for _, m := range matches {
		start, end := m[0], m[1]
		if start > pos {
			if err := appendAnalyzeStringTextElement(doc, root, "non-match", s[pos:start]); err != nil {
				return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("analyze-string: failed to build result: %v", err)}
			}
		}
		matchElem, err := createAnalyzeStringElement(doc, "match")
		if err != nil {
			return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("analyze-string: failed to build result: %v", err)}
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
						return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("analyze-string: failed to build result: %v", err)}
					}
				}
				groupElem, err := createAnalyzeStringElement(doc, "group")
				if err != nil {
					return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("analyze-string: failed to build result: %v", err)}
				}
				if err := groupElem.SetAttribute("nr", fmt.Sprintf("%d", g)); err != nil {
					return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("analyze-string: failed to build result: %v", err)}
				}
				if err := groupElem.AppendText([]byte(s[gs:ge])); err != nil {
					return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("analyze-string: failed to build result: %v", err)}
				}
				if err := matchElem.AddChild(groupElem); err != nil {
					return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("analyze-string: failed to build result: %v", err)}
				}
				groupPos = ge
			}
			if groupPos < end {
				if err := matchElem.AppendText([]byte(s[groupPos:end])); err != nil {
					return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("analyze-string: failed to build result: %v", err)}
				}
			}
		} else {
			if err := matchElem.AppendText([]byte(s[start:end])); err != nil {
				return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("analyze-string: failed to build result: %v", err)}
			}
		}
		if err := root.AddChild(matchElem); err != nil {
			return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("analyze-string: failed to build result: %v", err)}
		}
		pos = end
	}
	if pos < len(s) {
		if err := appendAnalyzeStringTextElement(doc, root, "non-match", s[pos:]); err != nil {
			return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("analyze-string: failed to build result: %v", err)}
		}
	}

	return Sequence{NodeItem{Node: root}}, nil
}

func createAnalyzeStringElement(doc *helium.Document, localName string) (*helium.Element, error) {
	elem, err := doc.CreateElement(localName)
	if err != nil {
		return nil, err
	}
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
}

type xpathRegexCacheKey struct {
	pattern string
	flags   string
}

var compiledXPathRegexCache sync.Map

func (r *compiledXPathRegex) MatchString(s string) (bool, error) {
	if r.backtrack != nil {
		return r.backtrack.MatchString(s)
	}
	return r.std.MatchString(s), nil
}

func (r *compiledXPathRegex) ReplaceAllString(s, replacement string) (string, error) {
	if r.backtrack != nil {
		return r.backtrack.Replace(s, replacement, -1, -1)
	}
	return r.std.ReplaceAllString(s, replacement), nil
}

func (r *compiledXPathRegex) Split(s string, n int) ([]string, error) {
	if r.backtrack == nil {
		return r.std.Split(s, n), nil
	}

	offsets := runeByteOffsets(s)
	var parts []string
	last := 0
	count := 0
	match, err := r.backtrack.FindStringMatch(s)
	for match != nil {
		if n > 0 && count >= n-1 {
			break
		}
		start := offsets[match.Index]
		end := offsets[match.Index+match.Length]
		parts = append(parts, s[last:start])
		last = end
		count++
		match, err = r.backtrack.FindNextMatch(match)
		if err != nil {
			return nil, err
		}
	}
	if err != nil {
		return nil, err
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
	match, err := r.backtrack.FindStringMatch(s)
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
		match, err = r.backtrack.FindNextMatch(match)
		if err != nil {
			return nil, err
		}
	}
	return result, err
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
		return cached.(*compiledXPathRegex), nil
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
		compiled := &compiledXPathRegex{
			backtrack:    re,
			numGroups:    len(re.GetGroupNumbers()) - 1,
			unicodeTable: simpleTable,
			negated:      simpleNegated,
			isSimple:     simpleOk,
		}
		actual, _ := compiledXPathRegexCache.LoadOrStore(cacheKey, compiled)
		return actual.(*compiledXPathRegex), nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, &XPathError{Code: errCodeFORX0002, Message: fmt.Sprintf("invalid regular expression: %s", err)}
	}
	compiled := &compiledXPathRegex{
		std:          re,
		unicodeTable: simpleTable,
		negated:      simpleNegated,
		isSimple:     simpleOk,
	}
	actual, _ := compiledXPathRegexCache.LoadOrStore(cacheKey, compiled)
	return actual.(*compiledXPathRegex), nil
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
	for _, item := range args[0] {
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
		if collationArgIdx >= len(args) || len(args[collationArgIdx]) == 0 {
			if fc.defaultCollation != "" {
				return resolveCollation(fc.defaultCollation, baseURI)
			}
			return codepointCollation, nil
		}
	} else if collationArgIdx >= len(args) || len(args[collationArgIdx]) == 0 {
		return codepointCollation, nil
	}
	uri, err := coerceArgToString(args[collationArgIdx])
	if err != nil {
		return nil, err
	}
	return resolveCollation(uri, baseURI)
}
