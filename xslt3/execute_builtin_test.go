package xslt3_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBuiltinRuleXDMChildren checks that the built-in template rules and an
// xsl:apply-templates without select process the XDM children of a node
// only. A DOCTYPE declaration is not a child of the document node, an entity
// reference is not a child of its element, and an attribute has no children,
// so the match rules never see any of them.
func TestBuiltinRuleXDMChildren(t *testing.T) {
	const misc = `<?p1 x?><!--c1--><!DOCTYPE r [<!ELEMENT r ANY>]><!--c2--><r/><?p2 y?>`
	const entity = `<!DOCTYPE r [<!ENTITY e "<y/>">]><r><x/>&e;<z/></r>`
	const docChildren = "<n>[p1:x]</n><n>[:c1]</n><n>[:c2]</n><n>[r:]</n><n>[p2:y]</n>"

	tests := []struct {
		name      string
		onNoMatch string
		src       string
		rules     string
		want      string
	}{
		{"text-only-copy document", "text-only-copy", misc, labelRule("node()"), docChildren},
		{"shallow-copy document", "shallow-copy", misc, labelRule("node()"), docChildren},
		{"shallow-skip document", "shallow-skip", misc, labelRule("node()"), docChildren},
		{"deep-skip document", "deep-skip", misc, labelRule("node()"), docChildren},
		{"fail document", "fail", misc, labelRule("node()"), docChildren},
		{"text-only-copy element", "text-only-copy", entity, labelRule("r/node()"), "<n>[x:]</n><n>[z:]</n>"},
		{"shallow-copy element", "shallow-copy", entity, labelRule("r/node()"), "<r><n>[x:]</n><n>[z:]</n></r>"},
		{"shallow-skip element", "shallow-skip", entity, labelRule("r/node()"), "<n>[x:]</n><n>[z:]</n>"},
		{
			name:      "apply-templates without select on the document",
			onNoMatch: "fail",
			src:       misc,
			rules:     `<xsl:template match="/"><xsl:apply-templates/></xsl:template>` + labelRule("node()"),
			want:      docChildren,
		},
		{
			name:      "apply-templates without select on an element",
			onNoMatch: "fail",
			src:       entity,
			rules: `<xsl:template match="/"><xsl:apply-templates select="r"/></xsl:template>` +
				`<xsl:template match="r"><xsl:apply-templates/></xsl:template>` + labelRule("r/node()"),
			want: "<n>[x:]</n><n>[z:]</n>",
		},
		{
			name:      "apply-templates without select on an attribute",
			onNoMatch: "fail",
			src:       `<r a="v"/>`,
			rules: `<xsl:template match="/"><xsl:apply-templates select="r/@a"/></xsl:template>` +
				`<xsl:template match="@a">[<xsl:apply-templates/>]</xsl:template>` + labelRule("node()"),
			want: "[]",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			xsltSrc := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output omit-xml-declaration="yes"/>
  <xsl:mode on-no-match="` + tc.onNoMatch + `"/>` + tc.rules + `
</xsl:stylesheet>`
			require.Equal(t, tc.want, transformDoctype(t, xsltSrc, tc.src))
		})
	}
}

// labelRule returns a template rule for match that writes the matched node
// as <n>[name:string-value]</n>.
func labelRule(match string) string {
	return `<xsl:template match="` + match + `"><n>` + patternDoctypeLabel + `</n></xsl:template>`
}
