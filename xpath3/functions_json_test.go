package xpath3_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
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

// fn:json-to-xml with validate:true() and duplicates:'retain' cannot succeed:
// validation against the json result schema requires unique keys, so the
// combination is a dynamic error err:FOJS0005 (QT3 json-to-xml-error-042).
func TestJSONToXML_ValidateRetainDuplicates_FOJS0005(t *testing.T) {
	_, err := evaluate(t.Context(), nil,
		`json-to-xml('{"A":1, "A":2}', map{'validate':true(), 'duplicates':'retain'})`)
	require.Error(t, err)
	var xerr *xpath3.XPathError
	require.True(t, errors.As(err, &xerr), "want *xpath3.XPathError, got %T: %v", err, err)
	require.Equal(t, "FOJS0005", xerr.Code)
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

// fn:parse-json must thread the evaluator's DoS budget into the streaming JSON
// parser so an oversized or deeply nested JSON string cannot build an unbounded
// in-memory map/array structure that bypasses the engine's limits. Each case
// fails (returns a value, no error) before the budget is threaded through
// parseJSONToken and passes after.
func TestParseJSON_Bounded(t *testing.T) {
	// Deeply nested arrays beyond the recursion limit must be rejected rather
	// than recursing unbounded over the goroutine stack.
	t.Run("deep nesting", func(t *testing.T) {
		depth := xpath3.DefaultMaxRecursionDepth + 1
		nested := strings.Repeat("[", depth) + strings.Repeat("]", depth)
		expr := `parse-json('` + nested + `')`
		_, err := evaluate(t.Context(), nil, expr)
		require.Error(t, err, "deeply nested JSON must be rejected")
		require.True(t, errors.Is(err, xpath3.ErrRecursionLimit),
			"expected ErrRecursionLimit, got %v", err)
	})

	// A wide array exceeding the configured node-set limit must be rejected.
	t.Run("wide array", func(t *testing.T) {
		var b strings.Builder
		b.WriteByte('[')
		for i := range 50 {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteByte('0')
		}
		b.WriteByte(']')
		compiled, err := xpath3.NewCompiler().Compile(`parse-json('` + b.String() + `')`)
		require.NoError(t, err)
		eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).MaxNodesForTesting(10)
		_, err = eval.Evaluate(t.Context(), compiled, nil)
		require.Error(t, err, "wide JSON array must be rejected")
		require.True(t, errors.Is(err, xpath3.ErrNodeSetLimit),
			"expected ErrNodeSetLimit, got %v", err)
	})

	// Construction must honor context cancellation.
	t.Run("cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := evaluate(ctx, nil, `parse-json('[1, 2, 3, 4, 5]')`)
		require.Error(t, err, "cancelled context must abort JSON construction")
		require.True(t, errors.Is(err, context.Canceled),
			"expected context.Canceled, got %v", err)
	})
}

// Per F&O 3.1 §17.5 JSON has a single number type, so every JSON number in a
// fn:parse-json / fn:json-doc result is an xs:double — including integral
// values such as 0 and -0 (QT3 json-doc-032, json-doc-033). The value is
// preserved (0 stays 0), only the type is xs:double, so deep-equal to the
// xs:integer literal 0 still holds via numeric promotion.
func TestParseJSON_NumbersAreDouble(t *testing.T) {
	cases := []struct {
		name string
		json string
	}{
		{"zero", "0"},           // json-doc-032
		{"negative zero", "-0"}, // json-doc-033
		{"positive integer", "1"},
		{"negative integer", "-5"},
		{"non-integral", "2.5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expr := `parse-json("` + escapeForXPathString(tc.json) + `") instance of xs:double`
			r, err := evaluate(t.Context(), nil, expr)
			require.NoError(t, err, expr)
			require.Equal(t, wantTrue, r.StringValue(), "expected xs:double for %s", tc.json)
		})
	}

	// Value is preserved for the integral cases: deep-equal to the xs:integer
	// literal 0 still holds (numeric promotion), matching json-doc-032's
	// assert-deep-eq 0.
	r, err := evaluate(t.Context(), nil, `deep-equal(parse-json("0"), 0)`)
	require.NoError(t, err)
	require.Equal(t, wantTrue, r.StringValue())

	r, err = evaluate(t.Context(), nil, `deep-equal(parse-json("-0"), 0)`)
	require.NoError(t, err)
	require.Equal(t, wantTrue, r.StringValue())
}

// fn:json-to-xml shares the same streaming parser and must be bounded too.
func TestJSONToXML_DeepNestingBounded(t *testing.T) {
	depth := xpath3.DefaultMaxRecursionDepth + 1
	nested := strings.Repeat("[", depth) + strings.Repeat("]", depth)
	_, err := evaluate(t.Context(), nil, `json-to-xml('`+nested+`')`)
	require.Error(t, err, "deeply nested JSON must be rejected by json-to-xml")
	require.True(t, errors.Is(err, xpath3.ErrRecursionLimit),
		"expected ErrRecursionLimit, got %v", err)
}
