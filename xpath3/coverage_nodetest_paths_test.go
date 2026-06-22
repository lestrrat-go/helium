package xpath3_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Path steps with kind tests (processing-instruction(target), element(name),
// attribute(name), comment(), text(), node()) exercise matchNodeTest and
// matchElementOrAttributeName branches that bare name tests do not.
func TestNodeTestPaths(t *testing.T) {
	const xml = `<?xml version="1.0"?>` +
		`<root a="1" b="2">` +
		`<?proc-a data?><?proc-b more?>` +
		`text-before<child>inner</child>text-after` +
		`<!--a comment-->` +
		`</root>`

	doc := mustParseXML(t, xml)
	root := doc.DocumentElement()

	cases := []struct {
		expr   string
		expect int
	}{
		{`child::processing-instruction()`, 2},
		{`child::processing-instruction("proc-a")`, 1},
		{`child::processing-instruction("missing")`, 0},
		{`child::element()`, 1},
		{`child::element(child)`, 1},
		{`child::element(other)`, 0},
		{`attribute::attribute()`, 2},
		{`attribute::attribute(a)`, 1},
		{`attribute::attribute(missing)`, 0},
		{`child::text()`, 2},
		{`child::comment()`, 1},
		{`child::node()`, 6},
		{`descendant::text()`, 3},
		{`child::*`, 1},
		{`attribute::*`, 2},
	}
	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			nodes, err := find(t.Context(), root, tc.expr)
			require.NoError(t, err, tc.expr)
			require.Len(t, nodes, tc.expect, tc.expr)
		})
	}

	// document-node(element(root)) matched from the document node.
	nodes, err := find(t.Context(), doc, `self::document-node(element(root))`)
	require.NoError(t, err)
	require.Len(t, nodes, 1)

	nodes, err = find(t.Context(), doc, `self::document-node(element(other))`)
	require.NoError(t, err)
	require.Empty(t, nodes)
}
