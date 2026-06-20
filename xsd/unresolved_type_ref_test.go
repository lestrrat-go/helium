package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestUnresolvedTypeRef verifies that a reference to a missing user-defined
// type (base type, list item type, union member type, or element type) is
// reported as a fatal schema parser error instead of being silently inserted
// as an empty placeholder, which would let an invalid schema compile and
// validate documents as if the missing type existed.
func TestUnresolvedTypeRef(t *testing.T) {
	t.Parallel()

	const wantMsg = "does not resolve to a(n) type definition"

	compileErrors := func(t *testing.T, schemaXML string) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
		requireCompileResultErr(t, err)
		_, errors := partitionCompileErrors(collector.Errors())
		return errors
	}

	t.Run("missing base type", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="derived">
    <xs:restriction base="MissingBase"/>
  </xs:simpleType>
  <xs:element name="root" type="derived"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
	})

	t.Run("missing list item type", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="myList">
    <xs:list itemType="MissingItem"/>
  </xs:simpleType>
  <xs:element name="root" type="myList"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
	})

	t.Run("missing union member type", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="myUnion">
    <xs:union memberTypes="xs:string MissingMember"/>
  </xs:simpleType>
  <xs:element name="root" type="myUnion"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
	})

	t.Run("missing element type", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="MissingType"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
	})

	// Inline (local) simpleTypes must also report unresolved type refs, not
	// just top-level named simpleTypes. Before recording source info for local
	// simple types, reportUnresolvedTypeRef returned early and these compiled
	// silently.
	t.Run("missing base type in inline simpleType", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="MissingBase"/>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
	})

	t.Run("missing list item type in inline simpleType", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:simpleType>
      <xs:list itemType="MissingItem"/>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
	})

	t.Run("missing union member type in inline simpleType", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:simpleType>
      <xs:union memberTypes="xs:string MissingMember"/>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
	})
}
