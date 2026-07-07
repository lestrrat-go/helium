package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// idListUnionSchema mirrors the relevant part of the QT3 fn/id/id.xsd fixture:
// an element whose <id> child is typed as a LIST of xs:ID (is-id only when the
// value is a singleton) and one whose <id> child is a UNION of an xs:ID-derived
// type and xs:integer (is-id only when the ID member is selected).
const idListUnionSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:element name="IDS2">
    <xs:complexType>
      <xs:sequence>
        <xs:any processContents="strict" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="Element-with-ID-list-child">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="id" type="t:ID-List"/>
        <xs:element name="data" type="xs:anyType"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="Element-with-ID-union-child">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="id" type="t:ID-Union"/>
        <xs:element name="data" type="xs:anyType"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:simpleType name="ID-List"><xs:list itemType="xs:ID"/></xs:simpleType>
  <xs:simpleType name="ID-Union"><xs:union memberTypes="t:Restricted-ID xs:integer"/></xs:simpleType>
  <xs:simpleType name="Restricted-ID">
    <xs:restriction base="xs:ID"><xs:pattern value="[a-z]+"/></xs:restriction>
  </xs:simpleType>
</xs:schema>`

const idListUnionDoc = `<IDS2 xmlns="urn:t">
  <Element-with-ID-list-child><id>xi</id><data>a</data></Element-with-ID-list-child>
  <Element-with-ID-list-child><id>ping pong</id><data>b</data></Element-with-ID-list-child>
  <Element-with-ID-union-child><id>omicron</id><data>c</data></Element-with-ID-union-child>
  <Element-with-ID-union-child><id>853</id><data>d</data></Element-with-ID-union-child>
</IDS2>`

// TestFnIDPSVIListUnion_XSD drives the real xsd validator's is-id (IDNodes)
// collector end-to-end. It reproduces QT3 fn-element-with-id-4/5: a <id> typed
// as a singleton list of xs:ID and a union whose xs:ID-derived member is
// selected are is-id nodes (found by fn:id / fn:element-with-id), while a
// multi-item ID-list and a union that selects the xs:integer member are NOT.
func TestFnIDPSVIListUnion_XSD(t *testing.T) {
	ctx := t.Context()
	schemaDoc := mustParseXML(t, idListUnionSchema)
	schema, err := xsd.NewCompiler().Compile(ctx, schemaDoc)
	require.NoError(t, err)

	doc := mustParseXML(t, idListUnionDoc)

	ann := make(xsd.TypeAnnotations)
	ids := make(xsd.IDNodes)
	require.NoError(t, xsd.NewValidator(schema).Annotations(&ann).IDNodes(&ids).Validate(ctx, doc))
	// The singleton-list <id> and the ID-member union <id> must be flagged is-id;
	// the multi-item list and the integer union member must not be. Two of the
	// four <id> elements qualify.
	require.Len(t, ids, 2, "exactly the singleton-list and ID-union <id> nodes are is-id")

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		SchemaDeclarations(schema.Declarations()).
		TypeAnnotations(ann).
		IDNodes(ids)

	assertString := func(t *testing.T, expr, want string) {
		t.Helper()
		seq := evalExprWithEval(t, eval, doc, expr)
		require.Equal(t, 1, seq.Len(), "expr %q", expr)
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok, "expr %q", expr)
		require.Equal(t, want, av.StringVal(), "expr %q", expr)
	}
	assertCount := func(t *testing.T, expr string, want int64) {
		t.Helper()
		seq := evalExprWithEval(t, eval, doc, expr)
		require.Equal(t, 1, seq.Len(), "expr %q", expr)
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok, "expr %q", expr)
		require.Equal(t, want, av.IntegerVal(), "expr %q", expr)
	}

	// Singleton list of xs:ID (QT3 fn-element-with-id-4).
	t.Run("singleton-list id() returns the ID element", func(t *testing.T) {
		assertString(t, `id('xi')/local-name()`, "id")
	})
	t.Run("singleton-list element-with-id() returns the parent", func(t *testing.T) {
		assertString(t, `element-with-id('xi')/local-name()`, "Element-with-ID-list-child")
	})
	t.Run("multi-item list is not is-id", func(t *testing.T) {
		assertCount(t, `count(id('ping'))`, 0)
		assertCount(t, `count(element-with-id('ping'))`, 0)
		assertCount(t, `count(id('pong'))`, 0)
	})

	// Union of (xs:ID-derived, xs:integer) (QT3 fn-element-with-id-5).
	t.Run("union ID-member id() returns the ID element", func(t *testing.T) {
		assertString(t, `id('omicron')/local-name()`, "id")
	})
	t.Run("union ID-member element-with-id() returns the parent", func(t *testing.T) {
		assertString(t, `element-with-id('omicron')/local-name()`, "Element-with-ID-union-child")
	})
	t.Run("union integer-member is not is-id", func(t *testing.T) {
		assertCount(t, `count(id('853'))`, 0)
		assertCount(t, `count(element-with-id('853'))`, 0)
	})
}

// TestFnIDUsesIDNodesSet_Mock isolates the Evaluator.IDNodes plumbing from the
// xsd adapter: a node placed in the is-id set is found by fn:id /
// fn:element-with-id even with no ID-subtype type annotation, while a node
// absent from the set (and lacking an ID annotation) is not.
func TestFnIDUsesIDNodesSet_Mock(t *testing.T) {
	doc := mustParseXML(t, `<root><a>foo</a><b>bar</b></root>`)
	root := doc.DocumentElement()
	a := root.FirstChild() // <a>

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		IDNodes(map[helium.Node]struct{}{a: {}})

	t.Run("is-id element found by fn:id", func(t *testing.T) {
		seq := evalExprWithEval(t, eval, doc, `id('foo')/local-name()`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, "a", av.StringVal())
	})
	t.Run("is-id element parent found by fn:element-with-id", func(t *testing.T) {
		seq := evalExprWithEval(t, eval, doc, `element-with-id('foo')/local-name()`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, "root", av.StringVal())
	})
	t.Run("node absent from is-id set is not found", func(t *testing.T) {
		seq := evalExprWithEval(t, eval, doc, `count(id('bar'))`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, int64(0), av.IntegerVal())
	})
}
