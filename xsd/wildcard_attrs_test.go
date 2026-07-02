package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestWildcardAttrRepresentation covers the version-INDEPENDENT XML representation
// of an xs:any / xs:anyAttribute wildcard's attributes (XSD §3.10.2): the
// permitted unqualified attributes are {id, namespace, processContents} for both,
// plus {minOccurs, maxOccurs} for the ELEMENT wildcard xs:any only. An unexpected
// unqualified attribute, or minOccurs/maxOccurs on xs:anyAttribute, is a schema
// error; a foreign-namespaced attribute is allowed. W3C Wildcards_w3c:
// wildI002/wildI003 (stray attr on xs:any) and wildQ002/wildQ003/wildQ004
// (occurrence attrs on xs:anyAttribute).
func TestWildcardAttrRepresentation(t *testing.T) {
	t.Parallel()

	const notAllowed = "is not allowed"

	compile := func(t *testing.T, src string) string {
		t.Helper()
		const mainXSD = "main.xsd"
		fsys := fstest.MapFS{mainXSD: &fstest.MapFile{Data: []byte(src)}}
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, _ = xsd.NewCompiler().Label(mainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		require.NoError(t, collector.Close())
		_, errStr := partitionCompileErrors(collector.Errors())
		return errStr
	}

	t.Run("stray unqualified attr on xs:any rejected (wildI003)", func(t *testing.T) {
		t.Parallel()
		errStr := compile(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="foo">
    <xs:complexType>
      <xs:sequence>
        <xs:any namespace="##other" processContents="lax" foo="bar"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`)
		require.Contains(t, errStr, notAllowed)
		require.Contains(t, errStr, "foo")
	})

	t.Run("unqualified attr alongside foreign attr on xs:any rejected (wildI002)", func(t *testing.T) {
		t.Parallel()
		// a:b is a foreign (non-schema) namespaced attribute and is ALLOWED; the
		// unqualified b="c" is the representation error.
		errStr := compile(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="foo">
    <xs:complexType>
      <xs:sequence>
        <xs:any namespace="##other" processContents="lax" a:b="c" b="c" xmlns:a="http://foo"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`)
		require.Contains(t, errStr, notAllowed)
		require.Contains(t, errStr, "'b'")
	})

	t.Run("minOccurs on xs:anyAttribute rejected (wildQ002)", func(t *testing.T) {
		t.Parallel()
		errStr := compile(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="foo">
    <xs:complexType>
      <xs:simpleContent>
        <xs:extension base="xs:string">
          <xs:anyAttribute minOccurs="2"/>
        </xs:extension>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`)
		require.Contains(t, errStr, notAllowed)
		require.Contains(t, errStr, "minOccurs")
	})

	t.Run("maxOccurs on xs:anyAttribute rejected (wildQ003)", func(t *testing.T) {
		t.Parallel()
		errStr := compile(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="foo">
    <xs:complexType>
      <xs:simpleContent>
        <xs:extension base="xs:string">
          <xs:anyAttribute maxOccurs="2"/>
        </xs:extension>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`)
		require.Contains(t, errStr, notAllowed)
		require.Contains(t, errStr, "maxOccurs")
	})

	t.Run("minOccurs+maxOccurs on xs:anyAttribute rejected (wildQ004)", func(t *testing.T) {
		t.Parallel()
		errStr := compile(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="foo">
    <xs:complexType>
      <xs:simpleContent>
        <xs:extension base="xs:string">
          <xs:anyAttribute minOccurs="2" maxOccurs="unbounded"/>
        </xs:extension>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`)
		require.Contains(t, errStr, notAllowed)
	})

	t.Run("minOccurs+maxOccurs on xs:any compiles clean (no over-rejection)", func(t *testing.T) {
		t.Parallel()
		errStr := compile(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="foo">
    <xs:complexType>
      <xs:sequence>
        <xs:any id="w" namespace="##other" processContents="lax" minOccurs="1" maxOccurs="2"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`)
		require.False(t, strings.Contains(errStr, notAllowed),
			"a well-formed xs:any must compile clean; got: %q", errStr)
	})

	t.Run("plain xs:anyAttribute compiles clean (no over-rejection)", func(t *testing.T) {
		t.Parallel()
		errStr := compile(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="foo">
    <xs:complexType>
      <xs:anyAttribute id="w" namespace="##other" processContents="lax"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`)
		require.False(t, strings.Contains(errStr, notAllowed),
			"a well-formed xs:anyAttribute must compile clean; got: %q", errStr)
	})

	t.Run("foreign-namespaced attr on wildcard allowed", func(t *testing.T) {
		t.Parallel()
		errStr := compile(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:a="urn:a">
  <xs:element name="foo">
    <xs:complexType>
      <xs:anyAttribute namespace="##other" processContents="lax" a:custom="x"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`)
		require.False(t, strings.Contains(errStr, notAllowed),
			"a foreign-namespaced attribute must be allowed; got: %q", errStr)
	})
}
