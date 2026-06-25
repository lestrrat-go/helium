package xpath1_test

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath1"
	"github.com/stretchr/testify/require"
)

var (
	errDoubleOneArg = errors.New("double() takes exactly 1 argument")
	errHelloOneArg  = errors.New("hello() takes exactly 1 argument")
)

func parseXML(t *testing.T, s string) *helium.Document {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
	require.NoError(t, err)
	return doc
}

func docElement(doc *helium.Document) helium.Node {
	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		if n.Type() == helium.ElementNode {
			return n
		}
	}
	return nil
}

func TestEvalPath(t *testing.T) {
	t.Run("root path", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "/")
		require.NoError(t, err)
		require.Equal(t, xpath1.NodeSetResult, r.Type)
		require.Len(t, r.NodeSet, 1)
		require.Equal(t, helium.DocumentNode, r.NodeSet[0].Type())
	})

	t.Run("absolute path nil context node", func(t *testing.T) {
		expr, err := xpath1.Compile("/")
		require.NoError(t, err)
		// A nil context node must produce an evaluation error, not a panic.
		_, err = expr.Evaluate(t.Context(), nil)
		require.Error(t, err)
		require.ErrorIs(t, err, xpath1.ErrNoContextNode)
	})

	t.Run("absolute child", func(t *testing.T) {
		doc := parseXML(t, `<root><a/><b/></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "/root/a")
		require.NoError(t, err)
		require.Equal(t, xpath1.NodeSetResult, r.Type)
		require.Len(t, r.NodeSet, 1)
		require.Equal(t, "a", r.NodeSet[0].Name())
	})

	t.Run("relative child", func(t *testing.T) {
		doc := parseXML(t, `<root><a><b/></a></root>`)
		root := docElement(doc)
		r, err := xpath1.Evaluate(t.Context(), root, "a/b")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 1)
		require.Equal(t, "b", r.NodeSet[0].Name())
	})

	t.Run("double slash", func(t *testing.T) {
		doc := parseXML(t, `<root><a><b/></a><b/></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "//b")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 2)
	})

	t.Run("dot", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		root := docElement(doc)
		r, err := xpath1.Evaluate(t.Context(), root, ".")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 1)
		require.Equal(t, root, r.NodeSet[0])
	})

	t.Run("dot-dot", func(t *testing.T) {
		doc := parseXML(t, `<root><a/></root>`)
		root := docElement(doc)
		a := root.FirstChild()
		r, err := xpath1.Evaluate(t.Context(), a, "..")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 1)
		require.Equal(t, root, r.NodeSet[0])
	})

	// unprefixed name test matches only no-namespace nodes. XPath 1.0 has no
	// default element namespace, so a node in a namespace must not match an
	// unprefixed test.
	t.Run("unprefixed name test matches no-namespace only", func(t *testing.T) {
		doc := parseXML(t, `<root><item xmlns="http://ex">A</item><item xmlns="">B</item></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "//item")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 1)
		require.Equal(t, "B", string(r.NodeSet[0].Content()))
	})

	// An undeclared prefix must not match a node merely because the document uses
	// the same lexical prefix; prefixes resolve from the evaluation namespace
	// context. With a binding supplied, the match succeeds.
	t.Run("undeclared prefix does not match lexically", func(t *testing.T) {
		doc := parseXML(t, `<root xmlns:p="urn:x"><p:foo/></root>`)

		r, err := xpath1.Evaluate(t.Context(), doc, "//p:foo")
		require.NoError(t, err)
		require.Empty(t, r.NodeSet, "unbound prefix must not match by lexical prefix")

		expr, err := xpath1.Compile("//p:foo")
		require.NoError(t, err)
		r2, err := xpath1.NewEvaluator().
			Namespaces(map[string]string{"p": nsURIX}).
			Evaluate(t.Context(), expr, doc)
		require.NoError(t, err)
		require.Len(t, r2.NodeSet, 1, "bound prefix must match")
	})

	t.Run("path expr not node-set", func(t *testing.T) {
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
	})
}

func TestEvalAxes(t *testing.T) {
	t.Run("attribute", func(t *testing.T) {
		doc := parseXML(t, `<root id="123"/>`)
		root := docElement(doc)
		r, err := xpath1.Evaluate(t.Context(), root, "@id")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 1)
		attr, ok := r.NodeSet[0].(*helium.Attribute)
		require.True(t, ok)
		require.Equal(t, "123", attr.Value())
	})

	t.Run("descendant", func(t *testing.T) {
		doc := parseXML(t, `<root><a><b><c/></b></a></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "/root/descendant::c")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 1)
		require.Equal(t, "c", r.NodeSet[0].Name())
	})

	t.Run("ancestor", func(t *testing.T) {
		doc := parseXML(t, `<root><a><b/></a></root>`)
		nodes, err := xpath1.Find(t.Context(), doc, "//b")
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		b := nodes[0]
		r, err := xpath1.Evaluate(t.Context(), b, "ancestor::root")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 1)
		require.Equal(t, "root", r.NodeSet[0].Name())
	})

	t.Run("following-sibling", func(t *testing.T) {
		doc := parseXML(t, `<root><a/><b/><c/></root>`)
		nodes, err := xpath1.Find(t.Context(), doc, "/root/a")
		require.NoError(t, err)
		r, err := xpath1.Evaluate(t.Context(), nodes[0], "following-sibling::*")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 2)
		require.Equal(t, "b", r.NodeSet[0].Name())
		require.Equal(t, "c", r.NodeSet[1].Name())
	})

	t.Run("preceding-sibling", func(t *testing.T) {
		doc := parseXML(t, `<root><a/><b/><c/></root>`)
		nodes, err := xpath1.Find(t.Context(), doc, "/root/c")
		require.NoError(t, err)
		r, err := xpath1.Evaluate(t.Context(), nodes[0], "preceding-sibling::*")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 2)
	})

	t.Run("self", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		root := docElement(doc)
		r, err := xpath1.Evaluate(t.Context(), root, "self::root")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 1)
		require.Equal(t, root, r.NodeSet[0])
	})

	t.Run("self no match", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		root := docElement(doc)
		r, err := xpath1.Evaluate(t.Context(), root, "self::other")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 0)
	})

	t.Run("descendant-or-self", func(t *testing.T) {
		doc := parseXML(t, `<root><a><b/></a></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "/root/descendant-or-self::*")
		require.NoError(t, err)
		// root, a, b
		require.Len(t, r.NodeSet, 3)
	})

	t.Run("ancestor-or-self", func(t *testing.T) {
		doc := parseXML(t, `<root><a><b/></a></root>`)
		nodes, err := xpath1.Find(t.Context(), doc, "//b")
		require.NoError(t, err)
		r, err := xpath1.Evaluate(t.Context(), nodes[0], "ancestor-or-self::*")
		require.NoError(t, err)
		// b, a, root
		require.Len(t, r.NodeSet, 3)
	})

	t.Run("namespace", func(t *testing.T) {
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
	})
}

func TestEvalWildcard(t *testing.T) {
	t.Run("unprefixed", func(t *testing.T) {
		doc := parseXML(t, `<root><a/><b/><c/></root>`)
		root := docElement(doc)
		r, err := xpath1.Evaluate(t.Context(), root, "*")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 3)
	})

	t.Run("prefixed", func(t *testing.T) {
		doc := parseXML(t, `<root xmlns:p="urn:x"><p:a/><p:b/><c/></root>`)
		nodes, err := xpath1.NewEvaluator().
			Namespaces(map[string]string{"p": nsURIX}).
			Find(t.Context(), xpath1.MustCompile("/root/p:*"), doc)
		require.NoError(t, err)
		require.Len(t, nodes, 2)
	})

	t.Run("xml prefix always bound", func(t *testing.T) {
		doc := parseXML(t, `<root xml:space="preserve"/>`)
		root := docElement(doc)
		nodes, err := xpath1.Find(t.Context(), root, "@xml:space")
		require.NoError(t, err)
		require.Len(t, nodes, 1)
	})
}

func TestEvalPredicate(t *testing.T) {
	t.Run("position", func(t *testing.T) {
		doc := parseXML(t, `<root><a/><a/><a/></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "/root/a[2]")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 1)
	})

	t.Run("last", func(t *testing.T) {
		doc := parseXML(t, `<root><a/><a/><a/></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "/root/a[last()]")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 1)
	})

	t.Run("boolean", func(t *testing.T) {
		doc := parseXML(t, `<root><a x="1"/><a/><a x="2"/></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "/root/a[@x]")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 2)
	})

	t.Run("count with predicate", func(t *testing.T) {
		doc := parseXML(t, `<root><a x="1"/><a/><a x="2"/></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "count(/root/a[@x])")
		require.NoError(t, err)
		require.Equal(t, 2.0, r.Number)
	})

	t.Run("last in predicate", func(t *testing.T) {
		doc := parseXML(t, `<root><a/><a/><a/></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "count(/root/a[position()=last()])")
		require.NoError(t, err)
		require.Equal(t, 1.0, r.Number)
	})
}

