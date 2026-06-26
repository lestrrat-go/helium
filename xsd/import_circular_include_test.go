package xsd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// An imported schema that circularly includes back to its own root must compile
// cleanly: main -> import imp.xsd, where imp.xsd -> include inc.xsd -> include
// imp.xsd. Before the fix loadImport initialized the imported sub-compiler with
// an empty includeVisited set, so the back-reference to imp.xsd was re-parsed and
// its components re-registered, producing spurious duplicate-component errors.
// loadImport now seeds the imported sub-compiler's includeVisited with the
// imported schema's resolved key, mirroring CompileFile's seeding of the root.
func TestCompileFile_ImportCircularInclude(t *testing.T) {
	const (
		mainNS = "urn:m"
		impNS  = "urn:i"
	)

	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.xsd")
	require.NoError(t, os.WriteFile(mainPath, []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:i="`+impNS+`" targetNamespace="`+mainNS+`">
  <xs:import namespace="`+impNS+`" schemaLocation="imp.xsd"/>
  <xs:element name="root" type="i:LeafType"/>
</xs:schema>`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "imp.xsd"), []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="`+impNS+`">
  <xs:include schemaLocation="inc.xsd"/>
  <xs:complexType name="LeafType">
    <xs:sequence>
      <xs:element name="x" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "inc.xsd"), []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="`+impNS+`">
  <xs:include schemaLocation="imp.xsd"/>
  <xs:complexType name="HelperType">
    <xs:sequence>
      <xs:element name="y" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`), 0o600))

	schema, err := xsd.NewCompiler().CompileFile(t.Context(), mainPath)
	require.NoError(t, err, "an imported schema that circularly includes back to its own root must compile without duplicate-component errors")
	require.NotNil(t, schema)
}
