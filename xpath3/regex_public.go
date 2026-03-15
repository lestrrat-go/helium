package xpath3

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
