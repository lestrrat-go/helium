package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestOverride_NestedIncludeDefaultOpenContent covers the gauntlet finding that an
// override replacement matching a component declared in a NESTED INCLUDED document
// of the target must be registered under THAT included document's per-document
// <xs:defaultOpenContent> (§4.2.5 — a replacement is governed by the document where
// the matched component was declared), not the direct override target's.
func TestOverride_NestedIncludeDefaultOpenContent(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"ovmain.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:m="urn:m" targetNamespace="urn:m">
  <xs:override schemaLocation="ovtarget.xsd">
    <xs:complexType name="T"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
  </xs:override>
  <xs:element name="doc" type="m:T"/>
</xs:schema>`)},
		"ovtarget.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:m">
  <xs:include schemaLocation="ovinc.xsd"/>
  <xs:defaultOpenContent mode="interleave">
    <xs:any namespace="urn:target" processContents="skip"/>
  </xs:defaultOpenContent>
</xs:schema>`)},
		"ovinc.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:m">
  <xs:defaultOpenContent mode="interleave">
    <xs:any namespace="urn:inc" processContents="skip"/>
  </xs:defaultOpenContent>
  <xs:complexType name="T"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
</xs:schema>`)},
	}
	data, err := fsys.ReadFile("ovmain.xsd")
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)
	schema, cerr := xsd.NewCompiler().Version(xsd.Version11).Label("ovmain.xsd").FS(fsys).Compile(t.Context(), doc)
	require.NoError(t, cerr)
	require.NotNil(t, schema)

	validate := func(instance string) error {
		idoc, perr := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, perr)
		return xsd.NewValidator(schema).Validate(t.Context(), idoc)
	}

	// The replacement T was declared in inc.xsd, so its default open content is
	// inc.xsd's (urn:inc), NOT the direct target's (urn:target).
	require.NoError(t, validate(`<m:doc xmlns:m="urn:m"><a>x</a><extra xmlns="urn:inc"/></m:doc>`),
		"the replacement must use the included document's defaultOpenContent (urn:inc)")
	require.Error(t, validate(`<m:doc xmlns:m="urn:m"><a>x</a><extra xmlns="urn:target"/></m:doc>`),
		"the direct target's defaultOpenContent (urn:target) must NOT govern the replacement")
}
