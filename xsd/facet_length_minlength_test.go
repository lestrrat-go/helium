package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestFacetLengthMinMaxCoOccurrence verifies that a `length` facet may co-occur
// with `minLength`/`maxLength` when the values are consistent (XSD Part 2
// §4.3.1/§4.3.2/§4.3.3): minLength ≤ length ≤ maxLength is accepted, while a
// genuine numeric conflict is still rejected. Version-independent (default 1.0).
func TestFacetLengthMinMaxCoOccurrence(t *testing.T) {
	compile := func(t *testing.T, s string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		_, cerr := xsd.NewCompiler().Compile(t.Context(), doc)
		return cerr
	}

	wrap := func(facets string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="t">
    <xs:restriction base="xs:NMTOKENS">
` + facets + `
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="e" type="t"/>
</xs:schema>`
	}

	t.Run("length with consistent minLength accepts", func(t *testing.T) {
		t.Parallel()
		s := wrap(`<xs:length value="5"/><xs:minLength value="1"/>`)
		require.NoError(t, compile(t, s))
	})

	t.Run("length with consistent maxLength accepts", func(t *testing.T) {
		t.Parallel()
		s := wrap(`<xs:length value="5"/><xs:maxLength value="10"/>`)
		require.NoError(t, compile(t, s))
	})

	t.Run("length equal to minLength and maxLength accepts", func(t *testing.T) {
		t.Parallel()
		s := wrap(`<xs:length value="5"/><xs:minLength value="5"/><xs:maxLength value="5"/>`)
		require.NoError(t, compile(t, s))
	})

	t.Run("length less than minLength rejects", func(t *testing.T) {
		t.Parallel()
		s := wrap(`<xs:length value="1"/><xs:minLength value="5"/>`)
		require.ErrorIs(t, compile(t, s), xsd.ErrCompilationFailed)
	})

	t.Run("length greater than maxLength rejects", func(t *testing.T) {
		t.Parallel()
		s := wrap(`<xs:length value="10"/><xs:maxLength value="5"/>`)
		require.ErrorIs(t, compile(t, s), xsd.ErrCompilationFailed)
	})

	t.Run("minLength greater than maxLength rejects", func(t *testing.T) {
		t.Parallel()
		s := wrap(`<xs:minLength value="10"/><xs:maxLength value="5"/>`)
		require.ErrorIs(t, compile(t, s), xsd.ErrCompilationFailed)
	})

	t.Run("length alone accepts", func(t *testing.T) {
		t.Parallel()
		s := wrap(`<xs:length value="5"/>`)
		require.NoError(t, compile(t, s))
	})

	t.Run("derived length outside inherited minLength rejects", func(t *testing.T) {
		t.Parallel()
		s := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="base">
    <xs:restriction base="xs:NMTOKENS"><xs:minLength value="5"/></xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="derived">
    <xs:restriction base="base"><xs:length value="2"/></xs:restriction>
  </xs:simpleType>
  <xs:element name="e" type="derived"/>
</xs:schema>`
		require.ErrorIs(t, compile(t, s), xsd.ErrCompilationFailed)
	})

	t.Run("derived length within inherited minLength accepts", func(t *testing.T) {
		t.Parallel()
		s := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="base">
    <xs:restriction base="xs:NMTOKENS"><xs:minLength value="2"/></xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="derived">
    <xs:restriction base="base"><xs:length value="5"/></xs:restriction>
  </xs:simpleType>
  <xs:element name="e" type="derived"/>
</xs:schema>`
		require.NoError(t, compile(t, s))
	})
}
