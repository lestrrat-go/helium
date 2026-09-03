package relaxng_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

const rngNS = "http://relaxng.org/ns/structure/1.0"

// halfSplit is a document in which an interleave branch's first member is
// present and its second is missing, so a composite branch that consumed only
// part of itself is left incomplete.
const halfSplit = "<r><a/><s/></r>"

// compileInterleave compiles schema and returns the concatenated fatal
// compile-error text.
func compileInterleave(t *testing.T, schema string) string {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
	require.NoError(t, err)

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, err = relaxng.NewCompiler().Label("test.rng").ErrorHandler(collector).Compile(t.Context(), doc)
	require.NoError(t, err)
	_ = collector.Close()
	_, compileErrors := partitionCompileErrors(collector.Errors())
	return compileErrors
}

// TestInterleaveNameClassConflict covers the compile-time RELAX NG §7.4
// interleave conflict check: overlapping element/text name classes and
// overlapping attribute name classes across interleave branches are fatal
// compile errors, reported in libxml2's exact wording.
func TestInterleaveNameClassConflict(t *testing.T) {
	t.Parallel()

	const elemConflictMsg = "element interleave: Relax-NG parser error : Element or text conflicts in interleave"
	const attrConflictMsg = "element interleave: Relax-NG parser error : Attributes conflicts in interleave"

	t.Run("two same-name element branches conflict", func(t *testing.T) {
		t.Parallel()
		errs := compileInterleave(t, `<element name="root" xmlns="`+rngNS+`">
  <interleave>
    <element name="a"><empty/></element>
    <element name="a"><text/></element>
  </interleave>
</element>`)
		require.Contains(t, errs, elemConflictMsg)
	})

	t.Run("text in two branches conflicts", func(t *testing.T) {
		t.Parallel()
		errs := compileInterleave(t, `<element name="root" xmlns="`+rngNS+`">
  <interleave>
    <text/>
    <text/>
  </interleave>
</element>`)
		require.Contains(t, errs, elemConflictMsg)
	})

	t.Run("mixed whose group contains text conflicts", func(t *testing.T) {
		t.Parallel()
		errs := compileInterleave(t, `<element name="root" xmlns="`+rngNS+`">
  <mixed>
    <element name="a"><empty/></element>
    <text/>
  </mixed>
</element>`)
		require.Contains(t, errs, elemConflictMsg)
	})

	t.Run("anyName element vs named element conflicts", func(t *testing.T) {
		t.Parallel()
		errs := compileInterleave(t, `<element name="root" xmlns="`+rngNS+`">
  <interleave>
    <element><anyName/><empty/></element>
    <element name="a"><empty/></element>
  </interleave>
</element>`)
		require.Contains(t, errs, elemConflictMsg)
	})

	t.Run("conflict reached through ref", func(t *testing.T) {
		t.Parallel()
		errs := compileInterleave(t, `<grammar xmlns="`+rngNS+`">
  <start>
    <element name="root">
      <interleave>
        <element name="a"><empty/></element>
        <ref name="optA"/>
      </interleave>
    </element>
  </start>
  <define name="optA">
    <optional><element name="a"><text/></element></optional>
  </define>
</grammar>`)
		require.Contains(t, errs, elemConflictMsg)
	})

	t.Run("combine interleave defines flags second define's line", func(t *testing.T) {
		t.Parallel()
		// The second <define> below starts on line 6.
		errs := compileInterleave(t, `<grammar xmlns="`+rngNS+`">
  <start><ref name="foo"/></start>
  <define name="foo" combine="interleave">
    <element name="a"><empty/></element>
  </define>
  <define name="foo" combine="interleave">
    <element name="a"><text/></element>
  </define>
</grammar>`)
		require.Contains(t, errs, elemConflictMsg)
		require.Contains(t, errs, ":6: "+elemConflictMsg)
	})

	t.Run("two attribute branches through refs conflict", func(t *testing.T) {
		t.Parallel()
		errs := compileInterleave(t, `<grammar xmlns="`+rngNS+`">
  <start>
    <element name="root">
      <interleave>
        <ref name="attrA"/>
        <ref name="attrB"/>
      </interleave>
    </element>
  </start>
  <define name="attrA"><attribute name="x"/></define>
  <define name="attrB"><attribute name="x"/></define>
</grammar>`)
		require.Contains(t, errs, attrConflictMsg)
	})

	t.Run("no conflict for anyName-except vs the excluded name", func(t *testing.T) {
		t.Parallel()
		errs := compileInterleave(t, `<element name="root" xmlns="`+rngNS+`">
  <interleave>
    <element><anyName><except><name>a</name></except></anyName><empty/></element>
    <element name="a"><empty/></element>
  </interleave>
</element>`)
		require.Empty(t, errs)
	})

	t.Run("no conflict for nsName in a different namespace than a plain name", func(t *testing.T) {
		t.Parallel()
		errs := compileInterleave(t, `<element name="root" xmlns="`+rngNS+`">
  <interleave>
    <element><nsName ns="urn:x"/><empty/></element>
    <element name="a"><empty/></element>
  </interleave>
</element>`)
		require.Empty(t, errs)
	})

	t.Run("no conflict between text and elements", func(t *testing.T) {
		t.Parallel()
		errs := compileInterleave(t, `<element name="root" xmlns="`+rngNS+`">
  <interleave>
    <text/>
    <element name="a"><empty/></element>
  </interleave>
</element>`)
		require.Empty(t, errs)
	})

	t.Run("mixed(a, b) has no conflict", func(t *testing.T) {
		t.Parallel()
		errs := compileInterleave(t, `<element name="root" xmlns="`+rngNS+`">
  <mixed>
    <element name="a"><empty/></element>
    <element name="b"><empty/></element>
  </mixed>
</element>`)
		require.Empty(t, errs)
	})
}

