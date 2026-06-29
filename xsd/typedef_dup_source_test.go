package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestTypeDefDiagnosticSource verifies that complex-type compile diagnostics
// (direct duplicate attribute use, and the extension cos-ct-extends-1-1 /
// cos-all-limited 'all'-group placement checks) for a type pulled in via
// xs:include or xs:import are attributed to the DECLARING file (whose line
// number they carry), not the top-level schema label. Before the fix these
// diagnostics always cited c.filename (the main schema) while reporting the
// included/imported file's line number, producing a mismatched file:line that
// points into the wrong document.
func TestTypeDefDiagnosticSource(t *testing.T) {
	t.Parallel()

	const (
		mainXSD = "main.xsd"
		incXSD  = "inc.xsd"
		impXSD  = "imp.xsd"
	)

	// assert compiles main.xsd (which include/imports the file holding the bad
	// type) and checks the diagnostic is attributed to declFile, not mainXSD.
	assertWithCompiler := func(t *testing.T, fsys fstest.MapFS, declFile, want string, compiler xsd.Compiler) {
		t.Helper()
		data, err := fsys.ReadFile(mainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = compiler.Label(mainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		requireCompileResultErr(t, err)
		require.NoError(t, collector.Close())
		_, errStr := partitionCompileErrors(collector.Errors())

		require.Contains(t, errStr, want, "expected the diagnostic %q; got: %q", want, errStr)
		require.Contains(t, errStr, declFile+":",
			"diagnostic must be attributed to the declaring file; got: %q", errStr)
		require.False(t, strings.Contains(errStr, mainXSD+":"),
			"diagnostic must not cite the top-level schema label; got: %q", errStr)
	}
	assert := func(t *testing.T, fsys fstest.MapFS, declFile, want string) {
		t.Helper()
		assertWithCompiler(t, fsys, declFile, want, xsd.NewCompiler())
	}
	assertV11 := func(t *testing.T, fsys fstest.MapFS, declFile, want string) {
		t.Helper()
		assertWithCompiler(t, fsys, declFile, want, xsd.NewCompiler().Version(xsd.Version11))
	}

	// includeMain wraps a body that lives entirely in inc.xsd (no namespace).
	includeMain := func(body string) fstest.MapFS {
		return fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="inc.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)},
			incXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
` + body + `
</xs:schema>`)},
		}
	}

	// importMain wraps a body that lives entirely in imp.xsd (urn:t namespace).
	importMain := func(body string) fstest.MapFS {
		return fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:t">
  <xs:import namespace="urn:t" schemaLocation="imp.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)},
			impXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t">
` + body + `
</xs:schema>`)},
		}
	}

	// A complex type that directly declares the same attribute twice.
	const dupAttrBody = `  <xs:complexType name="ct">
    <xs:attribute name="x" type="xs:string"/>
    <xs:attribute name="x" type="xs:int"/>
  </xs:complexType>`

	// cos-all-limited: extension appends an 'all' group onto a non-empty base.
	const allExtBody = `  <xs:complexType name="base">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="ct">
    <xs:complexContent>
      <xs:extension base="base">
        <xs:all>
          <xs:element name="b" type="xs:string"/>
        </xs:all>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>`

	// XSD 1.1 derivation-ok-restriction: explicitly prohibiting a required base
	// attribute is rejected. The diagnostic is generated after source tracking has
	// already selected the derived type's declaring file.
	const requiredAttrProhibitionBody = `  <xs:complexType name="base">
    <xs:attribute name="req" type="xs:string" use="required"/>
  </xs:complexType>
  <xs:complexType name="ct">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:attribute name="req" use="prohibited"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>`

	t.Run("duplicate attribute", func(t *testing.T) {
		t.Parallel()
		const want = "Duplicate attribute use"
		t.Run("included file", func(t *testing.T) {
			t.Parallel()
			assert(t, includeMain(dupAttrBody), incXSD, want)
		})
		t.Run("imported file", func(t *testing.T) {
			t.Parallel()
			assert(t, importMain(dupAttrBody), impXSD, want)
		})
	})

	// derivation-ok-restriction: a complexContent restriction whose content model
	// is not a valid restriction of the base (here it adds a particle the base
	// does not allow). The diagnostic must cite the declaring file.
	const badRestrictionBody = `  <xs:complexType name="base">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="ct">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:element name="a" type="xs:string"/>
          <xs:element name="b" type="xs:string"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>`

	t.Run("invalid content-model restriction", func(t *testing.T) {
		t.Parallel()
		const want = "not a valid restriction of the content model of the base complex type definition"
		t.Run("included file", func(t *testing.T) {
			t.Parallel()
			assert(t, includeMain(badRestrictionBody), incXSD, want)
		})
		t.Run("imported file", func(t *testing.T) {
			t.Parallel()
			body := strings.ReplaceAll(badRestrictionBody, `base="base"`, `base="t:base"`)
			assert(t, importMain(body), impXSD, want)
		})
	})

	t.Run("all-group extension over non-empty base", func(t *testing.T) {
		t.Parallel()
		const want = "The 'all' model group needs to be the only child of the model group."
		t.Run("included file", func(t *testing.T) {
			t.Parallel()
			assert(t, includeMain(allExtBody), incXSD, want)
		})
		t.Run("imported file", func(t *testing.T) {
			t.Parallel()
			// imp.xsd has a targetNamespace, so reference base with its t: prefix.
			body := strings.ReplaceAll(allExtBody, `base="base"`, `base="t:base"`)
			assert(t, importMain(body), impXSD, want)
		})
	})

	t.Run("required base attribute prohibition", func(t *testing.T) {
		t.Parallel()
		const want = "A matching attribute use for the 'required' attribute use 'req'"
		t.Run("included file", func(t *testing.T) {
			t.Parallel()
			assertV11(t, includeMain(requiredAttrProhibitionBody), incXSD, want)
		})
		t.Run("imported file", func(t *testing.T) {
			t.Parallel()
			body := strings.ReplaceAll(requiredAttrProhibitionBody, `base="base"`, `base="t:base"`)
			assertV11(t, importMain(body), impXSD, want)
		})
	})
}
