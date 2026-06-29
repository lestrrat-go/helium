package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// fileMain and fileA are the recurring schema-document names used across the
// override tests; named constants keep the repeated map keys lint-clean.
const (
	fileMain = "main.xsd"
	fileA    = "a.xsd"
)

// compileOverride compiles main.xsd from fsys under XSD 1.1 and returns the
// schema and compile error.
func compileOverride(t *testing.T, fsys fstest.MapFS) (*xsd.Schema, error) {
	t.Helper()
	data, err := fsys.ReadFile(fileMain)
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)
	return xsd.NewCompiler().Version(xsd.Version11).FS(fsys).Compile(t.Context(), doc)
}

func overrideValidate(t *testing.T, schema *xsd.Schema, instance string) error {
	t.Helper()
	idoc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
	require.NoError(t, err)
	return xsd.NewValidator(schema).Validate(t.Context(), idoc)
}

// TestOverride_SchemaValidity covers xs:override schema-level acceptance/rejection.
func TestOverride_SchemaValidity(t *testing.T) {
	const xs = `xmlns:xs="http://www.w3.org/2001/XMLSchema"`

	tests := []struct {
		name    string
		files   map[string]string
		wantErr bool
	}{
		{
			// over001: override an element declaration (wholesale replacement).
			name: "element replacement",
			files: map[string]string{
				fileMain: `<xs:schema ` + xs + `>
  <xs:override schemaLocation="a.xsd"><xs:element name="doc" type="xs:date"/></xs:override>
</xs:schema>`,
				fileA: `<xs:schema ` + xs + `>
  <xs:element name="doc"><xs:complexType><xs:sequence><xs:element name="para"/></xs:sequence></xs:complexType></xs:element>
</xs:schema>`,
			},
		},
		{
			// over010: override a simpleType. Kept doc references it by name.
			name: "simpleType replacement",
			files: map[string]string{
				fileMain: `<xs:schema ` + xs + `>
  <xs:override schemaLocation="a.xsd">
    <xs:simpleType name="t"><xs:restriction base="xs:integer"><xs:maxInclusive value="16"/></xs:restriction></xs:simpleType>
  </xs:override>
  <xs:element name="root" type="t"/>
</xs:schema>`,
				fileA: `<xs:schema ` + xs + `>
  <xs:simpleType name="t"><xs:restriction base="xs:integer"/></xs:simpleType>
</xs:schema>`,
			},
		},
		{
			// over026: an override child that overrides nothing is DROPPED, so a
			// reference to it dangles and the schema is invalid.
			name:    "unmatched child dropped leaves dangling ref",
			wantErr: true,
			files: map[string]string{
				fileMain: `<xs:schema ` + xs + `>
  <xs:override schemaLocation="a.xsd">
    <xs:element name="doc" type="zonedDate"/>
    <xs:simpleType name="zonedDate"><xs:restriction base="xs:date"/></xs:simpleType>
  </xs:override>
</xs:schema>`,
				fileA: `<xs:schema ` + xs + `>
  <xs:element name="doc" type="xs:string"/>
</xs:schema>`,
			},
		},
		{
			// over022: a document may not be overridden more than once.
			name:    "double override of same document",
			wantErr: true,
			files: map[string]string{
				fileMain: `<xs:schema ` + xs + `>
  <xs:override schemaLocation="a.xsd"><xs:element name="doc" type="xs:date"/></xs:override>
  <xs:override schemaLocation="a.xsd"><xs:element name="doc" type="xs:time"/></xs:override>
</xs:schema>`,
				fileA: `<xs:schema ` + xs + `>
  <xs:element name="doc" type="xs:string"/>
</xs:schema>`,
			},
		},
		{
			// over021: two children overriding the same component.
			name:    "duplicate override child",
			wantErr: true,
			files: map[string]string{
				fileMain: `<xs:schema ` + xs + `>
  <xs:override schemaLocation="a.xsd">
    <xs:element name="doc" type="xs:date"/>
    <xs:element name="doc" type="xs:time"/>
  </xs:override>
</xs:schema>`,
				fileA: `<xs:schema ` + xs + `>
  <xs:element name="doc" type="xs:string"/>
</xs:schema>`,
			},
		},
		{
			// over016: overridden document in a different namespace is rejected.
			name:    "namespace mismatch",
			wantErr: true,
			files: map[string]string{
				fileMain: `<xs:schema ` + xs + `>
  <xs:override schemaLocation="a.xsd"><xs:element name="doc" type="xs:date"/></xs:override>
</xs:schema>`,
				fileA: `<xs:schema ` + xs + ` targetNamespace="urn:other">
  <xs:element name="doc" type="xs:string"/>
</xs:schema>`,
			},
		},
		{
			// over023: a permissible circular override (the back-edge has no
			// effective override children) must terminate and compile.
			name: "permissible circular override",
			files: map[string]string{
				fileMain: `<xs:schema ` + xs + `>
  <xs:override schemaLocation="a.xsd"><xs:element name="doc" type="xs:date"/></xs:override>
</xs:schema>`,
				fileA: `<xs:schema ` + xs + `>
  <xs:override schemaLocation="main.xsd"/>
  <xs:element name="doc"><xs:complexType><xs:sequence><xs:element name="para"/></xs:sequence></xs:complexType></xs:element>
</xs:schema>`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := fstest.MapFS{}
			for name, body := range tt.files {
				fsys[name] = &fstest.MapFile{Data: []byte(body)}
			}
			_, err := compileOverride(t, fsys)
			if tt.wantErr {
				require.Error(t, err, "schema must be rejected")
				return
			}
			require.NoError(t, err, "schema must compile")
		})
	}
}