// TestGoldenSchemasInterleaveConflicts checks every committed RELAX NG golden
// schema for §7.4 interleave conflicts: every schema except interleave0_0.rng
// and tutor14_1.rng must compile without an interleave-conflict diagnostic,
// and those two must produce exactly the "Element or text conflicts in
// interleave" line at the line their <interleave> starts on (libxml2 parity;
// see the design's section 2.2).
func TestGoldenSchemasInterleaveConflicts(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(testdataBase, "test")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	expected := map[string]string{
		"interleave0_0.rng": "interleave0_0.rng:4: element interleave: Relax-NG parser error : " +
			"Element or text conflicts in interleave\n",
		"tutor14_1.rng": "tutor14_1.rng:21: element interleave: Relax-NG parser error : " +
			"Element or text conflicts in interleave\n",
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".rng" {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(filepath.Join(dir, name))
			require.NoError(t, err)
			doc, err := helium.NewParser().Parse(t.Context(), data)
			if err != nil {
				// Not well-formed XML (e.g. broken-xml.rng), irrelevant to the
				// §7.4 interleave-conflict check this test covers.
				return
			}

			collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
			_, err = relaxng.NewCompiler().Label(name).ErrorHandler(collector).Compile(t.Context(), doc)
			require.NoError(t, err)
			_ = collector.Close()
			_, compileErrors := partitionCompileErrors(collector.Errors())

			if want, ok := expected[name]; ok {
				require.Equal(t, want, compileErrors)
				return
			}
			require.NotContains(t, compileErrors, "conflicts in interleave")
		})
	}
}

