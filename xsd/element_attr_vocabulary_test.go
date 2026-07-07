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

// §3.2.2: `targetNamespace` and `inheritable` are XSD 1.1-ONLY additions to the
// <xs:attribute> attribute vocabulary. In XSD 1.0 they are unknown attributes and
// a schema carrying either is invalid (ibmData S3_2_3/s3_2_3si05); in XSD 1.1 the
// vocabulary admits them so the "is not allowed" representation error must NOT
// fire. Version-SPLIT (unlike the other closed-vocabulary attributes).
func TestAttribute_XSD11OnlyVocabulary(t *testing.T) {
	t.Parallel()

	const shell = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t">
  %s
</xs:schema>`

	reject := []struct {
		name string
		attr string
		body string
	}{
		{"global-targetNamespace", "targetNamespace",
			`<xs:attribute name="ga" type="xs:string" targetNamespace="urn:x"/>`},
		{"global-inheritable", "inheritable",
			`<xs:attribute name="ga" type="xs:string" inheritable="true"/>`},
		{"local-targetNamespace", "targetNamespace",
			`<xs:complexType name="ct"><xs:complexContent><xs:restriction base="xs:anyType"><xs:attribute name="a" type="xs:string" targetNamespace="urn:x"/></xs:restriction></xs:complexContent></xs:complexType>`},
		{"local-inheritable", "inheritable",
			`<xs:complexType name="ct"><xs:attribute name="a" type="xs:string" inheritable="true"/></xs:complexType>`},
	}
	for _, tc := range reject {
		t.Run("xsd10-reject/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schema, errs, cerr := compileWith(t, xsd.Version10, fmt.Sprintf(shell, tc.body))
			require.ErrorIs(t, cerr, xsd.ErrCompilationFailed,
				"XSD 1.0 must reject the 1.1-only '%s' on xs:attribute: %s", tc.attr, errs)
			require.Nil(t, schema)
			require.Contains(t, errs, "'"+tc.attr+"' is not allowed")
		})
	}

	// XSD 1.1 admits both: `inheritable` on a global attribute is unconditionally
	// valid, so the schema compiles clean.
	t.Run("xsd11-accept/global-inheritable", func(t *testing.T) {
		t.Parallel()
		_, errs, cerr := compileWith(t, xsd.Version11,
			fmt.Sprintf(shell, `<xs:attribute name="ga" type="xs:string" inheritable="true"/>`))
		require.NoError(t, cerr, "XSD 1.1 must accept inheritable on xs:attribute: %s", errs)
	})

	// XSD 1.1 must NOT raise the closed-vocabulary "is not allowed" error for
	// targetNamespace (the schema may be invalid for OTHER reasons, but never that).
	t.Run("xsd11-vocabulary-ok/local-targetNamespace", func(t *testing.T) {
		t.Parallel()
		_, errs, _ := compileWith(t, xsd.Version11,
			fmt.Sprintf(shell, `<xs:complexType name="ct"><xs:complexContent><xs:restriction base="xs:anyType"><xs:attribute name="a" type="xs:string" targetNamespace="urn:x"/></xs:restriction></xs:complexContent></xs:complexType>`))
		require.NotContains(t, errs, "'targetNamespace' is not allowed",
			"XSD 1.1 vocabulary admits targetNamespace on xs:attribute")
	})
}
