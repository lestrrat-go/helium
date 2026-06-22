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
