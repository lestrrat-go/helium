package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// nilledFwdSchema declares <doc>'s <n> child as a NILLABLE xs:integer, so a
// source <n xsi:nil="true"/> validates as a nilled element carrying the xs:integer
// annotation. The source references the schema through xsi:schemaLocation (NOT
// xsl:import-schema), so validation runs at the source-document level.
const nilledFwdSchema = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="urn:nilfwd"
           xmlns:t="urn:nilfwd"
           elementFormDefault="qualified">
  <xs:element name="doc" type="t:docType"/>
  <xs:complexType name="docType">
    <xs:sequence>
      <xs:element name="n" type="xs:integer" nillable="true"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

// TestNilledForwardedToXPath3 verifies that the PSVI nilled property tracked by
// xslt3 during source validation is forwarded into the xpath3 evaluator, so the
// xpath3-owned semantics — fn:data of a nilled element is the empty sequence,
// and an element(name, type) instance-of test excludes a nilled element while
// element(name, type?) still matches it — are nilled-aware inside an
// xslt3-evaluated expression. fn:nilled itself uses the xslt3 override and is
// asserted as a sanity control. xsl:strip-space is active to force the
// validated-copy source path (which records nilled flags on the copy the
// transform navigates).
func TestNilledForwardedToXPath3(t *testing.T) {
	t.Parallel()

	const schemaLoc = "mem:/nilfwd/schema.xsd"

	const stylesheet = `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:t="urn:nilfwd"
  version="3.0">
  <xsl:strip-space elements="*"/>
  <xsl:output method="text"/>
  <xsl:template match="/">
    <xsl:value-of select="string-join((
      if (nilled(/t:doc/t:n)) then 'nil' else 'notnil',
      if (/t:doc/t:n instance of element(t:n, xs:integer)) then 'is-int' else 'not-int',
      if (/t:doc/t:n instance of element(t:n, xs:integer?)) then 'q-is-int' else 'q-not-int',
      string(count(data(/t:doc/t:n)))
    ), '|')"/>
  </xsl:template>
</xsl:stylesheet>`

	const src = `<?xml version="1.0"?>
<doc xmlns="urn:nilfwd"
     xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
     xsi:schemaLocation="urn:nilfwd mem:/nilfwd/schema.xsd">
  <n xsi:nil="true"/>
</doc>`

	ctx := t.Context()
	ssDoc, err := helium.NewParser().Parse(ctx, []byte(stylesheet))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(ctx, ssDoc)
	require.NoError(t, err)

	resolver := &exactRuntimeURIResolver{files: map[string]string{schemaLoc: nilledFwdSchema}}
	source, err := helium.NewParser().Parse(ctx, []byte(src))
	require.NoError(t, err)

	out, err := ss.Transform(source).URIResolver(resolver).Serialize(ctx)
	require.NoError(t, err)
	require.True(t, resolver.askedFor(schemaLoc), "resolver must be asked for the source schema")

	// nil       : fn:nilled true (control, via xslt3 override)
	// not-int   : nilled element does NOT match element(t:n, xs:integer) — the forwarded fix
	// q-is-int  : nilled element DOES match element(t:n, xs:integer?)
	// 0         : data() of a nilled element is the empty sequence
	require.Equal(t, "nil|not-int|q-is-int|0", out)
}
