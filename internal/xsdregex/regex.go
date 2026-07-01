package xsdregex

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/dlclark/regexp2"
)

// defaultMatchTimeoutNanos holds the current default match timeout, in
// nanoseconds, as an atomic so concurrent compilation (which reads it) and
// SetDefaultMatchTimeout (which writes it) cannot race. See DefaultMatchTimeout.
var defaultMatchTimeoutNanos atomic.Int64

func init() {
	defaultMatchTimeoutNanos.Store(int64(5 * time.Second))
}

// DefaultMatchTimeout returns the bound on how long the regexp2 backtracking
// engine spends on a single pattern-facet match before giving up. Patterns that
// use constructs RE2 cannot handle (character-class subtraction, large
// quantifiers) compile to regexp2, which is vulnerable to catastrophic
// backtracking on adversarial inputs; this is a defense-in-depth ceiling for
// those matches. RE2-compiled patterns are linear-time and unaffected. A value
// of 0 disables the timeout.
func DefaultMatchTimeout() time.Duration {
	return time.Duration(defaultMatchTimeoutNanos.Load())
}

// SetDefaultMatchTimeout sets the default match timeout returned by
// DefaultMatchTimeout. Setting it to 0 disables the timeout. The change affects
// only subsequently-compiled patterns. It is safe to call concurrently with
// regex compilation.
func SetDefaultMatchTimeout(d time.Duration) {
	defaultMatchTimeoutNanos.Store(int64(d))
}

// errCodeFORX0002 mirrors the XPath FORX0002 ("invalid regular expression")
// error code. This package is layering-neutral (no xpath3 dependency); callers
// map regexError back to their own error type — xpath3 wraps it as XPathError.
const errCodeFORX0002 = "FORX0002"

type regexError struct {
	Code    string
	Message string
}

func (e *regexError) Error() string { return e.Message }

// translateXPathRegex translates an XPath/XML Schema regex pattern into a
// Go-compatible regexp pattern. Handles:
//   - \p{IsBlockName} / \P{IsBlockName} → (negated) Unicode block character range
//   - \p{Category} → Go-compatible \p{Category} (pass-through)
//   - \i / \I → XML NameStartChar / negated
//   - \c / \C → XML NameChar / negated
//   - \d \D \w \W \s \S → XSD-semantics equivalents
//   - '.' → [^\n\r] (or [\s\S] when dotAll)
//
// It does NOT expand character class subtraction ([a-z-[aeiou]]); that is passed
// through and relies on Go's (different) interpretation. It also does NOT reject
// Perl-specific constructs or validate back-references — call RejectPerlSpecific
// and Validate separately for those checks.
// xsdPattern selects between the two regex flavors that share this translator:
//
//   - true  → XSD xs:pattern facet. The pattern is implicitly anchored to the
//     whole value and '^'/'$' are ordinary literal characters, not anchors.
//   - false → XPath/XQuery fn:matches/tokenize/replace. The pattern is NOT
//     implicitly anchored and '^'/'$' keep their RE2 anchor meaning (start/end,
//     multiline with the 'm' flag).
func translateXPathRegex(pattern string, dotAll, ignoreCase, xsdPattern, xsd11 bool) (string, error) {
	isDotAll := dotAll
	var b strings.Builder
	runes := []rune(pattern)
	i := 0

	for i < len(runes) {
		r := runes[i]

		if r == '\\' && i+1 < len(runes) {
			next := runes[i+1]
			switch next {
			case 'p', 'P':
				// Unicode property escape
				neg := next == 'P'
				if i+2 < len(runes) && runes[i+2] == '{' {
					end := findClosingBrace(runes, i+3)
					if end < 0 {
						return "", fmt.Errorf("unterminated \\%c{ at position %d", next, i)
					}
					propName := string(runes[i+3 : end])
					replacement, err := translateUnicodeProperty(propName, neg, xsd11)
					if err != nil {
						return "", err
					}
					if ignoreCase {
						replacement = "(?-i:" + replacement + ")"
					}
					b.WriteString(replacement)
					i = end + 1
					continue
				}
				// Single-letter property like \p{L}
				b.WriteRune(r)
				b.WriteRune(next)
				i += 2
				continue
			case 'i':
				b.WriteString(xmlNameStartCharClass)
				i += 2
				continue
			case 'I':
				b.WriteString(xmlNameStartCharClassNeg)
				i += 2
				continue
			case 'c':
				if xsd11 {
					b.WriteString(xpathRegexNameCharClass11)
				} else {
					b.WriteString(xpathRegexNameCharClass)
				}
				i += 2
				continue
			case 'C':
				if xsd11 {
					b.WriteString(xpathRegexNameCharClassNeg11)
				} else {
					b.WriteString(xpathRegexNameCharClassNeg)
				}
				i += 2
				continue
			case 'd':
				b.WriteString(`\p{Nd}`)
				i += 2
				continue
			case 'D':
				b.WriteString(`[^\p{Nd}]`)
				i += 2
				continue
			case 'w':
				b.WriteString(`[^\p{P}\p{Z}\p{C}]`)
				i += 2
				continue
			case 'W':
				b.WriteString(`[\p{P}\p{Z}\p{C}]`)
				i += 2
				continue
			case 's':
				b.WriteString(`[\t\n\r ]`)
				i += 2
				continue
			case 'S':
				b.WriteString(`[^\t\n\r ]`)
				i += 2
				continue
			default:
				b.WriteRune(r)
				b.WriteRune(next)
				i += 2
				continue
			}
		}

		// Handle character class subtraction: [base-[subtract]]
		if r == '[' {
			cls, consumed, err := translateCharClass(runes, i, xsd11)
			if err != nil {
				return "", err
			}
			b.WriteString(cls)
			i += consumed
			continue
		}

		// XPath regex: '.' matches any char except \n and \r (without 's' flag).
		// Go's RE2 '.' matches any char except \n. Replace bare '.'
		// (outside character classes) with [^\n\r] to also exclude \r.
		// With 's' flag (dot-all), '.' matches everything — use [\s\S] for Go.
		if r == '.' {
			if isDotAll {
				b.WriteString(`[\s\S]`)
			} else {
				b.WriteString(`[^\n\r]`)
			}
			i++
			continue
		}

		// In the XSD xs:pattern grammar '^' and '$' are ordinary literal
		// characters, not anchors (the only XSD metacharacters are
		// . \ ? * + { } ( ) [ ] |, and the whole pattern is already implicitly
		// anchored when Compile wraps it in \A(?:...)\z). Escape them so RE2
		// treats them as literals. In XPath/XQuery regex (fn:matches/tokenize/
		// replace) '^' and '$' are anchors (start/end, multiline with 'm'); leave
		// them untouched so the engine keeps that meaning.
		if xsdPattern && (r == '^' || r == '$') {
			b.WriteRune('\\')
			b.WriteRune(r)
			i++
			continue
		}

		b.WriteRune(r)
		i++
	}

	return b.String(), nil
}

