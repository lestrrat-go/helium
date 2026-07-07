package xpath3_test

import (
	"errors"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// TestOptionValueFlattenBeforeCardinality is the regression guard for the F&O
// 3.1 option (function) conversion rules (§2.5): an xs:string-typed option value
// is atomized FIRST — arrays flatten to their members, so an empty-array member
// contributes nothing — and cardinality is applied AFTER. A raw pre-atomization
// seqLen gate wrongly rejects map{"opt": ([], "value")} with XPTY0004; these
// cases assert the empty-array member flattens away and the option takes effect.
func TestOptionValueFlattenBeforeCardinality(t *testing.T) {
	doc := mustParseXML(t, `<root/>`)

	// fn:json-to-xml duplicates option (the reviewer's case). Atomizing
	// ([], "use-first") yields the single "use-first"; the option applies and the
	// duplicate key resolves to the first value ("1"), not XPTY0004.
	t.Run("json-to-xml duplicates empty-array member flattens", func(t *testing.T) {
		seq := evalExpr(t, doc,
			`json-to-xml('{"a":1,"a":2}', map{"duplicates": ([], "use-first")})//*:number/string()`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, "1", av.StringVal())
	})

	// fn:parse-json duplicates option.
	t.Run("parse-json duplicates empty-array member flattens", func(t *testing.T) {
		seq := evalExpr(t, doc,
			`parse-json('{"a":1,"a":2}', map{"duplicates": ([], "use-last")})?a`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, int64(2), av.IntegerVal())
	})

	// fn:serialize string parameter (item-separator). Atomizing ([], "-") yields
	// the single "-"; the atomic sequence is serialized with that separator.
	t.Run("serialize item-separator empty-array member flattens", func(t *testing.T) {
		seq := evalExpr(t, doc,
			`serialize((1, 2, 3), map{"method": "adaptive", "item-separator": ([], "-")})`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, "1-2-3", av.StringVal())
	})

	// fn:serialize method parameter also flattens (and keeps its type — a plain
	// string "xml" here).
	t.Run("serialize method empty-array member flattens", func(t *testing.T) {
		seq := evalExpr(t, doc,
			`serialize(/root, map{"method": ([], "xml")})`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, "<root/>", av.StringVal())
	})

	// fn:serialize standalone parameter (union(xs:boolean, enum("omit"))) also
	// flattens: ([], "omit") atomizes to the single "omit" enum value.
	t.Run("serialize standalone empty-array member flattens", func(t *testing.T) {
		seq := evalExpr(t, doc,
			`serialize(/root, map{"method": "xml", "omit-xml-declaration": false(), "standalone": ([], "omit")})`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Contains(t, av.StringVal(), "<root/>")
	})
}

// TestOptionValueCardinalityStillEnforced confirms that dropping the raw gate
// did NOT drop cardinality enforcement: a genuinely too-long value (two atoms
// after flattening) is still an XPTY0004 type error.
func TestOptionValueCardinalityStillEnforced(t *testing.T) {
	doc := mustParseXML(t, `<root/>`)

	requireXPTY0004 := func(t *testing.T, expr string) {
		t.Helper()
		compiled, err := xpath3.NewCompiler().Compile(expr)
		require.NoError(t, err)
		_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.True(t, errors.As(err, &xerr), "want *xpath3.XPathError, got %T: %v", err, err)
		require.Equal(t, "XPTY0004", xerr.Code)
	}

	t.Run("json-to-xml duplicates two atoms XPTY0004", func(t *testing.T) {
		requireXPTY0004(t, `json-to-xml('{}', map{"duplicates": ("use-first", "reject")})`)
	})

	t.Run("serialize item-separator two atoms XPTY0004", func(t *testing.T) {
		requireXPTY0004(t, `serialize((1, 2), map{"method": "adaptive", "item-separator": ("-", "+")})`)
	})
}
