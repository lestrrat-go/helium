package xpath3

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// collationImpl provides string comparison operations for a specific collation.
type collationImpl struct {
	compare   func(a, b string) int
	indexOf   func(s, substr string) (pos, matchLen int)
	hasPrefix func(s, prefix string) (bool, int)
	hasSuffix func(s, suffix string) (bool, int)
	key       func(s string) []byte
}

// codepointCollation is the default XPath collation using byte-level comparison.
var codepointCollation = &collationImpl{
	compare: strings.Compare,
	indexOf: func(s, substr string) (int, int) {
		idx := strings.Index(s, substr)
		if idx < 0 {
			return -1, 0
		}
		return idx, len(substr)
	},
	hasPrefix: func(s, prefix string) (bool, int) {
		if strings.HasPrefix(s, prefix) {
			return true, len(prefix)
		}
		return false, 0
	},
	hasSuffix: func(s, suffix string) (bool, int) {
		if strings.HasSuffix(s, suffix) {
			return true, len(suffix)
		}
		return false, 0
	},
	key: func(s string) []byte {
		return []byte(s)
	},
}

// htmlASCIICaseInsensitiveCollation compares ASCII letters case-insensitively,
// all other characters by codepoint.
var htmlASCIICaseInsensitiveCollation = &collationImpl{
	compare: func(a, b string) int {
		return strings.Compare(foldASCIIString(a), foldASCIIString(b))
	},
	indexOf: func(s, substr string) (int, int) {
		ls := foldASCIIString(s)
		lsub := foldASCIIString(substr)
		idx := strings.Index(ls, lsub)
		if idx < 0 {
			return -1, 0
		}
		return idx, len(substr)
	},
	hasPrefix: func(s, prefix string) (bool, int) {
		if len(s) < len(prefix) {
			return false, 0
		}
		if foldASCIIString(s[:len(prefix)]) == foldASCIIString(prefix) {
			return true, len(prefix)
		}
		return false, 0
	},
	hasSuffix: func(s, suffix string) (bool, int) {
		if len(s) < len(suffix) {
			return false, 0
		}
		if foldASCIIString(s[len(s)-len(suffix):]) == foldASCIIString(suffix) {
			return true, len(suffix)
		}
		return false, 0
	},
	key: func(s string) []byte {
		return []byte(foldASCIIString(s))
	},
}

// makeUCACollation creates a collation based on the Unicode Collation Algorithm
// with optional parameters parsed from the URI query string.
func makeUCACollation(params string) *collationImpl {
	opts := parseUCAParams(params)
	if opts.caseLevel || opts.backwards {
		tagParts := []string{"und-u"}
		if opts.backwards {
			tagParts = append(tagParts, "kb-true")
		}
		if opts.caseLevel {
			tagParts = append(tagParts, "kc-true")
		}
		if extraTag, err := language.Parse(strings.Join(tagParts, "-")); err == nil {
			opts.collateOpts = append(opts.collateOpts, collate.OptionsFromTag(extraTag))
		}
	}

	cl := collate.New(opts.tag, opts.collateOpts...)
	numeric := opts.numeric
	ignoreVariables := opts.ignoreVariableRunes()
	normalize := func(s string) string {
		if !ignoreVariables {
			return s
		}
		return projectVariableRunes(s).value
	}
	cmp := func(a, b string) int {
		return cl.CompareString(normalize(a), normalize(b))
	}
	key := func(s string) []byte {
		buf := &collate.Buffer{}
		return append([]byte(nil), cl.KeyFromString(buf, normalize(s))...)
	}
	if opts.caseFirst != "" && (!opts.ignoreCase || opts.caseLevel) {
		caseBlindOpts := append([]collate.Option(nil), opts.collateOpts...)
		caseBlindOpts = append(caseBlindOpts, collate.IgnoreCase)
		caseBlind := collate.New(opts.tag, caseBlindOpts...)
		key = func(s string) []byte {
			s = normalize(s)
			buf := &collate.Buffer{}
			base := append([]byte(nil), caseBlind.KeyFromString(buf, s)...)
			caseKey := buildCaseFirstKey(s, opts.caseFirst)
			if len(caseKey) == 0 {
				return base
			}
			base = append(base, 0)
			return append(base, caseKey...)
		}
		cmp = func(a, b string) int {
			return bytes.Compare(key(a), key(b))
		}
	}

	return &collationImpl{
		compare: cmp,
		indexOf: func(s, substr string) (int, int) {
			if ignoreVariables {
				return ucaVariableIndexOf(cmp, s, substr, numeric)
			}
			return ucaIndexOf(cmp, s, substr, numeric)
		},
		hasPrefix: func(s, prefix string) (bool, int) {
			if ignoreVariables {
				return ucaVariableHasPrefix(cmp, s, prefix, numeric)
			}
			return ucaHasPrefix(cmp, s, prefix, numeric)
		},
		hasSuffix: func(s, suffix string) (bool, int) {
			if ignoreVariables {
				return ucaVariableHasSuffix(cmp, s, suffix, numeric)
			}
			return ucaHasSuffix(cmp, s, suffix, numeric)
		},
		key: key,
	}
}

