package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestRedefineGroupRestriction verifies the provably-sound core of
// src-redefine.6.2 (§4.2.3): a redefining <group> with no self-reference must be
// a valid restriction of the original group. A restriction can never REQUIRE an
// element the original group EXPLICITLY forbids (declares with maxOccurs=0). This
// mirrors W3C ModelGroups mgO013, which the corpus marks invalid. A redefinition
// that merely picks a subset of the original (choice{e1} restricting all{e1,e2?})
// stays valid. Version-independent.
func TestRedefineGroupRestriction(t *testing.T) {
	t.Parallel()

	compileErrors := func(t *testing.T, mainSchema, baseSchema string) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(mainSchema))
		require.NoError(t, err)
		fsys := fstest.MapFS{"base.xsd": &fstest.MapFile{Data: []byte(baseSchema)}}
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label("main.xsd").FS(fsys).ErrorHandler(collector).Compile(t.Context(), doc)
		requireCompileResultErr(t, err)
		_, errors := partitionCompileErrors(collector.Errors())
		return errors
	}

	const wantMsg = "src-redefine.6.2: The redefinition of group"

	t.Run("rejects requiring a forbidden element", func(t *testing.T) {
		t.Parallel()
		// base forbids e1 (maxOccurs=0); the redefinition requires it.
		base := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:all><xs:element name="e1" minOccurs="0" maxOccurs="0"/></xs:all></xs:group>
</xs:schema>`
		main := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="base.xsd">
    <xs:group name="g"><xs:all><xs:element name="e1"/></xs:all></xs:group>
  </xs:redefine>
  <xs:element name="doc"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element>
</xs:schema>`
		require.Contains(t, compileErrors(t, main, base), wantMsg)
	})

	t.Run("accepts a subset redefinition", func(t *testing.T) {
		t.Parallel()
		base := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:all><xs:element name="e1"/><xs:element name="e2" minOccurs="0"/></xs:all></xs:group>
</xs:schema>`
		main := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="base.xsd">
    <xs:group name="g"><xs:choice><xs:element name="e1"/></xs:choice></xs:group>
  </xs:redefine>
  <xs:element name="doc"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element>
</xs:schema>`
		require.Empty(t, compileErrors(t, main, base))
	})

	t.Run("ignores an element the original omits", func(t *testing.T) {
		t.Parallel()
		// The redefinition introduces a new required element the original never
		// declares. The conservative rule stays silent (only explicitly-forbidden
		// declared names are rejected), so this compiles.
		base := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:group>
</xs:schema>`
		main := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="base.xsd">
    <xs:group name="g"><xs:all><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:all></xs:group>
  </xs:redefine>
  <xs:element name="doc"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element>
</xs:schema>`
		require.Empty(t, compileErrors(t, main, base))
	})
}
