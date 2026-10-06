package xsd_test

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11Assert covers XSD 1.1 xs:assert on a complex type: the assertion
// is evaluated in 1.1, rejected in 1.0, and a malformed test expression is a
// compile error in 1.1.
func TestVersion11Assert(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="range">
    <xs:complexType>
      <xs:attribute name="min" type="xs:int"/>
      <xs:attribute name="max" type="xs:int"/>
      <xs:assert test="xs:integer(@min) le xs:integer(@max)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	compile := func(t *testing.T, c xsd.Compiler, s string) (*xsd.Schema, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		return c.Compile(t.Context(), doc)
	}
	validate := func(t *testing.T, schema *xsd.Schema, instance string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), doc)
	}

	t.Run("1.1 assertion satisfied", func(t *testing.T) {
		t.Parallel()
		schema, err := compile(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validate(t, schema, `<range min="1" max="5"/>`))
	})

	t.Run("1.1 assertion violated", func(t *testing.T) {
		t.Parallel()
		schema, err := compile(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.ErrorIs(t, validate(t, schema, `<range min="5" max="1"/>`), xsd.ErrValidationFailed)
	})

	t.Run("1.0 rejects xs:assert", func(t *testing.T) {
		t.Parallel()
		// xs:assert is not part of the XSD 1.0 complexType representation.
		_, err := compile(t, xsd.NewCompiler(), schemaXML)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	t.Run("1.1 malformed assert XPath is a compile error", func(t *testing.T) {
		t.Parallel()
		const bad = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:assert test="@a +"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		_, err := compile(t, xsd.NewCompiler().Version(xsd.Version11), bad)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})
}

// TestVersion11AssertTypedAnnotations checks that an xs:assert sees the PSVI
// type of a typed attribute wherever its type is declared: in the validated
// schema, in an imported schema, or in a schema an instance xsi:schemaLocation
// hint loads at validation time. Each assertion holds only when @n atomizes as
// xs:integer, and never as xs:untypedAtomic.
func TestVersion11AssertTypedAnnotations(t *testing.T) {
	t.Parallel()

	const itemType = `<xs:complexType name="itemType">
    <xs:attribute name="n" type="xs:integer"/>
    <xs:assert test="data(@n) instance of xs:integer and @n ge 0"/>
  </xs:complexType>`

	validate := func(t *testing.T, schema *xsd.Schema, instance string) (string, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		verr := xsd.NewValidator(schema).ErrorHandler(collector).Validate(t.Context(), doc)
		require.NoError(t, collector.Close())
		var msgs strings.Builder
		for _, e := range collector.Errors() {
			msgs.WriteString(e.Error())
		}
		return msgs.String(), verr
	}

	t.Run("type in the validated schema", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="item" type="itemType"/>
  `+itemType+`
</xs:schema>`))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
		require.NoError(t, err)
		_, err = validate(t, schema, `<item n="5"/>`)
		require.NoError(t, err)
	})

	t.Run("type in an imported schema", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{
			"typed.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:element name="item" type="t:itemType"/>
  ` + itemType + `
</xs:schema>`)},
		}
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:t">
  <xs:import namespace="urn:t" schemaLocation="typed.xsd"/>
  <xs:element name="root">
    <xs:complexType><xs:sequence><xs:element ref="t:item"/></xs:sequence></xs:complexType>
  </xs:element>
</xs:schema>`))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(xsd.Version11).FS(fsys).Compile(t.Context(), doc)
		require.NoError(t, err)
		_, err = validate(t, schema, `<root><t:item xmlns:t="urn:t" n="5"/></root>`)
		require.NoError(t, err)
	})

	t.Run("type in an instance schemaLocation hint", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{
			"hint.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:h" xmlns:h="urn:h">
  <xs:element name="item" type="h:itemType"/>
  ` + itemType + `
</xs:schema>`)},
		}
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType><xs:sequence><xs:any namespace="urn:h" processContents="strict"/></xs:sequence></xs:complexType>
  </xs:element>
</xs:schema>`))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(xsd.Version11).FS(fsys).Compile(t.Context(), doc)
		require.NoError(t, err)
		_, err = validate(t, schema, `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xsi:schemaLocation="urn:h hint.xsd"><h:item xmlns:h="urn:h" n="5"/></root>`)
		require.NoError(t, err)
	})

	t.Run("many asserted elements", func(t *testing.T) {
		t.Parallel()
		// Enough items that the per-assert copies span several node slabs, with
		// one violation far past the first slab.
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="list">
    <xs:complexType><xs:sequence>
      <xs:element name="item" type="itemType" maxOccurs="unbounded"/>
    </xs:sequence></xs:complexType>
  </xs:element>
  `+itemType+`
</xs:schema>`))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
		require.NoError(t, err)

		var sb strings.Builder
		sb.WriteString("<list>\n")
		for i := range 600 {
			n := i
			if i == 450 {
				n = -1
			}
			fmt.Fprintf(&sb, "<item n=\"%d\"/>\n", n)
		}
		sb.WriteString("</list>\n")
		msgs, err := validate(t, schema, sb.String())
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
		require.Equal(t, 1, strings.Count(msgs, "is not satisfied"), "got: %s", msgs)
		require.Contains(t, msgs, ":452:")
	})
}
