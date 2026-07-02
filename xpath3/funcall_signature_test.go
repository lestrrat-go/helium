package xpath3_test

import (
	"context"
	"iter"
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

func stType(name string) xpath3.SequenceType {
	return xpath3.SequenceType{
		ItemTest:   xpath3.AtomicOrUnionType{Prefix: "xs", Name: name},
		Occurrence: xpath3.OccurrenceExactlyOne,
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

	lib := map[string]xpath3.Function{
		"upper-case": userFunc{
			min: 1, max: 1,
			call: func(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
				return xpath3.SingleString("ok"), nil
			},
		},
	}

	compiled, err := xpath3.NewCompiler().Compile(`upper-case(1)`)
	require.NoError(t, err)

	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Functions(lib, nil).
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

	dbl := stType("double")
	var observed string
	lib := map[string]xpath3.Function{
		"takes-double": typedUserFunc{
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
		},
	}

	compiled, err := xpath3.NewCompiler().Compile(`takes-double(1)`)
	require.NoError(t, err)

	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Functions(lib, nil).
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

	anyAtomic := stType("anyAtomicType")

	newLib := func(observed *string) map[string]xpath3.Function {
		return map[string]xpath3.Function{
			"takes-any": typedUserFunc{
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
			},
		}
	}

	t.Run("node argument atomizes to xs:untypedAtomic", func(t *testing.T) {
		doc := mustParseXML(t, `<root>hello</root>`)

		compiled, err := xpath3.NewCompiler().Compile(`takes-any(/root)`)
		require.NoError(t, err)

		var observed string
		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Functions(newLib(&observed), nil).
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
			Functions(newLib(&observed), nil).
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

	vars := map[string]xpath3.Sequence{
		"v": xpath3.ItemSlice{
			xpath3.AtomicValue{
				TypeName: "Q{urn:test}myDecimal",
				BaseType: xpath3.TypeDecimal,
				Value:    big.NewRat(-3, 1),
			},
		},
	}

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

	vars := map[string]xpath3.Sequence{
		"v": xpath3.ItemSlice{
			xpath3.AtomicValue{
				TypeName: "Q{urn:test}myInt",
				BaseType: xpath3.TypeInteger,
				Value:    int64(2),
			},
		},
	}
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

	vars := map[string]xpath3.Sequence{
		"u": xpath3.ItemSlice{
			xpath3.AtomicValue{
				TypeName: "Q{urn:test}myURI",
				BaseType: xpath3.TypeAnyURI,
				Value:    "abc",
			},
		},
	}

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

// countingSequence is a lazy Sequence that records how far it was actually
// iterated. It lets a test prove that a function whose parameter is item()* /
// item()+ does NOT force the whole sequence through the signature gate.
type countingSequence struct {
	n        int
	maxIndex *int // highest index materialized (-1 if never)
}

func (c countingSequence) note(i int) xpath3.Item {
	if i > *c.maxIndex {
		*c.maxIndex = i
	}
	return xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: int64(i + 1)}
}

func (c countingSequence) Len() int              { return c.n }
func (c countingSequence) Get(i int) xpath3.Item { return c.note(i) }
func (c countingSequence) Materialize() []xpath3.Item {
	out := make([]xpath3.Item, c.n)
	for i := range out {
		out[i] = c.note(i)
	}
	return out
}
func (c countingSequence) Items() iter.Seq[xpath3.Item] {
	return func(yield func(xpath3.Item) bool) {
		for i := range c.n {
			if !yield(c.note(i)) {
				return
			}
		}
	}
}

// Finding 1 (round 7): the signature gate must NOT iterate a lazy sequence when
// the parameter item type is item() — count()/exists() (item()* param) read only
// Len(), so the gate must stay lazy and never touch a single element. The lazy
// sequence is produced by a registered function so it reaches the gate without
// being materialized by variable cloning.
func TestSignatureGateKeepsItemStarLazy(t *testing.T) {
	t.Parallel()

	for _, fn := range []string{"count", "exists"} {
		t.Run(fn, func(t *testing.T) {
			maxIdx := -1
			lib := map[string]xpath3.Function{
				"make-lazy": userFunc{
					min: 0, max: 0,
					call: func(_ context.Context, _ []xpath3.Sequence) (xpath3.Sequence, error) {
						return countingSequence{n: 1000, maxIndex: &maxIdx}, nil
					},
				},
			}
			_, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
				Functions(lib, nil).
				Evaluate(t.Context(), xpath3.NewCompiler().MustCompile(fn+"(make-lazy())"), nil)
			require.NoError(t, err)
			require.Equal(t, -1, maxIdx, "item()* gate must not iterate the lazy sequence")
		})
	}
}

// Finding 1 (round 7): count(1 to N) / exists(1 to N) over a huge lazy range must
// return promptly without materializing the range — the item()* gate must not
// force iteration.
//
// NOT parallel: the allocation assertion reads runtime.MemStats.TotalAlloc, which
// is a PROCESS-WIDE cumulative counter. Under t.Parallel() other concurrently
// running tests' allocations pollute the (m2-m1) delta, spuriously blowing the
// budget (observed on Windows CI). Running in the sequential phase isolates the
// measurement to this test's own work.
func TestSignatureGateLargeRangeIsLazy(t *testing.T) {
	for expr, check := range map[string]func(*xpath3.Result){
		`count(1 to 9000000)`: func(r *xpath3.Result) {
			n, ok := r.IsNumber()
			require.True(t, ok)
			require.Equal(t, float64(9000000), n)
		},
		`exists(1 to 9000000)`: func(r *xpath3.Result) {
			b, ok := r.IsBoolean()
			require.True(t, ok)
			require.True(t, b)
		},
	} {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)
		start := time.Now()
		result, err := evaluate(t.Context(), nil, expr)
		elapsed := time.Since(start)
		runtime.ReadMemStats(&m2)

		require.NoError(t, err, expr)
		check(result)
		require.Less(t, elapsed, 200*time.Millisecond, "%s must stay lazy", expr)
		allocKB := (m2.TotalAlloc - m1.TotalAlloc) / 1024
		require.Less(t, allocKB, uint64(50*1024), "%s must not materialize the range", expr)
	}
}

// Finding 1 (round 7): the item()+ gate must still reject an EMPTY sequence
// (cardinality), and item()* must still accept it — without iterating.
func TestSignatureGateItemPlusCardinality(t *testing.T) {
	t.Parallel()

	// fn:head has signature item()* — empty input is fine.
	result, err := evaluate(t.Context(), nil, `count(head(()))`)
	require.NoError(t, err)
	n, ok := result.IsNumber()
	require.True(t, ok)
	require.Equal(t, float64(0), n)
}

// typedDoubleLib registers a TypedFunction "takes-double" (xs:double param) that
// records the observed atomized type of its argument.
func typedDoubleLib(observed *string) map[string]xpath3.Function {
	dbl := stType("double")
	return map[string]xpath3.Function{
		"takes-double": typedUserFunc{
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
			params: []xpath3.SequenceType{dbl},
		},
	}
}

// Finding 2 (round 7): partial application of a TypedFunction (xs:double param)
// must enforce the signature on the placeholder-supplied argument — an xs:integer
// must be promoted to xs:double, exactly as a direct call would.
func TestPartialApplicationCoercesTypedParam(t *testing.T) {
	t.Parallel()

	var observed string
	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Functions(typedDoubleLib(&observed), nil).
		Evaluate(t.Context(), xpath3.NewCompiler().MustCompile(`takes-double(?)(1)`), nil)
	require.NoError(t, err)

	require.Equal(t, xpath3.TypeDouble, observed, "placeholder arg must be coerced to xs:double")
	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, xpath3.TypeDouble, s)
}

// Finding 2 (round 7): a non-coercible placeholder value supplied to a typed
// partial application raises XPTY0004, just like a direct call.
func TestPartialApplicationRejectsNonCoercible(t *testing.T) {
	t.Parallel()

	var observed string
	_, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Functions(typedDoubleLib(&observed), nil).
		Evaluate(t.Context(), xpath3.NewCompiler().MustCompile(`takes-double(?)(current-date())`), nil)
	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, lexicon.ErrXPTY0004, xpErr.Code)
}

// Finding 2 (round 7): typed atomization errors (FORG0001) propagate unchanged
// through partial application — an invalid xs:untypedAtomic→double cast must
// surface FORG0001, not XPTY0004.
func TestPartialApplicationPropagatesTypedError(t *testing.T) {
	t.Parallel()

	var observed string
	_, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Functions(typedDoubleLib(&observed), nil).
		Evaluate(t.Context(), xpath3.NewCompiler().MustCompile(`takes-double(?)(xs:untypedAtomic("abc"))`), nil)
	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "FORG0001", xpErr.Code)
}

