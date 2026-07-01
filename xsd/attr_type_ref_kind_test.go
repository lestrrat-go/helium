package xsd_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// An <xs:attribute>'s @type must resolve to a SIMPLE type definition and its
// @ref must resolve to a globally-declared ATTRIBUTE (XSD Structures §3.2.2 /
// src-resolve). A @type naming a complexType (including the ur-type xs:anyType)
// or a @ref naming a component in a different symbol space (an attributeGroup, a
// complexType, a global element) or a name declared nowhere is a schema error.
// Both rules are version-independent, enforced under XSD 1.0 (default) and 1.1.
func TestAttribute_TypeRefKindResolution(t *testing.T) {
	t.Parallel()

	// %s is spliced inside <xs:complexType name="host"> so a local attribute use is
	// legal; every referenceable component is declared at the schema level.
	const localShell = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:a="urn:a" targetNamespace="urn:a">
  <xs:attribute name="ga" type="xs:string"/>
  <xs:element name="el" type="xs:string"/>
  <xs:complexType name="ct"><xs:sequence/></xs:complexType>
  <xs:attributeGroup name="ag"><xs:attribute name="q" type="xs:string"/></xs:attributeGroup>
  <xs:simpleType name="st"><xs:restriction base="xs:string"/></xs:simpleType>
  <xs:complexType name="host">
    <xs:sequence/>
    %s
  </xs:complexType>
</xs:schema>`

	// %s is spliced directly under <xs:schema> to exercise a global declaration.
	const globalShell = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:a="urn:a" targetNamespace="urn:a">
  <xs:complexType name="ct"><xs:sequence/></xs:complexType>
  %s
</xs:schema>`

	const notSimple = "does not resolve to a(n) simple type definition"
	const notAttr = "does not resolve to a(n) attribute declaration"

	invalid := []struct {
		name   string
		shell  string
		attr   string
		expect string
	}{
		// @type resolving to a complex type (attD002).
		{"type-complex-local", localShell, `<xs:attribute name="x" type="a:ct"/>`, notSimple},
		{"type-complex-global", globalShell, `<xs:attribute name="bar" type="a:ct"/>`, notSimple},
		{"type-anyType", localShell, `<xs:attribute name="x" type="xs:anyType"/>`, notSimple},
		// @ref resolving to a non-attribute component (attE003/attE004).
		{"ref-attributegroup", localShell, `<xs:attribute ref="a:ag"/>`, notAttr},
		{"ref-complextype", localShell, `<xs:attribute ref="a:ct"/>`, notAttr},
		{"ref-element", localShell, `<xs:attribute ref="a:el"/>`, notAttr},
		{"ref-undeclared", localShell, `<xs:attribute ref="a:nope"/>`, notAttr},
	}
	for _, tc := range invalid {
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(tc.shell, tc.attr)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				schema, errs, cerr := compileWith(t, v, schemaXML)
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v must reject", v)
				require.Nil(t, schema)
				require.Contains(t, errs, tc.expect, "version=%v", v)
			}
		})
	}

	valid := []struct {
		name  string
		shell string
		attr  string
	}{
		{"type-user-simple", localShell, `<xs:attribute name="x" type="a:st"/>`},
		{"type-builtin-simple", localShell, `<xs:attribute name="x" type="xs:integer"/>`},
		{"type-anysimpletype", localShell, `<xs:attribute name="x" type="xs:anySimpleType"/>`},
		{"type-inline-simple", localShell, `<xs:attribute name="x"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:attribute>`},
		{"ref-global-attr", localShell, `<xs:attribute ref="a:ga"/>`},
	}
	for _, tc := range valid {
		t.Run("valid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(tc.shell, tc.attr)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				schema, errs, cerr := compileWith(t, v, schemaXML)
				require.NoError(t, cerr, "version=%v must accept: %s", v, errs)
				require.NotNil(t, schema)
			}
		})
	}
}
