package xsd_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestIDConstraintSchemaRules covers the version-independent identity-constraint
// schema-representation rules (XSD Structures 3.11.1) that helium enforces in
// both XSD 1.0 and 1.1: placement of xs:key/xs:unique/xs:keyref and their
// xs:selector/xs:field children, the @name NCName, @refer on non-keyref, the
// xs:selector/xs:field content model and attributes, and keyref field
// cardinality. Each invalid schema must be rejected in BOTH versions, while a
// structurally valid key/unique/keyref pair must still compile.
func TestIDConstraintSchemaRules(t *testing.T) {
	t.Parallel()

	// A structurally valid key/unique/keyref must keep compiling in both
	// versions — the new checks must not over-reject.
	const valid = `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:element name="root">
    <xsd:complexType>
      <xsd:sequence>
        <xsd:element ref="keyElement" maxOccurs="unbounded"/>
        <xsd:element ref="refElement" maxOccurs="unbounded"/>
      </xsd:sequence>
    </xsd:complexType>
    <xsd:key name="k">
      <xsd:annotation><xsd:documentation>ok</xsd:documentation></xsd:annotation>
      <xsd:selector xpath=".//keyElement"/>
      <xsd:field xpath="@a"/>
      <xsd:field xpath="@b"/>
    </xsd:key>
    <xsd:keyref name="kr" refer="k">
      <xsd:selector xpath=".//refElement"/>
      <xsd:field xpath="@a"/>
      <xsd:field xpath="@b"/>
    </xsd:keyref>
    <xsd:unique name="u">
      <xsd:selector xpath=".//keyElement"/>
      <xsd:field xpath="@a"/>
    </xsd:unique>
  </xsd:element>
  <xsd:element name="keyElement">
    <xsd:complexType>
      <xsd:attribute name="a" type="xsd:string"/>
      <xsd:attribute name="b" type="xsd:string"/>
    </xsd:complexType>
  </xsd:element>
  <xsd:element name="refElement">
    <xsd:complexType>
      <xsd:attribute name="a" type="xsd:string"/>
      <xsd:attribute name="b" type="xsd:string"/>
    </xsd:complexType>
  </xsd:element>
</xsd:schema>`

	invalid := map[string]string{
		// xs:unique at the top level (not a child of an element declaration).
		"placement-top-level": `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:element name="root" type="xsd:string"/>
  <xsd:unique name="u">
    <xsd:selector xpath=".//e"/>
    <xsd:field xpath="@a"/>
  </xsd:unique>
</xsd:schema>`,
		// xs:key inside an xs:complexType.
		"placement-in-complextype": `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:element name="root" type="xsd:string"/>
  <xsd:complexType name="t">
    <xsd:key name="k">
      <xsd:selector xpath=".//e"/>
      <xsd:field xpath="@a"/>
    </xsd:key>
  </xsd:complexType>
</xsd:schema>`,
		// xs:field at the top level (not under an identity constraint).
		"placement-orphan-field": `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:element name="root" type="xsd:string"/>
  <xsd:field xpath="@a"/>
</xsd:schema>`,
		// Constraint @name with a colon is not an NCName.
		"name-with-colon": `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:element name="root">
    <xsd:complexType><xsd:sequence/></xsd:complexType>
    <xsd:unique name="a:b">
      <xsd:selector xpath=".//e"/>
      <xsd:field xpath="@a"/>
    </xsd:unique>
  </xsd:element>
</xsd:schema>`,
		// Constraint @name that does not start with a name-start char.
		"name-bad-ncname": `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:element name="root">
    <xsd:complexType><xsd:sequence/></xsd:complexType>
    <xsd:unique name="1foo">
      <xsd:selector xpath=".//e"/>
      <xsd:field xpath="@a"/>
    </xsd:unique>
  </xsd:element>
</xsd:schema>`,
		// @refer on an xs:unique (only xs:keyref may carry @refer).
		"refer-on-unique": `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:element name="root">
    <xsd:complexType><xsd:sequence/></xsd:complexType>
    <xsd:unique name="u" refer="k">
      <xsd:selector xpath=".//e"/>
      <xsd:field xpath="@a"/>
    </xsd:unique>
    <xsd:key name="k">
      <xsd:selector xpath=".//e"/>
      <xsd:field xpath="@a"/>
    </xsd:key>
  </xsd:element>
</xsd:schema>`,
		// xs:selector carrying a disallowed @name attribute.
		"selector-bad-attr": `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:element name="root">
    <xsd:complexType><xsd:sequence/></xsd:complexType>
    <xsd:unique name="u">
      <xsd:selector name="x" xpath=".//e"/>
      <xsd:field xpath="@a"/>
    </xsd:unique>
  </xsd:element>
</xsd:schema>`,
		// xs:selector with a stray (non-annotation) child element.
		"selector-stray-child": `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:element name="root">
    <xsd:complexType><xsd:sequence/></xsd:complexType>
    <xsd:unique name="u">
      <xsd:selector xpath=".//e">
        <xsd:element name="stray" type="xsd:string"/>
      </xsd:selector>
      <xsd:field xpath="@a"/>
    </xsd:unique>
  </xsd:element>
</xsd:schema>`,
		// xs:field carrying a disallowed @id that is not a valid NCName.
		"field-bad-id": `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:element name="root">
    <xsd:complexType><xsd:sequence/></xsd:complexType>
    <xsd:unique name="u">
      <xsd:selector xpath=".//e"/>
      <xsd:field id="123" xpath="@a"/>
    </xsd:unique>
  </xsd:element>
</xsd:schema>`,
		// keyref with a different number of fields than the referenced key.
		"keyref-field-count": `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:element name="root">
    <xsd:complexType><xsd:sequence/></xsd:complexType>
    <xsd:key name="k">
      <xsd:selector xpath=".//e"/>
      <xsd:field xpath="@a"/>
    </xsd:key>
    <xsd:keyref name="kr" refer="k">
      <xsd:selector xpath=".//e"/>
      <xsd:field xpath="@a"/>
      <xsd:field xpath="@b"/>
    </xsd:keyref>
  </xsd:element>
</xsd:schema>`,
		// Duplicate schema-component @id within one document.
		"duplicate-id": `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:element name="a" id="dup"/>
  <xsd:element name="b" id="dup"/>
</xsd:schema>`,
	}

	for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
		label := "1.0"
		if v == xsd.Version11 {
			label = "1.1"
		}
		t.Run(label, func(t *testing.T) {
			t.Parallel()
			_, err := compileVer(t, valid, v)
			require.NoError(t, err, "valid identity constraints must compile")
			for name, src := range invalid {
				_, err := compileVer(t, src, v)
				require.Errorf(t, err, "invalid IDC schema %q must be rejected", name)
			}
		})
	}
}

func TestIDConstraintFullFormNameCollapseEmptyIsInvalidNCName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		nameAttr string
	}{
		{"empty", `name=""`},
		{"whitespace-only", `name="   "`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:element name="root">
    <xsd:complexType><xsd:sequence/></xsd:complexType>
    <xsd:key ` + tc.nameAttr + `>
      <xsd:selector xpath="."/>
      <xsd:field xpath="@id"/>
    </xsd:key>
  </xsd:element>
</xsd:schema>`
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				_, errs, cerr := compileWith(t, v, schemaXML)
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v must reject collapse-empty IDC @name", v)
				require.Contains(t, errs, "is not a valid 'xs:NCName'", "version=%v", v)
				require.NotContains(t, errs, "The attribute 'name' is required but missing.", "version=%v", v)
			}
		})
	}
}
