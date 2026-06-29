package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func stExactlyOne(typ string) xpath3.SequenceType {
	return xpath3.SequenceType{
		ItemTest:   xpath3.AtomicOrUnionType{Prefix: "xs", Name: typ},
		Occurrence: xpath3.OccurrenceExactlyOne,
	}
}

func TestEvalState_SetPositionSize(t *testing.T) {
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)

	ctxItem := xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: int64(1)}

	compiledPos, err := xpath3.NewCompiler().Compile(`position()`)
	require.NoError(t, err)
	state := eval.NewEvalState(nil)
	state.SetContextItem(ctxItem)
	state.SetPosition(3)
	state.SetSize(7)

	res, err := compiledPos.EvaluateReuse(t.Context(), state, nil)
	require.NoError(t, err)
	n, ok := res.IsNumber()
	require.True(t, ok)
	require.Equal(t, float64(3), n)

	compiledLast, err := xpath3.NewCompiler().Compile(`last()`)
	require.NoError(t, err)
	state2 := eval.NewEvalState(nil)
	state2.SetContextItem(ctxItem)
	state2.SetSize(7)
	res, err = compiledLast.EvaluateReuse(t.Context(), state2, nil)
	require.NoError(t, err)
	n, ok = res.IsNumber()
	require.True(t, ok)
	require.Equal(t, float64(7), n)
}

func TestEvalState_SetContextItem(t *testing.T) {
	compiled, err := xpath3.NewCompiler().Compile(`.`)
	require.NoError(t, err)

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	state := eval.NewEvalState(nil)
	state.SetContextItem(xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: int64(55)})

	res, err := compiled.EvaluateReuse(t.Context(), state, nil)
	require.NoError(t, err)
	require.Equal(t, 1, res.Sequence().Len())
	av, ok := res.Sequence().Get(0).(xpath3.AtomicValue)
	require.True(t, ok)
	require.Equal(t, int64(55), av.Value)
}

func TestMatchesSequenceType(t *testing.T) {
	intSeq := atomicSeq(intAtomic(1))
	require.True(t, xpath3.MatchesSequenceType(intSeq, stExactlyOne("integer")))
	require.False(t, xpath3.MatchesSequenceType(intSeq, stExactlyOne("string")))

	// A two-item sequence violates the exactly-one cardinality.
	twoSeq := xpath3.ItemSlice{intAtomic(1), intAtomic(2)}
	require.False(t, xpath3.MatchesSequenceType(twoSeq, stExactlyOne("integer")))
}

func TestCoerceToSequenceType(t *testing.T) {
	// integer promotes to double under function coercion rules.
	intSeq := atomicSeq(intAtomic(3))
	out, ok := xpath3.CoerceToSequenceType(intSeq, stExactlyOne("double"))
	require.True(t, ok)
	require.Equal(t, 1, out.Len())

	// A string does not coerce to integer.
	strSeq := atomicSeq(strAtomic("notnum"))
	_, ok = xpath3.CoerceToSequenceType(strSeq, stExactlyOne("integer"))
	require.False(t, ok)
}

func TestCheckFunctionParamCompat(t *testing.T) {
	// FunctionTest with AnyFunction => always compatible.
	fi := xpath3.FunctionItem{
		Arity:      1,
		ParamTypes: []xpath3.SequenceType{stExactlyOne("integer")},
	}
	anyFnST := xpath3.SequenceType{
		ItemTest:   xpath3.FunctionTest{AnyFunction: true},
		Occurrence: xpath3.OccurrenceExactlyOne,
	}
	require.True(t, xpath3.CheckFunctionParamCompat(fi, anyFnST))

	// Matching arity with compatible (identical) param types.
	typedFnST := xpath3.SequenceType{
		ItemTest: xpath3.FunctionTest{
			ParamTypes: []xpath3.SequenceType{stExactlyOne("integer")},
		},
		Occurrence: xpath3.OccurrenceExactlyOne,
	}
	require.True(t, xpath3.CheckFunctionParamCompat(fi, typedFnST))

	// Mismatched arity => incompatible.
	mismatchFnST := xpath3.SequenceType{
		ItemTest: xpath3.FunctionTest{
			ParamTypes: []xpath3.SequenceType{stExactlyOne("integer"), stExactlyOne("integer")},
		},
		Occurrence: xpath3.OccurrenceExactlyOne,
	}
	require.False(t, xpath3.CheckFunctionParamCompat(fi, mismatchFnST))
}

