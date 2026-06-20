package c14n_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/c14n"
	"github.com/lestrrat-go/helium/xpath1"
	"github.com/stretchr/testify/require"
)

const (
	testdataBase       = "../testdata/libxml2-compat/c14n"
	parseFailDupNSDecl = "duplicate namespace declaration handling"
)

// knownParseFailures lists test cases that fail during parsing due to
// helium parser limitations (not C14N bugs).
var knownParseFailures = map[string]string{
	"without-comments/example-3":     parseFailDupNSDecl,
	"without-comments/example-4":     "entity reference in single-quoted attribute",
	"without-comments/test-2":        parseFailDupNSDecl,
	"without-comments/test-3":        parseFailDupNSDecl,
	"with-comments/example-3":        parseFailDupNSDecl,
	"with-comments/example-4":        "entity reference in single-quoted attribute",
	"exc-without-comments/test-0":    parseFailDupNSDecl,
	"exc-without-comments/test-1":    parseFailDupNSDecl,
	"1-1-without-comments/example-3": parseFailDupNSDecl,
	"1-1-without-comments/example-4": "entity reference in single-quoted attribute",
}

func parseTestDoc(t *testing.T, path string) *helium.Document {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err, "reading test file %s", path)

	p := helium.NewParser().SubstituteEntities(true).LoadExternalDTD(true).DefaultDTDAttributes(true).BaseURI(path)

	doc, err := p.Parse(t.Context(), data)
	require.NoError(t, err, "parsing test file %s", path)
	return doc
}

func readExpected(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "reading expected file %s", path)
	return data
}

// parseXPathFile parses a .xpath file and returns the XPath expression
// string and namespace bindings.
func parseXPathFile(t *testing.T, path string) (string, map[string]string) {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err, "reading xpath file %s", path)

	// Parse the .xpath file as XML
	p := helium.NewParser().SubstituteEntities(true).LoadExternalDTD(true).DefaultDTDAttributes(true)
	doc, err := p.Parse(t.Context(), data)
	require.NoError(t, err, "parsing xpath file %s", path)

	// Find the XPath element
	var xpathElem *helium.Element
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() == helium.ElementNode {
			xpathElem = child.(*helium.Element)
			break
		}
	}
	require.NotNil(t, xpathElem, "no root element in xpath file %s", path)

	// Extract namespace bindings from the XPath element
	nss := make(map[string]string)
	for _, ns := range xpathElem.Namespaces() {
		if ns.Prefix() != "" && ns.URI() != "" {
			nss[ns.Prefix()] = ns.URI()
		}
	}

	// Get the text content (the XPath expression).
	// Only collect text nodes, skip comment nodes.
	var sb strings.Builder
	for child := xpathElem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() == helium.TextNode {
			sb.Write(child.Content())
		}
	}
	expr := strings.TrimSpace(sb.String())

	return expr, nss
}

// parseNSFile parses a .ns file and returns the list of inclusive namespace prefixes.
func parseNSFile(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "reading ns file %s", path)
	fields := strings.Fields(string(data))
	var prefixes []string
	for _, f := range fields {
		if f == "#default" {
			prefixes = append(prefixes, "")
		} else {
			prefixes = append(prefixes, f)
		}
	}
	return prefixes
}

// evaluateNodeSet evaluates an XPath expression on a document and returns the resulting node set.
func evaluateNodeSet(t *testing.T, doc *helium.Document, expr string, nss map[string]string) []helium.Node {
	t.Helper()

	compiled, err := xpath1.Compile(expr)
	require.NoError(t, err, "compiling xpath: %s", expr)

	ev := xpath1.NewEvaluator().Namespaces(nss)
	result, err := ev.Evaluate(t.Context(), compiled, doc)
	require.NoError(t, err, "evaluating xpath: %s", expr)
	require.Equal(t, xpath1.NodeSetResult, result.Type, "xpath result is not a node set")
	return result.NodeSet
}

