package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestNilledMock drives the schema-aware nilled property through a hand-supplied
// NilledElements set (no XSD validation), isolating the xpath3 plumbing:
// fn:nilled reports the property, fn:data of a nilled element yields the empty
// sequence, and element(name, type) instance-of excludes a nilled element while
// element(name, type?) still matches it.
func TestNilledMock(t *testing.T) {
	doc := mustParseXML(t, `<twig>42</twig>`)
	twig := doc.DocumentElement()

	nilledEval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		TypeAnnotations(map[helium.Node]string{twig: xpath3.TypeInteger}).
		NilledElements(map[helium.Node]struct{}{twig: {}})

	plainEval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		TypeAnnotations(map[helium.Node]string{twig: xpath3.TypeInteger})

	t.Run("fn:nilled true for a nilled element", func(t *testing.T) {
		seq := evalExprWithEval(t, nilledEval, doc, `nilled(/*)`)
		require.Equal(t, 1, seq.Len())
		b, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, true, b.BooleanVal())
	})

	t.Run("fn:nilled false for a non-nilled element", func(t *testing.T) {
		seq := evalExprWithEval(t, plainEval, doc, `nilled(/*)`)
		require.Equal(t, 1, seq.Len())
		b, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, false, b.BooleanVal())
	})

	t.Run("fn:nilled empty sequence for a non-element", func(t *testing.T) {
		// fn:nilled() of a non-element is the empty sequence, which the evaluator
		// represents as a typed-nil Sequence.
		seq := evalExprWithEval(t, nilledEval, doc, `nilled(/*/text())`)
		require.True(t, seq == nil || seq.Len() == 0)
	})

	t.Run("data() of a nilled element is the empty sequence", func(t *testing.T) {
		seq := evalExprWithEval(t, nilledEval, doc, `data(/*)`)
		require.Equal(t, 0, seq.Len())
	})

	t.Run("data() of a non-nilled element atomizes normally", func(t *testing.T) {
		seq := evalExprWithEval(t, plainEval, doc, `data(/*)`)
		require.Equal(t, 1, seq.Len())
	})

	t.Run("nilled element is not an instance of element(name, type)", func(t *testing.T) {
		seq := evalExprWithEval(t, nilledEval, doc, `/* instance of element(twig, xs:integer)`)
		b, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, false, b.BooleanVal())
	})

	t.Run("nilled element IS an instance of element(name, type?)", func(t *testing.T) {
		seq := evalExprWithEval(t, nilledEval, doc, `/* instance of element(twig, xs:integer?)`)
		b, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, true, b.BooleanVal())
	})

	t.Run("non-nilled element IS an instance of element(name, type)", func(t *testing.T) {
		seq := evalExprWithEval(t, plainEval, doc, `/* instance of element(twig, xs:integer)`)
		b, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, true, b.BooleanVal())
	})
}

// TestNilledXSD exercises the real xsd validator: a nillable element carrying a
// valid xsi:nil="true" is recorded in xsd.NilledElements; wired into the
// evaluator it makes fn:nilled true and fn:data () for that element, while a
// non-nilled instance of the same declaration reports false and atomizes.
func TestNilledXSD(t *testing.T) {
	const schemaSrc = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
	    targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
	  <xs:element name="twig" type="xs:int" nillable="true"/>
	</xs:schema>`

	schema, err := xsd.NewCompiler().Compile(t.Context(), mustParseXML(t, schemaSrc))
	require.NoError(t, err)

	evalFor := func(t *testing.T, src string) (xpath3.Evaluator, *helium.Document) {
		t.Helper()
		ctx := t.Context()
		doc := mustParseXML(t, src)
		ann := make(xsd.TypeAnnotations)
		ne := make(xsd.NilledElements)
		require.NoError(t, xsd.NewValidator(schema).Annotations(&ann).NilledElements(&ne).Validate(ctx, doc))
		nilled := make(map[helium.Node]struct{}, len(ne))
		for elem := range ne {
			nilled[elem] = struct{}{}
		}
		eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			SchemaDeclarations(schema.Declarations()).
			TypeAnnotations(ann).
			NilledElements(nilled)
		return eval, doc
	}

	t.Run("nilled instance", func(t *testing.T) {
		eval, doc := evalFor(t, `<twig xmlns="urn:t" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:nil="true"/>`)

		seq := evalExprWithEval(t, eval, doc, `nilled(/*)`)
		require.Equal(t, 1, seq.Len())
		b := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, true, b.BooleanVal())

		seq = evalExprWithEval(t, eval, doc, `data(/*)`)
		require.Equal(t, 0, seq.Len(), "nilled element has typed value ()")
	})

	t.Run("non-nilled instance", func(t *testing.T) {
		eval, doc := evalFor(t, `<twig xmlns="urn:t">42</twig>`)

		seq := evalExprWithEval(t, eval, doc, `nilled(/*)`)
		require.Equal(t, 1, seq.Len())
		b := seq.Get(0).(xpath3.AtomicValue)
		require.Equal(t, false, b.BooleanVal())

		seq = evalExprWithEval(t, eval, doc, `data(/*)`)
		require.Equal(t, 1, seq.Len())
	})
}
