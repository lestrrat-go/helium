package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestRedefineGroupRestriction verifies src-redefine.6.2 (§4.2.3): a
// redefining <group> with no self-reference must be a valid restriction of the
// original group. The default XSD 1.0 path uses the intensional Particle Valid
// (Restriction) rules; XSD 1.1 can accept language-subset cases such as a
// reordered xs:all.
func TestRedefineGroupRestriction(t *testing.T) {
	t.Parallel()

	compileErrors := func(t *testing.T, mainSchema, baseSchema string, configure ...func(xsd.Compiler) xsd.Compiler) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(mainSchema))
		require.NoError(t, err)
		fsys := fstest.MapFS{"base.xsd": &fstest.MapFile{Data: []byte(baseSchema)}}
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		compiler := xsd.NewCompiler().Label("main.xsd").FS(fsys).ErrorHandler(collector)
		for _, fn := range configure {
			compiler = fn(compiler)
		}
		_, err = compiler.Compile(t.Context(), doc)
		if err != nil {
			requireCompileResultErr(t, err)
		}
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

	t.Run("rejects dropping a required base member", func(t *testing.T) {
		t.Parallel()
		base := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:all><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:all></xs:group>
</xs:schema>`
		main := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="base.xsd">
    <xs:group name="g"><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:group>
  </xs:redefine>
  <xs:element name="doc"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element>
</xs:schema>`
		require.Contains(t, compileErrors(t, main, base), wantMsg)
	})

	t.Run("rejects dropping required base member hidden behind group ref", func(t *testing.T) {
		t.Parallel()
		base := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="h"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>
  <xs:group name="g"><xs:sequence><xs:group ref="h"/></xs:sequence></xs:group>
</xs:schema>`
		main := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="base.xsd">
    <xs:group name="g"><xs:sequence/></xs:group>
  </xs:redefine>
  <xs:element name="doc"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element>
</xs:schema>`
		require.Contains(t, compileErrors(t, main, base), wantMsg)
	})

	t.Run("referenced original group uses phase a even after earlier redefine", func(t *testing.T) {
		t.Parallel()
		base := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="h"><xs:choice><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:choice></xs:group>
  <xs:group name="g"><xs:sequence><xs:group ref="h"/></xs:sequence></xs:group>
</xs:schema>`
		main := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="base.xsd">
    <xs:group name="h"><xs:choice><xs:element name="a" type="xs:string"/></xs:choice></xs:group>
    <xs:group name="g"><xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence></xs:group>
  </xs:redefine>
  <xs:element name="doc"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element>
</xs:schema>`
		require.Empty(t, compileErrors(t, main, base))
	})

	t.Run("rejects adding an element the original omits", func(t *testing.T) {
		t.Parallel()
		base := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:group>
</xs:schema>`
		main := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="base.xsd">
    <xs:group name="g"><xs:all><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:all></xs:group>
  </xs:redefine>
  <xs:element name="doc"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element>
</xs:schema>`
		require.Contains(t, compileErrors(t, main, base), wantMsg)
	})

	t.Run("rejects reordered sequence", func(t *testing.T) {
		t.Parallel()
		base := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:sequence><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:sequence></xs:group>
</xs:schema>`
		main := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="base.xsd">
    <xs:group name="g"><xs:sequence><xs:element name="b" type="xs:string"/><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>
  </xs:redefine>
  <xs:element name="doc"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element>
</xs:schema>`
		require.Contains(t, compileErrors(t, main, base), wantMsg)
	})

	t.Run("reordered all is invalid in xsd 1.0 and valid in xsd 1.1", func(t *testing.T) {
		t.Parallel()
		base := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:all><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:all></xs:group>
</xs:schema>`
		main := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="base.xsd">
    <xs:group name="g"><xs:all><xs:element name="b" type="xs:string"/><xs:element name="a" type="xs:string"/></xs:all></xs:group>
  </xs:redefine>
  <xs:element name="doc"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element>
</xs:schema>`
		require.Contains(t, compileErrors(t, main, base), wantMsg)
		require.Empty(t, compileErrors(t, main, base, func(c xsd.Compiler) xsd.Compiler {
			return c.Version(xsd.Version11)
		}))
	})
}