func runC14NTest(t *testing.T, category, name string, mode c14n.Mode, can c14n.Canonicalizer) {
	t.Helper()

	key := category + "/" + name
	if reason, ok := knownParseFailures[key]; ok {
		t.Skipf("skipping due to parser limitation: %s", reason)
	}

	inputPath := filepath.Join(testdataBase, category, "test", name+".xml")
	expectedPath := filepath.Join(testdataBase, category, "result", name)

	doc := parseTestDoc(t, inputPath)
	expected := readExpected(t, expectedPath)

	// Check for .xpath file
	xpathPath := filepath.Join(testdataBase, category, "test", name+".xpath")
	if _, err := os.Stat(xpathPath); err == nil {
		expr, nss := parseXPathFile(t, xpathPath)
		nodes := evaluateNodeSet(t, doc, expr, nss)
		can = can.NodeSet(nodes)
	}

	// Check for .ns file (inclusive namespace prefixes for exclusive C14N)
	nsPath := filepath.Join(testdataBase, category, "test", name+".ns")
	if _, err := os.Stat(nsPath); err == nil {
		prefixes := parseNSFile(t, nsPath)
		can = can.InclusiveNamespaces(prefixes)
	}

	// Pass base URI for C14N 1.1 xml:base fixup
	if mode == c14n.C14N11 {
		can = can.BaseURI(inputPath)
	}

	got, err := can.CanonicalizeTo(doc)
	require.NoError(t, err)
	require.Equal(t, string(expected), string(got))
}

func TestC14N10WithoutComments(t *testing.T) {
	t.Parallel()
	names := []string{
		"example-1",
		"example-2",
		"example-3",
		"example-4",
		"example-5",
		"example-6",
		"example-7",
		"test-0",
		"test-1",
		"test-2",
		"test-3",
		"merlin-c14n-two-00",
		"merlin-c14n-two-01",
		"merlin-c14n-two-02",
		"merlin-c14n-two-03",
		"merlin-c14n-two-04",
		"merlin-c14n-two-05",
		"merlin-c14n-two-06",
		"merlin-c14n-two-07",
		"merlin-c14n-two-08",
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			runC14NTest(t, "without-comments", name, c14n.C14N10, c14n.NewCanonicalizer(c14n.C14N10))
		})
	}
}

func TestExclusiveC14N10WithoutComments(t *testing.T) {
	t.Parallel()
	names := []string{
		"test-0",
		"test-1",
		"test-2",
		"merlin-c14n-two-09",
		"merlin-c14n-two-10",
		"merlin-c14n-two-11",
		"merlin-c14n-two-12",
		"merlin-c14n-two-13",
		"merlin-c14n-two-14",
		"merlin-c14n-two-17",
		"merlin-c14n-two-18",
		"merlin-c14n-two-19",
		"merlin-c14n-two-20",
		"merlin-c14n-two-21",
		"merlin-c14n-two-22",
		"merlin-c14n-two-23",
		"merlin-c14n-two-24",
		"merlin-c14n-two-26",
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			runC14NTest(t, "exc-without-comments", name, c14n.ExclusiveC14N10, c14n.NewCanonicalizer(c14n.ExclusiveC14N10))
		})
	}
}

func TestC14N10WithComments(t *testing.T) {
	t.Parallel()
	names := []string{
		"example-1",
		"example-2",
		"example-3",
		"example-4",
		"example-5",
		"example-6",
		"example-7",
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			runC14NTest(t, "with-comments", name, c14n.C14N10, c14n.NewCanonicalizer(c14n.C14N10).Comments())
		})
	}
}