// TestOverride_ElementInstance proves an overridden element declaration replaces
// the original content model: the post-override type validates, the pre-override
// content model does not.
func TestOverride_ElementInstance(t *testing.T) {
	const xs = `xmlns:xs="http://www.w3.org/2001/XMLSchema"`
	fsys := fstest.MapFS{
		fileMain: &fstest.MapFile{Data: []byte(`<xs:schema ` + xs + ` elementFormDefault="qualified">
  <xs:override schemaLocation="a.xsd"><xs:element name="doc" type="xs:date"/></xs:override>
</xs:schema>`)},
		fileA: &fstest.MapFile{Data: []byte(`<xs:schema ` + xs + ` elementFormDefault="qualified">
  <xs:element name="doc"><xs:complexType><xs:sequence><xs:element name="para" maxOccurs="unbounded"/></xs:sequence></xs:complexType></xs:element>
</xs:schema>`)},
	}
	schema, err := compileOverride(t, fsys)
	require.NoError(t, err)

	require.NoError(t, overrideValidate(t, schema, `<doc>2010-01-01</doc>`),
		"date value satisfies the overriding declaration")
	require.Error(t, overrideValidate(t, schema, `<doc><para>x</para></doc>`),
		"the pre-override content model must be rejected")
}

// TestOverride_KeptComponentAndRef proves a non-overridden component is kept and
// a reference in the overridden document binds to the overriding component.
func TestOverride_KeptComponentAndRef(t *testing.T) {
	const xs = `xmlns:xs="http://www.w3.org/2001/XMLSchema"`
	fsys := fstest.MapFS{
		fileMain: &fstest.MapFile{Data: []byte(`<xs:schema ` + xs + ` elementFormDefault="qualified">
  <xs:override schemaLocation="a.xsd"><xs:element name="para" type="xs:dateTime"/></xs:override>
</xs:schema>`)},
		fileA: &fstest.MapFile{Data: []byte(`<xs:schema ` + xs + ` elementFormDefault="qualified">
  <xs:element name="doc"><xs:complexType><xs:sequence><xs:element ref="para" maxOccurs="unbounded"/></xs:sequence></xs:complexType></xs:element>
  <xs:element name="para" type="xs:date"/>
</xs:schema>`)},
	}
	schema, err := compileOverride(t, fsys)
	require.NoError(t, err)

	// doc was kept; para now requires a dateTime value.
	require.NoError(t, overrideValidate(t, schema, `<doc><para>2010-01-01T00:00:00</para></doc>`),
		"para satisfies the overriding dateTime declaration")
	require.Error(t, overrideValidate(t, schema, `<doc><para>2010-01-01</para></doc>`),
		"a plain date must be rejected by the overriding dateTime para")
}

