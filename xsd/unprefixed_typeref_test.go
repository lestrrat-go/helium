package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestUnprefixedTypeRefNamespace covers the resolution of an UNPREFIXED QName
// reference (@type/@base/…) with NO in-scope default namespace. Per XML
// Namespaces such a name is in NO namespace ({}), NOT the schema's
// targetNamespace — it only adopts the targetNamespace under a chameleon-include
// conversion (a document that declares no @targetNamespace of its own).
func TestUnprefixedTypeRefNamespace(t *testing.T) {
	compile := func(t *testing.T, src string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		_, cerr := xsd.NewCompiler().Compile(t.Context(), doc)
		return cerr
	}

	// W3C msMeta addB009 (test66059.xsd): a schema WITH a targetNamespace but NO
	// default xmlns declaring an unprefixed type="CatalogData". The name is in no
	// namespace and does NOT bind {targetNamespace}CatalogData, so the reference is
	// unresolved and the schema is INVALID.
	t.Run("addB009 unprefixed type ref with own TNS, no default xmlns -> invalid", func(t *testing.T) {
		t.Parallel()
		const s = `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema"
	targetNamespace="urn:books"
	xmlns:x="urn:books"
	elementFormDefault="unqualified"
	attributeFormDefault="unqualified">
	<xsd:complexType name="CatalogData">
		<xsd:sequence>
			<xsd:element name="book" type="x:bookdata" minOccurs="0" maxOccurs="unbounded"/>
		</xsd:sequence>
	</xsd:complexType>
	<xsd:complexType name="bookdata">
		<xsd:sequence>
			<xsd:element name="author" type="xsd:string"/>
		</xsd:sequence>
	</xsd:complexType>
	<xsd:element name="catalog" type="CatalogData"/>
</xsd:schema>`
		require.ErrorIs(t, compile(t, s), xsd.ErrCompilationFailed)
	})

	// PRESERVE: a default xmlns in scope qualifies the unprefixed ref, so
	// type="CatalogData" resolves to {urn:books}CatalogData and compiles.
	t.Run("default xmlns qualifies unprefixed type ref -> valid", func(t *testing.T) {
		t.Parallel()
		const s = `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema"
	xmlns="urn:books"
	targetNamespace="urn:books">
	<xsd:complexType name="CatalogData">
		<xsd:sequence>
			<xsd:element name="author" type="xsd:string"/>
		</xsd:sequence>
	</xsd:complexType>
	<xsd:element name="catalog" type="CatalogData"/>
</xsd:schema>`
		require.NoError(t, compile(t, s))
	})

	// PRESERVE: a prefixed ref binds to its prefix's namespace and compiles.
	t.Run("prefixed type ref -> valid", func(t *testing.T) {
		t.Parallel()
		const s = `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema"
	xmlns:x="urn:books"
	targetNamespace="urn:books">
	<xsd:complexType name="CatalogData">
		<xsd:sequence>
			<xsd:element name="author" type="xsd:string"/>
		</xsd:sequence>
	</xsd:complexType>
	<xsd:element name="catalog" type="x:CatalogData"/>
</xsd:schema>`
		require.NoError(t, compile(t, s))
	})

	// PRESERVE: a NO-targetNamespace schema compiled standalone — an unprefixed ref
	// resolves to {} (no namespace) and matches its own {} type.
	t.Run("no-TNS schema unprefixed ref -> valid", func(t *testing.T) {
		t.Parallel()
		const s = `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
	<xsd:complexType name="CatalogData">
		<xsd:sequence>
			<xsd:element name="author" type="xsd:string"/>
		</xsd:sequence>
	</xsd:complexType>
	<xsd:element name="catalog" type="CatalogData"/>
</xsd:schema>`
		require.NoError(t, compile(t, s))
	})

	// PRESERVE: a chameleon include — an including schema WITH a targetNamespace
	// includes a no-@targetNamespace schema whose unprefixed refs are coerced into
	// the including namespace as a unit. The included base="Base" ref (unprefixed,
	// no default xmlns, owning document has NO own TNS) must keep the coerced
	// targetNamespace and resolve to {urn:main}Base.
	t.Run("chameleon include unprefixed ref -> valid", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{
			"cham.xsd": &fstest.MapFile{Data: []byte(`<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
	<xsd:complexType name="Base">
		<xsd:sequence>
			<xsd:element name="author" type="xsd:string"/>
		</xsd:sequence>
	</xsd:complexType>
	<xsd:complexType name="Derived">
		<xsd:complexContent>
			<xsd:extension base="Base">
				<xsd:sequence>
					<xsd:element name="extra" type="xsd:string"/>
				</xsd:sequence>
			</xsd:extension>
		</xsd:complexContent>
	</xsd:complexType>
</xsd:schema>`)},
			importMainXSD: &fstest.MapFile{Data: []byte(`<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema"
	xmlns:m="urn:main"
	targetNamespace="urn:main">
	<xsd:include schemaLocation="cham.xsd"/>
	<xsd:element name="root" type="m:Derived"/>
</xsd:schema>`)},
		}
		data, err := fsys.ReadFile(importMainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		_, cerr := xsd.NewCompiler().FS(fsys).Compile(t.Context(), doc)
		require.NoError(t, cerr)
	})
}
