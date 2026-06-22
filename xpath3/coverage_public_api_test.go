package xpath3_test

import (
	"context"
	"math"
	"math/big"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

const (
	collationCodepoint = "http://www.w3.org/2005/xpath-functions/collation/codepoint"
	collationHTMLASCII = "http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive"
	collationUCA       = "http://www.w3.org/2013/collation/UCA"
)

func atomicSeq(av xpath3.AtomicValue) xpath3.Sequence {
	return xpath3.ItemSlice{av}
}

func intAtomic(n int64) xpath3.AtomicValue {
	return xpath3.AtomicValue{TypeName: xpath3.TypeInteger, Value: big.NewInt(n)}
}

func strAtomic(s string) xpath3.AtomicValue {
	return xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: s}
}

func TestFunctionLibrary_CRUD(t *testing.T) {
	lib := xpath3.NewFunctionLibrary()

	var fn xpath3.Function = identityFn{}

	// Get/GetNS on empty library.
	_, ok := lib.Get("missing")
	require.False(t, ok)
	_, ok = lib.GetNS("urn:x", "missing")
	require.False(t, ok)

	lib.Set("local", fn)
	got, ok := lib.Get("local")
	require.True(t, ok)
	require.NotNil(t, got)

	lib.SetNS("urn:x", "ns", fn)
	got, ok = lib.GetNS("urn:x", "ns")
	require.True(t, ok)
	require.NotNil(t, got)
	_, ok = lib.GetNS("urn:other", "ns")
	require.False(t, ok)

	lib.Delete("local")
	_, ok = lib.Get("local")
	require.False(t, ok)

	lib.DeleteNS("urn:x", "ns")
	_, ok = lib.GetNS("urn:x", "ns")
	require.False(t, ok)

	lib.Set("a", fn)
	lib.SetNS("urn:x", "b", fn)
	lib.Clear()
	_, ok = lib.Get("a")
	require.False(t, ok)
	_, ok = lib.GetNS("urn:x", "b")
	require.False(t, ok)
}

type identityFn struct{}

func (identityFn) MinArity() int { return 1 }
func (identityFn) MaxArity() int { return 1 }
func (identityFn) Call(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	return args[0], nil
}

func TestVariables_GetDeleteClear(t *testing.T) {
	v := xpath3.NewVariables()

	_, ok := v.Get("x")
	require.False(t, ok)

	v.Set("x", atomicSeq(intAtomic(5)))
	got, ok := v.Get("x")
	require.True(t, ok)
	require.Equal(t, 1, got.Len())

	v.Delete("x")
	_, ok = v.Get("x")
	require.False(t, ok)

	v.Set("a", atomicSeq(intAtomic(1)))
	v.Set("b", atomicSeq(intAtomic(2)))
	require.Equal(t, 2, v.Len())
	v.Clear()
	require.Equal(t, 0, v.Len())
}

func TestRegex_PublicAPI(t *testing.T) {
	re, err := xpath3.CompileRegex(`a(b+)c`, "")
	require.NoError(t, err)

	matched, err := re.MatchString("abbbc")
	require.NoError(t, err)
	require.True(t, matched)

	matched, err = re.MatchString("xyz")
	require.NoError(t, err)
	require.False(t, matched)

	idx, err := re.FindAllSubmatchIndex("abc and abbc", -1)
	require.NoError(t, err)
	require.Len(t, idx, 2)
	// First match "abc": full match + one capture group => 4 indices.
	require.Len(t, idx[0], 4)

	// Case-insensitive flag.
	rei, err := xpath3.CompileRegex(`abc`, "i")
	require.NoError(t, err)
	matched, err = rei.MatchString("ABC")
	require.NoError(t, err)
	require.True(t, matched)

	// Invalid pattern surfaces an error.
	_, err = xpath3.CompileRegex(`[`, "")
	require.Error(t, err)
}

func TestBuiltinFunctionQueries(t *testing.T) {
	require.True(t, xpath3.IsBuiltinFunction("abs"))
	require.False(t, xpath3.IsBuiltinFunction("definitely-not-a-builtin"))

	require.True(t, xpath3.IsBuiltinFunctionNS(xpath3.NSFn, "count"))
	require.False(t, xpath3.IsBuiltinFunctionNS("urn:nope", "count"))

	require.True(t, xpath3.BuiltinFunctionAcceptsArity(xpath3.NSFn, "abs", 1))
	require.False(t, xpath3.BuiltinFunctionAcceptsArity(xpath3.NSFn, "abs", 5))
	require.False(t, xpath3.BuiltinFunctionAcceptsArity(xpath3.NSFn, "nope", 1))
}