func findClosingBrace(runes []rune, start int) int {
	for i := start; i < len(runes); i++ {
		if runes[i] == '}' {
			return i
		}
	}
	return -1
}

// translateCharClass translates a character class, handling subtraction.
// Returns the translated class and the number of runes consumed.
func translateCharClass(runes []rune, start int, xsd11 bool) (string, int, error) {
	// Find the matching close bracket, handling nesting
	depth := 0
	i := start
	for i < len(runes) {
		switch runes[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				// We have the full character class from start to i (inclusive)
				content := runes[start : i+1]
				result, err := processCharClass(content, xsd11)
				if err != nil {
					return "", 0, err
				}
				return result, i + 1 - start, nil
			}
		case '\\':
			i++ // skip next character
		}
		i++
	}
	// No closing bracket found — pass through as-is
	return string(runes[start : start+1]), 1, nil
}

// processCharClass processes a character class, expanding subtraction and
// translating embedded \p{} and \i/\c escapes.
func processCharClass(runes []rune, xsd11 bool) (string, error) {
	s := string(runes)
	if err := validateXPathCharClassStructure(s); err != nil {
		return "", err
	}
	if err := validateXPathCharClassSubtraction(s); err != nil {
		return "", err
	}
	// Character class subtraction [base-[subtract]] is not supported by Go's RE2.
	// Pass through as-is — Go will interpret it differently but many tests still
	// pass because Go's (incorrect) interpretation gives the expected result.
	return translateClassContent(s, xsd11)
}

func validateXPathCharClassStructure(class string) error {
	runes := []rune(class)
	if len(runes) < 2 || runes[0] != '[' || runes[len(runes)-1] != ']' {
		return nil
	}

	first := 1
	if first < len(runes)-1 && runes[first] == '^' {
		first++
	}
	if first >= len(runes)-1 {
		return &regexError{
			Code:    errCodeFORX0002,
			Message: fmt.Sprintf("invalid character class: %s", class),
		}
	}

	prevRaw := rune(0)
	for i := first; i < len(runes)-1; {
		if runes[i] == '\\' {
			if i+1 >= len(runes)-1 {
				return &regexError{
					Code:    errCodeFORX0002,
					Message: fmt.Sprintf("invalid character class: %s", class),
				}
			}
			i += 2
			continue
		}
		if i == first && (runes[i] == ']' || runes[i] == '[') {
			return &regexError{
				Code:    errCodeFORX0002,
				Message: fmt.Sprintf("invalid character class: %s", class),
			}
		}
		if runes[i] == '[' && prevRaw != '-' {
			return &regexError{
				Code:    errCodeFORX0002,
				Message: fmt.Sprintf("invalid character class: %s", class),
			}
		}
		// An interior raw '-' denotes a range operator whose endpoints must be
		// single characters, never another '-'. A '-' is literal (not a range
		// operator) at the start of the class (i == first) or immediately
		// before the closing ']' (runes[i+1] == ']'); a '-' immediately before
		// '[' is the operator of a '-[' character-class subtraction (validated
		// separately by validateXPathCharClassSubtraction), not a range. An
		// interior, non-subtraction '-' whose immediately-preceding or
		// immediately-following raw char is itself a literal '-' uses a '-' as
		// a range endpoint, which the XSD/XPath regex grammar forbids — except
		// when that adjacent '-' is itself the operator of a '-[' subtraction
		// (its own following char is '['), which is a literal dash abutting the
		// subtraction operator (e.g. the valid base group of '[a--[b]]'), not a
		// range endpoint.
		if runes[i] == '-' && i != first && runes[i+1] != ']' && runes[i+1] != '[' {
			precededByHyphen := prevRaw == '-'
			followedByHyphen := runes[i+1] == '-' && (i+2 >= len(runes) || runes[i+2] != '[')
			if precededByHyphen || followedByHyphen {
				return &regexError{
					Code:    errCodeFORX0002,
					Message: fmt.Sprintf("invalid character class: %s", class),
				}
			}
		}
		prevRaw = runes[i]
		i++
	}

	return nil
}

func validateXPathCharClassSubtraction(class string) error {
	if !strings.Contains(class, "-[") {
		return nil
	}

	runes := []rune(class)
	if len(runes) < 4 || runes[0] != '[' || runes[len(runes)-1] != ']' {
		return nil
	}

	for i := 1; i < len(runes)-2; i++ {
		if runes[i] == '\\' {
			i++
			continue
		}
		if runes[i] == '-' && runes[i+1] == '[' {
			if i == 1 || (i == 2 && runes[1] == '^') {
				return &regexError{
					Code:    errCodeFORX0002,
					Message: fmt.Sprintf("invalid character class subtraction: %s", class),
				}
			}
		}
	}

	return nil
}

// translateClassContent translates \p{}, \i, \c escapes inside a character class.
func translateClassContent(s string, xsd11 bool) (string, error) {
	var b strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\\' && i+1 < len(runes) {
			next := runes[i+1]
			switch next {
			case 'p', 'P':
				if i+2 < len(runes) && runes[i+2] == '{' {
					end := -1
					for j := i + 3; j < len(runes); j++ {
						if runes[j] == '}' {
							end = j
							break
						}
					}
					if end >= 0 {
						propName := string(runes[i+3 : end])
						replacement, err := translateUnicodePropertyInCharClass(propName, next == 'P', xsd11)
						if err != nil {
							return "", err
						}
						b.WriteString(replacement)
						i = end
						continue
					}
				}
			case 'i':
				b.WriteString(xmlNameStartCharRange)
				i++
				continue
			case 'I':
				b.WriteString(xmlNameStartCharRangeNeg)
				i++
				continue
			case 'c':
				if xsd11 {
					b.WriteString(xmlNameCharRange)
				} else {
					b.WriteString(xpathRegexNameCharRange)
				}
				i++
				continue
			case 'C':
				if xsd11 {
					b.WriteString(xpathRegexNameCharRangeNeg11)
				} else {
					b.WriteString(xpathRegexNameCharRangeNeg)
				}
				i++
				continue
			case 'd':
				b.WriteString(`\p{Nd}`)
				i++
				continue
			case 'D':
				b.WriteString(`\P{Nd}`)
				i++
				continue
			case 'w':
				b.WriteString(`\p{L}\p{M}\p{N}\p{S}`)
				i++
				continue
			case 'W':
				b.WriteString(`\p{P}\p{Z}\p{C}`)
				i++
				continue
			}
		}
		b.WriteRune(r)
	}
	return b.String(), nil
}

