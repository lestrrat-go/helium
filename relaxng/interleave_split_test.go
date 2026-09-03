package relaxng_test

import (
	"strings"
	"testing"
	"time"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

// TestInterleaveSplitCompositeBranch covers an <interleave> branch whose content
// is a COMPOSITE — a nested interleave, or a choice/optional over a group or
// interleave. RELAX NG lets each branch's members appear anywhere among the
// other branches' members, so such a branch's members need not be contiguous in
// the document.
//
// The matcher reaches those documents through two spec identities (see
// validateInterleaveContent): interleave is associative, so a nested interleave
// is spliced into its parent, and interleave distributes over choice — with
// optional(Q) being choice(Q, empty) — so a choice branch is tried arm by arm.
// Only a bare <group> branch used to support splitting, which rejected valid
// documents for every other composite shape.
func TestInterleaveSplitCompositeBranch(t *testing.T) {
	t.Parallel()

	a := benchElementA
	b := `<element name="b"><empty/></element>`
	s := `<element name="s"><empty/></element>`
	content := func(branch string) string {
		return `<grammar xmlns="http://relaxng.org/ns/structure/1.0"><start>` +
			`<element name="r"><interleave>` + branch + s + `</interleave></element></start></grammar>`
	}
	grp := func(p ...string) string { return `<group>` + strings.Join(p, "") + `</group>` }
	ilv := func(p ...string) string { return `<interleave>` + strings.Join(p, "") + `</interleave>` }
	opt := func(p string) string { return `<optional>` + p + `</optional>` }
	cho := func(p ...string) string { return `<choice>` + strings.Join(p, "") + `</choice>` }
	zz := func(p string) string { return `<zeroOrMore>` + p + `</zeroOrMore>` }

	// split places s between a and b, so the branch's two members are not
	// adjacent in the document.
	const split = `<r><a/><s/><b/></r>`

	cases := []struct {
		name   string
		schema string
		doc    string
		valid  bool
	}{
		{"group split", content(grp(a, b)), split, true},
		{"zeroOrMore group split", content(zz(grp(a, b))), split, true},
		{"nested interleave split", content(ilv(a, b)), split, true},
		{"optional group split", content(opt(grp(a, b))), split, true},
		{"optional interleave split", content(opt(ilv(a, b))), split, true},
		{"choice of group split", content(cho(grp(a, b))), split, true},
		{"choice of interleave split", content(cho(ilv(a, b))), split, true},
		{"choice group-or-element split", content(cho(grp(a, b), a)), split, true},

		// The same shapes must still reject what the grammar does not allow.
		{"nested interleave missing member", content(ilv(a, b)), `<r><a/><s/></r>`, false},
		{"nested interleave undeclared element", content(ilv(a, b)), `<r><a/><s/><b/><c/></r>`, false},
		{"nested interleave duplicate member", content(ilv(a, b)), `<r><a/><s/><b/><a/></r>`, false},
		{"nested interleave missing sibling", content(ilv(a, b)), `<r><a/><b/></r>`, false},

		// optional(composite) is all-or-nothing: absent is fine, half is not.
		{"optional group absent", content(opt(grp(a, b))), `<r><s/></r>`, true},
		{"optional group half rejects", content(opt(grp(a, b))), `<r><a/><s/></r>`, false},
		{"optional interleave absent", content(opt(ilv(a, b))), `<r><s/></r>`, true},
		{"optional interleave half rejects", content(opt(ilv(a, b))), `<r><b/><s/></r>`, false},

		// A bare group branch is likewise all-or-nothing.
		{"group half rejects", content(grp(a, b)), `<r><a/><s/></r>`, false},
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

// TestInterleaveChoiceArmPrefersConsuming pins the arm-selection rule in
// validateInterleaveContent. An interleave signals success as soon as its
// required branches are satisfied — it does NOT require the content to be
// exhausted, since the caller may have further patterns to apply. So a choice
// arm made up entirely of optionals succeeds while consuming nothing, and would
// shadow the arm that actually describes the content if arms were taken in
// grammar order. The candidate that consumes the most content must win.
func TestInterleaveChoiceArmPrefersConsuming(t *testing.T) {
	t.Parallel()

	// allOptional comes FIRST in the choice and matches the empty sequence, so
	// taking arms in order would accept it and strand <x/> and <y/>.
	schema := `<grammar xmlns="http://relaxng.org/ns/structure/1.0"><start>` +
		`<element name="r"><interleave>` +
		`<element name="keep"><empty/></element>` +
		`<choice>` +
		`<interleave><optional><element name="p"><empty/></element></optional>` +
		`<optional><element name="q"><empty/></element></optional></interleave>` +
		`<interleave><element name="x"><empty/></element><element name="y"><empty/></element></interleave>` +
		`</choice>` +
		`</interleave></element></start></grammar>`

	cases := []struct {
		name  string
		doc   string
		valid bool
	}{
		{"consuming arm wins over vacuous arm", `<r><x/><keep/><y/></r>`, true},
		{"vacuous arm still matches its own content", `<r><keep/><p/></r>`, true},
		{"vacuous arm with nothing to consume", `<r><keep/></r>`, true},
		{"partial consuming arm rejects", `<r><x/><keep/></r>`, false},
		{"arms may not be mixed", `<r><p/><keep/><x/><y/></r>`, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			grammar := compileGrammar(t, schema)
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

// TestInterleaveSplitReportsRealCause covers the interleave diagnostic. When one
// branch matches the head element by NAME but fails on its own content, the
// remaining branches are left unconsumed only because that branch blocked the
// sequence. Blaming one of them ("Expecting an element X, got nothing" for an
// element the document never even reaches) points away from the real cause, so
// the blocked branch's own errors are reported instead.
func TestInterleaveSplitReportsRealCause(t *testing.T) {
	t.Parallel()

	// <first> requires an <inner> child. Given <first/> empty, the interleave
	// must report first's content failure, not "Expecting an element second".
	schema := `<grammar xmlns="http://relaxng.org/ns/structure/1.0"><start>` +
		`<element name="r"><interleave>` +
		`<element name="first"><element name="inner"><empty/></element></element>` +
		`<element name="second"><empty/></element>` +
		`</interleave></element></start></grammar>`

	grammar := compileGrammar(t, schema)
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<r><first/><second/></r>`))
	require.NoError(t, err)

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelError)
	verr := relaxng.NewValidator(grammar).ErrorHandler(collector).Validate(t.Context(), doc)
	require.Error(t, verr)

	var report strings.Builder
	for _, e := range collector.Errors() {
		report.WriteString(e.Error())
	}
	got := report.String()
	require.Contains(t, got, "element first: Relax-NG validity error : Element first failed to validate content",
		"the blocked branch's own failure must be reported")
	require.NotContains(t, got, "Expecting an element second",
		"an unreached branch must not be blamed")
}

// TestInterleaveSplitExpansionBounded guards the candidate-expansion cap. An
// interleave over many optional single elements must NOT be expanded (a single
// element already matches atomically), so validation stays fast instead of
// enumerating 2^n arms.
func TestInterleaveSplitExpansionBounded(t *testing.T) {
	t.Parallel()

	const n = 22
	var branches, body strings.Builder
	for i := range n {
		name := "e" + string(rune('a'+i%26)) + string(rune('a'+i/26))
		branches.WriteString(`<optional><element name="`)
		branches.WriteString(name)
		branches.WriteString(`"><empty/></element></optional>`)
		body.WriteString("<")
		body.WriteString(name)
		body.WriteString("/>")
	}
	schema := `<grammar xmlns="http://relaxng.org/ns/structure/1.0"><start>` +
		`<element name="r"><interleave>` + branches.String() + `</interleave></element></start></grammar>`

	grammar := compileGrammar(t, schema)
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<r>`+body.String()+`</r>`))
	require.NoError(t, err)

	start := time.Now()
	require.NoError(t, relaxng.NewValidator(grammar).Validate(t.Context(), doc))
	// 2^22 candidates would take minutes; the un-expanded match is microseconds.
	require.Less(t, time.Since(start), 5*time.Second,
		"an interleave of optional single elements must not be expanded")
}
