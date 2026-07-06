package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestIDCSingletonListEqualsAtomic covers the XSD 1.1 cvc-identity-constraint rule
// that a keyref field whose value is a SINGLETON list equals a key field whose
// value is ATOMIC when their single canonical values match (W3C saxonData Id id022,
// "Atomic value equal to singleton list"). The key field @key is atomic (xs:Name)
// and the keyref field @ref is a list (xs:list itemType="xs:Name"); a one-item ref
// list must resolve against the atomic key. A MULTI-item list must NOT collapse to
// an atomic, so a two-item ref never matches a single-valued key. In XSD 1.0 the
// singleton-list unwrap does not apply (byte-identical to origin): the atomic key
// and the list keyref value are keyed distinctly, so the keyref finds no match.
func TestIDCSingletonListEqualsAtomic(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" elementFormDefault="qualified" attributeFormDefault="unqualified">
  <xs:element name="doc">
    <xs:complexType>
       <xs:sequence>
          <xs:element ref="para" maxOccurs="unbounded"/>
       </xs:sequence>
    </xs:complexType>
    <xs:key name="k">
        <xs:selector xpath="para"/>
        <xs:field xpath="@key"/>
    </xs:key>
    <xs:keyref name="r" refer="k">
        <xs:selector xpath="para"/>
        <xs:field xpath="@ref"/>
    </xs:keyref>
  </xs:element>
  <xs:element name="para">
    <xs:complexType>
      <xs:simpleContent>
        <xs:extension base="xs:string">
            <xs:attribute name="key" type="xs:Name" use="required"/>
            <xs:attribute name="ref" type="list-of-tokens" use="optional"/>
        </xs:extension>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
  <xs:simpleType name="list-of-tokens">
    <xs:list itemType="xs:Name"/>
  </xs:simpleType>
</xs:schema>`

	cases := []struct {
		name     string
		instance string
		v11      bool
		valid    bool
	}{
		{
			name: "1.1 singleton ref list matches atomic key",
			instance: `<doc><para key="alpha"/><para key="beta" ref="alpha"/>` +
				`<para key="gamma" ref="beta"/></doc>`,
			v11:   true,
			valid: true,
		},
		{
			name: "1.1 multi-item ref list does not match single-valued key",
			instance: `<doc><para key="alpha"/><para key="beta" ref="alpha gamma"/>` +
				`<para key="gamma"/></doc>`,
			v11:   true,
			valid: false,
		},
		{
			name: "1.0 singleton ref list stays distinct from atomic key",
			instance: `<doc><para key="alpha"/><para key="beta" ref="alpha"/>` +
				`<para key="gamma" ref="beta"/></doc>`,
			v11:   false,
			valid: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v := compileValidatorVersion(t, schema, tc.v11)
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, v, doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err, "expected validation error")
		})
	}
}