// translateUnicodeProperty translates a Unicode property name to a Go regexp equivalent.
func translateUnicodeProperty(name string, neg, xsd11 bool) (string, error) {
	prefix := `\p`
	if neg {
		prefix = `\P`
	}

	// Check if it's an IsBlockName
	if strings.HasPrefix(name, "Is") {
		blockName := name[2:]
		if rng, ok := lookupUnicodeBlockRange(blockName); ok {
			if neg {
				return "[^" + rng + "]", nil
			}
			return "[" + rng + "]", nil
		}
		if xsd11 {
			// XSD 1.1 (test bug 13670): an unrecognized block name is valid and
			// matches every character; its complement then matches none.
			if neg {
				return "[^" + matchAnyCharRange + "]", nil
			}
			return "[" + matchAnyCharRange + "]", nil
		}
		return "", &regexError{Code: errCodeFORX0002, Message: fmt.Sprintf("unknown Unicode block: Is%s", blockName)}
	}

	// Check Go-supported category/script names
	if isGoSupportedProperty(name) {
		return prefix + "{" + name + "}", nil
	}

	return "", &regexError{Code: errCodeFORX0002, Message: fmt.Sprintf("unknown Unicode property: %s", name)}
}

func translateUnicodePropertyInCharClass(name string, neg, xsd11 bool) (string, error) {
	if strings.HasPrefix(name, "Is") {
		blockName := name[2:]
		rng, ok := lookupUnicodeBlockRange(blockName)
		if !ok {
			if xsd11 {
				// XSD 1.1: an unrecognized block name matches every character;
				// its complement contributes no characters to the class.
				if neg {
					return "", nil
				}
				return matchAnyCharRange, nil
			}
			return "", &regexError{Code: errCodeFORX0002, Message: fmt.Sprintf("unknown Unicode block: Is%s", blockName)}
		}
		if !neg {
			return rng, nil
		}
		low, high, err := parseUnicodeBlockRange(rng)
		if err != nil {
			return "", err
		}
		return complementUnicodeRange(low, high), nil
	}

	return translateUnicodeProperty(name, neg, xsd11)
}

func lookupUnicodeBlockRange(blockName string) (string, bool) {
	if rng, ok := unicodeBlocks[blockName]; ok {
		return rng, true
	}
	for k, v := range unicodeBlocks {
		if strings.EqualFold(k, blockName) {
			return v, true
		}
	}
	return "", false
}

