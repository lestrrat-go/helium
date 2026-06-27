package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11XPathUnboundPrefix covers XSD 1.1 static namespace validation of
// xs:assert and xs:alternative @test expressions: an unbound namespace prefix is
// a schema (compile) error, while a properly bound prefix compiles fine.
func TestVersion11XPathUnboundPrefix(t *testing.T) {
	compile := func(t *testing.T, s string) (*xsd.Schema, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		return xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
	}

	t.Run("assert with unbound prefix is a compile error", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:assert test="p:child"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	t.Run("assert with bound prefix compiles", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p">
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence><xs:element name="c" type="xs:string"/></xs:sequence>
      <xs:assert test="count(p:nope) ge 0"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.NoError(t, err)
	})

	t.Run("alternative with unbound prefix is a compile error", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T"><xs:sequence/></xs:complexType>
  <xs:element name="e" type="T">
    <xs:alternative test="p:child" type="T"/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	t.Run("alternative with bound prefix compiles", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p">
  <xs:complexType name="T"><xs:sequence/></xs:complexType>
  <xs:element name="e" type="T">
    <xs:alternative test="exists(p:nope)" type="T"/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.NoError(t, err)
	})
}