func TestC14N11WithoutComments(t *testing.T) {
	t.Parallel()
	names := []string{
		"example-1",
		"example-2",
		"example-3",
		"example-4",
		"example-5",
		"example-6",
		"example-7",
		"example-8",
		"xmllang-prop-1",
		"xmllang-prop-2",
		"xmllang-prop-3",
		"xmllang-prop-4",
		"xmlspace-prop-1",
		"xmlspace-prop-2",
		"xmlspace-prop-3",
		"xmlspace-prop-4",
		"xmlid-prop-1",
		"xmlid-prop-2",
		"xmlbase-prop-1",
		"xmlbase-prop-2",
		"xmlbase-prop-3",
		"xmlbase-prop-4",
		"xmlbase-prop-5",
		"xmlbase-prop-6",
		"xmlbase-prop-7",
		"xmlbase-c14n11spec-102",
		"xmlbase-c14n11spec2-102",
		"xmlbase-c14n11spec3-102",
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			runC14NTest(t, "1-1-without-comments", name, c14n.C14N11, c14n.NewCanonicalizer(c14n.C14N11))
		})
	}
}

func TestZeroValueCanonicalizer(t *testing.T) {
	t.Parallel()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root b="2" a="1"><child/></root>`))
	require.NoError(t, err)

	// Zero-value Canonicalizer defaults to C14N10 mode (iota = 0).
	var can c14n.Canonicalizer
	got, err := can.CanonicalizeTo(doc)
	require.NoError(t, err)
	require.Contains(t, string(got), `<root`)
}

func TestZeroValueCanonicalizerFluent(t *testing.T) {
	t.Parallel()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><!-- comment --><child/></root>`))
	require.NoError(t, err)

	var can c14n.Canonicalizer
	got, err := can.Comments().CanonicalizeTo(doc)
	require.NoError(t, err)
	require.Contains(t, string(got), `<!-- comment -->`)
}

func TestEmptyNodeSetEmitsEmpty(t *testing.T) {
	t.Parallel()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><child/></root>`))
	require.NoError(t, err)

	// An explicitly EMPTY node set must produce EMPTY output per the C14N spec.
	got, err := c14n.NewCanonicalizer(c14n.C14N10).NodeSet([]helium.Node{}).CanonicalizeTo(doc)
	require.NoError(t, err)
	require.Len(t, got, 0, "empty node set must emit empty output, got %q", string(got))
}

func TestNoNodeSetEmitsFullDocument(t *testing.T) {
	t.Parallel()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><child/></root>`))
	require.NoError(t, err)

	// Without NodeSet, the whole document is canonicalized.
	got, err := c14n.NewCanonicalizer(c14n.C14N10).CanonicalizeTo(doc)
	require.NoError(t, err)
	require.Equal(t, `<root><child></child></root>`, string(got))
}

func TestRelativeNamespaceURIRejected(t *testing.T) {
	t.Parallel()
	// C14N spec requires failure on relative namespace URIs.
	xml := `<?xml version="1.0"?><root xmlns:bad="relative/uri"><child/></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	_, err = c14n.NewCanonicalizer(c14n.C14N10).CanonicalizeTo(doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "relative namespace URI")
}

func TestActiveRelativeNamespaceURIRejected(t *testing.T) {
	t.Parallel()
	// A programmatically built DOM may set an *active* namespace with a
	// relative URI via SetActiveNamespace. This active namespace is emitted
	// during canonicalization, so the C14N relative-URI check must reject it
	// even though it was never added as a declared namespace.
	doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)

	root := doc.CreateElement("root")
	require.NoError(t, doc.SetDocumentElement(root))

	child := doc.CreateElement("child")
	require.NoError(t, root.AddChild(child))
	require.NoError(t, child.SetActiveNamespace("p", "relative/uri"))

	_, err := c14n.NewCanonicalizer(c14n.C14N10).CanonicalizeTo(doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "relative namespace URI")
}

func TestAbsoluteNamespaceURIAccepted(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><root xmlns:ok="http://example.com"><child/></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	_, err = c14n.NewCanonicalizer(c14n.C14N10).CanonicalizeTo(doc)
	require.NoError(t, err)
}

func TestEmptyNamespaceURIAccepted(t *testing.T) {
	t.Parallel()
	// Empty namespace URI (default namespace undeclaration) must be allowed.
	xml := `<?xml version="1.0"?><root xmlns="http://example.com"><child xmlns=""/></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	_, err = c14n.NewCanonicalizer(c14n.C14N10).CanonicalizeTo(doc)
	require.NoError(t, err)
}

