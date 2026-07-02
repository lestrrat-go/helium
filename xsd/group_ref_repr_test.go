package xsd_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// A <xs:group> appearing as a particle inside a content model — directly under
// an xs:complexType, inside an xs:extension/xs:restriction derivation body, or
// nested in an xs:sequence/xs:choice/xs:all — is a model group REFERENCE
// (§3.8.2). Its only naming attribute is 'ref'; a '@name' (the top-level
// DEFINITION form) is a schema-representation error, as is a missing 'ref'.
// This is version-independent, so it is enforced under both XSD 1.0 (default)
// and XSD 1.1. Mirrors W3C xsdtests groupC004-groupC008.
func TestGroupReference_NameNotAllowedInContentModel(t *testing.T) {
	t.Parallel()

	const defn = `
  <xs:group name="xyz">
    <xs:sequence>
      <xs:element name="A"/>
      <xs:element name="B"/>
    </xs:sequence>
  </xs:group>`

	// Each %s is the offending <xs:group> particle under test.
	invalidShells := map[string]string{
		"direct-complexType": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="ct">%s</xs:complexType>` + defn + `
</xs:schema>`,
		"in-sequence": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="ct"><xs:sequence minOccurs="0">%s</xs:sequence></xs:complexType>` + defn + `
</xs:schema>`,
		"in-choice": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="ct"><xs:choice minOccurs="0">%s</xs:choice></xs:complexType>` + defn + `
</xs:schema>`,
		"in-extension": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base"><xs:sequence><xs:element name="A"/></xs:sequence></xs:complexType>
  <xs:complexType name="ct"><xs:complexContent><xs:extension base="base">%s</xs:extension></xs:complexContent></xs:complexType>` + defn + `
</xs:schema>`,
		"in-restriction": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base"><xs:sequence minOccurs="0"><xs:element name="A" minOccurs="0"/><xs:element name="B" minOccurs="0"/></xs:sequence></xs:complexType>
  <xs:complexType name="ct"><xs:complexContent><xs:restriction base="base">%s</xs:restriction></xs:complexContent></xs:complexType>` + defn + `
</xs:schema>`,
	}

	// Both the DEFINITION form (a '@name') and a bare <xs:group> (no 'ref') are
	// representation errors as content-model particles.
	offenders := map[string]string{
		"name-form":   `<xs:group name="xyz"/>`,
		"missing-ref": `<xs:group/>`,
	}

	for ctx, shell := range invalidShells {
		for oname, particle := range offenders {
			t.Run("invalid/"+oname+"/"+ctx, func(t *testing.T) {
				t.Parallel()
				schemaXML := fmt.Sprintf(shell, particle)
				for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
					schema, errs, cerr := compileWith(t, v, schemaXML)
					require.ErrorIs(t, cerr, xsd.ErrCompilationFailed,
						"version=%v ctx=%s must reject %s in a content model; errs=%s", v, ctx, particle, errs)
					require.Nil(t, schema)
				}
			})
		}
	}

	// A proper <xs:group ref> in an unconditionally-valid position must still
	// compile (no over-rejection). Extension/restriction add orthogonal
	// derivation constraints, so the ref form is exercised only where a plain
	// group reference is always valid.
	validShells := map[string]string{
		"direct-complexType": invalidShells["direct-complexType"],
		"in-sequence":        invalidShells["in-sequence"],
		"in-choice":          invalidShells["in-choice"],
	}
	for ctx, shell := range validShells {
		t.Run("valid/ref-form/"+ctx, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(shell, `<xs:group ref="xyz"/>`)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				_, errs, cerr := compileWith(t, v, schemaXML)
				require.NoError(t, cerr,
					"version=%v ctx=%s must accept a proper <xs:group ref>; errs=%s", v, ctx, errs)
			}
		})
	}

	// A group reference carrying a child annotation is valid content (annotation?).
	t.Run("valid/ref-with-annotation", func(t *testing.T) {
		t.Parallel()
		schemaXML := fmt.Sprintf(invalidShells["in-sequence"],
			`<xs:group ref="xyz"><xs:annotation><xs:documentation>ok</xs:documentation></xs:annotation></xs:group>`)
		for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
			_, errs, cerr := compileWith(t, v, schemaXML)
			require.NoError(t, cerr, "version=%v must accept a group ref with an annotation child; errs=%s", v, errs)
		}
	})
}
