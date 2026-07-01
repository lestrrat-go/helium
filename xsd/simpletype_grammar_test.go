package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestSimpleTypeGrammar exercises the schema-representation content model of
// xs:simpleType and its restriction/list/union derivation bodies (XSD Structures
// §3.14.2/§3.15.2/§3.16.2), enforced version-independently in the default (XSD
// 1.0) compiler. Each invalid schema was formerly accepted (false-accept); each
// valid schema must still compile cleanly. Mirrors W3C msMeta/SimpleType_w3c
// groups stB*/stC*/stD*/stE* and the stray-facet stF*/stJ*/stK* cases.
func TestSimpleTypeGrammar(t *testing.T) {
	t.Parallel()

	const head = `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">`
	const tail = `</xsd:schema>`

	t.Run("rejects", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			schema string
		}{
			{"annotation-only", `<xsd:simpleType name="t"><xsd:annotation/></xsd:simpleType>`},
			{"two-annotations", `<xsd:simpleType name="t"><xsd:annotation/><xsd:annotation/></xsd:simpleType>`},
			{"two-restrictions", `<xsd:simpleType name="t"><xsd:restriction base="xsd:string"/><xsd:restriction base="xsd:string"/></xsd:simpleType>`},
			{"restriction-then-annotation", `<xsd:simpleType name="t"><xsd:restriction base="xsd:string"/><xsd:annotation/></xsd:simpleType>`},
			{"restriction-then-stray", `<xsd:simpleType name="t"><xsd:restriction base="xsd:string"/><xsd:attribute name="a"/></xsd:simpleType>`},
			{"two-lists", `<xsd:simpleType name="t"><xsd:list itemType="xsd:string"/><xsd:list itemType="xsd:string"/></xsd:simpleType>`},
			{"list-then-restriction", `<xsd:simpleType name="t"><xsd:list itemType="xsd:string"/><xsd:restriction base="xsd:string"/></xsd:simpleType>`},
			{"annotation-after-derivation-then-derivation", `<xsd:simpleType name="t"><xsd:annotation/><xsd:annotation/><xsd:restriction base="xsd:string"/></xsd:simpleType>`},
			{"restriction-two-simpleTypes", `<xsd:simpleType name="t"><xsd:restriction><xsd:simpleType><xsd:restriction base="xsd:string"/></xsd:simpleType><xsd:simpleType><xsd:restriction base="xsd:integer"/></xsd:simpleType></xsd:restriction></xsd:simpleType>`},
			{"restriction-with-attribute", `<xsd:simpleType name="t"><xsd:restriction base="xsd:integer"><xsd:maxExclusive value="5"/><xsd:attribute name="a"/></xsd:restriction></xsd:simpleType>`},
			{"restriction-base-and-simpleType", `<xsd:simpleType name="t"><xsd:restriction base="xsd:string"><xsd:simpleType><xsd:restriction base="xsd:string"/></xsd:simpleType></xsd:restriction></xsd:simpleType>`},
			{"restriction-stray-facet-lowercase", `<xsd:simpleType name="b"><xsd:list itemType="xsd:string"/></xsd:simpleType><xsd:simpleType name="t"><xsd:restriction base="b"><xsd:whitespace value="preserve"/></xsd:restriction></xsd:simpleType>`},
			{"restriction-stray-duration", `<xsd:simpleType name="b"><xsd:list itemType="xsd:integer"/></xsd:simpleType><xsd:simpleType name="t"><xsd:restriction base="b"><xsd:duration value="P1D"/></xsd:restriction></xsd:simpleType>`},
			{"list-empty-itemType", `<xsd:simpleType name="t"><xsd:list itemType=""/></xsd:simpleType>`},
			{"list-two-simpleTypes", `<xsd:simpleType name="t"><xsd:list><xsd:simpleType><xsd:restriction base="xsd:integer"/></xsd:simpleType><xsd:simpleType><xsd:restriction base="xsd:string"/></xsd:simpleType></xsd:list></xsd:simpleType>`},
			{"list-simpleType-then-annotation", `<xsd:simpleType name="t"><xsd:list><xsd:simpleType><xsd:restriction base="xsd:integer"/></xsd:simpleType><xsd:annotation/></xsd:list></xsd:simpleType>`},
			{"list-itemType-and-simpleType", `<xsd:simpleType name="t"><xsd:list itemType="xsd:integer"><xsd:simpleType><xsd:restriction base="xsd:integer"/></xsd:simpleType></xsd:list></xsd:simpleType>`},
			{"union-only-annotation", `<xsd:simpleType name="t"><xsd:union><xsd:annotation/></xsd:union></xsd:simpleType>`},
			{"union-two-annotations", `<xsd:simpleType name="t"><xsd:union><xsd:annotation/><xsd:annotation/><xsd:simpleType><xsd:restriction base="xsd:integer"/></xsd:simpleType></xsd:union></xsd:simpleType>`},
			{"union-simpleType-then-annotation", `<xsd:simpleType name="t"><xsd:union><xsd:simpleType><xsd:restriction base="xsd:integer"/></xsd:simpleType><xsd:annotation/></xsd:union></xsd:simpleType>`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				require.NotEmpty(t, compileSchemaErrors(t, head+tc.schema+tail),
					"expected %s to be rejected", tc.name)
			})
		}
	})

	t.Run("accepts", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			schema string
		}{
			{"restriction-base", `<xsd:simpleType name="t"><xsd:restriction base="xsd:string"><xsd:maxLength value="5"/></xsd:restriction></xsd:simpleType>`},
			{"annotation-then-restriction", `<xsd:simpleType name="t"><xsd:annotation/><xsd:restriction base="xsd:string"/></xsd:simpleType>`},
			{"restriction-inline-simpleType-then-facet", `<xsd:simpleType name="t"><xsd:restriction><xsd:simpleType><xsd:restriction base="xsd:string"/></xsd:simpleType><xsd:maxLength value="5"/></xsd:restriction></xsd:simpleType>`},
			{"list-itemType", `<xsd:simpleType name="t"><xsd:list itemType="xsd:string"/></xsd:simpleType>`},
			{"list-inline-simpleType", `<xsd:simpleType name="t"><xsd:list><xsd:simpleType><xsd:restriction base="xsd:integer"/></xsd:simpleType></xsd:list></xsd:simpleType>`},
			{"union-memberTypes", `<xsd:simpleType name="t"><xsd:union memberTypes="xsd:string xsd:integer"/></xsd:simpleType>`},
			{"union-annotation-then-members", `<xsd:simpleType name="t"><xsd:union><xsd:annotation/><xsd:simpleType><xsd:restriction base="xsd:integer"/></xsd:simpleType><xsd:simpleType><xsd:restriction base="xsd:string"/></xsd:simpleType></xsd:union></xsd:simpleType>`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				require.Empty(t, compileSchemaErrors(t, head+tc.schema+tail),
					"expected %s to compile cleanly", tc.name)
			})
		}
	})
}