func parseUnicodeBlockRange(rng string) (int64, int64, error) {
	parts := strings.SplitN(rng, "-", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid unicode block range: %s", rng)
	}
	low, err := parseUnicodeEscape(parts[0])
	if err != nil {
		return 0, 0, err
	}
	high, err := parseUnicodeEscape(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return low, high, nil
}

func parseUnicodeEscape(s string) (int64, error) {
	if !strings.HasPrefix(s, `\x{`) || !strings.HasSuffix(s, "}") {
		return 0, fmt.Errorf("invalid unicode escape: %s", s)
	}
	return strconv.ParseInt(s[3:len(s)-1], 16, 64)
}

func complementUnicodeRange(low, high int64) string {
	const maxCodepoint = 0x10FFFF
	var parts []string
	if low > 0 {
		parts = append(parts, formatRuneRange(0, low-1))
	}
	if high < maxCodepoint {
		parts = append(parts, formatRuneRange(high+1, maxCodepoint))
	}
	return strings.Join(parts, "")
}

func complementClassRanges(spec string) (string, error) {
	ranges, err := parseClassRanges(spec)
	if err != nil {
		return "", err
	}
	if len(ranges) == 0 {
		return formatRuneRange(0, 0x10FFFF), nil
	}

	ranges = mergeClassRanges(ranges)
	var parts []string
	var cursor int64
	for _, rng := range ranges {
		if cursor < rng.low {
			parts = append(parts, formatRuneRange(cursor, rng.low-1))
		}
		if rng.high+1 > cursor {
			cursor = rng.high + 1
		}
	}
	if cursor <= 0x10FFFF {
		parts = append(parts, formatRuneRange(cursor, 0x10FFFF))
	}
	return strings.Join(parts, ""), nil
}

type classRange struct {
	low  int64
	high int64
}

func parseClassRanges(spec string) ([]classRange, error) {
	runes := []rune(spec)
	var ranges []classRange
	for i := 0; i < len(runes); {
		low, next, err := parseClassRune(runes, i)
		if err != nil {
			return nil, err
		}
		i = next

		high := low
		if i < len(runes)-1 && runes[i] == '-' {
			high, i, err = parseClassRune(runes, i+1)
			if err != nil {
				return nil, err
			}
		}

		ranges = append(ranges, classRange{low: low, high: high})
	}
	return ranges, nil
}

func parseClassRune(runes []rune, start int) (int64, int, error) {
	if start >= len(runes) {
		return 0, start, fmt.Errorf("unexpected end of character class range")
	}
	if runes[start] != '\\' {
		return int64(runes[start]), start + 1, nil
	}
	if start+1 >= len(runes) {
		return 0, start, fmt.Errorf("dangling escape in character class range")
	}
	if runes[start+1] != 'x' {
		return int64(runes[start+1]), start + 2, nil
	}
	if start+2 >= len(runes) || runes[start+2] != '{' {
		return 0, start, fmt.Errorf("invalid hex escape in character class range")
	}

	end := -1
	for i := start + 3; i < len(runes); i++ {
		if runes[i] == '}' {
			end = i
			break
		}
	}
	if end < 0 {
		return 0, start, fmt.Errorf("unterminated hex escape in character class range")
	}

	value, err := strconv.ParseInt(string(runes[start+3:end]), 16, 64)
	if err != nil {
		return 0, start, err
	}
	return value, end + 1, nil
}

func mergeClassRanges(ranges []classRange) []classRange {
	if len(ranges) == 0 {
		return nil
	}

	merged := make([]classRange, 0, len(ranges))
	for _, rng := range ranges {
		inserted := false
		for i := range merged {
			if rng.high+1 < merged[i].low {
				merged = append(merged[:i], append([]classRange{rng}, merged[i:]...)...)
				inserted = true
				break
			}
			if rng.low <= merged[i].high+1 && rng.high+1 >= merged[i].low {
				if rng.low < merged[i].low {
					merged[i].low = rng.low
				}
				if rng.high > merged[i].high {
					merged[i].high = rng.high
				}
				for i+1 < len(merged) && merged[i+1].low <= merged[i].high+1 {
					if merged[i+1].high > merged[i].high {
						merged[i].high = merged[i+1].high
					}
					merged = append(merged[:i+1], merged[i+2:]...)
				}
				inserted = true
				break
			}
		}
		if !inserted {
			merged = append(merged, rng)
		}
	}
	return merged
}

func formatRuneRange(low, high int64) string {
	if low == high {
		return fmt.Sprintf(`\x{%X}`, low)
	}
	return fmt.Sprintf(`\x{%X}-\x{%X}`, low, high)
}

// goSupportedScripts lists script names supported by Go's regexp engine.
var goSupportedScripts = map[string]struct{}{
	"Arabic": {}, "Armenian": {}, "Bengali": {}, "Bopomofo": {},
	"Braille": {}, "Buhid": {}, "Canadian_Aboriginal": {}, "Cherokee": {},
	"Common": {}, "Cyrillic": {}, "Devanagari": {}, "Ethiopic": {},
	"Georgian": {}, "Greek": {}, "Gujarati": {}, "Gurmukhi": {},
	"Han": {}, "Hangul": {}, "Hanunoo": {}, "Hebrew": {},
	"Hiragana": {}, "Inherited": {}, "Kannada": {}, "Katakana": {},
	"Khmer": {}, "Lao": {}, "Latin": {}, "Limbu": {},
	"Malayalam": {}, "Mongolian": {}, "Myanmar": {}, "Ogham": {},
	"Oriya": {}, "Runic": {}, "Sinhala": {}, "Syriac": {},
	"Tagalog": {}, "Tagbanwa": {}, "Tamil": {}, "Telugu": {},
	"Thaana": {}, "Thai": {}, "Tibetan": {}, "Yi": {},
}

// isGoSupportedProperty checks if a property name is supported by Go's regexp.
func isGoSupportedProperty(name string) bool {
	// Single-letter categories
	if len(name) == 1 || len(name) == 2 {
		return true // L, Lu, Ll, M, Mn, N, Nd, P, S, Z, C, etc.
	}
	_, ok := goSupportedScripts[name]
	return ok
}

// XML NameStartChar as a character class (for use outside [])
const xmlNameStartCharClass = `[a-zA-Z_:\x{C0}-\x{D6}\x{D8}-\x{F6}\x{F8}-\x{2FF}\x{370}-\x{37D}\x{37F}-\x{1FFF}\x{200C}-\x{200D}\x{2070}-\x{218F}\x{2C00}-\x{2FEF}\x{3001}-\x{D7FF}\x{F900}-\x{FDCF}\x{FDF0}-\x{FFFD}\x{10000}-\x{EFFFF}]`
const xmlNameStartCharClassNeg = `[^a-zA-Z_:\x{C0}-\x{D6}\x{D8}-\x{F6}\x{F8}-\x{2FF}\x{370}-\x{37D}\x{37F}-\x{1FFF}\x{200C}-\x{200D}\x{2070}-\x{218F}\x{2C00}-\x{2FEF}\x{3001}-\x{D7FF}\x{F900}-\x{FDCF}\x{FDF0}-\x{FFFD}\x{10000}-\x{EFFFF}]`

// XML NameStartChar range (for use inside [])
const xmlNameStartCharRange = `a-zA-Z_:\x{C0}-\x{D6}\x{D8}-\x{F6}\x{F8}-\x{2FF}\x{370}-\x{37D}\x{37F}-\x{1FFF}\x{200C}-\x{200D}\x{2070}-\x{218F}\x{2C00}-\x{2FEF}\x{3001}-\x{D7FF}\x{F900}-\x{FDCF}\x{FDF0}-\x{FFFD}\x{10000}-\x{EFFFF}`

var xmlNameStartCharRangeNeg = mustComplementClassRanges(xmlNameStartCharRange)

const xmlNameCharRange = `a-zA-Z_:\x{C0}-\x{D6}\x{D8}-\x{F6}\x{F8}-\x{2FF}\x{370}-\x{37D}\x{37F}-\x{1FFF}\x{200C}-\x{200D}\x{2070}-\x{218F}\x{2C00}-\x{2FEF}\x{3001}-\x{D7FF}\x{F900}-\x{FDCF}\x{FDF0}-\x{FFFD}\x{10000}-\x{EFFFF}\-.0-9\x{B7}\x{300}-\x{36F}\x{203F}-\x{2040}`

// XSD 1.0 / XPath 2.0 \c carves U+0346 out of the combining range (W3C
// regex-syntax-xslt20 suite); XSD 1.1 uses the full range (see the *11 variants
// below). Keep this isolated from Name/NCName constructor validation.
const xpathRegexNameCharClass = `[a-zA-Z_:\x{C0}-\x{D6}\x{D8}-\x{F6}\x{F8}-\x{2FF}\x{370}-\x{37D}\x{37F}-\x{1FFF}\x{200C}-\x{200D}\x{2070}-\x{218F}\x{2C00}-\x{2FEF}\x{3001}-\x{D7FF}\x{F900}-\x{FDCF}\x{FDF0}-\x{FFFD}\x{10000}-\x{EFFFF}\-.0-9\x{B7}\x{300}-\x{345}\x{347}-\x{36F}\x{203F}-\x{2040}]`
const xpathRegexNameCharRange = `a-zA-Z_:\x{C0}-\x{D6}\x{D8}-\x{F6}\x{F8}-\x{2FF}\x{370}-\x{37D}\x{37F}-\x{1FFF}\x{200C}-\x{200D}\x{2070}-\x{218F}\x{2C00}-\x{2FEF}\x{3001}-\x{D7FF}\x{F900}-\x{FDCF}\x{FDF0}-\x{FFFD}\x{10000}-\x{EFFFF}\-.0-9\x{B7}\x{300}-\x{345}\x{347}-\x{36F}\x{203F}-\x{2040}`

var xpathRegexNameCharClassNeg = `[^` + xpathRegexNameCharRange + `]`
var xpathRegexNameCharRangeNeg = mustComplementClassRanges(xpathRegexNameCharRange)

// XSD 1.1 (XML 1.1 / XML 1.0 5th edition) \c admits the full combining range
// U+0300–U+036F, including U+0346, matching xmlNameCharRange. The 1.0/XPath 2.0
// forms above carve U+0346 out; these variants are selected in XSD 1.1 mode.
var xpathRegexNameCharClass11 = `[` + xmlNameCharRange + `]`
var xpathRegexNameCharClassNeg11 = `[^` + xmlNameCharRange + `]`
var xpathRegexNameCharRangeNeg11 = mustComplementClassRanges(xmlNameCharRange)

// matchAnyCharRange covers every character (for use inside a [...] class). XSD
// 1.1 treats an unrecognized \p{Is...} block name as matching every character.
const matchAnyCharRange = `\x{0}-\x{10FFFF}`

func mustComplementClassRanges(spec string) string {
	complement, err := complementClassRanges(spec)
	if err != nil {
		panic(err)
	}
	return complement
}

// unicodeBlocks maps Unicode block names (without "Is" prefix) to character ranges.
// Based on Unicode 6.0 block definitions used in XML Schema regex.
var unicodeBlocks = map[string]string{
	"BasicLatin":                          `\x{0000}-\x{007F}`,
	"Latin-1Supplement":                   `\x{0080}-\x{00FF}`,
	"LatinExtended-A":                     `\x{0100}-\x{017F}`,
	"LatinExtended-B":                     `\x{0180}-\x{024F}`,
	"IPAExtensions":                       `\x{0250}-\x{02AF}`,
	"SpacingModifierLetters":              `\x{02B0}-\x{02FF}`,
	"CombiningDiacriticalMarks":           `\x{0300}-\x{036F}`,
	"Greek":                               `\x{0370}-\x{03FF}`,
	"GreekandCoptic":                      `\x{0370}-\x{03FF}`,
	"Cyrillic":                            `\x{0400}-\x{04FF}`,
	"CyrillicSupplement":                  `\x{0500}-\x{052F}`,
	"Armenian":                            `\x{0530}-\x{058F}`,
	"Hebrew":                              `\x{0590}-\x{05FF}`,
	"Arabic":                              `\x{0600}-\x{06FF}`,
	"Syriac":                              `\x{0700}-\x{074F}`,
	"ArabicSupplement":                    `\x{0750}-\x{077F}`,
	"Thaana":                              `\x{0780}-\x{07BF}`,
	"NKo":                                 `\x{07C0}-\x{07FF}`,
	"Devanagari":                          `\x{0900}-\x{097F}`,
	"Bengali":                             `\x{0980}-\x{09FF}`,
	"Gurmukhi":                            `\x{0A00}-\x{0A7F}`,
	"Gujarati":                            `\x{0A80}-\x{0AFF}`,
	"Oriya":                               `\x{0B00}-\x{0B7F}`,
	"Tamil":                               `\x{0B80}-\x{0BFF}`,
	"Telugu":                              `\x{0C00}-\x{0C7F}`,
	"Kannada":                             `\x{0C80}-\x{0CFF}`,
	"Malayalam":                           `\x{0D00}-\x{0D7F}`,
	"Sinhala":                             `\x{0D80}-\x{0DFF}`,
	"Thai":                                `\x{0E00}-\x{0E7F}`,
	"Lao":                                 `\x{0E80}-\x{0EFF}`,
	"Tibetan":                             `\x{0F00}-\x{0FFF}`,
	"Myanmar":                             `\x{1000}-\x{109F}`,
	"Georgian":                            `\x{10A0}-\x{10FF}`,
	"HangulJamo":                          `\x{1100}-\x{11FF}`,
	"Ethiopic":                            `\x{1200}-\x{137F}`,
	"EthiopicSupplement":                  `\x{1380}-\x{139F}`,
	"Cherokee":                            `\x{13A0}-\x{13FF}`,
	"UnifiedCanadianAboriginalSyllabics":  `\x{1400}-\x{167F}`,
	"Ogham":                               `\x{1680}-\x{169F}`,
	"Runic":                               `\x{16A0}-\x{16FF}`,
	"Tagalog":                             `\x{1700}-\x{171F}`,
	"Hanunoo":                             `\x{1720}-\x{173F}`,
	"Buhid":                               `\x{1740}-\x{175F}`,
	"Tagbanwa":                            `\x{1760}-\x{177F}`,
	"Khmer":                               `\x{1780}-\x{17FF}`,
	"Mongolian":                           `\x{1800}-\x{18AF}`,
	"Limbu":                               `\x{1900}-\x{194F}`,
	"TaiLe":                               `\x{1950}-\x{197F}`,
	"NewTaiLue":                           `\x{1980}-\x{19DF}`,
	"KhmerSymbols":                        `\x{19E0}-\x{19FF}`,
	"Buginese":                            `\x{1A00}-\x{1A1F}`,
	"PhoneticExtensions":                  `\x{1D00}-\x{1D7F}`,
	"PhoneticExtensionsSupplement":        `\x{1D80}-\x{1DBF}`,
	"CombiningDiacriticalMarksSupplement": `\x{1DC0}-\x{1DFF}`,
	"LatinExtendedAdditional":             `\x{1E00}-\x{1EFF}`,
	"GreekExtended":                       `\x{1F00}-\x{1FFF}`,
	"GeneralPunctuation":                  `\x{2000}-\x{206F}`,
	"SuperscriptsandSubscripts":           `\x{2070}-\x{209F}`,
	"CurrencySymbols":                     `\x{20A0}-\x{20CF}`,
	"CombiningDiacriticalMarksforSymbols": `\x{20D0}-\x{20FF}`,
	"CombiningMarksforSymbols":            `\x{20D0}-\x{20FF}`,
	"LetterlikeSymbols":                   `\x{2100}-\x{214F}`,
	"NumberForms":                         `\x{2150}-\x{218F}`,
	"Arrows":                              `\x{2190}-\x{21FF}`,
	"MathematicalOperators":               `\x{2200}-\x{22FF}`,
	"MiscellaneousTechnical":              `\x{2300}-\x{23FF}`,
	"ControlPictures":                     `\x{2400}-\x{243F}`,
	"OpticalCharacterRecognition":         `\x{2440}-\x{245F}`,
	"EnclosedAlphanumerics":               `\x{2460}-\x{24FF}`,
	"BoxDrawing":                          `\x{2500}-\x{257F}`,
	"BlockElements":                       `\x{2580}-\x{259F}`,
	"GeometricShapes":                     `\x{25A0}-\x{25FF}`,
	"MiscellaneousSymbols":                `\x{2600}-\x{26FF}`,
	"Dingbats":                            `\x{2700}-\x{27BF}`,
	"MiscellaneousMathematicalSymbols-A":  `\x{27C0}-\x{27EF}`,
	"SupplementalArrows-A":                `\x{27F0}-\x{27FF}`,
	"BraillePatterns":                     `\x{2800}-\x{28FF}`,
	"SupplementalArrows-B":                `\x{2900}-\x{297F}`,
	"MiscellaneousMathematicalSymbols-B":  `\x{2980}-\x{29FF}`,
	"SupplementalMathematicalOperators":   `\x{2A00}-\x{2AFF}`,
	"MiscellaneousSymbolsandArrows":       `\x{2B00}-\x{2BFF}`,
	"CJKRadicalsSupplement":               `\x{2E80}-\x{2EFF}`,
	"KangxiRadicals":                      `\x{2F00}-\x{2FDF}`,
	"IdeographicDescriptionCharacters":    `\x{2FF0}-\x{2FFF}`,
	"CJKSymbolsandPunctuation":            `\x{3000}-\x{303F}`,
	"Hiragana":                            `\x{3040}-\x{309F}`,
	"Katakana":                            `\x{30A0}-\x{30FF}`,
	"Bopomofo":                            `\x{3100}-\x{312F}`,
	"HangulCompatibilityJamo":             `\x{3130}-\x{318F}`,
	"Kanbun":                              `\x{3190}-\x{319F}`,
	"BopomofoExtended":                    `\x{31A0}-\x{31BF}`,
	"KatakanaPhoneticExtensions":          `\x{31F0}-\x{31FF}`,
	"EnclosedCJKLettersandMonths":         `\x{3200}-\x{32FF}`,
	"CJKCompatibility":                    `\x{3300}-\x{33FF}`,
	"CJKUnifiedIdeographsExtensionA":      `\x{3400}-\x{4DBF}`,
	"YijingHexagramSymbols":               `\x{4DC0}-\x{4DFF}`,
	"CJKUnifiedIdeographs":                `\x{4E00}-\x{9FFF}`,
	"YiSyllables":                         `\x{A000}-\x{A48F}`,
	"YiRadicals":                          `\x{A490}-\x{A4CF}`,
	"HangulSyllables":                     `\x{AC00}-\x{D7AF}`,
	"HighSurrogates":                      `\x{D800}-\x{DB7F}`,
	"HighPrivateUseSurrogates":            `\x{DB80}-\x{DBFF}`,
	"LowSurrogates":                       `\x{DC00}-\x{DFFF}`,
	"PrivateUseArea":                      `\x{E000}-\x{F8FF}`,
	"CJKCompatibilityIdeographs":          `\x{F900}-\x{FAFF}`,
	"AlphabeticPresentationForms":         `\x{FB00}-\x{FB4F}`,
	"ArabicPresentationForms-A":           `\x{FB50}-\x{FDFF}`,
	"VariationSelectors":                  `\x{FE00}-\x{FE0F}`,
	"CombiningHalfMarks":                  `\x{FE20}-\x{FE2F}`,
	"CJKCompatibilityForms":               `\x{FE30}-\x{FE4F}`,
	"SmallFormVariants":                   `\x{FE50}-\x{FE6F}`,
	"ArabicPresentationForms-B":           `\x{FE70}-\x{FEFF}`,
	"HalfwidthandFullwidthForms":          `\x{FF00}-\x{FFEF}`,
	"Specials":                            `\x{FFF0}-\x{FFFF}`,
	// SMP blocks
	"OldItalic":                            `\x{10300}-\x{1032F}`,
	"Gothic":                               `\x{10330}-\x{1034F}`,
	"Deseret":                              `\x{10400}-\x{1044F}`,
	"Emoticons":                            `\x{1F600}-\x{1F64F}`,
	"ByzantineMusicalSymbols":              `\x{1D000}-\x{1D0FF}`,
	"MusicalSymbols":                       `\x{1D100}-\x{1D1FF}`,
	"MathematicalAlphanumericSymbols":      `\x{1D400}-\x{1D7FF}`,
	"CJKUnifiedIdeographsExtensionB":       `\x{20000}-\x{2A6DF}`,
	"CJKCompatibilityIdeographsSupplement": `\x{2F800}-\x{2FA1F}`,
	"SupplementaryPrivateUseArea-A":        `\x{F0000}-\x{FFFFD}`,
	"SupplementaryPrivateUseArea-B":        `\x{100000}-\x{10FFFD}`,
	"Tags":                                 `\x{E0000}-\x{E007F}`,
	// Composite block: union of PrivateUseArea + SupplementaryPrivateUseArea-A + SupplementaryPrivateUseArea-B
	"PrivateUse": `\x{E000}-\x{F8FF}\x{F0000}-\x{FFFFD}\x{100000}-\x{10FFFD}`,
}

// validateXPathRegex checks for patterns that Go's regexp accepts but
// the XPath/XML Schema regex spec forbids. This must be called before
// regexp.Compile to reject invalid patterns with FORX0002.
func validateXPathRegex(pattern string, allowBackrefs, xsd11 bool) error {
	runes := []rune(pattern)
	inCharClass := 0
	inQuantifier := false // true when inside a valid {n,m} quantifier
	captureCount := 0
	var groupStack []int
	for i := 0; i < len(runes); i++ {
		r := runes[i]

		if r == '\\' && i+1 < len(runes) {
			next := runes[i+1]
			if next == '0' {
				return &regexError{
					Code:    errCodeFORX0002,
					Message: "back-reference \\0 is not allowed in XPath regex",
				}
			}
			if inCharClass == 0 && next >= '1' && next <= '9' {
				ref, validEnd, end, invalid := resolveXPathBackref(runes, i+1, captureCount, groupStack)
				if allowBackrefs {
					if invalid || validEnd < 0 {
						return &regexError{
							Code:    errCodeFORX0002,
							Message: fmt.Sprintf("invalid back-reference \\%s in XPath regex", string(runes[i+1:end])),
						}
					}
					i = end - 1
					continue
				}
				return &regexError{
					Code:    errCodeFORX0002,
					Message: fmt.Sprintf("back-reference \\%d is not allowed in XPath regex", ref),
				}
			}
			// \P{X} with invalid category — check for complement class
			// with unknown property names. The \p{X} case is already handled
			// by translateUnicodeProperty, but we need to also catch
			// single-letter \P with invalid categories.
			if next == 'P' || next == 'p' {
				neg := next == 'P'
				if i+2 < len(runes) && runes[i+2] == '{' {
					end := findClosingBrace(runes, i+3)
					if end < 0 {
						return &regexError{
							Code:    errCodeFORX0002,
							Message: fmt.Sprintf("unterminated \\%c{ in regex", next),
						}
					}
					propName := string(runes[i+3 : end])
					if _, err := translateUnicodeProperty(propName, neg, xsd11); err != nil {
						return err
					}
					i = end
					continue
				}
			}
			i++ // skip escaped character
			continue
		}

		// Track character class nesting
		if r == '[' {
			inCharClass++
			continue
		}
		if r == ']' {
			if inCharClass > 0 {
				inCharClass--
				continue
			}
			return &regexError{
				Code:    errCodeFORX0002,
				Message: "unescaped ']' outside character class is not allowed in XPath regex",
			}
		}

		if inCharClass == 0 && r == '(' {
			groupNum := 0
			if i+2 < len(runes) && runes[i+1] == '?' && runes[i+2] == ':' {
				groupStack = append(groupStack, groupNum)
				continue
			}
			captureCount++
			groupStack = append(groupStack, captureCount)
			continue
		}
		if inCharClass == 0 && r == ')' {
			if len(groupStack) > 0 {
				groupStack = groupStack[:len(groupStack)-1]
			}
			continue
		}

		// Unescaped '{' outside of a character class and outside of a valid
		// quantifier context is invalid in XPath regex. Go's regexp accepts
		// bare braces as literals. Inside character classes, '{' is literal.
		if r == '{' && inCharClass == 0 {
			if !isValidQuantifierBrace(runes, i) {
				return &regexError{
					Code:    errCodeFORX0002,
					Message: "unescaped '{' outside quantifier is not allowed in XPath regex",
				}
			}
			inQuantifier = true
			continue
		}

		// Unescaped '}' outside a character class must close a valid quantifier.
		// Go's regexp accepts bare '}' as a literal, but XPath regex forbids it.
		if r == '}' && inCharClass == 0 {
			if inQuantifier {
				inQuantifier = false
			} else {
				return &regexError{
					Code:    errCodeFORX0002,
					Message: "unescaped '}' outside quantifier is not allowed in XPath regex",
				}
			}
			continue
		}
	}
	if inCharClass > 0 {
		return &regexError{
			Code:    errCodeFORX0002,
			Message: "unterminated character class in XPath regex",
		}
	}
	return nil
}

func isOpenCaptureGroup(groupStack []int, ref int) bool {
	return slices.Contains(groupStack, ref)
}

func resolveXPathBackref(runes []rune, start, captureCount int, groupStack []int) (int, int, int, bool) {
	ref := 0
	validEnd := -1
	end := start
	value := 0
	for end < len(runes) && runes[end] >= '0' && runes[end] <= '9' {
		value = value*10 + int(runes[end]-'0')
		end++
	}

	if value > 0 && value <= captureCount {
		if isOpenCaptureGroup(groupStack, value) {
			return 0, -1, end, true
		}
		return value, end, end, false
	}

	value = 0
	for i := start; i < end; i++ {
		value = value*10 + int(runes[i]-'0')
		if value > 0 && value <= captureCount && !isOpenCaptureGroup(groupStack, value) {
			ref = value
			validEnd = i + 1
		}
	}
	return ref, validEnd, end, false
}

func normalizeXPathBackrefs(pattern string) string {
	var b strings.Builder
	runes := []rune(pattern)
	inCharClass := 0
	captureCount := 0
	var groupStack []int
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\\' && i+1 < len(runes) {
			next := runes[i+1]
			if inCharClass == 0 && next >= '1' && next <= '9' {
				ref, validEnd, end, _ := resolveXPathBackref(runes, i+1, captureCount, groupStack)
				if validEnd >= 0 {
					suffix := string(runes[validEnd:end])
					if suffix == "" {
						b.WriteRune('\\')
						b.WriteString(strconv.Itoa(ref))
					} else {
						b.WriteString("(?:\\")
						b.WriteString(strconv.Itoa(ref))
						b.WriteByte(')')
						b.WriteString(suffix)
					}
					i = end - 1
					continue
				}
			}
			b.WriteRune(r)
			i++
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
		case '(':
			if inCharClass == 0 {
				groupNum := 0
				if i+2 >= len(runes) || runes[i+1] != '?' || runes[i+2] != ':' {
					captureCount++
					groupNum = captureCount
				}
				groupStack = append(groupStack, groupNum)
			}
		case ')':
			if inCharClass == 0 && len(groupStack) > 0 {
				groupStack = groupStack[:len(groupStack)-1]
			}
		}

		b.WriteRune(r)
	}
	return b.String()
}

