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
	// membership for from an index, and never by scanning what it has recorded,
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

// denseDeclarations is how many namespace declarations declDenseDoc puts on one
// element: far more than a poll interval, so binding them spans many polls.
const denseDeclarations = 20000

// declDenseDoc builds a document whose CHILD element carries decls namespace
// declarations.
//
// The declarations must NOT sit on the element the collection starts from: that
// element's scope is seeded from its in-scope axis in one map copy, and the
// per-element binding loop runs only BELOW it. A document that declares
// everything on the collection root exercises none of the per-element work.
func declDenseDoc(decls int) string {
	var b strings.Builder
	b.WriteString(`<root xmlns:r="urn:example:r"><child`)
	for i := range decls {
		fmt.Fprintf(&b, ` xmlns:p%d="urn:example:ns:%d"`, i, i)
	}
	b.WriteString(`>text</child></root>`)
	return b.String()
}

// parseRoot parses src and returns the element the collection starts from.
func parseRoot(t *testing.T, src string) *helium.Element {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)
	root := doc.DocumentElement()
	require.NotNil(t, root)
	return root
}

// TestPrefixSetIndexesLargeSets pins what keeps the collectors linear in the
// number of prefixes one element carries. Every per-element membership question
// the walk asks goes through a prefixSet: whether enter has already recorded a
// prefix, and whether the emission has already emitted one. A set that kept
// scanning its list would make an element with thousands of declarations or
// prefixed attributes cost the square of that count, and an attacker reaches
// both collectors before any signature is checked, through the ds:SignedInfo
// canonicalization and through a ds:RetrievalMethod's transform pipeline.
//
// The collectors take the set only as a *prefixSet (bind, emitNamespace), so the
// property is checked on the set's own state: from prefixIndexThreshold members
// on, the index holds every member and answers every lookup.
func TestPrefixSetIndexesLargeSets(t *testing.T) {
	sizes := []int{0, 1, prefixIndexThreshold - 1, prefixIndexThreshold, prefixIndexThreshold + 1, 4000}
	for _, size := range sizes {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			var s prefixSet
			for i := range size {
				prefix := "p" + strconv.Itoa(i)
				require.False(t, s.contains(prefix), "set reports %q before it was added", prefix)
				s.add(prefix)
			}

			if size < prefixIndexThreshold {
				require.Nil(t, s.index, "a set of %d prefixes built an index", size)
			} else {
				require.Len(t, s.index, size, "a set of %d prefixes does not index all of them", size)
			}
			for i := range size {
				require.True(t, s.contains("p"+strconv.Itoa(i)))
			}
			require.False(t, s.contains("absent"))
			require.False(t, s.contains(""))
		})
	}
}

// cancelOnErrContext cancels itself on a selected Err call. That makes a
// cancellation at a collector poll deterministic without coupling the test to
// scheduler timing.
type cancelOnErrContext struct {
	done     chan struct{}
	cancelAt int
	calls    int
}

func (*cancelOnErrContext) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (c *cancelOnErrContext) Done() <-chan struct{} {
	return c.done
}

func (c *cancelOnErrContext) Err() error {
	c.calls++
	if c.calls == c.cancelAt {
		close(c.done)
	}
	select {
	case <-c.done:
		return context.Canceled
	default:
		return nil
	}
}

func (*cancelOnErrContext) Value(any) any {
	return nil
}

// TestNodeSetCollectionCancellationDuringChildScope pins that namespace scope
// entry polls while it binds a child's declarations. The first Err call starts
// collection; the second comes from the periodic poll inside the dense child.
func TestNodeSetCollectionCancellationDuringChildScope(t *testing.T) {
	root := parseRoot(t, declDenseDoc(denseDeclarations))
	ctx := &cancelOnErrContext{done: make(chan struct{}), cancelAt: 2}
	c := &subtreeCollector{fullAxis: true}

	err := c.collect(ctx, root)
	require.ErrorIs(t, err, context.Canceled)
	require.LessOrEqual(t, len(c.scope), ctxPollInterval,
		"collector bound %d child declarations before observing cancellation", len(c.scope))
}

