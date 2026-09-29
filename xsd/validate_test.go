package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestValidateDocumentLevelElements pins how Validate treats a document that
// carries more than one document-level element, which only the DOM API can
// build. Every element child of the document node is validated as a root
// against the global element declarations, in document order, and the
// diagnostics come out in that same order.
func TestValidateDocumentLevelElements(t *testing.T) {
	t.Parallel()

	const schemaSrc = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a" type="xs:int"/>
  <xs:element name="b">
    <xs:complexType>
      <xs:attribute name="n" type="xs:int" use="required"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaSrc))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Compile(t.Context(), schemaDoc)
	require.NoError(t, err)

	doc := helium.NewDefaultDocument()
	a, err := doc.CreateElement("a")
	require.NoError(t, err)
	require.NoError(t, a.AddChild(doc.CreateText([]byte("not-an-int"))))
	require.NoError(t, doc.AddChild(a))

	require.NoError(t, doc.AddChild(doc.CreateComment([]byte("between roots"))))

	b, err := doc.CreateElement("b")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(b))

	c, err := doc.CreateElement("c")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(c))

	var errs string
	err = validateWithOutput(t, xsd.NewValidator(schema).Label("multi.xml"), doc, &errs)
	require.ErrorIs(t, err, xsd.ErrValidationFailed)
	require.Equal(t, expectedDocumentLevelErrors, errs)
}

const expectedDocumentLevelErrors = `multi.xml:0: Schemas validity error : Element 'a': 'not-an-int' is not a valid value of the atomic type 'xs:int'.
multi.xml:0: Schemas validity error : Element 'b': The attribute 'n' is required but missing.
multi.xml:0: Schemas validity error : Element 'c': No matching global declaration available for the validation root.
`