// TestInterleaveSplitCompositeBranch covers an <interleave> branch whose content
// is a COMPOSITE — a nested interleave, or a choice/optional over a group or
// interleave. RELAX NG lets each branch's members appear anywhere among the
// other branches' members, so such a branch's members need not be contiguous in
// the document.
//
// The compile-time partition (interleave.go) routes every content node to the
// one branch whose reachable leaves accept it, regardless of how deep inside a
// composite branch those leaves sit — collectInterleaveLeaves walks through
// group/choice/interleave/optional/zeroOrMore/oneOrMore. Only a bare <group>
// branch used to support splitting, which rejected valid documents for every
// other composite shape.
//
// Test corpus from #1475 by dac-lu.
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
		{"nested interleave missing member", content(ilv(a, b)), halfSplit, false},
		{"nested interleave undeclared element", content(ilv(a, b)), `<r><a/><s/><b/><c/></r>`, false},
		{"nested interleave duplicate member", content(ilv(a, b)), `<r><a/><s/><b/><a/></r>`, false},
		{"nested interleave missing sibling", content(ilv(a, b)), `<r><a/><b/></r>`, false},

		// optional(composite) is all-or-nothing: absent is fine, half is not.
		{"optional group absent", content(opt(grp(a, b))), `<r><s/></r>`, true},
		{"optional group half rejects", content(opt(grp(a, b))), halfSplit, false},
		{"optional interleave absent", content(opt(ilv(a, b))), `<r><s/></r>`, true},
		{"optional interleave half rejects", content(opt(ilv(a, b))), `<r><b/><s/></r>`, false},

		// A bare group branch is likewise all-or-nothing.
		{"group half rejects", content(grp(a, b)), halfSplit, false},
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

// TestInterleaveChoiceArmPrefersConsuming pins the arm-selection rule for a
// choice branch inside an interleave. Its own branch's sub-sequence may hold
// more than one node the choice must account for, so a choice arm that matches
// but leaves part of that sub-sequence behind (here, an arm made entirely of
// optionals, which matches the empty sequence trivially) must not shadow the
// arm that actually describes the remaining content — validateInterleaveContent
// retries with the exact-choice mode (7.5), which keeps the arm that leaves the
// fewest nodes behind.
//
// Test corpus from #1475 by dac-lu.
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
// the blocked branch's own errors are reported instead (validateInterleaveContent,
// section 7.3 step 4: errors appended after an element is consumed survive
// suppression and are emitted before "Invalid sequence in interleave").
//
// Test corpus from #1475 by dac-lu.
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

// TestInterleaveSplitExpansionBounded guards against exponential blow-up on an
// interleave with many branches. The partition engine routes each content node
// to its branch in a single O(children × wild branches) pass — it never
// enumerates candidate matchings — so validation stays fast no matter how many
// optional single-element branches an interleave has.
//
// Test corpus from #1475 by dac-lu.
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
	require.Less(t, time.Since(start), 5*time.Second,
		"an interleave of many single-element branches must stay linear")
}

// TestInterleaveCompositeBranch is the branch-shape matrix from issue #1474: an
// interleave branch whose content is a composite (group/interleave/choice/ref,
// bare or wrapped in optional/zeroOrMore) interleaved with a plain sibling
// element <s>. Every shape accepts the branch's members in any order relative
// to <s>, and every shape but "choice(group(a,b), a)" rejects a document that
// gives the branch only half its members — that one shape accepts through its
// bare "a" arm instead.
func TestInterleaveCompositeBranch(t *testing.T) {
	t.Parallel()

	a := `<element name="a"><empty/></element>`
	b := `<element name="b"><empty/></element>`
	s := `<element name="s"><empty/></element>`
	grp := func(p ...string) string { return `<group>` + strings.Join(p, "") + `</group>` }
	ilv := func(p ...string) string { return `<interleave>` + strings.Join(p, "") + `</interleave>` }
	opt := func(p string) string { return `<optional>` + p + `</optional>` }
	cho := func(p ...string) string { return `<choice>` + strings.Join(p, "") + `</choice>` }
	zz := func(p string) string { return `<zeroOrMore>` + p + `</zeroOrMore>` }

	cases := []struct {
		name          string
		branch        string
		defines       string
		acceptPartial bool // whether <r><a/><s/></r> (missing b) still validates
	}{
		{"group(a,b)", grp(a, b), "", false},
		{"zeroOrMore(group(a,b))", zz(grp(a, b)), "", false},
		{"interleave(a,b)", ilv(a, b), "", false},
		{"optional(group(a,b))", opt(grp(a, b)), "", false},
		{"optional(interleave(a,b))", opt(ilv(a, b)), "", false},
		{"choice(group(a,b))", cho(grp(a, b)), "", false},
		{"choice(interleave(a,b))", cho(ilv(a, b)), "", false},
		{"choice(group(a,b), a)", cho(grp(a, b), a), "", true},
		{"ref to interleave(a,b)", `<ref name="ab"/>`, `<define name="ab">` + ilv(a, b) + `</define>`, false},
	}

	accept := []string{`<r><a/><b/><s/></r>`, `<r><a/><s/><b/></r>`, `<r><s/><a/><b/></r>`}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			schema := `<grammar xmlns="` + rngNS + `"><start>` +
				`<element name="r"><interleave>` + tc.branch + s + `</interleave></element></start>` +
				tc.defines + `</grammar>`
			grammar := compileGrammar(t, schema)

			for _, docStr := range accept {
				doc, err := helium.NewParser().Parse(t.Context(), []byte(docStr))
				require.NoError(t, err)
				verr := relaxng.NewValidator(grammar).Validate(t.Context(), doc)
				require.NoError(t, verr, "%s: %s should validate", tc.name, docStr)
			}

			doc, err := helium.NewParser().Parse(t.Context(), []byte(halfSplit))
			require.NoError(t, err)
			verr := relaxng.NewValidator(grammar).Validate(t.Context(), doc)
			if tc.acceptPartial {
				require.NoError(t, verr, "%s: the bare-a arm should still validate", tc.name)
				return
			}
			require.Error(t, verr, "%s: a document missing b should be rejected", tc.name)
		})
	}
}

