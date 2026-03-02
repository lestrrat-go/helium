package c14n_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/c14n"
	"github.com/lestrrat-go/helium/xpath"
	"github.com/stretchr/testify/require"
)

const testdataBase = "../testdata/libxml2-compat/c14n"

// knownParseFailures lists test cases that fail during parsing due to
// helium parser limitations (not C14N bugs).
var knownParseFailures = map[string]string{
	"without-comments/example-3":     "duplicate namespace declaration handling",
	"without-comments/example-4":     "entity reference in single-quoted attribute",
	"without-comments/test-2":        "duplicate namespace declaration handling",
	"without-comments/test-3":        "duplicate namespace declaration handling",
	"with-comments/example-3":        "duplicate namespace declaration handling",
	"with-comments/example-4":        "entity reference in single-quoted attribute",
	"exc-without-comments/test-0":    "duplicate namespace declaration handling",
	"exc-without-comments/test-1":    "duplicate namespace declaration handling",
	"1-1-without-comments/example-3": "duplicate namespace declaration handling",
	"1-1-without-comments/example-4": "entity reference in single-quoted attribute",
}

func parseTestDoc(t *testing.T, path string) *helium.Document {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err, "reading test file %s", path)

	p := helium.NewParser()
	p.SetOption(helium.ParseNoEnt)
	p.SetOption(helium.ParseDTDLoad)
	p.SetOption(helium.ParseDTDAttr)
	p.SetBaseURI(path)

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
	p := helium.NewParser()
	p.SetOption(helium.ParseNoEnt)
	p.SetOption(helium.ParseDTDLoad)
	p.SetOption(helium.ParseDTDAttr)
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

	ctx := xpath.NewContext(
		xpath.WithNamespaces(nss),
	)

	result, err := xpath.EvaluateWith(doc, expr, ctx)
	require.NoError(t, err, "evaluating xpath: %s", expr)
	require.Equal(t, xpath.NodeSetResult, result.Type, "xpath result is not a node set")
	return result.NodeSet
}

func runC14NTest(t *testing.T, category, name string, mode c14n.Mode, opts ...c14n.Option) {
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
		opts = append(opts, c14n.WithNodeSet(nodes))
	}

	// Check for .ns file (inclusive namespace prefixes for exclusive C14N)
	nsPath := filepath.Join(testdataBase, category, "test", name+".ns")
	if _, err := os.Stat(nsPath); err == nil {
		prefixes := parseNSFile(t, nsPath)
		opts = append(opts, c14n.WithInclusiveNamespaces(prefixes))
	}

	// Pass base URI for C14N 1.1 xml:base fixup
	if mode == c14n.C14N11 {
		opts = append(opts, c14n.WithBaseURI(inputPath))
	}

	got, err := c14n.CanonicalizeTo(doc, mode, opts...)
	require.NoError(t, err)
	require.Equal(t, string(expected), string(got))
}

func TestC14N10WithoutComments(t *testing.T) {
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
			runC14NTest(t, "without-comments", name, c14n.C14N10)
		})
	}
}

func TestExclusiveC14N10WithoutComments(t *testing.T) {
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
			runC14NTest(t, "exc-without-comments", name, c14n.ExclusiveC14N10)
		})
	}
}

func TestC14N10WithComments(t *testing.T) {
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
			runC14NTest(t, "with-comments", name, c14n.C14N10, c14n.WithComments())
		})
	}
}

func TestC14N11WithoutComments(t *testing.T) {
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
			runC14NTest(t, "1-1-without-comments", name, c14n.C14N11)
		})
	}
}

func TestRelativeNamespaceURIRejected(t *testing.T) {
	// C14N spec requires failure on relative namespace URIs.
	xml := `<?xml version="1.0"?><root xmlns:bad="relative/uri"><child/></root>`
	doc, err := helium.Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	_, err = c14n.CanonicalizeTo(doc, c14n.C14N10)
	require.Error(t, err)
	require.Contains(t, err.Error(), "relative namespace URI")
}

func TestAbsoluteNamespaceURIAccepted(t *testing.T) {
	xml := `<?xml version="1.0"?><root xmlns:ok="http://example.com"><child/></root>`
	doc, err := helium.Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	_, err = c14n.CanonicalizeTo(doc, c14n.C14N10)
	require.NoError(t, err)
}

func TestEmptyNamespaceURIAccepted(t *testing.T) {
	// Empty namespace URI (default namespace undeclaration) must be allowed.
	xml := `<?xml version="1.0"?><root xmlns="http://example.com"><child xmlns=""/></root>`
	doc, err := helium.Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	_, err = c14n.CanonicalizeTo(doc, c14n.C14N10)
	require.NoError(t, err)
}
