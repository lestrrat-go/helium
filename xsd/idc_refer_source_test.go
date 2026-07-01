package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestKeyRefUnboundPrefixSource verifies that the "namespace prefix is not bound"
// diagnostic for an xs:keyref/@refer with an unbound prefix is attributed to the
// DECLARING file (whose line number it carries), not the top-level schema label.
// The keyref lives entirely in the included file, so the diagnostic's line number
// is meaningful only when paired with that file. Before the fix the diagnostic
// cited c.filename (the main schema) while reporting the included file's line,
// producing a mismatched file:line.
func TestKeyRefUnboundPrefixSource(t *testing.T) {
	t.Parallel()

	const (
		mainXSD = "kr_main.xsd"
		incXSD  = "kr_inc.xsd"
	)

	const unbound = "whose namespace prefix"

	fsys := fstest.MapFS{
		mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="kr_inc.xsd"/>
  <xs:element name="doc" type="xs:string"/>
</xs:schema>`)},
		incXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="id" type="xs:string"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="k"><xs:selector xpath="item"/><xs:field xpath="@id"/></xs:key>
    <xs:keyref name="kr" refer="p:k"><xs:selector xpath="item"/><xs:field xpath="@id"/></xs:keyref>
  </xs:element>
</xs:schema>`)},
	}

	data, err := fsys.ReadFile(mainXSD)
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, err = xsd.NewCompiler().Label(mainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
	requireCompileResultErr(t, err)
	require.NoError(t, collector.Close())
	_, errStr := partitionCompileErrors(collector.Errors())

	require.Contains(t, errStr, unbound, "expected the unbound-prefix diagnostic")
	require.Contains(t, errStr, incXSD+":",
		"diagnostic must be attributed to the declaring (included) file; got: %q", errStr)
	require.False(t, strings.Contains(errStr, mainXSD+":"),
		"diagnostic must not cite the top-level schema label; got: %q", errStr)
}
