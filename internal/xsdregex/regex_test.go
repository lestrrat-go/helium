package xsdregex_test

import (
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
		for _, pat := range []string{`[+-]`, `[-+]`, `[a-z-+]`} {
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