func TestEvalNodeTest(t *testing.T) {
	t.Run("node()", func(t *testing.T) {
		doc := parseXML(t, `<root><a/>text<!-- c --></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "/root/node()")
		require.NoError(t, err)
		// Should include element, text, and comment
		require.GreaterOrEqual(t, len(r.NodeSet), 3)
	})

	t.Run("text()", func(t *testing.T) {
		doc := parseXML(t, `<root>hello</root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "/root/text()")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 1)
		require.Equal(t, helium.TextNode, r.NodeSet[0].Type())
		require.Equal(t, "hello", string(r.NodeSet[0].Content()))
	})

	t.Run("comment()", func(t *testing.T) {
		doc := parseXML(t, `<root><!-- a comment --></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "/root/comment()")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 1)
		require.Equal(t, helium.CommentNode, r.NodeSet[0].Type())
	})

	t.Run("CDATA as text", func(t *testing.T) {
		doc := parseXML(t, `<root><![CDATA[hello]]></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "/root/text()")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 1)
	})

	t.Run("processing-instruction()", func(t *testing.T) {
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
	})
}

func TestEvalComparison(t *testing.T) {
	t.Run("equals", func(t *testing.T) {
		doc := parseXML(t, `<root><a>hello</a></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "/root/a = 'hello'")
		require.NoError(t, err)
		require.Equal(t, xpath1.BooleanResult, r.Type)
		require.True(t, r.Bool)
	})

	t.Run("not equals", func(t *testing.T) {
		doc := parseXML(t, `<root><a>hello</a></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "/root/a != 'world'")
		require.NoError(t, err)
		require.True(t, r.Bool)
	})

	t.Run("numeric", func(t *testing.T) {
		doc := parseXML(t, `<root><price>35</price></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "/root/price > 30")
		require.NoError(t, err)
		require.True(t, r.Bool)
	})

	t.Run("node-set boolean", func(t *testing.T) {
		doc := parseXML(t, `<root><a/></root>`)
		for _, tc := range []struct {
			expr string
			want bool
		}{
			// node-set vs boolean: collapse the whole node-set to a boolean
			{`/root/a = false()`, false},
			{`/root/a = true()`, true},
			{`/root/a != false()`, true},
			{`/root/nonexistent = false()`, true},
			{`/root/nonexistent = true()`, false},
			// mirrored (boolean on the left)
			{`false() = /root/a`, false},
			{`true() = /root/a`, true},
			{`false() = /root/nonexistent`, true},
			// relational operators: boolean(node-set) compared numerically (true=1)
			{`/root/a > false()`, true},          // 1 > 0
			{`/root/a < true()`, false},          // 1 < 1
			{`/root/a >= true()`, true},          // 1 >= 1
			{`/root/nonexistent < true()`, true}, // 0 < 1
			{`false() < /root/a`, true},          // mirrored: 0 < 1
			// GUARD: number/string node-set comparisons keep existential semantics
			{`/root/a = ''`, true},
			{`/root/a = 'x'`, false},
			{`/root/a = 0`, false},
		} {
			t.Run(tc.expr, func(t *testing.T) {
				r, err := xpath1.Evaluate(t.Context(), doc, tc.expr)
				require.NoError(t, err)
				require.Equal(t, xpath1.BooleanResult, r.Type)
				require.Equal(t, tc.want, r.Bool)
			})
		}
	})

	t.Run("branches", func(t *testing.T) {
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
	})
}

func TestEvalArithmetic(t *testing.T) {
	t.Run("addition", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "1 + 2")
		require.NoError(t, err)
		require.Equal(t, xpath1.NumberResult, r.Type)
		require.Equal(t, 3.0, r.Number)
	})

	t.Run("multiplication", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "3 * 4")
		require.NoError(t, err)
		require.Equal(t, 12.0, r.Number)
	})

	t.Run("division", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "10 div 3")
		require.NoError(t, err)
		require.InDelta(t, 3.333, r.Number, 0.01)
	})

	t.Run("mod", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "10 mod 3")
		require.NoError(t, err)
		require.Equal(t, 1.0, r.Number)
	})

	t.Run("negation", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "-5")
		require.NoError(t, err)
		require.Equal(t, -5.0, r.Number)
	})

	t.Run("coercion", func(t *testing.T) {
		doc := parseXML(t, `<root><n>4</n></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "/root/n - 1")
		require.NoError(t, err)
		require.Equal(t, 3.0, r.Number)
	})

	t.Run("sum string", func(t *testing.T) {
		doc := parseXML(t, `<root><n>1.5</n><n>2.5</n></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "sum(/root/n)")
		require.NoError(t, err)
		require.Equal(t, 4.0, r.Number)
	})
}

func TestEvalLogical(t *testing.T) {
	t.Run("or", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "true() or false()")
		require.NoError(t, err)
		require.True(t, r.Bool)
	})

	t.Run("and", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "true() and false()")
		require.NoError(t, err)
		require.False(t, r.Bool)
	})
}

func TestEvalUnion(t *testing.T) {
	t.Run("node-sets", func(t *testing.T) {
		doc := parseXML(t, `<root><a/><b/></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "/root/a | /root/b")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 2)
	})

	t.Run("not node-set", func(t *testing.T) {
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
	})
}

