package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestElementOnlyNBSPRejected covers element-only content type with a non-breaking
// space (U+00A0) preceding a child element. NBSP is Unicode whitespace but NOT one
// of the four XSD/XML whitespace characters, so it is significant character content
// that must be rejected in element-only content. A naive strings.TrimSpace would
// strip the NBSP and wrongly accept the document.
func TestElementOnlyNBSPRejected(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="child"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	instance := `<root>` + nbsp + `<child/></root>`

	v := compileValidator(t, schema)
	doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
	require.NoError(t, err)

	var errs string
	err = validateWithOutput(t, v, doc, &errs)
	require.Error(t, err, "NBSP character content in element-only type must be rejected")
	require.True(t, strings.Contains(errs, "element-only"),
		"error must report element-only content violation")
}

// TestElementOnlyXMLWhitespaceAllowed is the positive counterpart: ordinary XSD
// whitespace (space) between elements in element-only content is ignorable and the
// document must validate. This confirms the NBSP fix does not over-correct.
func TestElementOnlyXMLWhitespaceAllowed(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="child"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	instance := "<root>  <child/>  </root>"

	v := compileValidator(t, schema)
	doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
	require.NoError(t, err)

	var errs string
	err = validateWithOutput(t, v, doc, &errs)
	require.NoError(t, err, "ignorable XML whitespace in element-only type must validate: %s", errs)
}

// TestXsiTypeWhitespaceCollapsed covers an xsi:type attribute whose raw value
// carries leading, trailing, and internal whitespace. xsi:type is an xs:QName,
// whose whiteSpace facet is "collapse", so " t:derived " must normalize to
// "t:derived" and resolve to the derived type rather than failing resolution.
func TestXsiTypeWhitespaceCollapsed(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
            xmlns:t="urn:t" targetNamespace="urn:t">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:extension base="t:base">
        <xs:sequence>
          <xs:element name="b" type="xs:string"/>
        </xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:base"/>
</xs:schema>`

	instance := `<t:root xmlns:t="urn:t" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"` +
		` xsi:type="  t:derived  "><a>x</a><b>y</b></t:root>`

	v := compileValidator(t, schema)
	doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
	require.NoError(t, err)

	var errs string
	err = validateWithOutput(t, v, doc, &errs)
	require.NoError(t, err, "whitespace-padded xsi:type must collapse and resolve: %s", errs)
}
