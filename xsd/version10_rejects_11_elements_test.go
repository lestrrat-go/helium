package xsd_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestXSD10RejectsXSD11Elements checks that a schema compiled as XSD 1.0 reports
// the elements only XSD 1.1 defines (xs:assert, xs:openContent,
// xs:defaultOpenContent, xs:override, xs:alternative) instead of ignoring them.
// XSD 1.0 §2.4 requires the §3 XML representations, which have none of them, and
// xmllint rejects each one. A 1.1 element guarded by vc:minVersion="1.1" is
// removed by conditional inclusion before these checks, so it still compiles.
func TestXSD10RejectsXSD11Elements(t *testing.T) {
	t.Parallel()

	schemas := map[string]string{
		"assert": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence><xs:element name="a"/></xs:sequence>
      <xs:assert test="false()"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
		"openContent": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:openContent><xs:any processContents="lax"/></xs:openContent>
      <xs:sequence><xs:element name="a"/></xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
		"defaultOpenContent": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:defaultOpenContent><xs:any processContents="lax"/></xs:defaultOpenContent>
  <xs:element name="root"/>
</xs:schema>`,
		"override": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:override schemaLocation="other.xsd"/>
  <xs:element name="root"/>
</xs:schema>`,
		"alternative": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:alternative test="@a" type="xs:string"/>
  </xs:element>
</xs:schema>`,
	}
	for name, schema := range schemas {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, errs, err := compileWith(t, xsd.Version10, schema)
			require.ErrorIs(t, err, xsd.ErrCompilationFailed)
			require.Contains(t, errs, name)
		})
	}

	t.Run("vc:minVersion guarded 1.1 elements compile in 1.0", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning">
  <xs:defaultOpenContent vc:minVersion="1.1"><xs:any processContents="lax"/></xs:defaultOpenContent>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence><xs:element name="a"/></xs:sequence>
      <xs:assert vc:minVersion="1.1" test="false()"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		_, errs, err := compileWith(t, xsd.Version10, schema)
		require.NoError(t, err, errs)
	})
}
