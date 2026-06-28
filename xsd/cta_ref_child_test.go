package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11CTARefChildRejected verifies that an <xs:element ref="..."> may
// carry only (annotation?): a LOCAL xs:alternative child is a schema error (CTA
// belongs to the referenced GLOBAL declaration, not the ref), while an
// annotation-only ref still compiles.
func TestVersion11CTARefChildRejected(t *testing.T) {
	compile := func(t *testing.T, s string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		_, cerr := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
		return cerr
	}

	t.Run("ref with local xs:alternative is a schema error", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T"><xs:sequence/><xs:attribute name="kind" type="xs:string"/></xs:complexType>
  <xs:element name="item" type="T"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="item">
          <xs:alternative test="@kind='x'" type="T"/>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.ErrorIs(t, compile(t, s), xsd.ErrCompilationFailed)
	})

	t.Run("ref with inline complexType child is a schema error", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="item" type="xs:string"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="item">
          <xs:complexType><xs:sequence/></xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.ErrorIs(t, compile(t, s), xsd.ErrCompilationFailed)
	})

	t.Run("ref with annotation-only child compiles", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="item" type="xs:string"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="item">
          <xs:annotation><xs:documentation>ok</xs:documentation></xs:annotation>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.NoError(t, compile(t, s))
	})
}
