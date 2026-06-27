package xslt3_test

import (
	"context"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// analyzeStringStylesheet builds an xsl:analyze-string stylesheet using the
// given regex; the matching and non-matching substrings are emitted verbatim
// so output is easy to assert.
func analyzeStringStylesheet(regex string) string {
	return `<?xml version="1.0"?>` +
		`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
		`<xsl:output method="xml" omit-xml-declaration="yes"/>` +
		`<xsl:template match="/"><out>` +
		`<xsl:analyze-string select="string(.)" regex="` + regex + `">` +
		`<xsl:matching-substring><m><xsl:value-of select="."/></m></xsl:matching-substring>` +
		`<xsl:non-matching-substring><n><xsl:value-of select="."/></n></xsl:non-matching-substring>` +
		`</xsl:analyze-string>` +
		`</out></xsl:template>` +
		`</xsl:stylesheet>`
}

// A normal xsl:analyze-string still produces correct alternating
// matching / non-matching output. This pins byte-identical behavior across the
// incremental-processing change.
func TestAnalyzeStringNormalOutput(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(analyzeStringStylesheet("[0-9]")))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc>a1b2c3</doc>`))
	require.NoError(t, err)

	result, err := ss.Transform(source).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result,
		"<out><n>a</n><m>1</m><n>b</n><m>2</m><n>c</n><m>3</m></out>")
}

// An xsl:analyze-string with an empty-matching regex over a large input matches
// at every character boundary, amplifying a bounded input string into an
// unbounded number of match/segment allocations. The work must be bounded
// against the execution resource budget (MaxResourceBytes) and fail with
// ErrResourceTooLarge rather than exhausting memory.
func TestAnalyzeStringEmptyMatchIsCapped(t *testing.T) {
	t.Parallel()

	// regex "x*" matches a zero-length string at every position of an all-'a'
	// input, so an L-char input yields L+1 matches.
	doc, err := helium.NewParser().Parse(t.Context(), []byte(analyzeStringStylesheet("x*")))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<doc>`+strings.Repeat("a", 5000)+`</doc>`))
	require.NoError(t, err)

	// Cap well below the resulting match count; the breach must surface
	// ErrResourceTooLarge through the dynamic-error wrapping.
	_, err = ss.Transform(source).
		MaxResourceBytes(1000).
		Serialize(t.Context())
	require.Error(t, err)
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge,
		"empty-matching xsl:analyze-string over a large input must honor the resource cap")
	require.ErrorIs(t, err, xslt3.ErrDynamicError,
		"the analyze-string cap breach is a runtime (dynamic) error")
}

// A multi-line "^" (flags="m") is a leading-context anchor that matches at every
// line start, so an input of N newlines yields ~N matches. Unlike "x*", this
// pattern cannot stream incrementally on RE2; it is matched in one bounded
// FindAll pass whose limit is the cap, so the cap is enforced without first
// materializing every line-start match.
func TestAnalyzeStringMultilineAnchorIsCapped(t *testing.T) {
	t.Parallel()

	stylesheet := `<?xml version="1.0"?>` +
		`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
		`<xsl:output method="xml" omit-xml-declaration="yes"/>` +
		`<xsl:template match="/"><out>` +
		`<xsl:analyze-string select="string(.)" regex="^" flags="m">` +
		`<xsl:matching-substring><m/></xsl:matching-substring>` +
		`<xsl:non-matching-substring><n><xsl:value-of select="."/></n></xsl:non-matching-substring>` +
		`</xsl:analyze-string>` +
		`</out></xsl:template>` +
		`</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(stylesheet))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)

	// A long run of newlines: each line start is a (zero-length) match, so the
	// match count grows with the input far past the cap below.
	source, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<doc>`+strings.Repeat("\n", 5000)+`</doc>`))
	require.NoError(t, err)

	_, err = ss.Transform(source).
		MaxResourceBytes(1000).
		Serialize(t.Context())
	require.Error(t, err)
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge,
		"a multiline-anchor xsl:analyze-string over a large input must honor the resource cap")
	require.ErrorIs(t, err, xslt3.ErrDynamicError,
		"the analyze-string cap breach is a runtime (dynamic) error")
}

// A cancelled context is honored promptly by xsl:analyze-string rather than
// running its per-segment loop to completion.
func TestAnalyzeStringHonorsCancelledContext(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(analyzeStringStylesheet("x*")))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<doc>`+strings.Repeat("a", 5000)+`</doc>`))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err = ss.Transform(source).Serialize(ctx)
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled,
		"a cancelled context must be honored during xsl:analyze-string")
}
