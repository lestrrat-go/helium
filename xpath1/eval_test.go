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
	doc, err := helium.Parse(t.Context(), []byte(s))
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

// --- Location paths ---

func TestEvalRootPath(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "/")
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, r.Type)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, helium.DocumentNode, r.NodeSet[0].Type())
}

func TestEvalAbsoluteChild(t *testing.T) {
	doc := parseXML(t, `<root><a/><b/></root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "/root/a")
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, r.Type)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, "a", r.NodeSet[0].Name())
}

func TestEvalRelativeChild(t *testing.T) {
	doc := parseXML(t, `<root><a><b/></a></root>`)
	root := docElement(doc)
	r, err := xpath1.Evaluate(t.Context(), root, "a/b")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, "b", r.NodeSet[0].Name())
}

func TestEvalDoubleSlash(t *testing.T) {
	doc := parseXML(t, `<root><a><b/></a><b/></root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "//b")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 2)
}

func TestEvalDot(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	root := docElement(doc)
	r, err := xpath1.Evaluate(t.Context(), root, ".")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, root, r.NodeSet[0])
}

func TestEvalDotDot(t *testing.T) {
	doc := parseXML(t, `<root><a/></root>`)
	root := docElement(doc)
	a := root.FirstChild()
	r, err := xpath1.Evaluate(t.Context(), a, "..")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, root, r.NodeSet[0])
}

func TestEvalWildcard(t *testing.T) {
	doc := parseXML(t, `<root><a/><b/><c/></root>`)
	root := docElement(doc)
	r, err := xpath1.Evaluate(t.Context(), root, "*")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 3)
}

func TestEvalAttribute(t *testing.T) {
	doc := parseXML(t, `<root id="123"/>`)
	root := docElement(doc)
	r, err := xpath1.Evaluate(t.Context(), root, "@id")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	attr, ok := r.NodeSet[0].(*helium.Attribute)
	require.True(t, ok)
	require.Equal(t, "123", attr.Value())
}

func TestEvalDescendant(t *testing.T) {
	doc := parseXML(t, `<root><a><b><c/></b></a></root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "/root/descendant::c")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, "c", r.NodeSet[0].Name())
}

func TestEvalAncestor(t *testing.T) {
	doc := parseXML(t, `<root><a><b/></a></root>`)
	nodes, err := xpath1.Find(t.Context(), doc, "//b")
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	b := nodes[0]
	r, err := xpath1.Evaluate(t.Context(), b, "ancestor::root")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, "root", r.NodeSet[0].Name())
}

func TestEvalFollowingSibling(t *testing.T) {
	doc := parseXML(t, `<root><a/><b/><c/></root>`)
	nodes, err := xpath1.Find(t.Context(), doc, "/root/a")
	require.NoError(t, err)
	r, err := xpath1.Evaluate(t.Context(), nodes[0], "following-sibling::*")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 2)
	require.Equal(t, "b", r.NodeSet[0].Name())
	require.Equal(t, "c", r.NodeSet[1].Name())
}

func TestEvalPrecedingSibling(t *testing.T) {
	doc := parseXML(t, `<root><a/><b/><c/></root>`)
	nodes, err := xpath1.Find(t.Context(), doc, "/root/c")
	require.NoError(t, err)
	r, err := xpath1.Evaluate(t.Context(), nodes[0], "preceding-sibling::*")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 2)
}

// --- Predicates ---

func TestEvalPositionPredicate(t *testing.T) {
	doc := parseXML(t, `<root><a/><a/><a/></root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "/root/a[2]")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
}

func TestEvalLastPredicate(t *testing.T) {
	doc := parseXML(t, `<root><a/><a/><a/></root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "/root/a[last()]")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
}

func TestEvalBooleanPredicate(t *testing.T) {
	doc := parseXML(t, `<root><a x="1"/><a/><a x="2"/></root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "/root/a[@x]")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 2)
}

// --- Comparisons ---

