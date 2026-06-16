package xpath3_test

import (
	"context"
	"math/big"
	"runtime"
	"testing"
	"time"

	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// userFunc is a minimal user-defined Function for signature-enforcement tests.
type userFunc struct {
	min, max int
	call     func(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error)
}

func (f userFunc) MinArity() int { return f.min }
func (f userFunc) MaxArity() int { return f.max }
func (f userFunc) Call(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	return f.call(ctx, args)
}

// typedUserFunc additionally declares parameter/return types (TypedFunction).
type typedUserFunc struct {
	userFunc
	params []xpath3.SequenceType
	ret    *xpath3.SequenceType
}

func (f typedUserFunc) FuncParamTypes() []xpath3.SequenceType { return f.params }
func (f typedUserFunc) FuncReturnType() *xpath3.SequenceType  { return f.ret }

func stType(name string, occ xpath3.Occurrence) xpath3.SequenceType {
	return xpath3.SequenceType{
		ItemTest:   xpath3.AtomicOrUnionType{Prefix: "xs", Name: name},
		Occurrence: occ,
	}
}

// Finding 1: a built-in called via Q{uri}local syntax must still enforce its
// registered signature, so a cardinality-violating arg raises XPTY0004.
func TestURIQualifiedBuiltinEnforcesSignature(t *testing.T) {
	t.Parallel()
	_, err := evaluate(t.Context(), nil, `Q{http://www.w3.org/2005/xpath-functions}abs((1, 2))`)
	require.Error(t, err)

	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, lexicon.ErrXPTY0004, xpErr.Code)
}

// Finding 2: a user function registered under an unqualified name that shadows a
// built-in must NOT be subjected to the built-in signature. Calling it with an
// argument the built-in would reject must succeed.
func TestUserOverrideSkipsBuiltinSignature(t *testing.T) {
	t.Parallel()

	lib := xpath3.NewFunctionLibrary()
	lib.Set("upper-case", userFunc{
		min: 1, max: 1,
		call: func(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
			return xpath3.SingleString("ok"), nil
		},
	})

	compiled, err := xpath3.NewCompiler().Compile(`upper-case(1)`)
	require.NoError(t, err)

	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Functions(lib).
		Evaluate(t.Context(), compiled, nil)
	require.NoError(t, err)

	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "ok", s)
}

// Finding 3: a TypedFunction declaring xs:double must observe a coerced xs:double
// when called with an xs:integer (function-conversion / numeric promotion).
func TestTypedUserFunctionObservesCoercedArg(t *testing.T) {
	t.Parallel()

	dbl := stType("double", xpath3.OccurrenceExactlyOne)
	var observed string
	lib := xpath3.NewFunctionLibrary()
	lib.Set("takes-double", typedUserFunc{
		userFunc: userFunc{
			min: 1, max: 1,
			call: func(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
				av, err := xpath3.AtomizeItem(args[0].Get(0))
				if err != nil {
					return nil, err
				}
				observed = av.TypeName
				return xpath3.SingleString(av.TypeName), nil
			},
		},
		params: []xpath3.SequenceType{dbl},
		ret:    nil,
	})

	compiled, err := xpath3.NewCompiler().Compile(`takes-double(1)`)
	require.NoError(t, err)

	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Functions(lib).
		Evaluate(t.Context(), compiled, nil)
	require.NoError(t, err)

	require.Equal(t, xpath3.TypeDouble, observed)
	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, xpath3.TypeDouble, s)
}