// TestInterleaveRejectsIncompleteBranch covers the converse of issue #1474: an
// interleave must still reject a document that gives one of its branches only
// PART of what that branch requires, even though the partition splits the
// document into the right sub-sequences.
func TestInterleaveRejectsIncompleteBranch(t *testing.T) {
	t.Parallel()

	t.Run("interleave(group(a,b), s) missing b", func(t *testing.T) {
		t.Parallel()
		schema := `<grammar xmlns="` + rngNS + `"><start><element name="r"><interleave>` +
			`<group><element name="a"><empty/></element><element name="b"><empty/></element></group>` +
			`<element name="s"><empty/></element>` +
			`</interleave></element></start></grammar>`
		grammar := compileGrammar(t, schema)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(halfSplit))
		require.NoError(t, err)
		require.Error(t, relaxng.NewValidator(grammar).Validate(t.Context(), doc))
	})

	t.Run("interleave(group(a,b,c), s) missing c", func(t *testing.T) {
		t.Parallel()
		schema := `<grammar xmlns="` + rngNS + `"><start><element name="r"><interleave>` +
			`<group><element name="a"><empty/></element><element name="b"><empty/></element>` +
			`<element name="c"><empty/></element></group>` +
			`<element name="s"><empty/></element>` +
			`</interleave></element></start></grammar>`
		grammar := compileGrammar(t, schema)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<r><a/><b/><s/></r>`))
		require.NoError(t, err)
		require.Error(t, relaxng.NewValidator(grammar).Validate(t.Context(), doc))
	})

	t.Run("mixed(a, b) missing b", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="r" xmlns="` + rngNS + `">` +
			`<mixed><element name="a"><empty/></element><element name="b"><empty/></element></mixed>` +
			`</element>`
		grammar := compileGrammar(t, schema)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<r>t<a/>t</r>`))
		require.NoError(t, err)
		require.Error(t, relaxng.NewValidator(grammar).Validate(t.Context(), doc))
	})

	t.Run("oneOrMore(group(a,b)) leaves a dangling a", func(t *testing.T) {
		t.Parallel()
		schema := `<grammar xmlns="` + rngNS + `"><start><element name="r"><interleave>` +
			`<oneOrMore><group><element name="a"><empty/></element><element name="b"><empty/></element></group></oneOrMore>` +
			`<zeroOrMore><element name="c"><empty/></element></zeroOrMore>` +
			`</interleave></element></start></grammar>`
		grammar := compileGrammar(t, schema)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<r><a/><c/><b/><c/><a/></r>`))
		require.NoError(t, err)

		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelError)
		verr := relaxng.NewValidator(grammar).ErrorHandler(collector).Validate(t.Context(), doc)
		require.Error(t, verr)
		var report strings.Builder
		for _, e := range collector.Errors() {
			report.WriteString(e.Error())
		}
		require.Contains(t, report.String(), "Relax-NG validity error : Extra element a in interleave")
	})
}

