package xsdregex_test

import (
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/lestrrat-go/helium/internal/xsdregex"
	"github.com/stretchr/testify/require"
)

func TestCaretDollarAreLiterals(t *testing.T) {
	// In the XSD xs:pattern grammar (Compile) '^' and '$' are literal
	// characters, not anchors, and the pattern is implicitly anchored to the
	// whole value. So "^a$" must match only the literal string "^a$".
	re, err := xsdregex.Compile("^a$")
	require.NoError(t, err, "compiling ^a$")

	require.True(t, re.MatchString("^a$"), `pattern "^a$" should match literal "^a$"`)
	require.False(t, re.MatchString("a"), `pattern "^a$" should not match "a"`)
	require.False(t, re.MatchString("^a"), `pattern "^a$" should not match "^a"`)
	require.False(t, re.MatchString("a$"), `pattern "^a$" should not match "a$"`)
	require.False(t, re.MatchString(""), `pattern "^a$" should not match empty`)
}

// TestCaretDollarModeDistinction pins the two opposite regex flavors that share
// this translator. Compile is the XSD xs:pattern facet, where '^'/'$' are
// literal characters; Translate is the XPath/XQuery flavor (fn:matches/
// tokenize/replace), where '^'/'$' must stay RE2 anchors.
func TestCaretDollarModeDistinction(t *testing.T) {
	t.Run("xsd pattern escapes anchors as literals", func(t *testing.T) {
		out, err := xsdregex.Translate("^a$", false, false)
		require.NoError(t, err)
		// In the XPath flavor '^'/'$' are anchors, so Translate must NOT escape
		// them; the output is left as-is for RE2 to treat as zero-width anchors.
		require.Equal(t, "^a$", out, "XPath Translate keeps ^/$ as anchors")
	})

	t.Run("xsd compile treats anchors as literals", func(t *testing.T) {
		// Compile (xs:pattern) must treat the same pattern as the literal
		// three-character string, anchored to the whole value.
		re, err := xsdregex.Compile("^a$")
		require.NoError(t, err)
		require.True(t, re.MatchString("^a$"), "xs:pattern ^a$ matches literal ^a$")
		require.False(t, re.MatchString("a"), "xs:pattern ^a$ does not match bare a")
	})
}

// TestNameCharU0346FlavorDistinction pins the second flavor divergence in this
// shared translator: the \c (XML NameChar) escape. Every XSD xs:pattern facet —
// 1.0 or 1.1 (Compile / CompileVersion) — follows the XML 1.0 5th-edition NameChar
// definition, which INCLUDES the combining mark U+0346 (W3C regex test reZ006i /
// bug 13606). The older XPath-2.0 flavor (Translate, used by fn:matches and the
// regex-syntax-xslt20 QT3 suite) keeps the pre-5th-edition carve-out that EXCLUDES
// U+0346. So the same pattern must classify U+0346 differently across the two.
func TestNameCharU0346FlavorDistinction(t *testing.T) {
	const u0346 = "͆" // COMBINING KAVYKA ABOVE RIGHT (Mn), a NameChar since XML 1.0 5th ed.

	t.Run("xsd pattern facet includes U+0346", func(t *testing.T) {
		re10, err := xsdregex.Compile(`[\c]`)
		require.NoError(t, err)
		require.True(t, re10.MatchString(u0346), "XSD 1.0 xs:pattern [\\c] matches U+0346")

		re11, err := xsdregex.CompileVersion(`[\c]`, true)
		require.NoError(t, err)
		require.True(t, re11.MatchString(u0346), "XSD 1.1 xs:pattern [\\c] matches U+0346")
	})

	t.Run("xpath flavor keeps the carve-out", func(t *testing.T) {
		out, err := xsdregex.Translate(`[\c]`, false, false)
		require.NoError(t, err)
		anchored := regexp.MustCompile("^(?:" + out + ")$")
		require.False(t, anchored.MatchString(u0346), "XPath Translate [\\c] excludes U+0346")
	})
}

