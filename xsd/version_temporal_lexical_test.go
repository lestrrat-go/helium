package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestVersion11GYearMonthEnumerationRejectsLeadingZeroExpandedYear(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:simpleType>
      <xs:restriction base="xs:gYearMonth">
        <xs:explicitTimezone value="optional"/>
        <xs:enumeration value="0000-02"/>
        <xs:enumeration value="-0000-12"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`

	schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), schemaDoc)
	require.NoError(t, err)

	validate := func(t *testing.T, instance string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), doc)
	}

	require.NoError(t, validate(t, `<doc>0000-02</doc>`))
	require.NoError(t, validate(t, `<doc>-0000-12</doc>`))
	require.ErrorIs(t, validate(t, `<doc>00000-02</doc>`), xsd.ErrValidationFailed)
}