// stMap / stArray build map(K,V) and array(T) SequenceTypes, driving
// isItemTypeSubtype's MapTest/ArrayTest branches via CheckFunctionParamCompat.
func stMap(keyT, valT string) xpath3.SequenceType {
	return xpath3.SequenceType{
		ItemTest: xpath3.MapTest{
			KeyType: xpath3.AtomicOrUnionType{Prefix: "xs", Name: keyT},
			ValType: stExactlyOne(valT),
		},
		Occurrence: xpath3.OccurrenceExactlyOne,
	}
}

func stArray(memberT string) xpath3.SequenceType {
	return xpath3.SequenceType{
		ItemTest: xpath3.ArrayTest{
			MemberType: stExactlyOne(memberT),
		},
		Occurrence: xpath3.OccurrenceExactlyOne,
	}
}

func TestCheckFunctionParamCompat_NestedTypes(t *testing.T) {
	// Function declares a map(string, integer) parameter; the target test asks
	// for the same -> compatible (drives isItemTypeSubtype MapTest branch).
	fiMap := xpath3.FunctionItem{
		Arity:      1,
		ParamTypes: []xpath3.SequenceType{stMap("string", "integer")},
	}
	mapFnST := xpath3.SequenceType{
		ItemTest:   xpath3.FunctionTest{ParamTypes: []xpath3.SequenceType{stMap("string", "integer")}},
		Occurrence: xpath3.OccurrenceExactlyOne,
	}
	require.True(t, xpath3.CheckFunctionParamCompat(fiMap, mapFnST))

	// Array parameter compatibility (drives ArrayTest branch).
	fiArr := xpath3.FunctionItem{
		Arity:      1,
		ParamTypes: []xpath3.SequenceType{stArray("integer")},
	}
	arrFnST := xpath3.SequenceType{
		ItemTest:   xpath3.FunctionTest{ParamTypes: []xpath3.SequenceType{stArray("integer")}},
		Occurrence: xpath3.OccurrenceExactlyOne,
	}
	require.True(t, xpath3.CheckFunctionParamCompat(fiArr, arrFnST))

	// Mismatched atomic param types -> incompatible (drives AtomicOrUnionType branch).
	fiInt := xpath3.FunctionItem{
		Arity:      1,
		ParamTypes: []xpath3.SequenceType{stExactlyOne("integer")},
	}
	strFnST := xpath3.SequenceType{
		ItemTest:   xpath3.FunctionTest{ParamTypes: []xpath3.SequenceType{stExactlyOne("string")}},
		Occurrence: xpath3.OccurrenceExactlyOne,
	}
	require.False(t, xpath3.CheckFunctionParamCompat(fiInt, strFnST))
}

func TestMatchesSequenceType_MapArray(t *testing.T) {
	// A real map value matched against map(string, integer).
	mapSeq, err := evalSeq(t, `map { "a": 1, "b": 2 }`)
	require.NoError(t, err)
	require.True(t, xpath3.MatchesSequenceType(mapSeq, stMap("string", "integer")))

	arrSeq, err := evalSeq(t, `[1, 2, 3]`)
	require.NoError(t, err)
	require.True(t, xpath3.MatchesSequenceType(arrSeq, stArray("integer")))
}

func evalSeq(t *testing.T, expr string) (xpath3.Sequence, error) {
	t.Helper()
	r, err := evaluate(t.Context(), nil, expr)
	if err != nil {
		return nil, err
	}
	return r.Sequence(), nil
}

// instanceOf compiles and evaluates an `instance of` expression against a
// context node, returning its boolean result. It exercises matchesItemType /
// isItemTypeSubtype / matchNodeTest across many item-type shapes.
func instanceOf(t *testing.T, ctxXML, expr string) bool {
	t.Helper()
	doc := mustParseXML(t, ctxXML)
	root := doc.DocumentElement()
	r, err := evaluate(t.Context(), root, expr)
	require.NoError(t, err, expr)
	b, ok := r.IsBoolean()
	require.True(t, ok, "expected boolean for %q", expr)
	return b
}