func TestLiteralAnchorChars(t *testing.T) {
	t.Run("bare caret", func(t *testing.T) {
		re, err := xsdregex.Compile("^")
		require.NoError(t, err)
		require.True(t, re.MatchString("^"))
		require.False(t, re.MatchString(""))
	})
	t.Run("bare dollar", func(t *testing.T) {
		re, err := xsdregex.Compile("$")
		require.NoError(t, err)
		require.True(t, re.MatchString("$"))
		require.False(t, re.MatchString(""))
	})
	t.Run("dollar amount", func(t *testing.T) {
		// In XSD regex a bare '$' is a literal character, not an end anchor.
		// This proves '$' is matched literally and is NOT treated as an anchor.
		re, err := xsdregex.Compile(`$[0-9]+`)
		require.NoError(t, err)
		require.True(t, re.MatchString("$123"), "bare $ is a literal, so $123 matches")
		require.False(t, re.MatchString("123"), "$ is not an anchor, so 123 alone does not match")
	})
}

func TestXSDMetacharsStillWork(t *testing.T) {
	// A pattern using genuine XSD regex metacharacters must keep working after
	// the ^/$ literal fix.
	re, err := xsdregex.Compile(`(ab)+|c*\d{2,3}`)
	require.NoError(t, err, "compiling pattern with XSD metachars")

	require.True(t, re.MatchString("ababab"), "(ab)+ branch")
	require.True(t, re.MatchString("12"), `\d{2,3} branch (2 digits)`)
	require.True(t, re.MatchString("ccc123"), `c*\d{2,3} branch`)
	require.False(t, re.MatchString("1"), "single digit should fail {2,3}")
	require.False(t, re.MatchString("abx"), "trailing junk should fail (implicit anchoring)")
}

func TestNegatedCharClassUnaffected(t *testing.T) {
	// '^' as the first char of a character class is negation, not a literal,
	// and must keep that meaning.
	re, err := xsdregex.Compile(`[^a]`)
	require.NoError(t, err)
	require.True(t, re.MatchString("b"))
	require.False(t, re.MatchString("a"))
}

func TestUnknownBlockNegatedInCharClass(t *testing.T) {
	// XSD 1.1 (test bug 13670): an unrecognized \p{Is<block>} matches every
	// character and its negation \P{Is<block>} matches none. A character class
	// whose only members are such negated-unknown-block contributions therefore
	// denotes the empty set, which must compile to a never-matching construct
	// rather than an empty [] that RE2 cannot parse.
	t.Run("xsd11 negated-unknown-block sole member matches nothing", func(t *testing.T) {
		re, err := xsdregex.CompileVersion(`[\P{IsFoo}]`, true)
		require.NoError(t, err, `[\P{IsFoo}] must compile in XSD 1.1`)
		require.False(t, re.MatchString(""), "matches no empty value")
		require.False(t, re.MatchString("a"), "matches no character")
		require.False(t, re.MatchString("x"), "matches no character")
	})

	t.Run("xsd11 complement of negated-unknown-block matches any single char", func(t *testing.T) {
		re, err := xsdregex.CompileVersion(`[^\P{IsFoo}]`, true)
		require.NoError(t, err, `[^\P{IsFoo}] must compile in XSD 1.1`)
		require.True(t, re.MatchString("a"), "matches any single character")
		require.True(t, re.MatchString("x"), "matches any single character")
		require.False(t, re.MatchString(""), "a class still requires one character")
	})

	t.Run("xsd11 negated-unknown-block with another member keeps that member", func(t *testing.T) {
		re, err := xsdregex.CompileVersion(`[a\P{IsFoo}]`, true)
		require.NoError(t, err)
		require.True(t, re.MatchString("a"), "the concrete member still matches")
		require.False(t, re.MatchString("b"), "no other character matches")
	})

	t.Run("xsd11 standalone negated-unknown-block matches nothing", func(t *testing.T) {
		re, err := xsdregex.CompileVersion(`\P{IsFoo}`, true)
		require.NoError(t, err)
		require.False(t, re.MatchString("a"))
		require.False(t, re.MatchString("x"))
	})

	t.Run("xsd10 unknown block still rejected", func(t *testing.T) {
		// The default (XSD 1.0 / xpath3 / relaxng) path passes xsd11=false and
		// must keep the FORX0002 rejection, byte-identical.
		_, err := xsdregex.CompileVersion(`[\P{IsFoo}]`, false)
		require.Error(t, err, "xsd11=false must reject an unknown block")
		require.Contains(t, err.Error(), "unknown Unicode block: IsFoo")

		_, err = xsdregex.Compile(`[\P{IsFoo}]`)
		require.Error(t, err, "Compile (XSD 1.0) must reject an unknown block")
		require.Contains(t, err.Error(), "unknown Unicode block: IsFoo")
	})
}

