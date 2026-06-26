package relaxng_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

// TestNameClassOverlapExceptChoice covers the attribute name-class conflict
// check. An <anyName> with an <except> listing several names does NOT overlap a
// <choice> of exactly those excluded names, so a grammar pairing them as two
// attributes must compile cleanly. A choice that includes a name NOT excluded
// genuinely overlaps and must still be reported.
func TestNameClassOverlapExceptChoice(t *testing.T) {
	t.Parallel()

	compile := func(t *testing.T, schema string) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err)

		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = relaxng.NewCompiler().ErrorHandler(collector).Compile(t.Context(), doc)
		require.NoError(t, err)
		_ = collector.Close()
		_, compileErrors := partitionCompileErrors(collector.Errors())
		return compileErrors
	}

	t.Run("disjoint anyName-except vs choice compiles", func(t *testing.T) {
		const schema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root">
      <attribute>
        <anyName>
          <except><name>foo</name><name>bar</name></except>
        </anyName>
      </attribute>
      <attribute>
        <choice><name>foo</name><name>bar</name></choice>
      </attribute>
    </element>
  </start>
</grammar>`
		require.Empty(t, compile(t, schema),
			"anyName-except{foo,bar} and choice(foo,bar) are disjoint and must not conflict")
	})

	t.Run("genuinely overlapping anyName-except vs choice errors", func(t *testing.T) {
		const schema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root">
      <attribute>
        <anyName>
          <except><name>foo</name></except>
        </anyName>
      </attribute>
      <attribute>
        <choice><name>foo</name><name>bar</name></choice>
      </attribute>
    </element>
  </start>
</grammar>`
		require.NotEmpty(t, compile(t, schema),
			"anyName-except{foo} overlaps choice(foo,bar) on bar and must conflict")
	})
}