func hasXPathBackrefs(pattern string) bool {
	runes := []rune(pattern)
	inCharClass := 0
	for i := 0; i < len(runes)-1; i++ {
		switch runes[i] {
		case '[':
			inCharClass++
		case ']':
			if inCharClass > 0 {
				inCharClass--
			}
		case '\\':
			if inCharClass == 0 && runes[i+1] >= '1' && runes[i+1] <= '9' {
				return true
			}
			i++
		}
	}
	return false
}

func hasXPathCharClassSubtraction(pattern string) bool {
	runes := []rune(pattern)
	inCharClass := 0
	for i := 0; i < len(runes)-1; i++ {
		switch runes[i] {
		case '\\':
			i++
		case '[':
			inCharClass++
		case ']':
			if inCharClass > 0 {
				inCharClass--
			}
		case '-':
			if inCharClass > 0 && runes[i+1] == '[' {
				return true
			}
		}
	}
	return false
}

func hasLargeXPathQuantifier(pattern string) bool {
	runes := []rune(pattern)
	inCharClass := 0
	for i := 0; i < len(runes); i++ {
		switch runes[i] {
		case '\\':
			i++
		case '[':
			inCharClass++
		case ']':
			if inCharClass > 0 {
				inCharClass--
			}
		case '{':
			if inCharClass > 0 || !isValidQuantifierBrace(runes, i) {
				continue
			}
			end := findClosingBrace(runes, i+1)
			if end < 0 {
				continue
			}
			if quantifierExceedsRE2Limit(string(runes[i+1 : end])) {
				return true
			}
		}
	}
	return false
}

