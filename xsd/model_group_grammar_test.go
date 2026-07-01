package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestModelGroupChildGrammar verifies the XML-representation content model of an
// inline model group compositor (xs:all/xs:sequence/xs:choice), §3.8.2:
// (annotation?, particle*) with at most one leading annotation, no non-particle
// (e.g. xs:simpleType/xs:complexType) children, no @name attribute, and — for the
// default XSD 1.0 processor — an xs:all restricted to element particles only. Each
// invalid schema was previously accepted (false-accept in msMeta/ModelGroups_w3c).
func TestModelGroupChildGrammar(t *testing.T) {
	t.Parallel()

	const mainXSD = "main.xsd"

	compile := func(t *testing.T, src string) string {
		t.Helper()
		fsys := fstest.MapFS{mainXSD: &fstest.MapFile{Data: []byte(src)}}
		data, err := fsys.ReadFile(mainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, _ = xsd.NewCompiler().Label(mainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		require.NoError(t, collector.Close())
		_, errStr := partitionCompileErrors(collector.Errors())
		return errStr
	}

	wrap := func(body string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` +
			`<xs:complexType name="t">` + body + `</xs:complexType>` +
			`<xs:element name="root" type="t"/></xs:schema>`
	}

	invalid := map[string]string{
		"all: two annotations": `<xs:all>
			<xs:annotation><xs:documentation>a</xs:documentation></xs:annotation>
			<xs:annotation><xs:documentation>b</xs:documentation></xs:annotation>
		</xs:all>`,
		"all: element then annotation": `<xs:all>
			<xs:element name="e1"/>
			<xs:annotation><xs:documentation>b</xs:documentation></xs:annotation>
		</xs:all>`,
		"all: simpleType child": `<xs:all>
			<xs:simpleType name="s"><xs:restriction base="xs:integer"/></xs:simpleType>
		</xs:all>`,
		"all: complexType child": `<xs:all><xs:complexType name="c"/></xs:all>`,
		"all: group child (1.0)": `<xs:all><xs:group name="g"/></xs:all>`,
		"all: sequence child":    `<xs:all><xs:sequence><xs:element name="e1"/></xs:sequence></xs:all>`,
		"all: name attribute":    `<xs:all name="bad"><xs:element name="e1"/></xs:all>`,
		"sequence: two annotations": `<xs:sequence>
			<xs:annotation><xs:documentation>a</xs:documentation></xs:annotation>
			<xs:annotation><xs:documentation>b</xs:documentation></xs:annotation>
		</xs:sequence>`,
		"sequence: element then annotation": `<xs:sequence>
			<xs:element name="e1"/>
			<xs:annotation><xs:documentation>b</xs:documentation></xs:annotation>
		</xs:sequence>`,
		"sequence: attribute child": `<xs:sequence><xs:attribute name="a"/></xs:sequence>`,
		"choice: element then annotation": `<xs:choice>
			<xs:element name="e1"/>
			<xs:annotation><xs:documentation>b</xs:documentation></xs:annotation>
		</xs:choice>`,
	}
	for name, body := range invalid {
		t.Run("invalid/"+name, func(t *testing.T) {
			t.Parallel()
			errStr := compile(t, wrap(body))
			require.NotEmpty(t, errStr, "expected schema rejection for %q", name)
		})
	}

	valid := map[string]string{
		"all: annotation then elements": `<xs:all>
			<xs:annotation><xs:documentation>ok</xs:documentation></xs:annotation>
			<xs:element name="e1"/><xs:element name="e2"/>
		</xs:all>`,
		"sequence: annotation then particles": `<xs:sequence>
			<xs:annotation><xs:documentation>ok</xs:documentation></xs:annotation>
			<xs:element name="e1"/>
			<xs:sequence><xs:element name="e2"/></xs:sequence>
		</xs:sequence>`,
		"choice: annotation then particles": `<xs:choice>
			<xs:annotation><xs:documentation>ok</xs:documentation></xs:annotation>
			<xs:element name="e1"/><xs:element name="e2"/>
		</xs:choice>`,
	}
	for name, body := range valid {
		t.Run("valid/"+name, func(t *testing.T) {
			t.Parallel()
			errStr := compile(t, wrap(body))
			require.Empty(t, errStr, "expected valid model group to compile for %q", name)
		})
	}
}