func TestPredeclaredNamespaces(t *testing.T) {
	ns := xpath3.PredeclaredNamespaces()
	require.Equal(t, xpath3.NSFn, ns["fn"])
	require.NotEmpty(t, ns["xs"])

	// Mutating the returned copy must not affect package state.
	ns["fn"] = "tampered"
	ns2 := xpath3.PredeclaredNamespaces()
	require.Equal(t, xpath3.NSFn, ns2["fn"])
}

func TestCollationHelpers(t *testing.T) {
	require.True(t, xpath3.IsCollationSupported(collationCodepoint))
	require.True(t, xpath3.IsCollationSupported(collationHTMLASCII))
	require.False(t, xpath3.IsCollationSupported("urn:bogus-collation"))

	keyFn, err := xpath3.ResolveCollationKeyFunc(collationCodepoint)
	require.NoError(t, err)
	require.Equal(t, keyFn("abc"), keyFn("abc"))
	require.NotEqual(t, keyFn("abc"), keyFn("abd"))

	_, err = xpath3.ResolveCollationKeyFunc("urn:bogus")
	require.Error(t, err)

	cmpFn, err := xpath3.ResolveCollationCompareFunc(collationCodepoint)
	require.NoError(t, err)
	require.Equal(t, 0, cmpFn("a", "a"))
	require.Negative(t, cmpFn("a", "b"))
	require.Positive(t, cmpFn("b", "a"))

	_, err = xpath3.ResolveCollationCompareFunc("urn:bogus")
	require.Error(t, err)

	require.False(t, xpath3.CollationHasUnsupportedOptions(collationCodepoint))
	require.False(t, xpath3.CollationHasUnsupportedOptions(collationUCA))
	require.True(t, xpath3.CollationHasUnsupportedOptions(collationUCA+"?alternate=shifted"))
}

func TestAtomicEquals(t *testing.T) {
	require.True(t, xpath3.AtomicEquals(intAtomic(3), intAtomic(3)))
	require.False(t, xpath3.AtomicEquals(intAtomic(3), intAtomic(4)))
	require.True(t, xpath3.AtomicEquals(strAtomic("x"), strAtomic("x")))
	// Incomparable types return false rather than erroring.
	require.False(t, xpath3.AtomicEquals(strAtomic("x"), intAtomic(1)))
}

func TestAtomicValue_StringIsNaN(t *testing.T) {
	s := intAtomic(7).String()
	require.Contains(t, s, "7")

	nan := xpath3.AtomicValue{TypeName: xpath3.TypeDouble, Value: xpath3.NewDouble(math.NaN())}
	require.True(t, nan.IsNaN())

	notNaN := xpath3.AtomicValue{TypeName: xpath3.TypeDouble, Value: xpath3.NewDouble(1.5)}
	require.False(t, notNaN.IsNaN())

	// Non-float types are never NaN.
	require.False(t, intAtomic(1).IsNaN())
}

func TestFloatValue_IsSpecial(t *testing.T) {
	require.False(t, xpath3.NewDouble(1.0).IsSpecial())
	require.True(t, xpath3.NewDouble(math.NaN()).IsSpecial())
	require.True(t, xpath3.NewDouble(math.Inf(1)).IsSpecial())
	require.True(t, xpath3.NewDouble(math.Inf(-1)).IsSpecial())
	require.True(t, xpath3.NewDouble(math.Copysign(0, -1)).IsSpecial())
}

func TestBuiltinIsSubtypeOf(t *testing.T) {
	require.True(t, xpath3.BuiltinIsSubtypeOf(xpath3.TypeInteger, xpath3.TypeDecimal))
	require.True(t, xpath3.BuiltinIsSubtypeOf(xpath3.TypeInteger, xpath3.TypeInteger))
	require.False(t, xpath3.BuiltinIsSubtypeOf(xpath3.TypeString, xpath3.TypeInteger))
}

func TestExpression_ValidateAndCopy(t *testing.T) {
	expr, err := xpath3.NewCompiler().Compile(`fn:abs(-1)`)
	require.NoError(t, err)

	// Valid: fn prefix is predeclared.
	require.NoError(t, expr.Validate(map[string]string{}))

	bad, err := xpath3.NewCompiler().Compile(`undeclared:foo()`)
	require.NoError(t, err)
	require.Error(t, bad.Validate(map[string]string{}))

	// Result.Copy on an empty result yields an empty result whose sequence is nil.
	empty := xpath3.Result{}
	cp := empty.Copy()
	require.Nil(t, cp.Sequence())

	r, err := evaluate(t.Context(), nil, `(1, 2, 3)`)
	require.NoError(t, err)
	rc := r.Copy()
	require.Equal(t, 3, rc.Sequence().Len())
}

