package xmldsig1

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
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
		rng := rand.New(rand.NewSource(diffCorpusSeed))
		generated, comparisons := 0, 0
		size := diffCorpusSize()
		for i := range size {
			src := randomNamespaceDoc(rng)
			doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
			if err != nil {
				continue
			}
			generated++
			for j, target := range diffTargets(doc) {
				comparisons += requireSameCanonicalBytes(t, fmt.Sprintf("generated#%d[%d] %s", i, j, src), target)
			}
		}
		require.Greater(t, generated, diffCorpusSize()*3/4, "too few generated documents parsed")
		t.Logf("compared %d canonicalizations over %d generated documents", comparisons, generated)
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

// randomNamespaceDoc builds one namespace-dense document. Prefixes are drawn
// from a small pool and URIs from an even smaller one, so rebinding, redundant
// redeclaration, and reuse across siblings all occur often.
func randomNamespaceDoc(rng *rand.Rand) string {
	g := &docGenerator{
		rng:      rng,
		prefixes: []string{"a", "b", "c", "d"},
		uris:     []string{"urn:x:1", "urn:x:2", "urn:x:3"},
		budget:   6 + rng.Intn(25),
	}
	var b strings.Builder
	g.writeElement(&b, 0, nil)
	return b.String()
}

type docGenerator struct {
	rng      *rand.Rand
	prefixes []string
	uris     []string
	budget   int
}

// writeElement emits one element and its descendants. inScope is the prefix list
// an ancestor has declared, so a child usually names a declared prefix and
// occasionally names one nothing declares.
func (g *docGenerator) writeElement(b *strings.Builder, depth int, inScope []string) {
	g.budget--

	var decls []string
	var declared []string
	for n := g.rng.Intn(4); n > 0; n-- {
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
