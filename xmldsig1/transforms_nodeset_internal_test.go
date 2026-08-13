package xmldsig1

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/c14n"
	"github.com/stretchr/testify/require"
)

// diffMethods is every canonicalization method a Reference or a
// CanonicalizationMethod may name, paired with the InclusiveNamespaces prefix
// lists that change what Exclusive C14N renders.
var diffMethods = []string{C14N10, C14N10Comments, ExcC14N10, ExcC14N10Comments, C14N11URI, C14N11Comments}

var diffPrefixLists = [][]string{
	nil,
	{"a"},
	{"a", "b", "absent"},
	{"a", "b", "c", "d", "x", "y"},
}

// canonicalizeSubtreeFullAxis canonicalizes elem's subtree from a node set that
// carries the COMPLETE in-scope namespace axis on every element. It is the
// reference implementation the reduced, mode-aware node set
// (collectCanonicalizationNodes) must match byte for byte: the reduction is a
// performance change only, and a single differing byte would change a digest and
// break interoperability with signatures other implementations produced.
func canonicalizeSubtreeFullAxis(t *testing.T, method string, elem *helium.Element, prefixes []string) ([]byte, error) {
	t.Helper()
	mode, comments, err := resolveC14NMode(method)
	require.NoError(t, err)
	nodes, err := collectSubtreeNodes(t.Context(), elem)
	require.NoError(t, err)
	return canonicalizeNodeSetMode(mode, comments, nodes, elem.OwnerDocument(), prefixes)
}

// requireSameCanonicalBytes canonicalizes elem under every method and prefix
// list both ways and requires the results — bytes and errors alike — to agree.
// It reports how many canonicalizations it compared.
func requireSameCanonicalBytes(t *testing.T, label string, elem *helium.Element) int {
	t.Helper()
	for _, method := range diffMethods {
		for _, prefixes := range diffPrefixLists {
			want, wantErr := canonicalizeSubtreeFullAxis(t, method, elem, prefixes)
			got, gotErr := canonicalizeSubtree(t.Context(), method, elem, prefixes)
			if wantErr != nil || gotErr != nil {
				require.Equal(t, fmt.Sprint(wantErr), fmt.Sprint(gotErr),
					"%s: %s prefixes=%v: error mismatch", label, method, prefixes)
				continue
			}
			require.Equal(t, string(want), string(got),
				"%s: %s prefixes=%v: canonical bytes differ", label, method, prefixes)
		}
	}
	return len(diffMethods) * len(diffPrefixLists)
}

// diffTargets returns the elements of doc to canonicalize: the document element
// and every descendant, so each document exercises the node set with an apex at
// every depth.
func diffTargets(doc *helium.Document) []*helium.Element {
	root := doc.DocumentElement()
	if root == nil {
		return nil
	}
	targets := []*helium.Element{root}
	for i := 0; i < len(targets); i++ {
		for child := range helium.Children(targets[i]) {
			if elem, ok := helium.AsNode[*helium.Element](child); ok {
				targets = append(targets, elem)
			}
		}
	}
	return targets
}

