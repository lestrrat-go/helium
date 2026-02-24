package xpath_test

import (
	"math"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath"
	"github.com/stretchr/testify/require"
)

func parseXML(t *testing.T, s string) *helium.Document {
	t.Helper()
	doc, err := helium.Parse([]byte(s))
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
	r, err := xpath.Evaluate(doc, "/")
	require.NoError(t, err)
	require.Equal(t, xpath.NodeSetResult, r.Type)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, helium.DocumentNode, r.NodeSet[0].Type())
}

func TestEvalAbsoluteChild(t *testing.T) {
	doc := parseXML(t, `<root><a/><b/></root>`)
	r, err := xpath.Evaluate(doc, "/root/a")
	require.NoError(t, err)
	require.Equal(t, xpath.NodeSetResult, r.Type)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, "a", r.NodeSet[0].Name())
}

func TestEvalRelativeChild(t *testing.T) {
	doc := parseXML(t, `<root><a><b/></a></root>`)
	root := docElement(doc)
	r, err := xpath.Evaluate(root, "a/b")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, "b", r.NodeSet[0].Name())
}

func TestEvalDoubleSlash(t *testing.T) {
	doc := parseXML(t, `<root><a><b/></a><b/></root>`)
	r, err := xpath.Evaluate(doc, "//b")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 2)
}

func TestEvalDot(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	root := docElement(doc)
	r, err := xpath.Evaluate(root, ".")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, root, r.NodeSet[0])
}

func TestEvalDotDot(t *testing.T) {
	doc := parseXML(t, `<root><a/></root>`)
	root := docElement(doc)
	a := root.FirstChild()
	r, err := xpath.Evaluate(a, "..")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, root, r.NodeSet[0])
}

func TestEvalWildcard(t *testing.T) {
	doc := parseXML(t, `<root><a/><b/><c/></root>`)
	root := docElement(doc)
	r, err := xpath.Evaluate(root, "*")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 3)
}

func TestEvalAttribute(t *testing.T) {
	doc := parseXML(t, `<root id="123"/>`)
	root := docElement(doc)
	r, err := xpath.Evaluate(root, "@id")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	attr, ok := r.NodeSet[0].(*helium.Attribute)
	require.True(t, ok)
	require.Equal(t, "123", attr.Value())
}

func TestEvalDescendant(t *testing.T) {
	doc := parseXML(t, `<root><a><b><c/></b></a></root>`)
	r, err := xpath.Evaluate(doc, "/root/descendant::c")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, "c", r.NodeSet[0].Name())
}

func TestEvalAncestor(t *testing.T) {
	doc := parseXML(t, `<root><a><b/></a></root>`)
	nodes, err := xpath.Find(doc, "//b")
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	b := nodes[0]
	r, err := xpath.Evaluate(b, "ancestor::root")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, "root", r.NodeSet[0].Name())
}

func TestEvalFollowingSibling(t *testing.T) {
	doc := parseXML(t, `<root><a/><b/><c/></root>`)
	nodes, err := xpath.Find(doc, "/root/a")
	require.NoError(t, err)
	r, err := xpath.Evaluate(nodes[0], "following-sibling::*")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 2)
	require.Equal(t, "b", r.NodeSet[0].Name())
	require.Equal(t, "c", r.NodeSet[1].Name())
}

func TestEvalPrecedingSibling(t *testing.T) {
	doc := parseXML(t, `<root><a/><b/><c/></root>`)
	nodes, err := xpath.Find(doc, "/root/c")
	require.NoError(t, err)
	r, err := xpath.Evaluate(nodes[0], "preceding-sibling::*")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 2)
}

// --- Predicates ---

func TestEvalPositionPredicate(t *testing.T) {
	doc := parseXML(t, `<root><a/><a/><a/></root>`)
	r, err := xpath.Evaluate(doc, "/root/a[2]")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
}

func TestEvalLastPredicate(t *testing.T) {
	doc := parseXML(t, `<root><a/><a/><a/></root>`)
	r, err := xpath.Evaluate(doc, "/root/a[last()]")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
}

func TestEvalBooleanPredicate(t *testing.T) {
	doc := parseXML(t, `<root><a x="1"/><a/><a x="2"/></root>`)
	r, err := xpath.Evaluate(doc, "/root/a[@x]")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 2)
}

// --- Comparisons ---

func TestEvalEquals(t *testing.T) {
	doc := parseXML(t, `<root><a>hello</a></root>`)
	r, err := xpath.Evaluate(doc, "/root/a = 'hello'")
	require.NoError(t, err)
	require.Equal(t, xpath.BooleanResult, r.Type)
	require.True(t, r.Boolean)
}

func TestEvalNotEquals(t *testing.T) {
	doc := parseXML(t, `<root><a>hello</a></root>`)
	r, err := xpath.Evaluate(doc, "/root/a != 'world'")
	require.NoError(t, err)
	require.True(t, r.Boolean)
}

