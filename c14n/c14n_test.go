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

const testdataBase = "../testdata/libxml2-compat/c14n"

func parseTestDoc(t *testing.T, path string) *helium.Document {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err, "reading test file %s", path)

	p := helium.NewParser().BlockXXE(false).SubstituteEntities(true).LoadExternalDTD(true).DefaultDTDAttributes(true).BaseURI(path).FS(helium.PermissiveFS())

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
	p := helium.NewParser().BlockXXE(false).SubstituteEntities(true).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(helium.PermissiveFS())
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

func runC14NTest(t *testing.T, category, name string, can c14n.Canonicalizer) {
	t.Helper()

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
			runC14NTest(t, "without-comments", name, c14n.NewCanonicalizer(c14n.C14N10))
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
			runC14NTest(t, "exc-without-comments", name, c14n.NewCanonicalizer(c14n.ExclusiveC14N10))
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
			runC14NTest(t, "with-comments", name, c14n.NewCanonicalizer(c14n.C14N10).Comments())
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
			runC14NTest(t, "1-1-without-comments", name, c14n.NewCanonicalizer(c14n.C14N11))
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

func TestEntityReferenceReplacementTextInContent(t *testing.T) {
	t.Parallel()
	// The default parser leaves internal general entity references unexpanded,
	// so an EntityRef node is present in element content. Canonicalization must
	// emit the entity's replacement text per the W3C Canonical XML spec, or the
	// content is excluded from any digest computed over the canonical output.
	src := `<!DOCTYPE r [<!ENTITY x "hello-entity-content">]><r>before-&x;-after</r>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	got, err := c14n.NewCanonicalizer(c14n.C14N10).CanonicalizeTo(doc)
	require.NoError(t, err)
	require.Equal(t, `<r>before-hello-entity-content-after</r>`, string(got))
}

func TestEntityReferenceReplacementTextWithMarkup(t *testing.T) {
	t.Parallel()
	// An internal entity whose replacement text contains markup must have that
	// markup canonicalized (nested element emitted), not dropped.
	src := `<!DOCTYPE r [<!ENTITY x "<b>bold</b>">]><r>a&x;b</r>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	got, err := c14n.NewCanonicalizer(c14n.C14N10).CanonicalizeTo(doc)
	require.NoError(t, err)
	require.Equal(t, `<r>a<b>bold</b>b</r>`, string(got))
}

func TestEntityReferenceReplacementTextInAttribute(t *testing.T) {
	t.Parallel()
	// Attribute-context entity expansion already works; lock it in alongside
	// the element-content fix.
	src := `<!DOCTYPE r [<!ENTITY x "hello-entity-content">]><r a="before-&x;-after"/>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	got, err := c14n.NewCanonicalizer(c14n.C14N10).CanonicalizeTo(doc)
	require.NoError(t, err)
	require.Equal(t, `<r a="before-hello-entity-content-after"></r>`, string(got))
}

func TestEntityReferenceReplacementNamespaceContextPerSite(t *testing.T) {
	t.Parallel()
	// The same entity, whose replacement text is a namespace-prefixed element,
	// is referenced twice under DIFFERENT in-scope bindings for that prefix.
	// Per W3C Canonical XML the replacement is canonicalized as if textually
	// substituted at each reference site, so each expansion must reflect ITS OWN
	// binding for the prefix — the second must not inherit the first site's
	// namespace resolution cached on the shared Entity declaration node.
	src := `<!DOCTYPE r [<!ENTITY x "<p:x/>">]>` +
		`<r xmlns:p="urn:default"><a xmlns:p="urn:one">&x;</a><b xmlns:p="urn:two">&x;</b></r>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	got, err := c14n.NewCanonicalizer(c14n.C14N10).CanonicalizeTo(doc)
	require.NoError(t, err)
	require.Equal(t,
		`<r xmlns:p="urn:default"><a xmlns:p="urn:one"><p:x></p:x></a><b xmlns:p="urn:two"><p:x></p:x></b></r>`,
		string(got))
}

func TestEntityReferenceReplacementNamespaceContextPerSiteExclusive(t *testing.T) {
	t.Parallel()
	// Exclusive C14N: the same entity, whose replacement is an unprefixed element
	// carrying a q-prefixed attribute, is referenced twice under DIFFERENT q
	// bindings. Exclusive mode emits a namespace only where it is visibly utilized;
	// the q:a attribute utilizes q on the entity-replacement element. Each expansion
	// must render ITS OWN reference-site binding for q, not the binding cached on the
	// shared Entity declaration node at first parse.
	src := `<!DOCTYPE r [<!ENTITY x "<e q:a=&#34;v&#34;/>">]>` +
		`<r><a xmlns:q="urn:one">&x;</a><b xmlns:q="urn:two">&x;</b></r>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	got, err := c14n.NewCanonicalizer(c14n.ExclusiveC14N10).CanonicalizeTo(doc)
	require.NoError(t, err)
	require.Equal(t,
		`<r><a><e xmlns:q="urn:one" q:a="v"></e></a><b><e xmlns:q="urn:two" q:a="v"></e></b></r>`,
		string(got))
}

func TestEntityReferenceReplacementUnprefixedAttrSortOrder(t *testing.T) {
	t.Parallel()
	// An entity-replacement element carries an unprefixed attribute and a prefixed
	// one, expanded under an in-scope default namespace. An unprefixed attribute is
	// in NO namespace — the default namespace never applies to attributes — so it
	// must sort ahead of the prefixed attribute (empty namespace URI sorts first).
	// The canonical output must match the equivalent fully-expanded document.
	entitySrc := `<!DOCTYPE r [<!ENTITY x "<e z=&#34;0&#34; p:a=&#34;1&#34;/>">]>` +
		`<r><a xmlns="urn:def" xmlns:p="urn:a">&x;</a></r>`
	expandedSrc := `<r><a xmlns="urn:def" xmlns:p="urn:a"><e z="0" p:a="1"/></a></r>`

	entityDoc, err := helium.NewParser().Parse(t.Context(), []byte(entitySrc))
	require.NoError(t, err)
	expandedDoc, err := helium.NewParser().Parse(t.Context(), []byte(expandedSrc))
	require.NoError(t, err)

	entityGot, err := c14n.NewCanonicalizer(c14n.C14N10).CanonicalizeTo(entityDoc)
	require.NoError(t, err)
	expandedGot, err := c14n.NewCanonicalizer(c14n.C14N10).CanonicalizeTo(expandedDoc)
	require.NoError(t, err)

	require.Equal(t, string(expandedGot), string(entityGot))
	require.Equal(t,
		`<r><a xmlns="urn:def" xmlns:p="urn:a"><e z="0" p:a="1"></e></a></r>`,
		string(entityGot))
}

func TestEntityReferenceReplacementUnboundPrefixAtSecondSiteErrors(t *testing.T) {
	t.Parallel()
	// The same entity, whose replacement carries a q-prefixed attribute, is
	// referenced first where q is bound and then where q is UNBOUND. The second
	// expansion cannot borrow the first site's cached binding — a prefixed name
	// whose prefix is not in scope at the reference site is not namespace-well-formed
	// there, so canonicalization must error rather than emit a borrowed binding.
	src := `<!DOCTYPE r [<!ENTITY x "<e q:a=&#34;v&#34;/>">]>` +
		`<r><a xmlns:q="urn:one">&x;</a><b>&x;</b></r>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	_, err = c14n.NewCanonicalizer(c14n.ExclusiveC14N10).CanonicalizeTo(doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not in scope")

	// Inclusive C14N10 reaches the same out-of-scope prefix via the attribute axis.
	_, err = c14n.NewCanonicalizer(c14n.C14N10).CanonicalizeTo(doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not in scope")
}

func TestEntityReferenceReplacementElementUnboundPrefixAtSecondSiteErrors(t *testing.T) {
	t.Parallel()
	// The same entity, whose replacement is a p-prefixed ELEMENT, is referenced
	// first where p is bound and then where p is UNBOUND. The second expansion
	// cannot borrow the first site's cached binding — an element name whose prefix
	// is not in scope at the reference site is not namespace-well-formed there, so
	// canonicalization must error rather than emit a dangling prefix. This mirrors
	// the attribute-axis case on the element-name axis and must hold in every mode.
	src := `<!DOCTYPE r [<!ENTITY x "<p:e/>">]>` +
		`<r><a xmlns:p="urn:one">&x;</a><b>&x;</b></r>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	_, err = c14n.NewCanonicalizer(c14n.C14N10).CanonicalizeTo(doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not in scope")

	_, err = c14n.NewCanonicalizer(c14n.C14N11).CanonicalizeTo(doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not in scope")

	_, err = c14n.NewCanonicalizer(c14n.ExclusiveC14N10).CanonicalizeTo(doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not in scope")

	nodes := evaluateNodeSet(t, doc, "//. | //@* | //namespace::*", nil)
	_, err = c14n.NewCanonicalizer(c14n.C14N10).NodeSet(nodes).CanonicalizeTo(doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not in scope")

	_, err = c14n.NewCanonicalizer(c14n.ExclusiveC14N10).NodeSet(nodes).CanonicalizeTo(doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not in scope")
}

func TestEntityReferenceReplacementNamespaceContextPerSiteNodeSet(t *testing.T) {
	t.Parallel()
	// Node-set C14N: the same entity, whose replacement is a q-prefixed element, is
	// referenced twice under DIFFERENT q bindings, with the whole tree selected. The
	// node-set namespace rendering must reflect each reference site's own binding for
	// q rather than the URI cached on the shared Entity declaration node.
	src := `<!DOCTYPE r [<!ENTITY x "<p:x/>">]>` +
		`<r xmlns:p="urn:default"><a xmlns:p="urn:one">&x;</a><b xmlns:p="urn:two">&x;</b></r>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	nodes := evaluateNodeSet(t, doc, "//. | //@* | //namespace::*", nil)
	got, err := c14n.NewCanonicalizer(c14n.C14N10).NodeSet(nodes).CanonicalizeTo(doc)
	require.NoError(t, err)
	require.Equal(t,
		`<r xmlns:p="urn:default"><a xmlns:p="urn:one"><p:x></p:x></a><b xmlns:p="urn:two"><p:x></p:x></b></r>`,
		string(got))
}

func TestEntityReferenceReplacementDefaultNamespaceContextPerSiteExclusive(t *testing.T) {
	t.Parallel()
	// Exclusive C14N: the same entity, whose replacement is an unprefixed element, is
	// referenced first inside an element with default namespace urn:one and then
	// inside a sibling with NO default namespace. At the first site the replacement
	// element is in urn:one and visibly utilizes the default namespace; at the second
	// it is in no namespace. The default-namespace resolution must follow each
	// reference site, not the default binding cached on the shared Entity node.
	src := `<!DOCTYPE r [<!ENTITY x "<c/>">]>` +
		`<r><a xmlns="urn:one">&x;</a><b>&x;</b></r>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	got, err := c14n.NewCanonicalizer(c14n.ExclusiveC14N10).CanonicalizeTo(doc)
	require.NoError(t, err)
	// <a> itself is in urn:one (unprefixed under the default namespace) so it
	// visibly utilizes and emits xmlns="urn:one"; the replacement <c> inherits it
	// and adds nothing. At <b> there is no default namespace, so the replacement
	// <c> is in no namespace and must NOT re-emit the first site's urn:one.
	require.Equal(t,
		`<r><a xmlns="urn:one"><c></c></a><b><c></c></b></r>`,
		string(got))
}

func TestEntityReferenceReplacementDefaultNamespaceReverseOrderExclusive(t *testing.T) {
	t.Parallel()
	// Exclusive C14N, reverse order of the case above: the same entity, whose
	// replacement is an unprefixed element, is referenced first inside an element
	// with NO default namespace and then inside a sibling that declares a default
	// namespace. Parsing caches the replacement element's active namespace at the
	// first (no-default) site as nil, so the second site must re-derive the
	// element's own namespace from the reference-site default binding rather than
	// trusting the cached nil. At the first site <e> is in no namespace; at the
	// second it is in urn:two and must emit xmlns="urn:two".
	src := `<!DOCTYPE r [<!ENTITY x "<e/>">]>` +
		`<r xmlns:p="urn:p"><p:a>&x;</p:a><p:b xmlns="urn:two">&x;</p:b></r>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	got, err := c14n.NewCanonicalizer(c14n.ExclusiveC14N10).CanonicalizeTo(doc)
	require.NoError(t, err)
	require.Equal(t,
		`<r><p:a xmlns:p="urn:p"><e></e></p:a><p:b xmlns:p="urn:p"><e xmlns="urn:two"></e></p:b></r>`,
		string(got))
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

	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.SetDocumentElement(root))

	child, err := doc.CreateElement("child")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(child))
	require.NoError(t, child.SetActiveNamespace("p", "relative/uri"))

	_, err = c14n.NewCanonicalizer(c14n.C14N10).CanonicalizeTo(doc)
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

// TestC14N11XMLBaseLexicalJoin verifies that C14N 1.1 xml:base fixup is the
// lexical join of the in-document xml:base values (W3C xml-c14n11 §2.4 / libxml2
// xmlC14NFixupBaseAttr), with no external/retrieval base URI. The omitted root
// carries the absolute-path xml:base "/c/", which must be emitted verbatim on
// the visible child — NOT re-relativized into "../../c/".
func TestC14N11XMLBaseLexicalJoin(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><root xml:base="/c/"><child>text</child></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	// Visible node-set: the child element subtree only (root excluded).
	nodes := collectDescendantElements(t, doc)

	got, err := c14n.NewCanonicalizer(c14n.C14N11).NodeSet(nodes).CanonicalizeTo(doc)
	require.NoError(t, err)

	require.Contains(t, string(got), `xml:base="/c/"`, "absolute-path xml:base must be joined verbatim, got: %s", string(got))
	require.NotContains(t, string(got), "../../c/", "xml:base must not be re-relativized against a retrieval base URI, got: %s", string(got))
}

// TestC14N11ExcludedOwnXMLBase covers a rendered element whose own xml:base is
// excluded from the node set, with no omitted ancestor carrying xml:base. The
// default (libxml2) mode still emits the element's own value; strict W3C mode
// omits it (no omitted-ancestor contribution → no fixup).
func TestC14N11ExcludedOwnXMLBase(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><root><hidden><child xml:base="x">text</child></hidden></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	nodes := collectDescendantElements(t, doc)

	gotDefault, err := c14n.NewCanonicalizer(c14n.C14N11).NodeSet(nodes).CanonicalizeTo(doc)
	require.NoError(t, err)
	require.Contains(t, string(gotDefault), `xml:base="x"`, "libxml2 default emits the element's own xml:base, got: %s", string(gotDefault))

	gotStrict, err := c14n.NewCanonicalizer(c14n.C14N11).NodeSet(nodes).StrictXMLAttributes().CanonicalizeTo(doc)
	require.NoError(t, err)
	require.NotContains(t, string(gotStrict), "xml:base", "strict mode omits an excluded own xml:base with no omitted-ancestor base, got: %s", string(gotStrict))
}

// TestC14N11ExcludedOwnXMLLang covers a rendered element whose own xml:lang is
// excluded from the node set, with an ancestor carrying a different xml:lang.
// The element's own value must block inheritance of the ancestor value in BOTH
// modes (the previous bug imported the ancestor "en"); the default emits the own
// value, strict omits it.
func TestC14N11ExcludedOwnXMLLang(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><a xml:lang="en"><hidden><child xml:lang="fr">text</child></hidden></a>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	nodes := collectDescendantElements(t, doc)

	gotDefault, err := c14n.NewCanonicalizer(c14n.C14N11).NodeSet(nodes).CanonicalizeTo(doc)
	require.NoError(t, err)
	require.Contains(t, string(gotDefault), `xml:lang="fr"`, "libxml2 default emits the element's own xml:lang, got: %s", string(gotDefault))
	require.NotContains(t, string(gotDefault), `xml:lang="en"`, "the element's own xml:lang must block ancestor inheritance, got: %s", string(gotDefault))

	gotStrict, err := c14n.NewCanonicalizer(c14n.C14N11).NodeSet(nodes).StrictXMLAttributes().CanonicalizeTo(doc)
	require.NoError(t, err)
	require.NotContains(t, string(gotStrict), "xml:lang", "strict mode omits an excluded own xml:lang (still blocking inheritance), got: %s", string(gotStrict))
}

// TestC14N11StrictXMLBaseIncludesOwnValue verifies that when strict-mode fixup
// runs (an omitted ancestor carries xml:base), the element's own xml:base value
// is still part of the join sequence even though the attribute is excluded from
// the node set — matching default mode here.
func TestC14N11StrictXMLBaseIncludesOwnValue(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><root xml:base="a/"><child xml:base="b">text</child></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	nodes := collectDescendantElements(t, doc)

	gotStrict, err := c14n.NewCanonicalizer(c14n.C14N11).NodeSet(nodes).StrictXMLAttributes().CanonicalizeTo(doc)
	require.NoError(t, err)
	require.Contains(t, string(gotStrict), `xml:base="a/b"`, "strict join must include the element's own value, got: %s", string(gotStrict))
}

// TestC14N11StrictWholeDocumentUnaffected verifies the StrictXMLAttributes
// toggle does not change whole-document output (it governs node-set processing
// only): an empty xml:base must be dropped in both default and strict, with no
// node set.
func TestC14N11StrictWholeDocumentUnaffected(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><root xml:base=""><child/></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	gotDefault, err := c14n.NewCanonicalizer(c14n.C14N11).CanonicalizeTo(doc)
	require.NoError(t, err)
	gotStrict, err := c14n.NewCanonicalizer(c14n.C14N11).StrictXMLAttributes().CanonicalizeTo(doc)
	require.NoError(t, err)
	require.Equal(t, string(gotDefault), string(gotStrict), "strict must not change whole-document output")
	require.NotContains(t, string(gotStrict), "xml:base", "empty xml:base must be dropped, got: %s", string(gotStrict))
}

// TestC14N11StrictFailClosedXMLBase verifies that a degenerate xml:base on an
// omitted ancestor (an empty-authority "//" that cannot be canonicalized
// faithfully) is a best-effort no-error result in the default mode but an
// operation failure under StrictXMLAttributes.
func TestC14N11StrictFailClosedXMLBase(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><root xml:base="//"><child xml:base="x">t</child></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	nodes := collectDescendantElements(t, doc)

	// Default (libxml2-compat): best-effort, no error.
	_, err = c14n.NewCanonicalizer(c14n.C14N11).NodeSet(nodes).CanonicalizeTo(doc)
	require.NoError(t, err, "default mode must stay permissive")

	// Strict: fail closed.
	_, err = c14n.NewCanonicalizer(c14n.C14N11).NodeSet(nodes).StrictXMLAttributes().CanonicalizeTo(doc)
	require.Error(t, err, "strict mode must reject an un-canonicalizable xml:base")
	require.Contains(t, err.Error(), "xml:base")
}

// TestC14N11StrictFailClosedSingleValue verifies the strict fail-closed guard
// also covers a lone degenerate xml:base with no ancestor to join against (the
// term is validated even though no join occurs). The visible element's own
// xml:base "urn://" must error in strict mode but emit best-effort by default.
func TestC14N11StrictFailClosedSingleValue(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><root><child xml:base="urn://">t</child></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	// Node set includes the child, its text, and its own xml:base attribute; root
	// is omitted (a gap) but carries no xml:base.
	nodes := evaluateNodeSet(t, doc, "//child | //child/@xml:base | //child/text()", nil)

	gotDefault, err := c14n.NewCanonicalizer(c14n.C14N11).NodeSet(nodes).CanonicalizeTo(doc)
	require.NoError(t, err, "default mode emits the lone value verbatim")
	require.Contains(t, string(gotDefault), `xml:base="urn://"`)

	_, err = c14n.NewCanonicalizer(c14n.C14N11).NodeSet(nodes).StrictXMLAttributes().CanonicalizeTo(doc)
	require.Error(t, err, "strict mode must reject a lone degenerate xml:base")
}

// TestC14N11StrictFailClosedOmittedAttr verifies the strict guard also covers an
// xml:base carried in the node set on an omitted element (rendered verbatim via
// the attribute axis, not the fixup path).
func TestC14N11StrictFailClosedOmittedAttr(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><root><hidden xml:base="//"><child>t</child></hidden></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	// hidden is omitted but its xml:base attribute is in the node set.
	nodes := evaluateNodeSet(t, doc, "//hidden/@xml:base | //child | //child/text()", nil)

	gotDefault, err := c14n.NewCanonicalizer(c14n.C14N11).NodeSet(nodes).CanonicalizeTo(doc)
	require.NoError(t, err, "default mode emits the omitted-element attribute verbatim")
	require.Contains(t, string(gotDefault), `xml:base="//"`)

	_, err = c14n.NewCanonicalizer(c14n.C14N11).NodeSet(nodes).StrictXMLAttributes().CanonicalizeTo(doc)
	require.Error(t, err, "strict mode must reject a degenerate xml:base on an omitted element")
}

// TestC14N10StrictFailClosedXMLBase verifies the strict fail-closed guard also
// covers C14N 1.0, where xml:base is an ordinary (visible or inherited) attribute
// rather than a fixup result — caught at the shared writeAttribute chokepoint.
func TestC14N10StrictFailClosedXMLBase(t *testing.T) {
	t.Parallel()
	// Visible own degenerate xml:base.
	xml := `<?xml version="1.0"?><root><child xml:base="urn://">t</child></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	nodes := evaluateNodeSet(t, doc, "//child | //child/@xml:base | //child/text()", nil)

	_, err = c14n.NewCanonicalizer(c14n.C14N10).NodeSet(nodes).CanonicalizeTo(doc)
	require.NoError(t, err, "default C14N 1.0 stays permissive")
	_, err = c14n.NewCanonicalizer(c14n.C14N10).NodeSet(nodes).StrictXMLAttributes().CanonicalizeTo(doc)
	require.Error(t, err, "strict C14N 1.0 must reject a degenerate visible xml:base")

	// Inherited degenerate xml:base from an omitted ancestor across a gap.
	xml2 := `<?xml version="1.0"?><a xml:base="urn://"><hidden><child>t</child></hidden></a>`
	doc2, err := helium.NewParser().Parse(t.Context(), []byte(xml2))
	require.NoError(t, err)
	nodes2 := collectDescendantElements(t, doc2)
	_, err = c14n.NewCanonicalizer(c14n.C14N10).NodeSet(nodes2).StrictXMLAttributes().CanonicalizeTo(doc2)
	require.Error(t, err, "strict C14N 1.0 must reject a degenerate inherited xml:base")
}

// TestC14N10ExcludedOwnXMLLang covers the C14N 1.0 inheritance-blocking
// divergence. With a rendered element's own xml:lang excluded from the node set,
// the default (libxml2) blocks only on rendered attributes and so imports the
// ancestor "en"; strict mode blocks on the element's full attribute axis and so
// imports nothing.
func TestC14N10ExcludedOwnXMLLang(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><a xml:lang="en"><hidden><child xml:lang="fr">text</child></hidden></a>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	nodes := collectDescendantElements(t, doc)

	gotDefault, err := c14n.NewCanonicalizer(c14n.C14N10).NodeSet(nodes).CanonicalizeTo(doc)
	require.NoError(t, err)
	require.Contains(t, string(gotDefault), `xml:lang="en"`, "libxml2 default imports the ancestor xml:lang, got: %s", string(gotDefault))

	gotStrict, err := c14n.NewCanonicalizer(c14n.C14N10).NodeSet(nodes).StrictXMLAttributes().CanonicalizeTo(doc)
	require.NoError(t, err)
	require.NotContains(t, string(gotStrict), "xml:lang", "strict mode blocks inheritance via the excluded own xml:lang, got: %s", string(gotStrict))
}

// TestC14N10OmittedElementNSSuppression verifies that a namespace node carried
// in the node set on an omitted intermediate element is not re-emitted as text
// when the nearest visible ancestor already renders the same prefix and value
// (the inclusive-C14N suppression rule). Here "mid" is excluded but its inherited
// p-namespace node is in the node set; root already declares xmlns:p, so it must
// not reappear between root and child.
func TestC14N10OmittedElementNSSuppression(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><root xmlns:p="urn:p"><mid><child/></mid></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	// Everything (elements, attributes, namespace nodes) except the "mid"
	// element: mid is omitted but its namespace nodes remain in the set.
	nodes := evaluateNodeSet(t, doc, "(//. | //@* | //namespace::*)[not(self::mid)]", nil)

	got, err := c14n.NewCanonicalizer(c14n.C14N10).NodeSet(nodes).CanonicalizeTo(doc)
	require.NoError(t, err)
	require.Equal(t, `<root xmlns:p="urn:p"><child></child></root>`, string(got))
}

// TestExclusiveC14NOmittedElementNSSuppression verifies that in exclusive C14N a
// namespace node on an omitted element is not re-emitted as text when it was
// already rendered (here via the inclusive prefix on the visible root).
func TestExclusiveC14NOmittedElementNSSuppression(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><root xmlns:p="urn:p"><mid><child/></mid></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	nodes := evaluateNodeSet(t, doc, "(//. | //@* | //namespace::*)[not(self::mid)]", nil)

	got, err := c14n.NewCanonicalizer(c14n.ExclusiveC14N10).
		NodeSet(nodes).
		InclusiveNamespaces([]string{"p"}).
		CanonicalizeTo(doc)
	require.NoError(t, err)
	require.Equal(t, `<root xmlns:p="urn:p"><child></child></root>`, string(got))
}

// TestC14N10OmittedElementAttributes verifies that an omitted element still
// emits its in-node-set attributes as text (libxml2 processes the attribute axis
// for non-visible elements). Here root is excluded but its attribute @a is in the
// node set, so it must appear as leading text before the visible child.
func TestC14N10OmittedElementAttributes(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><root a="1"><child>x</child></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	nodes := evaluateNodeSet(t, doc, "//child | //child/text() | //@a", nil)

	got, err := c14n.NewCanonicalizer(c14n.C14N10).NodeSet(nodes).CanonicalizeTo(doc)
	require.NoError(t, err)
	require.Equal(t, ` a="1"<child>x</child>`, string(got))
}

// TestRelativeNamespaceColonRejected verifies that a relative namespace URI
// containing a colon outside a scheme (e.g. "a/b:c") is still rejected. C14N
// requires an operation failure on relative namespace URIs, and a stray colon
// does not make a reference absolute.
func TestRelativeNamespaceColonRejected(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><root xmlns="a/b:c"><child/></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	_, err = c14n.NewCanonicalizer(c14n.C14N10).CanonicalizeTo(doc)
	require.Error(t, err, "relative namespace URI with a non-scheme colon must be rejected")
	require.Contains(t, err.Error(), "relative namespace URI")
}

// TestMalformedNamespaceURIRejected verifies a scheme-bearing but malformed
// namespace URI containing a raw space (which url.Parse tolerates in an opaque
// part but libxml2 rejects) is rejected.
func TestMalformedNamespaceURIRejected(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0"?><root xmlns:p="urn:foo bar"><p:child/></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	_, err = c14n.NewCanonicalizer(c14n.C14N10).CanonicalizeTo(doc)
	require.Error(t, err, "namespace URI with a raw space must be rejected")
	require.Contains(t, err.Error(), "namespace URI")
}
