package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11CTARefChildRejected verifies that an <xs:element ref="..."> may
// carry only (annotation?): a LOCAL xs:alternative child is a schema error (CTA
// belongs to the referenced GLOBAL declaration, not the ref), while an
// annotation-only ref still compiles.
func TestVersion11CTARefChildRejected(t *testing.T) {
	compile := func(t *testing.T, s string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		_, cerr := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
		return cerr
	}

	t.Run("ref with local xs:alternative is a schema error", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T"><xs:sequence/><xs:attribute name="kind" type="xs:string"/></xs:complexType>
  <xs:element name="item" type="T"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="item">
          <xs:alternative test="@kind='x'" type="T"/>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.ErrorIs(t, compile(t, s), xsd.ErrCompilationFailed)
	})

	t.Run("ref with inline complexType child is a schema error", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="item" type="xs:string"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="item">
          <xs:complexType><xs:sequence/></xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.ErrorIs(t, compile(t, s), xsd.ErrCompilationFailed)
	})

	t.Run("ref with annotation-only child compiles", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="item" type="xs:string"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="item">
          <xs:annotation><xs:documentation>ok</xs:documentation></xs:annotation>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.NoError(t, compile(t, s))
	})

	// A FOREIGN-namespace element child under a ref is TOLERATED, matching helium's
	// consistent behavior across every other schema component (complexType, global
	// element, attribute, model groups all silently ignore foreign-ns element
	// children). Rejecting them only here would be inconsistent.
	t.Run("ref with foreign-namespace element child compiles", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:o="urn:o">
  <xs:element name="item" type="xs:string"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="item"><o:notAnnotation/></xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.NoError(t, compile(t, s))
	})
}

// TestVersion11CTARefChildSource verifies the ref-child violation diagnostic is
// attributed to the file that declared the offending ref: a ref with a local
// xs:alternative in an INCLUDED schema must cite the included file, not the
// top-level schema label.
func TestVersion11CTARefChildSource(t *testing.T) {
	t.Parallel()

	const (
		mainXSD = "rc_main.xsd"
		incXSD  = "rc_inc.xsd"
	)

	fsys := fstest.MapFS{
		mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="rc_inc.xsd"/>
  <xs:element name="top" type="xs:string"/>
</xs:schema>`)},
		incXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T"><xs:sequence/><xs:attribute name="kind" type="xs:string"/></xs:complexType>
  <xs:element name="item" type="T"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="item">
          <xs:alternative test="@kind='x'" type="T"/>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
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

	require.Contains(t, errStr, "Expected is (annotation?)", "expected the ref-child diagnostic; got: %q", errStr)
	require.Contains(t, errStr, incXSD+":",
		"diagnostic must be attributed to the included file; got: %q", errStr)
	require.False(t, strings.Contains(errStr, mainXSD+":"),
		"diagnostic must not cite the top-level schema label; got: %q", errStr)
}
