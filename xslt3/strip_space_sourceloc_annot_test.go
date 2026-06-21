package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// ssaSchema types <doc>'s <n> child as xs:integer so a source validated against
// it carries an xs:integer annotation on <n>. The source references this schema
// purely through xsi:schemaLocation (NOT via xsl:import-schema), so the
// stylesheet itself is NOT schema-aware.
const ssaSchema = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="urn:ssa"
           xmlns:t="urn:ssa"
           elementFormDefault="qualified">
  <xs:element name="doc" type="t:docType"/>
  <xs:complexType name="docType">
    <xs:sequence>
      <xs:element name="n" type="xs:integer"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

// TestStripSpaceSourceSchemaLocationAnnotationPreserved is the regression test
// for finding 664-9: when the stylesheet is NOT schema-aware (no
// xsl:import-schema) but the source document is typed solely via
// xsi:schemaLocation, AND xsl:strip-space rules are active, the strip copy that
// the transform runs against must still receive the type annotations gathered
// during source validation.
//
// Before the fix, the schemaActive gate that decided whether to build the
// original->copy node map only looked at ss.schemaAware / ss.schemas /
// cfg.sourceSchemas — none of which are set for the xsi:schemaLocation-discovered
// path. So the map stayed nil, remapValidationNode left the annotations on the
// ORIGINAL nodes, and the transform (navigating the COPY) saw only untyped
// nodes. "n instance of element(*, xs:integer)" therefore returned false.
//
// With the broadened gate the map is built whenever strip rules exist and the
// source could be typed (here: it declares xsi:schemaLocation), so the annotation
// rides onto the copy and the instance-of test passes. The no-strip control and
// a no-schemaLocation control bracket the behaviour.
func TestStripSpaceSourceSchemaLocationAnnotationPreserved(t *testing.T) {
	t.Parallel()

	const schemaLoc = "mem:/ssa/schema.xsd"

	stylesheet := func(strip bool) string {
		stripDecl := ""
		if strip {
			stripDecl = `<xsl:strip-space elements="*"/>`
		}
		return `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:t="urn:ssa"
  version="3.0">
  ` + stripDecl + `
  <xsl:output method="text"/>
  <xsl:template match="/">
    <xsl:value-of select="if (/t:doc/t:n instance of element(*, xs:integer)) then 'TYPED' else 'UNTYPED'"/>
  </xsl:template>
</xsl:stylesheet>`
	}

	const src = `<?xml version="1.0"?>
<doc xmlns="urn:ssa"
     xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
     xsi:schemaLocation="urn:ssa mem:/ssa/schema.xsd">
  <n>42</n>
</doc>`

	run := func(t *testing.T, strip bool) string {
		t.Helper()
		ctx := t.Context()
		ssDoc, err := helium.NewParser().Parse(ctx, []byte(stylesheet(strip)))
		require.NoError(t, err)
		ss, err := xslt3.NewCompiler().Compile(ctx, ssDoc)
		require.NoError(t, err)

		resolver := &exactRuntimeURIResolver{files: map[string]string{
			schemaLoc: ssaSchema,
		}}
		source, err := helium.NewParser().Parse(ctx, []byte(src))
		require.NoError(t, err)
		out, err := ss.Transform(source).URIResolver(resolver).Serialize(ctx)
		require.NoError(t, err)
		require.True(t, resolver.askedFor(schemaLoc),
			"resolver must be asked for the source schema %q; got %v", schemaLoc, resolver.asked)
		return out
	}

	// Control: with no strip-space rule the transform runs on the original
	// (validated) tree, so the annotation is naturally present.
	t.Run("no-strip control is typed", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "TYPED", run(t, false),
			"without strip-space the validated source must carry the xs:integer annotation")
	})

	// The regression: strip-space active must NOT lose the annotation on the copy.
	t.Run("strip-space preserves annotation on copy", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "TYPED", run(t, true),
			"strip-space must remap source type annotations onto the copy the transform runs on")
	})
}