// TestCanonicalizationNodeSetMatchesFullAxis is the acceptance test for the
// reduced canonicalization node set: over the package's own signature fixtures,
// the W3C interop vectors, the c14n suite's documents, and a seeded corpus of
// generated documents, the canonical octets must be identical to those the
// complete namespace axis produces — in every canonicalization method, with and
// without an Exclusive C14N PrefixList.
func TestCanonicalizationNodeSetMatchesFullAxis(t *testing.T) {
	t.Run("corpus documents", func(t *testing.T) {
		paths := corpusDocuments(t)
		require.Greater(t, len(paths), 50, "corpus is too small to be evidence")
		compared, comparisons := 0, 0
		for _, path := range paths {
			src, err := os.ReadFile(path)
			require.NoError(t, err)
			doc, err := helium.NewParser().Parse(t.Context(), src)
			if err != nil {
				// A few fixtures exist to be rejected by the parser; they carry
				// no node set to compare.
				continue
			}
			compared++
			for i, target := range diffTargets(doc) {
				comparisons += requireSameCanonicalBytes(t, fmt.Sprintf("%s[%d]", path, i), target)
			}
		}
		require.Greater(t, compared, 50, "too few corpus documents parsed")
		t.Logf("compared %d canonicalizations over %d corpus documents", comparisons, compared)
	})

	// The generated documents cover the shapes a fixture corpus does not reach on
	// purpose: a prefix rebound at several depths, a redundant redeclaration, a
	// default namespace changed and reset, prefixed attributes, and xml:* names
	// on an omitted ancestor.
	t.Run("generated documents", func(t *testing.T) {
		requireGeneratedCorpusMatches(t, randomNamespaceDoc, diffCorpusSize())
	})

	// An element declaring many prefixes at once is the shape the walk answers
	// membership for from an index rather than by scanning what it has recorded,
	// and the corpus above never declares more than three on one element. This
	// one straddles that count from both sides, with the same rebinding,
	// redundant redeclaration, and default-namespace churn.
	t.Run("generated documents with dense declarations", func(t *testing.T) {
		requireGeneratedCorpusMatches(t, denseNamespaceDoc, diffCorpusSize()/4)
	})

	// A name whose prefix nothing in scope declares cannot be parsed — it is
	// namespace-not-well-formed — but a programmatically built DOM can hold one,
	// and the in-scope axis then reports the element's own active namespace. The
	// reduced set must agree there too.
	t.Run("undeclared prefix on a built element", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root xmlns:a="urn:x:1"><a:kept/></root>`))
		require.NoError(t, err)
		root := doc.DocumentElement()

		child, err := doc.CreateElementNS("orphan", helium.NewNamespace("q", "urn:x:undeclared"))
		require.NoError(t, err)
		require.NoError(t, child.SetAttributeNS("at", "v", helium.NewNamespace("r", "urn:x:attr")))
		require.NoError(t, root.AddChild(child))

		requireSameCanonicalBytes(t, "undeclared prefix", root)
		requireSameCanonicalBytes(t, "undeclared prefix apex", child)
	})
}

// costDeclarations is how many namespace declarations the cost document puts on
// one element, and costPerDeclaration is what each of them may cost the walk.
//
// A declaration costs the collectors a fraction of a microsecond, a little over
// one under the race detector, and the document is parsed outside the
// measurement — so the budget is several times what the walk actually spends,
// while a per-element scan that grows with the declaration count spends tens of
// microseconds apiece at this size, and more at any larger one.
//
// costRuns repeats each measurement and keeps the cheapest, so a scheduler
// hiccup during one run cannot fail the case; a cost that is quadratic in the
// declaration count is quadratic in every run.
const (
	costDeclarations   = 20000
	costPerDeclaration = 6 * time.Microsecond
	costRuns           = 3
)

// declDenseDoc builds a document whose CHILD element carries decls namespace
// declarations.
//
// The declarations must NOT sit on the element the collection starts from: that
// element's scope is seeded from its in-scope axis in one map copy, and the
// per-element binding loop runs only BELOW it. A document that declares
// everything on the collection root exercises none of the per-element work this
// case bounds.
func declDenseDoc(decls int) string {
	var b strings.Builder
	b.WriteString(`<root xmlns:r="urn:example:r"><child`)
	for i := range decls {
		fmt.Fprintf(&b, ` xmlns:p%d="urn:example:ns:%d"`, i, i)
	}
	b.WriteString(`>text</child></root>`)
	return b.String()
}

// collectC14N10Nodes and collectFullAxisNodes give the two collectors the one
// signature the cost table below shares. Inclusive C14N 1.0 is the mode
// Verifier.Verify canonicalizes SignedInfo under.
func collectC14N10Nodes(ctx context.Context, elem *helium.Element) ([]helium.Node, error) {
	return collectCanonicalizationNodes(ctx, elem, c14n.C14N10)
}

func collectFullAxisNodes(ctx context.Context, elem *helium.Element) ([]helium.Node, error) {
	return collectSubtreeNodes(ctx, elem)
}

// TestNodeSetCollectionCost bounds what an element carrying many namespace
// declarations may cost the two node-set collectors. Both walk the subtree
// through the same per-element binding loop, so a membership test that scans
// what the element has already recorded makes both quadratic in the declaration
// count — and an attacker reaches both before any signature is checked, through
// the ds:SignedInfo canonicalization and through a ds:RetrievalMethod's
// transform pipeline.
//
// The bound is stated against the INPUT — one budget per declaration written —
// rather than against a second measurement of the same code, so it cannot be
// satisfied by a run that is merely as slow as another run.
func TestNodeSetCollectionCost(t *testing.T) {
	// no t.Parallel(): the case measures elapsed time, which a concurrent test
	// competing for the same cores would inflate.

	doc, err := helium.NewParser().Parse(t.Context(), []byte(declDenseDoc(costDeclarations)))
	require.NoError(t, err)
	root := doc.DocumentElement()
	require.NotNil(t, root)

	budget := time.Duration(costDeclarations) * costPerDeclaration

	for _, tc := range []struct {
		name    string
		collect func(context.Context, *helium.Element) ([]helium.Node, error)
	}{
		{name: "canonicalization node set", collect: collectC14N10Nodes},
		{name: "full-axis node set", collect: collectFullAxisNodes},
	} {
		t.Run(tc.name, func(t *testing.T) {
			best := time.Duration(-1)
			for range costRuns {
				start := time.Now()
				nodes, err := tc.collect(t.Context(), root)
				elapsed := time.Since(start)
				require.NoError(t, err)
				// Every declaration is in scope on the child, so a set this
				// small would mean the walk never did the work being bounded.
				require.GreaterOrEqual(t, len(nodes), costDeclarations,
					"collector returned %d nodes for %d declarations", len(nodes), costDeclarations)
				if best < 0 || elapsed < best {
					best = elapsed
				}
			}
			require.Less(t, best, budget,
				"collecting a node set over %d declarations took %v, over the %v budget",
				costDeclarations, best, budget)
		})
	}
}

// diffCorpusSeed fixes the generated corpus so a failure is reproducible: the
// same seed always produces the same documents, and the counter-example is
// printed with the failure.
const diffCorpusSeed = 20260813

// defaultDiffCorpusSize keeps the committed run to a few seconds. Set
// HELIUM_XMLDSIG_DIFF_DOCS to sweep a larger corpus.
const defaultDiffCorpusSize = 400

func diffCorpusSize() int {
	if v := os.Getenv("HELIUM_XMLDSIG_DIFF_DOCS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultDiffCorpusSize
}

// corpusDocuments lists every XML document in the package's own test data (the
// W3C XMLDSig interop vectors included) and in the c14n suite's data.
func corpusDocuments(t *testing.T) []string {
	t.Helper()
	var paths []string
	for _, root := range []string{"testdata", filepath.Join("..", "testdata", "libxml2-compat", "c14n")} {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(path, ".xml") {
				paths = append(paths, path)
			}
			return nil
		})
		require.NoError(t, err)
	}
	return paths
}

// requireGeneratedCorpusMatches draws size documents from build, seeded so a
// failure is reproducible, and requires the reduced node set to match the full
// axis at every apex of each.
func requireGeneratedCorpusMatches(t *testing.T, build func(*rand.Rand) string, size int) {
	t.Helper()
	rng := rand.New(rand.NewSource(diffCorpusSeed))
	generated, comparisons := 0, 0
	for i := range size {
		src := build(rng)
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		if err != nil {
			continue
		}
		generated++
		for j, target := range diffTargets(doc) {
			comparisons += requireSameCanonicalBytes(t, fmt.Sprintf("generated#%d[%d] %s", i, j, src), target)
		}
	}
	require.Greater(t, generated, size*3/4, "too few generated documents parsed")
	t.Logf("compared %d canonicalizations over %d generated documents", comparisons, generated)
}

// randomNamespaceDoc builds one namespace-dense document. Prefixes are drawn
// from a small pool and URIs from an even smaller one, so rebinding, redundant
// redeclaration, and reuse across siblings all occur often.
func randomNamespaceDoc(rng *rand.Rand) string {
	g := &docGenerator{
		rng:      rng,
		prefixes: []string{"a", "b", "c", "d"},
		uris:     []string{"urn:x:1", "urn:x:2", "urn:x:3"},
		maxDecls: 3,
		budget:   6 + rng.Intn(25),
	}
	var b strings.Builder
	g.writeElement(&b, 0, nil)
	return b.String()
}

// densePrefixes is a pool wide enough that one element can declare more prefixes
// than the walk indexes membership for, and the URIs stay as few as above so a
// wide element still rebinds and redundantly redeclares.
var densePrefixes = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n"}

// denseNamespaceDoc builds one document whose elements declare up to
// len(densePrefixes) prefixes each, so the corpus covers elements on both sides
// of the count at which the walk switches to an index. The element budget is
// smaller than randomNamespaceDoc's because each element is far wider.
func denseNamespaceDoc(rng *rand.Rand) string {
	g := &docGenerator{
		rng:      rng,
		prefixes: densePrefixes,
		uris:     []string{"urn:x:1", "urn:x:2", "urn:x:3"},
		maxDecls: len(densePrefixes),
		budget:   4 + rng.Intn(10),
	}
	var b strings.Builder
	g.writeElement(&b, 0, nil)
	return b.String()
}

type docGenerator struct {
	rng      *rand.Rand
	prefixes []string
	uris     []string
	maxDecls int
	budget   int
}

// writeElement emits one element and its descendants. inScope is the prefix list
// an ancestor has declared, so a child usually names a declared prefix and
// occasionally names one nothing declares.
func (g *docGenerator) writeElement(b *strings.Builder, depth int, inScope []string) {
	g.budget--

	var decls []string
	var declared []string
	for n := g.rng.Intn(g.maxDecls + 1); n > 0; n-- {
		prefix := g.prefixes[g.rng.Intn(len(g.prefixes))]
		if slices.Contains(declared, prefix) {
			// One element may bind a prefix once; a second xmlns:p attribute is
			// a duplicate attribute, not a rebinding.
			continue
		}
		declared = append(declared, prefix)
		uri := g.uris[g.rng.Intn(len(g.uris))]
		decls = append(decls, fmt.Sprintf(` xmlns:%s="%s"`, prefix, uri))
		inScope = append(inScope, prefix)
	}
	switch g.rng.Intn(4) {
	case 0:
		decls = append(decls, fmt.Sprintf(` xmlns="%s"`, g.uris[g.rng.Intn(len(g.uris))]))
	case 1:
		// A default-namespace reset, which C14N must undeclare rather than leak.
		decls = append(decls, ` xmlns=""`)
	}

	name := "e"
	if len(inScope) > 0 && g.rng.Intn(3) > 0 {
		name = inScope[g.rng.Intn(len(inScope))] + ":e"
	}

	var attrs []string
	for n := g.rng.Intn(3); n > 0; n-- {
		if len(inScope) > 0 && g.rng.Intn(2) == 0 {
			attrs = append(attrs, fmt.Sprintf(` %s:at%d="v"`, inScope[g.rng.Intn(len(inScope))], n))
			continue
		}
		attrs = append(attrs, fmt.Sprintf(` at%d="v"`, n))
	}
	if g.rng.Intn(6) == 0 {
		attrs = append(attrs, ` xml:lang="en"`)
	}
	if g.rng.Intn(8) == 0 {
		attrs = append(attrs, ` xml:base="urn:base"`)
	}

	fmt.Fprintf(b, "<%s%s%s>", name, strings.Join(decls, ""), strings.Join(attrs, ""))

	if depth < 4 {
		for n := g.rng.Intn(4); n > 0 && g.budget > 0; n-- {
			switch g.rng.Intn(8) {
			case 0:
				b.WriteString("text")
			case 1:
				b.WriteString("<!--c-->")
			case 2:
				b.WriteString("<?pi data?>")
			default:
				g.writeElement(b, depth+1, inScope)
			}
		}
	}

	fmt.Fprintf(b, "</%s>", name)
}
