package relaxng_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

// compileGrammar compiles a RELAX NG grammar from a string, failing the test on
// any compile error.
func compileGrammar(t *testing.T, schema string) *relaxng.Grammar {
	t.Helper()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
	require.NoError(t, err)

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	grammar, err := relaxng.NewCompiler().ErrorHandler(collector).Compile(t.Context(), doc)
	require.NoError(t, err)
	_ = collector.Close()
	_, compileErrors := partitionCompileErrors(collector.Errors())
	require.Empty(t, compileErrors, "grammar should compile without errors")

	return grammar
}

// TestNaiveGroupBacktracking exercises the bare-<group> start path, which uses
// validateGroup (no element-content context). A greedy zeroOrMore member must
// not strand a later mandatory member: zeroOrMore should yield items back so the
// mandatory member can still match.
func TestNaiveGroupBacktracking(t *testing.T) {
	t.Parallel()

	// start is a bare <group> whose first member greedily matches zero-or-more
	// "root" elements and whose second member requires exactly one "root".
	const schema = `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <group>
      <zeroOrMore>
        <element name="root"><empty/></element>
      </zeroOrMore>
      <element name="root"><empty/></element>
    </group>
  </start>
</grammar>`

	grammar := compileGrammar(t, schema)

	// zeroOrMore matches 0, the mandatory element matches the single root.
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
	require.NoError(t, err)

	err = relaxng.NewValidator(grammar).Validate(t.Context(), doc)
	require.NoError(t, err, "single root should validate against group(zeroOrMore(root), root)")
}

// TestNaiveGroupBacktrackingInvalid ensures the backtracking fix does not make
// the naive group accept content it should reject. With a fixed trailing
// member after a greedy zeroOrMore, an instance that supplies only the
// optional kind and never the mandatory one must still fail.
func TestNaiveGroupBacktrackingInvalid(t *testing.T) {
	t.Parallel()

	// start is a bare <group> of zeroOrMore "a" followed by a mandatory "b".
	const schema = `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <group>
      <zeroOrMore>
        <element name="a"><empty/></element>
      </zeroOrMore>
      <element name="b"><empty/></element>
    </group>
  </start>
</grammar>`

	grammar := compileGrammar(t, schema)

	// Only "a" elements, never the mandatory "b": must be rejected even after
	// the greedy zeroOrMore yields items back.
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<a/>`))
	require.NoError(t, err)

	err = relaxng.NewValidator(grammar).Validate(t.Context(), doc)
	require.Error(t, err, "document with no mandatory trailing element must be rejected")
}

// TestNaiveGroupBacktrackingFlexKinds covers the optional and oneOrMore branches
// of backtrackGroupNaive (the originally-added test only exercised zeroOrMore).
// The naive group path matches the single top-level document element, so each
// flexible member competes for that one element.
func TestNaiveGroupBacktrackingFlexKinds(t *testing.T) {
	t.Parallel()

	mk := func(members string) string {
		return `<grammar xmlns="http://relaxng.org/ns/structure/1.0"><start><group>` +
			members + `</group></start></grammar>`
	}
	root := `<element name="root"><empty/></element>`

	cases := []struct {
		name   string
		schema string
		valid  bool
	}{
		// optional greedily takes the root, the mandatory member then fails, and
		// backtracking yields the optional to zero so the mandatory matches.
		{"optional yields for mandatory", mk(`<optional>` + root + `</optional>` + root), true},
		// oneOrMore takes the only root (it cannot go below one), leaving nothing
		// for the mandatory member: correctly rejected.
		{"oneOrMore cannot yield below one", mk(`<oneOrMore>` + root + `</oneOrMore>` + root), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			grammar := compileGrammar(t, tc.schema)
			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
			require.NoError(t, err)
			verr := relaxng.NewValidator(grammar).Validate(t.Context(), doc)
			if tc.valid {
				require.NoError(t, verr)
				return
			}
			require.Error(t, verr)
		})
	}
}

// TestMultiFlexibleGroupBacktracking covers groups with two or more flexible
// members (zeroOrMore/oneOrMore/optional) that must each yield content. The
// backtracker must cascade reductions recursively so a second flexible member
// does not re-grab content a later mandatory member needs. This exercises both
// the naive-group path (backtrackGroupNaive) and the element-content path
// (backtrackGroupFlexible).
func TestMultiFlexibleGroupBacktracking(t *testing.T) {
	t.Parallel()

	naive := func(members string) string {
		return `<grammar xmlns="http://relaxng.org/ns/structure/1.0"><start><group>` +
			members + `</group></start></grammar>`
	}
	content := func(members string) string {
		return `<grammar xmlns="http://relaxng.org/ns/structure/1.0"><start>` +
			`<element name="root"><group>` + members + `</group></element></start></grammar>`
	}
	root := `<element name="root"><empty/></element>`
	a := `<element name="a"><empty/></element>`
	z := func(p string) string { return `<zeroOrMore>` + p + `</zeroOrMore>` }
	o := func(p string) string { return `<optional>` + p + `</optional>` }

	cases := []struct {
		name   string
		schema string
		doc    string
		valid  bool
	}{
		// Naive path: two zeroOrMore both yield 0 so the mandatory member matches.
		{"naive zz+m", naive(z(root) + z(root) + root), `<root/>`, true},
		// Naive path: optional + zeroOrMore both yield 0 for the mandatory member.
		{"naive oz+m", naive(o(root) + z(root) + root), `<root/>`, true},
		// Element-content path: two zeroOrMore yield 0 so the mandatory a matches.
		{"content zz+m", content(z(a) + z(a) + a), `<root><a/></root>`, true},
		// Element-content path: with more content, the flexible members can still
		// consume while leaving exactly one a for the mandatory member.
		{"content zz+m many", content(z(a) + z(a) + a), `<root><a/><a/><a/></root>`, true},
		// Guard against false-accept: no element for the mandatory member.
		{"content zz+m empty rejects", content(z(a) + z(a) + a), `<root></root>`, false},
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