// Finding 4: a TypedFunction declaring xs:anyAtomicType must accept a NODE
// argument. Per the function-conversion rules the node is atomized (a node
// atomizes to xs:untypedAtomic), and xs:untypedAtomic is a subtype of
// xs:anyAtomicType, so the call must succeed and observe the atomized value
// rather than failing the static signature check with XPTY0004.
func TestTypedUserFunctionAnyAtomicAcceptsNode(t *testing.T) {
	t.Parallel()

	anyAtomic := stType("anyAtomicType", xpath3.OccurrenceExactlyOne)

	newLib := func(observed *string) *xpath3.FunctionLibrary {
		lib := xpath3.NewFunctionLibrary()
		lib.Set("takes-any", typedUserFunc{
			userFunc: userFunc{
				min: 1, max: 1,
				call: func(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
					av, err := xpath3.AtomizeItem(args[0].Get(0))
					if err != nil {
						return nil, err
					}
					*observed = av.TypeName
					return xpath3.SingleString(av.TypeName), nil
				},
			},
			params: []xpath3.SequenceType{anyAtomic},
		})
		return lib
	}

	t.Run("node argument atomizes to xs:untypedAtomic", func(t *testing.T) {
		doc := mustParseXML(t, `<root>hello</root>`)

		compiled, err := xpath3.NewCompiler().Compile(`takes-any(/root)`)
		require.NoError(t, err)

		var observed string
		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Functions(newLib(&observed)).
			Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)

		require.Equal(t, xpath3.TypeUntypedAtomic, observed)
		s, ok := result.IsString()
		require.True(t, ok)
		require.Equal(t, xpath3.TypeUntypedAtomic, s)
	})

	t.Run("atomic argument retains its type", func(t *testing.T) {
		compiled, err := xpath3.NewCompiler().Compile(`takes-any(42)`)
		require.NoError(t, err)

		var observed string
		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Functions(newLib(&observed)).
			Evaluate(t.Context(), compiled, nil)
		require.NoError(t, err)

		require.Equal(t, xpath3.TypeInteger, observed)
		s, ok := result.IsString()
		require.True(t, ok)
		require.Equal(t, xpath3.TypeInteger, s)
	})
}

// Finding 1: the signature gate must propagate a typed atomization error rather
// than collapsing it into XPTY0004. Atomizing a map/function item where an
// atomic is required raises FOTY0013, which try/catch dispatches on.
func TestSignatureGatePropagatesFOTY0013(t *testing.T) {
	t.Parallel()
	_, err := evaluate(t.Context(), nil, `upper-case(map{})`)
	require.Error(t, err)

	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "FOTY0013", xpErr.Code)
}

// Finding 1: an invalid xs:untypedAtomic→numeric cast inside the signature gate
// raises FORG0001 (invalid cast), not XPTY0004.
func TestSignatureGatePropagatesFORG0001(t *testing.T) {
	t.Parallel()
	_, err := evaluate(t.Context(), nil, `abs(xs:untypedAtomic("abc"))`)
	require.Error(t, err)

	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "FORG0001", xpErr.Code)
}

// Finding 1: a plain type/cardinality mismatch (no typed atomization/cast error)
// must still yield XPTY0004.
func TestSignatureGatePlainMismatchIsXPTY0004(t *testing.T) {
	t.Parallel()
	for _, expr := range []string{`upper-case((1, 2))`, `abs((1, 2))`, `upper-case(current-date())`} {
		_, err := evaluate(t.Context(), nil, expr)
		require.Error(t, err, expr)

		var xpErr *xpath3.XPathError
		require.ErrorAs(t, err, &xpErr, expr)
		require.Equal(t, lexicon.ErrXPTY0004, xpErr.Code, expr)
	}
}

