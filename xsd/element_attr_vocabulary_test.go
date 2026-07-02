package xsd_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// §3.3.2 / §3.2.2: xs:element and xs:attribute declarations have a CLOSED
// attribute vocabulary with an `##other` anyAttribute admitting only
// foreign-namespaced attributes. An unknown UNQUALIFIED (no-namespace) attribute
// is a schema-representation error, while vc:/xml:/other-namespace attributes and
// every legitimate xs attribute are accepted. Version-INDEPENDENT.
func TestElementAttribute_ClosedAttrVocabulary(t *testing.T) {
	t.Parallel()

	const shell = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"
    xmlns:foo="urn:foo" targetNamespace="urn:t" xmlns:t="urn:t">
  %s
</xs:schema>`

	invalid := []struct {
		name string
		body string
	}{
		// elemK007: nullable is a misspelling of nillable on a global element.
		{"global-nullable", `<xs:element name="foo" nullable="true"/>`},
		// elemN006: a random unqualified attribute on a global element.
		{"global-foo", `<xs:element name="myElem" type="xs:string" foo="bar"/>`},
		// A random unqualified attribute on a local element.
		{"local-foo", `<xs:complexType name="ct"><xs:sequence><xs:element name="e" type="xs:string" bogus="1"/></xs:sequence></xs:complexType>`},
		// A random unqualified attribute on an element ref.
		{"ref-foo", `<xs:element name="g" type="xs:string"/><xs:complexType name="ct"><xs:sequence><xs:element ref="t:g" bogus="1"/></xs:sequence></xs:complexType>`},
		// An attribute-only attribute (use) on an element is unknown to xs:element.
		{"element-use", `<xs:element name="e2" type="xs:string" use="optional"/>`},
		// attH001: a random unqualified attribute on a local attribute.
		{"attr-value", `<xs:complexType name="ct"><xs:attribute name="a" value="string" use="optional"/></xs:complexType>`},
		// A random unqualified attribute on a global attribute.
		{"global-attr-foo", `<xs:attribute name="ga" type="xs:string" foo="bar"/>`},
	}
	for _, tc := range invalid {
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(shell, tc.body)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				schema, errs, cerr := compileWith(t, v, schemaXML)
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v must reject unknown attribute: %s", v, errs)
				require.Nil(t, schema)
				require.Contains(t, errs, "is not allowed", "version=%v", v)
			}
		})
	}

	valid := []struct {
		name string
		body string
	}{
		// A global element carrying its full legitimate attribute set plus
		// foreign-namespace (vc:, xml:, other) attributes.
		{"global-full", `<xs:element name="g" type="xs:string" id="g1" default="x" nillable="true" abstract="false" final="restriction" block="extension" vc:minVersion="1.1" xml:lang="en" foo:bar="baz"/>`},
		// A local element with occurs/form/ref legitimate attributes.
		{"local-full", `<xs:complexType name="ct"><xs:sequence><xs:element name="e" type="xs:string" minOccurs="0" maxOccurs="2" fixed="v" nillable="true" block="extension" form="qualified" id="e1" vc:minVersion="1.1"/></xs:sequence></xs:complexType>`},
		// An element ref with its legitimate attribute set.
		{"ref-full", `<xs:element name="h" type="xs:string"/><xs:complexType name="ct"><xs:sequence><xs:element ref="t:h" minOccurs="0" maxOccurs="1" id="r1" vc:minVersion="1.1"/></xs:sequence></xs:complexType>`},
		// A global attribute with its legitimate attribute set + foreign attrs.
		{"global-attr-full", `<xs:attribute name="ga" type="xs:string" id="ga1" default="x" vc:minVersion="1.1" xml:lang="en"/>`},
		// A local attribute use with its legitimate attribute set + foreign attrs.
		{"local-attr-full", `<xs:complexType name="ct"><xs:attribute name="a" type="xs:string" use="optional" default="x" form="qualified" id="a1" vc:minVersion="1.1" foo:bar="baz"/></xs:complexType>`},
	}
	for _, tc := range valid {
		t.Run("valid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(shell, tc.body)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				_, errs, cerr := compileWith(t, v, schemaXML)
				require.NoError(t, cerr, "version=%v must accept legitimate + foreign attributes: %s", v, errs)
			}
		})
	}
}