func TestCharClassHyphenRangeEndpoint(t *testing.T) {
	// In the XSD/XPath regex grammar a '-' is a literal only at the start of a
	// character class or immediately before the closing ']'. An interior '-'
	// is a range operator, and its endpoints must be single characters — never
	// another '-'. Go's RE2 happily accepts '[--z]' (range '-'..'z') and
	// '[!--]' (range '!'..'-'), so the translator must reject these explicitly.
	t.Run("reject hyphen as range endpoint", func(t *testing.T) {
		for _, pat := range []string{`[--z]`, `[!--]`, `[^--z]`, `[a--z]`} {
			t.Run(pat, func(t *testing.T) {
				_, err := xsdregex.Compile(pat + "*")
				require.Error(t, err, "pattern %q must be rejected", pat)
				require.Contains(t, err.Error(), "invalid character class")
			})
		}
	})

	t.Run("accept valid hyphen neighbors", func(t *testing.T) {
		// '[+-]' is '+' plus a trailing literal '-'; '[-+]' is a leading literal
		// '-' plus '+'. Both keep the '-' in a literal position.
		for _, pat := range []string{`[+-]`, `[-+]`} {
			t.Run(pat, func(t *testing.T) {
				_, err := xsdregex.Compile(pat + "*")
				require.NoError(t, err, "pattern %q must be accepted", pat)
			})
		}
	})

	// A literal '-' immediately abutting a '-[' character-class subtraction
	// operator is part of the base positive group, not a range endpoint, so the
	// range-endpoint rule must NOT fire on it. '[--[a]]' (base {-} minus {a})
	// and '[^--[a]]' (negated base {-} minus {a}) are valid and compile.
	t.Run("accept hyphen abutting subtraction operator", func(t *testing.T) {
		for _, pat := range []string{`[--[a]]`, `[^--[a]]`} {
			t.Run(pat, func(t *testing.T) {
				_, err := xsdregex.Compile(pat + "*")
				require.NoError(t, err, "pattern %q must be accepted", pat)
			})
		}
	})

	// '[a--[b]]' (base {a,-} minus {b}) is also a valid XSD subtraction, but the
	// regexp2/RE2 subtraction path passes the class through unexpanded, so the
	// engine misreads 'a--' as a reverse-order range and rejects it — a
	// pre-existing limitation, unchanged by this fix. The structural
	// char-class validator must still NOT be the thing that rejects it: any
	// error must come from the engine, never an "invalid character class"
	// FORX0002, proving the range-endpoint rule does not fire on the literal
	// dash abutting the '-[' operator.
	t.Run("range-endpoint rule does not fire on subtraction base dash", func(t *testing.T) {
		_, err := xsdregex.Compile(`[a--[b]]*`)
		if err != nil {
			require.NotContains(t, err.Error(), "invalid character class",
				"structural validator must not reject the subtraction base dash")
		}
	})
}