// pollAxisDeclarations is how many bindings axisDenseDoc declares on the
// collection root for the poll case, and pollAxisChildren how many children
// repeat that whole axis below it. One element's axis is twice a poll interval,
// so a poll that counted only walked TREE nodes would let a whole axis pass
// between two polls.
const (
	pollAxisDeclarations = 2 * ctxPollInterval
	pollAxisChildren     = 8
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

// pollGapContext records, at every Err call, how many members the collector
// gained since the previous call, and how many it held at the call that cancels
// the context. It reads the collector's own member count, so what a case using
// it checks is the work done between two polls and not the time that work took.
type pollGapContext struct {
	cancelOnErrContext
	c        *subtreeCollector
	last     int
	maxGap   int
	atCancel int
}

func (p *pollGapContext) Err() error {
	n := len(p.c.nodes)
	p.maxGap = max(p.maxGap, n-p.last)
	p.last = n
	err := p.cancelOnErrContext.Err()
	if p.calls == p.cancelAt {
		p.atCancel = n
	}
	return err
}

// TestNodeSetCollectionPollsBoundedWork requires the full-axis collector to poll
// its context at least once every ctxPollInterval members, and to stop at the
// first poll that reports the context done. The full axis is the node set an
// XPath filter transform is evaluated over, and it is quadratic in the document
// by the transform's own data model, so a deadline, not a size bound, is what
// keeps it from running to completion on an attacker's document. A deadline is
// only as prompt as the longest span of work between two polls.
func TestNodeSetCollectionPollsBoundedWork(t *testing.T) {
	root := parseRoot(t, axisDenseDoc(pollAxisDeclarations, pollAxisChildren))
	members := pollAxisDeclarations * (pollAxisChildren + 1)

	t.Run("polls every interval", func(t *testing.T) {
		c := &subtreeCollector{fullAxis: true}
		ctx := &pollGapContext{cancelOnErrContext: cancelOnErrContext{done: make(chan struct{})}, c: c}

		require.NoError(t, c.collect(ctx, root))
		require.GreaterOrEqual(t, len(c.nodes), members,
			"collector returned %d members, fewer than the %d namespace nodes the document puts in scope", len(c.nodes), members)
		// The span after the last poll counts too: a walk that stops polling part
		// way through never calls Err again to report it.
		maxGap := max(ctx.maxGap, len(c.nodes)-ctx.last)
		require.LessOrEqual(t, maxGap, ctxPollInterval,
			"collector added %d members between two polls, over the %d-member poll interval", maxGap, ctxPollInterval)
	})

	t.Run("stops at the poll that sees cancellation", func(t *testing.T) {
		c := &subtreeCollector{fullAxis: true}
		// Cancel about half way through, so the walk is deep inside the axis.
		cancelAt := members / ctxPollInterval / 2
		ctx := &pollGapContext{cancelOnErrContext: cancelOnErrContext{done: make(chan struct{}), cancelAt: cancelAt}, c: c}

		err := c.collect(ctx, root)
		require.ErrorIs(t, err, context.Canceled)
		require.Less(t, len(c.nodes), members, "collector ran to completion on a cancelled context")
		require.Equal(t, ctx.atCancel, len(c.nodes),
			"collector held %d members when its context reported cancellation and %d when it returned",
			ctx.atCancel, len(c.nodes))
	})
}

// chargedNodeCount is how many members each cancellation case below collects or
// filters. It is several poll intervals' worth, so a walk that charges the work
// it does reaches a poll well inside the set, and one that charges nothing never
// polls at all however large the set grows.
const chargedNodeCount = 4 * ctxPollInterval

// topLevelCommentDoc builds a document whose children are count comments and
// nothing else. A whole-document collection admits every one of them as a
// member, so appending them is the only work it does — which is what makes this
// document decide whether those appends are charged.
//
// A parsed document carries a document element too, and reaches the same appends
// around it. It cannot stand in here: collecting that element's subtree polls the
// context at its own entry, and that entry check would report a cancellation no
// matter what the sibling appends do.
func topLevelCommentDoc(t *testing.T, count int) *helium.Document {
	t.Helper()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneExplicitNo)
	for i := range count {
		require.NoError(t, doc.AddChild(doc.CreateComment(fmt.Appendf(nil, "c%d", i))))
	}
	return doc
}

