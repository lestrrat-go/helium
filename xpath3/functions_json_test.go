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

// fn:json-to-xml retains a JSON number's EXACT lexical form in the <number>
// element (F&O 3.1, W3C bug 28179 / QT3 json-to-xml-030/031/033) — the opposite
// of fn:parse-json, which canonicalizes every number to xs:double. In
// particular a large integer must NOT be rendered in scientific notation, and
// -0 must be retained. This locks the split between the two consumers of the
// shared JSON parser so the parse-json xs:double change cannot leak into
// json-to-xml's number representation.
func TestJSONToXML_NumbersRetainLexicalForm(t *testing.T) {
	cases := []struct {
		name string
		json string
		want string
	}{
		{"large integer no scientific notation", `{"a": 1000000000000}`, "1000000000000"},
		{"negative zero retained", `-0`, "-0"},
		{"upper-case exponent retained", `23E0`, "23E0"},
		{"lower-case exponent with sign retained", `0.23e+02`, "0.23e+02"},
		{"plain zero", `0`, "0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expr := `json-to-xml("` + escapeForXPathString(tc.json) + `")//*:number/string()`
			r, err := evaluate(t.Context(), nil, expr)
			require.NoError(t, err, expr)
			require.Equal(t, tc.want, r.StringValue(), "json-to-xml must retain lexical form for %s", tc.json)
		})
	}
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

// Every fn:json-to-xml result element is in the xpath-functions namespace
// (F&O 3.1 §17.6.1), and the declaration is present on every element so that a
// descendant selected by an XPath step still serializes with its xmlns
// declaration (QT3 json-to-xml-011/013/014).
func TestJSONToXML_Namespace(t *testing.T) {
	const fnNS = "http://www.w3.org/2005/xpath-functions"

	r, err := evaluate(t.Context(), nil, `namespace-uri(json-to-xml('["x"]')/*)`)
	require.NoError(t, err)
	require.Equal(t, fnNS, r.StringValue(), "root element must be in the fn namespace")

	// A descendant string selected on its own must serialize with the xmlns
	// declaration it inherits, not lose it.
	r, err = evaluate(t.Context(), nil,
		`serialize(json-to-xml('["data"]')//*:string, map{'omit-xml-declaration':true()})`)
	require.NoError(t, err)
	require.Contains(t, r.StringValue(), `xmlns="`+fnNS+`"`,
		"extracted descendant must carry the fn namespace declaration")
	require.Equal(t, fnNS, mustNSURI(t, `namespace-uri(json-to-xml('{"k":"v"}')//*:string)`),
		"descendant string must be in the fn namespace")
}

func mustNSURI(t *testing.T, expr string) string {
	t.Helper()
	r, err := evaluate(t.Context(), nil, expr)
	require.NoError(t, err, expr)
	return r.StringValue()
}

// fn:json-to-xml preserves duplicate JSON object keys by default: the default
// value of the duplicates option is "retain" (F&O 3.1 §17.6.1: "if validate is
// true then reject, otherwise retain"), unlike fn:parse-json whose default is
// use-first (QT3 json-to-xml-018).
func TestJSONToXML_DuplicateKeysRetainedByDefault(t *testing.T) {
	r, err := evaluate(t.Context(), nil,
		`count(json-to-xml('{"a":3, "b":4, "a":5}')//*:number)`)
	require.NoError(t, err)
	require.Equal(t, "3", r.StringValue(), "all three entries, including the duplicate key, must be retained")

	r, err = evaluate(t.Context(), nil,
		`string-join(json-to-xml('{"a":3, "b":4, "a":5}')//*:number/@key, ",")`)
	require.NoError(t, err)
	require.Equal(t, "a,b,a", r.StringValue(), "duplicate key must appear twice in document order")
}

// The escaped / escaped-key attributes are emitted ONLY under escape=true() and
// ONLY when the string value / key attribute actually contains a backslash escape
// (F&O 3.1 §17.6.1). A value or key without a backslash carries no such
// attribute (QT3 fo-test-fn-json-to-xml-004 / json-to-xml-021 / -024).
func TestJSONToXML_EscapedAttributesConditional(t *testing.T) {
	// escape=true(): value "x" decodes to a lone backslash (\), so escaped="true";
	// value "y" is "%", no backslash, so no escaped attribute.
	cases := []struct {
		desc string
		expr string
		want string
	}{
		{
			"value with backslash gets escaped=true",
			`json-to-xml('{"x":"\\", "y":"%"}', map{'escape':true()})//*:string[@key='x']/@escaped`,
			wantTrue,
		},
		{
			"value without backslash gets no escaped attribute",
			`string(json-to-xml('{"x":"\\", "y":"%"}', map{'escape':true()})//*:string[@key='y']/@escaped)`,
			"",
		},
		{
			"key with backslash gets escaped-key=true",
			`json-to-xml('{"a\\":3}', map{'escape':true()})//*:number/@escaped-key`,
			wantTrue,
		},
		{
			"key without backslash gets no escaped-key attribute",
			`string(json-to-xml('{"a":3}', map{'escape':true()})//*:number/@escaped-key)`,
			"",
		},
		{
			"escape=false() never emits escaped even when the key contains a backslash",
			`string(json-to-xml('{"a\\":3}', map{'escape':false()})//*:number/@escaped-key)`,
			"",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			r, err := evaluate(t.Context(), nil, tc.expr)
			require.NoError(t, err, tc.expr)
			require.Equal(t, tc.want, r.StringValue(), tc.expr)
		})
	}
}