// Finding (named function ref): a named function reference (f#N) to a
// TypedFunction must coerce its arguments exactly like a direct call. A
// TypedFunction declaring xs:double invoked via (takes-double#1)(1) must observe a
// coerced xs:double, not the original xs:integer — mirroring takes-double(1).
func TestNamedFunctionRefCoercesTypedParam(t *testing.T) {
	t.Parallel()

	var observed string
	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Functions(typedDoubleLib(&observed), nil).
		Evaluate(t.Context(), xpath3.NewCompiler().MustCompile(`(takes-double#1)(1)`), nil)
	require.NoError(t, err)

	require.Equal(t, xpath3.TypeDouble, observed, "named-ref arg must be coerced to xs:double")
	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, xpath3.TypeDouble, s)
}

// Finding (named function ref): a non-coercible argument supplied through a named
// function reference raises XPTY0004, just like a direct call.
func TestNamedFunctionRefRejectsNonCoercible(t *testing.T) {
	t.Parallel()

	var observed string
	_, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Functions(typedDoubleLib(&observed), nil).
		Evaluate(t.Context(), xpath3.NewCompiler().MustCompile(`(takes-double#1)(current-date())`), nil)
	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, lexicon.ErrXPTY0004, xpErr.Code)
}

// Finding (function-lookup): fn:function-lookup resolves a TypedFunction and the
// returned function item must coerce its arguments like a direct call — an
// xs:integer supplied to an xs:double parameter must be observed as xs:double.
func TestFunctionLookupCoercesTypedParam(t *testing.T) {
	t.Parallel()

	var observed string
	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Functions(typedDoubleLib(&observed), nil).
		Evaluate(t.Context(), xpath3.NewCompiler().MustCompile(`function-lookup(QName("", "takes-double"), 1)(1)`), nil)
	require.NoError(t, err)

	require.Equal(t, xpath3.TypeDouble, observed, "function-lookup arg must be coerced to xs:double")
	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, xpath3.TypeDouble, s)
}

