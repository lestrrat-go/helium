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

// costDeclarations is how many namespace declarations each cost document
// carries, and costConcentration is how many elements the reference document
// spreads them over — the document under test puts every one of them on a single
// element.
//
// The two documents give the collectors the same total work: the same
// declarations bound, the same number of namespace nodes emitted, the same node
// set built. They differ only in how many declarations sit on ONE element, which
// is the count a membership test that scans what the element has already
// recorded costs the square of. So the bound is a RATIO between them, and no
// wall-clock constant enters it: a machine that is
// uniformly slow, or a race-detector build, or a loaded runner scales both sides
// alike and cancels out. An absolute budget cannot express that — it conflates
// the growth this case exists to pin with the speed of whatever machine runs it.
//
// costMaxRatio sits between the two regimes. A collector linear in the
// per-element count measures 1.9 to 2.4 here — the concentrated document's
// running scope holds every prefix at once, so its map costs more per lookup —
// and that holds under the race detector with the machine loaded. Put the
// scan back and the same measurement reads 21 to 28, since concentrating the
// declarations squares a cost the reference document pays costConcentration
// times over a costConcentration-th of the count each. Seven is about the
// geometric middle, roughly three times either regime.
//
// costRuns repeats each measurement and keeps the cheapest, so a scheduler
// hiccup during one run cannot inflate the ratio; a cost that is quadratic in
// the per-element declaration count is quadratic in every run.
const (
	costDeclarations  = 20000
	costConcentration = 32
	costMaxRatio      = 7
	costRuns          = 5
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

// declSpreadDoc builds the reference document for declDenseDoc: elements
// children carrying decls declarations each, no prefix shared between them, so
// the walk binds elements*decls prefixes and emits a namespace node for every
// one of them exactly as the dense document does. Only the per-element count
// differs.
func declSpreadDoc(elements, decls int) string {
	var b strings.Builder
	b.WriteString(`<root xmlns:r="urn:example:r">`)
	for e := range elements {
		fmt.Fprintf(&b, `<child%d`, e)
		for i := range decls {
			fmt.Fprintf(&b, ` xmlns:p%d_%d="urn:example:ns:%d:%d"`, e, i, e, i)
		}
		fmt.Fprintf(&b, `>text</child%d>`, e)
	}
	b.WriteString(`</root>`)
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

// costDocument parses src and returns the element the collection starts from.
// Parsing is deliberately outside every measurement: it is the cost of building
// the DOM, not of walking it.
func costDocument(t *testing.T, src string) *helium.Element {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)
	root := doc.DocumentElement()
	require.NotNil(t, root)
	return root
}

// bestCollectionCost collects the node set under root costRuns times and returns
// the cheapest run, requiring every run to produce the whole node set.
func bestCollectionCost(t *testing.T, collect func(context.Context, *helium.Element) ([]helium.Node, error), root *helium.Element) time.Duration {
	t.Helper()
	best := time.Duration(-1)
	for range costRuns {
		start := time.Now()
		nodes, err := collect(t.Context(), root)
		elapsed := time.Since(start)
		require.NoError(t, err)
		// Every declaration is in scope on the element that carries it, so a
		// set this small would mean the walk never did the work being compared.
		require.GreaterOrEqual(t, len(nodes), costDeclarations,
			"collector returned %d nodes for %d declarations", len(nodes), costDeclarations)
		if best < 0 || elapsed < best {
			best = elapsed
		}
	}
	return best
}

// TestNodeSetCollectionCost bounds what an element carrying many namespace
// declarations may cost the two node-set collectors. Both walk the subtree
// through the same per-element binding loop, so a membership test that scans
// what the element has already recorded makes both quadratic in the declaration
// count — and an attacker reaches both before any signature is checked, through
// the ds:SignedInfo canonicalization and through a ds:RetrievalMethod's
// transform pipeline.
//
// The bound is stated against a document that does the SAME total work with the
// declarations spread over many elements, so what it measures is the growth in
// the per-element count alone and not the speed of the machine. Comparing the
// concentrated document against a second measurement of ITSELF — at half the
// size, say — would prove nothing: the quadratic scan measures the same 2.6x
// there that the linear walk does, because both grow by the same factor when the
// whole document grows.
func TestNodeSetCollectionCost(t *testing.T) {
	// no t.Parallel(): the case measures elapsed time, which a concurrent test
	// competing for the same cores would inflate.

	concentrated := costDocument(t, declDenseDoc(costDeclarations))
	spread := costDocument(t, declSpreadDoc(costConcentration, costDeclarations/costConcentration))

	for _, tc := range []struct {
		name    string
		collect func(context.Context, *helium.Element) ([]helium.Node, error)
	}{
		{name: "canonicalization node set", collect: collectC14N10Nodes},
		{name: "full-axis node set", collect: collectFullAxisNodes},
	} {
		t.Run(tc.name, func(t *testing.T) {
			onOneElement := bestCollectionCost(t, tc.collect, concentrated)
			overManyElements := bestCollectionCost(t, tc.collect, spread)

			require.Less(t, onOneElement, costMaxRatio*overManyElements,
				"collecting a node set cost %v with %d declarations on one element and %v with the same %d spread over %d, a factor of %.1f: the per-element cost grows with the declaration count",
				onOneElement, costDeclarations, overManyElements, costDeclarations, costConcentration,
				float64(onOneElement)/float64(overManyElements))
		})
	}
}

// The deadline case walks a document whose every element repeats the whole
// in-scope namespace axis: deadlineDeclarations bindings on the collection root,
// and deadlineChildren children below it, so the full axis holds over five
// million node-set members. That is the shape a ds:RetrievalMethod's XPath
// filter transform reaches before any signature is checked, and the one where a
// poll that counted only walked TREE nodes let a quarter of those members —
// ctxPollInterval elements, one whole axis each — pass inside a single unpolled
// span.
//
// deadlineWindow is long enough that the deadline cannot expire before the
// collection starts, where the entry check would catch it and the walk itself
// would never be exercised.
//
// deadlineOverrunBudget is what the collector may still spend after the deadline
// passes: one poll interval of members at ten microseconds each, hundreds of
// times what a member costs, plus ten milliseconds for the timer granularity and
// scheduler noise that dominate at this scale. It is a constant, so it does not
// grow with the document — a collector that goes on collecting a share of the
// axis fails it here and by a wider margin at any larger size.
//
// deadlineRuns repeats the measurement and keeps the fastest, so one descheduled
// run cannot fail the case.
const (
	deadlineDeclarations  = 5000
	deadlineChildren      = 1024
	deadlineWindow        = 20 * time.Millisecond
	deadlineOverrunBudget = ctxPollInterval*10*time.Microsecond + 10*time.Millisecond
	deadlineRuns          = 3
)

// axisDenseDoc builds a document declaring decls prefixes on its root and
// carrying children element children below it. Every child inherits the whole
// axis, so the full-axis node set is decls per element.
func axisDenseDoc(decls, children int) string {
	var b strings.Builder
	b.WriteString(`<root`)
	for i := range decls {
		fmt.Fprintf(&b, ` xmlns:p%d="urn:example:ns:%d"`, i, i)
	}
	b.WriteString(`>`)
	for i := range children {
		fmt.Fprintf(&b, `<c%d>t</c%d>`, i, i)
	}
	b.WriteString(`</root>`)
	return b.String()
}

// TestNodeSetCollectionHonorsDeadline requires the full-axis collector to stop
// within a bounded time of its context's deadline. The full axis is the node set
// an XPath filter transform is evaluated over, and it is quadratic in the
// document by the transform's own data model, so a deadline — not a size bound —
// is what keeps it from running to completion on an attacker's document.
//
// The bound is stated against the poll interval and the document's size, not
// against a second measurement of the same code, so a run that merely matches
// another slow run cannot satisfy it.
func TestNodeSetCollectionHonorsDeadline(t *testing.T) {
	// no t.Parallel(): the case measures elapsed time, which a concurrent test
	// competing for the same cores would inflate.

	doc, err := helium.NewParser().Parse(t.Context(), []byte(axisDenseDoc(deadlineDeclarations, deadlineChildren)))
	require.NoError(t, err)
	root := doc.DocumentElement()
	require.NotNil(t, root)

	best := time.Duration(-1)
	for range deadlineRuns {
		// The context is created here, immediately before the call, so the whole
		// window is spent inside the collection.
		ctx, cancel := context.WithTimeout(t.Context(), deadlineWindow)
		start := time.Now()
		_, err := collectSubtreeNodes(ctx, root)
		elapsed := time.Since(start)
		cancel()
		require.ErrorIs(t, err, context.DeadlineExceeded,
			"a full-axis collection over %d members ran to completion inside a %v deadline",
			deadlineDeclarations*(deadlineChildren+1), deadlineWindow)
		if best < 0 || elapsed < best {
			best = elapsed
		}
	}

	require.Less(t, best, deadlineWindow+deadlineOverrunBudget,
		"the collector kept working for %v after a %v deadline, over the %v it may overrun by",
		best-deadlineWindow, deadlineWindow, deadlineOverrunBudget)
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