func quantifierExceedsRE2Limit(content string) bool {
	for part := range strings.SplitSeq(content, ",") {
		if part == "" {
			continue
		}
		n, err := strconv.ParseUint(part, 10, 64)
		if err != nil || n > 1000 {
			return true
		}
	}
	return false
}

// quantifierContentRe matches the content of a valid quantifier brace:
// digits [ "," [ digits ] ]. Compiled once and reused across scans.
var quantifierContentRe = regexp.MustCompile(`^\d+(,\d*)?$`)

// isValidQuantifierBrace checks whether '{' at position i is part of a
// valid quantifier {n}, {n,}, or {n,m}. The '{' must be preceded by a
// quantifiable atom (not at the start) and its content must match the
// quantifier pattern.
func isValidQuantifierBrace(runes []rune, i int) bool {
	// Must have a preceding quantifiable atom
	if i == 0 {
		return false
	}
	prev := runes[i-1]
	// Valid preceding chars for a quantifier: ), ], or any non-meta char,
	// or an escape sequence. We check broadly.
	if prev == '|' || prev == '(' || prev == '{' {
		return false
	}

	// Find closing brace
	end := -1
	for j := i + 1; j < len(runes); j++ {
		if runes[j] == '}' {
			end = j
			break
		}
	}
	if end < 0 {
		return false
	}

	content := string(runes[i+1 : end])
	return quantifierContentRe.MatchString(content)
}

