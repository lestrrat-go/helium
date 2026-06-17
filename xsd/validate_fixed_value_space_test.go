package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestFixedValueSpace verifies that fixed value constraints are compared in the
// declared simple type's value space (applying its whitespace facet), not by
// unconditional TrimSpace (element content) or raw string equality (attribute
// value). A lexically distinct but value-equal instance must be accepted; a
// value-distinct instance must be rejected. For string-family types whose
// whiteSpace facet is "preserve", trailing whitespace is significant.
func TestFixedValueSpace(t *testing.T) {
	type testCase struct {
		name       string
		typ        string // type for the element/attribute
		fixed      string // fixed value
		instance   string // instance value
		wantReject bool
	}

	cases := []testCase{
		// integer — lexical variants that are value-equal to the fixed value
		// must be accepted; a value-distinct instance must be rejected.
		{name: "integer plus sign", typ: "xs:integer", fixed: "1", instance: "+1"},
		{name: "integer leading zero", typ: "xs:integer", fixed: "1", instance: "01"},
		{name: "integer value mismatch", typ: "xs:integer", fixed: "1", instance: "2", wantReject: true},

		// decimal — trailing-zero forms are value-equal.
		{name: "decimal trailing zero", typ: "xs:decimal", fixed: "5", instance: "5.0"},
		{name: "decimal value mismatch", typ: "xs:decimal", fixed: "5", instance: "6", wantReject: true},

		// boolean — "true"/"1" are value-equal.
		{name: "boolean true vs 1", typ: "xs:boolean", fixed: "true", instance: "1"},
		{name: "boolean value mismatch", typ: "xs:boolean", fixed: "true", instance: "0", wantReject: true},

		// string (whiteSpace=preserve) — trailing whitespace is significant, so a
		// fixed value with a trailing space must NOT match an instance without it.
		{name: "string trailing space significant", typ: "xs:string", fixed: "abc ", instance: "abc", wantReject: true},
		{name: "string exact match", typ: "xs:string", fixed: "abc", instance: "abc"},
		{name: "string value mismatch", typ: "xs:string", fixed: "abc", instance: "xyz", wantReject: true},

		// token (whiteSpace=collapse) — leading/trailing/internal whitespace is
		// collapsed, so a padded instance value-matches the fixed value.
		{name: "token collapses whitespace", typ: "xs:token", fixed: "abc", instance: "abc"},
	}

	for _, tc := range cases {
		t.Run("element/"+tc.name, func(t *testing.T) {
			t.Parallel()

			schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="` + tc.typ + `" fixed="` + tc.fixed + `"/>
</xs:schema>`
			instanceXML := `<root>` + tc.instance + `</root>`

			runFixedValueCase(t, schemaXML, instanceXML, tc.wantReject)
		})

		t.Run("attribute/"+tc.name, func(t *testing.T) {
			t.Parallel()

			schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a" type="` + tc.typ + `" fixed="` + tc.fixed + `"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
			instanceXML := `<root a="` + tc.instance + `"/>`

			runFixedValueCase(t, schemaXML, instanceXML, tc.wantReject)
		})
	}
}

func runFixedValueCase(t *testing.T, schemaXML, instanceXML string, wantReject bool) {
	t.Helper()

	schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)

	schema, err := xsd.NewCompiler().Compile(t.Context(), schemaDOC)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
	require.NoError(t, err)

	var errs string
	err = validateWithOutput(t, xsd.NewValidator(schema), doc, &errs)

	if wantReject {
		require.Error(t, err)
		require.Contains(t, errs, "fixed value constraint")
		return
	}
	require.NoError(t, err, "validation errors: %s", errs)
}
