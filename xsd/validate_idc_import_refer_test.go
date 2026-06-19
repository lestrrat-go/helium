package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestIDCImportedKeyRefReferCitesImportedFile is the regression test for the
// source/line mismatch in deferred @refer checking across an xs:import. A
// keyref with a dangling @refer declared in an IMPORTED schema is detected by
// the (top-level) compiler's checkKeyRefRefers, which previously reported the
// PARENT compiler's filename together with the IMPORTED schema's line number —
// a mismatched source/line pair. The error must cite the imported schema's
// file (the file the line number belongs to).
func TestIDCImportedKeyRefReferCitesImportedFile(t *testing.T) {
	t.Parallel()

	// other.xsd (imported) declares a keyref whose @refer names no existing
	// key/unique. The dangling refer is only caught at the top level, after the
	// imported declarations are merged.
	fsys := fstest.MapFS{
		importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:o="urn:other" targetNamespace="urn:main">
  <xs:import namespace="urn:other" schemaLocation="other.xsd"/>
  <xs:element name="root" type="o:rootType"/>
</xs:schema>`)},
		importOtherXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:other" elementFormDefault="qualified">
  <xs:complexType name="rootType">
    <xs:sequence>
      <xs:element name="ref" maxOccurs="unbounded">
        <xs:complexType>
          <xs:attribute name="r" type="xs:string"/>
        </xs:complexType>
      </xs:element>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="holder" type="rootType">
    <xs:keyref name="danglingRef" refer="nonexistentKey">
      <xs:selector xpath="ref"/>
      <xs:field xpath="@r"/>
    </xs:keyref>
  </xs:element>
</xs:schema>`)},
	}

	data, err := fsys.ReadFile(importMainXSD)
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, err = xsd.NewCompiler().Label(importMainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
	requireCompileResultErr(t, err)

	var b strings.Builder
	for _, e := range collector.Errors() {
		b.WriteString(e.Error())
	}
	got := b.String()

	require.Contains(t, got, "nonexistentKey", "the dangling refer must be reported; got: %q", got)
	require.Contains(t, got, importOtherXSD,
		"the deferred @refer error must cite the IMPORTED schema (where its line lives), not the importing schema; got: %q", got)
	require.NotContains(t, got, importMainXSD+":",
		"the error must not cite the importing schema's filename with the imported line; got: %q", got)
}
