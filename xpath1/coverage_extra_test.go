package xpath1_test

import (
	"context"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath1"
	"github.com/stretchr/testify/require"
)

// --- Namespace axis ---

func TestEvalNamespaceAxis(t *testing.T) {
	doc := parseXML(t, `<root xmlns:a="urn:a" xmlns:b="urn:b"><child/></root>`)

	t.Run("wildcard selects namespace nodes", func(t *testing.T) {
		r, err := xpath1.Evaluate(t.Context(), doc, "/root/namespace::*")
		require.NoError(t, err)
		require.Equal(t, xpath1.NodeSetResult, r.Type)
		// at least the two declared namespaces (a, b) plus the implicit xml
		require.GreaterOrEqual(t, len(r.NodeSet), 2)
	})

	t.Run("named namespace node", func(t *testing.T) {
		r, err := xpath1.Evaluate(t.Context(), doc, "/root/namespace::a")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 1)
		require.Equal(t, "a", r.NodeSet[0].Name())
		require.Equal(t, "urn:a", string(r.NodeSet[0].Content()))
	})

	t.Run("named namespace node no match", func(t *testing.T) {
		r, err := xpath1.Evaluate(t.Context(), doc, "/root/namespace::zzz")
		require.NoError(t, err)
		require.Empty(t, r.NodeSet)
	})
}

// --- namespace-uri() function ---

func TestEvalNamespaceURIFunction(t *testing.T) {
	doc := parseXML(t, `<root xmlns:p="urn:x"><p:child/></root>`)

	t.Run("on prefixed element", func(t *testing.T) {
		nodes, err := xpath1.NewEvaluator().
			Namespaces(map[string]string{"p": nsURIX}).
			Find(t.Context(), xpath1.MustCompile("//p:child"), doc)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		r, err := xpath1.Evaluate(t.Context(), nodes[0], "namespace-uri(.)")
		require.NoError(t, err)
		require.Equal(t, xpath1.StringResult, r.Type)
		require.Equal(t, nsURIX, r.String)
	})

	t.Run("no-namespace element", func(t *testing.T) {
		r, err := xpath1.Evaluate(t.Context(), doc, "namespace-uri(/root)")
		require.NoError(t, err)
		require.Equal(t, "", r.String)
	})

	t.Run("empty node-set arg yields empty string", func(t *testing.T) {
		r, err := xpath1.Evaluate(t.Context(), doc, "namespace-uri(/root/nonexistent)")
		require.NoError(t, err)
		require.Equal(t, xpath1.StringResult, r.Type)
		require.Equal(t, "", r.String)
	})
}

// --- name()/local-name() on empty and non-element nodes ---

func TestEvalNameFunctionsEmptyAndContext(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	root := docElement(doc)

	t.Run("name() context node default", func(t *testing.T) {
		r, err := xpath1.Evaluate(t.Context(), root, "name()")
		require.NoError(t, err)
		require.Equal(t, "root", r.String)
	})

	t.Run("local-name() empty node-set", func(t *testing.T) {
		r, err := xpath1.Evaluate(t.Context(), doc, "local-name(/root/nope)")
		require.NoError(t, err)
		require.Equal(t, "", r.String)
	})

	t.Run("name() of text node is empty", func(t *testing.T) {
		txtdoc := parseXML(t, `<root>hi</root>`)
		nodes, err := xpath1.Find(t.Context(), txtdoc, "/root/text()")
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		r, err := xpath1.Evaluate(t.Context(), nodes[0], "name(.)")
		require.NoError(t, err)
		require.Equal(t, "", r.String)
	})
}

// --- Filter expression with predicate ---

func TestEvalFilterExpr(t *testing.T) {
	doc := parseXML(t, `<root><a/><a/><a/></root>`)
	// (/root/a)[2] is a primary-expr (parenthesized) filtered by a predicate.
	r, err := xpath1.Evaluate(t.Context(), doc, "(/root/a)[2]")
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, r.Type)
	require.Len(t, r.NodeSet, 1)
}

func TestEvalFilterExprNotNodeSet(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	// A non-node-set primary expr followed by a predicate must error.
	_, err := xpath1.Evaluate(t.Context(), doc, "(1+2)[1]")
	require.Error(t, err)
	require.ErrorIs(t, err, xpath1.ErrFilterNotNodeSet)
}

// --- Path expression base not a node-set ---

func TestEvalPathExprNotNodeSet(t *testing.T) {
	doc := parseXML(t, `<root><a/></root>`)
	compiled, err := xpath1.Compile("ext:scalar()/a")
	require.NoError(t, err)
	ev := xpath1.NewEvaluator().
		Namespaces(map[string]string{nsPrefixExt: nsURIExt}).
		FunctionNS(nsURIExt, "scalar", xpath1.FunctionFunc(func(_ context.Context, _ []*xpath1.Result) (*xpath1.Result, error) {
			return &xpath1.Result{Type: xpath1.StringResult, String: "x"}, nil
		}))
	_, err = ev.Evaluate(t.Context(), compiled, doc)
	require.Error(t, err)
	require.ErrorIs(t, err, xpath1.ErrPathNotNodeSet)
}

