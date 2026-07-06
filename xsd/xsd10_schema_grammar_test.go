package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

const xsd10GrammarMainXSD = "main.xsd"

// compileXSD10Grammar compiles a single-document schema (no external includes)
// with the default XSD 1.0 processor and returns the fatal-error string.
func compileXSD10Grammar(t *testing.T, src string) string {
	t.Helper()
	fsys := fstest.MapFS{xsd10GrammarMainXSD: &fstest.MapFile{Data: []byte(src)}}
	data, err := fsys.ReadFile(xsd10GrammarMainXSD)
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, _ = xsd.NewCompiler().Label(xsd10GrammarMainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
	require.NoError(t, collector.Close())
	_, errStr := partitionCompileErrors(collector.Errors())
	return errStr
}

// TestXSD10ElementStrayChild verifies that an <xs:element> declaration rejects an
// XSD-namespace child outside its §3.3.2 content model
// (annotation?, ((simpleType | complexType)?, (unique | key | keyref)*)) — an
// <xs:attribute> (msMeta/Attribute_w3c attQ002) or an <xs:group> reference
// (msMeta/Group_w3c groupO024) is a schema-representation error in XSD 1.0.
func TestXSD10ElementStrayChild(t *testing.T) {
	t.Parallel()

	invalid := map[string]string{
		"attQ002: attribute child of element": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
			<xs:element name="root">
				<xs:attribute name="att"/>
				<xs:complexType><xs:sequence><xs:element name="foo"/></xs:sequence></xs:complexType>
			</xs:element>
		</xs:schema>`,
		"groupO024: group ref child of element": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
			<xs:group name="A"><xs:sequence><xs:element name="a1"/></xs:sequence></xs:group>
			<xs:element name="doc">
				<xs:complexType><xs:sequence>
					<xs:element name="elem"><xs:group ref="A"/></xs:element>
				</xs:sequence></xs:complexType>
			</xs:element>
		</xs:schema>`,
		"element with sequence child": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
			<xs:element name="root"><xs:sequence><xs:element name="e"/></xs:sequence></xs:element>
		</xs:schema>`,
	}
	for name, src := range invalid {
		t.Run("invalid/"+name, func(t *testing.T) {
			t.Parallel()
			require.NotEmpty(t, compileXSD10Grammar(t, src), "expected rejection for %q", name)
		})
	}

	valid := map[string]string{
		"element with complexType + key": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
			<xs:element name="root">
				<xs:annotation><xs:documentation>ok</xs:documentation></xs:annotation>
				<xs:complexType><xs:sequence><xs:element name="foo" type="xs:string"/></xs:sequence></xs:complexType>
				<xs:unique name="u"><xs:selector xpath="foo"/><xs:field xpath="."/></xs:unique>
			</xs:element>
		</xs:schema>`,
		"element with simpleType": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
			<xs:element name="root">
				<xs:simpleType><xs:restriction base="xs:integer"/></xs:simpleType>
			</xs:element>
		</xs:schema>`,
	}
	for name, src := range valid {
		t.Run("valid/"+name, func(t *testing.T) {
			t.Parallel()
			require.Empty(t, compileXSD10Grammar(t, src), "expected valid element to compile for %q", name)
		})
	}
}

// TestXSD10AnyStrayChild verifies that an <xs:any> wildcard rejects any element
// child other than <xs:annotation> — its content model is (annotation?). A nested
// <xs:group> reference (msMeta/Group_w3c groupO026) is a schema-representation
// error, version-INDEPENDENT.
func TestXSD10AnyStrayChild(t *testing.T) {
	t.Parallel()

	invalid := map[string]string{
		"groupO026: group ref child of any": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
			<xs:group name="foo"><xs:sequence>
				<xs:any namespace="##any"><xs:group ref="bar"/></xs:any>
			</xs:sequence></xs:group>
			<xs:group name="bar"><xs:sequence><xs:element name="elem"/></xs:sequence></xs:group>
		</xs:schema>`,
		"any with element child": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
			<xs:complexType name="t"><xs:sequence>
				<xs:any><xs:element name="e"/></xs:any>
			</xs:sequence></xs:complexType>
		</xs:schema>`,
	}
	for name, src := range invalid {
		t.Run("invalid/"+name, func(t *testing.T) {
			t.Parallel()
			require.NotEmpty(t, compileXSD10Grammar(t, src), "expected rejection for %q", name)
		})
	}

	valid := map[string]string{
		"any with annotation child": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
			<xs:complexType name="t"><xs:sequence>
				<xs:any><xs:annotation><xs:documentation>ok</xs:documentation></xs:annotation></xs:any>
			</xs:sequence></xs:complexType>
		</xs:schema>`,
		"bare any": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
			<xs:complexType name="t"><xs:sequence><xs:any/></xs:sequence></xs:complexType>
		</xs:schema>`,
	}
	for name, src := range valid {
		t.Run("valid/"+name, func(t *testing.T) {
			t.Parallel()
			require.Empty(t, compileXSD10Grammar(t, src), "expected valid any to compile for %q", name)
		})
	}
}

// TestXSD10GlobalAttributeXSINamespace verifies that a global <xs:attribute>
// whose {target namespace} is the XSI namespace is rejected in XSD 1.0
// (msMeta/Attribute_w3c attKa015). The XSI namespace is reserved for the four
// processor attributes; a schema may not add to it.
func TestXSD10GlobalAttributeXSINamespace(t *testing.T) {
	t.Parallel()

	invalid := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
		targetNamespace="http://www.w3.org/2001/XMLSchema-instance">
		<xs:attribute name="ga1" type="xs:integer"/>
	</xs:schema>`
	require.NotEmpty(t, compileXSD10Grammar(t, invalid), "expected rejection of global attribute in XSI namespace")

	valid := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:ok">
		<xs:attribute name="ga1" type="xs:integer"/>
	</xs:schema>`
	require.Empty(t, compileXSD10Grammar(t, valid), "expected a normal-namespace global attribute to compile")
}

// TestXSD10AttrGroupProhibitedDefault verifies that an <xs:attribute> with
// use="prohibited" carrying a default value constraint, declared directly inside
// an <xs:attributeGroup>, is rejected in XSD 1.0 (msMeta/Attribute_w3c attKb005):
// the "default requires use=optional" schema-representation rule is
// version-INDEPENDENT and applies inside an attribute group too.
func TestXSD10AttrGroupProhibitedDefault(t *testing.T) {
	t.Parallel()

	invalid := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t">
		<xs:attributeGroup name="attG">
			<xs:attribute name="aga1" use="prohibited" default="abc"/>
			<xs:attribute name="aga2"/>
		</xs:attributeGroup>
	</xs:schema>`
	require.NotEmpty(t, compileXSD10Grammar(t, invalid), "expected rejection of prohibited attribute with default")

	// A prohibited attribute WITHOUT a value constraint is a pointless-but-valid
	// use inside a group (libxml2 warns and skips it); it must still compile.
	valid := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t">
		<xs:attributeGroup name="attG">
			<xs:attribute name="aga1" use="prohibited"/>
			<xs:attribute name="aga2"/>
		</xs:attributeGroup>
	</xs:schema>`
	require.Empty(t, compileXSD10Grammar(t, valid), "expected a prohibited attribute without a value constraint to compile")
}

// TestXSD10GlobalAttributeNoName verifies that a top-level <xs:attribute> with no
// @name is rejected in XSD 1.0 (msMeta/Attribute_w3c attQ005): @name is required
// on a global attribute declaration.
func TestXSD10GlobalAttributeNoName(t *testing.T) {
	t.Parallel()

	invalid := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t">
		<xs:attribute/>
	</xs:schema>`
	require.NotEmpty(t, compileXSD10Grammar(t, invalid), "expected rejection of global attribute without name")
}

// TestXSD10RedefineStrayChild verifies that <xs:redefine> rejects an
// XSD-namespace child outside its content model
// (annotation | (simpleType | complexType | group | attributeGroup))* — an
// <xs:element> (sunData xsd003-1.e) or an <xs:attribute> (sunData xsd003-2.e) is
// a schema-representation error, version-INDEPENDENT.
func TestXSD10RedefineStrayChild(t *testing.T) {
	t.Parallel()

	const modXSD = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns="urn:t">
		<xs:simpleType name="st"><xs:restriction base="xs:string"/></xs:simpleType>
		<xs:element name="root" type="xs:string"/>
		<xs:attribute name="gAtt" type="xs:string"/>
	</xs:schema>`

	compileWithMod := func(t *testing.T, mainSrc string) string {
		t.Helper()
		fsys := fstest.MapFS{
			xsd10GrammarMainXSD: &fstest.MapFile{Data: []byte(mainSrc)},
			"mod.xsd":           &fstest.MapFile{Data: []byte(modXSD)},
		}
		data, err := fsys.ReadFile(xsd10GrammarMainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, _ = xsd.NewCompiler().Label(xsd10GrammarMainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		require.NoError(t, collector.Close())
		_, errStr := partitionCompileErrors(collector.Errors())
		return errStr
	}

	invalid := map[string]string{
		"xsd003-1.e: element in redefine": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns="urn:t">
			<xs:redefine schemaLocation="mod.xsd"><xs:element name="root"/></xs:redefine>
		</xs:schema>`,
		"xsd003-2.e: attribute in redefine": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns="urn:t">
			<xs:redefine schemaLocation="mod.xsd"><xs:attribute name="gAtt" type="st"/></xs:redefine>
		</xs:schema>`,
	}
	for name, src := range invalid {
		t.Run("invalid/"+name, func(t *testing.T) {
			t.Parallel()
			require.NotEmpty(t, compileWithMod(t, src), "expected rejection for %q", name)
		})
	}

	// A redefine carrying only a valid self-restricting simpleType must compile.
	valid := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns="urn:t">
		<xs:redefine schemaLocation="mod.xsd">
			<xs:simpleType name="st"><xs:restriction base="st"><xs:maxLength value="5"/></xs:restriction></xs:simpleType>
		</xs:redefine>
	</xs:schema>`
	require.Empty(t, compileWithMod(t, valid), "expected a valid redefine to compile")
}