func TestEvalNumericComparison(t *testing.T) {
	doc := parseXML(t, `<root><price>35</price></root>`)
	r, err := xpath.Evaluate(doc, "/root/price > 30")
	require.NoError(t, err)
	require.True(t, r.Boolean)
}

// --- Arithmetic ---

func TestEvalArithmetic(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "1 + 2")
	require.NoError(t, err)
	require.Equal(t, xpath.NumberResult, r.Type)
	require.Equal(t, 3.0, r.Number)
}

func TestEvalMultiplication(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "3 * 4")
	require.NoError(t, err)
	require.Equal(t, 12.0, r.Number)
}

func TestEvalDivision(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "10 div 3")
	require.NoError(t, err)
	require.InDelta(t, 3.333, r.Number, 0.01)
}

func TestEvalMod(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "10 mod 3")
	require.NoError(t, err)
	require.Equal(t, 1.0, r.Number)
}

func TestEvalNegation(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "-5")
	require.NoError(t, err)
	require.Equal(t, -5.0, r.Number)
}

// --- Boolean operators ---

func TestEvalOr(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "true() or false()")
	require.NoError(t, err)
	require.True(t, r.Boolean)
}

func TestEvalAnd(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "true() and false()")
	require.NoError(t, err)
	require.False(t, r.Boolean)
}

// --- Union ---

func TestEvalUnion(t *testing.T) {
	doc := parseXML(t, `<root><a/><b/></root>`)
	r, err := xpath.Evaluate(doc, "/root/a | /root/b")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 2)
}

// --- String functions ---

func TestEvalStringValue(t *testing.T) {
	doc := parseXML(t, `<root>hello</root>`)
	r, err := xpath.Evaluate(doc, "string(/root)")
	require.NoError(t, err)
	require.Equal(t, "hello", r.String)
}

func TestEvalConcat(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "concat('a', 'b', 'c')")
	require.NoError(t, err)
	require.Equal(t, "abc", r.String)
}

func TestEvalStartsWith(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "starts-with('hello', 'hel')")
	require.NoError(t, err)
	require.True(t, r.Boolean)
}

func TestEvalContains(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "contains('hello world', 'world')")
	require.NoError(t, err)
	require.True(t, r.Boolean)
}

func TestEvalSubstringBefore(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "substring-before('1999/04/01', '/')")
	require.NoError(t, err)
	require.Equal(t, "1999", r.String)
}

func TestEvalSubstringAfter(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "substring-after('1999/04/01', '/')")
	require.NoError(t, err)
	require.Equal(t, "04/01", r.String)
}

func TestEvalSubstring(t *testing.T) {
	doc := parseXML(t, `<root/>`)

	r, err := xpath.Evaluate(doc, "substring('12345', 2, 3)")
	require.NoError(t, err)
	require.Equal(t, "234", r.String)

	r, err = xpath.Evaluate(doc, "substring('12345', 2)")
	require.NoError(t, err)
	require.Equal(t, "2345", r.String)
}

func TestEvalStringLength(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "string-length('hello')")
	require.NoError(t, err)
	require.Equal(t, 5.0, r.Number)
}

func TestEvalNormalizeSpace(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "normalize-space('  hello   world  ')")
	require.NoError(t, err)
	require.Equal(t, "hello world", r.String)
}

func TestEvalTranslate(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "translate('bar', 'abc', 'ABC')")
	require.NoError(t, err)
	require.Equal(t, "BAr", r.String)
}

// --- Boolean functions ---

func TestEvalNot(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "not(false())")
	require.NoError(t, err)
	require.True(t, r.Boolean)
}

func TestEvalBoolean(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "boolean(1)")
	require.NoError(t, err)
	require.True(t, r.Boolean)

	r, err = xpath.Evaluate(doc, "boolean(0)")
	require.NoError(t, err)
	require.False(t, r.Boolean)
}

// --- Number functions ---

func TestEvalCount(t *testing.T) {
	doc := parseXML(t, `<root><a/><a/><a/></root>`)
	r, err := xpath.Evaluate(doc, "count(/root/a)")
	require.NoError(t, err)
	require.Equal(t, 3.0, r.Number)
}

func TestEvalSum(t *testing.T) {
	doc := parseXML(t, `<root><n>1</n><n>2</n><n>3</n></root>`)
	r, err := xpath.Evaluate(doc, "sum(/root/n)")
	require.NoError(t, err)
	require.Equal(t, 6.0, r.Number)
}

func TestEvalFloor(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "floor(2.7)")
	require.NoError(t, err)
	require.Equal(t, 2.0, r.Number)
}

func TestEvalCeiling(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "ceiling(2.3)")
	require.NoError(t, err)
	require.Equal(t, 3.0, r.Number)
}

func TestEvalRound(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "round(2.5)")
	require.NoError(t, err)
	require.Equal(t, 3.0, r.Number)

	r, err = xpath.Evaluate(doc, "round(2.4)")
	require.NoError(t, err)
	require.Equal(t, 2.0, r.Number)
}

