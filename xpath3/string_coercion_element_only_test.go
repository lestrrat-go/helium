package xpath3_test

import (
	"errors"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// TestStringArgElementOnlyRaisesFOTY0012 verifies that atomizing an
// element-only-typed node as an xs:string?-argument raises err:FOTY0012 (the
// element has no typed value), matching fn:data. Covers QT3
// fn-string-length-23 (/*/string-length(.)) and fn-normalize-space-24
// (/*/normalize-space(.)), and — for the generic function-conversion route —
// any other xs:string?-argument function (fn:upper-case).
func TestStringArgElementOnlyRaisesFOTY0012(t *testing.T) {
	doc := mustParseXML(t, `<root><child>hi</child></root>`)
	root := doc.DocumentElement()
	child := root.FirstChild() // <child>

	decls := contentKindDecls{kinds: map[string]xpath3.ContentTypeKind{
		xpath3.QAnnotation("urn:t", "rootType"):  xpath3.ContentTypeElementOnly,
		xpath3.QAnnotation("urn:t", "mixedType"): xpath3.ContentTypeMixed,
	}}

	elementOnlyEval := func() xpath3.Evaluator {
		return xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			TypeAnnotations(map[helium.Node]string{
				root: xpath3.QAnnotation("urn:t", "rootType"),
			}).
			SchemaDeclarations(decls)
	}

	requireFOTY0012 := func(t *testing.T, expr string) {
		t.Helper()
		compiled, err := xpath3.NewCompiler().Compile(expr)
		require.NoError(t, err)
		_, err = elementOnlyEval().Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.True(t, errors.As(err, &xerr), "want *xpath3.XPathError, got %T: %v", err, err)
		require.Equal(t, "FOTY0012", xerr.Code)
	}

	// fn:string-length uses seqToStringErr / AtomizeSequence.
	t.Run("string-length element-only raises FOTY0012", func(t *testing.T) {
		requireFOTY0012(t, `string-length(/*)`)
	})

	// fn:normalize-space uses coerceArgToString / atomizeStream.
	t.Run("normalize-space element-only raises FOTY0012", func(t *testing.T) {
		requireFOTY0012(t, `normalize-space(/*)`)
	})

	// Spec-consistency for the generic function-conversion route: any
	// xs:string?-arg function atomizing an element-only node raises FOTY0012.
	t.Run("upper-case element-only raises FOTY0012", func(t *testing.T) {
		requireFOTY0012(t, `upper-case(/*)`)
	})

	// A mixed-content node has a typed value (xs:untypedAtomic) → no FOTY0012.
	t.Run("mixed-content node atomizes normally", func(t *testing.T) {
		eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			TypeAnnotations(map[helium.Node]string{
				root: xpath3.QAnnotation("urn:t", "mixedType"),
			}).
			SchemaDeclarations(decls)
		seq := evalExprWithEval(t, eval, doc, `string-length(/*)`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, int64(2), av.Value) // len("hi")
	})

	// An unannotated node atomizes to its string value unchanged.
	t.Run("unannotated child atomizes normally", func(t *testing.T) {
		_ = child
		seq := evalExprWithEval(t, elementOnlyEval(), doc, `string-length(/*/child)`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, int64(2), av.Value) // "hi"
	})
}

// TestStringArgCardinalityPreserved is the regression guard for the xs:string?
// cardinality semantics: atomization happens FIRST (flattening arrays, so an
// empty-array member contributes nothing), THEN cardinality is applied. An
// earlier attempt that exposed the raw item stream broke this by counting an
// empty array as an item pre-atomization. normalize-space(([], " x ")) must
// therefore be "x", not XPTY0004.
func TestStringArgCardinalityPreserved(t *testing.T) {
	doc := mustParseXML(t, `<root/>`)

	// Non-schema-aware path (byte-identical to before).
	t.Run("empty array member flattens away", func(t *testing.T) {
		seq := evalExpr(t, doc, `normalize-space(([], " x "))`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, "x", av.StringVal())
	})

	// Same under a schema-aware evaluator (the content-kind-aware atomization
	// route must still flatten arrays before applying cardinality).
	t.Run("empty array member flattens away (schema-aware)", func(t *testing.T) {
		decls := contentKindDecls{kinds: map[string]xpath3.ContentTypeKind{}}
		eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			SchemaDeclarations(decls)
		seq := evalExprWithEval(t, eval, doc, `normalize-space(([], " x "))`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, "x", av.StringVal())
	})
}
