package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestStripSpaceValidatesBeforeStripping verifies that strict source-schema
// validation runs on the ORIGINAL (un-stripped) source tree, BEFORE xsl:strip-space
// removes whitespace-only text nodes. If validation ran after stripping, a
// whitespace-only element that the schema requires to be empty would have its
// content removed first, masking the validation error.
//
// The schema declares <s> as a simpleType restricting xs:string to length 0, so
// non-empty content (including a single space) is invalid. The source <s> </s>
// holds a whitespace-only text node that xsl:strip-space elements="s" would remove.
// With strip-before-validate (the regression), the whitespace node is gone before
// validation, <s> looks empty, and validation wrongly passes. With validate-before-
// strip (correct, matching the no-strip control), validation sees the space and
// fails. See finding 664-8.
func TestStripSpaceValidatesBeforeStripping(t *testing.T) {
	t.Parallel()

	stylesheet := func(strip bool) string {
		stripDecl := ""
		if strip {
			stripDecl = `<xsl:strip-space elements="s"/>`
		}
		return `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  version="3.0" default-validation="strict">
  ` + stripDecl + `
  <xsl:import-schema>
    <xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
      <xs:simpleType name="emptyString">
        <xs:restriction base="xs:string">
          <xs:length value="0"/>
        </xs:restriction>
      </xs:simpleType>
      <xs:element name="s" type="emptyString"/>
      <xs:complexType name="docType">
        <xs:sequence>
          <xs:element ref="s"/>
        </xs:sequence>
      </xs:complexType>
      <xs:element name="doc" type="docType"/>
    </xs:schema>
  </xsl:import-schema>
  <xsl:output method="text"/>
  <xsl:template match="/">done</xsl:template>
</xsl:stylesheet>`
	}

	const src = `<doc><s> </s></doc>`

	run := func(t *testing.T, strip bool) error {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(stylesheet(strip)))
		require.NoError(t, err)
		ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
		require.NoError(t, err)
		source, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		_, err = xslt3.TransformString(t.Context(), source, ss)
		return err
	}

	// Control: without strip-space the whitespace-only <s> content is present at
	// validation time and must fail strict validation.
	t.Run("no-strip control fails validation", func(t *testing.T) {
		t.Parallel()
		err := run(t, false)
		require.Error(t, err, "whitespace-only <s> must fail length-0 validation")
		require.Contains(t, err.Error(), "validation failed")
	})

	// With strip-space, validation must STILL fail: it runs on the original tree
	// before the whitespace node is removed. Before the fix this wrongly passed.
	t.Run("strip-space still fails validation", func(t *testing.T) {
		t.Parallel()
		err := run(t, true)
		require.Error(t, err, "strip-space must not mask the validation error")
		require.Contains(t, err.Error(), "validation failed")
	})
}