// TestOverride_Transitive covers a double override (over009): an override of a
// schema document that itself contains an xs:override. The OUTER override wins.
func TestOverride_Transitive(t *testing.T) {
	const xs = `xmlns:xs="http://www.w3.org/2001/XMLSchema"`
	fsys := fstest.MapFS{
		// main overrides mid; mid overrides leaf. para must end up as the OUTER
		// (main) override type.
		fileMain: &fstest.MapFile{Data: []byte(`<xs:schema ` + xs + ` elementFormDefault="qualified">
  <xs:override schemaLocation="mid.xsd"><xs:element name="para" type="zoned"/></xs:override>
  <xs:simpleType name="zoned"><xs:restriction base="xs:string"><xs:pattern value="[A-Z]+"/></xs:restriction></xs:simpleType>
</xs:schema>`)},
		"mid.xsd": &fstest.MapFile{Data: []byte(`<xs:schema ` + xs + ` elementFormDefault="qualified">
  <xs:override schemaLocation="leaf.xsd"><xs:element name="para" type="zoneless"/></xs:override>
  <xs:simpleType name="zoneless"><xs:restriction base="xs:string"><xs:pattern value="[0-9]+"/></xs:restriction></xs:simpleType>
</xs:schema>`)},
		"leaf.xsd": &fstest.MapFile{Data: []byte(`<xs:schema ` + xs + ` elementFormDefault="qualified">
  <xs:element name="doc"><xs:complexType><xs:sequence><xs:element ref="para" maxOccurs="unbounded"/></xs:sequence></xs:complexType></xs:element>
  <xs:element name="para" type="xs:string"/>
</xs:schema>`)},
	}
	schema, err := compileOverride(t, fsys)
	require.NoError(t, err)

	require.NoError(t, overrideValidate(t, schema, `<doc><para>ABC</para></doc>`),
		"para must use the OUTER override type (uppercase pattern)")
	require.Error(t, overrideValidate(t, schema, `<doc><para>123</para></doc>`),
		"the inner override type (digits) must not win")
}

// TestOverride_IndirectChameleon covers over020: overriding a no-namespace
// document that itself xs:includes another no-namespace document. The override
// child and the included chameleon components all adopt the overriding namespace.
func TestOverride_IndirectChameleon(t *testing.T) {
	const xs = `xmlns:xs="http://www.w3.org/2001/XMLSchema"`
	const ns = "urn:o"
	fsys := fstest.MapFS{
		fileMain: &fstest.MapFile{Data: []byte(`<xs:schema ` + xs + ` targetNamespace="` + ns + `" elementFormDefault="qualified">
  <xs:override schemaLocation="a.xsd"><xs:element name="doc" type="xs:date"/></xs:override>
</xs:schema>`)},
		fileA: &fstest.MapFile{Data: []byte(`<xs:schema ` + xs + ` elementFormDefault="qualified">
  <xs:include schemaLocation="b.xsd"/>
</xs:schema>`)},
		"b.xsd": &fstest.MapFile{Data: []byte(`<xs:schema ` + xs + ` elementFormDefault="qualified">
  <xs:element name="doc"><xs:complexType><xs:sequence><xs:element name="para"/></xs:sequence></xs:complexType></xs:element>
</xs:schema>`)},
	}
	schema, err := compileOverride(t, fsys)
	require.NoError(t, err)

	require.NoError(t, overrideValidate(t, schema, `<doc xmlns="`+ns+`">2010-01-01</doc>`),
		"the override reaches the doubly-chameleon-included doc")
	require.Error(t, overrideValidate(t, schema, `<doc xmlns="`+ns+`"><para/></doc>`),
		"the pre-override content model must be rejected")
}