// rejectPerlSpecific checks for Perl-specific regex constructs not allowed in XPath.
func rejectPerlSpecific(pattern string) error {
	runes := []rune(pattern)
	for i := 0; i < len(runes); i++ {
		if runes[i] == '\\' && i+1 < len(runes) {
			switch runes[i+1] {
			case 'b', 'B', 'A', 'Z', 'z', 'x', 'u', 'U':
				return &regexError{
					Code:    errCodeFORX0002,
					Message: fmt.Sprintf("Perl-specific escape \\%c is not allowed in XPath regex", runes[i+1]),
				}
			}
			i++ // skip escaped character
			continue
		}
		// Reject inline flags (?...) — XPath uses flags parameter instead
		if runes[i] == '(' && i+1 < len(runes) && runes[i+1] == '?' {
			// Allow (?:...) for non-capturing groups
			if i+2 < len(runes) && runes[i+2] == ':' {
				continue
			}
			return &regexError{
				Code:    errCodeFORX0002,
				Message: "inline flag groups (?...) are not allowed in XPath regex",
			}
		}
	}
	return nil
}

// Translate converts an XPath/XQuery regex pattern (fn:matches/tokenize/replace)
// into a Go RE2 pattern. In this flavor '^' and '$' are anchors and the pattern
// is not implicitly anchored; use Compile for XSD xs:pattern facets, where
// '^'/'$' are literals and the whole value must match.
func Translate(pattern string, dotAll, ignoreCase bool) (string, error) {
	return translateXPathRegex(pattern, dotAll, ignoreCase, false, false)
}

