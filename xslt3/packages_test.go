package xslt3_test

import (
	"io"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// modeAllPackageResolver serves a single fixed package source for any name.
type modeAllPackageResolver struct {
	source string
}

func (r modeAllPackageResolver) ResolvePackage(_ string, _ string) (io.ReadCloser, string, error) {
	return io.NopCloser(strings.NewReader(r.source)), "", nil
}

// A used package exposes a public template match="b" mode="#all". The using
// stylesheet applies templates to <b> in a named mode "m"; the package template
// must be eligible (via the #all fallback) even though "m" is not the default
// mode.
func TestUsePackageModeAllTemplateAppliedInNamedMode(t *testing.T) {
	t.Parallel()

	pkg := `<?xml version="1.0"?>
<xsl:package name="http://example.com/pkg" package-version="1.0" version="3.0"
             declared-modes="no"
             xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:expose component="mode" names="*" visibility="public"/>
  <xsl:template match="b" mode="#all">
    <hit-all/>
  </xsl:template>
</xsl:package>`

	using := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:use-package name="http://example.com/pkg"/>
  <xsl:template match="/">
    <out>
      <named><xsl:apply-templates select="//b" mode="m"/></named>
      <default><xsl:apply-templates select="//b"/></default>
    </out>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(using))
	require.NoError(t, err)

	ss, err := xslt3.NewCompiler().
		PackageResolver(modeAllPackageResolver{source: pkg}).
		Compile(t.Context(), doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(t.Context(), []byte(`<root><b/></root>`))
	require.NoError(t, err)

	result, err := ss.Transform(src).Serialize(t.Context())
	require.NoError(t, err)

	// The #all package template must fire in both the named mode "m" and the
	// default mode.
	require.Equal(t, 2, strings.Count(result, "<hit-all"),
		"package template match=\"b\" mode=\"#all\" should fire in named mode and default mode; got: %s", result)
}
