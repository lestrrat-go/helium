package xsd_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// A LOCAL (anonymous/inline) xs:complexType — nested in an xs:element or an
// xs:alternative — must NOT carry a @name (XSD Structures §3.4.2: localComplexType
// has no {name}). This is version-independent, so it is enforced under both XSD
// 1.0 (default) and XSD 1.1. W3C ComplexType_w3c ctA042. A top-level named
// complexType and an anonymous local complexType without @name still compile.
func TestLocalComplexType_MustNotHaveName(t *testing.T) {
	t.Parallel()

	// ctA042: a local complexType inside an xs:element carries @name="fooType".
	const named = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="myElement">
    <xs:complexType name="fooType">
      <xs:simpleContent>
        <xs:extension base="xs:string">
          <xs:attribute name="attrTest"/>
        </xs:extension>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	t.Run("invalid/named-local", func(t *testing.T) {
		t.Parallel()
		for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
			schema, errs, cerr := compileWith(t, v, named)
			require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v must reject named local complexType", v)
			require.Nil(t, schema)
			require.Contains(t, errs, "A local complexType definition must not have a 'name' attribute.", "version=%v", v)
		}
	})

	// An anonymous local complexType (no @name) must still compile.
	const anon = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="myElement">
    <xs:complexType>
      <xs:simpleContent>
        <xs:extension base="xs:string">
          <xs:attribute name="attrTest"/>
        </xs:extension>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	t.Run("valid/anonymous-local", func(t *testing.T) {
		t.Parallel()
		for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
			_, errs, cerr := compileWith(t, v, anon)
			require.NoError(t, cerr, "version=%v must accept anonymous local complexType: %s", v, errs)
		}
	})

	// A top-level named complexType is legitimate.
	const global = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="fooType">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:attribute name="attrTest"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="myElement" type="fooType"/>
</xs:schema>`

	t.Run("valid/global-named", func(t *testing.T) {
		t.Parallel()
		for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
			_, errs, cerr := compileWith(t, v, global)
			require.NoError(t, cerr, "version=%v must accept top-level named complexType: %s", v, errs)
		}
	})
}
