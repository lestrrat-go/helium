package xsd_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// patternSchema wraps a single xs:pattern value in a minimal simpleType schema.
// The pattern is embedded verbatim (none of the test patterns contain XML-special
// characters), so a Go %q escape does not corrupt backslash sequences.
func patternSchema(pattern string) string {
	return fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="t"/>
  <xs:simpleType name="t">
    <xs:restriction base="xs:string">
      <xs:pattern value="%s"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`, pattern)
}

// TestPatternRegexGrammarRejectsNonXSDConstructs verifies that xs:pattern values
// using regex constructs valid in the XPath/.NET flavors but NOT in the XML
// Schema regex grammar (XML Schema Part 2 Appendix F) are rejected at schema
// compile time in the default (XSD 1.0) mode, rather than silently accepted.
//
// Three classes are covered: reluctant (non-greedy) quantifiers, '(?...)' group
// extensions (including non-capturing '(?:...)'), and unbalanced parentheses.
func TestPatternRegexGrammarRejectsNonXSDConstructs(t *testing.T) {
	t.Parallel()

	invalid := []struct {
		name    string
		pattern string
	}{
		// reluctant quantifiers
		{"reluctant-star", `a.*?c`},
		{"reluctant-plus", `([0-9]+?)([a-z]+?)`},
		{"reluctant-question", `ab??bc`},
		{"reluctant-brace-range", `ab{1,3}?bc`},
		{"reluctant-brace-open", `ab{0,}?bc`},
		{"reluctant-group", `(a+|b){0,1}?`},
		// group extensions '(?...)'
		{"noncapturing-group", `a(?:b|c|d)(.)`},
		{"noncapturing-star", `(?:..)*a`},
		{"noncapturing-nested", `(?:(?:(?:(a))))`},
		{"noncapturing-mixed", `(a+)(?:b*)(ccc)`},
		{"noncapturing-anchored", `^(?:a?b?)*$`},
		// combined non-capturing + reluctant
		{"noncapturing-reluctant", `(.)(?:b|c|d){4,5}?a`},
		// unbalanced parentheses
		{"swapped-parens", `)(`},
		{"unclosed-paren", `(abc`},
		{"extra-close-paren", `abc)`},
	}

	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			errs := compileSchemaErrors(t, patternSchema(tc.pattern))
			require.NotEmpty(t, errs,
				"pattern %q is not a valid XSD regex and must be rejected at compile time", tc.pattern)
			require.Contains(t, errs, "not a valid regular expression",
				"rejection should be reported as an invalid-regex schema error")
		})
	}
}

// TestPatternRegexGrammarRejectsRangeAfterRange verifies that a character-class
// range operator '-' whose left endpoint was already consumed as the END of a
// preceding range is rejected at compile time in the default (XSD 1.0) mode.
// The XSD 1.0 charRange grammar (Part 2, Appendix F, productions 15/16) treats a
// mid-group '-' as a range operator needing a fresh single-character left
// endpoint; W3C ms Regex_w3c reF20-23/reG26-33/reH19-21 expect these invalid in
// XSD 1.0 (and valid in XSD 1.1).
func TestPatternRegexGrammarRejectsRangeAfterRange(t *testing.T) {
	t.Parallel()

	for _, pattern := range []string{`[^a-d-b-c]`, `[a-c-1-4x-z-7-9]`, `[a-a-x-x]`} {
		t.Run(pattern, func(t *testing.T) {
			errs := compileSchemaErrors(t, patternSchema(pattern))
			require.NotEmpty(t, errs,
				"pattern %q is not a valid XSD 1.0 regex and must be rejected at compile time", pattern)
			require.Contains(t, errs, "not a valid regular expression",
				"rejection should be reported as an invalid-regex schema error")
		})
	}
}

// TestPatternRegexGrammarAcceptsValidPatterns guards against over-rejection: a
// set of patterns that ARE valid XML Schema regular expressions must still
// compile without error in the default (XSD 1.0) mode.
func TestPatternRegexGrammarAcceptsValidPatterns(t *testing.T) {
	t.Parallel()

	valid := []struct {
		name    string
		pattern string
	}{
		{"char-class-plus", `[0-9]+`},
		{"capturing-star", `(abc)*`},
		{"alternation-plus", `(a|b)+`},
		{"optionals", `a?b?c?`},
		{"brace-range", `a{2,3}`},
		{"escaped-quantifier-optional", `a\?\??`},
		{"nested-groups", `(foo|bar)(baz)?`},
		{"dot-star", `a.*c`},
		{"unicode-property-optional", `\p{L}?`},
		{"name-char-escapes", `\i\c*`},
		{"repeated-group", `x(~~)*`},
		{"literal-question-then-optional", `a\??`},
		{"grouped-optionals", `(a)?(b)?`},
		{"digits-with-brace", `\d{3}-\d{4}`},
	}

	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			errs := compileSchemaErrors(t, patternSchema(tc.pattern))
			require.Empty(t, errs,
				"pattern %q is a valid XSD regex and must compile without error", tc.pattern)
		})
	}
}
