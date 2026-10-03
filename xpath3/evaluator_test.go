package xpath3_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestEvaluator(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a>hello</a><b>world</b></root>`))
	require.NoError(t, err)

	compiler := xpath3.NewCompiler()

	t.Run("basic evaluation", func(t *testing.T) {
		expr, err := compiler.Compile("//a/text()")
		require.NoError(t, err)

		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Evaluate(t.Context(), expr, doc)
		require.NoError(t, err)

		nodes, err := result.Nodes()
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, testHello, string(nodes[0].Content()))
	})

	t.Run("with variables", func(t *testing.T) {
		expr, err := compiler.Compile("$x")
		require.NoError(t, err)

		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Variables(map[string]xpath3.Sequence{"x": xpath3.SingleString("test-value")}).
			Evaluate(t.Context(), expr, doc)
		require.NoError(t, err)

		s, ok := result.IsString()
		require.True(t, ok)
		require.Equal(t, "test-value", s)
	})

	t.Run("with namespaces", func(t *testing.T) {
		nsDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<root xmlns:ns="http://example.com"><ns:item>found</ns:item></root>`))
		require.NoError(t, err)

		expr, err := compiler.Compile("//ns:item/text()")
		require.NoError(t, err)

		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Namespaces(map[string]string{"ns": "http://example.com"}).
			Evaluate(t.Context(), expr, nsDoc)
		require.NoError(t, err)

		nodes, err := result.Nodes()
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "found", string(nodes[0].Content()))
	})

	t.Run("evaluator immutability", func(t *testing.T) {
		expr, err := compiler.Compile("$x")
		require.NoError(t, err)

		base := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)

		e1 := base.Variables(map[string]xpath3.Sequence{"x": xpath3.SingleString("one")})
		e2 := base.Variables(map[string]xpath3.Sequence{"x": xpath3.SingleString("two")})

		r1, err := e1.Evaluate(t.Context(), expr, doc)
		require.NoError(t, err)
		s1, ok := r1.IsString()
		require.True(t, ok)
		require.Equal(t, "one", s1)

		r2, err := e2.Evaluate(t.Context(), expr, doc)
		require.NoError(t, err)
		s2, ok := r2.IsString()
		require.True(t, ok)
		require.Equal(t, "two", s2)
	})

	t.Run("zero value evaluator", func(t *testing.T) {
		expr, err := compiler.Compile("//a/text()")
		require.NoError(t, err)

		// A zero-value Evaluator must not panic.
		var ev xpath3.Evaluator
		result, err := ev.Evaluate(t.Context(), expr, doc)
		require.NoError(t, err)

		nodes, err := result.Nodes()
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, testHello, string(nodes[0].Content()))
	})

	t.Run("zero value evaluator with fluent methods", func(t *testing.T) {
		expr, err := compiler.Compile("$x")
		require.NoError(t, err)

		// Fluent methods on a zero-value Evaluator must not panic.
		var ev xpath3.Evaluator
		result, err := ev.Variables(map[string]xpath3.Sequence{"x": xpath3.SingleString("from-zero")}).Evaluate(t.Context(), expr, doc)
		require.NoError(t, err)

		s, ok := result.IsString()
		require.True(t, ok)
		require.Equal(t, "from-zero", s)
	})

	t.Run("nil expression returns error", func(t *testing.T) {
		_, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Evaluate(t.Context(), nil, doc)
		require.EqualError(t, err, "xpath3: expression has no compiled program")
	})

	t.Run("MustCompile", func(t *testing.T) {
		expr := compiler.MustCompile("1 + 2")
		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Evaluate(t.Context(), expr, doc)
		require.NoError(t, err)

		n, ok := result.IsNumber()
		require.True(t, ok)
		require.InDelta(t, 3.0, n, 0.001)
	})
}

func TestParserInjection(t *testing.T) {
	// fn:parse-xml on a string literal whose element name is 9 bytes long.
	// An injected parser with MaxNameLength(8) must reject it; without the
	// injection (default parser) the same parse succeeds. This proves the
	// injected parser's policy reaches the fn:parse-xml parse site.
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
	require.NoError(t, err)

	expr, err := xpath3.NewCompiler().Compile(`parse-xml("<elementXY/>")`)
	require.NoError(t, err)

	t.Run("default parser parses long element name", func(t *testing.T) {
		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Evaluate(t.Context(), expr, doc)
		require.NoError(t, err)
		nodes, err := result.Nodes()
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})

	t.Run("injected parser enforces MaxNameLength", func(t *testing.T) {
		_, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Parser(helium.NewParser().MaxNameLength(8)).
			Evaluate(t.Context(), expr, doc)
		require.Error(t, err)
	})
}

