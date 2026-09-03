package relaxng_test

import (
	"runtime"
	"strings"
	"testing"
	"time"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

// Accept/reject safety evidence for sharing validState.seq instead of copying
// it. The harness that produced these figures is checked in as
// group_backtrack_differential_test.go. Apart from two long-standing helpers in
// relaxng_test.go it touches only the exported API, so the same file runs
// unchanged on an older checkout, and it prints one line per
// case: the case identity, the verdict (VALID/INVALID/SCHEMA-ERROR), and the
// exact error text. Comparing error text and not just the verdict is what makes
// the diff meaningful — a silently reworded diagnostic would otherwise pass
// unnoticed.
//
// Procedure. Run it from a checkout of this branch
// (perf-relaxng-group-backtrack) against the merge base of this branch with
// origin/main. Resolve that base at run time and archive the same object you
// name as the base: this branch is rebased as main moves, so no hardcoded hash
// stays correct.
//
//	BASE=$(git merge-base HEAD origin/main)
//	go test ./relaxng -run '^TestGroupBacktrackDifferential$' -timeout 30m \
//	    -relaxng.differential.out=/tmp/head.txt
//	mkdir -p /tmp/base
//	git archive "$BASE" | tar -x -C /tmp/base
//	cp relaxng/group_backtrack_differential_test.go /tmp/base/relaxng/
//	go -C /tmp/base test ./relaxng -run '^TestGroupBacktrackDifferential$' \
//	    -timeout 30m -relaxng.differential.out=/tmp/base.txt
//	diff /tmp/base.txt /tmp/head.txt
//
// Result: the diff is empty and both files hash to sha256
// 171fc1e474859abe156923f8cd753746b1df91d3cd56e64679b750c3c8a24fb5. Each run
// takes well under a minute and writes 35,819 lines.
//
// Corpus 1, the golden cross-product. Every schema in
// testdata/libxml2-compat/relaxng/test validated against every instance in the
// same directory. 8 of the 105 schemas do not compile and are recorded once
// each as SCHEMA-ERROR (6 pre-existing plus interleave0_0.rng and
// tutor14_1.rng, which violate RELAX NG §7.4's interleave name-class
// disjointness), leaving 97 schemas x 163 instances = 15,811 validated pairs,
// 15,819 lines in all: 517 VALID, 15,294 INVALID.
//
// Corpus 2, randomized group grammars. 20,000 seeded random grammars (seed 1)
// over three shapes, including a bare <group> under <start>, which is the only
// shape that reaches the naive backtracker: 2,889 VALID, 17,111 INVALID
// spanning 184 distinct error texts. This half exercises no interleave shape,
// so it is unchanged by the interleave partitioning engine.
//
// 13,429 of the 32,405 INVALID lines carry diagnostic text. The rest are
// failures the validator reports only through the returned error without
// emitting anything to the error handler, which is identical on both
// revisions.
//
// Coverage under both corpora together (add -coverprofile to either command
// above, then `go tool cover -func`): backtrackGroupFlexible 100%,
// advanceFlexibleContent 100%, backtrackGroupNaive 100%, advanceFlexibleNaive
// 88.9%. Both halves of the corpus are load-bearing: the golden cross-product
// alone (add -relaxng.differential.cases=0) leaves the naive backtracker nearly
// untouched (backtrackGroupNaive 19.4%, advanceFlexibleNaive 0%), because no
// golden schema puts a bare <group> under <start>. Only the randomized set
// exercises that path.
//
// An ordinary `go test ./relaxng` run needs no flags: the harness then walks
// every eighth golden schema and 200 random grammars, discards the output, and
// TestGroupBacktrackDifferentialDeterministic checks that two runs of that
// subset agree byte for byte.
//
// `go test -race ./relaxng` passes on both revisions.

// manyChildrenDoc builds `<root>` with n `<a/>` children.
func manyChildrenDoc(n int) string {
	var d strings.Builder
	d.WriteString(`<root>`)
	for range n {
		d.WriteString(`<a/>`)
	}
	d.WriteString(`</root>`)
	return d.String()
}

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

// TestMultiFlexibleGroupBacktrackingNotExponential guards against a re-introduced
// algorithmic-complexity DoS: a group of N greedy zeroOrMore members followed by
// a mandatory member, matched against M elements, drove the cascading backtracker
// to O(M^N) before group-result memoization was added (N=9/M=25 took ~45s). With
// memoization each distinct (child-range, input-position) subproblem is computed
// once, so the same input validates near-instantly.
func TestMultiFlexibleGroupBacktrackingNotExponential(t *testing.T) {
	t.Parallel()

	const N, M = 10, 30
	a := `<element name="a"><empty/></element>`
	var schema strings.Builder
	schema.WriteString(`<grammar xmlns="http://relaxng.org/ns/structure/1.0"><start><element name="root"><group>`)
	for range N {
		schema.WriteString(`<zeroOrMore>` + a + `</zeroOrMore>`)
	}
	schema.WriteString(a + `</group></element></start></grammar>`)

	var docStr strings.Builder
	docStr.WriteString(`<root>`)
	for range M {
		docStr.WriteString(`<a/>`)
	}
	docStr.WriteString(`</root>`)

	grammar := compileGrammar(t, schema.String())
	doc, err := helium.NewParser().Parse(t.Context(), []byte(docStr.String()))
	require.NoError(t, err)

	start := time.Now()
	verr := relaxng.NewValidator(grammar).Validate(t.Context(), doc)
	elapsed := time.Since(start)

	require.NoError(t, verr, "group(zeroOrMore(a) x%d, a) over %d elements should validate", N, M)
	require.Less(t, elapsed, 5*time.Second, "validation must not be exponential (took %s)", elapsed)
}

// TestGroupBacktrackAllocationBound guards against validState.clone() and its
// callers deep-copying the whole remaining sibling slice on every repetition.
// That copy makes both the flexible-group backtracker and the ordinary,
// no-backtracking zeroOrMore path allocate quadratically or worse in child
// count. Allocated bytes are asserted rather than wall time: bytes are
// deterministic across machines and CI load, wall time is not.
func TestGroupBacktrackAllocationBound(t *testing.T) {
	a := `<element name="a"><empty/></element>`

	t.Run("flexible group backtracking", func(t *testing.T) {
		const m = 1600
		schema := `<grammar xmlns="http://relaxng.org/ns/structure/1.0"><start><element name="root"><group>` +
			`<zeroOrMore>` + a + `</zeroOrMore>` + a + `</group></element></start></grammar>`
		grammar := compileGrammar(t, schema)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(manyChildrenDoc(m)))
		require.NoError(t, err)

		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		verr := relaxng.NewValidator(grammar).Validate(t.Context(), doc)
		runtime.ReadMemStats(&after)
		require.NoError(t, verr, "group(zeroOrMore(a), a) over %d elements should validate", m)

		delta := after.TotalAlloc - before.TotalAlloc
		require.Less(t, delta, uint64(100<<20),
			"validating group(zeroOrMore(a), a) over %d elements allocated %d bytes, want under 100 MB", m, delta)
	})

	t.Run("plain zeroOrMore, no backtracking", func(t *testing.T) {
		const m = 16000
		schema := `<grammar xmlns="http://relaxng.org/ns/structure/1.0"><start><element name="root">` +
			`<zeroOrMore>` + a + `</zeroOrMore></element></start></grammar>`
		grammar := compileGrammar(t, schema)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(manyChildrenDoc(m)))
		require.NoError(t, err)

		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		verr := relaxng.NewValidator(grammar).Validate(t.Context(), doc)
		runtime.ReadMemStats(&after)
		require.NoError(t, verr, "zeroOrMore(a) over %d elements should validate", m)

		delta := after.TotalAlloc - before.TotalAlloc
		require.Less(t, delta, uint64(100<<20),
			"validating zeroOrMore(a) over %d elements allocated %d bytes, want under 100 MB", m, delta)
	})
}