// TestOverride_NestedSiblingScope is the OVR-001 regression: a child of a NESTED
// xs:override that matches nothing in its OWN target must NOT leak into the active
// override set used for a LATER SIBLING include/override target. Here d.xsd
// overrides leaf1.xsd with a `y` that matches nothing in leaf1, then includes
// leaf2.xsd which declares `y` as xs:string; the leaked `y` must not turn leaf2's
// string `y` into an integer.
func TestOverride_NestedSiblingScope(t *testing.T) {
	const xs = `xmlns:xs="http://www.w3.org/2001/XMLSchema"`
	fsys := fstest.MapFS{
		fileMain: &fstest.MapFile{Data: []byte(`<xs:schema ` + xs + `>
  <xs:override schemaLocation="d.xsd"/>
</xs:schema>`)},
		"d.xsd": &fstest.MapFile{Data: []byte(`<xs:schema ` + xs + `>
  <xs:override schemaLocation="leaf1.xsd">
    <xs:element name="y" type="xs:integer"/>
  </xs:override>
  <xs:include schemaLocation="leaf2.xsd"/>
</xs:schema>`)},
		"leaf1.xsd": &fstest.MapFile{Data: []byte(`<xs:schema ` + xs + `>
  <xs:element name="x" type="xs:string"/>
</xs:schema>`)},
		"leaf2.xsd": &fstest.MapFile{Data: []byte(`<xs:schema ` + xs + `>
  <xs:element name="y" type="xs:string"/>
  <xs:element name="doc"><xs:complexType><xs:sequence><xs:element ref="y"/></xs:sequence></xs:complexType></xs:element>
</xs:schema>`)},
	}
	schema, err := compileOverride(t, fsys)
	require.NoError(t, err)

	// y must remain xs:string (leaf2), so a non-numeric value is valid. Under the
	// leak bug y becomes xs:integer and "abc" is wrongly rejected.
	require.NoError(t, overrideValidate(t, schema, `<doc><y>abc</y></doc>`),
		"sibling include's y must keep its own type, not the unmatched nested-override y")
}

// TestOverride_NestedChildContext is the OVR-002 regression: an override child
// declared in a NESTED override whose owning document sets
// elementFormDefault="qualified" must be registered with THAT document's form
// default, even though the ROOT schema is unqualified. The override child carries
// a local element `para`; under the correct context it is namespace-qualified.
func TestOverride_NestedChildContext(t *testing.T) {
	const xs = `xmlns:xs="http://www.w3.org/2001/XMLSchema"`
	const ns = "urn:o"
	fsys := fstest.MapFS{
		// root: unqualified (no elementFormDefault).
		fileMain: &fstest.MapFile{Data: []byte(`<xs:schema ` + xs + ` targetNamespace="` + ns + `">
  <xs:override schemaLocation="mid.xsd"/>
</xs:schema>`)},
		// mid: qualified. Its override child `doc` owns a local `para`.
		"mid.xsd": &fstest.MapFile{Data: []byte(`<xs:schema ` + xs + ` targetNamespace="` + ns + `" elementFormDefault="qualified">
  <xs:override schemaLocation="leaf.xsd">
    <xs:element name="doc"><xs:complexType><xs:sequence><xs:element name="para" type="xs:string"/></xs:sequence></xs:complexType></xs:element>
  </xs:override>
</xs:schema>`)},
		"leaf.xsd": &fstest.MapFile{Data: []byte(`<xs:schema ` + xs + ` targetNamespace="` + ns + `">
  <xs:element name="doc" type="xs:string"/>
</xs:schema>`)},
	}
	schema, err := compileOverride(t, fsys)
	require.NoError(t, err)

	// mid declares elementFormDefault="qualified", so `para` lives in ns. Under the
	// wrong-context bug it is registered unqualified and a qualified para fails.
	require.NoError(t, overrideValidate(t, schema, `<doc xmlns="`+ns+`"><para>x</para></doc>`),
		"nested override child's local element must be qualified per its OWN document")
	require.Error(t, overrideValidate(t, schema, `<doc xmlns="`+ns+`"><para xmlns="">x</para></doc>`),
		"an unqualified para must be rejected when the owning document is qualified")
}