func TestDocEmptyArgFragmentBaseURI(t *testing.T) {
	// doc("") resolves to the base URI verbatim. When that base URI carries a
	// fragment identifier the call must raise FODC0005, the same as a fragment
	// in an explicit argument.
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
	require.NoError(t, err)

	expr, err := xpath3.NewCompiler().Compile(`doc("")`)
	require.NoError(t, err)

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		BaseURI("file:///tmp/doc.xml#frag").
		Evaluate(t.Context(), expr, doc)
	require.Error(t, err)

	var xerr *xpath3.XPathError
	require.ErrorAs(t, err, &xerr)
	require.Equal(t, "FODC0005", xerr.Code)
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

// evalToString evaluates expr against node and returns its xs:string result.
func evalToString(t *testing.T, eval xpath3.Evaluator, expr *xpath3.Expression, node helium.Node) string {
	t.Helper()
	res, err := eval.Evaluate(t.Context(), expr, node)
	require.NoError(t, err)
	s, ok := res.IsString()
	require.True(t, ok)
	return s
}

func TestEvaluatorFocus(t *testing.T) {
	doc := mustParseXML(t, "<root><a/><b/></root>")
	root := doc.DocumentElement()

	focusExpr, err := xpath3.NewCompiler().Compile(`string-join((string(.), string(position()), string(last())), "|")`)
	require.NoError(t, err)

	base := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)

	t.Run("sets item, position, and size", func(t *testing.T) {
		eval := base.Focus(xpath3.SingleString("item").Get(0), 2, 5)
		require.Equal(t, "item|2|5", evalToString(t, eval, focusExpr, root))
	})

	t.Run("matches the separate setters", func(t *testing.T) {
		item := xpath3.SingleString("item").Get(0)
		separate := base.ContextItem(item).Position(3).Size(4)
		combined := base.Focus(item, 3, 4)
		require.Equal(t, evalToString(t, separate, focusExpr, root), evalToString(t, combined, focusExpr, root))
	})

	t.Run("nil item uses the context node and non-positive values default to 1", func(t *testing.T) {
		eval := base.Focus(nil, 0, -1)
		require.Equal(t, "|1|1", evalToString(t, eval, focusExpr, root))
	})

	t.Run("replaces an earlier focus without changing the original", func(t *testing.T) {
		first := base.Focus(xpath3.SingleString("first").Get(0), 2, 2)
		second := first.Focus(nil, 1, 3)
		require.Equal(t, "first|2|2", evalToString(t, first, focusExpr, root))
		require.Equal(t, "|1|3", evalToString(t, second, focusExpr, root))
	})

	t.Run("variables are shared across evaluations", func(t *testing.T) {
		vars := map[string]xpath3.Sequence{
			"x": xpath3.SingleString("one"),
			"y": xpath3.SingleString("two"),
		}
		eval := base.Variables(vars)
		// Changing the caller's map after Variables must not affect the
		// evaluator, which cloned it.
		vars["x"] = xpath3.SingleString("changed")
		varsExpr, err := xpath3.NewCompiler().Compile(`concat($x, $y, position())`)
		require.NoError(t, err)
		require.Equal(t, "onetwo1", evalToString(t, eval.Focus(nil, 1, 2), varsExpr, root))
		require.Equal(t, "onetwo2", evalToString(t, eval.Focus(nil, 2, 2), varsExpr, root))
	})
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
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Functions(map[string]xpath3.Function{"capture": captured}, nil)
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

// ebvExprs cover every kind of result EvaluateEBV answers: node-list
// producers at the root (location paths, E1/path, unions, filters), node
// sequences built otherwise, single atomic values of every kind with an
// effective boolean value, sequences that have none (FORG0006), and dynamic
// errors.
var ebvExprs = []string{
	"/r", "/r/b/c", "b", ".", "..", "@*", "//@*[1]", "//text()", "(//b)[2]", "(//b)[100]",
	"//b | //c", "//b except //b", "//b intersect //b[c]", "(//b)[1]/c", "reverse(//b)/c", "//b[c][last()]",
	"//b[@id = 'b3']", "//b[number(substring(@id, 2)) > 2]", "$nodes", "$nodes//c", "$empty", "$nodes/nosuch",
	"(//b, 1)", "(1, //b)", "//b/string(@id)", "(//b)[1]/string(@id)", "//b/nosuch/string()",
	"()", "1", "0", "-0.0", "0.0e0", "1.5", "xs:double('NaN')", "xs:float('INF')", "''", "'a'", "true()",
	"false()", "xs:anyURI('')", "xs:anyURI('u')", "xs:untypedAtomic('')", "xs:untypedAtomic('x')",
	"xs:NCName('n')", "xs:date('2020-01-01')", "xs:QName('xs:int')", "(1, 2)", "('a', 'b')", "map{}",
	"[1]", "true#0", "exists(//b)", "not(//b)", "empty(//nosuch)", "count(//b) > 0", "//b and //c",
	"//nosuch or false()", "if (//b) then 0 else 1", "some $b in //b satisfies $b/c",
	"error()", "1 div 0", "//b[error()]", "//b[xs:integer('x')]", "1 + 'a'",
}

// ebvDoc is the document ebvExprs run against.
const ebvDoc = `<r><b id="b1"><c/></b><b id="b2">t</b><b id="b3"><c/><c/></b><!--x--></r>`

// requireEvaluateEBV checks that EvaluateEBV of expr from node gives the
// value and the error that Evaluate followed by Result.EBV gives.
func requireEvaluateEBV(t *testing.T, eval xpath3.Evaluator, node helium.Node, expr, msg string) {
	t.Helper()
	compiled, err := xpath3.NewCompiler().Compile(expr)
	require.NoError(t, err, expr)
	var want bool
	r, wantErr := eval.Evaluate(t.Context(), compiled, node)
	if wantErr == nil {
		want, wantErr = r.EBV()
	}
	got, gotErr := eval.EvaluateEBV(t.Context(), compiled, node)
	var wantX *xpath3.XPathError
	if wantErr != nil && !errors.As(wantErr, &wantX) {
		require.ErrorIs(t, gotErr, wantErr, msg)
		return
	}
	requireSameEBV(t, want, wantErr, got, gotErr, msg)
}

// TestEvaluateEBV checks that EvaluateEBV answers every kind of result with
// the value and error of Evaluate followed by Result.EBV, from every kind of
// context node and from an absent one.
func TestEvaluateEBV(t *testing.T) {
	t.Parallel()
	doc := mustParseXML(t, ebvDoc)
	all, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), xpath3.NewCompiler().MustCompile("//node() | //@*"), doc)
	require.NoError(t, err)
	nodes, err := all.Nodes()
	require.NoError(t, err)
	b, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(t.Context(), xpath3.NewCompiler().MustCompile("/r/b"), doc)
	require.NoError(t, err)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Variables(map[string]xpath3.Sequence{
		"nodes": b.Sequence(),
		"empty": xpath3.EmptySequence(),
	})
	contexts := append([]helium.Node{nil, doc}, nodes...)
	for _, expr := range slices.Concat(ebvExprs, resultEBVExprs) {
		for i, node := range contexts {
			requireEvaluateEBV(t, eval, node, expr, fmt.Sprintf("%s from context %d", expr, i))
		}
	}
}

