package relaxng_test

import (
	"strings"
	"testing"

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
	a := benchElementA
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
// The guard is the growth in validation STEPS between two input sizes. The
// validator polls its context once per pattern step, so the step count is the
// work it did, identical on every run and machine. Doubling the input
// multiplies a correct matcher's steps by a small constant; an exponential
// regression multiplies them by ~2^N, so the larger input gets a step budget of
// maxGrow times what the smaller one took.
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

	const (
		baseN = 1000
		// maxBaseSteps is far above the 13 steps per triple the matcher takes.
		maxBaseSteps = 1 << 20
		// maxGrow is the step growth allowed for a 2x input. The matcher's steps
		// grow 2x; polynomial growth up to cubic stays below 8x, and an
		// exponential regression is cut off by the budget it implies.
		maxGrow = 8
	)

	base := validationSteps(t, grammar, interleavedTriplesDoc(t, baseN), maxBaseSteps)
	validationSteps(t, grammar, interleavedTriplesDoc(t, 2*baseN), maxGrow*base)
}

// interleavedTriplesDoc parses a <root> of n interleaved a/c/b triples.
func interleavedTriplesDoc(t *testing.T, n int) *helium.Document {
	t.Helper()
	var docStr strings.Builder
	docStr.WriteString(`<root>`)
	for range n {
		docStr.WriteString(`<a/><c/><b/>`)
	}
	docStr.WriteString(`</root>`)
	doc, err := helium.NewParser().Parse(t.Context(), []byte(docStr.String()))
	require.NoError(t, err)
	return doc
}