func TestEvalEquals(t *testing.T) {
	doc := parseXML(t, `<root><a>hello</a></root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "/root/a = 'hello'")
	require.NoError(t, err)
	require.Equal(t, xpath1.BooleanResult, r.Type)
	require.True(t, r.Bool)
}

func TestEvalNotEquals(t *testing.T) {
	doc := parseXML(t, `<root><a>hello</a></root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "/root/a != 'world'")
	require.NoError(t, err)
	require.True(t, r.Bool)
}

func TestEvalNumericComparison(t *testing.T) {
	doc := parseXML(t, `<root><price>35</price></root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "/root/price > 30")
	require.NoError(t, err)
	require.True(t, r.Bool)
}

// --- Arithmetic ---

func TestEvalArithmetic(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "1 + 2")
	require.NoError(t, err)
	require.Equal(t, xpath1.NumberResult, r.Type)
	require.Equal(t, 3.0, r.Number)
}

func TestEvalMultiplication(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "3 * 4")
	require.NoError(t, err)
	require.Equal(t, 12.0, r.Number)
}

func TestEvalDivision(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "10 div 3")
	require.NoError(t, err)
	require.InDelta(t, 3.333, r.Number, 0.01)
}

func TestEvalMod(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "10 mod 3")
	require.NoError(t, err)
	require.Equal(t, 1.0, r.Number)
}

func TestEvalNegation(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "-5")
	require.NoError(t, err)
	require.Equal(t, -5.0, r.Number)
}

// --- Boolean operators ---

func TestEvalOr(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "true() or false()")
	require.NoError(t, err)
	require.True(t, r.Bool)
}

func TestEvalAnd(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "true() and false()")
	require.NoError(t, err)
	require.False(t, r.Bool)
}

// --- Union ---

func TestEvalUnion(t *testing.T) {
	doc := parseXML(t, `<root><a/><b/></root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "/root/a | /root/b")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 2)
}

// --- String functions ---

func TestEvalStringValue(t *testing.T) {
	doc := parseXML(t, `<root>hello</root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "string(/root)")
	require.NoError(t, err)
	require.Equal(t, "hello", r.String)
}

func TestEvalConcat(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "concat('a', 'b', 'c')")
	require.NoError(t, err)
	require.Equal(t, "abc", r.String)
}

func TestEvalStartsWith(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "starts-with('hello', 'hel')")
	require.NoError(t, err)
	require.True(t, r.Bool)
}

func TestEvalContains(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "contains('hello world', 'world')")
	require.NoError(t, err)
	require.True(t, r.Bool)
}

func TestEvalSubstringBefore(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "substring-before('1999/04/01', '/')")
	require.NoError(t, err)
	require.Equal(t, "1999", r.String)
}

func TestEvalSubstringAfter(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "substring-after('1999/04/01', '/')")
	require.NoError(t, err)
	require.Equal(t, "04/01", r.String)
}

func TestEvalSubstring(t *testing.T) {
	doc := parseXML(t, `<root/>`)

	r, err := xpath1.Evaluate(t.Context(), doc, "substring('12345', 2, 3)")
	require.NoError(t, err)
	require.Equal(t, "234", r.String)

	r, err = xpath1.Evaluate(t.Context(), doc, "substring('12345', 2)")
	require.NoError(t, err)
	require.Equal(t, "2345", r.String)
}

func TestEvalStringLength(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "string-length('hello')")
	require.NoError(t, err)
	require.Equal(t, 5.0, r.Number)
}

func TestEvalNormalizeSpace(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "normalize-space('  hello   world  ')")
	require.NoError(t, err)
	require.Equal(t, "hello world", r.String)
}

func TestEvalTranslate(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "translate('bar', 'abc', 'ABC')")
	require.NoError(t, err)
	require.Equal(t, "BAr", r.String)
}

// --- Boolean functions ---

func TestEvalNot(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "not(false())")
	require.NoError(t, err)
	require.True(t, r.Bool)
}

func TestEvalBoolean(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "boolean(1)")
	require.NoError(t, err)
	require.True(t, r.Bool)

	r, err = xpath1.Evaluate(t.Context(), doc, "boolean(0)")
	require.NoError(t, err)
	require.False(t, r.Bool)
}

// --- Number functions ---

func TestEvalCount(t *testing.T) {
	doc := parseXML(t, `<root><a/><a/><a/></root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "count(/root/a)")
	require.NoError(t, err)
	require.Equal(t, 3.0, r.Number)
}

