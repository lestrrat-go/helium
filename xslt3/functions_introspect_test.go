package xslt3_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFunctionAvailableArityPastInt32 verifies that function-available reads an
// arity past 2^31-1 as a number on every platform. Parsing it into a 32-bit int
// fails, and the arity would fall back to "any", reporting a fixed-arity
// function as available.
func TestFunctionAvailableArityPastInt32(t *testing.T) {
	t.Parallel()

	const ss = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out><xsl:value-of select="function-available('string-length', 4294967297),
      function-available('concat', 4294967297)"/></out>
  </xsl:template>
</xsl:stylesheet>`
	out, err := transformStr(t, ss)
	require.NoError(t, err)
	require.Contains(t, out, "<out>false true</out>")
}