// collectDescendantElements returns the element nodes named "child" and all of
// their descendant nodes within doc.
func collectDescendantElements(t *testing.T, doc *helium.Document) []helium.Node {
	t.Helper()
	const name = "child"
	var out []helium.Node
	var walk func(n helium.Node, inside bool)
	walk = func(n helium.Node, inside bool) {
		include := inside
		if elem, ok := helium.AsNode[*helium.Element](n); ok && elem.LocalName() == name {
			include = true
		}
		if include {
			out = append(out, n)
		}
		for child := range helium.Children(n) {
			walk(child, include)
		}
	}
	walk(doc, false)
	return out
}

// TestC14N11HTTPBaseURIFixup verifies that when the configured base URI is an
// absolute http(s) URI, C14N 1.1 xml:base fixup produces a proper relative URI
// reference rather than a filesystem path. The root element carries xml:base
// but is excluded from the node-set, so its base contribution must be folded
// into the visible child's xml:base.
func TestC14N11HTTPBaseURIFixup(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><root xml:base="/c/"><child>text</child></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	// Visible node-set: the child element subtree only (root excluded).
	nodes := collectDescendantElements(t, doc)

	got, err := c14n.NewCanonicalizer(c14n.C14N11).
		NodeSet(nodes).
		BaseURI("http://example.com/a/b/doc.xml").
		CanonicalizeTo(doc)
	require.NoError(t, err)

	// The base URI is an http URI; xml:base fixup must yield a URI reference,
	// never a filesystem path derived from filepath.Abs.
	require.NotContains(t, string(got), "file:", "http base URI must not be turned into a file path")
	// root xml:base "/c/" resolved against http://example.com/a/b/doc.xml is
	// http://example.com/c/; relative to the visible ancestor base
	// http://example.com/a/b/doc.xml this is "../../c/".
	require.Contains(t, string(got), `xml:base="../../c/"`, "expected URI-relative xml:base, got: %s", string(got))
}

// TestC14N11SchemeOnlyBaseURIFixup verifies the fix also covers absolute URIs
// that carry a scheme but no // authority component (e.g. "mem:/a/b/doc.xml").
// Such URIs must not be routed through filepath.Abs.
func TestC14N11SchemeOnlyBaseURIFixup(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><root xml:base="/c/"><child>text</child></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	nodes := collectDescendantElements(t, doc)

	got, err := c14n.NewCanonicalizer(c14n.C14N11).
		NodeSet(nodes).
		BaseURI("mem:/a/b/doc.xml").
		CanonicalizeTo(doc)
	require.NoError(t, err)

	require.NotContains(t, string(got), "file:", "scheme-only URI base must not become a file path")
	// root xml:base "/c/" resolved against mem:/a/b/doc.xml is mem:/c/;
	// relative to the ancestor base mem:/a/b/doc.xml this is "../../c/".
	require.Contains(t, string(got), `xml:base="../../c/"`, "expected URI-relative xml:base, got: %s", string(got))
}

