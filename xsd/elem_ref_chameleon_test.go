package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestElementRefChameleonFallback verifies that the no-targetNamespace ({})
// fallback for an element @ref fires ONLY for a chameleon-eligible ref — an
// unprefixed ref with no in-scope default namespace, whose lexical name
// resolves to the schema targetNamespace but whose global element lives in the
// absent namespace of an imported no-targetNamespace schema. A prefixed ref, or
// an unprefixed ref bound by an in-scope xmlns="..." default namespace, is NOT
// eligible: the fallback must not mask a genuine src-resolve failure by
// silently resolving to {}local.
func TestElementRefChameleonFallback(t *testing.T) {
	t.Parallel()

	const (
		nsResolveErr = "does not resolve to a(n)"
		fileMain     = "erc_main.xsd"
		fileNoNS     = "erc_nons.xsd"
	)

	// noNSDoc has NO targetNamespace and declares a global element "foo".
	noNSDoc := &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="foo" type="xs:string"/>
</xs:schema>`)}

	compile := func(t *testing.T, mainDoc string) (string, error) {
		t.Helper()
		fsys := fstest.MapFS{
			fileMain: &fstest.MapFile{Data: []byte(mainDoc)},
			fileNoNS: noNSDoc,
		}
		data, err := fsys.ReadFile(fileMain)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, cerr := xsd.NewCompiler().Label(fileMain).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		require.NoError(t, collector.Close())
		var sb strings.Builder
		for _, e := range collector.Errors() {
			sb.WriteString(e.Error())
			sb.WriteString("\n")
		}
		return sb.String(), cerr
	}

	t.Run("unprefixed ref with no default ns resolves via empty-namespace fallback", func(t *testing.T) {
		t.Parallel()
		// The chameleon case: ref="foo" resolves lexically to {urn:main}foo, but
		// the global lives in {} (the imported no-targetNamespace schema).
		_, cerr := compile(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:main">
  <xs:import schemaLocation="erc_nons.xsd"/>
  <xs:complexType name="ct"><xs:sequence>
    <xs:element ref="foo"/>
  </xs:sequence></xs:complexType>
</xs:schema>`)
		require.NoError(t, cerr, "a chameleon-eligible unprefixed ref must resolve to the {} global")
	})

	t.Run("prefixed ref does not fall back to empty namespace", func(t *testing.T) {
		t.Parallel()
		// ref="o:foo" resolves to {urn:main}foo (o bound to the target namespace);
		// no such global exists — only {}foo. The prefixed ref is NOT eligible, so
		// it must report unresolved instead of silently resolving to {}foo.
		errs, cerr := compile(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:main" xmlns:o="urn:main">
  <xs:import schemaLocation="erc_nons.xsd"/>
  <xs:complexType name="ct"><xs:sequence>
    <xs:element ref="o:foo"/>
  </xs:sequence></xs:complexType>
</xs:schema>`)
		require.Error(t, cerr, "a prefixed ref must not fall back to the {} global")
		require.Contains(t, errs, "{urn:main}foo")
		require.Contains(t, errs, nsResolveErr)
	})

	t.Run("unprefixed ref bound by default namespace does not fall back", func(t *testing.T) {
		t.Parallel()
		// xmlns="urn:main" qualifies the unprefixed ref to {urn:main}foo; no such
		// global exists — only {}foo. A default-namespace-bound ref is NOT eligible.
		errs, cerr := compile(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:main" xmlns="urn:main">
  <xs:import schemaLocation="erc_nons.xsd"/>
  <xs:complexType name="ct"><xs:sequence>
    <xs:element ref="foo"/>
  </xs:sequence></xs:complexType>
</xs:schema>`)
		require.Error(t, cerr, "a default-ns-bound ref must not fall back to the {} global")
		require.Contains(t, errs, "{urn:main}foo")
		require.Contains(t, errs, nsResolveErr)
	})
}