func TestXSD10CharClassRangeAfterRange(t *testing.T) {
	// The XSD 1.0 charRange grammar (Part 2, Appendix F, productions 15/16)
	// treats a mid-group '-' as a range operator; when its left endpoint was
	// already consumed as the END of a preceding range there is no valid left
	// operand, so the pattern is invalid. W3C ms Regex_w3c reF20-23/reG26-33/
	// reH19-21 expect these invalid in XSD 1.0 and VALID in XSD 1.1 (whose
	// rewritten grammar admits a mid-group '-' as a literal singleChar).
	t.Run("xsd10 rejects range operator after a completed range", func(t *testing.T) {
		for _, pat := range []string{`[^a-d-b-c]`, `[a-c-1-4x-z-7-9]`, `[a-a-x-x]`, `[a-z-+]`, `[a-z-9]`} {
			t.Run(pat, func(t *testing.T) {
				_, err := xsdregex.Compile(pat + "*")
				require.Error(t, err, "pattern %q must be rejected in XSD 1.0", pat)
				require.Contains(t, err.Error(), "invalid character range")
			})
		}
	})

	t.Run("xsd11 accepts them", func(t *testing.T) {
		for _, pat := range []string{`[^a-d-b-c]`, `[a-c-1-4x-z-7-9]`, `[a-a-x-x]`, `[a-z-+]`} {
			t.Run(pat, func(t *testing.T) {
				_, err := xsdregex.CompileVersion(pat+"*", true)
				require.NoError(t, err, "pattern %q must be accepted in XSD 1.1", pat)
			})
		}
	})

	t.Run("xsd10 still accepts ordinary multi-range classes", func(t *testing.T) {
		for _, pat := range []string{`[a-z0-9]`, `[a-z-]`, `[-a-z]`, `[a-c-[b]]`, `[a-z--[b-z]]`, `[abc-]`, `[+-/]`, `[!-~-]`, `[0-9a-fA-F]`} {
			t.Run(pat, func(t *testing.T) {
				_, err := xsdregex.Compile(pat + "*")
				require.NoError(t, err, "pattern %q must still compile in XSD 1.0", pat)
			})
		}
	})
}

func TestPrivateUseBlockAllRanges(t *testing.T) {
	// \p{IsPrivateUse} is the union of the BMP Private Use Area and both
	// supplementary private-use planes. The individual block names still name
	// their separate ranges.
	const privateUsePattern = `\p{IsPrivateUse}`

	bmpLo, bmpHi := string(rune(0xE000)), string(rune(0xF8FF))
	suppA, suppB := string(rune(0xF0000)), string(rune(0x10FFFD))
	tagsHi := string(rune(0xE007F))
	cjkCompatLo := string(rune(0xF900))

	for _, tc := range []struct {
		name    string
		pattern string
		input   string
		want    bool
	}{
		{"bmp-low-matches", privateUsePattern, bmpLo, true},
		{"bmp-high-matches", privateUsePattern, bmpHi, true},
		{"supp-a-matches", privateUsePattern, suppA, true},
		{"supp-b-matches", privateUsePattern, suppB, true},
		{"tags-boundary-not-matched", privateUsePattern, tagsHi, false},
		{"cjk-compat-boundary-not-matched", privateUsePattern, cjkCompatLo, false},
		{"class-bmp-matches", `[\p{IsPrivateUse}]`, bmpLo, true},
		{"class-supp-a-matches", `[\p{IsPrivateUse}]`, suppA, true},
		{"class-supp-b-matches", `[\p{IsPrivateUse}]`, suppB, true},
		{"class-tags-boundary-not-matched", `[\p{IsPrivateUse}]`, tagsHi, false},
		{"supp-a-block-still-matches", `\p{IsSupplementaryPrivateUseArea-A}`, suppA, true},
		{"supp-b-block-still-matches", `\p{IsSupplementaryPrivateUseArea-B}`, suppB, true},
		{"neg-bmp-not-matched", `\P{IsPrivateUse}`, bmpLo, false},
		{"neg-supp-a-not-matched", `\P{IsPrivateUse}`, suppA, false},
		{"neg-supp-b-not-matched", `\P{IsPrivateUse}`, suppB, false},
		{"neg-cjk-compat-boundary-matched", `\P{IsPrivateUse}`, cjkCompatLo, true},
		{"neg-class-bmp-not-matched", `[^\p{IsPrivateUse}]`, bmpHi, false},
		{"neg-class-supp-a-not-matched", `[^\p{IsPrivateUse}]`, suppA, false},
		{"neg-class-supp-b-not-matched", `[^\p{IsPrivateUse}]`, suppB, false},
		{"neg-class-tags-boundary-matched", `[^\p{IsPrivateUse}]`, tagsHi, true},
		{"prop-neg-class-bmp-not-matched", `[\P{IsPrivateUse}]`, bmpLo, false},
		{"prop-neg-class-supp-a-not-matched", `[\P{IsPrivateUse}]`, suppA, false},
		{"prop-neg-class-supp-b-not-matched", `[\P{IsPrivateUse}]`, suppB, false},
		{"prop-neg-class-cjk-compat-boundary-matched", `[\P{IsPrivateUse}]`, cjkCompatLo, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			re, err := xsdregex.Compile(tc.pattern)
			require.NoError(t, err, "pattern %q must compile", tc.pattern)
			require.Equal(t, tc.want, re.MatchString(tc.input),
				"pattern %q on U+%04X", tc.pattern, []rune(tc.input)[0])
		})
	}
}