// --- Union operator non-node-set ---

func TestEvalUnionNotNodeSet(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	compiled, err := xpath1.Compile("ext:scalar() | /root")
	require.NoError(t, err)
	ev := xpath1.NewEvaluator().
		Namespaces(map[string]string{nsPrefixExt: nsURIExt}).
		FunctionNS(nsURIExt, "scalar", xpath1.FunctionFunc(func(_ context.Context, _ []*xpath1.Result) (*xpath1.Result, error) {
			return &xpath1.Result{Type: xpath1.StringResult, String: "x"}, nil
		}))
	_, err = ev.Evaluate(t.Context(), compiled, doc)
	require.Error(t, err)
	require.ErrorIs(t, err, xpath1.ErrUnionNotNodeSet)
}

// --- Variable error paths ---

func TestEvalVariableUndefined(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	compiled, err := xpath1.Compile("$x")
	require.NoError(t, err)

	t.Run("no variables configured", func(t *testing.T) {
		_, err := compiled.Evaluate(t.Context(), doc)
		require.ErrorIs(t, err, xpath1.ErrUndefinedVariable)
	})

	t.Run("variable not in map", func(t *testing.T) {
		_, err := xpath1.NewEvaluator().
			Variables(map[string]any{"y": float64(1)}).
			Evaluate(t.Context(), compiled, doc)
		require.ErrorIs(t, err, xpath1.ErrUndefinedVariable)
	})

	t.Run("unsupported variable type", func(t *testing.T) {
		_, err := xpath1.NewEvaluator().
			Variables(map[string]any{"x": []int{1, 2}}).
			Evaluate(t.Context(), compiled, doc)
		require.ErrorIs(t, err, xpath1.ErrUnsupportedVariableType)
	})
}

func TestEvalVariableStringAndBool(t *testing.T) {
	doc := parseXML(t, `<root/>`)

	t.Run("string var", func(t *testing.T) {
		compiled := xpath1.MustCompile("$s")
		r, err := xpath1.NewEvaluator().
			Variables(map[string]any{"s": "hello"}).
			Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		require.Equal(t, xpath1.StringResult, r.Type)
		require.Equal(t, "hello", r.String)
	})

	t.Run("bool var", func(t *testing.T) {
		compiled := xpath1.MustCompile("$b")
		r, err := xpath1.NewEvaluator().
			Variables(map[string]any{"b": true}).
			Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		require.Equal(t, xpath1.BooleanResult, r.Type)
		require.True(t, r.Bool)
	})
}

// --- Comparison branch coverage ---

func TestEvalComparisonBranches(t *testing.T) {
	doc := parseXML(t, `<root><a>5</a><b>5</b><c>10</c></root>`)
	for _, tc := range []struct {
		expr string
		want bool
	}{
		// node-set vs node-set, string equality
		{`/root/a = /root/b`, true},
		{`/root/a = /root/c`, false},
		{`/root/a < /root/c`, true},
		{`/root/c > /root/a`, true},
		// node-set (right) vs number scalar (left) -- exercises compareNodeSetRight
		{`5 = /root/a`, true},
		{`10 > /root/a`, true},
		{`3 < /root/a`, true},
		// node-set vs number scalar (left node-set)
		{`/root/a = 5`, true},
		{`/root/c >= 10`, true},
		// scalar number vs number relational
		{`5 < 10`, true},
		{`10 <= 10`, true},
		// string vs string equality and inequality
		{`'x' = 'x'`, true},
		{`'x' != 'y'`, true},
		// string relational coerces to numbers
		{`'5' < '10'`, true},
		// boolean scalar equality
		{`true() = true()`, true},
		{`true() != false()`, true},
		// number vs string equality (number coercion)
		{`5 = '5'`, true},
	} {
		t.Run(tc.expr, func(t *testing.T) {
			r, err := xpath1.Evaluate(t.Context(), doc, tc.expr)
			require.NoError(t, err)
			require.Equal(t, xpath1.BooleanResult, r.Type)
			require.Equal(t, tc.want, r.Bool)
		})
	}
}

// --- Type-test predicate matching alternate node types ---

func TestEvalProcessingInstruction(t *testing.T) {
	doc := parseXML(t, `<root><?go run?><?stop now?></root>`)

	t.Run("all PIs", func(t *testing.T) {
		r, err := xpath1.Evaluate(t.Context(), doc, "/root/processing-instruction()")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 2)
	})

	t.Run("named PI", func(t *testing.T) {
		r, err := xpath1.Evaluate(t.Context(), doc, "/root/processing-instruction('go')")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 1)
		require.Equal(t, helium.ProcessingInstructionNode, r.NodeSet[0].Type())
	})
}

