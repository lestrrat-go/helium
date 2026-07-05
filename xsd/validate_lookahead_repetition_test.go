package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestLookaheadGroupRepetition covers a content model where a repeating nested
// model group sits as a non-final particle inside an outer repeating sequence:
//
//	sequence(minOccurs=0, maxOccurs=2){ sequence(maxOccurs=2){ a }, b }
//
// The lookahead used to position the following particle (`b`) must honor the
// inner group's own minOccurs/maxOccurs. A single-pass scan under-counts the
// inner group's consumed length and mis-positions `b`, wrongly rejecting valid
// content. An over-max repetition of the inner group must still be rejected.
func TestLookaheadGroupRepetition(t *testing.T) {
	t.Parallel()

	schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence minOccurs="0" maxOccurs="2">
        <xs:sequence maxOccurs="2">
          <xs:element name="a" type="xs:string"/>
        </xs:sequence>
        <xs:element name="b" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Compile(t.Context(), schemaDOC)
	require.NoError(t, err)

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{"single a then b", `<root><a>1</a><b>1</b></root>`, true},
		{"repeated inner a then b", `<root><a>1</a><a>2</a><b>1</b></root>`, true},
		{"two outer reps", `<root><a>1</a><a>2</a><b>1</b><a>3</a><a>4</a><b>2</b></root>`, true},
		{testLabelEmpty, `<root></root>`, true},
		// Inner group permits at most two `a`; a third must be rejected.
		{"inner over max", `<root><a>1</a><a>2</a><a>3</a><b>1</b></root>`, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, xsd.NewValidator(schema), doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err)
		})
	}
}