type ucaParams struct {
	tag         language.Tag
	collateOpts []collate.Option
	numeric     bool
	ignoreCase  bool
	caseFirst   string
	strength    string
	alternate   string
	caseLevel   bool
	backwards   bool
}

func parseUCAParams(query string) ucaParams {
	p := ucaParams{tag: language.Und, strength: "tertiary"}

	if query == "" {
		return p
	}

	params := strings.Split(query, ";")
	for _, param := range params {
		kv := strings.SplitN(param, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key, val := kv[0], kv[1]
		switch key {
		case "lang":
			p.tag = language.Make(val)
		case "strength":
			p.strength = val
			switch val {
			case "primary":
				p.collateOpts = append(p.collateOpts, collate.IgnoreCase, collate.IgnoreDiacritics)
				p.ignoreCase = true
			case "secondary":
				p.collateOpts = append(p.collateOpts, collate.IgnoreCase)
				p.ignoreCase = true
			case "tertiary":
				// default strength, no options needed
			}
		case "alternate":
			if val == "blanked" || val == "shifted" {
				p.alternate = val
			}
		case "caseFirst":
			if val == "upper" || val == "lower" {
				p.caseFirst = val
			}
		case "caseLevel":
			p.caseLevel = val == "yes"
		case "backwards":
			p.backwards = val == "yes"
		case "numeric":
			if val == "yes" {
				p.collateOpts = append(p.collateOpts, collate.Numeric)
				p.numeric = true
			}
		}
	}
	return p
}

func (p ucaParams) ignoreVariableRunes() bool {
	switch p.alternate {
	case "blanked":
		return p.strength != "identical"
	case "shifted":
		switch p.strength {
		case "primary", "secondary", "tertiary":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func isUCAVariableRune(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r)
}

type projectedString struct {
	value  string
	starts []int
	ends   []int
}

func projectVariableRunes(s string) projectedString {
	var b strings.Builder
	var starts []int
	var ends []int
	for i, r := range s {
		if isUCAVariableRune(r) {
			continue
		}
		b.WriteRune(r)
		starts = append(starts, i)
		ends = append(ends, i+utf8.RuneLen(r))
	}
	return projectedString{
		value:  b.String(),
		starts: starts,
		ends:   ends,
	}
}

// ucaIndexOf finds substr in s using UCA collation, scanning rune by rune.
// Tries varying match lengths to handle numeric collation (e.g., "001" == "1").
func ucaIndexOf(cmp func(a, b string) int, s, substr string, numeric bool) (int, int) {
	start, end := ucaIndexOfRange(cmp, s, substr, numeric)
	if start < 0 {
		return -1, 0
	}
	sRunes := []rune(s)
	bytePos := len(string(sRunes[:start]))
	byteLen := len(string(sRunes[start:end]))
	return bytePos, byteLen
}

func ucaIndexOfRange(cmp func(a, b string) int, s, substr string, numeric bool) (int, int) {
	if substr == "" {
		return 0, 0
	}
	subRuneLen := len([]rune(substr))
	sRunes := []rune(s)

	for i := 0; i < len(sRunes); i++ {
		// Try match lengths from subRuneLen to end of string
		minLen := subRuneLen
		if minLen > len(sRunes)-i {
			minLen = len(sRunes) - i
		}
		maxLen := len(sRunes) - i
		for matchLen := minLen; matchLen <= maxLen; matchLen++ {
			candidate := string(sRunes[i : i+matchLen])
			if cmp(candidate, substr) == 0 {
				if numeric && splitsDigitRun(sRunes, i, i+matchLen) {
					continue
				}
				return i, i + matchLen
			}
		}
		// Also try shorter matches (for cases where collation compresses)
		for matchLen := minLen - 1; matchLen > 0; matchLen-- {
			candidate := string(sRunes[i : i+matchLen])
			if cmp(candidate, substr) == 0 {
				if numeric && splitsDigitRun(sRunes, i, i+matchLen) {
					continue
				}
				return i, i + matchLen
			}
		}
	}
	return -1, -1
}

// splitsDigitRun returns true if the match boundary [start, end) in runes
// would split a contiguous run of digits. This is used with numeric collation
// to avoid matching "10" inside "100".
func splitsDigitRun(runes []rune, start, end int) bool {
	// Check if start splits a digit run (digit before and at start)
	if start > 0 && isDigit(runes[start-1]) && isDigit(runes[start]) {
		return true
	}
	// Check if end splits a digit run (digit at end-1 and at end)
	if end > 0 && end < len(runes) && isDigit(runes[end-1]) && isDigit(runes[end]) {
		return true
	}
	return false
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// ucaHasPrefix checks if s starts with prefix under UCA collation.
func ucaHasPrefix(cmp func(a, b string) int, s, prefix string, numeric bool) (bool, int) {
	matchEnd, ok := ucaHasPrefixRange(cmp, s, prefix, numeric)
	if !ok {
		return false, 0
	}
	sRunes := []rune(s)
	return true, len(string(sRunes[:matchEnd]))
}

func ucaHasPrefixRange(cmp func(a, b string) int, s, prefix string, numeric bool) (int, bool) {
	prefixRunes := []rune(prefix)
	sRunes := []rune(s)
	if len(sRunes) < len(prefixRunes) {
		return 0, false
	}
	// Try match lengths from len(prefixRunes) upward to handle numeric expansion
	for matchLen := len(prefixRunes); matchLen <= len(sRunes); matchLen++ {
		candidate := string(sRunes[:matchLen])
		if cmp(candidate, prefix) == 0 {
			if numeric && splitsDigitRun(sRunes, 0, matchLen) {
				continue
			}
			return matchLen, true
		}
	}
	// Also try shorter matches
	for matchLen := len(prefixRunes) - 1; matchLen > 0; matchLen-- {
		candidate := string(sRunes[:matchLen])
		if cmp(candidate, prefix) == 0 {
			if numeric && splitsDigitRun(sRunes, 0, matchLen) {
				continue
			}
			return matchLen, true
		}
	}
	return 0, false
}

// ucaHasSuffix checks if s ends with suffix under UCA collation.
func ucaHasSuffix(cmp func(a, b string) int, s, suffix string, numeric bool) (bool, int) {
	matchStart, ok := ucaHasSuffixRange(cmp, s, suffix, numeric)
	if !ok {
		return false, 0
	}
	sRunes := []rune(s)
	return true, len(string(sRunes[matchStart:]))
}

func ucaHasSuffixRange(cmp func(a, b string) int, s, suffix string, numeric bool) (int, bool) {
	suffixRunes := []rune(suffix)
	sRunes := []rune(s)
	if len(sRunes) < len(suffixRunes) {
		return 0, false
	}
	// Try match lengths from len(suffixRunes) upward
	for matchLen := len(suffixRunes); matchLen <= len(sRunes); matchLen++ {
		start := len(sRunes) - matchLen
		candidate := string(sRunes[start:])
		if cmp(candidate, suffix) == 0 {
			if numeric && splitsDigitRun(sRunes, start, len(sRunes)) {
				continue
			}
			return start, true
		}
	}
	// Also try shorter matches
	for matchLen := len(suffixRunes) - 1; matchLen > 0; matchLen-- {
		start := len(sRunes) - matchLen
		candidate := string(sRunes[start:])
		if cmp(candidate, suffix) == 0 {
			if numeric && splitsDigitRun(sRunes, start, len(sRunes)) {
				continue
			}
			return start, true
		}
	}
	return 0, false
}

func ucaVariableIndexOf(cmp func(a, b string) int, s, substr string, numeric bool) (int, int) {
	projectedS := projectVariableRunes(s)
	projectedSub := projectVariableRunes(substr)
	if projectedSub.value == "" {
		return 0, 0
	}
	start, end := ucaIndexOfRange(cmp, projectedS.value, projectedSub.value, numeric)
	if start < 0 {
		return -1, 0
	}
	byteStart := projectedS.starts[start]
	byteEnd := projectedS.ends[end-1]
	return byteStart, byteEnd - byteStart
}

func ucaVariableHasPrefix(cmp func(a, b string) int, s, prefix string, numeric bool) (bool, int) {
	projectedS := projectVariableRunes(s)
	projectedPrefix := projectVariableRunes(prefix)
	if projectedPrefix.value == "" {
		return true, 0
	}
	matchEnd, ok := ucaHasPrefixRange(cmp, projectedS.value, projectedPrefix.value, numeric)
	if !ok {
		return false, 0
	}
	return true, projectedS.ends[matchEnd-1]
}

func ucaVariableHasSuffix(cmp func(a, b string) int, s, suffix string, numeric bool) (bool, int) {
	projectedS := projectVariableRunes(s)
	projectedSuffix := projectVariableRunes(suffix)
	if projectedSuffix.value == "" {
		return true, 0
	}
	matchStart, ok := ucaHasSuffixRange(cmp, projectedS.value, projectedSuffix.value, numeric)
	if !ok {
		return false, 0
	}
	return true, len(s) - projectedS.starts[matchStart]
}

func foldASCIIString(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b.WriteByte(c)
	}
	return b.String()
}

func buildCaseFirstKey(s, caseFirst string) []byte {
	var key []byte
	for _, r := range s {
		lower := unicode.ToLower(r)
		upper := unicode.ToUpper(r)
		if lower == upper {
			continue
		}
		switch caseFirst {
		case "lower":
			if unicode.IsLower(r) {
				key = append(key, 0x01)
			} else {
				key = append(key, 0x02)
			}
		case "upper":
			if unicode.IsUpper(r) {
				key = append(key, 0x01)
			} else {
				key = append(key, 0x02)
			}
		}
	}
	return key
}

// codepointCollationURI is the default XPath collation URI.
const codepointCollationURI = "http://www.w3.org/2005/xpath-functions/collation/codepoint"

// ucaCollationURI is the Unicode Collation Algorithm base URI.
const ucaCollationURI = "http://www.w3.org/2013/collation/UCA"

// htmlASCIICaseInsensitiveURI is the HTML ASCII case-insensitive collation URI.
const htmlASCIICaseInsensitiveURI = "http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive"

// caseblindCollationURI is the QT3 test suite's case-blind collation URI.
const caseblindCollationURI = "http://www.w3.org/2010/09/qt-fots-catalog/collation/caseblind"

// resolveCollation resolves a collation URI string to a collation implementation.
// If baseURI is non-empty, relative URIs are resolved against it.
func resolveCollation(uri, baseURI string) (*collationImpl, error) {
	// Resolve relative URI if baseURI is provided
	if baseURI != "" && !strings.Contains(uri, "://") {
		base, err := url.Parse(baseURI)
		if err == nil {
			ref, err := url.Parse(uri)
			if err == nil {
				uri = base.ResolveReference(ref).String()
			}
		}
	}

	switch {
	case uri == codepointCollationURI:
		return codepointCollation, nil
	case uri == htmlASCIICaseInsensitiveURI:
		return htmlASCIICaseInsensitiveCollation, nil
	case uri == caseblindCollationURI:
		return htmlASCIICaseInsensitiveCollation, nil
	case strings.HasPrefix(uri, ucaCollationURI):
		params := ""
		if idx := strings.Index(uri, "?"); idx >= 0 {
			params = uri[idx+1:]
		}
		return makeUCACollation(params), nil
	default:
		return nil, &XPathError{Code: errCodeFOCH0002, Message: fmt.Sprintf("unsupported collation: %s", uri)}
	}
}