// userAbsLib registers an untyped user function under the built-in local name
// "abs" that accepts (and echoes) a single xs:string argument. The built-in
// fn:abs signature is (xs:numeric?) as xs:numeric?, which would reject a string,
// so this library distinguishes "built-in signature applied" from "user function
// invoked".
func userAbsLib() map[string]xpath3.Function {
	return map[string]xpath3.Function{
		"abs": userFunc{
			min: 1, max: 1,
			call: func(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
				av, err := xpath3.AtomizeItem(args[0].Get(0))
				if err != nil {
					return nil, err
				}
				return xpath3.SingleString("user:" + av.TypeName), nil
			},
		},
	}
}

// Finding B-XPATH3-FUNCREF-BUILTIN-SIG: a named function reference to a user
// override of a built-in name (abs#1) must bind the user function's own
// signature, not the built-in's. Calling it with an xs:string — which the
// built-in fn:abs signature would reject — must reach the user function.
func TestNamedFunctionRefUserOverrideSkipsBuiltinSignature(t *testing.T) {
	t.Parallel()

	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Functions(userAbsLib(), nil).
		Evaluate(t.Context(), xpath3.NewCompiler().MustCompile(`(abs#1)("hello")`), nil)
	require.NoError(t, err)

	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "user:"+xpath3.TypeString, s)
}

// Finding B-XPATH3-FUNCREF-BUILTIN-SIG: fn:function-lookup of a user override of
// a built-in name must likewise bind the user function's own signature, not the
// built-in's.
func TestFunctionLookupUserOverrideSkipsBuiltinSignature(t *testing.T) {
	t.Parallel()

	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Functions(userAbsLib(), nil).
		Evaluate(t.Context(), xpath3.NewCompiler().MustCompile(`function-lookup(QName("", "abs"), 1)("hello")`), nil)
	require.NoError(t, err)

	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "user:"+xpath3.TypeString, s)
}

// Finding 2 (round 7): a fixed (curried) argument must also be coerced — when the
// double parameter is curried with an xs:integer literal, the body still observes
// xs:double.
func TestPartialApplicationCoercesFixedArg(t *testing.T) {
	t.Parallel()

	dbl := stType("double")
	str := stType("string")
	var observed string
	lib := map[string]xpath3.Function{
		"takes-double-str": typedUserFunc{
			userFunc: userFunc{
				min: 2, max: 2,
				call: func(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
					av, err := xpath3.AtomizeItem(args[0].Get(0))
					if err != nil {
						return nil, err
					}
					observed = av.TypeName
					return xpath3.SingleString(av.TypeName), nil
				},
			},
			params: []xpath3.SequenceType{dbl, str},
		},
	}

	// Curry the first (double) arg with an integer; the placeholder fills the string.
	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Functions(lib, nil).
		Evaluate(t.Context(), xpath3.NewCompiler().MustCompile(`takes-double-str(1, ?)("x")`), nil)
	require.NoError(t, err)
	require.Equal(t, xpath3.TypeDouble, observed, "curried integer must be coerced to xs:double")
	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, xpath3.TypeDouble, s)
}
