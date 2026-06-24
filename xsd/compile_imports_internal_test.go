package xsd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestResolveSchemaURI verifies the single canonical URI-resolution helper:
// an absolute-URI ref is passed through untouched (never filepath.Join'ed,
// which would collapse "//" and drop the host), a relative ref against a URI
// base resolves per RFC 3986, and a no-authority base preserves OmitHost so
// "mem:/..." stays "mem:/..." (not "mem:///...") while "file:///..." keeps its
// "///".
func TestResolveSchemaURI(t *testing.T) {
	for _, tc := range []struct {
		name string
		base string
		ref  string
		want string
	}{
		// Absolute-URI ref: returned verbatim regardless of base.
		{"absolute cross-host", httpsMain, "https://cdn.example.com/part.xsd", "https://cdn.example.com/part.xsd"},
		{"absolute file location", httpsMain, "file:///etc/x.xsd", "file:///etc/x.xsd"},
		{"absolute opaque ref, uri base", httpsMain, "urn:schemas:s", "urn:schemas:s"},
		{"absolute single-slash ref, uri base", httpsMain, "mem:/o/s.xsd", "mem:/o/s.xsd"},
		// Relative ref against an http authority base: RFC 3986 resolution.
		{"https sibling", httpsMain, part, "https://example.com/s/part.xsd"},
		{"https subdir", httpsMain, "sub/part.xsd", "https://example.com/s/sub/part.xsd"},
		{"https parent", "https://example.com/s/sub/main.xsd", "../part.xsd", "https://example.com/s/part.xsd"},
		{"https root-relative", httpsMain, "/p/s.xsd", "https://example.com/p/s.xsd"},
		// Relative ref against a file:/// base: canonical "///" preserved.
		{"file sibling keeps ///", "file:///tmp/s/main.xsd", part, "file:///tmp/s/part.xsd"},
		{"file parent keeps ///", "file:///tmp/s/sub/main.xsd", "../part.xsd", "file:///tmp/s/part.xsd"},
		{"file root-relative keeps ///", "file:///tmp/s/main.xsd", "/p/s.xsd", "file:///p/s.xsd"},
		// No-authority single-slash base (OmitHost): "mem:/..." must stay
		// "mem:/...", never gain an empty "//" authority ("mem:///...").
		{"mem sibling", "mem:/schemas/main.xsd", part, "mem:/schemas/part.xsd"},
		{"mem subdir", "mem:/schemas/main.xsd", "sub/part.xsd", "mem:/schemas/sub/part.xsd"},
		{"mem parent", "mem:/schemas/sub/main.xsd", "../part.xsd", "mem:/schemas/part.xsd"},
		{"mem root-relative", "mem:/schemas/main.xsd", "/p/s.xsd", "mem:/p/s.xsd"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveSchemaURI(tc.ref, tc.base)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

// TestURIScheme confirms the shared scheme-detector: multi-character schemes
// count, bare local paths and single-letter Windows drive letters do not.
func TestURIScheme(t *testing.T) {
	require.Equal(t, "https", URIScheme("https://x/y"))
	require.Equal(t, "mem", URIScheme("mem:/x"))
	require.Equal(t, "file", URIScheme("file:///x"))
	require.Equal(t, "urn", URIScheme("urn:a:b"))
	require.Equal(t, "", URIScheme("/tmp/x"))
	require.Equal(t, "", URIScheme("part.xsd"))
	require.Equal(t, "", URIScheme(`C:\x`))
}

// TestValidateSchemaPathURIAware verifies that xs:include/xs:import/xs:redefine
// schema-location resolution (via the validateSchemaPath wrapper) is URI-aware.
func TestValidateSchemaPathURIAware(t *testing.T) {
	for _, tc := range []struct {
		name    string
		baseDir string
		loc     string
		want    string
	}{
		{"absolute cross-host", httpsMain, "https://cdn.example.com/part.xsd", "https://cdn.example.com/part.xsd"},
		{"relative sibling", httpsMain, part, "https://example.com/s/part.xsd"},
		{"mem sibling preserves OmitHost", "mem:/schemas/main.xsd", part, "mem:/schemas/part.xsd"},
		{"file relative parent", "file:///tmp/s/sub/main.xsd", "../part.xsd", "file:///tmp/s/part.xsd"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateSchemaPath(tc.baseDir, tc.loc)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

// part is the relative schema-location used in these cases.
const part = "part.xsd"

// httpsMain is the canonical https base schema URI reused across cases.
const httpsMain = "https://example.com/s/main.xsd"

// TestValidateSchemaPathLocalUnchanged verifies that genuine local-filesystem
// bases keep their historical filepath.Join behavior and ".."-escape guard, so
// existing xsd public-API callers and the golden/IDC suite are unaffected.
func TestValidateSchemaPathLocalUnchanged(t *testing.T) {
	t.Run("relative join", func(t *testing.T) {
		got, err := validateSchemaPath("/tmp/s", part)
		require.NoError(t, err)
		// ResolveSchemaURI resolves in forward-slash (fs.FS-key) space, so the
		// result is slash-separated on every OS — never filepath.Join's "\tmp\s".
		require.Equal(t, "/tmp/s/"+part, got)
	})
	t.Run("empty base cleans", func(t *testing.T) {
		got, err := validateSchemaPath("", "./a/../part.xsd")
		require.NoError(t, err)
		require.Equal(t, part, got)
	})
	t.Run("escape denied", func(t *testing.T) {
		_, err := validateSchemaPath("/tmp/s", "../../etc/passwd")
		require.ErrorIs(t, err, errSchemaPathEscape)
	})
	t.Run("absolute local location lands inside base", func(t *testing.T) {
		got, err := validateSchemaPath("/tmp/s", "/etc/passwd")
		require.NoError(t, err)
		require.Equal(t, "/tmp/s/etc/passwd", got)
	})
	// Windows-shaped fixtures (plain strings) exercise the forward-slash
	// resolution and the escape guard on Linux. The returned name is an fs.FS
	// key, which must be slash-separated on every OS — never "schemas\\x".
	t.Run("windows-shaped base joins with forward slashes", func(t *testing.T) {
		got, err := validateSchemaPath(`schemas\sub`, "part.xsd")
		require.NoError(t, err)
		require.Equal(t, "schemas/sub/part.xsd", got)
	})
	t.Run("windows-shaped backslash escape denied", func(t *testing.T) {
		_, err := validateSchemaPath("schemas", `..\..\etc\passwd`)
		require.ErrorIs(t, err, errSchemaPathEscape)
	})
}

// TestSchemaBaseDir verifies the nested-include base used for the import
// sub-compiler: a URI loc is the base itself (RFC 3986 replaces the last path
// segment); a local path uses its containing directory.
func TestSchemaBaseDir(t *testing.T) {
	require.Equal(t, "https://example.com/s/main.xsd", schemaBaseDir("https://example.com/s/main.xsd"))
	require.Equal(t, "file:///tmp/s/main.xsd", schemaBaseDir("file:///tmp/s/main.xsd"))
	// schemaBaseDir uses path.Dir (slash space), so the parent is slash-separated
	// on every OS — never filepath.Dir's "\tmp\s" on Windows.
	require.Equal(t, "/tmp/s", schemaBaseDir("/tmp/s/main.xsd"))
	// An fs.FS-key loc is slash-separated; its parent stays slash-separated on
	// every OS (a backslash-shaped input is normalized, exercising Windows on Linux).
	require.Equal(t, "schemas", schemaBaseDir(`schemas\intermediate.xsd`))
}
