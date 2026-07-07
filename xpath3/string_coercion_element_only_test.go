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

	// The xs:string?-argument URI/codepoint functions share the same
	// atomize-then-count coercion: an empty-array member must flatten away
	// rather than count as a second item (XPTY0004).
	uriCases := []struct {
		name, expr, want string
	}{
		{"encode-for-uri", `encode-for-uri(([], "a b"))`, "a%20b"},
		{"iri-to-uri", `iri-to-uri(([], "a b"))`, "a%20b"},
		{"escape-html-uri", `escape-html-uri(([], "a b"))`, "a b"},
	}
	for _, tc := range uriCases {
		t.Run(tc.name+" empty array member flattens away", func(t *testing.T) {
			seq := evalExpr(t, doc, tc.expr)
			require.Equal(t, 1, seq.Len())
			av, ok := seq.Get(0).(xpath3.AtomicValue)
			require.True(t, ok)
			require.Equal(t, tc.want, av.StringVal())
		})
	}

	t.Run("codepoint-equal empty array member flattens away", func(t *testing.T) {
		seq := evalExpr(t, doc, `codepoint-equal(([], "abc"), "abc")`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, true, av.Value)
	})
}

// TestDocFamilyCardinalityFlattens guards the docURIArg family (fn:doc,
// fn:doc-available, fn:collection, fn:uri-collection) against the raw
// pre-atomization cardinality gate. Their xs:string? URI argument must be
// atomized FIRST — an empty-array member flattens away — THEN cardinality is
// applied, so doc-available(([], "x")) resolves the single URI (yielding false
// without a resolver) instead of raising XPTY0004. A genuinely too-long
// argument (two atoms after flattening) still raises XPTY0004.
func TestDocFamilyCardinalityFlattens(t *testing.T) {
	doc := mustParseXML(t, `<root/>`)

	t.Run("doc-available empty array member flattens away", func(t *testing.T) {
		seq := evalExpr(t, doc, `doc-available(([], "http://example.com/x"))`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, false, av.Value) // no resolver ⇒ not available, NOT XPTY0004
	})

	t.Run("doc-available two atoms still XPTY0004", func(t *testing.T) {
		compiled, err := xpath3.NewCompiler().Compile(`doc-available(("a", "b"))`)
		require.NoError(t, err)
		_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.True(t, errors.As(err, &xerr))
		require.Equal(t, "XPTY0004", xerr.Code)
	})
}

// TestDynamicCallElementOnlyRaisesFOTY0012 verifies that atomizing an
// element-only-typed node against an xs:string? parameter surfaces the real
// dynamic error err:FOTY0012 (the node has no typed value) rather than a
// generic XPTY0004, across every function-item invocation path: a named
// function reference (invoked via fn:for-each), fn:function-lookup, an inline
// function parameter, and an inline function return type. Each path previously
// collapsed the typed error into XPTY0004 by using the boolean
// coerceToSequenceType; they now route through the error-propagating coercion.
func TestDynamicCallElementOnlyRaisesFOTY0012(t *testing.T) {
	doc := mustParseXML(t, `<root><child>hi</child></root>`)
	root := doc.DocumentElement()

	decls := contentKindDecls{kinds: map[string]xpath3.ContentTypeKind{
		xpath3.QAnnotation("urn:t", "rootType"): xpath3.ContentTypeElementOnly,
	}}
	newEval := func() xpath3.Evaluator {
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
		_, err = newEval().Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.True(t, errors.As(err, &xerr), "want *xpath3.XPathError, got %T: %v", err, err)
		require.Equal(t, "FOTY0012", xerr.Code)
	}

	// Named function reference invoked through a higher-order function.
	t.Run("named ref via for-each", func(t *testing.T) {
		requireFOTY0012(t, `for-each(/*, upper-case#1)`)
	})

	// fn:function-lookup dynamic invocation.
	t.Run("function-lookup", func(t *testing.T) {
		requireFOTY0012(t, `function-lookup(QName('http://www.w3.org/2005/xpath-functions','upper-case'), 1)(/*)`)
	})

	// Inline function parameter coercion.
	t.Run("inline function parameter", func(t *testing.T) {
		requireFOTY0012(t, `(function($x as xs:string?) { $x })(/*)`)
	})

	// Inline function return-type coercion.
	t.Run("inline function return type", func(t *testing.T) {
		requireFOTY0012(t, `(function($x) as xs:string? { $x })(/*)`)
	})
}
