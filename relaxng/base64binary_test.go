package relaxng_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

// TestBase64BinaryDatatype covers <data type="base64Binary"/> from the XSD
// datatype library. Validation is routed through the shared XSD value validator
// (internal/xsd/value), so RELAX NG and XSD agree: the base64 alphabet (plus
// whitespace) is accepted — including unpadded forms, which the XSD layer treats
// as value-equivalent to their padded counterparts — while characters outside
// the alphabet are rejected.
func TestBase64BinaryDatatype(t *testing.T) {
	t.Parallel()

	const schema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0"
    datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes">
  <start>
    <element name="e">
      <data type="base64Binary"/>
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

	valid := []string{
		"",          // empty is valid base64Binary
		"aGVsbG8=",  // "hello"
		"aGVsbG9v",  // "helloo", no padding
		"YQ==",      // "a"
		"YWI=",      // "ab"
		"aGVs bG8=", // internal whitespace is permitted
		// Unpadded forms are accepted as value-equivalent to their padded
		// counterparts, matching the shared XSD value validator.
		"TQ",  // "M" without padding (== "TQ==")
		"YWI", // "ab" without padding (== "YWI=")
	}
	for _, v := range valid {
		t.Run("valid/"+v, func(t *testing.T) {
			xmlDoc, err := helium.NewParser().Parse(t.Context(), []byte("<e>"+v+"</e>"))
			require.NoError(t, err)
			err = relaxng.NewValidator(grammar).Validate(t.Context(), xmlDoc)
			require.NoError(t, err, "%q should be accepted as base64Binary", v)
		})
	}

	invalid := []string{
		"!!!!",    // chars outside the base64 alphabet
		"YQ==.",   // trailing char outside the alphabet
		"a*b*c*d", // alphabet violation embedded in otherwise base64 text
	}
	for _, v := range invalid {
		t.Run("invalid/"+v, func(t *testing.T) {
			xmlDoc, err := helium.NewParser().Parse(t.Context(), []byte("<e>"+v+"</e>"))
			require.NoError(t, err)
			err = relaxng.NewValidator(grammar).Validate(t.Context(), xmlDoc)
			require.Error(t, err, "%q should be rejected as base64Binary", v)
		})
	}
}