func TestEvalStringFunctions(t *testing.T) {
	t.Run("string value", func(t *testing.T) {
		doc := parseXML(t, `<root>hello</root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "string(/root)")
		require.NoError(t, err)
		require.Equal(t, "hello", r.String)
	})

	t.Run("concat", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "concat('a', 'b', 'c')")
		require.NoError(t, err)
		require.Equal(t, "abc", r.String)
	})

	t.Run("starts-with", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "starts-with('hello', 'hel')")
		require.NoError(t, err)
		require.True(t, r.Bool)
	})

	t.Run("contains", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "contains('hello world', 'world')")
		require.NoError(t, err)
		require.True(t, r.Bool)
	})

	t.Run("substring-before", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "substring-before('1999/04/01', '/')")
		require.NoError(t, err)
		require.Equal(t, "1999", r.String)
	})

	t.Run("substring-after", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "substring-after('1999/04/01', '/')")
		require.NoError(t, err)
		require.Equal(t, "04/01", r.String)
	})

	t.Run("substring", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)

		r, err := xpath1.Evaluate(t.Context(), doc, "substring('12345', 2, 3)")
		require.NoError(t, err)
		require.Equal(t, "234", r.String)

		r, err = xpath1.Evaluate(t.Context(), doc, "substring('12345', 2)")
		require.NoError(t, err)
		require.Equal(t, "2345", r.String)
	})

	t.Run("string-length", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "string-length('hello')")
		require.NoError(t, err)
		require.Equal(t, 5.0, r.Number)
	})

	t.Run("normalize-space", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "normalize-space('  hello   world  ')")
		require.NoError(t, err)
		require.Equal(t, "hello world", r.String)
	})

	t.Run("translate", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "translate('bar', 'abc', 'ABC')")
		require.NoError(t, err)
		require.Equal(t, "BAr", r.String)
	})

	t.Run("string literal", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "'hello'")
		require.NoError(t, err)
		require.Equal(t, xpath1.StringResult, r.Type)
		require.Equal(t, "hello", r.String)
	})

	t.Run("string no arg", func(t *testing.T) {
		doc := parseXML(t, `<root>content</root>`)
		root := docElement(doc)
		r, err := xpath1.Evaluate(t.Context(), root, "string()")
		require.NoError(t, err)
		require.Equal(t, "content", r.String)
	})

	t.Run("string-length no arg", func(t *testing.T) {
		doc := parseXML(t, `<root>hello</root>`)
		root := docElement(doc)
		r, err := xpath1.Evaluate(t.Context(), root, "string-length()")
		require.NoError(t, err)
		require.Equal(t, 5.0, r.Number)
	})

	t.Run("normalize-space no arg", func(t *testing.T) {
		doc := parseXML(t, `<root>  a   b  </root>`)
		root := docElement(doc)
		r, err := xpath1.Evaluate(t.Context(), root, "normalize-space()")
		require.NoError(t, err)
		require.Equal(t, "a b", r.String)
	})

	t.Run("substring edges", func(t *testing.T) {
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
	})

	t.Run("translate removal", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		// "to" shorter than "from": extra "from" chars are removed.
		r, err := xpath1.Evaluate(t.Context(), doc, "translate('abcdef', 'abcdef', 'xy')")
		require.NoError(t, err)
		require.Equal(t, "xy", r.String)
	})
}

func TestEvalNumericFunctions(t *testing.T) {
	t.Run("not", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "not(false())")
		require.NoError(t, err)
		require.True(t, r.Bool)
	})

	t.Run("boolean", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "boolean(1)")
		require.NoError(t, err)
		require.True(t, r.Bool)

		r, err = xpath1.Evaluate(t.Context(), doc, "boolean(0)")
		require.NoError(t, err)
		require.False(t, r.Bool)
	})

	t.Run("count", func(t *testing.T) {
		doc := parseXML(t, `<root><a/><a/><a/></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "count(/root/a)")
		require.NoError(t, err)
		require.Equal(t, 3.0, r.Number)
	})

	t.Run("sum", func(t *testing.T) {
		doc := parseXML(t, `<root><n>1</n><n>2</n><n>3</n></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "sum(/root/n)")
		require.NoError(t, err)
		require.Equal(t, 6.0, r.Number)
	})

	t.Run("floor", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "floor(2.7)")
		require.NoError(t, err)
		require.Equal(t, 2.0, r.Number)
	})

	t.Run("ceiling", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "ceiling(2.3)")
		require.NoError(t, err)
		require.Equal(t, 3.0, r.Number)
	})

	t.Run("round", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "round(2.5)")
		require.NoError(t, err)
		require.Equal(t, 3.0, r.Number)

		r, err = xpath1.Evaluate(t.Context(), doc, "round(2.4)")
		require.NoError(t, err)
		require.Equal(t, 2.0, r.Number)
	})

	t.Run("number", func(t *testing.T) {
		doc := parseXML(t, `<root>42</root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "number(/root)")
		require.NoError(t, err)
		require.Equal(t, 42.0, r.Number)
	})

	t.Run("number NaN", func(t *testing.T) {
		doc := parseXML(t, `<root>abc</root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "number(/root)")
		require.NoError(t, err)
		require.True(t, math.IsNaN(r.Number))
	})

	t.Run("number no arg", func(t *testing.T) {
		doc := parseXML(t, `<root>3.5</root>`)
		root := docElement(doc)
		r, err := xpath1.Evaluate(t.Context(), root, "number()")
		require.NoError(t, err)
		require.Equal(t, 3.5, r.Number)
	})

	t.Run("round special", func(t *testing.T) {
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
	})
}

func TestEvalNodeFunctions(t *testing.T) {
	t.Run("local-name", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "local-name(/root)")
		require.NoError(t, err)
		require.Equal(t, "root", r.String)
	})

	t.Run("name", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		r, err := xpath1.Evaluate(t.Context(), doc, "name(/root)")
		require.NoError(t, err)
		require.Equal(t, "root", r.String)
	})

	t.Run("namespace-uri", func(t *testing.T) {
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
	})

	t.Run("empty and context", func(t *testing.T) {
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
	})
}

func TestEvalVariable(t *testing.T) {
	t.Run("number", func(t *testing.T) {
		doc := parseXML(t, `<root><a/><b/></root>`)
		expr, err := xpath1.Compile("$x + 1")
		require.NoError(t, err)
		r, err := xpath1.NewEvaluator().Variables(map[string]any{
			"x": float64(41),
		}).Evaluate(t.Context(), expr, doc)
		require.NoError(t, err)
		require.Equal(t, 42.0, r.Number)
	})

	t.Run("node-set document order", func(t *testing.T) {
		doc := parseXML(t, `<root><a>AAA</a><b>BBB</b></root>`)
		nodes, err := xpath1.Find(t.Context(), doc, "/root/*")
		require.NoError(t, err)
		require.Len(t, nodes, 2)
		a, b := nodes[0], nodes[1]
		require.Equal(t, "a", a.Name())
		require.Equal(t, "b", b.Name())

		// Bind $v in reversed (non-document) order. string($v) must return the
		// value of the first node in document order ("AAA"), not the slice's
		// first element ("BBB").
		expr, err := xpath1.Compile("string($v)")
		require.NoError(t, err)
		r, err := xpath1.NewEvaluator().Variables(map[string]any{
			"v": []helium.Node{b, a},
		}).Evaluate(t.Context(), expr, doc)
		require.NoError(t, err)
		require.Equal(t, "AAA", r.String)
	})

	t.Run("node-set typed nil", func(t *testing.T) {
		doc := parseXML(t, `<root><a/></root>`)
		a := docElement(doc)
		require.NotNil(t, a)

		// A typed-nil concrete node pointer is non-nil at the interface level,
		// so a naive `n != nil` filter lets it through and DeduplicateNodes
		// panics dereferencing it. It must be filtered out instead.
		expr, err := xpath1.Compile("count($v)")
		require.NoError(t, err)
		r, err := xpath1.NewEvaluator().Variables(map[string]any{
			"v": []helium.Node{a, (*helium.Element)(nil)},
		}).Evaluate(t.Context(), expr, doc)
		require.NoError(t, err)
		require.Equal(t, 1.0, r.Number)
	})

	t.Run("undefined", func(t *testing.T) {
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
	})

	t.Run("string and bool", func(t *testing.T) {
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
	})
}

func TestEvalFilterExpr(t *testing.T) {
	t.Run("with predicate", func(t *testing.T) {
		doc := parseXML(t, `<root><a/><a/><a/></root>`)
		// (/root/a)[2] is a primary-expr (parenthesized) filtered by a predicate.
		r, err := xpath1.Evaluate(t.Context(), doc, "(/root/a)[2]")
		require.NoError(t, err)
		require.Equal(t, xpath1.NodeSetResult, r.Type)
		require.Len(t, r.NodeSet, 1)
	})

	t.Run("not node-set", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		// A non-node-set primary expr followed by a predicate must error.
		_, err := xpath1.Evaluate(t.Context(), doc, "(1+2)[1]")
		require.Error(t, err)
		require.ErrorIs(t, err, xpath1.ErrFilterNotNodeSet)
	})
}

func TestEvalID(t *testing.T) {
	t.Run("xml:id", func(t *testing.T) {
		// xml:id should be recognized without a DTD
		doc := parseXML(t, `<root><a xml:id="foo">A</a><b xml:id="bar">B</b></root>`)

		t.Run("single id", func(t *testing.T) {
			r, err := xpath1.Evaluate(t.Context(), doc, `id("foo")`)
			require.NoError(t, err)
			require.Equal(t, xpath1.NodeSetResult, r.Type)
			require.Len(t, r.NodeSet, 1)
			require.Equal(t, "a", r.NodeSet[0].Name())
		})

		t.Run("multiple ids space-separated", func(t *testing.T) {
			r, err := xpath1.Evaluate(t.Context(), doc, `id("foo bar")`)
			require.NoError(t, err)
			require.Equal(t, xpath1.NodeSetResult, r.Type)
			require.Len(t, r.NodeSet, 2)
		})

		t.Run("nonexistent id", func(t *testing.T) {
			r, err := xpath1.Evaluate(t.Context(), doc, `id("nonexistent")`)
			require.NoError(t, err)
			require.Equal(t, xpath1.NodeSetResult, r.Type)
			require.Len(t, r.NodeSet, 0)
		})
	})

	t.Run("DTD", func(t *testing.T) {
		// DTD-declared ID attribute
		doc := parseXML(t, `<!DOCTYPE root [
			<!ELEMENT root (item*)>
			<!ELEMENT item (#PCDATA)>
			<!ATTLIST item myid ID #IMPLIED>
		]>
		<root><item myid="x1">first</item><item myid="x2">second</item></root>`)

		r, err := xpath1.Evaluate(t.Context(), doc, `id("x1")`)
		require.NoError(t, err)
		require.Equal(t, xpath1.NodeSetResult, r.Type)
		require.Len(t, r.NodeSet, 1)
		require.Equal(t, "item", r.NodeSet[0].Name())
	})

	t.Run("deduplicates", func(t *testing.T) {
		// Same ID repeated should not produce duplicate nodes
		doc := parseXML(t, `<root><a xml:id="foo">A</a></root>`)
		r, err := xpath1.Evaluate(t.Context(), doc, `id("foo foo")`)
		require.NoError(t, err)
		require.Equal(t, xpath1.NodeSetResult, r.Type)
		require.Len(t, r.NodeSet, 1)
	})
}

func TestEvalLang(t *testing.T) {
	t.Run("namespace aware", func(t *testing.T) {
		t.Run("xml:lang matches", func(t *testing.T) {
			doc := parseXML(t, `<root xml:lang="en"><child/></root>`)
			child := docElement(doc).(*helium.Element).FirstChild()
			r, err := xpath1.Evaluate(t.Context(), child, `lang("en")`)
			require.NoError(t, err)
			require.Equal(t, xpath1.BooleanResult, r.Type)
			require.True(t, r.Bool)
		})

		t.Run("non-xml namespace lang ignored", func(t *testing.T) {
			// A "lang" attribute in a non-XML namespace must NOT be treated as xml:lang
			doc := parseXML(t, `<root xmlns:x="urn:other" x:lang="en"><child/></root>`)
			child := docElement(doc).(*helium.Element).FirstChild()
			r, err := xpath1.Evaluate(t.Context(), child, `lang("en")`)
			require.NoError(t, err)
			require.Equal(t, xpath1.BooleanResult, r.Type)
			require.False(t, r.Bool)
		})

		t.Run("unprefixed lang ignored", func(t *testing.T) {
			// An unprefixed "lang" attribute has no namespace -- not xml:lang
			doc := parseXML(t, `<root lang="en"><child/></root>`)
			child := docElement(doc).(*helium.Element).FirstChild()
			r, err := xpath1.Evaluate(t.Context(), child, `lang("en")`)
			require.NoError(t, err)
			require.Equal(t, xpath1.BooleanResult, r.Type)
			require.False(t, r.Bool)
		})
	})
}

func TestCustomFunction(t *testing.T) {
	t.Run("unqualified", func(t *testing.T) {
		doc := parseXML(t, `<root><n>5</n></root>`)
		compiled, err := xpath1.Compile("double(number(/root/n))")
		require.NoError(t, err)

		ev := xpath1.NewEvaluator().Function("double", xpath1.FunctionFunc(func(_ context.Context, args []*xpath1.Result) (*xpath1.Result, error) {
			if len(args) != 1 {
				return nil, errDoubleOneArg
			}
			return &xpath1.Result{Type: xpath1.NumberResult, Number: args[0].Number * 2}, nil
		}))

		r, err := ev.Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		require.Equal(t, xpath1.NumberResult, r.Type)
		require.Equal(t, 10.0, r.Number)
	})

	t.Run("namespaced", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		compiled, err := xpath1.Compile("ext:hello('world')")
		require.NoError(t, err)

		ev := xpath1.NewEvaluator().
			Namespaces(map[string]string{
				nsPrefixExt: nsURIExt,
			}).
			FunctionNS(nsURIExt, "hello", xpath1.FunctionFunc(func(_ context.Context, args []*xpath1.Result) (*xpath1.Result, error) {
				if len(args) != 1 {
					return nil, errHelloOneArg
				}
				return &xpath1.Result{Type: xpath1.StringResult, String: "Hello, " + args[0].String + "!"}, nil
			}))

		r, err := ev.Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		require.Equal(t, xpath1.StringResult, r.Type)
		require.Equal(t, "Hello, world!", r.String)
	})

	t.Run("unknown", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		compiled, err := xpath1.Compile("myfunc()")
		require.NoError(t, err)

		_, err = xpath1.NewEvaluator().Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		require.True(t, errors.Is(err, xpath1.ErrUnknownFunction))
	})

	t.Run("namespaced unresolved prefix", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		compiled, err := xpath1.Compile("ext:foo()")
		require.NoError(t, err)

		// No namespace binding for "ext"
		_, err = xpath1.NewEvaluator().Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		require.True(t, errors.Is(err, xpath1.ErrUnknownFunctionNamespace))
	})

	t.Run("namespaced not found", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		compiled, err := xpath1.Compile("ext:missing()")
		require.NoError(t, err)

		// Namespace is bound but no function registered
		_, err = xpath1.NewEvaluator().Namespaces(map[string]string{
			"ext": nsURIExt,
		}).Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		require.True(t, errors.Is(err, xpath1.ErrUnknownFunction))
	})

	t.Run("builtin not overridden", func(t *testing.T) {
		doc := parseXML(t, `<root><a/><a/><a/></root>`)
		compiled, err := xpath1.Compile("count(/root/a)")
		require.NoError(t, err)

		// Register a custom "count" that returns 999 -- should not override built-in
		ev := xpath1.NewEvaluator().Function("count", xpath1.FunctionFunc(func(_ context.Context, _ []*xpath1.Result) (*xpath1.Result, error) {
			return &xpath1.Result{Type: xpath1.NumberResult, Number: 999}, nil
		}))

		r, err := ev.Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		require.Equal(t, 3.0, r.Number) // built-in wins
	})

	t.Run("context values", func(t *testing.T) {
		doc := parseXML(t, `<root><a/><b/><c/></root>`)
		compiled, err := xpath1.Compile("/root/*[mypos()]")
		require.NoError(t, err)

		ev := xpath1.NewEvaluator().Function("mypos", xpath1.FunctionFunc(func(ctx context.Context, _ []*xpath1.Result) (*xpath1.Result, error) {
			fctx := xpath1.GetFunctionContext(ctx)
			// Return true only for position 2
			return &xpath1.Result{
				Type: xpath1.BooleanResult,
				Bool: fctx.Position() == 2,
			}, nil
		}))

		r, err := ev.Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 1)
		require.Equal(t, "b", r.NodeSet[0].Name())
	})

	t.Run("with path expr", func(t *testing.T) {
		// Verify QName function calls work when followed by a path expression
		doc := parseXML(t, `<root><a><b>hello</b></a></root>`)
		compiled, err := xpath1.Compile("ext:identity(/root/a)/b")
		require.NoError(t, err)

		ev := xpath1.NewEvaluator().
			Namespaces(map[string]string{
				nsPrefixExt: nsURIExt,
			}).
			FunctionNS(nsURIExt, "identity", xpath1.FunctionFunc(func(_ context.Context, args []*xpath1.Result) (*xpath1.Result, error) {
				return args[0], nil
			}))

		r, err := ev.Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		require.Equal(t, xpath1.NodeSetResult, r.Type)
		require.Len(t, r.NodeSet, 1)
		require.Equal(t, "b", r.NodeSet[0].Name())
	})

	t.Run("Function helper", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		compiled, cErr := xpath1.Compile("myfunc()")
		require.NoError(t, cErr)
		ev := xpath1.NewEvaluator().Function("myfunc", xpath1.FunctionFunc(func(_ context.Context, _ []*xpath1.Result) (*xpath1.Result, error) {
			return &xpath1.Result{Type: xpath1.BooleanResult, Bool: true}, nil
		}))
		r, rErr := ev.Evaluate(t.Context(), compiled, doc)
		require.NoError(t, rErr)
		require.True(t, r.Bool)
	})

	t.Run("FunctionNS helper", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		compiled, cErr := xpath1.Compile("t:myfunc()")
		require.NoError(t, cErr)
		ev := xpath1.NewEvaluator().
			Namespaces(map[string]string{
				"t": "urn:test",
			}).
			FunctionNS("urn:test", "myfunc", xpath1.FunctionFunc(func(_ context.Context, _ []*xpath1.Result) (*xpath1.Result, error) {
				return &xpath1.Result{Type: xpath1.BooleanResult, Bool: true}, nil
			}))
		r, rErr := ev.Evaluate(t.Context(), compiled, doc)
		require.NoError(t, rErr)
		require.True(t, r.Bool)
	})
}

func TestLimits(t *testing.T) {
	t.Run("recursion", func(t *testing.T) {
		// Build a left-deep or-chain: "1 or 1 or 1 or ..."
		// The parser handles "or" iteratively (loop in parseOrExpr),
		// so parse depth stays at 1. But eval() recurses into the
		// left-deep BinaryExpr tree, reaching depth > 5000.
		var b strings.Builder
		terms := 5100
		b.WriteString("1")
		for i := 1; i < terms; i++ {
			b.WriteString(" or 1")
		}
		expr := b.String()

		doc := parseXML(t, `<root/>`)
		_, err := xpath1.Evaluate(t.Context(), doc, expr)
		require.Error(t, err)
		require.True(t, errors.Is(err, xpath1.ErrRecursionLimit))
	})

	t.Run("op", func(t *testing.T) {
		doc := parseXML(t, `<root><a/><b/><c/><d/><e/></root>`)
		compiled, err := xpath1.Compile("/root/*")
		require.NoError(t, err)

		// With a very small op limit, evaluation should fail
		_, err = xpath1.NewEvaluator().OpLimit(1).Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		require.True(t, errors.Is(err, xpath1.ErrOpLimit))

		// With a generous limit, it should succeed
		r, err := xpath1.NewEvaluator().OpLimit(10000).Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 5)

		// Without limit (zero), it should succeed
		r, err = xpath1.NewEvaluator().OpLimit(0).Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 5)
	})

	t.Run("op function calls", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		compiled, err := xpath1.Compile("concat('a', 'b', 'c')")
		require.NoError(t, err)

		// concat counts as 1 function-call op; limit of 0 means unlimited
		r, err := xpath1.NewEvaluator().OpLimit(0).Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err)
		require.Equal(t, "abc", r.String)

		// With limit too low for the function call
		_, err = xpath1.NewEvaluator().OpLimit(0).Evaluate(t.Context(), compiled, doc)
		require.NoError(t, err) // 0 = unlimited
	})

	t.Run("parse depth", func(t *testing.T) {
		// Build expression with 5100 nested parentheses: (((((...1...)))))
		var b strings.Builder
		depth := 5100
		for range depth {
			b.WriteString("(")
		}
		b.WriteString("1")
		for range depth {
			b.WriteString(")")
		}
		expr := b.String()

		_, err := xpath1.Compile(expr)
		require.Error(t, err)
		require.Contains(t, err.Error(), "nesting too deep")
	})

	t.Run("normal expressions unaffected", func(t *testing.T) {
		// Verify that normal expressions with moderate complexity still work
		doc := parseXML(t, `<bookstore>
			<book><title>A</title><price>30</price></book>
			<book><title>B</title><price>40</price></book>
		</bookstore>`)

		r, err := xpath1.Evaluate(t.Context(), doc, "/bookstore/book[price>35]/title")
		require.NoError(t, err)
		require.Len(t, r.NodeSet, 1)
		require.Equal(t, "B", string(r.NodeSet[0].Content()))
	})
}

func TestCompile(t *testing.T) {
	t.Run("MustCompile", func(t *testing.T) {
		expr := xpath1.MustCompile("/root")
		require.NotNil(t, expr)
	})

	t.Run("MustCompile panics", func(t *testing.T) {
		require.Panics(t, func() {
			xpath1.MustCompile("[invalid")
		})
	})
}

func TestFind(t *testing.T) {
	t.Run("node-set", func(t *testing.T) {
		doc := parseXML(t, `<root><a/><b/></root>`)
		nodes, err := xpath1.Find(t.Context(), doc, "/root/*")
		require.NoError(t, err)
		require.Len(t, nodes, 2)
	})

	t.Run("not node-set", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		_, err := xpath1.Find(t.Context(), doc, "1 + 2")
		require.Error(t, err)
	})
}

func TestBuiltinArgErrors(t *testing.T) {
	t.Run("arg count", func(t *testing.T) {
		doc := parseXML(t, `<root><a/></root>`)

		// Each expression compiles fine but must fail at evaluation time because
		// the function is called with the wrong number of arguments.
		exprs := []string{
			"last(1)",
			"position(1)",
			"count()",
			"count(1, 2)",
			"id()",
			"id(1, 2)",
			"string(1, 2)",
			"concat('a')",
			"starts-with('a')",
			"contains('a')",
			"substring-before('a')",
			"substring-after('a')",
			"substring('a')",
			"substring('a', 1, 2, 3)",
			"string-length('a', 'b')",
			"normalize-space('a', 'b')",
			"translate('a', 'b')",
			"boolean()",
			"not()",
			"true(1)",
			"false(1)",
			"lang()",
			"number(1, 2)",
			"sum()",
			"floor()",
			"ceiling()",
			"round()",
			"local-name(1, 2)",
			"name(1, 2)",
			"namespace-uri(1, 2)",
		}

		for _, expr := range exprs {
			t.Run(expr, func(t *testing.T) {
				compiled, err := xpath1.Compile(expr)
				require.NoError(t, err)
				_, err = compiled.Evaluate(t.Context(), doc)
				require.Error(t, err, "expected arg-count error for %q", expr)
			})
		}
	})

	t.Run("count non node-set", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		_, err := xpath1.Evaluate(t.Context(), doc, "count('not a node-set')")
		require.Error(t, err)
	})

	t.Run("sum non node-set", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		_, err := xpath1.Evaluate(t.Context(), doc, "sum(1)")
		require.Error(t, err)
	})

	t.Run("node arg not node-set", func(t *testing.T) {
		doc := parseXML(t, `<root/>`)
		for _, expr := range []string{
			"name('x')",
			"local-name(1)",
			"namespace-uri(true())",
		} {
			t.Run(expr, func(t *testing.T) {
				_, err := xpath1.Evaluate(t.Context(), doc, expr)
				require.Error(t, err)
			})
		}
	})
}

func TestEvalBookstoreExample(t *testing.T) {
	doc := parseXML(t, `<bookstore>
		<book><title>A</title><price>30</price></book>
		<book><title>B</title><price>40</price></book>
		<book><title>C</title><price>25</price></book>
	</bookstore>`)

	r, err := xpath1.Evaluate(t.Context(), doc, "/bookstore/book[price>35]/title")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, "title", r.NodeSet[0].Name())
	require.Equal(t, "B", string(r.NodeSet[0].Content()))
}
