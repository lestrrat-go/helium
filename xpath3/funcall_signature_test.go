package xpath3_test

import (
	"context"
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