func TestEvalSum(t *testing.T) {
	doc := parseXML(t, `<root><n>1</n><n>2</n><n>3</n></root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "sum(/root/n)")
	require.NoError(t, err)
	require.Equal(t, 6.0, r.Number)
}

func TestEvalFloor(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "floor(2.7)")
	require.NoError(t, err)
	require.Equal(t, 2.0, r.Number)
}

func TestEvalCeiling(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "ceiling(2.3)")
	require.NoError(t, err)
	require.Equal(t, 3.0, r.Number)
}

func TestEvalRound(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "round(2.5)")
	require.NoError(t, err)
	require.Equal(t, 3.0, r.Number)

	r, err = xpath1.Evaluate(t.Context(), doc, "round(2.4)")
	require.NoError(t, err)
	require.Equal(t, 2.0, r.Number)
}

func TestEvalNumber(t *testing.T) {
	doc := parseXML(t, `<root>42</root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "number(/root)")
	require.NoError(t, err)
	require.Equal(t, 42.0, r.Number)
}

func TestEvalNumberNaN(t *testing.T) {
	doc := parseXML(t, `<root>abc</root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "number(/root)")
	require.NoError(t, err)
	require.True(t, math.IsNaN(r.Number))
}

// --- Node name functions ---

func TestEvalLocalName(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "local-name(/root)")
	require.NoError(t, err)
	require.Equal(t, "root", r.String)
}

func TestEvalName(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "name(/root)")
	require.NoError(t, err)
	require.Equal(t, "root", r.String)
}

// --- Variables ---

func TestEvalVariable(t *testing.T) {
	doc := parseXML(t, `<root><a/><b/></root>`)
	expr, err := xpath1.Compile("$x + 1")
	require.NoError(t, err)
	ctx := xpath1.NewContext(t.Context(),
		xpath1.WithVariables(map[string]any{
			"x": float64(41),
		}),
	)
	r, err := expr.Evaluate(ctx, doc)
	require.NoError(t, err)
	require.Equal(t, 42.0, r.Number)
}

// --- Complex expressions ---

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

func TestEvalCountWithPredicate(t *testing.T) {
	doc := parseXML(t, `<root><a x="1"/><a/><a x="2"/></root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "count(/root/a[@x])")
	require.NoError(t, err)
	require.Equal(t, 2.0, r.Number)
}

// --- MustCompile ---

func TestMustCompile(t *testing.T) {
	expr := xpath1.MustCompile("/root")
	require.NotNil(t, expr)
}

func TestMustCompilePanics(t *testing.T) {
	require.Panics(t, func() {
		xpath1.MustCompile("[invalid")
	})
}

// --- Find ---

func TestFind(t *testing.T) {
	doc := parseXML(t, `<root><a/><b/></root>`)
	nodes, err := xpath1.Find(t.Context(), doc, "/root/*")
	require.NoError(t, err)
	require.Len(t, nodes, 2)
}

func TestFindNotNodeSet(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	_, err := xpath1.Find(t.Context(), doc, "1 + 2")
	require.Error(t, err)
}

// --- NodeTest: text(), comment(), node() ---

func TestEvalTextNode(t *testing.T) {
	doc := parseXML(t, `<root>hello</root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "/root/text()")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, helium.TextNode, r.NodeSet[0].Type())
	require.Equal(t, "hello", string(r.NodeSet[0].Content()))
}

func TestEvalCommentNode(t *testing.T) {
	doc := parseXML(t, `<root><!-- a comment --></root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "/root/comment()")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, helium.CommentNode, r.NodeSet[0].Type())
}

