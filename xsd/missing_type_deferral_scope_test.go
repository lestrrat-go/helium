package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestMissingTypeDeferralScope guards the boundary of the §5.3 unused-missing-
// component deferral. A missing NO-namespace type on an UNUSED declaration is
// deferred until validation (Saxon Missing missing001/003/006). But the deferral
// must NOT swallow the genuine defects that only coincidentally manifested as an
// unresolved type reference: a wrong-symbol-space @type, a misplaced top-level
// compositor, a referenced global element, a failed redefine target, or an
// xs:override replacement child. Those stay compile-time schema errors.
func TestMissingTypeDeferralScope(t *testing.T) {
	t.Parallel()

	const wantMsg = "does not resolve to a(n) type definition"

	compileErrors := func(t *testing.T, schemaXML string) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
		requireCompileResultErr(t, err)
		_, errors := partitionCompileErrors(collector.Errors())
		return errors
	}
	compileOK := func(t *testing.T, schemaXML string) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
		require.NoError(t, err)
		require.NoError(t, collector.Close())
		_, errors := partitionCompileErrors(collector.Errors())
		require.Empty(t, errors)
	}

	// A TRUE unused missing type stays deferred and compiles (must not regress).
	t.Run("unused missing element type still compiles", func(t *testing.T) {
		t.Parallel()
		compileOK(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="good" type="xs:integer"/>
  <xs:element name="bad" type="absent"/>
</xs:schema>`)
	})

	t.Run("unused missing element type with missing subst head still compiles", func(t *testing.T) {
		t.Parallel()
		compileOK(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="good" type="xs:integer"/>
  <xs:element name="bad" type="absent" substitutionGroup="missing"/>
</xs:schema>`)
	})

	t.Run("unused named list with missing item type still compiles", func(t *testing.T) {
		t.Parallel()
		compileOK(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="bad" type="list"/>
  <xs:simpleType name="list"><xs:list itemType="absent"/></xs:simpleType>
</xs:schema>`)
	})

	// elemM002: element @type names an existing global ATTRIBUTE (wrong symbol space).
	t.Run("element type naming an attribute is rejected", func(t *testing.T) {
		t.Parallel()
		require.Contains(t, compileErrors(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="myElem" type="foo"/>
  <xs:attribute name="foo"/>
</xs:schema>`), wantMsg)
	})

	// element @type naming an existing model group (wrong symbol space).
	t.Run("element type naming a group is rejected", func(t *testing.T) {
		t.Parallel()
		require.Contains(t, compileErrors(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="myElem" type="g"/>
  <xs:group name="g"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>
</xs:schema>`), wantMsg)
	})

	// mgP059-mgP062: a misplaced compositor/derivation element at schema top level.
	t.Run("misplaced top-level compositor is rejected", func(t *testing.T) {
		t.Parallel()
		got := compileErrors(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc" type="foo"/>
  <xs:sequence>
    <xs:element name="a" type="xs:string"/>
  </xs:sequence>
</xs:schema>`)
		require.Contains(t, got, "is not allowed as a child of the schema element")
	})

	t.Run("misplaced top-level extension is rejected", func(t *testing.T) {
		t.Parallel()
		got := compileErrors(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc" type="foo"/>
  <xs:extension>
    <xs:element name="a" type="xs:string"/>
  </xs:extension>
</xs:schema>`)
		require.Contains(t, got, "is not allowed as a child of the schema element")
	})

	// idB005: a GLOBAL element with a missing type that IS referenced by a
	// content-model @ref is genuinely needed.
	t.Run("referenced global element with missing type is rejected", func(t *testing.T) {
		t.Parallel()
		require.Contains(t, compileErrors(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType><xs:sequence>
      <xs:element ref="keyElement"/>
    </xs:sequence></xs:complexType>
  </xs:element>
  <xs:element name="keyElement" type="keyinfo"/>
</xs:schema>`), wantMsg)
	})

	// addB030: a redefine whose target document FAILS to load leaves a referenced
	// redefined type genuinely missing.
	t.Run("missing type from a failed redefine is rejected", func(t *testing.T) {
		t.Parallel()
		const mainXSD = "main.xsd"
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="t" type="tabletype"/>
  <xs:redefine schemaLocation="does-not-exist.xsd">
    <xs:complexType name="tabletype">
      <xs:complexContent>
        <xs:extension base="tabletype">
          <xs:sequence><xs:element name="c" type="xs:string"/></xs:sequence>
        </xs:extension>
      </xs:complexContent>
    </xs:complexType>
  </xs:redefine>
</xs:schema>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		doc.SetURL(mainXSD)
		fsys := fstest.MapFS{mainXSD: {Data: []byte(schemaXML)}}
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, cerr := xsd.NewCompiler().Label(mainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		requireCompileResultErr(t, cerr)
		_, errors := partitionCompileErrors(collector.Errors())
		require.Contains(t, errors, wantMsg)
	})
}

// TestOverrideChildMissingTypeRejected covers over024: an xs:override replacement
// element declaration whose @type is a missing component is a composition error,
// not a deferrable §5.3 unused-missing-component. XSD 1.1 (xs:override is 1.1-only).
func TestOverrideChildMissingTypeRejected(t *testing.T) {
	t.Parallel()

	const mainFile = "main.xsd"
	targetXSD := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType><xs:sequence>
    <xs:element name="para" type="xs:string" maxOccurs="unbounded"/>
  </xs:sequence></xs:complexType></xs:element>
</xs:schema>`
	mainXSD := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:override schemaLocation="target.xsd">
    <xs:element name="doc" type="zuludate"/>
  </xs:override>
  <xs:simpleType name="zuluDate">
    <xs:restriction base="xs:date"><xs:pattern value=".*Z"/></xs:restriction>
  </xs:simpleType>
</xs:schema>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(mainXSD))
	require.NoError(t, err)
	doc.SetURL(mainFile)
	fsys := fstest.MapFS{
		mainFile:     {Data: []byte(mainXSD)},
		"target.xsd": {Data: []byte(targetXSD)},
	}
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, cerr := xsd.NewCompiler().Label(mainFile).Version(xsd.Version11).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
	requireCompileResultErr(t, cerr)
	_, errors := partitionCompileErrors(collector.Errors())
	require.Contains(t, errors, "does not resolve to a(n) type definition")
}
