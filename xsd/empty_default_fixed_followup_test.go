package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDefaultAndFixedBothPresentEmpty checks that a schema declaring both
// default="" and fixed="x" (both physically present, even when default is
// empty) is rejected as a schema parser error. Presence-based mutual-exclusion
// rules must use attribute presence, not non-empty value.
func TestDefaultAndFixedBothPresentEmpty(t *testing.T) {
	t.Parallel()

	t.Run("global element default empty + fixed present", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string" default="" fixed="x"/>
</xs:schema>`
		_, errs := compileWithErrors(t, schemaXML)
		require.Contains(t, errs, "'default' and 'fixed' are mutually exclusive")
	})

	t.Run("attribute default empty + fixed present", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a" type="xs:string" default="" fixed="x"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		_, errs := compileWithErrors(t, schemaXML)
		require.Contains(t, errs, "'default' and 'fixed' are mutually exclusive")
	})
}

// TestForeignNamespacedFixedNotXSD checks that a foreign-namespaced fixed=""
// attribute is NOT treated as the XSD fixed="" constraint. Schema attribute
// lookup must require an unqualified (no-namespace) attribute.
func TestForeignNamespacedFixedNotXSD(t *testing.T) {
	t.Parallel()

	// Here other:fixed="" must be ignored, so the element has no fixed
	// constraint and non-empty content must validate cleanly.
	schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:other="urn:other">
  <xs:element name="root" type="xs:string" other:fixed=""/>
</xs:schema>`

	_, errs := compileWithErrors(t, schemaXML)
	require.Empty(t, errs, "unexpected compile errors")

	require.NoError(t, compileAndValidate(t, schemaXML, `<root>x</root>`, nil))
}

// TestQualifiedAttributeDefaultInsertedInNamespace checks that a default value
// inserted for an absent qualified attribute carries its declared namespace, so
// an xs:key field like @t:a finds it.
func TestQualifiedAttributeDefaultInsertedInNamespace(t *testing.T) {
	t.Parallel()

	schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:t="urn:t"
  targetNamespace="urn:t"
  elementFormDefault="qualified"
  attributeFormDefault="qualified">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="a" type="xs:string" default="dv"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="k">
      <xs:selector xpath="t:item"/>
      <xs:field xpath="@t:a"/>
    </xs:key>
  </xs:element>
</xs:schema>`

	// Two items both omit @t:a, so the inserted default "dv" makes their key
	// sequences collide -> a duplicate-key error proves the default was
	// inserted in the urn:t namespace and matched by @t:a.
	instanceXML := `<t:root xmlns:t="urn:t"><t:item/><t:item/></t:root>`

	var out string
	err := compileAndValidate(t, schemaXML, instanceXML, &out)
	require.Error(t, err)
	require.Contains(t, out, "Duplicate key-sequence")
}

// TestEmptyDefaultInvalidForType checks that a retained empty default like
// default="" on an xs:integer attribute is rejected: "" is not a valid integer,
// so the schema (or validation) must report an error rather than silently
// inserting an invalid default.
func TestEmptyDefaultInvalidForType(t *testing.T) {
	t.Parallel()

	schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a" type="xs:integer" default=""/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	_, errs := compileWithErrors(t, schemaXML)
	require.NotEmpty(t, errs, "expected a schema error for invalid empty default")
}

// TestEmptyDefaultInvalidForTypeViaRef checks the same invalid-constraint
// detection for a ref'd attribute use: <xs:attribute ref="a" default=""/>
// where global "a" is xs:integer. The empty default is invalid for the
// resolved integer type and must produce a schema error.
func TestEmptyDefaultInvalidForTypeViaRef(t *testing.T) {
	t.Parallel()

	t.Run("default empty on integer ref", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="a" type="xs:integer"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute ref="a" default=""/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		_, errs := compileWithErrors(t, schemaXML)
		require.NotEmpty(t, errs, "expected a schema error for invalid empty default on ref")
	})

	t.Run("fixed empty on integer ref", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="a" type="xs:integer"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute ref="a" fixed=""/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		_, errs := compileWithErrors(t, schemaXML)
		require.NotEmpty(t, errs, "expected a schema error for invalid empty fixed on ref")
	})
}
