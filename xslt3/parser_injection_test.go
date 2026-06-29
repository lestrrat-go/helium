package xslt3_test

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// injectionResolver serves a fixed set of URIs from an in-memory map.
type injectionResolver struct {
	files map[string]string
}

func (r *injectionResolver) Resolve(uri string) (io.ReadCloser, error) {
	body, ok := r.files[uri]
	if !ok {
		body, ok = r.files[filepath.FromSlash(uri)]
	}
	if !ok {
		return nil, io.EOF
	}
	return io.NopCloser(strings.NewReader(body)), nil
}

func (r *injectionResolver) ResolveURI(uri string) (io.ReadCloser, error) {
	return r.Resolve(uri)
}

// TestParserInjectionGovernsRuntimeParse proves that a helium.Parser injected
// via Compiler.Parser governs the parse policy of the runtime document parse
// performed by fn:doc: a parser with MaxNameLength(8) rejects a document whose
// element name exceeds 8 bytes, while the default compiler accepts it.
func TestParserInjectionGovernsRuntimeParse(t *testing.T) {
	t.Parallel()

	const stylesheet = `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:param name="url"/>
  <xsl:template match="/">
    <out><xsl:value-of select="doc($url)/*/local-name()"/></out>
  </xsl:template>
</xsl:stylesheet>`

	// The runtime document's root element name "longelementname" is 15 bytes,
	// exceeding MaxNameLength(8) but accepted by the default parser.
	const docBody = `<?xml version="1.0"?><longelementname/>`
	docPath := filepath.FromSlash("/docs/in.xml")

	resolver := &injectionResolver{files: map[string]string{docPath: docBody}}

	compileAndRun := func(t *testing.T, c xslt3.Compiler) (string, error) {
		t.Helper()
		ssDoc, err := helium.NewParser().Parse(t.Context(), []byte(stylesheet))
		require.NoError(t, err)
		ss, err := c.Compile(t.Context(), ssDoc)
		require.NoError(t, err)
		source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
		require.NoError(t, err)
		return ss.Transform(source).
			URIResolver(resolver).
			SetParameter("url", xpath3.SingleString(docPath)).
			Serialize(t.Context())
	}

	t.Run("default parser accepts long name", func(t *testing.T) {
		t.Parallel()
		out, err := compileAndRun(t, xslt3.NewCompiler())
		require.NoError(t, err)
		require.Contains(t, out, "longelementname")
	})

	t.Run("injected parser MaxNameLength enforced", func(t *testing.T) {
		t.Parallel()
		c := xslt3.NewCompiler().Parser(helium.NewParser().MaxNameLength(8))
		out, err := compileAndRun(t, c)
		// The injected limit must reach the runtime fn:doc parse: either the parse
		// fails outright, or the over-limit name never surfaces in the output.
		if err == nil {
			require.NotContains(t, out, "longelementname",
				"injected MaxNameLength must govern the runtime fn:doc parse")
		}
	})
}