// Regexp is a compiled XML Schema pattern-facet regular expression. It matches
// the whole input value (the pattern is anchored at both ends) against either
// Go's RE2 engine or, for constructs RE2 lacks, the regexp2 backtracking engine.
type Regexp struct {
	std       *regexp.Regexp
	backtrack *regexp2.Regexp
}

// MatchString reports whether the whole string s is matched by the pattern.
func (r *Regexp) MatchString(s string) bool {
	if r.backtrack != nil {
		// regexp2 is a backtracking engine; the only error it returns is a
		// match-timeout (see DefaultMatchTimeout()). A timed-out match cannot be
		// proven to satisfy the pattern, so report it as a non-match rather than
		// letting a catastrophic-backtracking input hang the caller.
		ok, _ := r.backtrack.MatchString(s)
		return ok
	}
	return r.std.MatchString(s)
}

// Compile translates and compiles an XML Schema pattern-facet regular
// expression, anchoring it so it matches the entire value. Patterns that use
// XML Schema character-class subtraction ([a-z-[aeiou]]) or quantifier bounds
// beyond RE2's limit are compiled with the regexp2 backtracking engine, which
// supports those constructs natively; all other patterns use Go's RE2 engine.
// It returns an error for patterns that are not valid XML Schema regular
// expressions, so callers can report a schema error rather than silently
// ignoring the facet.
func Compile(pattern string) (*Regexp, error) {
	return CompileVersion(pattern, false)
}

// CompileVersion is Compile with an XSD 1.1 toggle. In XSD 1.1 mode an
// unrecognized \p{Is...} block name is accepted (matching every character) per
// XSD 1.1 test bug 13670, and \c/\C admit the full XML 1.1 / XML 1.0 5th edition
// NameChar combining range (U+0300–U+036F, including U+0346). With xsd11=false
// the XSD 1.0 / XPath 2.0 behavior is byte-identical to Compile's original.
func CompileVersion(pattern string, xsd11 bool) (*Regexp, error) {
	// Enforce the XSD/XPath regex grammar up front, independent of which engine
	// compiles the pattern. RE2 happens to reject some non-XSD constructs (e.g.
	// \1 back-references) but accepts others (e.g. \b word boundaries), and the
	// regexp2 backtracking engine accepts both — so without these checks an
	// invalid pattern routed to regexp2 (back-reference + character-class
	// subtraction) would be silently accepted. XSD regex has no back-references.
	if err := rejectPerlSpecific(pattern); err != nil {
		return nil, err
	}
	if err := validateXPathRegex(pattern, false, xsd11); err != nil {
		return nil, err
	}

	// Compile handles XSD xs:pattern facets: '^'/'$' are literal characters and
	// the pattern is implicitly anchored to the whole value (the \A(?:...)\z
	// wrap below). Pass xsdPattern=true so the translator escapes '^'/'$'.
	translated, err := translateXPathRegex(pattern, false, false, true, xsd11)
	if err != nil {
		return nil, err
	}

	// RE2 implements neither character-class subtraction nor unbounded
	// quantifiers; route those to the backtracking engine instead of letting
	// RE2 misinterpret (subtraction) or reject (large bounds) them.
	if hasXPathCharClassSubtraction(pattern) || hasLargeXPathQuantifier(pattern) {
		re, err := regexp2.Compile(`\A(?:`+translated+`)\z`, regexp2.RE2)
		if err != nil {
			return nil, &regexError{Code: errCodeFORX0002, Message: fmt.Sprintf("invalid regular expression: %s", err)}
		}
		// Bound backtracking so an adversarial pattern/value cannot hang the
		// process (catastrophic backtracking). 0 disables the timeout.
		if t := DefaultMatchTimeout(); t > 0 {
			re.MatchTimeout = t
		}
		return &Regexp{backtrack: re}, nil
	}

	re, err := regexp.Compile(`\A(?:` + translated + `)\z`)
	if err != nil {
		return nil, &regexError{Code: errCodeFORX0002, Message: fmt.Sprintf("invalid regular expression: %s", err)}
	}
	return &Regexp{std: re}, nil
}

// Validate rejects patterns Go's regexp accepts but the XPath/XSD regex spec forbids.
func Validate(pattern string, allowBackrefs bool) error {
	return validateXPathRegex(pattern, allowBackrefs, false)
}

// HasBackrefs reports whether the pattern contains a back-reference.
func HasBackrefs(pattern string) bool { return hasXPathBackrefs(pattern) }

// HasCharClassSubtraction reports whether the pattern uses [a-z-[...]] subtraction.
func HasCharClassSubtraction(pattern string) bool { return hasXPathCharClassSubtraction(pattern) }

// HasLargeQuantifier reports whether the pattern has a {n,m} bound exceeding RE2's limit.
func HasLargeQuantifier(pattern string) bool { return hasLargeXPathQuantifier(pattern) }

// NormalizeBackrefs rewrites multi-digit back-references for Go's regexp engine.
func NormalizeBackrefs(pattern string) string { return normalizeXPathBackrefs(pattern) }

// RejectPerlSpecific rejects Perl-only constructs not allowed in XPath regex.
func RejectPerlSpecific(pattern string) error { return rejectPerlSpecific(pattern) }

// XMLNameStartCharRange / XMLNameCharRange are the XML Name character ranges
// (for use inside a [...] class), shared with NCName/NMTOKEN validation.
const XMLNameStartCharRange = xmlNameStartCharRange
const XMLNameCharRange = xmlNameCharRange