// ebvSchema types the n elements of ebvSchemaDoc, so that their node items
// carry type annotations.
const ebvSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="IntList"><xs:list itemType="xs:int"/></xs:simpleType>
  <xs:element name="r">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="v" type="IntList" maxOccurs="unbounded"/>
        <xs:element name="n" type="xs:integer" maxOccurs="unbounded"/>
      </xs:sequence>
      <xs:attribute name="a" type="xs:integer"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

// annotatedEBVDoc returns a document of n integer elements validated
// against ebvSchema, and an evaluator that carries its type annotations.
func annotatedEBVDoc(t *testing.T, n int) (*helium.Document, xpath3.Evaluator) {
	t.Helper()
	ctx := t.Context()
	schema, err := xsd.NewCompiler().Compile(ctx, mustParseXML(t, ebvSchema))
	require.NoError(t, err)
	var sb strings.Builder
	sb.WriteString(`<r a="5"><v>1 2</v><v>3</v>`)
	for i := range n {
		fmt.Fprintf(&sb, "<n>%d</n>", i)
	}
	sb.WriteString("</r>")
	doc := mustParseXML(t, sb.String())
	ann := make(xsd.TypeAnnotations)
	require.NoError(t, xsd.NewValidator(schema).Annotations(&ann).Validate(ctx, doc))
	return doc, xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		SchemaDeclarations(schema.Declarations()).
		TypeAnnotations(ann)
}