func TestDefaultMatchTimeoutAccessors(t *testing.T) {
	orig := xsdregex.DefaultMatchTimeout()
	t.Cleanup(func() { xsdregex.SetDefaultMatchTimeout(orig) })

	require.Equal(t, 5*time.Second, orig, "default timeout should be 5s")

	xsdregex.SetDefaultMatchTimeout(2 * time.Second)
	require.Equal(t, 2*time.Second, xsdregex.DefaultMatchTimeout())

	xsdregex.SetDefaultMatchTimeout(0)
	require.Equal(t, time.Duration(0), xsdregex.DefaultMatchTimeout(), "0 should disable")
}

func TestDefaultMatchTimeoutNoRace(t *testing.T) {
	// Concurrent SetDefaultMatchTimeout and Compile must not race (run with
	// -race). Compile reads the default timeout when routing a pattern to the
	// regexp2 backtracking engine (character-class subtraction here).
	orig := xsdregex.DefaultMatchTimeout()
	t.Cleanup(func() { xsdregex.SetDefaultMatchTimeout(orig) })

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := range 100 {
				xsdregex.SetDefaultMatchTimeout(time.Duration(j) * time.Millisecond)
			}
		}()
		go func() {
			defer wg.Done()
			for range 100 {
				_, err := xsdregex.Compile(`[a-z-[aeiou]]+`)
				require.NoError(t, err)
			}
		}()
	}
	wg.Wait()
}

// TestCompileRejectsNonXSDConstructs verifies that the XSD Compile path rejects
// regex constructs (reluctant quantifiers, '(?...)' group extensions, unbalanced
// parentheses) that are valid in the XPath flavor but not in the XML Schema
// regex grammar (XML Schema Part 2 Appendix F).
func TestCompileRejectsNonXSDConstructs(t *testing.T) {
	invalid := []string{
		`a.*?c`, `([0-9]+?)([a-z]+?)`, `ab??bc`, `ab{1,3}?bc`, `(a+|b){0,1}?`,
		`a(?:b|c|d)(.)`, `(?:..)*a`, `(a+)(?:b*)(ccc)`, `^(?:a?b?)*$`,
		`(.)(?:b|c|d){4,5}?a`, `)(`, `(abc`, `abc)`,
	}
	for _, p := range invalid {
		_, err := xsdregex.Compile(p)
		require.Errorf(t, err, "XSD Compile must reject non-XSD construct %q", p)
	}
}

// TestCompileAcceptsValidXSDPatterns guards against over-rejection: patterns that
// ARE valid XML Schema regular expressions must still compile.
func TestCompileAcceptsValidXSDPatterns(t *testing.T) {
	valid := []string{
		`[0-9]+`, `(abc)*`, `(a|b)+`, `a?b?c?`, `a{2,3}`, `a\?\??`, `\p{L}?`,
		`(foo|bar)(baz)?`, `a.*c`, `\i\c*`, `x(~~)*`, `(a)?(b)?`, `\d{3}-\d{4}`,
	}
	for _, p := range valid {
		_, err := xsdregex.Compile(p)
		require.NoErrorf(t, err, "XSD Compile must accept valid XSD pattern %q", p)
	}
}

// TestXPathFlavorKeepsReluctantAndNonCapturing verifies that the shared XPath
// flavor (Translate/Validate, used by xpath3) still permits reluctant quantifiers
// and non-capturing groups — the new XSD-only strictness must not leak into it.
func TestXPathFlavorKeepsReluctantAndNonCapturing(t *testing.T) {
	xpathValid := []string{`a.*?c`, `a{2,3}?`, `(?:ab)+c`, `(?:a|b)*`}
	for _, p := range xpathValid {
		require.NoErrorf(t, xsdregex.Validate(p, false), "Validate must accept XPath-valid %q", p)
		_, err := xsdregex.Translate(p, false, false)
		require.NoErrorf(t, err, "Translate must accept XPath-valid %q", p)
	}
}