func TestEvalNumber(t *testing.T) {
	doc := parseXML(t, `<root>42</root>`)
	r, err := xpath.Evaluate(doc, "number(/root)")
	require.NoError(t, err)
	require.Equal(t, 42.0, r.Number)
}

func TestEvalNumberNaN(t *testing.T) {
	doc := parseXML(t, `<root>abc</root>`)
	r, err := xpath.Evaluate(doc, "number(/root)")
	require.NoError(t, err)
	require.True(t, math.IsNaN(r.Number))
}

// --- Node name functions ---

func TestEvalLocalName(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "local-name(/root)")
	require.NoError(t, err)
	require.Equal(t, "root", r.String)
}

func TestEvalName(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "name(/root)")
	require.NoError(t, err)
	require.Equal(t, "root", r.String)
}

// --- Variables ---

func TestEvalVariable(t *testing.T) {
	doc := parseXML(t, `<root><a/><b/></root>`)
	expr, err := xpath.Compile("$x + 1")
	require.NoError(t, err)
	r, err := expr.EvaluateWithContext(doc, &xpath.Context{
		Variables: map[string]interface{}{
			"x": float64(41),
		},
	})
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

	r, err := xpath.Evaluate(doc, "/bookstore/book[price>35]/title")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, "title", r.NodeSet[0].Name())
	require.Equal(t, "B", string(r.NodeSet[0].Content()))
}

func TestEvalCountWithPredicate(t *testing.T) {
	doc := parseXML(t, `<root><a x="1"/><a/><a x="2"/></root>`)
	r, err := xpath.Evaluate(doc, "count(/root/a[@x])")
	require.NoError(t, err)
	require.Equal(t, 2.0, r.Number)
}

// --- MustCompile ---

func TestMustCompile(t *testing.T) {
	expr := xpath.MustCompile("/root")
	require.NotNil(t, expr)
}

func TestMustCompilePanics(t *testing.T) {
	require.Panics(t, func() {
		xpath.MustCompile("[invalid")
	})
}

// --- Find ---

func TestFind(t *testing.T) {
	doc := parseXML(t, `<root><a/><b/></root>`)
	nodes, err := xpath.Find(doc, "/root/*")
	require.NoError(t, err)
	require.Len(t, nodes, 2)
}

func TestFindNotNodeSet(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	_, err := xpath.Find(doc, "1 + 2")
	require.Error(t, err)
}

// --- NodeTest: text(), comment(), node() ---

func TestEvalTextNode(t *testing.T) {
	doc := parseXML(t, `<root>hello</root>`)
	r, err := xpath.Evaluate(doc, "/root/text()")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, helium.TextNode, r.NodeSet[0].Type())
	require.Equal(t, "hello", string(r.NodeSet[0].Content()))
}

func TestEvalCommentNode(t *testing.T) {
	doc := parseXML(t, `<root><!-- a comment --></root>`)
	r, err := xpath.Evaluate(doc, "/root/comment()")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, helium.CommentNode, r.NodeSet[0].Type())
}

func TestEvalNodeTest(t *testing.T) {
	doc := parseXML(t, `<root><a/>text<!-- c --></root>`)
	r, err := xpath.Evaluate(doc, "/root/node()")
	require.NoError(t, err)
	// Should include element, text, and comment
	require.GreaterOrEqual(t, len(r.NodeSet), 3)
}

// --- Self axis ---

func TestEvalSelfAxis(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	root := docElement(doc)
	r, err := xpath.Evaluate(root, "self::root")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 1)
	require.Equal(t, root, r.NodeSet[0])
}

func TestEvalSelfAxisNoMatch(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	root := docElement(doc)
	r, err := xpath.Evaluate(root, "self::other")
	require.NoError(t, err)
	require.Len(t, r.NodeSet, 0)
}

// --- Descendant-or-self ---

func TestEvalDescendantOrSelf(t *testing.T) {
	doc := parseXML(t, `<root><a><b/></a></root>`)
	r, err := xpath.Evaluate(doc, "/root/descendant-or-self::*")
	require.NoError(t, err)
	// root, a, b
	require.Len(t, r.NodeSet, 3)
}

// --- Ancestor-or-self ---

func TestEvalAncestorOrSelf(t *testing.T) {
	doc := parseXML(t, `<root><a><b/></a></root>`)
	nodes, err := xpath.Find(doc, "//b")
	require.NoError(t, err)
	r, err := xpath.Evaluate(nodes[0], "ancestor-or-self::*")
	require.NoError(t, err)
	// b, a, root
	require.Len(t, r.NodeSet, 3)
}

// --- String literal ---

func TestEvalStringLiteral(t *testing.T) {
	doc := parseXML(t, `<root/>`)
	r, err := xpath.Evaluate(doc, "'hello'")
	require.NoError(t, err)
	require.Equal(t, xpath.StringResult, r.Type)
	require.Equal(t, "hello", r.String)
}