// Finding (round 4): the static signature gate must accept a schema-DERIVED
// numeric atomic value whose BaseType is a built-in numeric type, since the
// numeric builtins promote it via PromoteSchemaType. A value of a custom type
// derived from xs:decimal IS numeric and abs/round must accept it — the gate
// must not raise XPTY0004 before the builtin can promote it.
func TestSignatureGateAcceptsSchemaDerivedNumeric(t *testing.T) {
	t.Parallel()

	vars := xpath3.VariablesFromMap(map[string]xpath3.Sequence{
		"v": xpath3.ItemSlice{
			xpath3.AtomicValue{
				TypeName: "Q{urn:test}myDecimal",
				BaseType: xpath3.TypeDecimal,
				Value:    big.NewRat(-3, 1),
			},
		},
	})

	cases := map[string]float64{
		`abs($v)`:   3,  // |-3| = 3
		`round($v)`: -3, // round(-3) = -3 (the gate must let the value through)
		`$v + 1`:    -2, // arithmetic also accepts the schema-derived decimal
	}
	for expr, want := range cases {
		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Variables(vars).
			Evaluate(t.Context(), xpath3.NewCompiler().MustCompile(expr), nil)
		require.NoError(t, err, expr)

		n, ok := result.IsNumber()
		require.True(t, ok, expr)
		require.Equal(t, want, n, expr)
	}
}

// Finding 1 (round 5): numeric promotion of an integer-DERIVED subtype to xs:double
// must succeed. The signature gate admits xs:positiveInteger/xs:int/xs:long/… as a
// numeric, so the subsequent cast must coerce the whole integer-derived family (not
// just exact xs:integer) — otherwise a valid promotion is rejected with XPTY0004
// before the builtin (here fn:substring) runs.
func TestSignatureGatePromotesIntegerDerivedSubtypes(t *testing.T) {
	t.Parallel()

	exprs := []string{
		`substring("abcd", xs:positiveInteger("2"))`,
		`substring("abcd", xs:int("2"))`,
		`substring("abcd", xs:long("2"))`,
		`substring("abcd", xs:short("2"))`,
		`substring("abcd", xs:byte("2"))`,
		`substring("abcd", xs:nonNegativeInteger("2"))`,
	}
	for _, expr := range exprs {
		result, err := evaluate(t.Context(), nil, expr)
		require.NoError(t, err, expr)
		s, ok := result.IsString()
		require.True(t, ok, expr)
		require.Equal(t, "bcd", s, expr)
	}
}

// Finding 1 (round 5): a schema-DERIVED integer (BaseType xs:integer) supplied to an
// xs:double parameter must promote — the gate accepts it via BaseType ancestry and the
// cast must agree by normalizing through PromoteSchemaType.
func TestSignatureGatePromotesSchemaDerivedInteger(t *testing.T) {
	t.Parallel()

	vars := xpath3.VariablesFromMap(map[string]xpath3.Sequence{
		"v": xpath3.ItemSlice{
			xpath3.AtomicValue{
				TypeName: "Q{urn:test}myInt",
				BaseType: xpath3.TypeInteger,
				Value:    int64(2),
			},
		},
	})
	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Variables(vars).
		Evaluate(t.Context(), xpath3.NewCompiler().MustCompile(`substring("abcd", $v)`), nil)
	require.NoError(t, err)
	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "bcd", s)
}

// Finding 1 (round 5): a genuinely non-numeric value supplied where a numeric is
// required still raises XPTY0004 — broadening the cast must not admit non-numerics.
func TestSignatureGateRejectsNonNumericForNumericParam(t *testing.T) {
	t.Parallel()
	_, err := evaluate(t.Context(), nil, `substring("abcd", current-date())`)
	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, lexicon.ErrXPTY0004, xpErr.Code)
}

// Finding 2 (round 5): when a singleton (xs:string?) parameter receives an ARRAY whose
// later members would themselves fail atomization (a map → FOTY0013), the cardinality
// failure observed at the 2nd atom must win. The stop signal must propagate out of the
// recursive array-member atomization so the map is never reached.
func TestSignatureGateArrayCardinalityBeatsLaterAtomError(t *testing.T) {
	t.Parallel()
	_, err := evaluate(t.Context(), nil, `upper-case([1, 2, map{}])`)
	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, lexicon.ErrXPTY0004, xpErr.Code)
}

