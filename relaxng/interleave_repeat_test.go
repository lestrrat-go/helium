package relaxng_test

import (
	"strings"
	"testing"
	"time"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

// TestInterleaveRepeatableMemberGroup covers an <interleave> whose repeatable
// member (zeroOrMore/oneOrMore) wraps a multi-element group. Another interleave
// branch may consume elements between the group members, across iterations.
// Before the fix the round-robin matcher could not split a repeatable group's
// expansion around an interleaved sibling and wrongly rejected valid content.
func TestInterleaveRepeatableMemberGroup(t *testing.T) {
	t.Parallel()

	content := func(members string) string {
		return `<grammar xmlns="http://relaxng.org/ns/structure/1.0"><start>` +
			`<element name="root"><interleave>` + members + `</interleave></element></start></grammar>`
	}
	a := `<element name="a"><empty/></element>`
	b := `<element name="b"><empty/></element>`
	c := `<element name="c"><empty/></element>`
	z := func(p string) string { return `<zeroOrMore>` + p + `</zeroOrMore>` }
	o := func(p string) string { return `<oneOrMore>` + p + `</oneOrMore>` }
	grp := func(p ...string) string { return `<group>` + strings.Join(p, "") + `</group>` }

	cases := []struct {
		name   string
		schema string
		doc    string
		valid  bool
	}{
		// Repeatable group(a,b) with c interleaved between/around members.
		{"zz group(a,b) + zz c", content(z(grp(a, b)) + z(c)), `<root><a/><c/><b/><a/><c/><b/></root>`, true},
		// c may not appear at all; pure paired groups.
		{"zz group(a,b) no c", content(z(grp(a, b)) + z(c)), `<root><a/><b/><a/><b/></root>`, true},
		// oneOrMore group requires at least one complete pair.
		{"oo group(a,b) + zz c", content(o(grp(a, b)) + z(c)), `<root><a/><c/><b/><a/><b/></root>`, true},
		// Empty content: zeroOrMore group + zeroOrMore c accepts nothing.
		{"zz group(a,b) empty ok", content(z(grp(a, b)) + z(c)), `<root></root>`, true},
		// oneOrMore group requires one pair: empty must be rejected.
		{"oo group(a,b) empty rejects", content(o(grp(a, b)) + z(c)), `<root><c/></root>`, false},
		// Dangling unpaired trailing 'a' (missing its 'b') must be rejected.
		{"zz group(a,b) dangling a rejects", content(z(grp(a, b)) + z(c)), `<root><a/><b/><a/></root>`, false},
		// Dangling partial group with c interleaved must still be rejected.
		{"zz group(a,b) dangling a + c rejects", content(z(grp(a, b)) + z(c)), `<root><a/><c/><b/><c/><a/></root>`, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			grammar := compileGrammar(t, tc.schema)
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.doc))
			require.NoError(t, err)
			verr := relaxng.NewValidator(grammar).Validate(t.Context(), doc)
			if tc.valid {
				require.NoError(t, verr, "%s should validate", tc.name)
				return
			}
			require.Error(t, verr, "%s should be rejected", tc.name)
		})
	}
}

// TestInterleaveRepeatableMemberGroupNotExponential guards against algorithmic
// blowup in the interleave matcher: a long document of interleaved group(a,b)
// pairs and c elements must validate with polynomial growth, never exponential.
//
// The guard is the SCALING RATIO between two input sizes, not an absolute
// wall-clock bound. Doubling the input multiplies a correct (quadratic-ish)
// matcher's time by a small constant (~5x here); an exponential regression would
// multiply it by ~2^N. Because the ratio is measured on a single machine it is
// independent of how fast that machine is, so this cannot flake on a slow or
// loaded CI runner the way an absolute threshold does. (A genuinely exponential
// regression would also fail to finish within `go test -timeout`, so it is
// caught regardless of this assertion.)
func TestInterleaveRepeatableMemberGroupNotExponential(t *testing.T) {
	t.Parallel()

	schema := `<grammar xmlns="http://relaxng.org/ns/structure/1.0"><start>` +
		`<element name="root"><interleave>` +
		`<zeroOrMore><group>` +
		`<element name="a"><empty/></element><element name="b"><empty/></element>` +
		`</group></zeroOrMore>` +
		`<zeroOrMore><element name="c"><empty/></element></zeroOrMore>` +
		`</interleave></element></start></grammar>`

	grammar := compileGrammar(t, schema)

	// validateN builds a <root> of n interleaved a/c/b triples, validates it
	// (which must succeed), and returns how long validation took.
	validateN := func(n int) time.Duration {
		var docStr strings.Builder
		docStr.WriteString(`<root>`)
		for range n {
			docStr.WriteString(`<a/><c/><b/>`)
		}
		docStr.WriteString(`</root>`)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(docStr.String()))
		require.NoError(t, err)

		start := time.Now()
		verr := relaxng.NewValidator(grammar).Validate(t.Context(), doc)
		elapsed := time.Since(start)
		require.NoError(t, verr, "interleave of %d group(a,b) pairs with c must validate", n)
		return elapsed
	}

	// fastest validates size n a few times and keeps the shortest run, so a
	// one-off GC pause or scheduler hiccup cannot inflate the measurement.
	fastest := func(n, trials int) time.Duration {
		best := validateN(n)
		for range trials - 1 {
			if d := validateN(n); d < best {
				best = d
			}
		}
		return best
	}

	const (
		baseN   = 1000
		trials  = 3
		maxGrow = 20.0 // observed ~5x for a 2x input; exponential would be astronomically higher
	)

	validateN(baseN) // warm up the allocator/caches before the first timed run
	base := fastest(baseN, trials)
	grown := fastest(2*baseN, trials)
	growth := float64(grown) / float64(base)
	require.Less(t, growth, maxGrow,
		"validation time grew %.1fx for a 2x larger input (base %s at N=%d, grown %s at N=%d); "+
			"expected polynomial growth, suspect exponential blowup",
		growth, base, baseN, grown, 2*baseN)
}
