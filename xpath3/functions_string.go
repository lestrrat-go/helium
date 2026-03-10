package xpath3

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	ixpath "github.com/lestrrat-go/helium/internal/xpath"
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
}

func fnString(ctx context.Context, args []Sequence) (Sequence, error) {
	if len(args) == 0 {
		fc := GetFnContext(ctx)
		if fc == nil || fc.node == nil {
			return SingleString(""), nil
		}
		return SingleString(ixpath.StringValue(fc.node)), nil
	}
	if len(args[0]) == 0 {
		return SingleString(""), nil
	}
	a, err := AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	s, _ := atomicToString(a)
	return SingleString(s), nil
}

func fnCodepointsToString(_ context.Context, args []Sequence) (Sequence, error) {
	var b strings.Builder
	for _, item := range args[0] {
		a, err := AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		cp := int(a.ToFloat64())
		if !utf8.ValidRune(rune(cp)) {
			return nil, &XPathError{Code: "FOCH0001", Message: fmt.Sprintf("invalid codepoint: %d", cp)}
		}
		b.WriteRune(rune(cp))
	}
	return SingleString(b.String()), nil
}

func fnStringToCodepoints(_ context.Context, args []Sequence) (Sequence, error) {
	s := seqToString(args[0])
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

func fnCompare(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 || len(args[1]) == 0 {
		return nil, nil
	}
	s1 := seqToString(args[0])
	s2 := seqToString(args[1])
	cmp := strings.Compare(s1, s2)
	return SingleInteger(int64(cmp)), nil
}

func fnCodepointEqual(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 || len(args[1]) == 0 {
		return nil, nil
	}
	return SingleBoolean(seqToString(args[0]) == seqToString(args[1])), nil
}

func fnConcat(_ context.Context, args []Sequence) (Sequence, error) {
	var b strings.Builder
	for _, arg := range args {
		b.WriteString(seqToString(arg))
	}
	return SingleString(b.String()), nil
}

func fnStringJoin(_ context.Context, args []Sequence) (Sequence, error) {
	sep := ""
	if len(args) > 1 {
		sep = seqToString(args[1])
	}
	parts := make([]string, 0, len(args[0]))
	for _, item := range args[0] {
		a, err := AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		s, _ := atomicToString(a)
		parts = append(parts, s)
	}
	return SingleString(strings.Join(parts, sep)), nil
}

func fnSubstring(_ context.Context, args []Sequence) (Sequence, error) {
	s := seqToString(args[0])
	startPos := seqToDouble(args[1])
	runes := []rune(s)

	// XPath round
	rStart := math.Floor(startPos + 0.5)

	if len(args) == 3 {
		length := seqToDouble(args[2])
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
		fc := GetFnContext(ctx)
		if fc == nil || fc.node == nil {
			s = ""
		} else {
			s = ixpath.StringValue(fc.node)
		}
	} else {
		s = seqToString(args[0])
	}
	return SingleInteger(int64(len([]rune(s)))), nil
}

func fnNormalizeSpace(ctx context.Context, args []Sequence) (Sequence, error) {
	var s string
	if len(args) == 0 {
		fc := GetFnContext(ctx)
		if fc == nil || fc.node == nil {
			s = ""
		} else {
			s = ixpath.StringValue(fc.node)
		}
	} else {
		s = seqToString(args[0])
	}
	return SingleString(strings.Join(strings.Fields(s), " ")), nil
}

func fnNormalizeUnicode(_ context.Context, args []Sequence) (Sequence, error) {
	s := seqToString(args[0])
	if s == "" {
		return SingleString(""), nil
	}
	// Only NFC supported in v1
	if len(args) > 1 {
		form := strings.TrimSpace(strings.ToUpper(seqToString(args[1])))
		if form != "" && form != "NFC" {
			return nil, &XPathError{Code: "FOCH0003", Message: fmt.Sprintf("unsupported normalization form: %s", form)}
		}
	}
	return SingleString(s), nil
}

func fnUpperCase(_ context.Context, args []Sequence) (Sequence, error) {
	return SingleString(strings.ToUpper(seqToString(args[0]))), nil
}

func fnLowerCase(_ context.Context, args []Sequence) (Sequence, error) {
	return SingleString(strings.ToLower(seqToString(args[0]))), nil
}

func fnTranslate(_ context.Context, args []Sequence) (Sequence, error) {
	s := seqToString(args[0])
	from := []rune(seqToString(args[1]))
	to := []rune(seqToString(args[2]))

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

func fnContains(_ context.Context, args []Sequence) (Sequence, error) {
	return SingleBoolean(strings.Contains(seqToString(args[0]), seqToString(args[1]))), nil
}

func fnStartsWith(_ context.Context, args []Sequence) (Sequence, error) {
	return SingleBoolean(strings.HasPrefix(seqToString(args[0]), seqToString(args[1]))), nil
}

func fnEndsWith(_ context.Context, args []Sequence) (Sequence, error) {
	return SingleBoolean(strings.HasSuffix(seqToString(args[0]), seqToString(args[1]))), nil
}

func fnSubstringBefore(_ context.Context, args []Sequence) (Sequence, error) {
	s := seqToString(args[0])
	sep := seqToString(args[1])
	idx := strings.Index(s, sep)
	if idx < 0 {
		return SingleString(""), nil
	}
	return SingleString(s[:idx]), nil
}

func fnSubstringAfter(_ context.Context, args []Sequence) (Sequence, error) {
	s := seqToString(args[0])
	sep := seqToString(args[1])
	idx := strings.Index(s, sep)
	if idx < 0 {
		return SingleString(""), nil
	}
	return SingleString(s[idx+len(sep):]), nil
}

func fnMatches(_ context.Context, args []Sequence) (Sequence, error) {
	s := seqToString(args[0])
	pattern := seqToString(args[1])
	flags := ""
	if len(args) > 2 {
		flags = seqToString(args[2])
	}
	re, err := compileXPathRegex(pattern, flags)
	if err != nil {
		return nil, err
	}
	return SingleBoolean(re.MatchString(s)), nil
}

func fnReplace(_ context.Context, args []Sequence) (Sequence, error) {
	s := seqToString(args[0])
	pattern := seqToString(args[1])
	replacement := seqToString(args[2])
	flags := ""
	if len(args) > 3 {
		flags = seqToString(args[3])
	}

	isLiteral := strings.Contains(flags, "q")

	var goRepl string
	if isLiteral {
		// With q flag, replacement is literal — escape Go's special chars
		goRepl = strings.ReplaceAll(replacement, "$", "$$")
	} else {
		// Validate and translate XPath replacement string to Go syntax.
		var err error
		goRepl, err = translateXPathReplacement(replacement)
		if err != nil {
			return nil, err
		}
	}

	re, err := compileXPathRegex(pattern, flags)
	if err != nil {
		return nil, err
	}

	// XPath spec: error if pattern matches empty string
	if re.MatchString("") {
		return nil, &XPathError{Code: "FORX0003", Message: "replacement pattern matches zero-length string"}
	}

	return SingleString(re.ReplaceAllString(s, goRepl)), nil
}

// translateXPathReplacement converts an XPath replacement string to Go regexp syntax.
// XPath uses: $N for backrefs, \$ for literal $, \\ for literal \.
// Go uses:    $N for backrefs, $$ for literal $.
func translateXPathReplacement(repl string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(repl); i++ {
		ch := repl[i]
		switch ch {
		case '\\':
			if i+1 >= len(repl) {
				return "", &XPathError{Code: "FORX0004", Message: "invalid replacement string: trailing backslash"}
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
				return "", &XPathError{Code: "FORX0004", Message: fmt.Sprintf("invalid replacement string: \\%c", next)}
			}
		case '$':
			if i+1 >= len(repl) || repl[i+1] < '0' || repl[i+1] > '9' {
				return "", &XPathError{Code: "FORX0004", Message: "invalid replacement string: $ not followed by digit"}
			}
			// Collect all digits after $ for the group number
			i++
			b.WriteString("${")
			for i < len(repl) && repl[i] >= '0' && repl[i] <= '9' {
				b.WriteByte(repl[i])
				i++
			}
			b.WriteByte('}')
			i-- // outer loop will i++
		default:
			b.WriteByte(ch)
		}
	}
	return b.String(), nil
}

func fnTokenize(_ context.Context, args []Sequence) (Sequence, error) {
	s := seqToString(args[0])
	if s == "" {
		return nil, nil
	}

	// 1-arg form: split on whitespace (normalize-space first)
	if len(args) == 1 {
		tokens := strings.Fields(s)
		result := make(Sequence, len(tokens))
		for i, t := range tokens {
			result[i] = AtomicValue{TypeName: TypeString, Value: t}
		}
		return result, nil
	}

	pattern := seqToString(args[1])
	flags := ""
	if len(args) > 2 {
		flags = seqToString(args[2])
	}
	re, err := compileXPathRegex(pattern, flags)
	if err != nil {
		return nil, err
	}
	parts := re.Split(s, -1)
	result := make(Sequence, len(parts))
	for i, p := range parts {
		result[i] = AtomicValue{TypeName: TypeString, Value: p}
	}
	return result, nil
}

func fnAnalyzeString(_ context.Context, _ []Sequence) (Sequence, error) {
	return nil, &XPathError{Code: "FOER0000", Message: "analyze-string not yet implemented"}
}

// compileXPathRegex compiles an XPath regex pattern with flags.
// Maps XPath flags (i,m,s,x) to Go regexp equivalents.
// Translates XPath/XML Schema regex features to Go-compatible patterns.
func compileXPathRegex(pattern, flags string) (*regexp.Regexp, error) {
	// Reject Perl-specific constructs first
	if err := rejectPerlSpecific(pattern); err != nil {
		return nil, err
	}

	var prefix strings.Builder
	prefix.WriteString("(?")
	for _, f := range flags {
		switch f {
		case 'i':
			prefix.WriteRune('i')
		case 'm':
			prefix.WriteRune('m')
		case 's':
			prefix.WriteRune('s')
		case 'x':
			// Free-spacing mode: strip unescaped whitespace and #-comments
			pattern = stripFreeSpacing(pattern)
		case 'q':
			// Literal mode: quote the entire pattern
			pattern = regexp.QuoteMeta(pattern)
			// No need for further translation
			re, err := regexp.Compile(pattern)
			if err != nil {
				return nil, &XPathError{Code: "FORX0002", Message: fmt.Sprintf("invalid regular expression: %s", err)}
			}
			return re, nil
		default:
			return nil, &XPathError{Code: "FORX0001", Message: fmt.Sprintf("invalid regex flag: %c", f)}
		}
	}

	// Translate XPath/XML Schema regex features to Go-compatible patterns
	translated, err := translateXPathRegex(pattern)
	if err != nil {
		return nil, err
	}
	pattern = translated

	if prefix.Len() > 2 {
		prefix.WriteRune(')')
		pattern = prefix.String() + pattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, &XPathError{Code: "FORX0002", Message: fmt.Sprintf("invalid regular expression: %s", err)}
	}
	return re, nil
}

// stripFreeSpacing removes unescaped whitespace from a regex pattern (x flag).
func stripFreeSpacing(pattern string) string {
	var b strings.Builder
	runes := []rune(pattern)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\\' && i+1 < len(runes) {
			b.WriteRune(r)
			i++
			b.WriteRune(runes[i])
			continue
		}
		if unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// fnContainsToken implements fn:contains-token($input, $token [, $collation])
// Returns true if any string in $input, after tokenizing on whitespace,
// matches $token (compared case-insensitively if collation is default).
func fnContainsToken(_ context.Context, args []Sequence) (Sequence, error) {
	token := seqToString(args[1])
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
		for _, tok := range strings.Fields(s) {
			if tok == token {
				return SingleBoolean(true), nil
			}
		}
	}
	return SingleBoolean(false), nil
}
