package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestStringValueElementOnlyStripsWhitespace_Mock drives the schema-aware
// dm:string-value path with a hand-rolled ContentTypeKindProvider: an element
// annotated with an element-only complex type has its INSIGNIFICANT inter-element
// whitespace stripped from its string value (XDM 3.1 PSVI construction), so
// fn:string / fn:string-length (no argument) see only the child element string
// values. A mixed-content element keeps the whitespace.
func TestStringValueElementOnlyStripsWhitespace_Mock(t *testing.T) {
	doc := mustParseXML(t, "<root>\n\t<a>x</a>\n\t<b>y</b>\n</root>")
	root := doc.DocumentElement()

	decls := contentKindDecls{kinds: map[string]xpath3.ContentTypeKind{
		xpath3.QAnnotation("urn:t", "rootType"):  xpath3.ContentTypeElementOnly,
		xpath3.QAnnotation("urn:t", "mixedType"): xpath3.ContentTypeMixed,
	}}

	t.Run("element-only string() strips insignificant whitespace", func(t *testing.T) {
		eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			TypeAnnotations(map[helium.Node]string{
				root: xpath3.QAnnotation("urn:t", "rootType"),
			}).
			SchemaDeclarations(decls)

		seq := evalExprWithEval(t, eval, doc, `/*/string()`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, "xy", av.StringVal())
	})

	t.Run("element-only string-length() counts stripped value", func(t *testing.T) {
		eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			TypeAnnotations(map[helium.Node]string{
				root: xpath3.QAnnotation("urn:t", "rootType"),
			}).
			SchemaDeclarations(decls)

		seq := evalExprWithEval(t, eval, doc, `/*/string-length()`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, "xs:integer", av.TypeName)
		require.Equal(t, int64(2), av.IntegerVal())
	})

	t.Run("mixed content keeps whitespace", func(t *testing.T) {
		eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			TypeAnnotations(map[helium.Node]string{
				root: xpath3.QAnnotation("urn:t", "mixedType"),
			}).
			SchemaDeclarations(decls)

		seq := evalExprWithEval(t, eval, doc, `/*/string()`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, "\n\tx\n\ty\n", av.StringVal())
	})
}

// TestStringValueElementOnlyStripsWhitespace_XSD exercises the real xsd adapter
// end-to-end: a schema-validated element with element-only complex content has
// its inter-element whitespace stripped from the string value (fn:string /
// fn:string-length / fn:normalize-space with no argument, which use the string
// value).
func TestStringValueElementOnlyStripsWhitespace_XSD(t *testing.T) {
	const schemaSrc = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
	    targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
	  <xs:element name="root" type="t:rootType"/>
	  <xs:complexType name="rootType">
	    <xs:sequence>
	      <xs:element name="a" type="xs:string"/>
	      <xs:element name="b" type="xs:string"/>
	    </xs:sequence>
	  </xs:complexType>
	</xs:schema>`

	ctx := t.Context()
	schemaDoc := mustParseXML(t, schemaSrc)
	schema, err := xsd.NewCompiler().Compile(ctx, schemaDoc)
	require.NoError(t, err)

	doc := mustParseXML(t, "<root xmlns=\"urn:t\">\n\t<a>x</a>\n\t<b>y</b>\n</root>")

	ann := make(xsd.TypeAnnotations)
	require.NoError(t, xsd.NewValidator(schema).Annotations(&ann).Validate(ctx, doc))

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		SchemaDeclarations(schema.Declarations()).
		TypeAnnotations(ann)

	t.Run("string() strips insignificant whitespace", func(t *testing.T) {
		seq := evalExprWithEval(t, eval, doc, `/*/string()`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, "xy", av.StringVal())
	})

	t.Run("string-length() counts stripped value", func(t *testing.T) {
		seq := evalExprWithEval(t, eval, doc, `/*/string-length()`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, int64(2), av.IntegerVal())
	})

	t.Run("normalize-space() strips insignificant whitespace", func(t *testing.T) {
		seq := evalExprWithEval(t, eval, doc, `/*/normalize-space()`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, "xy", av.StringVal())
	})
}