// TestC14N11BaseURIFixupQueryFragment verifies that C14N 1.1 xml:base fixup
// preserves the query and fragment components of the element's base URI when
// relativizing. Only the path is relativized; the query+fragment must carry
// through unchanged into the canonical output.
func TestC14N11BaseURIFixupQueryFragment(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><root xml:base="/c/page?q=1&amp;r=2#frag"><child>text</child></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	// Visible node-set: the child element subtree only (root excluded), so the
	// root's xml:base (carrying a query and fragment) is folded into the child.
	nodes := collectDescendantElements(t, doc)

	got, err := c14n.NewCanonicalizer(c14n.C14N11).
		NodeSet(nodes).
		BaseURI("http://example.com/a/b/doc.xml").
		CanonicalizeTo(doc)
	require.NoError(t, err)

	// root xml:base "/c/page?q=1&r=2#frag" resolved against
	// http://example.com/a/b/doc.xml is http://example.com/c/page?q=1&r=2#frag;
	// relative to the visible ancestor base http://example.com/a/b/doc.xml the
	// path component is "../../c/page" and the query+fragment carry through.
	require.Contains(t, string(got), `xml:base="../../c/page?q=1&amp;r=2#frag"`, "query and fragment must be preserved, got: %s", string(got))
}

// TestC14N11BaseURIFixupEmptyPathQueryFragment verifies that when both the base
// URI and the resolved target have no path component, xml:base fixup does not
// inject a spurious "." segment. Injecting "." would relativize to "./?q=1#frag"
// which resolves to "http://example.com/?q=1#frag" (with a slash) — a different
// URI than "http://example.com?q=1#frag". The query+fragment must attach to the
// bare authority instead.
func TestC14N11BaseURIFixupEmptyPathQueryFragment(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><root xml:base="?q=1#frag"><child>text</child></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	// Visible node-set: the child element subtree only (root excluded), so the
	// root's xml:base is folded into the child.
	nodes := collectDescendantElements(t, doc)

	got, err := c14n.NewCanonicalizer(c14n.C14N11).
		NodeSet(nodes).
		BaseURI("http://example.com").
		CanonicalizeTo(doc)
	require.NoError(t, err)

	// root xml:base "?q=1#frag" resolved against http://example.com is
	// http://example.com?q=1#frag; relative to the same base it must be the bare
	// "?q=1#frag" with no leading "." or "/".
	require.Contains(t, string(got), `xml:base="?q=1#frag"`, "query and fragment must attach to authority without a spurious path, got: %s", string(got))
}

// TestC14N11BaseURIFixupSamePathQueryFragment verifies that when the resolved
// target shares the base document's exact path but differs only by a
// query/fragment, the relativized xml:base resolves back to the exact target
// document — not its containing directory. Here the target is the base's own
// filename ("doc.xml") plus a query+fragment, so the relative reference is
// "doc.xml?q=1#frag", which resolves to
// "http://example.com/a/b/doc.xml?q=1#frag" (the target document).
func TestC14N11BaseURIFixupSamePathQueryFragment(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><root xml:base="?q=1#frag"><child>text</child></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	// Visible node-set: the child element subtree only (root excluded), so the
	// root's xml:base is folded into the child.
	nodes := collectDescendantElements(t, doc)

	got, err := c14n.NewCanonicalizer(c14n.C14N11).
		NodeSet(nodes).
		BaseURI("http://example.com/a/b/doc.xml").
		CanonicalizeTo(doc)
	require.NoError(t, err)

	// root xml:base "?q=1#frag" resolved against http://example.com/a/b/doc.xml
	// is http://example.com/a/b/doc.xml?q=1#frag; relative to that base it is the
	// filename "doc.xml" carrying the query+fragment, which resolves back to the
	// exact target document.
	require.Contains(t, string(got), `xml:base="doc.xml?q=1#frag"`, "same-path query and fragment must resolve to the target document, got: %s", string(got))
}

