package xsd

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestValidateSchemaPathURIAware verifies that xs:include/xs:import/xs:redefine
// schema-location resolution is URI-aware: an absolute-URI location is passed
// through untouched (never filepath.Join'ed, which would collapse "//" and drop
// the host), and a relative location against a URI base resolves per RFC 3986.
func TestValidateSchemaPathURIAware(t *testing.T) {
	for _, tc := range []struct {
		name    string
		baseDir string
		loc     string
		want    string
	}{
		// Absolute-URI location: returned verbatim regardless of base.
		{"absolute cross-host", "https://example.com/s/main.xsd", "https://cdn.example.com/part.xsd", "https://cdn.example.com/part.xsd"},
		{"absolute file location", "https://example.com/s/main.xsd", "file:///etc/x.xsd", "file:///etc/x.xsd"},
		// Relative location against an http base: RFC 3986 resolution.
		{"relative sibling", "https://example.com/s/main.xsd", part, "https://example.com/s/part.xsd"},
		{"relative subdir", "https://example.com/s/main.xsd", "sub/part.xsd", "https://example.com/s/sub/part.xsd"},
		{"relative parent", "https://example.com/s/sub/main.xsd", "../part.xsd", "https://example.com/s/part.xsd"},
		// Relative location against a file:// base.
		{"file relative sibling", "file:///tmp/s/main.xsd", part, "file:///tmp/s/part.xsd"},
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

// TestValidateSchemaPathLocalUnchanged verifies that genuine local-filesystem
// bases keep their historical filepath.Join behavior and ".."-escape guard, so
// existing xsd public-API callers and the golden/IDC suite are unaffected.
func TestValidateSchemaPathLocalUnchanged(t *testing.T) {
	t.Run("relative join", func(t *testing.T) {
		got, err := validateSchemaPath("/tmp/s", part)
		require.NoError(t, err)
		require.Equal(t, filepath.Join("/tmp/s", part), got)
	})
	t.Run("empty base cleans", func(t *testing.T) {
		got, err := validateSchemaPath("", "./a/../part.xsd")
		require.NoError(t, err)
		require.Equal(t, filepath.Clean("./a/../part.xsd"), got)
	})
	t.Run("escape denied", func(t *testing.T) {
		_, err := validateSchemaPath("/tmp/s", "../../etc/passwd")
		require.ErrorIs(t, err, errSchemaPathEscape)
	})
	t.Run("absolute local location lands inside base", func(t *testing.T) {
		got, err := validateSchemaPath("/tmp/s", "/etc/passwd")
		require.NoError(t, err)
		require.Equal(t, filepath.Join("/tmp/s", "/etc/passwd"), got)
	})
}

// TestSchemaBaseDir verifies the nested-include base used for the import
// sub-compiler: a URI loc is the base itself (RFC 3986 replaces the last path
// segment); a local path uses its containing directory.
func TestSchemaBaseDir(t *testing.T) {
	require.Equal(t, "https://example.com/s/main.xsd", schemaBaseDir("https://example.com/s/main.xsd"))
	require.Equal(t, "file:///tmp/s/main.xsd", schemaBaseDir("file:///tmp/s/main.xsd"))
	require.Equal(t, filepath.Dir("/tmp/s/main.xsd"), schemaBaseDir("/tmp/s/main.xsd"))
}
