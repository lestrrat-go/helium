package relaxng_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

const typeInteger = "integer"

// compileXSDDataSchema compiles a grammar whose single element carries a
// <data type="..."/> from the XSD datatype library and returns the grammar.
func compileXSDDataSchema(t *testing.T, typeName string) *relaxng.Grammar {
	t.Helper()
	schema := `<grammar xmlns="http://relaxng.org/ns/structure/1.0"
    datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes">
  <start>
    <element name="e">
      <data type="` + typeName + `"/>
    </element>
  </start>
</grammar>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	grammar, err := relaxng.NewCompiler().ErrorHandler(collector).Compile(t.Context(), doc)
	require.NoError(t, err)
	_ = collector.Close()
	_, compileErrors := partitionCompileErrors(collector.Errors())
	require.Empty(t, compileErrors, "schema should compile without errors")
	return grammar
}

func validateXSDInstance(t *testing.T, grammar *relaxng.Grammar, value string) error {
	t.Helper()
	xmlDoc, err := helium.NewParser().Parse(t.Context(), []byte("<e>"+value+"</e>"))
	require.NoError(t, err)
	return relaxng.NewValidator(grammar).Validate(t.Context(), xmlDoc)
}

// TestXSDDataTypeLexical exercises the XSD datatype library path of <data>.
// Date/time/duration values that are not lexically valid must be rejected
// (the previous code accepted them by mere length/prefix), and an unknown
// datatype name must be rejected rather than silently accepted.
func TestXSDDataTypeLexical(t *testing.T) {
	t.Parallel()

	cases := []struct {
		typeName string
		valid    []string
		invalid  []string
	}{
		{
			typeName: "date",
			valid:    []string{"2020-01-15", "2020-12-31Z", "2020-06-18+09:00"},
			invalid:  []string{"abcdefghij", "2020-13-01", "2020-01-32", "not-a-date"},
		},
		{
			typeName: "dateTime",
			valid:    []string{"2020-01-15T10:20:30", "2020-01-15T10:20:30Z"},
			invalid:  []string{"2020-01-15 10:20:30", "abcdefghijklmnopqrs", "2020-01-15T25:00:00"},
		},
		{
			typeName: "time",
			valid:    []string{"10:20:30", "23:59:59Z"},
			invalid:  []string{"abcdefgh", "25:00:00", "10-20-30"},
		},
		{
			typeName: "duration",
			valid:    []string{"P1Y", "PT1H30M", "P1Y2M3DT4H5M6S"},
			invalid:  []string{"Pxyz", "P", "PT", "1Y"},
		},
		{
			typeName: typeInteger,
			valid:    []string{"5", "-3", "+42"},
			invalid:  []string{"5.0", "abc", "1e3"},
		},
		{
			typeName: "boolean",
			valid:    []string{"true", "false", "1", "0"},
			invalid:  []string{"yes", "True", "2"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.typeName, func(t *testing.T) {
			grammar := compileXSDDataSchema(t, tc.typeName)
			for _, v := range tc.valid {
				t.Run("valid/"+v, func(t *testing.T) {
					require.NoError(t, validateXSDInstance(t, grammar, v),
						"%q should be accepted as %s", v, tc.typeName)
				})
			}
			for _, v := range tc.invalid {
				t.Run("invalid/"+v, func(t *testing.T) {
					require.Error(t, validateXSDInstance(t, grammar, v),
						"%q should be rejected as %s", v, tc.typeName)
				})
			}
		})
	}
}

// TestXSDDataTypeUnknown verifies that an unknown XSD datatype name is rejected
// instead of being silently accepted.
func TestXSDDataTypeUnknown(t *testing.T) {
	t.Parallel()
	grammar := compileXSDDataSchema(t, "notARealXSDType")
	require.Error(t, validateXSDInstance(t, grammar, "anything"),
		"an unknown XSD datatype name should be rejected")
}

// compileXSDValueSchema compiles a grammar whose single element carries a
// <value type="..."> literal from the XSD datatype library.
func compileXSDValueSchema(t *testing.T, typeName, literal string) *relaxng.Grammar {
	t.Helper()
	schema := `<grammar xmlns="http://relaxng.org/ns/structure/1.0"
    datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes">
  <start>
    <element name="e">
      <value type="` + typeName + `">` + literal + `</value>
    </element>
  </start>
</grammar>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	grammar, err := relaxng.NewCompiler().ErrorHandler(collector).Compile(t.Context(), doc)
	require.NoError(t, err)
	_ = collector.Close()
	_, compileErrors := partitionCompileErrors(collector.Errors())
	require.Empty(t, compileErrors, "schema should compile without errors")
	return grammar
}

// TestXSDValueValueSpace verifies that <value> with an XSD datatype compares in
// value space, so lexically distinct but value-equal forms match.
func TestXSDValueValueSpace(t *testing.T) {
	t.Parallel()

	grammar := compileXSDValueSchema(t, typeInteger, "5")

	for _, v := range []string{"5", "+5", "05"} {
		t.Run("match/"+v, func(t *testing.T) {
			require.NoError(t, validateXSDInstance(t, grammar, v),
				"%q should value-match integer 5", v)
		})
	}
	for _, v := range []string{"6", "5.0", "abc"} {
		t.Run("nomatch/"+v, func(t *testing.T) {
			require.Error(t, validateXSDInstance(t, grammar, v),
				"%q should not value-match integer 5", v)
		})
	}
}

// TestXSDValueInvalidLexical verifies that a <value> literal which is not a
// valid lexical form for its XSD datatype is rejected even when the instance
// text is byte-identical to that literal. The previous code returned success on
// the lexical equality fast-path before validating either form, wrongly
// accepting identical-but-invalid typed values. This must hold for both
// value-space-comparable types (integer, date) and constrained NON-comparable
// string-family types (NCName), the latter of which the prior fix missed
// because its lexical gate ran only for comparable types.
func TestXSDValueInvalidLexical(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		typeName string
		literal  string
		instance string
	}{
		{"integer-5.0", typeInteger, "5.0", "5.0"},
		{"date-not-a-date", "date", "not-a-date", "not-a-date"},
		// NCName is constrained but NOT value-space-comparable. "1foo" is an
		// invalid NCName (leading digit), so an identical-but-invalid lexical
		// must still be rejected.
		{"ncname-leading-digit", "NCName", "1foo", "1foo"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			grammar := compileXSDValueSchema(t, tc.typeName, tc.literal)
			require.Error(t, validateXSDInstance(t, grammar, tc.instance),
				"identical-but-invalid lexical %q should be rejected as %s",
				tc.instance, tc.typeName)
		})
	}

	// A valid value with distinct lexical forms must still value-match.
	t.Run("valid-value-equality", func(t *testing.T) {
		grammar := compileXSDValueSchema(t, typeInteger, "5")
		require.NoError(t, validateXSDInstance(t, grammar, "+5"),
			"valid integer +5 should value-match 5")
	})

	// A valid NCName literal must still match a byte-identical valid instance
	// (lexical equality for a constrained non-comparable type).
	t.Run("valid-ncname-equality", func(t *testing.T) {
		grammar := compileXSDValueSchema(t, "NCName", "foo")
		require.NoError(t, validateXSDInstance(t, grammar, "foo"),
			"valid NCName foo should match literal foo")
	})
}