// TestEvaluateEBVTypeAnnotations checks EvaluateEBV against Evaluate
// followed by Result.EBV under type annotations, where node results hold
// annotated items and predicates compare typed values.
func TestEvaluateEBVTypeAnnotations(t *testing.T) {
	t.Parallel()
	doc, eval := annotatedEBVDoc(t, 5)
	root := doc.DocumentElement()
	exprs := []string{
		"//n", "//v", "/r/*", "//nosuch", "//n[. = 3]", "//n[. > 3]", "//n[. = 30]", "//v[. = 2]", "//v[. = 7]",
		"(//n)[2]", "//n | //v", "/r[@a = 5]", "/r[@a = '5']", "@a", "data(@a)", "data(//n)", "data(//n[1])",
		"data(//n[2])", "data(//v[1])", "//n[1] = 0", "//n[. instance of xs:integer]", "exists(//n[. eq 4])",
		"//n[. eq 'x']", "sum(//n) > 0",
	}
	for _, expr := range exprs {
		for _, node := range []helium.Node{doc, root} {
			requireEvaluateEBV(t, eval, node, expr, expr)
		}
	}
}

// TestEvaluateEBVStopsEarly checks that a node path that is the whole
// expression stops at its first node under EvaluateEBV, with and without
// type annotations: over 2,000 elements it stays within an operation limit
// and a node-set limit of 100 that Evaluate exceeds. The limits of the nodes
// it does not reach no longer fire, and neither do the predicate errors of
// those nodes, as XPath 3.1 §2.3.4 permits.
func TestEvaluateEBVStopsEarly(t *testing.T) {
	t.Parallel()
	doc := earlyStopDoc(t, 2000)
	annotated, annotatedEval := annotatedEBVDoc(t, 2000)
	cases := []struct {
		eval xpath3.Evaluator
		doc  *helium.Document
		expr string
	}{
		{xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions), doc, "//r/b"},
		{xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions), doc, "//c"},
		{xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions), doc, "/r/b/c"},
		{xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions), doc, "//b/@id"},
		{xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions), doc, "//b[@id = '3']/c"},
		{xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions), doc, "(/r)/b"},
		{annotatedEval, annotated, "//n"},
		{annotatedEval, annotated, "/r/n"},
		{annotatedEval, annotated, "(/r)/n"},
	}
	for _, tc := range cases {
		compiled := xpath3.NewCompiler().MustCompile(tc.expr)
		for _, limited := range []xpath3.Evaluator{tc.eval.OpLimit(100), tc.eval.MaxNodesForTesting(100)} {
			got, err := limited.EvaluateEBV(t.Context(), compiled, tc.doc)
			require.NoError(t, err, tc.expr)
			require.True(t, got, tc.expr)
		}
		_, err := tc.eval.OpLimit(100).Evaluate(t.Context(), compiled, tc.doc)
		require.ErrorIs(t, err, xpath3.ErrOpLimit, tc.expr)
		_, err = tc.eval.MaxNodesForTesting(100).Evaluate(t.Context(), compiled, tc.doc)
		require.ErrorIs(t, err, xpath3.ErrNodeSetLimit, tc.expr)
	}

	compiled := xpath3.NewCompiler().MustCompile("//b[if (@id = '0') then true() else error()]")
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	got, err := eval.EvaluateEBV(t.Context(), compiled, doc)
	require.NoError(t, err)
	require.True(t, got)
	_, err = eval.Evaluate(t.Context(), compiled, doc)
	var xerr *xpath3.XPathError
	require.ErrorAs(t, err, &xerr)
	require.Equal(t, "FOER0000", xerr.Code)
}
