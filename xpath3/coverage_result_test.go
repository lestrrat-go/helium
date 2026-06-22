package xpath3_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResult_StringValue(t *testing.T) {
	// Empty sequence => "".
	r, err := evaluate(t.Context(), nil, `()`)
	require.NoError(t, err)
	require.Equal(t, "", r.StringValue())

	// Single atomic.
	r, err = evaluate(t.Context(), nil, `42`)
	require.NoError(t, err)
	require.Equal(t, "42", r.StringValue())

	// Multi-item sequence => space-joined values (loop branch).
	r, err = evaluate(t.Context(), nil, `(1, 2, 3)`)
	require.NoError(t, err)
	require.Equal(t, "1 2 3", r.StringValue())

	r, err = evaluate(t.Context(), nil, `("a", "b")`)
	require.NoError(t, err)
	require.Equal(t, "a b", r.StringValue())

	// Single-node result returns the node string value directly.
	doc := mustParseXML(t, "<root>text-content</root>")
	root := doc.DocumentElement()
	r, err = evaluate(t.Context(), root, `.`)
	require.NoError(t, err)
	require.Equal(t, "text-content", r.StringValue())
}

func TestResult_TypePredicates(t *testing.T) {
	// IsNumber across the numeric type hierarchy.
	for _, expr := range []string{`1`, `1.5`, `xs:double("2.0")`, `xs:float("3.0")`} {
		r, err := evaluate(t.Context(), nil, expr)
		require.NoError(t, err)
		_, ok := r.IsNumber()
		require.True(t, ok, expr)
	}

	// A string is not a number.
	r, err := evaluate(t.Context(), nil, `"x"`)
	require.NoError(t, err)
	_, ok := r.IsNumber()
	require.False(t, ok)

	// IsString / IsBoolean.
	s, ok := r.IsString()
	require.True(t, ok)
	require.Equal(t, "x", s)

	r, err = evaluate(t.Context(), nil, `true()`)
	require.NoError(t, err)
	b, ok := r.IsBoolean()
	require.True(t, ok)
	require.True(t, b)

	// IsAtomic / Atomics.
	r, err = evaluate(t.Context(), nil, `(1, 2)`)
	require.NoError(t, err)
	require.False(t, r.IsAtomic())
	atoms, err := r.Atomics()
	require.NoError(t, err)
	require.Len(t, atoms, 2)

	r, err = evaluate(t.Context(), nil, `1`)
	require.NoError(t, err)
	require.True(t, r.IsAtomic())

	// IsNodeSet / Nodes.
	doc := mustParseXML(t, "<root><a/><b/></root>")
	root := doc.DocumentElement()
	r, err = evaluate(t.Context(), root, `*`)
	require.NoError(t, err)
	require.True(t, r.IsNodeSet())
	nodes, err := r.Nodes()
	require.NoError(t, err)
	require.Len(t, nodes, 2)

	// A non-node result is not a node set; Nodes returns ErrNotNodeSet.
	r, err = evaluate(t.Context(), root, `1`)
	require.NoError(t, err)
	require.False(t, r.IsNodeSet())
	_, err = r.Nodes()
	require.Error(t, err)
}