// TestC14N11BaseURIFixupEmptyRelativeQueryFragment exercises the empty
// relative-reference branch directly: when the resolved target's path is
// identical to the base document's directory path, the relativized path is the
// empty string. Injecting "." there would yield ".?q=1#frag" which resolves to
// the directory with a query rather than the exact target; keeping the relative
// reference as the bare "?q=1#frag" resolves back to the exact target document.
func TestC14N11BaseURIFixupEmptyRelativeQueryFragment(t *testing.T) {
	t.Parallel()
	// Base URI is the directory http://example.com/a/b/; the xml:base "?q=1#frag"
	// resolves to http://example.com/a/b/?q=1#frag, whose path equals the base
	// directory, so the relative path component is empty.
	xml := `<?xml version="1.0"?><root xml:base="?q=1#frag"><child>text</child></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	nodes := collectDescendantElements(t, doc)

	got, err := c14n.NewCanonicalizer(c14n.C14N11).
		NodeSet(nodes).
		BaseURI("http://example.com/a/b/").
		CanonicalizeTo(doc)
	require.NoError(t, err)

	// The relative path is empty (target path == base directory), so the result
	// must be the bare "?q=1#frag" — resolving it against the base yields the
	// exact target http://example.com/a/b/?q=1#frag, whereas ".?q=1#frag" would
	// also resolve there but introduce a spurious "." segment.
	require.Contains(t, string(got), `xml:base="?q=1#frag"`, "empty relative reference must carry only the query+fragment, got: %s", string(got))
}

// TestC14N11BaseURIFixupOpaqueURI verifies that C14N 1.1 xml:base fixup handles
// opaque (non-hierarchical) absolute URIs such as "urn:target". These carry
// their data in the URL's Opaque field rather than Path, so path-based
// relativization is meaningless and previously produced an empty synthetic
// value that panicked in writeAttribute (nil attr → writeAttrValue(nil)). The
// fixup must instead emit the target URI absolutely with no panic.
func TestC14N11BaseURIFixupOpaqueURI(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><root xml:base="urn:target"><child>text</child></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	// Visible node-set: the child element subtree only (root excluded), so the
	// root's xml:base is folded into the child.
	nodes := collectDescendantElements(t, doc)

	got, err := c14n.NewCanonicalizer(c14n.C14N11).
		NodeSet(nodes).
		BaseURI("urn:base").
		CanonicalizeTo(doc)
	require.NoError(t, err)

	// The base "urn:base" and target "urn:target" are opaque URIs; the target
	// cannot be relativized against the base, so it must be emitted absolutely.
	require.Contains(t, string(got), `xml:base="urn:target"`, "opaque xml:base must canonicalize absolutely without panic, got: %s", string(got))
}

// TestC14N11BaseURIFixupBaseCarriesQuery is a convergence regression: when the
// configured base URI carries a query (or fragment) that the target does NOT,
// a naive relativizer can produce a relative reference that resolves back to the
// BASE rather than the TARGET. Here the base is "http://example.com?old=1" and
// the hidden ancestor's xml:base is the absolute "http://example.com" (no query).
// The naive relative reference would be the empty string, which resolves against
// the base to "http://example.com?old=1" — silently re-attaching "?old=1" and
// changing the URI. The round-trip convergence check must detect this and fall
// back to emitting the absolute target so the canonical xml:base resolves to the
// exact target "http://example.com".
func TestC14N11BaseURIFixupBaseCarriesQuery(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><root xml:base="http://example.com"><child>text</child></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	// Visible node-set: the child element subtree only (root excluded), so the
	// root's absolute xml:base is folded into the child.
	nodes := collectDescendantElements(t, doc)

	got, err := c14n.NewCanonicalizer(c14n.C14N11).
		NodeSet(nodes).
		BaseURI("http://example.com?old=1").
		CanonicalizeTo(doc)
	require.NoError(t, err)

	// The emitted xml:base must resolve to the exact target "http://example.com",
	// not the base "http://example.com?old=1". An empty relative reference would
	// resolve back to the base, so the absolute target must be emitted instead.
	require.Contains(t, string(got), `xml:base="http://example.com"`, "base-carries-query: xml:base must resolve to the target, not the base, got: %s", string(got))
	require.NotContains(t, string(got), `old=1`, "the base's query must not leak into the canonical xml:base, got: %s", string(got))
}
