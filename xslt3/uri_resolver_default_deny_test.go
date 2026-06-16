package xslt3_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// recordingURIResolver records every URI it is asked to resolve and serves
// the bytes registered for that URI. It lets a test prove that a runtime XSLT
// instruction routed its retrieval through Invocation.URIResolver instead of
// reading the host filesystem directly.
type recordingURIResolver struct {
	mu       sync.Mutex
	requests []string
	files    map[string][]byte
}

func (r *recordingURIResolver) ResolveURI(uri string) (io.ReadCloser, error) {
	r.mu.Lock()
	r.requests = append(r.requests, uri)
	r.mu.Unlock()
	data, ok := r.files[uri]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(strings.NewReader(string(data))), nil
}

func (r *recordingURIResolver) seen(uri string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, got := range r.requests {
		if got == uri {
			return true
		}
	}
	return false
}

// writeTempXML writes an XML file into the test's temp dir and returns its
// absolute path.
func writeTempXML(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

const sourceDocStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:source-document href="%HREF%">
      <out><xsl:value-of select="/data/@v"/></out>
    </xsl:source-document>
  </xsl:template>
</xsl:stylesheet>`

// TestSourceDocumentDefaultDeny verifies xsl:source-document does NOT read the
// host filesystem when no URIResolver is installed: even though the file
// physically exists, retrieval must be denied with FODC0002.
func TestSourceDocumentDefaultDeny(t *testing.T) {
	path := writeTempXML(t, "data.xml", `<data v="hello"/>`)

	ss := compileStylesheetString(t, strings.ReplaceAll(sourceDocStylesheet, "%HREF%", path))
	source := parseTransformSource(t)

	_, err := ss.Transform(source).Serialize(t.Context())
	require.Error(t, err, "source-document must default-deny without a URIResolver")
	require.Contains(t, err.Error(), "FODC0002")
}

// TestSourceDocumentRoutesThroughResolver verifies that with a recording
// resolver installed, xsl:source-document retrieves its document through the
// resolver (receiving the resolved URI) rather than via os.ReadFile.
func TestSourceDocumentRoutesThroughResolver(t *testing.T) {
	path := writeTempXML(t, "data.xml", `<data v="hello"/>`)

	resolver := &recordingURIResolver{files: map[string][]byte{
		path: []byte(`<data v="hello"/>`),
	}}

	ss := compileStylesheetString(t, strings.ReplaceAll(sourceDocStylesheet, "%HREF%", path))
	source := parseTransformSource(t)

	result, err := ss.Transform(source).URIResolver(resolver).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, "<out>hello</out>")
	require.True(t, resolver.seen(path), "resolver should have been asked to resolve %q; got %v", path, resolver.requests)
}

const streamAvailableStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out><xsl:value-of select="stream-available('%HREF%')"/></out>
  </xsl:template>
</xsl:stylesheet>`

// TestStreamAvailableDefaultDeny verifies fn:stream-available returns false
// (rather than stat-ing the host filesystem) when no URIResolver is installed,
// even though the referenced file exists on disk.
func TestStreamAvailableDefaultDeny(t *testing.T) {
	path := writeTempXML(t, "data.xml", `<data/>`)

	ss := compileStylesheetString(t, strings.ReplaceAll(streamAvailableStylesheet, "%HREF%", path))
	source := parseTransformSource(t)

	result, err := ss.Transform(source).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, "<out>false</out>", "stream-available must report false without a resolver")
}

// TestStreamAvailableRoutesThroughResolver verifies fn:stream-available probes
// availability via the installed URIResolver and returns true for an XML
// resource it can retrieve.
func TestStreamAvailableRoutesThroughResolver(t *testing.T) {
	path := writeTempXML(t, "data.xml", `<data/>`)

	resolver := &recordingURIResolver{files: map[string][]byte{
		path: []byte(`<data/>`),
	}}

	ss := compileStylesheetString(t, strings.ReplaceAll(streamAvailableStylesheet, "%HREF%", path))
	source := parseTransformSource(t)

	result, err := ss.Transform(source).URIResolver(resolver).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, "<out>true</out>")
	require.True(t, resolver.seen(path), "resolver should have been asked to resolve %q; got %v", path, resolver.requests)
}
