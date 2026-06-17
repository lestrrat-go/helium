package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestIDCFieldMultipleNodes covers cvc-identity-constraint.3: for each selected
// node, every field XPath must evaluate to either an empty node-set or a
// node-set with exactly one member. A field that selects more than one node is a
// validation error rather than being silently reduced to its first node.
func TestIDCFieldMultipleNodes(t *testing.T) {
	t.Parallel()

	const schemaSrc = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="row" maxOccurs="unbounded">
          <xs:complexType>
            <xs:sequence>
              <xs:element name="v" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
            </xs:sequence>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="rowUnique">
      <xs:selector xpath="row"/>
      <xs:field xpath="v"/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	compile := func(t *testing.T) xsd.Validator {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaSrc))
		require.NoError(t, err)
		s, err := xsd.NewCompiler().Compile(t.Context(), doc)
		require.NoError(t, err)
		return xsd.NewValidator(s)
	}

	cases := []struct {
		name        string
		instance    string
		valid       bool
		wantMessage string
	}{
		{
			name:        "field selecting two nodes fails",
			instance:    `<root><row><v>1</v><v>2</v></row></root>`,
			valid:       false,
			wantMessage: "The XPath 'v' of a field of unique identity-constraint 'rowUnique' evaluates to a node-set with more than one member.",
		},
		{
			name:     "field selecting one node passes",
			instance: `<root><row><v>1</v></row><row><v>2</v></row></root>`,
			valid:    true,
		},
		{
			name:     "field selecting zero nodes passes for unique",
			instance: `<root><row/></root>`,
			valid:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v := compile(t)

			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, v, doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err, "expected validation error")
			require.True(t, strings.Contains(errs, tc.wantMessage),
				"expected error message %q in %q", tc.wantMessage, errs)
		})
	}
}
