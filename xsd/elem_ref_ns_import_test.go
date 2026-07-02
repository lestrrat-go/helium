package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestElementRefNonImportedNamespace verifies src-resolve §3.3.2: an
// <xs:element> @type or @ref into a namespace that the entry schema document did
// not DIRECTLY import is a schema error, even when that namespace's components
// are present in the assembly because ANOTHER document imported it transitively
// (W3C Element_w3c elemZ006/elemZ007, Schema_w3c schZ004/schZ005). A reference
// into a directly-imported namespace, the entry document's own targetNamespace,
// a self-namespace reference inside an imported sub-document, and the built-in
// xml: namespace all still compile. The rule is version-independent.
func TestElementRefNonImportedNamespace(t *testing.T) {
	t.Parallel()

	const (
		nsResolveErr = "does not resolve to a(n)"
		fileMain     = "ern_main.xsd"
		fileA        = "ern_a.xsd"
		fileB        = "ern_b.xsd"
		fileInc      = "ern_inc.xsd"
	)

	compile := func(t *testing.T, fsys fstest.MapFS) (string, error) {
		t.Helper()
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

	// bDoc declares targetNamespace "urn:b" with both a type and a global element.
	bDoc := &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:b">
  <xs:complexType name="b"><xs:sequence><xs:element name="b"/></xs:sequence></xs:complexType>
  <xs:element name="b"/>
</xs:schema>`)}

	t.Run("type into transitively-imported ns via import rejects", func(t *testing.T) {
		t.Parallel()
		// main imports urn:a only; a.xsd imports urn:b. main references {urn:b}b.
		fsys := fstest.MapFS{
			fileMain: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:m" xmlns:a="urn:a" xmlns:b="urn:b">
  <xs:import namespace="urn:a" schemaLocation="ern_a.xsd"/>
  <xs:complexType name="ct"><xs:sequence>
    <xs:element name="a" type="a:a"/>
    <xs:element name="b" type="b:b"/>
  </xs:sequence></xs:complexType>
</xs:schema>`)},
			fileA: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:a">
  <xs:import namespace="urn:b" schemaLocation="ern_b.xsd"/>
  <xs:complexType name="a"><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType>
</xs:schema>`)},
			fileB: bDoc,
		}
		errs, cerr := compile(t, fsys)
		require.Error(t, cerr, "must reject a @type into a non-directly-imported namespace")
		require.Contains(t, errs, "{urn:b}b")
		require.Contains(t, errs, nsResolveErr)
		require.NotContains(t, errs, "{urn:a}a", "a directly-imported reference must not be flagged")
	})

	t.Run("ref into transitively-imported ns via include rejects", func(t *testing.T) {
		t.Parallel()
		// main includes inc.xsd (same targetNamespace); inc.xsd imports urn:b.
		// main itself does NOT import urn:b, so ref="b:b" is illegal.
		fsys := fstest.MapFS{
			fileMain: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:m" xmlns:m="urn:m" xmlns:b="urn:b">
  <xs:include schemaLocation="ern_inc.xsd"/>
  <xs:complexType name="ct"><xs:sequence>
    <xs:element ref="m:a"/>
    <xs:element ref="b:b"/>
  </xs:sequence></xs:complexType>
</xs:schema>`)},
			fileInc: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:m">
  <xs:import namespace="urn:b" schemaLocation="ern_b.xsd"/>
  <xs:element name="a"/>
</xs:schema>`)},
			fileB: bDoc,
		}
		errs, cerr := compile(t, fsys)
		require.Error(t, cerr, "must reject a @ref into a non-directly-imported namespace")
		require.Contains(t, errs, "{urn:b}b")
		require.Contains(t, errs, nsResolveErr)
		require.NotContains(t, errs, "{urn:m}a", "an own-targetNamespace reference must not be flagged")
	})

	t.Run("directly imported ns compiles", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{
			fileMain: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:m" xmlns:b="urn:b">
  <xs:import namespace="urn:b" schemaLocation="ern_b.xsd"/>
  <xs:complexType name="ct"><xs:sequence>
    <xs:element name="b" type="b:b"/>
    <xs:element ref="b:b"/>
  </xs:sequence></xs:complexType>
</xs:schema>`)},
			fileB: bDoc,
		}
		_, cerr := compile(t, fsys)
		require.NoError(t, cerr, "a directly-imported reference must compile")
	})

	t.Run("sub-document self-namespace reference is not over-rejected", func(t *testing.T) {
		t.Parallel()
		// A reference inside an IMPORTED sub-document into that document's own
		// targetNamespace is legal; the entry-only scoping of the check must not
		// flag it even though urn:a is import-declared and the entry document
		// would not resolve a self-reference against it.
		fsys := fstest.MapFS{
			fileMain: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:m" xmlns:a="urn:a">
  <xs:import namespace="urn:a" schemaLocation="ern_a.xsd"/>
  <xs:element name="root" type="a:ct"/>
</xs:schema>`)},
			fileA: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:a" xmlns:a="urn:a">
  <xs:simpleType name="st"><xs:restriction base="xs:string"/></xs:simpleType>
  <xs:complexType name="ct"><xs:sequence>
    <xs:element name="v" type="a:st"/>
  </xs:sequence></xs:complexType>
</xs:schema>`)},
		}
		_, cerr := compile(t, fsys)
		require.NoError(t, cerr, "a self-namespace reference in an imported sub-document must compile")
	})

	t.Run("xml namespace type compiles", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{
			fileMain: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:xml="http://www.w3.org/XML/1998/namespace" targetNamespace="urn:m">
  <xs:complexType name="ct"><xs:sequence>
    <xs:element name="s" type="xs:string"/>
  </xs:sequence></xs:complexType>
</xs:schema>`)},
		}
		_, cerr := compile(t, fsys)
		require.NoError(t, cerr)
	})
}