// TestInterleaveNullableMemberAfterSibling covers the shapes the round-robin
// engine's documented residual rejected: a group whose trailing member is
// nullable (optional/zeroOrMore) can still be split by a sibling interleave
// branch, whichever position the sibling arrives at. It also covers the
// exact-choice retry (7.5): a choice arm that matches but leaves part of its
// own sub-sequence behind must retry picking the arm consuming the most,
// without breaking the ordinary (non-interleave) greedy-choice-then-backtrack
// path that already handles this shape outside an interleave.
func TestInterleaveNullableMemberAfterSibling(t *testing.T) {
	t.Parallel()

	t.Run("group(a, optional(b)) split by a sibling", func(t *testing.T) {
		t.Parallel()
		schema := `<grammar xmlns="` + rngNS + `"><start><element name="r"><interleave>` +
			`<group><element name="a"><empty/></element>` +
			`<optional><element name="b"><empty/></element></optional></group>` +
			`<element name="s"><empty/></element>` +
			`</interleave></element></start></grammar>`
		grammar := compileGrammar(t, schema)
		for _, docStr := range []string{`<r><a/><s/><b/></r>`, `<r><a/><b/><s/></r>`, halfSplit} {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(docStr))
			require.NoError(t, err)
			require.NoError(t, relaxng.NewValidator(grammar).Validate(t.Context(), doc), "%s should validate", docStr)
		}
	})

	t.Run("group(a, zeroOrMore(b)) split by a sibling", func(t *testing.T) {
		t.Parallel()
		schema := `<grammar xmlns="` + rngNS + `"><start><element name="r"><interleave>` +
			`<group><element name="a"><empty/></element>` +
			`<zeroOrMore><element name="b"><empty/></element></zeroOrMore></group>` +
			`<element name="s"><empty/></element>` +
			`</interleave></element></start></grammar>`
		grammar := compileGrammar(t, schema)
		for _, docStr := range []string{`<r><a/><s/><b/></r>`, `<r><a/><b/><s/></r>`, halfSplit} {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(docStr))
			require.NoError(t, err)
			require.NoError(t, relaxng.NewValidator(grammar).Validate(t.Context(), doc), "%s should validate", docStr)
		}
	})

	t.Run("choice(d, group(d,d)) interleaved with optional(e)", func(t *testing.T) {
		t.Parallel()
		schema := `<grammar xmlns="` + rngNS + `"><start><element name="r"><interleave>` +
			`<choice><element name="d"><empty/></element>` +
			`<group><element name="d"><empty/></element><element name="d"><empty/></element></group></choice>` +
			`<optional><element name="e"><empty/></element></optional>` +
			`</interleave></element></start></grammar>`
		grammar := compileGrammar(t, schema)
		for _, docStr := range []string{`<r><d/><d/><e/></r>`, `<r><d/><e/></r>`} {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(docStr))
			require.NoError(t, err)
			require.NoError(t, relaxng.NewValidator(grammar).Validate(t.Context(), doc), "%s should validate", docStr)
		}
	})

	t.Run("choice(d, group(d,d)) then d in a plain group (no interleave)", func(t *testing.T) {
		t.Parallel()
		schema := `<grammar xmlns="` + rngNS + `"><start><element name="r"><group>` +
			`<choice><element name="d"><empty/></element>` +
			`<group><element name="d"><empty/></element><element name="d"><empty/></element></group></choice>` +
			`<element name="d"><empty/></element>` +
			`</group></element></start></grammar>`
		grammar := compileGrammar(t, schema)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<r><d/><d/></r>`))
		require.NoError(t, err)
		require.NoError(t, relaxng.NewValidator(grammar).Validate(t.Context(), doc))
	})
}

// manyCompositeBranchesSchema builds an interleave of k optional(group(x_i,y_i))
// branches over k distinct element-name pairs, plus a sibling <s/> branch.
func manyCompositeBranchesSchema(k int) string {
	var branches strings.Builder
	branches.WriteString(`<grammar xmlns="` + rngNS + `"><start><element name="r"><interleave>`)
	for i := range k {
		branches.WriteString(`<optional><group><element name="x`)
		fmtInt(&branches, i)
		branches.WriteString(`"><empty/></element><element name="y`)
		fmtInt(&branches, i)
		branches.WriteString(`"><empty/></element></group></optional>`)
	}
	branches.WriteString(`<element name="s"><empty/></element>`)
	branches.WriteString(`</interleave></element></start></grammar>`)
	return branches.String()
}