func TestEvalCDATAAsText(t *testing.T) {
	doc := parseXML(t, `<root><![CDATA[hello]]></root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "/root/text()")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
}

// --- Wildcard with prefix on attribute axis and element axis ---

func TestEvalPrefixedWildcard(t *testing.T) {
	doc := parseXML(t, `<root xmlns:p="urn:x"><p:a/><p:b/><c/></root>`)
	nodes, err := xpath1.NewEvaluator().
		Namespaces(map[string]string{"p": nsURIX}).
		Find(t.Context(), xpath1.MustCompile("/root/p:*"), doc)
	require.NoError(t, err)
	require.Len(t, nodes, 2)
}

// --- xml prefix is always bound ---

func TestEvalXMLPrefixAlwaysBound(t *testing.T) {
	doc := parseXML(t, `<root xml:space="preserve"/>`)
	root := docElement(doc)
	nodes, err := xpath1.Find(t.Context(), root, "@xml:space")
	require.NoError(t, err)
	require.Len(t, nodes, 1)
}

// --- string() with no argument uses context node ---

func TestEvalStringNoArg(t *testing.T) {
	doc := parseXML(t, `<root>content</root>`)
	root := docElement(doc)
	r, err := xpath1.Evaluate(t.Context(), root, "string()")
	require.NoError(t, err)
	require.Equal(t, "content", r.String)
}

func TestEvalStringLengthNoArg(t *testing.T) {
	doc := parseXML(t, `<root>hello</root>`)
	root := docElement(doc)
	r, err := xpath1.Evaluate(t.Context(), root, "string-length()")
	require.NoError(t, err)
	require.Equal(t, 5.0, r.Number)
}

func TestEvalNormalizeSpaceNoArg(t *testing.T) {
	doc := parseXML(t, `<root>  a   b  </root>`)
	root := docElement(doc)
	r, err := xpath1.Evaluate(t.Context(), root, "normalize-space()")
	require.NoError(t, err)
	require.Equal(t, "a b", r.String)
}

func TestEvalNumberNoArg(t *testing.T) {
	doc := parseXML(t, `<root>3.5</root>`)
	root := docElement(doc)
	r, err := xpath1.Evaluate(t.Context(), root, "number()")
	require.NoError(t, err)
	require.Equal(t, 3.5, r.Number)
}

func TestEvalLastInPredicate(t *testing.T) {
	doc := parseXML(t, `<root><a/><a/><a/></root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "count(/root/a[position()=last()])")
	require.NoError(t, err)
	require.Equal(t, 1.0, r.Number)
}

// --- round() special values ---

func TestEvalRoundSpecial(t *testing.T) {
	doc := parseXML(t, `<root>abc</root>`)

	t.Run("NaN passthrough", func(t *testing.T) {
		r, err := xpath1.Evaluate(t.Context(), doc, "round(number(/root))")
		require.NoError(t, err)
		require.Equal(t, xpath1.NumberResult, r.Type)
	})

	t.Run("negative half rounds toward zero", func(t *testing.T) {
		r, err := xpath1.Evaluate(t.Context(), doc, "round(-0.5)")
		require.NoError(t, err)
		require.Equal(t, 0.0, r.Number)
	})
}

// --- substring edge cases ---

func TestEvalSubstringEdges(t *testing.T) {
	doc := parseXML(t, `<root/>`)

	t.Run("two-arg NaN start yields empty", func(t *testing.T) {
		r, err := xpath1.Evaluate(t.Context(), doc, "substring('hello', number('x'))")
		require.NoError(t, err)
		require.Equal(t, "", r.String)
	})

	t.Run("three-arg negative start", func(t *testing.T) {
		r, err := xpath1.Evaluate(t.Context(), doc, "substring('12345', 0, 3)")
		require.NoError(t, err)
		require.Equal(t, "12", r.String)
	})
}

// --- translate removal path ---

func TestEvalTranslateRemoval(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	// "to" shorter than "from": extra "from" chars are removed.
	r, err := xpath1.Evaluate(t.Context(), doc, "translate('abcdef', 'abcdef', 'xy')")
	require.NoError(t, err)
	require.Equal(t, "xy", r.String)
}

// --- arithmetic with string/nan operands ---

func TestEvalArithmeticCoercion(t *testing.T) {
	doc := parseXML(t, `<root><n>4</n></root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "/root/n - 1")
	require.NoError(t, err)
	require.Equal(t, 3.0, r.Number)
}

// --- sum on node-set ---

func TestEvalSumString(t *testing.T) {
	doc := parseXML(t, `<root><n>1.5</n><n>2.5</n></root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "sum(/root/n)")
	require.NoError(t, err)
	require.Equal(t, 4.0, r.Number)
}
