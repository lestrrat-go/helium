package xsd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// A circular xs:include chain (main -> inc -> main) must compile cleanly: the
// re-include of the top-level schema has to be recognized as already-loaded
// rather than re-parsed. Before the fix includeVisited only contained documents
// pulled in via loadInclude/loadRedefine, so the back-reference to main re-parsed
// it and re-registered its declarations, producing spurious duplicate-component
// errors. CompileFile seeds includeVisited with the root's resolved key to close
// the cycle.
func TestCompileFile_CircularInclude(t *testing.T) {
	const ns = "urn:c"

	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.xsd")
	require.NoError(t, os.WriteFile(mainPath, []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="`+ns+`" targetNamespace="`+ns+`">
  <xs:include schemaLocation="inc.xsd"/>
  <xs:element name="root" type="t:LeafType"/>
</xs:schema>`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "inc.xsd"), []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="`+ns+`">
  <xs:include schemaLocation="main.xsd"/>
  <xs:complexType name="LeafType">
    <xs:sequence>
      <xs:element name="x" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`), 0o600))

	schema, err := xsd.NewCompiler().CompileFile(t.Context(), mainPath)
	require.NoError(t, err, "circular include back to the root schema must compile without duplicate-component errors")
	require.NotNil(t, schema)
}