// manyCompositeBranchesDoc builds <r> with every x_i then <s/> then every
// y_i, for i = 1..k-1 — branch 0 is entirely omitted, and every other
// branch's two members are split apart by the sibling <s/>.
func manyCompositeBranchesDoc(k int) string {
	var body strings.Builder
	body.WriteString(`<r>`)
	for i := 1; i < k; i++ {
		body.WriteString(`<x`)
		fmtInt(&body, i)
		body.WriteString(`/>`)
	}
	body.WriteString(`<s/>`)
	for i := 1; i < k; i++ {
		body.WriteString(`<y`)
		fmtInt(&body, i)
		body.WriteString(`/>`)
	}
	body.WriteString(`</r>`)
	return body.String()
}

func fmtInt(b *strings.Builder, n int) {
	b.WriteString(strconv.Itoa(n))
}

// TestInterleaveManyCompositeBranchesLinear is the cap probe from #1474/the
// design's evidence gathering (repro2/cap2_test.go): an interleave of k
// optional(group(x_i,y_i)) branches plus a sibling <s/>, fed a document that
// omits branch 0 and splits every other branch's two members across <s/>,
// must accept in well under a second at k = 9, 12 and 40 — the partition
// engine routes in one O(children × wild branches) pass, so it never needs a
// cap on the number of branches (unlike a round-robin engine that must
// silently degrade past some k).
func TestInterleaveManyCompositeBranchesLinear(t *testing.T) {
	t.Parallel()

	for _, k := range []int{9, 12, 40} {
		t.Run(strconv.Itoa(k), func(t *testing.T) {
			t.Parallel()
			grammar := compileGrammar(t, manyCompositeBranchesSchema(k))
			doc, err := helium.NewParser().Parse(t.Context(), []byte(manyCompositeBranchesDoc(k)))
			require.NoError(t, err)

			start := time.Now()
			verr := relaxng.NewValidator(grammar).Validate(t.Context(), doc)
			require.NoError(t, verr)
			require.Less(t, time.Since(start), time.Second, "k=%d should validate in well under a second", k)
		})
	}
}

