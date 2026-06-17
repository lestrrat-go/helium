package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestIDCKeyMissingField covers cvc-identity-constraint.4.2.1: every field of an
// xs:key must evaluate to a node for each selected node. An xs:key with an absent
// field is a validation error, while xs:unique (and xs:keyref) tolerate absent
// fields by dropping the node from the qualified node-set.
func TestIDCKeyMissingField(t *testing.T) {
	t.Parallel()

	schema := func(kind string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="id" type="xs:string"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:` + kind + ` name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="@id"/>
    </xs:` + kind + `>
  </xs:element>
</xs:schema>`
	}

	compile := func(t *testing.T, src string) xsd.Validator {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		s, err := xsd.NewCompiler().Compile(t.Context(), doc)
		require.NoError(t, err)
		return xsd.NewValidator(s)
	}

	cases := []struct {
		name        string
		schemaKind  string
		instance    string
		valid       bool
		wantMessage string
	}{
		{
			name:        "key with missing field fails",
			schemaKind:  "key",
			instance:    `<root><item/></root>`,
			valid:       false,
			wantMessage: "Not all fields of key identity-constraint 'itemKey' evaluate to a node.",
		},
		{
			name:       "unique with missing field passes",
			schemaKind: "unique",
			instance:   `<root><item/></root>`,
			valid:      true,
		},
		{
			name:       "key with all fields present and unique passes",
			schemaKind: "key",
			instance:   `<root><item id="a"/><item id="b"/></root>`,
			valid:      true,
		},
		{
			name:        "key with one of several nodes missing field fails",
			schemaKind:  "key",
			instance:    `<root><item id="a"/><item/></root>`,
			valid:       false,
			wantMessage: "Not all fields of key identity-constraint 'itemKey' evaluate to a node.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v := compile(t, schema(tc.schemaKind))

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
