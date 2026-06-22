package xpath3_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Top-level JSON scalars route through jsonToXDM's scalar branches
// (bool, json.Number integer/float, null) via the parse-json streaming decoder.
func TestParseJSON_TopLevelScalars(t *testing.T) {
	cases := []struct {
		json string
		want string
	}{
		{`42`, "42"},         // integer
		{`1.5`, want1Dot5},   // float (has '.')
		{`1e3`, "1000"},      // float (has 'e')
		{`true`, wantTrue},   // boolean
		{`false`, wantFalse}, // boolean
		{`"str"`, "str"},     // string
	}
	for _, tc := range cases {
		expr := `parse-json("` + escapeForXPathString(tc.json) + `")`
		r, err := evaluate(t.Context(), nil, expr)
		require.NoError(t, err, expr)
		require.Equal(t, tc.want, r.StringValue(), expr)
	}

	// JSON null -> empty sequence.
	r, err := evaluate(t.Context(), nil, `parse-json("null")`)
	require.NoError(t, err)
	require.Equal(t, "", r.StringValue())
}

// escapeForXPathString doubles embedded double-quotes for an XPath "..." literal.
func escapeForXPathString(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == '"' {
			out = append(out, '"', '"')
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

// parse-json over nested structures with mixed scalar leaves exercises the
// array / object construction plus the scalar leaf conversions.
func TestParseJSON_NestedStructures(t *testing.T) {
	r, err := evaluate(t.Context(), nil,
		`parse-json('{"n": 1, "f": 2.5, "b": true, "z": null, "s": "x", "a": [1, 2, 3]}')?n`)
	require.NoError(t, err)
	require.Equal(t, "1", r.StringValue())

	r, err = evaluate(t.Context(), nil,
		`parse-json('[1, 2.5, true, null, "x"]')?1`)
	require.NoError(t, err)
	require.Equal(t, "1", r.StringValue())

	// xml-to-json over a parsed structure.
	r, err = evaluate(t.Context(), nil,
		`xml-to-json(json-to-xml('{"a": [1, true, null, "s"]}'))`)
	require.NoError(t, err)
	require.Contains(t, r.StringValue(), "\"a\"")
}