// TestSimpleTypeGrammarVersionGatedFacets exercises the version-dependent edge of
// the simpleType restriction-body grammar: xs:assertion and xs:explicitTimezone
// are XSD 1.1-only constraining facets. In 1.1 they are valid facet children of a
// restriction; in 1.0 they are unrecognized XSD-namespace elements, so the
// schema-representation grammar rejects them as stray children (rather than
// letting parseFacets silently ignore them and false-accept the schema).
func TestSimpleTypeGrammarVersionGatedFacets(t *testing.T) {
	t.Parallel()

	const head = `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">`
	const tail = `</xsd:schema>`

	for _, tc := range []struct {
		name   string
		schema string
	}{
		{"assertion", `<xsd:simpleType name="t"><xsd:restriction base="xsd:string"><xsd:assertion test="true()"/></xsd:restriction></xsd:simpleType>`},
		{"explicitTimezone", `<xsd:simpleType name="t"><xsd:restriction base="xsd:dateTime"><xsd:explicitTimezone value="required"/></xsd:restriction></xsd:simpleType>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			schema := head + tc.schema + tail
			require.NotEmpty(t, compileSchemaErrorsVersion(t, schema, false),
				"expected %s to be rejected in XSD 1.0", tc.name)
			require.Empty(t, compileSchemaErrorsVersion(t, schema, true),
				"expected %s to compile cleanly in XSD 1.1", tc.name)
		})
	}
}

// compileSchemaErrorsVersion is compileSchemaErrors with a selectable XSD version:
// v11=true opts into XSD 1.1, otherwise the default (XSD 1.0) semantics apply.
func compileSchemaErrorsVersion(t *testing.T, schemaXML string, v11 bool) string {
	t.Helper()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	c := xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector)
	if v11 {
		c = c.Version(xsd.Version11)
	}
	_, _ = c.Compile(t.Context(), doc)
	_ = collector.Close()

	var b strings.Builder
	for _, e := range collector.Errors() {
		b.WriteString(e.Error())
		b.WriteString("\n")
	}
	return b.String()
}
