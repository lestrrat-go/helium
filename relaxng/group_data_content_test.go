package relaxng_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

// TestGroupDataContentTypeError covers the RELAX NG content-type restriction
// (data/value/list content may not be mixed with element content in the same
// context). The detection must see data wrapped in a composite pattern such as
// <group>, not only when it appears as a direct child of <element>.
func TestGroupDataContentTypeError(t *testing.T) {
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

	t.Run("data and element mixed via group is rejected", func(t *testing.T) {
		const schema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root">
      <group>
        <data type="string"/>
        <element name="child"><text/></element>
      </group>
    </element>
  </start>
</grammar>`
		require.NotEmpty(t, compile(t, schema),
			"data wrapped in a group mixed with element content must be a content-type error")
	})

	t.Run("data and element mixed directly is rejected", func(t *testing.T) {
		const schema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root">
      <data type="string"/>
      <element name="child"><text/></element>
    </element>
  </start>
</grammar>`
		require.NotEmpty(t, compile(t, schema),
			"data mixed directly with element content must be a content-type error")
	})

	t.Run("data-only content compiles", func(t *testing.T) {
		const schema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root">
      <group>
        <data type="string"/>
      </group>
    </element>
  </start>
</grammar>`
		require.Empty(t, compile(t, schema),
			"data-only content with no element content must compile")
	})

	t.Run("data behind a ref mixed with element content is rejected", func(t *testing.T) {
		const schema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root">
      <ref name="d"/>
      <element name="child"><text/></element>
    </element>
  </start>
  <define name="d"><data type="string"/></define>
</grammar>`
		require.NotEmpty(t, compile(t, schema),
			"data referenced via <ref> mixed with element content must be a content-type error")
	})

	t.Run("data behind a parentRef mixed with element content is rejected", func(t *testing.T) {
		const schema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root">
      <grammar>
        <start>
          <element name="inner">
            <parentRef name="d"/>
            <element name="child"><text/></element>
          </element>
        </start>
      </grammar>
    </element>
  </start>
  <define name="d"><data type="string"/></define>
</grammar>`
		require.NotEmpty(t, compile(t, schema),
			"data referenced via <parentRef> mixed with element content must be a content-type error")
	})

	t.Run("element behind a ref mixed with data is rejected", func(t *testing.T) {
		const schema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root">
      <data type="string"/>
      <ref name="d"/>
    </element>
  </start>
  <define name="d"><element name="child"><text/></element></define>
</grammar>`
		require.NotEmpty(t, compile(t, schema),
			"element referenced via <ref> mixed with data content must be a content-type error")
	})

	t.Run("element-only behind a ref compiles", func(t *testing.T) {
		const schema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root">
      <ref name="d"/>
      <element name="child"><text/></element>
    </element>
  </start>
  <define name="d"><element name="other"><text/></element></define>
</grammar>`
		require.Empty(t, compile(t, schema),
			"a ref resolving to an element contributes element content, not data; must compile")
	})

	t.Run("element content with attribute data compiles", func(t *testing.T) {
		const schema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root">
      <group>
        <attribute name="a"><data type="string"/></attribute>
        <element name="child"><text/></element>
      </group>
    </element>
  </start>
</grammar>`
		require.Empty(t, compile(t, schema),
			"data inside an attribute is attribute content, not element content; must not conflict")
	})
}