func TestEvaluatorBuilders(t *testing.T) {
	doc := mustParseXML(t, "<root><a/><b/></root>")
	root := doc.DocumentElement()

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Position(2).
		Size(5).
		PreservedIDAnnotations(map[helium.Node]string{}).
		AllowXML11Chars()

	compiled, err := xpath3.NewCompiler().Compile(`position()`)
	require.NoError(t, err)
	res, err := eval.Evaluate(t.Context(), compiled, root)
	require.NoError(t, err)
	n, ok := res.IsNumber()
	require.True(t, ok)
	require.Equal(t, float64(2), n)

	compiledLast, err := xpath3.NewCompiler().Compile(`last()`)
	require.NoError(t, err)
	res, err = eval.Evaluate(t.Context(), compiledLast, root)
	require.NoError(t, err)
	n, ok = res.IsNumber()
	require.True(t, ok)
	require.Equal(t, float64(5), n)
}

func TestVariableAndFunctionResolver(t *testing.T) {
	doc := mustParseXML(t, "<root/>")

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		VariableResolver(varResolver{}).
		FunctionResolver(funcResolver{})

	compiled, err := xpath3.NewCompiler().Compile(`$dynamic`)
	require.NoError(t, err)
	res, err := eval.Evaluate(t.Context(), compiled, doc)
	require.NoError(t, err)
	n, ok := res.IsNumber()
	require.True(t, ok)
	require.Equal(t, float64(99), n)
}

type varResolver struct{}

func (varResolver) ResolveVariable(_ context.Context, name string) (xpath3.Sequence, bool, error) {
	if name == "dynamic" {
		return atomicSeq(intAtomic(99)), true, nil
	}
	return nil, false, nil
}

type funcResolver struct{}

func (funcResolver) ResolveFunction(_ context.Context, _, _ string, _ int) (xpath3.Function, bool, error) {
	return nil, false, nil
}

func TestFnContextNode(t *testing.T) {
	doc := mustParseXML(t, "<root><child/></root>")
	root := doc.DocumentElement()

	captured := &capturingFn{}
	lib := xpath3.NewFunctionLibrary()
	lib.Set("capture", captured)

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Functions(lib)
	compiled, err := xpath3.NewCompiler().Compile(`capture()`)
	require.NoError(t, err)
	_, err = eval.Evaluate(t.Context(), compiled, root)
	require.NoError(t, err)
	require.NotNil(t, captured.node)
	require.Equal(t, root, captured.node)
	// Direct (non-dynamic) call: IsDynamicCall is false.
	require.False(t, captured.dynamic)
}

type capturingFn struct {
	node    helium.Node
	dynamic bool
}

func (*capturingFn) MinArity() int { return 0 }
func (*capturingFn) MaxArity() int { return 0 }
func (c *capturingFn) Call(ctx context.Context, _ []xpath3.Sequence) (xpath3.Sequence, error) {
	c.node = xpath3.FnContextNode(ctx)
	c.dynamic = xpath3.IsDynamicCall(ctx)
	return atomicSeq(intAtomic(1)), nil
}

func TestCodeQName(t *testing.T) {
	// Trigger a real XPathError (division by zero -> FOAR0001).
	_, err := evaluate(t.Context(), nil, `1 idiv 0`)
	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	qn := xpErr.CodeQName()
	require.NotEmpty(t, qn.Local)
}

func TestExpression_AST_StreamInfo(t *testing.T) {
	expr, err := xpath3.NewCompiler().Compile(`child::a/descendant::b`)
	require.NoError(t, err)

	require.NotNil(t, expr.AST())

	si := expr.StreamInfo()
	require.True(t, si.HasDownwardStep)

	// Nil expression yields a zero StreamInfo without panicking.
	var nilExpr *xpath3.Expression
	require.Equal(t, xpath3.StreamInfo{}, nilExpr.StreamInfo())
}

func TestPrettyTokens(t *testing.T) {
	l, err := xpath3.NewLexerForTesting(`1 + 2`)
	require.NoError(t, err)
	require.NotEmpty(t, l.PrettyTokens())
}
