package xsd_test

import (
	"fmt"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestIntegerFractionDigitsFixed verifies that xs:integer and every type derived
// from it carry the built-in FIXED facet fractionDigits=0 (§3.3.13). A
// restriction setting fractionDigits to any non-zero value is a schema error in
// BOTH XSD 1.0 and 1.1; an explicit fractionDigits="0" is permitted (it equals
// the fixed value), and xs:decimal — which does NOT fix fractionDigits — still
// accepts a non-zero fractionDigits.
func TestIntegerFractionDigitsFixed(t *testing.T) {
	t.Parallel()

	const wantMsg = "does not match the fixed value '0'"

	schemaFor := func(base, frac string) string {
		return fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="t">
    <xs:restriction base="%s">
      <xs:fractionDigits value="%s"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="t"/>
</xs:schema>`, base, frac)
	}

	compile := func(t *testing.T, schemaXML string) (string, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, cerr := xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
		_, errStr := partitionCompileErrors(collector.Errors())
		return errStr, cerr
	}

	// Every integer-family built-in fixes fractionDigits=0; a non-zero value is a
	// compile error.
	integerFamily := []string{
		"xs:integer", "xs:nonPositiveInteger", "xs:negativeInteger",
		"xs:long", "xs:int", "xs:short", "xs:byte",
		"xs:nonNegativeInteger", "xs:unsignedLong", "xs:unsignedInt",
		"xs:unsignedShort", "xs:unsignedByte", "xs:positiveInteger",
	}
	for _, base := range integerFamily {
		t.Run("reject non-zero fractionDigits on "+base, func(t *testing.T) {
			t.Parallel()
			errStr, cerr := compile(t, schemaFor(base, "1"))
			requireCompileResultErr(t, cerr)
			require.Contains(t, errStr, wantMsg)
		})
	}

	t.Run("accept fractionDigits=0 on xs:integer", func(t *testing.T) {
		t.Parallel()
		_, cerr := compile(t, schemaFor("xs:integer", "0"))
		require.NoError(t, cerr)
	})

	t.Run("accept non-zero fractionDigits on xs:decimal", func(t *testing.T) {
		t.Parallel()
		_, cerr := compile(t, schemaFor("xs:decimal", "1"))
		require.NoError(t, cerr)
	})

	t.Run("reject non-zero fractionDigits on a user integer restriction", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="mine">
    <xs:restriction base="xs:integer">
      <xs:minInclusive value="0"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="t">
    <xs:restriction base="mine">
      <xs:fractionDigits value="2"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="t"/>
</xs:schema>`
		errStr, cerr := compile(t, schemaXML)
		requireCompileResultErr(t, cerr)
		require.Contains(t, errStr, wantMsg)
	})
}