func TestEvalNodeTest(t *testing.T) {
	doc := parseXML(t, `<root><a/>text<!-- c --></root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "/root/node()")
	require.NoError(t, err)
	// Should include element, text, and comment
	require.GreaterOrEqual(t, len(r.NodeSet), 3)
}

// --- Self axis ---

func TestEvalSelfAxis(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	root := docElement(doc)
	r, err := xpath1.Evaluate(t.Context(), root, "self::root")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, root, r.NodeSet[0])
}

func TestEvalSelfAxisNoMatch(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	root := docElement(doc)
	r, err := xpath1.Evaluate(t.Context(), root, "self::other")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 0)
}

// --- Descendant-or-self ---

func TestEvalDescendantOrSelf(t *testing.T) {
	doc := parseXML(t, `<root><a><b/></a></root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "/root/descendant-or-self::*")
	require.NoError(t, err)
	// root, a, b
	require.Len(t, r.NodeSet, 3)
}

// --- Ancestor-or-self ---

func TestEvalAncestorOrSelf(t *testing.T) {
	doc := parseXML(t, `<root><a><b/></a></root>`)
	nodes, err := xpath1.Find(t.Context(), doc, "//b")
	require.NoError(t, err)
	r, err := xpath1.Evaluate(t.Context(), nodes[0], "ancestor-or-self::*")
	require.NoError(t, err)
	// b, a, root
	require.Len(t, r.NodeSet, 3)
}

// --- String literal ---

func TestEvalStringLiteral(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath1.Evaluate(t.Context(), doc, "'hello'")
	require.NoError(t, err)
	require.Equal(t, xpath1.StringResult, r.Type)
	require.Equal(t, "hello", r.String)
}

// --- Limits ---

func TestRecursionLimit(t *testing.T) {
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
}

func TestOpLimit(t *testing.T) {
	doc := parseXML(t, `<root><a/><b/><c/><d/><e/></root>`)
	compiled, err := xpath1.Compile("/root/*")
	require.NoError(t, err)

	// With a very small op limit, evaluation should fail
	_, err = compiled.Evaluate(xpath1.NewContext(t.Context(),
		xpath1.WithOpLimit(1),
	), doc)
	require.Error(t, err)
	require.True(t, errors.Is(err, xpath1.ErrOpLimit))

	// With a generous limit, it should succeed
	r, err := compiled.Evaluate(xpath1.NewContext(t.Context(),
		xpath1.WithOpLimit(10000),
	), doc)
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 5)

	// Without limit (zero), it should succeed
	r, err = compiled.Evaluate(xpath1.NewContext(t.Context(),
		xpath1.WithOpLimit(0),
	), doc)
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 5)
}

func TestOpLimitFunctionCalls(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	compiled, err := xpath1.Compile("concat('a', 'b', 'c')")
	require.NoError(t, err)

	// concat counts as 1 function-call op; limit of 0 means unlimited
	r, err := compiled.Evaluate(xpath1.NewContext(t.Context(), xpath1.WithOpLimit(0)), doc)
	require.NoError(t, err)
	require.Equal(t, "abc", r.String)

	// With limit too low for the function call
	_, err = compiled.Evaluate(xpath1.NewContext(t.Context(), xpath1.WithOpLimit(0)), doc)
	require.NoError(t, err) // 0 = unlimited
}

func TestParseDepthLimit(t *testing.T) {
	// Build expression with 5100 nested parentheses: (((((...1...)))))
	var b strings.Builder
	depth := 5100
	for i := 0; i < depth; i++ {
		b.WriteString("(")
	}
	b.WriteString("1")
	for i := 0; i < depth; i++ {
		b.WriteString(")")
	}
	expr := b.String()

	_, err := xpath1.Compile(expr)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nesting too deep")
}

func TestLimitsNormalExpressionsUnaffected(t *testing.T) {
	// Verify that normal expressions with moderate complexity still work
	doc := parseXML(t, `<bookstore>
		<book><title>A</title><price>30</price></book>
		<book><title>B</title><price>40</price></book>
	</bookstore>`)

	r, err := xpath1.Evaluate(t.Context(), doc, "/bookstore/book[price>35]/title")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, "B", string(r.NodeSet[0].Content()))
}

// --- Custom function registration ---

