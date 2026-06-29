package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestOpenContent_ImportDefaultOpenContentVCExcluded covers the gauntlet finding
// that an imported schema's <xs:defaultOpenContent> must be read AFTER the
// conditional-inclusion pre-pass, exactly like xs:include/xs:redefine. A
// vc:minVersion="9.9" default open content is excluded under 1.1, so the imported
// complex type stays CLOSED and an extra child must be rejected. The buggy order
// read the default BEFORE conditional inclusion and wrongly applied it.
func TestOpenContent_ImportDefaultOpenContentVCExcluded(t *testing.T) {
	t.Parallel()
	const impXSD = "imp.xsd"
	fsys := fstest.MapFS{
		importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:imp="urn:imp" targetNamespace="urn:m">
  <xs:import namespace="urn:imp" schemaLocation="imp.xsd"/>
  <xs:element name="doc" type="imp:T"/>
</xs:schema>`)},
		impXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning" targetNamespace="urn:imp">
  <xs:defaultOpenContent vc:minVersion="9.9" mode="interleave">
    <xs:any namespace="##any" processContents="skip"/>
  </xs:defaultOpenContent>
  <xs:complexType name="T"><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType>
</xs:schema>`)},
	}
	data, err := fsys.ReadFile(importMainXSD)
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)
	schema, cerr := xsd.NewCompiler().Version(xsd.Version11).Label(importMainXSD).FS(fsys).Compile(t.Context(), doc)
	require.NoError(t, cerr)
	require.NotNil(t, schema)

	idoc, perr := helium.NewParser().Parse(t.Context(),
		[]byte(`<m:doc xmlns:m="urn:m"><a/><extra/></m:doc>`))
	require.NoError(t, perr)
	require.Error(t, xsd.NewValidator(schema).Validate(t.Context(), idoc),
		"the vc-excluded imported defaultOpenContent must NOT apply; the extra child must be rejected")

	// Sanity: the closed type still accepts its declared content.
	vdoc, perr := helium.NewParser().Parse(t.Context(),
		[]byte(`<m:doc xmlns:m="urn:m"><a/></m:doc>`))
	require.NoError(t, perr)
	require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), vdoc))
}