// Finding 2 (round 5): legitimate array atomization still works fully — a single-member
// array satisfies a singleton param, and array-accepting functions atomize all members.
func TestSignatureGateArrayAtomizationStillWorks(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		`upper-case(["a"])`:         "A",
		`string-join([1,2,3], ",")`: "1,2,3",
	}
	for expr, want := range cases {
		result, err := evaluate(t.Context(), nil, expr)
		require.NoError(t, err, expr)
		s, ok := result.IsString()
		require.True(t, ok, expr)
		require.Equal(t, want, s, expr)
	}
}

// Finding (round 6): xs:anyURI promotes to xs:string per the function-conversion
// rules, and so do its SUBTYPES. A schema-DERIVED anyURI (BaseType xs:anyURI)
// supplied to fn:upper-case (xs:string? param) must be admitted by the static
// signature gate, promoted to xs:string, and upper-cased — not rejected with
// XPTY0004 before the builtin runs.
func TestSignatureGatePromotesSchemaDerivedAnyURI(t *testing.T) {
	t.Parallel()

	vars := xpath3.VariablesFromMap(map[string]xpath3.Sequence{
		"u": xpath3.ItemSlice{
			xpath3.AtomicValue{
				TypeName: "Q{urn:test}myURI",
				BaseType: xpath3.TypeAnyURI,
				Value:    "abc",
			},
		},
	})

	cases := map[string]string{
		`upper-case($u)`:    "ABC",
		`string-length($u)`: "3",
		`concat($u, "x")`:   "abcx",
	}
	for expr, want := range cases {
		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Variables(vars).
			Evaluate(t.Context(), xpath3.NewCompiler().MustCompile(expr), nil)
		require.NoError(t, err, expr)

		switch expr {
		case `string-length($u)`:
			n, ok := result.IsNumber()
			require.True(t, ok, expr)
			require.Equal(t, float64(3), n, expr)
		default:
			s, ok := result.IsString()
			require.True(t, ok, expr)
			require.Equal(t, want, s, expr)
		}
	}
}

// Finding (round 6): a plain (built-in) xs:anyURI supplied to an xs:string? param
// must continue to promote to xs:string and be accepted — the exact-type path
// previously handled this, and the subtype-aware gate must not regress it.
func TestSignatureGatePromotesPlainAnyURI(t *testing.T) {
	t.Parallel()
	result, err := evaluate(t.Context(), nil, `upper-case(xs:anyURI("abc"))`)
	require.NoError(t, err)
	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "ABC", s)
}

// Finding (round 6): a genuinely non-anyURI, non-string value supplied to an
// xs:string param still raises XPTY0004 — widening anyURI acceptance must not
// admit unrelated types.
func TestSignatureGateRejectsNonStringForStringParam(t *testing.T) {
	t.Parallel()
	_, err := evaluate(t.Context(), nil, `upper-case(current-date())`)
	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, lexicon.ErrXPTY0004, xpErr.Code)
}

// Finding 2: a too-long sequence supplied to a singleton (xs:string?) parameter
// must be rejected promptly with XPTY0004 — without atomizing the whole range.
// A 10M-item lazy range would allocate ~1GB if atomized eagerly; the cap keeps
// allocation and time tiny.
func TestSignatureGateRejectsLongSequencePromptly(t *testing.T) {
	t.Parallel()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	start := time.Now()
	_, err := evaluate(t.Context(), nil, `upper-case(1 to 10000000)`)
	elapsed := time.Since(start)
	runtime.ReadMemStats(&m2)

	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, lexicon.ErrXPTY0004, xpErr.Code)

	// Eager atomization of 10M items took ~800ms / ~1GB before the fix; the
	// incremental cap keeps both small. Generous bounds avoid CI flakiness.
	require.Less(t, elapsed, 200*time.Millisecond, "should reject without atomizing whole range")
	allocKB := (m2.TotalAlloc - m1.TotalAlloc) / 1024
	require.Less(t, allocKB, uint64(50*1024), "should not allocate the whole atomized sequence")
}
