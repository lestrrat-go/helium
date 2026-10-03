package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

const namespaceAxisXML = `<root xmlns="urn:root" xmlns:xlink="http://www.w3.org/1999/xlink">
  <child xmlns:local="urn:local"/>
</root>`

func parseNamespaceAxisDoc(t *testing.T) *helium.Document {
	t.Helper()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(namespaceAxisXML))
	require.NoError(t, err)
	return doc
}

func TestNamespaceAxisStringValueAndParent(t *testing.T) {
	doc := parseNamespaceAxisDoc(t)

	result, err := evaluate(t.Context(), doc, `string(/*/namespace::xlink)`)
	require.NoError(t, err)

	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "http://www.w3.org/1999/xlink", s)

	nodes, err := find(t.Context(), doc, `/*/namespace::*/..`)
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Equal(t, "root", nodes[0].Name())
}

func TestNamespaceAxisNodeIdentity(t *testing.T) {
	doc := parseNamespaceAxisDoc(t)

	result, err := evaluate(t.Context(), doc, `/*/namespace::xlink is /*/*[1]/namespace::xlink`)
	require.NoError(t, err)

	sameInherited, ok := result.IsBoolean()
	require.True(t, ok)
	require.False(t, sameInherited)

	result, err = evaluate(t.Context(), doc, `/*/namespace::xlink is /*/namespace::*[. = 'http://www.w3.org/1999/xlink']`)
	require.NoError(t, err)

	sameLogical, ok := result.IsBoolean()
	require.True(t, ok)
	require.True(t, sameLogical)

	result, err = evaluate(t.Context(), doc, `generate-id(/*/namespace::xlink) eq generate-id(/*/namespace::*[. = 'http://www.w3.org/1999/xlink'])`)
	require.NoError(t, err)

	sameID, ok := result.IsBoolean()
	require.True(t, ok)
	require.True(t, sameID)
}

func TestNamespaceAxisPath(t *testing.T) {
	doc := parseNamespaceAxisDoc(t)

	result, err := evaluate(t.Context(), doc, `path((//namespace::xml)[1])`)
	require.NoError(t, err)

	xmlPath, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "/Q{urn:root}root[1]/namespace::xml", xmlPath)

	result, err = evaluate(t.Context(), doc, `path((//namespace::*[name()=''])[1])`)
	require.NoError(t, err)

	defaultPath, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, `/Q{urn:root}root[1]/namespace::*[Q{http://www.w3.org/2005/xpath-functions}local-name()=""]`, defaultPath)
}

// TestParentlessNamespaceNodeIdentity checks that parentless namespace nodes,
// such as the ones XSLT's xsl:namespace builds, keep their own identity: two
// of them with the same prefix are different nodes for the set operators,
// path deduplication, `is` and generate-id (XPath 3.1 §2.1.2). Only namespace
// nodes of an element are identified by that element and their prefix.
func TestParentlessNamespaceNodeIdentity(t *testing.T) {
	doc := parseNamespaceAxisDoc(t)
	ns := xpath3.ItemSlice{
		xpath3.NodeItem{Node: helium.NewNamespaceNodeWrapper(helium.NewNamespace("p", "urn:1"), nil)},
		xpath3.NodeItem{Node: helium.NewNamespaceNodeWrapper(helium.NewNamespace("p", "urn:2"), nil)},
	}

	tests := []struct {
		name string
		expr string
		want string
	}{
		{"union", `string-join(($ns | ()) ! string(), ',')`, "urn:1,urn:2"},
		{"union self", `string-join(($ns | $ns[1]) ! string(), ',')`, "urn:1,urn:2"},
		{"intersect", `string-join(($ns intersect $ns[2]) ! string(), ',')`, "urn:2"},
		{"except", `string-join(($ns except $ns[2]) ! string(), ',')`, "urn:1"},
		{"path", `string-join($ns/. ! string(), ',')`, "urn:1,urn:2"},
		{"path in explicit order", `string-join(reverse($ns)/. ! string(), ',')`, "urn:2,urn:1"},
		{"is", `string($ns[1] is $ns[2])`, wantFalse},
		{"is self", `string($ns[1] is $ns[1])`, wantTrue},
		{"generate-id", `string(generate-id($ns[1]) = generate-id($ns[2]))`, wantFalse},
		{"generate-id self", `string(generate-id($ns[1]) = generate-id($ns[1]))`, wantTrue},
		{"element namespace union", `string(count(/*/namespace::* | /*/namespace::*) = count(/*/namespace::*))`,
			wantTrue},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			compiled, err := xpath3.NewCompiler().Compile(tc.expr)
			require.NoError(t, err)
			result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
				Variables(map[string]xpath3.Sequence{"ns": ns}).
				Evaluate(t.Context(), compiled, doc)
			require.NoError(t, err)
			got, ok := result.IsString()
			require.True(t, ok)
			require.Equal(t, tc.want, got)
		})
	}
}