// cancelledContext returns a context that is already cancelled, so what a case
// using it measures is whether the work it hands the context to looks.
func cancelledContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	return ctx
}

// documentChildNodes returns doc's children as the node slice the node-set
// filters take.
func documentChildNodes(doc *helium.Document) []helium.Node {
	var nodes []helium.Node
	for child := range helium.Children(doc) {
		nodes = append(nodes, child)
	}
	return nodes
}

// signatureSubtreeNodes returns a node set drawn entirely from inside a
// Signature element, together with that element. removeSignatureNodes drops
// every member of such a set, so it keeps nothing at all — the shape that tells
// a filter charging only what it keeps from one charging the nodes it reads.
func signatureSubtreeNodes(t *testing.T, count int) ([]helium.Node, *helium.Element) {
	t.Helper()
	src := `<root><sig>` + strings.Repeat(`<e a="1"/>`, count) + `</sig></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)
	root := doc.DocumentElement()
	require.NotNil(t, root)
	sig, ok := helium.AsNode[*helium.Element](root.FirstChild())
	require.True(t, ok)
	nodes, err := collectSubtreeNodes(t.Context(), sig)
	require.NoError(t, err)
	require.Greater(t, len(nodes), chargedNodeCount)
	return nodes, sig
}

// TestNodeSetGrowthHonorsCancellation requires every node-set the transform
// pipeline builds or narrows to stop on a cancelled context. Each of these sets
// is reached before any signature is checked — a ds:RetrievalMethod names any
// element in the document and runs its own transforms — and each is bounded by a
// deadline, with no size cap behind it, so a stage that reads a whole set without
// polling spends the document's cost after the deadline has passed.
func TestNodeSetGrowthHonorsCancellation(t *testing.T) {
	// The whole-document node set a URI="" or "#xpointer(/)" reference stands for.
	t.Run("whole-document collection", func(t *testing.T) {
		doc := topLevelCommentDoc(t, chargedNodeCount)
		_, err := collectDocumentNodes(cancelledContext(t), doc)
		require.ErrorIs(t, err, context.Canceled)
	})

	// The same set built for a canonicalization consumer, which reads a reduced
	// namespace membership but the identical document children.
	t.Run("whole-document canonicalization collection", func(t *testing.T) {
		doc := topLevelCommentDoc(t, chargedNodeCount)
		_, err := collectCanonicalizationDocumentNodes(cancelledContext(t), doc, c14n.C14N10)
		require.ErrorIs(t, err, context.Canceled)
	})

	// A comment-excluding Reference form drops every comment from the
	// materialized set.
	t.Run("comment removal", func(t *testing.T) {
		nodes := documentChildNodes(topLevelCommentDoc(t, chargedNodeCount))
		_, err := removeCommentNodes(cancelledContext(t), nodes)
		require.ErrorIs(t, err, context.Canceled)
	})

	// The enveloped-signature transform on an explicit node set, which tests each
	// member against the Signature's subtree.
	t.Run("signature-node removal", func(t *testing.T) {
		nodes, sig := signatureSubtreeNodes(t, chargedNodeCount)
		_, err := removeSignatureNodes(cancelledContext(t), nodes, sig)
		require.ErrorIs(t, err, context.Canceled)
	})
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
		// A default-namespace reset, which C14N must undeclare, and must never leak.
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
