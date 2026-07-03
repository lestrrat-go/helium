package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestSkipDatatypeIntegrityChecks verifies that the document-wide XSD 1.1
// xs:ID/xs:IDREF/xs:ENTITY integrity walks can be suppressed for a
// fragment-validating caller, while content/type validation still applies.
func TestSkipDatatypeIntegrityChecks(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:s" xmlns:s="urn:s" elementFormDefault="qualified">
  <xs:element name="doc" type="s:docType"/>
  <xs:complexType name="docType"><xs:sequence><xs:element ref="s:test" minOccurs="0" maxOccurs="unbounded"/></xs:sequence></xs:complexType>
  <xs:element name="test"><xs:complexType><xs:attribute name="id" type="xs:ID"/></xs:complexType></xs:element>
</xs:schema>`
	// Duplicate xs:ID values (dup) — a document-wide integrity violation.
	const dupIDs = `<s:doc xmlns:s="urn:s"><s:test id="dup"/><s:test id="dup"/></s:doc>`

	compile := func(t *testing.T) *xsd.Schema {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err)
		s, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
		require.NoError(t, err)
		return s
	}
	validate := func(t *testing.T, v xsd.Validator, instance string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return v.Validate(t.Context(), doc)
	}

	t.Run("default 1.1 rejects duplicate xs:ID", func(t *testing.T) {
		t.Parallel()
		require.Error(t, validate(t, xsd.NewValidator(compile(t)), dupIDs))
	})

	t.Run("SkipDatatypeIntegrityChecks(true) accepts duplicate xs:ID", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validate(t, xsd.NewValidator(compile(t)).SkipDatatypeIntegrityChecks(true), dupIDs))
	})

	t.Run("SkipDatatypeIntegrityChecks(true) still enforces content model", func(t *testing.T) {
		t.Parallel()
		// A stray undeclared child is a content-model error, not a datatype-integrity
		// one, so it is still rejected.
		const badContent = `<s:doc xmlns:s="urn:s"><s:bogus/></s:doc>`
		require.Error(t, validate(t, xsd.NewValidator(compile(t)).SkipDatatypeIntegrityChecks(true), badContent))
	})
}
