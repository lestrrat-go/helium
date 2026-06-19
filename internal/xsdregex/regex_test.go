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
		re, err := xsdregex.Compile(`\$[0-9]+`)
		require.NoError(t, err)
		require.True(t, re.MatchString("$100"))
		require.False(t, re.MatchString("100"))
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
