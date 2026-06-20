package xpath3_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// MaxResourceBytes bounds resolver-backed reads from fn:unparsed-text and
// fn:json-doc so a hostile resource cannot exhaust memory.
func TestMaxResourceBytesUnparsedText(t *testing.T) {
	t.Parallel()
	compiled, err := xpath3.NewCompiler().Compile(`unparsed-text("data.txt")`)
	require.NoError(t, err)

	res := testURIResolver{"http://example.com/base/data.txt": strings.Repeat("a", 100)}

	t.Run("over limit fails", func(t *testing.T) {
		_, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			BaseURI("http://example.com/base/").
			URIResolver(res).
			MaxResourceBytes(10).
			Evaluate(t.Context(), compiled, nil)
		require.Error(t, err)
		var xpErr *xpath3.XPathError
		require.ErrorAs(t, err, &xpErr)
		require.Equal(t, "FOUT1170", xpErr.Code)
	})

	t.Run("under limit succeeds", func(t *testing.T) {
		result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			BaseURI("http://example.com/base/").
			URIResolver(res).
			MaxResourceBytes(1000).
			Evaluate(t.Context(), compiled, nil)
		require.NoError(t, err)
		s, ok := result.IsString()
		require.True(t, ok)
		require.Len(t, s, 100)
	})
}

func TestMaxResourceBytesJSONDoc(t *testing.T) {
	t.Parallel()
	compiled, err := xpath3.NewCompiler().Compile(`json-doc("data.json")?name`)
	require.NoError(t, err)

	// A small valid JSON object padded past the cap with whitespace.
	body := `{"name":"helium"}` + strings.Repeat(" ", 100)
	res := testURIResolver{"http://example.com/base/data.json": body}

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		BaseURI("http://example.com/base/").
		URIResolver(res).
		MaxResourceBytes(10).
		Evaluate(t.Context(), compiled, nil)
	require.Error(t, err)
	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "FODC0002", xpErr.Code)
}

// H1: fn:doc / fn:unparsed-text must be secure-by-default — no implicit
// HTTP fetches, no implicit os.ReadFile reads, no XXE in fetched docs.

func TestFnDocNoNetworkByDefault(t *testing.T) {
	t.Parallel()
	// If any request reaches the server, the test fails.
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte("<root/>"))
	}))
	defer srv.Close()

	compiled, err := xpath3.NewCompiler().Compile(`doc("` + srv.URL + `/x")`)
	require.NoError(t, err)

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Evaluate(t.Context(), compiled, nil)
	require.Error(t, err)

	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "FODC0002", xpErr.Code)
	require.Zero(t, hits.Load(), "no HTTP request should be issued by default")
}

func TestFnDocNoFileReadByDefault(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.xml")
	require.NoError(t, os.WriteFile(path, []byte("<root>secret</root>"), 0644))

	compiled, err := xpath3.NewCompiler().Compile(`doc("` + path + `")`)
	require.NoError(t, err)

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Evaluate(t.Context(), compiled, nil)
	require.Error(t, err)

	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "FODC0002", xpErr.Code)
}

func TestFnDocNoFileURIReadByDefault(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.xml")
	require.NoError(t, os.WriteFile(path, []byte("<root>secret</root>"), 0644))

	compiled, err := xpath3.NewCompiler().Compile(`doc("file://` + path + `")`)
	require.NoError(t, err)

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Evaluate(t.Context(), compiled, nil)
	require.Error(t, err)

	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "FODC0002", xpErr.Code)
}

func TestFnUnparsedTextNoNetworkByDefault(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte("hello"))
	}))
	defer srv.Close()

	compiled, err := xpath3.NewCompiler().Compile(`unparsed-text("` + srv.URL + `/x")`)
	require.NoError(t, err)

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Evaluate(t.Context(), compiled, nil)
	require.Error(t, err)

	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "FOUT1170", xpErr.Code)
	require.Zero(t, hits.Load(), "no HTTP request should be issued by default")
}

func TestFnUnparsedTextNoFileReadByDefault(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.txt")
	require.NoError(t, os.WriteFile(path, []byte("secret"), 0644))

	compiled, err := xpath3.NewCompiler().Compile(`unparsed-text("file://` + path + `")`)
	require.NoError(t, err)

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Evaluate(t.Context(), compiled, nil)
	require.Error(t, err)

	var xpErr *xpath3.XPathError
	require.ErrorAs(t, err, &xpErr)
	require.Equal(t, "FOUT1170", xpErr.Code)
}

func TestFnDocWithExplicitHTTPClientSucceeds(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<root><name>helium</name></root>`))
	}))
	defer srv.Close()

	compiled, err := xpath3.NewCompiler().Compile(`string(doc("` + srv.URL + `/x")/root/name)`)
	require.NoError(t, err)

	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		HTTPClient(srv.Client()).
		Evaluate(t.Context(), compiled, nil)
	require.NoError(t, err)
	s, ok := result.IsString()
	require.True(t, ok)
	require.Equal(t, "helium", s)
}

// When fn:doc retrieves an XML doc that declares an external entity,
// the parser used by loadDoc must NOT load the external entity.
func TestFnDocBlocksXXEInRetrievedDoc(t *testing.T) {
	t.Parallel()
	// Create a secret file the external entity would try to read.
	dir := t.TempDir()
	secretPath := filepath.Join(dir, "secret.txt")
	require.NoError(t, os.WriteFile(secretPath, []byte("PWNED"), 0644))

	// Resource doc declares an external entity pointing at the secret.
	resourceBody := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY xxe SYSTEM "file://` + secretPath + `">
]>
<root>&xxe;</root>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(resourceBody))
	}))
	defer srv.Close()

	compiled, err := xpath3.NewCompiler().Compile(`string(doc("` + srv.URL + `/x")/root)`)
	require.NoError(t, err)

	result, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		HTTPClient(srv.Client()).
		Evaluate(t.Context(), compiled, nil)
	// Either the parse errors out (acceptable) or it succeeds with the
	// entity *not* expanded. Either way, "PWNED" must not appear.
	if err == nil {
		s, ok := result.IsString()
		require.True(t, ok)
		require.False(t, strings.Contains(s, "PWNED"),
			"BlockXXE must prevent external entity expansion, got: %q", s)
	}
}