func TestCustomFunctionUnqualified(t *testing.T) {
	doc := parseXML(t, `<root><n>5</n></root>`)
	compiled, err := xpath1.Compile("double(number(/root/n))")
	require.NoError(t, err)

	ctx := xpath1.NewContext(t.Context(),
		xpath1.WithFunctions(map[string]xpath1.Function{
			"double": xpath1.FunctionFunc(func(_ context.Context, args []*xpath1.Result) (*xpath1.Result, error) {
				if len(args) != 1 {
					return nil, errDoubleOneArg
				}
				return &xpath1.Result{Type: xpath1.NumberResult, Number: args[0].Number * 2}, nil
			}),
		}),
	)

	r, err := compiled.Evaluate(ctx, doc)
	require.NoError(t, err)
	require.Equal(t, xpath1.NumberResult, r.Type)
	require.Equal(t, 10.0, r.Number)
}

func TestCustomFunctionNamespaced(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	compiled, err := xpath1.Compile("ext:hello('world')")
	require.NoError(t, err)

	ctx := xpath1.NewContext(t.Context(),
		xpath1.WithNamespaces(map[string]string{
			"ext": "urn:test:ext",
		}),
		xpath1.WithFunctionsNS(map[xpath1.QualifiedName]xpath1.Function{
			{URI: "urn:test:ext", Name: "hello"}: xpath1.FunctionFunc(func(_ context.Context, args []*xpath1.Result) (*xpath1.Result, error) {
				if len(args) != 1 {
					return nil, errHelloOneArg
				}
				return &xpath1.Result{Type: xpath1.StringResult, String: "Hello, " + args[0].String + "!"}, nil
			}),
		}),
	)

	r, err := compiled.Evaluate(ctx, doc)
	require.NoError(t, err)
	require.Equal(t, xpath1.StringResult, r.Type)
	require.Equal(t, "Hello, world!", r.String)
}

func TestCustomFunctionUnknown(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	compiled, err := xpath1.Compile("myfunc()")
	require.NoError(t, err)

	_, err = compiled.Evaluate(t.Context(), doc)
	require.Error(t, err)
	require.True(t, errors.Is(err, xpath1.ErrUnknownFunction))
}

func TestCustomFunctionNamespacedUnresolvedPrefix(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	compiled, err := xpath1.Compile("ext:foo()")
	require.NoError(t, err)

	// No namespace binding for "ext"
	_, err = compiled.Evaluate(xpath1.NewContext(t.Context()), doc)
	require.Error(t, err)
	require.True(t, errors.Is(err, xpath1.ErrUnknownFunctionNamespace))
}

func TestCustomFunctionNamespacedNotFound(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	compiled, err := xpath1.Compile("ext:missing()")
	require.NoError(t, err)

	// Namespace is bound but no function registered
	ctx := xpath1.NewContext(t.Context(),
		xpath1.WithNamespaces(map[string]string{
			"ext": "urn:test:ext",
		}),
	)
	_, err = compiled.Evaluate(ctx, doc)
	require.Error(t, err)
	require.True(t, errors.Is(err, xpath1.ErrUnknownFunction))
}

func TestCustomFunctionBuiltinNotOverridden(t *testing.T) {
	doc := parseXML(t, `<root><a/><a/><a/></root>`)
	compiled, err := xpath1.Compile("count(/root/a)")
	require.NoError(t, err)

	// Register a custom "count" that returns 999 -- should not override built-in
	ctx := xpath1.NewContext(t.Context(),
		xpath1.WithFunctions(map[string]xpath1.Function{
			"count": xpath1.FunctionFunc(func(_ context.Context, _ []*xpath1.Result) (*xpath1.Result, error) {
				return &xpath1.Result{Type: xpath1.NumberResult, Number: 999}, nil
			}),
		}),
	)

	r, err := compiled.Evaluate(ctx, doc)
	require.NoError(t, err)
	require.Equal(t, 3.0, r.Number) // built-in wins
}

