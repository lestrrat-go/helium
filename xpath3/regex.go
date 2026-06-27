package xpath3

import "github.com/lestrrat-go/helium/internal/xsdregex"

// The XPath/XML Schema regex translator lives in internal/xsdregex so that the
// xsd package (which must not import xpath3) can share a single implementation.
// The thin wrappers below keep xpath3's existing local API and map translation
// errors back to XPathError with code FORX0002.

func xsdRegexError(err error) error {
	if err == nil {
		return nil
	}
	return &XPathError{Code: errCodeFORX0002, Message: err.Error()}
}

func translateXPathRegex(pattern string, dotAll, ignoreCase bool) (string, error) {
	out, err := xsdregex.Translate(pattern, dotAll, ignoreCase)
	if err != nil {
		return "", xsdRegexError(err)
	}
	return out, nil
}

func validateXPathRegex(pattern string, allowBackrefs bool) error {
	return xsdRegexError(xsdregex.Validate(pattern, allowBackrefs))
}

func rejectPerlSpecific(pattern string) error {
	return xsdRegexError(xsdregex.RejectPerlSpecific(pattern))
}

func hasXPathBackrefs(pattern string) bool { return xsdregex.HasBackrefs(pattern) }

func hasXPathCharClassSubtraction(pattern string) bool {
	return xsdregex.HasCharClassSubtraction(pattern)
}

func hasLargeXPathQuantifier(pattern string) bool { return xsdregex.HasLargeQuantifier(pattern) }

func normalizeXPathBackrefs(pattern string) string { return xsdregex.NormalizeBackrefs(pattern) }

const (
	xmlNameStartCharRange = xsdregex.XMLNameStartCharRange
	xmlNameCharRange      = xsdregex.XMLNameCharRange
)

// Regex is a compiled XPath regular expression that can be used by external
// packages (e.g., xslt3) for regex matching with XPath/XML Schema semantics.
type Regex struct {
	inner *compiledXPathRegex
}

// CompileRegex compiles an XPath regular expression with the given flags.
// Flags follow the XPath F&O specification: 'i' (case-insensitive),
// 'm' (multi-line), 's' (dot-all), 'x' (free-spacing), 'q' (literal).
func CompileRegex(pattern, flags string) (*Regex, error) {
	re, err := compileXPathRegex(pattern, flags)
	if err != nil {
		return nil, err
	}
	return &Regex{inner: re}, nil
}

// MatchString reports whether the string s contains any match of the regex.
func (r *Regex) MatchString(s string) (bool, error) {
	return r.inner.MatchString(s)
}

// FindAllSubmatchIndex returns a slice of all successive matches of the
// regex in s, where each match is represented as a slice of index pairs
// (start, end) for the full match and each capture group.
// A return value of nil indicates no match.
// n limits the number of matches; -1 means no limit.
func (r *Regex) FindAllSubmatchIndex(s string, n int) ([][]int, error) {
	return r.inner.FindAllStringSubmatchIndex(s, n)
}

// EachSubmatchIndex streams the successive matches of the regex in s, calling
// fn once per match with the (start, end) byte-index pairs for the full match
// and each capture group (the same layout as a single FindAllSubmatchIndex
// entry; an unmatched group is reported as -1, -1). The slice handed to fn is
// only valid for the duration of the call — copy it to retain it. Iteration
// stops early, and EachSubmatchIndex returns nil, as soon as fn returns false.
//
// Unlike FindAllSubmatchIndex, matches are produced one at a time and (for the
// streaming engines) never accumulated, so live memory stays bounded regardless
// of how many matches s contains. This lets callers enforce a match-count budget
// (or honor a cancelled context) DURING enumeration — an empty- or near-empty-
// matching regex over a large input is rejected without first materializing a
// match slice proportional to the input size.
//
// limit caps the maximum number of matches ever produced; pass a non-positive
// value for no cap. A leading-context pattern (a multi-line ^, \A, \b, ...)
// cannot be streamed incrementally on the RE2 engine and is matched against the
// whole string in one bounded pass, so a caller enforcing a budget of N should
// pass limit = N+1 to keep that pass's allocation proportional to the budget
// rather than to the input's match count.
func (r *Regex) EachSubmatchIndex(s string, limit int, fn func(m []int) bool) error {
	return r.inner.eachStringSubmatchIndex(s, limit, fn)
}
