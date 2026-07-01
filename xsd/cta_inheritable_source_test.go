package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11CTAInheritableRestrictionSource verifies that the XSD 1.1
// inheritable restriction-consistency diagnostic is attributed to the file that
// actually declared the derived type. The derived type (changing @lang's
// inheritable) lives in an INCLUDED file, so the diagnostic's line number is only
// meaningful when paired with that file; before the fix it cited c.filename (the
// top-level schema) while reporting the included file's line.
func TestVersion11CTAInheritableRestrictionSource(t *testing.T) {
	t.Parallel()

	const (
		mainXSD = "ih_main.xsd"
		incXSD  = "ih_inc.xsd"
	)

	fsys := fstest.MapFS{
		mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="ih_inc.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)},
		// Base/@lang is inheritable; Derived restricts it to inheritable="false",
		// which is an XSD 1.1 derivation-ok-restriction error declared entirely here.
		incXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="lang" inheritable="true"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:sequence/>
        <xs:attribute name="lang" inheritable="false"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`)},
	}

	data, err := fsys.ReadFile(mainXSD)
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, err = xsd.NewCompiler().Version(xsd.Version11).Label(mainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
	requireCompileResultErr(t, err)
	require.NoError(t, collector.Close())
	_, errStr := partitionCompileErrors(collector.Errors())

	require.Contains(t, errStr, "inheritable", "expected the inheritable-consistency diagnostic; got: %q", errStr)
	require.Contains(t, errStr, incXSD+":",
		"diagnostic must be attributed to the included file; got: %q", errStr)
	require.False(t, strings.Contains(errStr, mainXSD+":"),
		"diagnostic must not cite the top-level schema label; got: %q", errStr)
}