func TestInstanceOf_ItemTypes(t *testing.T) {
	const xml = `<root att="v"><child/><!--c--><?pi data?></root>`

	cases := []struct {
		expr   string
		expect bool
	}{
		// atomic types & numeric hierarchy.
		{`1 instance of xs:integer`, true},
		{`1 instance of xs:decimal`, true},
		{`1 instance of xs:double`, false},
		{`1.5 instance of xs:decimal`, true},
		{`"x" instance of xs:string`, true},
		{`"x" instance of xs:integer`, false},
		{`true() instance of xs:boolean`, true},
		// cardinality.
		{`() instance of item()*`, true},
		{`() instance of item()`, false},
		{`() instance of item()?`, true},
		{`(1, 2) instance of xs:integer+`, true},
		{`(1, 2) instance of xs:integer`, false},
		{`1 instance of item()`, true},
		// node kind tests on the context tree.
		{`. instance of element()`, true},
		{`. instance of node()`, true},
		{`. instance of attribute()`, false},
		{`child::child[1] instance of element()`, true},
		{`@att instance of attribute()`, true},
		{`@att instance of node()`, true},
		{`comment() instance of comment()`, true},
		{`processing-instruction() instance of processing-instruction()`, true},
		{`. instance of element(root)`, true},
		{`. instance of element(other)`, false},
		// function / map / array item types.
		{`fn:abs#1 instance of function(*)`, true},
		{`map { "a": 1 } instance of map(*)`, true},
		{`[1, 2] instance of array(*)`, true},
		{`map { "a": 1 } instance of array(*)`, false},
		{`function($x) { $x } instance of function(*)`, true},
	}

	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			require.Equal(t, tc.expect, instanceOf(t, xml, tc.expr))
		})
	}
}

func TestTreatAs(t *testing.T) {
	// treat as success returns the value; failure raises XPDY0050/XPTY0004.
	r, err := evaluate(t.Context(), nil, `1 treat as xs:integer`)
	require.NoError(t, err)
	n, ok := r.IsNumber()
	require.True(t, ok)
	require.Equal(t, float64(1), n)

	_, err = evaluate(t.Context(), nil, `"x" treat as xs:integer`)
	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
}

// instance of against typed map / array / function / document-node tests
// exercises matchesItemType and isItemTypeSubtype's typed branches (key/value,
// member, param/return, inner node tests).
func TestInstanceOf_TypedItemTests(t *testing.T) {
	const xml = `<root><child/></root>`
	doc := mustParseXML(t, xml)
	root := doc.DocumentElement()

	cases := []struct {
		expr   string
		expect bool
	}{
		{`map { "a": 1 } instance of map(xs:string, xs:integer)`, true},
		{`map { "a": "v" } instance of map(xs:string, xs:integer)`, false},
		{`[1, 2] instance of array(xs:integer)`, true},
		{`["x"] instance of array(xs:integer)`, false},
		{`fn:abs#1 instance of function(*)`, true},
		{`function($x as xs:integer) as xs:integer { $x } instance of function(xs:integer) as xs:integer`, true},
		{`. instance of document-node()`, false},
		{`child::child instance of element(child)`, true},
		{`child::child instance of element(other)`, false},
		{`child::child instance of element(*)`, true},
		// typed function: arity mismatch -> false.
		{`fn:abs#1 instance of function(xs:double, xs:double) as xs:double`, false},
		// inline function with matching param/return.
		{`function($x as xs:integer) as xs:integer { $x } instance of function(xs:integer) as xs:integer`, true},
		// map as function(K) as V.
		{`map { "a": 1 } instance of function(xs:string) as xs:integer?`, true},
		{`map { "a": 1 } instance of function(xs:anyAtomicType) as item()*`, true},
		// array as function(xs:integer) as V.
		{`[1, 2] instance of function(xs:integer) as item()*`, true},
		// map with value type mismatch.
		{`map { "a": "s" } instance of map(xs:string, xs:integer)`, false},
		// empty map matches any typed map.
		{`map { } instance of map(xs:string, xs:integer)`, true},
		// array member type.
		{`[1, 2] instance of array(xs:integer)`, true},
		{`["s"] instance of array(xs:integer)`, false},
		{`[1, 2] instance of array(*)`, true},
	}
	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			r, err := evaluate(t.Context(), root, tc.expr)
			require.NoError(t, err, tc.expr)
			b, ok := r.IsBoolean()
			require.True(t, ok, tc.expr)
			require.Equal(t, tc.expect, b, tc.expr)
		})
	}

	// document-node test against the actual document node.
	r, err := evaluate(t.Context(), doc, `. instance of document-node()`)
	require.NoError(t, err)
	b, ok := r.IsBoolean()
	require.True(t, ok)
	require.True(t, b)

	r, err = evaluate(t.Context(), doc, `. instance of document-node(element(root))`)
	require.NoError(t, err)
	b, ok = r.IsBoolean()
	require.True(t, ok)
	require.True(t, b)
}
