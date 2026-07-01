package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestNamedGroupContentModel verifies the schema-representation rule that a named
// model group definition (<xs:group name="...">) has content model
// (annotation?, (all | choice | sequence)): exactly one model group child, at most
// one annotation which must precede it, and no other element children (XSD
// Structures §3.7.2). This is version-independent, so it is enforced in both XSD
// 1.0 (default) and XSD 1.1. Mirrors W3C msMeta/Group groupO002/O003/O010-O013/
// O015-O022/O027.
func TestNamedGroupContentModel(t *testing.T) {
	t.Parallel()

	const head = `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">`
	const tail = `</xsd:schema>`

	for _, tc := range []struct {
		name    string
		group   string
		wantErr bool
	}{
		{
			name:  "valid annotation then sequence",
			group: `<xsd:group name="A"><xsd:annotation/><xsd:sequence><xsd:element name="a1"/></xsd:sequence></xsd:group>`,
		},
		{
			name:  "valid choice only",
			group: `<xsd:group name="A"><xsd:choice><xsd:element name="a1"/></xsd:choice></xsd:group>`,
		},
		{
			name:    "two annotations (O002)",
			group:   `<xsd:group name="A"><xsd:annotation/><xsd:annotation/><xsd:sequence><xsd:element name="a1"/></xsd:sequence></xsd:group>`,
			wantErr: true,
		},
		{
			name:    "annotation after model group (O003)",
			group:   `<xsd:group name="A"><xsd:sequence><xsd:element name="a1"/></xsd:sequence><xsd:annotation/></xsd:group>`,
			wantErr: true,
		},
		{
			name:    "element child (O010)",
			group:   `<xsd:group name="A"><xsd:annotation/><xsd:element name="a"/></xsd:group>`,
			wantErr: true,
		},
		{
			name:    "complexType child (O011)",
			group:   `<xsd:group name="A"><xsd:complexType/></xsd:group>`,
			wantErr: true,
		},
		{
			name:    "attribute child (O013)",
			group:   `<xsd:group name="A"><xsd:attribute name="att1"/></xsd:group>`,
			wantErr: true,
		},
		{
			name:    "two model groups all+choice (O015)",
			group:   `<xsd:group name="A"><xsd:all><xsd:element name="a1"/></xsd:all><xsd:choice><xsd:element name="c1"/></xsd:choice></xsd:group>`,
			wantErr: true,
		},
		{
			name:    "two model groups sequence+sequence (O022)",
			group:   `<xsd:group name="A"><xsd:sequence><xsd:element name="d1"/></xsd:sequence><xsd:sequence><xsd:element name="c1"/></xsd:sequence></xsd:group>`,
			wantErr: true,
		},
		{
			name:    "group ref as child (O027)",
			group:   `<xsd:group name="A"><xsd:group ref="A"/></xsd:group>`,
			wantErr: true,
		},
		{
			name:    "no model group (annotation only)",
			group:   `<xsd:group name="A"><xsd:annotation/></xsd:group>`,
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			schema := head + tc.group + tail
			_, errs := compileWithErrors(t, schema)
			if tc.wantErr {
				require.NotEmpty(t, errs, "expected a schema error rejecting the group content model")
				require.Contains(t, errs, "}group'")
			}
			if !tc.wantErr {
				require.Empty(t, errs, "unexpected schema error for a valid named group")
			}
		})
	}
}

// TestNamedGroupContentModelVersion11 confirms the same content-model rule fires
// under the XSD 1.1 opt-in (the rule is version-independent).
func TestNamedGroupContentModelVersion11(t *testing.T) {
	t.Parallel()

	schema := `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">` +
		`<xsd:group name="A"><xsd:all><xsd:element name="a1"/></xsd:all><xsd:choice><xsd:element name="c1"/></xsd:choice></xsd:group>` +
		`</xsd:schema>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, cerr := xsd.NewCompiler().Version(xsd.Version11).Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
	requireCompileResultErr(t, cerr)
	require.Error(t, cerr, "a two-model-group named group must be rejected under Version11")
	require.NoError(t, collector.Close())
	_, errs := partitionCompileErrors(collector.Errors())
	require.Contains(t, errs, "}group'")
}

// TestNamedGroupDefinitionOccursRejected verifies the schema-representation rule
// (§3.7.2) that a named model group DEFINITION must not carry minOccurs/maxOccurs
// — those attributes belong to the group REFERENCE (particle) form (§3.8.2). It is
// a version-independent XSD rule. Mirrors W3C msMeta/Group groupD001-groupD004.
// A group reference legitimately carrying the occurrence attributes stays valid.
func TestNamedGroupDefinitionOccursRejected(t *testing.T) {
	t.Parallel()

	const head = `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">`
	const tail = `</xsd:schema>`
	const body = `<xsd:sequence><xsd:element name="a1"/></xsd:sequence>`

	for _, tc := range []struct {
		name    string
		schema  string
		wantErr bool
	}{
		{
			name:    "minOccurs on definition (D001)",
			schema:  `<xsd:group name="A" minOccurs="1">` + body + `</xsd:group>`,
			wantErr: true,
		},
		{
			name:    "maxOccurs on definition (D003)",
			schema:  `<xsd:group name="A" maxOccurs="1">` + body + `</xsd:group>`,
			wantErr: true,
		},
		{
			name:    "both occurs on definition",
			schema:  `<xsd:group name="A" minOccurs="0" maxOccurs="2">` + body + `</xsd:group>`,
			wantErr: true,
		},
		{
			name:   "definition without occurs is valid",
			schema: `<xsd:group name="A">` + body + `</xsd:group>`,
		},
		{
			name: "reference with occurs is valid",
			schema: `<xsd:group name="A">` + body + `</xsd:group>` +
				`<xsd:complexType name="T"><xsd:sequence>` +
				`<xsd:group ref="A" minOccurs="0" maxOccurs="unbounded"/>` +
				`</xsd:sequence></xsd:complexType>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, errs := compileWithErrors(t, head+tc.schema+tail)
			if tc.wantErr {
				require.NotEmpty(t, errs, "expected a schema error rejecting the group definition")
				require.Contains(t, errs, "}group'")
				return
			}
			require.Empty(t, errs, "unexpected schema error for a valid group")
		})
	}
}

// TestNamedGroupDefinitionOccursRejectedVersion11 confirms the rule is enforced
// under the XSD 1.1 opt-in too (version-independent).
func TestNamedGroupDefinitionOccursRejectedVersion11(t *testing.T) {
	t.Parallel()

	schema := `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">` +
		`<xsd:group name="A" maxOccurs="1"><xsd:sequence><xsd:element name="a1"/></xsd:sequence></xsd:group>` +
		`</xsd:schema>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, cerr := xsd.NewCompiler().Version(xsd.Version11).Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
	requireCompileResultErr(t, cerr)
	require.Error(t, cerr, "maxOccurs on a group definition must be rejected under Version11")
	require.NoError(t, collector.Close())
	_, errs := partitionCompileErrors(collector.Errors())
	require.Contains(t, errs, "}group'")
}
