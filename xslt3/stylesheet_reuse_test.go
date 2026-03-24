package xslt3_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestStylesheetReuseSerialization(t *testing.T) {
	// The stylesheet has no xsl:output declaration (so ss.outputs[""] starts nil).
	// The template produces a primary xsl:result-document with
	// omit-xml-declaration="yes". During executeTransform, this creates a new
	// OutputDef on ss.outputs[""] with OmitDeclaration=true (line 437+451).
	//
	// If the stylesheet is mutated, the second transform inherits
	// OmitDeclaration=true even without xsl:result-document, changing the
	// serialized output.
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document omit-xml-declaration="yes">
      <out>hello</out>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	source := parseTransformSource(t)

	// First transform — xsl:result-document sets omit-xml-declaration=yes
	result1, err := ss.Transform(source).Serialize(t.Context())
	require.NoError(t, err)

	// Second transform with the SAME compiled stylesheet
	result2, err := ss.Transform(source).Serialize(t.Context())
	require.NoError(t, err)

	// Both results must be identical. If the stylesheet was mutated, the
	// second run may differ (e.g., xml declaration present vs absent).
	require.Equal(t, result1, result2,
		"second transform produced different output — stylesheet was mutated during first execution")
}

func TestStylesheetReuseCharacterMap(t *testing.T) {
	// A stylesheet with a character-map. The ResolvedCharMap is written
	// to ss.outputs[""] during execution. Verify repeated execution
	// produces identical results.
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="xml" use-character-maps="copy"/>
  <xsl:character-map name="copy">
    <xsl:output-character character="&#169;" string="(c)"/>
  </xsl:character-map>
  <xsl:template match="/">
    <out>&#169;</out>
  </xsl:template>
</xsl:stylesheet>`)

	source := parseTransformSource(t)

	result1, err := ss.Transform(source).Serialize(t.Context())
	require.NoError(t, err)
	require.True(t, strings.Contains(result1, "(c)"), "character map not applied: %s", result1)

	result2, err := ss.Transform(source).Serialize(t.Context())
	require.NoError(t, err)

	require.Equal(t, result1, result2,
		"second transform produced different output — stylesheet was mutated during first execution")
}

func TestStylesheetReuseConcurrent(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="xml" use-character-maps="copy"/>
  <xsl:character-map name="copy">
    <xsl:output-character character="&#169;" string="(c)"/>
  </xsl:character-map>
  <xsl:template match="/">
    <out>&#169;</out>
  </xsl:template>
</xsl:stylesheet>`)

	source, err := helium.Parse(t.Context(), []byte(`<root/>`))
	require.NoError(t, err)

	// Run multiple transforms concurrently on the same stylesheet.
	// Under -race this will detect data races on shared stylesheet state.
	var wg sync.WaitGroup
	errs := make([]error, 10)
	results := make([]string, 10)
	for i := range 10 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			r, err := ss.Transform(source).Serialize(t.Context())
			errs[idx] = err
			results[idx] = r
		}(i)
	}
	wg.Wait()

	for i := range 10 {
		require.NoError(t, errs[i], "goroutine %d failed", i)
	}

	// All results must be identical
	for i := 1; i < 10; i++ {
		require.Equal(t, results[0], results[i], "goroutine %d produced different output", i)
	}
}
