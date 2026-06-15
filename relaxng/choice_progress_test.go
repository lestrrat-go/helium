package relaxng_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

// TestStartChoiceProgressPreference covers the naive choice matcher reached when
// grammar.start is a bare <choice>. A zero-length <empty/> branch must not shadow
// a later consuming branch: choice(empty, element root{empty}) against <root/>
// should validate.
func TestStartChoiceProgressPreference(t *testing.T) {
	t.Parallel()

	const schema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <choice>
      <empty/>
      <element name="root"><empty/></element>
    </choice>
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

	t.Run("consuming branch chosen over empty", func(t *testing.T) {
		xmlDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
		require.NoError(t, err)

		err = relaxng.NewValidator(grammar).Validate(t.Context(), xmlDoc)
		require.NoError(t, err, "<root/> should validate against choice(empty, element root)")
	})

	t.Run("genuinely invalid instance rejected", func(t *testing.T) {
		xmlDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<wrong/>`))
		require.NoError(t, err)

		err = relaxng.NewValidator(grammar).Validate(t.Context(), xmlDoc)
		require.Error(t, err, "<wrong/> should be rejected")
	})
}
