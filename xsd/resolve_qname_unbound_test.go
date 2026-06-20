package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestResolveQNameUnboundPrefix verifies that a QName-valued schema attribute
// (here @type) whose prefix is not bound in scope is rejected at compile time
// rather than silently mapping to the no-namespace declaration. Without this the
// unbound prefix "p" resolved to {} and an instance would validate against the
// unrelated no-namespace type T.
func TestResolveQNameUnboundPrefix(t *testing.T) {
	t.Parallel()

	t.Run("unbound type prefix rejected at compile", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="T">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
  <xs:element name="e" type="p:T"/>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.NotEmpty(t, errs, "expected a compile error for unbound type prefix")
		require.Contains(t, errs, "p:T")
	})

	t.Run("bound type prefix compiles cleanly", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p" targetNamespace="urn:p">
  <xs:simpleType name="T">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
  <xs:element name="e" type="p:T"/>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.Empty(t, errs, "expected no compile error for bound type prefix")
	})

	t.Run("unprefixed type resolves without error", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="T">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
  <xs:element name="e" type="T"/>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.Empty(t, errs, "expected no compile error for unprefixed type ref")
	})

	// The predeclared xml: prefix is always in scope, so a ref through it must
	// not be flagged as unbound. xml:lang is not a type, so it surfaces the
	// later "does not resolve" diagnostic — what matters is the absence of the
	// "is not bound" unbound-prefix error.
	t.Run("xml prefix not flagged as unbound", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xml:lang"/>
</xs:schema>`
		errs := compileSchemaErrors(t, schemaXML)
		require.NotContains(t, errs, "is not bound", "xml prefix must not be reported as unbound")
	})
}