// TestInterleaveGroupMemoAcrossPartitions covers the memo-key hazard: two
// choice arms, each an interleave containing a group starting with the SAME
// define R, must not share a group-validation memo entry just because they
// probe R at the same input position with the same remaining length — the
// memo key's sibling-array id (validState.run, 7.1) distinguishes them.
func TestInterleaveGroupMemoAcrossPartitions(t *testing.T) {
	t.Parallel()

	schema := `<grammar xmlns="` + rngNS + `"><start><element name="r"><choice>` +
		`<interleave><group><ref name="R"/>` +
		`<element name="a"><empty/></element><element name="z"><empty/></element></group>` +
		`<element name="b"><empty/></element></interleave>` +
		`<interleave><group><ref name="R"/><element name="b"><empty/></element></group>` +
		`<element name="a"><empty/></element></interleave>` +
		`</choice></element></start>` +
		`<define name="R"><group><element name="x"><empty/></element>` +
		`<optional><element name="y"><empty/></element></optional></group></define>` +
		`</grammar>`
	grammar := compileGrammar(t, schema)
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<r><x/><a/><b/></r>`))
	require.NoError(t, err)
	require.NoError(t, relaxng.NewValidator(grammar).Validate(t.Context(), doc),
		"the second choice arm must validate independently of the first arm's memoized R")
}

// TestInterleaveMixedContent covers text/data/attribute leaves inside an
// interleave: mixed(a, b) still enforces the group's element ORDER (mixed is
// interleave(text, group(a,b)), and only the text placement is free);
// interleave(text, element) accepts text anywhere around the element; an
// interleave with a data/attribute branch validates the attribute and the
// element's text content independently; and a CDATA section between branches
// with no text-accepting branch is never routed and so is rejected.
func TestInterleaveMixedContent(t *testing.T) {
	t.Parallel()

	t.Run("mixed(a, b) enforces group order despite free text", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="r" xmlns="` + rngNS + `">` +
			`<mixed><element name="a"><empty/></element><element name="b"><empty/></element></mixed>` +
			`</element>`
		grammar := compileGrammar(t, schema)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<r>t1<a/>t2<b/>t3</r>`))
		require.NoError(t, err)
		require.NoError(t, relaxng.NewValidator(grammar).Validate(t.Context(), doc))

		doc, err = helium.NewParser().Parse(t.Context(), []byte(`<r>t1<b/>t2<a/></r>`))
		require.NoError(t, err)
		require.Error(t, relaxng.NewValidator(grammar).Validate(t.Context(), doc))
	})

	t.Run("interleave(text, a) accepts text anywhere", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="r" xmlns="` + rngNS + `">` +
			`<interleave><text/><element name="a"><empty/></element></interleave>` +
			`</element>`
		grammar := compileGrammar(t, schema)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<r> x <a/> y </r>`))
		require.NoError(t, err)
		require.NoError(t, relaxng.NewValidator(grammar).Validate(t.Context(), doc))
	})

	t.Run("interleave(attribute k, data int) validates both independently", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="r" xmlns="` + rngNS + `" datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes">` +
			`<interleave><attribute name="k"><text/></attribute><data type="int"/></interleave>` +
			`</element>`
		grammar := compileGrammar(t, schema)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<r k="1"> 42 </r>`))
		require.NoError(t, err)
		require.NoError(t, relaxng.NewValidator(grammar).Validate(t.Context(), doc))

		doc, err = helium.NewParser().Parse(t.Context(), []byte(`<r k="1"> xx </r>`))
		require.NoError(t, err)
		require.Error(t, relaxng.NewValidator(grammar).Validate(t.Context(), doc))
	})

	t.Run("CDATA between branches rejects when no text branch exists", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="r" xmlns="` + rngNS + `">` +
			`<interleave><element name="a"><empty/></element><element name="b"><empty/></element></interleave>` +
			`</element>`
		grammar := compileGrammar(t, schema)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<r><a/><![CDATA[x]]><b/></r>`))
		require.NoError(t, err)
		require.Error(t, relaxng.NewValidator(grammar).Validate(t.Context(), doc))
	})
}

// TestInterleaveDiagnostics pins the exact diagnostic text for every row of
// the interleave failure table (see the design's section 7.4). Four rows
// reuse the golden schemas/instances (tutor8_2.rng and tutor9_5.rng, copied
// inline from testdata/libxml2-compat/relaxng/test) verbatim, so the wording
// stays pinned even if those golden files move.
func TestInterleaveDiagnostics(t *testing.T) {
	t.Parallel()

	const headSchema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <ref name="head"/>
  </start>
  <define name="head">
    <element name="head">
      <interleave>
        <ref name="title"/>
        <optional>
          <ref name="base"/>
        </optional>
        <zeroOrMore>
          <ref name="style"/>
        </zeroOrMore>
        <zeroOrMore>
          <ref name="script"/>
        </zeroOrMore>
        <zeroOrMore>
          <ref name="link"/>
        </zeroOrMore>
        <zeroOrMore>
          <ref name="meta"/>
        </zeroOrMore>
      </interleave>
    </element>
  </define>
  <define name="title">
    <element name="title">
      <text/>
    </element>
  </define>
  <define name="base">
    <element name="base">
      <text/>
    </element>
  </define>
  <define name="style">
    <element name="style">
      <text/>
    </element>
  </define>
  <define name="script">
    <element name="script">
      <text/>
    </element>
  </define>
  <define name="meta">
    <element name="meta">
      <text/>
    </element>
  </define>
  <define name="link">
    <element name="link">
      <text/>
    </element>
  </define>
</grammar>`

	const cardSchema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="addressBook">
      <zeroOrMore>
        <element name="card">
          <ref name="card.attlist"/>
        </element>
      </zeroOrMore>
    </element>
  </start>
  <define name="card.attlist" combine="interleave">
    <attribute name="name">
      <text/>
    </attribute>
  </define>
  <define name="card.attlist" combine="interleave">
    <attribute name="email">
      <text/>
    </attribute>
  </define>
</grammar>`

	validate := func(t *testing.T, schema, doc string) string {
		t.Helper()
		grammar := compileGrammar(t, schema)
		parsed, err := helium.NewParser().Parse(t.Context(), []byte(doc))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelError)
		verr := relaxng.NewValidator(grammar).Label("test.xml").ErrorHandler(collector).Validate(t.Context(), parsed)
		require.Error(t, verr)
		var report strings.Builder
		for _, e := range collector.Errors() {
			report.WriteString(e.Error())
		}
		return report.String()
	}

	t.Run("required branch got nothing (tutor8_2_5)", func(t *testing.T) {
		t.Parallel()
		got := validate(t, headSchema, "<head>\n  <meta>meta2</meta>\n</head>")
		require.Equal(t,
			"test.xml:1: element head: Relax-NG validity error : Expecting an element title, got nothing\n"+
				"test.xml:1: element head: Relax-NG validity error : Invalid sequence in interleave\n"+
				"test.xml:1: element head: Relax-NG validity error : Element head failed to validate content\n",
			got)
	})

	t.Run("attribute branch unmatched (tutor9_5_2)", func(t *testing.T) {
		t.Parallel()
		got := validate(t, cardSchema, "<addressBook>\n  <card name=\"foo\"/>\n</addressBook>")
		require.Equal(t,
			"test.xml:2: element card: Relax-NG validity error : Invalid sequence in interleave\n"+
				"test.xml:2: element card: Relax-NG validity error : Element card failed to validate attributes\n",
			got)
	})

	t.Run("branch left an element, duplicate title (tutor8_2_4)", func(t *testing.T) {
		t.Parallel()
		got := validate(t, headSchema,
			"<head>\n  <meta>meta1</meta>\n  <title>foo</title>\n  <meta>meta2</meta>\n  <title>error</title>\n</head>")
		require.Equal(t,
			"Relax-NG validity error : Extra element title in interleave\n"+
				"test.xml:5: element title: Relax-NG validity error : Element head failed to validate content\n",
			got)
	})

	t.Run("branch left an element, duplicate base (tutor8_2_6)", func(t *testing.T) {
		t.Parallel()
		got := validate(t, headSchema,
			"<head>\n  <base>base</base>\n  <title>foo</title>\n  <base>error</base>\n</head>")
		require.Equal(t,
			"Relax-NG validity error : Extra element base in interleave\n"+
				"test.xml:4: element base: Relax-NG validity error : Element head failed to validate content\n",
			got)
	})

	t.Run("branch element has bad content", func(t *testing.T) {
		t.Parallel()
		schema := `<grammar xmlns="` + rngNS + `"><start><element name="r"><interleave>` +
			`<element name="first"><element name="inner"><empty/></element></element>` +
			`<element name="second"><empty/></element>` +
			`</interleave></element></start></grammar>`
		got := validate(t, schema, `<r><first/><second/></r>`)
		require.Contains(t, got, "element first: Relax-NG validity error : Element first failed to validate content")
		require.Contains(t, got, "element r: Relax-NG validity error : Invalid sequence in interleave")
		require.Contains(t, got, "element r: Relax-NG validity error : Element r failed to validate content")
	})

	t.Run("branch got nodes but does not match them", func(t *testing.T) {
		t.Parallel()
		schema := `<grammar xmlns="` + rngNS + `"><start><element name="r"><interleave>` +
			`<group><element name="a"><empty/></element><element name="b"><empty/></element></group>` +
			`<element name="s"><empty/></element>` +
			`</interleave></element></start></grammar>`
		got := validate(t, schema, halfSplit)
		require.Equal(t,
			"test.xml:1: element r: Relax-NG validity error : Invalid sequence in interleave\n"+
				"test.xml:1: element r: Relax-NG validity error : Element r failed to validate content\n",
			got)
	})
}