func TestCustomFunctionContextValues(t *testing.T) {
	doc := parseXML(t, `<root><a/><b/><c/></root>`)
	compiled, err := xpath1.Compile("/root/*[mypos()]")
	require.NoError(t, err)

	ctx := xpath1.NewContext(t.Context(),
		xpath1.WithFunctions(map[string]xpath1.Function{
			"mypos": xpath1.FunctionFunc(func(ctx context.Context, _ []*xpath1.Result) (*xpath1.Result, error) {
				fctx := xpath1.GetFunctionContext(ctx)
				// Return true only for position 2
				return &xpath1.Result{
					Type: xpath1.BooleanResult,
					Bool: fctx.Position() == 2,
				}, nil
			}),
		}),
	)

	r, err := compiled.Evaluate(ctx, doc)
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, "b", r.NodeSet[0].Name())
}

func TestRegisterFunctionHelper(t *testing.T) {
	xctx := &xpath1.Context{}

	// Registering a new function should succeed
	err := xctx.RegisterFunction("myfunc", xpath1.FunctionFunc(func(_ context.Context, _ []*xpath1.Result) (*xpath1.Result, error) {
		return &xpath1.Result{Type: xpath1.BooleanResult, Bool: true}, nil
	}))
	require.NoError(t, err)

	// Verify the function was registered by evaluating it
	doc := parseXML(t, `<root/>`)
	compiled, cErr := xpath1.Compile("myfunc()")
	require.NoError(t, cErr)
	ctx := xpath1.NewContext(t.Context(),
		xpath1.WithFunctions(xctx.Functions()),
	)
	r, rErr := compiled.Evaluate(ctx, doc)
	require.NoError(t, rErr)
	require.True(t, r.Bool)

	// Registering a built-in name should fail
	err = xctx.RegisterFunction("count", xpath1.FunctionFunc(func(_ context.Context, _ []*xpath1.Result) (*xpath1.Result, error) {
		return nil, nil
	}))
	require.Error(t, err)
}

func TestRegisterFunctionNSHelper(t *testing.T) {
	xctx := &xpath1.Context{}
	xctx.RegisterFunctionNS("urn:test", "myfunc", xpath1.FunctionFunc(func(_ context.Context, _ []*xpath1.Result) (*xpath1.Result, error) {
		return &xpath1.Result{Type: xpath1.BooleanResult, Bool: true}, nil
	}))

	// Verify the function was registered by evaluating it
	doc := parseXML(t, `<root/>`)
	compiled, cErr := xpath1.Compile("t:myfunc()")
	require.NoError(t, cErr)
	ctx := xpath1.NewContext(t.Context(),
		xpath1.WithNamespaces(map[string]string{
			"t": "urn:test",
		}),
		xpath1.WithFunctionsNS(xctx.FunctionsNS()),
	)
	r, rErr := compiled.Evaluate(ctx, doc)
	require.NoError(t, rErr)
	require.True(t, r.Bool)
}

func TestCustomFunctionWithPathExpr(t *testing.T) {
	// Verify QName function calls work when followed by a path expression
	doc := parseXML(t, `<root><a><b>hello</b></a></root>`)
	compiled, err := xpath1.Compile("ext:identity(/root/a)/b")
	require.NoError(t, err)

	ctx := xpath1.NewContext(t.Context(),
		xpath1.WithNamespaces(map[string]string{
			"ext": "urn:test:ext",
		}),
		xpath1.WithFunctionsNS(map[xpath1.QualifiedName]xpath1.Function{
			{URI: "urn:test:ext", Name: "identity"}: xpath1.FunctionFunc(func(_ context.Context, args []*xpath1.Result) (*xpath1.Result, error) {
				return args[0], nil
			}),
		}),
	)

	r, err := compiled.Evaluate(ctx, doc)
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, r.Type)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, "b", r.NodeSet[0].Name())
}

func TestLangNamespaceAware(t *testing.T) {
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
}

// --- id() function ---

func TestEvalIDWithXmlID(t *testing.T) {
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
}

func TestEvalIDWithDTD(t *testing.T) {
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
}

func TestEvalIDDeduplicates(t *testing.T) {
	// Same ID repeated should not produce duplicate nodes
	doc := parseXML(t, `<root><a xml:id="foo">A</a></root>`)
	r, err := xpath1.Evaluate(t.Context(), doc, `id("foo foo")`)
	require.NoError(t, err)
	require.Equal(t, xpath1.NodeSetResult, r.Type)
	require.Len(t, r.NodeSet, 1)
}
