package relaxng_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

// compileErrorsFor compiles the given RELAX NG schema and returns the fatal
// compile-error text collected during compilation (empty when the schema
// compiles cleanly).
func compileErrorsFor(t *testing.T, schema string) string {
	t.Helper()

	schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
	require.NoError(t, err, "schema should parse")

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, err = relaxng.NewCompiler().ErrorHandler(collector).Compile(t.Context(), schemaDoc)
	require.NoError(t, err, "Compile should not return a hard error")
	_ = collector.Close()
	_, compileErrors := partitionCompileErrors(collector.Errors())
	return compileErrors
}

// TestUnboundPrefixInNameIsCompileError covers D-RNG-002: a QName whose prefix
// is not bound to any in-scope namespace declaration must be a fatal compile
// error rather than being silently treated as the empty namespace. Otherwise a
// schema such as <element name="p:admin"> (without xmlns:p) would wrongly match
// a no-namespace <admin/> instance.
func TestUnboundPrefixInNameIsCompileError(t *testing.T) {
	t.Parallel()

	t.Run("element name unbound prefix", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="p:admin" xmlns="http://relaxng.org/ns/structure/1.0">
  <empty/>
</element>`
		require.NotEmpty(t, compileErrorsFor(t, schema),
			"unbound prefix on <element name> must be a fatal compile error")
	})

	t.Run("attribute name unbound prefix", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <attribute name="p:id"/>
</element>`
		require.NotEmpty(t, compileErrorsFor(t, schema),
			"unbound prefix on <attribute name> must be a fatal compile error")
	})

	t.Run("name class unbound prefix", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <element>
    <name>p:admin</name>
    <empty/>
  </element>
</element>`
		require.NotEmpty(t, compileErrorsFor(t, schema),
			"unbound prefix in <name> name class must be a fatal compile error")
	})

	t.Run("bound prefix compiles cleanly", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="p:admin"
    xmlns="http://relaxng.org/ns/structure/1.0"
    xmlns:p="urn:example:p">
  <empty/>
</element>`
		require.Empty(t, compileErrorsFor(t, schema),
			"a bound prefix must compile without error")
	})

	t.Run("implicit xml prefix compiles cleanly", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <attribute name="xml:lang"/>
</element>`
		require.Empty(t, compileErrorsFor(t, schema),
			"the implicit xml prefix must always be bound")
	})
}

// TestNBSPNotXMLWhitespace covers D-RNG-003: XML whitespace is only #x20, #x9,
// #xA, #xD. A U+00A0 NBSP must NOT be treated as ignorable whitespace, so an
// NBSP between element children, or an NBSP value for an <empty/> pattern, is
// significant content and must make the instance invalid.
func TestNBSPNotXMLWhitespace(t *testing.T) {
	t.Parallel()

	const nbsp = " "

	t.Run("empty pattern rejects NBSP content", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <empty/>
</element>`

		err := validateWith(t, schema, `<a></a>`)
		require.NoError(t, err, "truly empty element matches <empty/>")

		err = validateWith(t, schema, `<a> </a>`)
		require.NoError(t, err, "XML-whitespace-only content matches <empty/>")

		err = validateWith(t, schema, "<a>"+nbsp+"</a>")
		require.Error(t, err, "NBSP is significant content and must not match <empty/>")
	})

	t.Run("NBSP between element children is significant", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="root" xmlns="http://relaxng.org/ns/structure/1.0">
  <element name="a"><empty/></element>
  <element name="b"><empty/></element>
</element>`

		err := validateWith(t, schema, "<root><a/> <b/></root>")
		require.NoError(t, err, "XML whitespace between children is ignorable")

		err = validateWith(t, schema, "<root><a/>"+nbsp+"<b/></root>")
		require.Error(t, err, "NBSP between children is significant text, not ignorable whitespace")
	})

	t.Run("empty attribute pattern rejects NBSP value", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <attribute name="x"><empty/></attribute>
</element>`

		err := validateWith(t, schema, `<a x=""/>`)
		require.NoError(t, err, "empty attribute value matches <empty/>")

		err = validateWith(t, schema, "<a x=\""+nbsp+"\"/>")
		require.Error(t, err, "NBSP attribute value is significant and must not match <empty/>")
	})
}
