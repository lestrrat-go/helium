package html_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/html"
	"github.com/stretchr/testify/require"
)

// findElement returns the first element named name found in a depth-first walk
// rooted at n, or nil.
func findElement(n helium.Node, name string) *helium.Element {
	for c := firstChild(n); c != nil; c = c.NextSibling() {
		if c.Type() != helium.ElementNode {
			continue
		}
		if c.Name() == name {
			if e, ok := c.(*helium.Element); ok {
				return e
			}
		}
		if e := findElement(c, name); e != nil {
			return e
		}
	}
	return nil
}

// firstChild guards against a typed-nil DocumentElement wrapper.
func firstChild(n helium.Node) helium.Node {
	if n == nil {
		return nil
	}
	return n.FirstChild()
}

// elementChildNames lists, in order, the names of the element children of n.
func elementChildNames(n helium.Node) []string {
	var names []string
	for c := firstChild(n); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode {
			names = append(names, c.Name())
		}
	}
	return names
}

// TestStartElementMultiColonName pins that a malformed multi-colon tag name is
// preserved as an element (not silently dropped) in default non-strict parsing,
// and that its unmatched end tag does not mis-promote later siblings. Splitting
// the tokenizer name at the first colon left a colon in the local part, which
// CreateElementNS rejected; the node was then dropped and the surrounding tree
// corrupted (span promoted out of div). The element must exist with its text,
// and span must stay a child of div alongside it.
func TestStartElementMultiColonName(t *testing.T) {
	const input = `<div><foo:bar:baz>inner</foo:bar:baz><span>s</span></div>`

	doc, err := html.NewParser().Parse(t.Context(), []byte(input))
	require.NoError(t, err)

	div := findElement(doc, "div")
	require.NotNil(t, div, "div element is present")

	// The multi-colon element survives with its full colon-bearing name and
	// its text content intact.
	multi := findElement(div, "foo:bar:baz")
	require.NotNil(t, multi, "multi-colon element is not dropped")
	require.Equal(t, "inner", string(multi.Content()), "text stays inside the element")

	// span remains a child of div (sibling of the multi-colon element), not
	// promoted to a sibling of div by the unmatched end tag.
	require.Equal(t,
		[]string{"foo:bar:baz", "span"},
		elementChildNames(div),
		"span stays a child of div next to the multi-colon element")

	span := findElement(div, "span")
	require.NotNil(t, span)
	require.Equal(t, "s", string(span.Content()))
}
