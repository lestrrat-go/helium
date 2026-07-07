package xpath3_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// contentKindDecls is a minimal SchemaDeclarations that also implements the
// optional xpath3.ContentTypeKindProvider, mapping a fixed set of type
// annotations to content-type kinds. It isolates the fn:data element-only
// FOTY0012 plumbing from the xsd adapter.
type contentKindDecls struct {
	kinds map[string]xpath3.ContentTypeKind
}

func (contentKindDecls) LookupSchemaElement(string, string) (string, bool)   { return "", false }
func (contentKindDecls) LookupSchemaAttribute(string, string) (string, bool) { return "", false }
func (contentKindDecls) LookupSchemaType(string, string) (string, bool)      { return "", false }
func (contentKindDecls) IsSubtypeOf(string, string) bool                     { return false }
func (contentKindDecls) IsSubstitutionGroupMember(string, string, string, string) bool {
	return false
}
func (contentKindDecls) ValidateCast(context.Context, string, string) error { return nil }
func (contentKindDecls) ValidateCastWithNS(context.Context, string, string, map[string]string) error {
	return nil
}
func (contentKindDecls) ListItemType(string) (string, bool) { return "", false }
func (contentKindDecls) UnionMemberTypes(string) []string   { return nil }

func (d contentKindDecls) SchemaTypeContentKind(typeName string) (xpath3.ContentTypeKind, bool) {
	k, ok := d.kinds[typeName]
	return k, ok
}

// TestFnDataElementOnlyRaisesFOTY0012_Mock drives the optional-interface path
// with a hand-rolled provider: an element node annotated with an element-only
// complex type has no typed value, so fn:data must raise err:FOTY0012, while a
// mixed- or simple-content annotation still atomizes to the node's string value.
func TestFnDataElementOnlyRaisesFOTY0012_Mock(t *testing.T) {
	doc := mustParseXML(t, `<root><child>hi</child></root>`)
	root := doc.DocumentElement()
	child := root.FirstChild() // <child>

	decls := contentKindDecls{kinds: map[string]xpath3.ContentTypeKind{
		xpath3.QAnnotation("urn:t", "rootType"):  xpath3.ContentTypeElementOnly,
		xpath3.QAnnotation("urn:t", "mixedType"): xpath3.ContentTypeMixed,
	}}

	t.Run("element-only raises FOTY0012", func(t *testing.T) {
		eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			TypeAnnotations(map[helium.Node]string{
				root: xpath3.QAnnotation("urn:t", "rootType"),
			}).
			SchemaDeclarations(decls)

		compiled, err := xpath3.NewCompiler().Compile(`data(/*)`)
		require.NoError(t, err)
		_, err = eval.Evaluate(t.Context(), compiled, doc)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.True(t, errors.As(err, &xerr), "want *xpath3.XPathError, got %T: %v", err, err)
		require.Equal(t, "FOTY0012", xerr.Code)
	})

	t.Run("mixed content atomizes normally", func(t *testing.T) {
		eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			TypeAnnotations(map[helium.Node]string{
				root: xpath3.QAnnotation("urn:t", "mixedType"),
			}).
			SchemaDeclarations(decls)

		seq := evalExprWithEval(t, eval, doc, `data(/*)`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, xpath3.TypeUntypedAtomic, av.TypeName)
	})

	t.Run("unannotated child atomizes normally", func(t *testing.T) {
		eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			TypeAnnotations(map[helium.Node]string{
				child: xpath3.QAnnotation("urn:t", "unknownType"), // not element-only per provider
			}).
			SchemaDeclarations(decls)

		seq := evalExprWithEval(t, eval, doc, `data(/*/child)`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, "hi", av.StringVal())
	})
}

// TestFnDataElementOnlyRaisesFOTY0012_XSD exercises the real xsd
// SchemaDeclarations adapter (schemaDecls.SchemaTypeContentKind) end-to-end:
// a schema-validated element whose complex type has element-only content must
// make fn:data raise err:FOTY0012, while a simple-content child atomizes.
func TestFnDataElementOnlyRaisesFOTY0012_XSD(t *testing.T) {
	const schemaSrc = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
	    targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
	  <xs:element name="root" type="t:rootType"/>
	  <xs:complexType name="rootType">
	    <xs:sequence>
	      <xs:element name="child" type="xs:string"/>
	    </xs:sequence>
	  </xs:complexType>
	</xs:schema>`

	ctx := t.Context()
	schemaDoc := mustParseXML(t, schemaSrc)
	schema, err := xsd.NewCompiler().Compile(ctx, schemaDoc)
	require.NoError(t, err)

	doc := mustParseXML(t, `<root xmlns="urn:t"><child>hi</child></root>`)

	ann := make(xsd.TypeAnnotations)
	require.NoError(t, xsd.NewValidator(schema).Annotations(&ann).Validate(ctx, doc))

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		SchemaDeclarations(schema.Declarations()).
		TypeAnnotations(ann)

	t.Run("element-only root raises FOTY0012", func(t *testing.T) {
		compiled, err := xpath3.NewCompiler().Compile(`/*/data()`)
		require.NoError(t, err)
		_, err = eval.Evaluate(ctx, compiled, doc)
		require.Error(t, err)
		var xerr *xpath3.XPathError
		require.True(t, errors.As(err, &xerr), "want *xpath3.XPathError, got %T: %v", err, err)
		require.Equal(t, "FOTY0012", xerr.Code)
	})

	t.Run("simple-content child atomizes normally", func(t *testing.T) {
		seq := evalExprWithEval(t, eval, doc, `data(/*/*:child)`)
		require.Equal(t, 1, seq.Len())
		av, ok := seq.Get(0).(xpath3.AtomicValue)
		require.True(t, ok)
		require.Equal(t, "hi", av.StringVal())
	})
}
