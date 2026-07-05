package xsd_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// A non-file-scheme ABSOLUTE-URI schemaLocation on an optional xs:include flows
// through ResolveSchemaURI (returned verbatim) into the direct filesystem loader
// (PermissiveRoot). os.Open of such a URI fails with a platform-dependent errno
// (ENOENT on Linux, EINVAL on macOS/Windows); PermissiveRoot classifies it as a
// resolution miss (fs.ErrNotExist) so the optional include is demoted to a
// warning-and-skip CONSISTENTLY across platforms, and the schema still compiles.
func TestNestedIncludeAbsoluteURISchemaLocationDemoted(t *testing.T) {
	t.Parallel()

	// The include target is a network URI the local FS can never serve; the
	// schema is self-sufficient once the include is skipped (root is a builtin).
	const schema = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="http://example.com/missing.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
	require.NoError(t, err)

	_, err = xsd.NewCompiler().FS(helium.PermissiveFS()).BaseDir(".").Compile(t.Context(), doc)
	require.NoError(t, err, "an optional include with an absolute-URI schemaLocation the local FS cannot serve must warn/skip, not fail compile")
}

// The same absolute-URI import miss is demoted (warn/skip), but a schema that
// genuinely DEPENDS on a component from the missing target still fails: the
// unresolved reference surfaces as a fatal compilation error rather than being
// masked by the demoted fetch. This confirms the required/strict context.
func TestNestedImportAbsoluteURIMissingDependencyFatal(t *testing.T) {
	t.Parallel()

	const schema = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           xmlns:t="urn:example:t">
  <xs:import namespace="urn:example:t" schemaLocation="http://example.com/missing.xsd"/>
  <xs:element name="root" type="t:MissingType"/>
</xs:schema>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
	require.NoError(t, err)

	_, err = xsd.NewCompiler().FS(helium.PermissiveFS()).BaseDir(".").Compile(t.Context(), doc)
	require.Error(t, err, "a schema depending on a component from a missed absolute-URI import must fail compile (the miss is not masked)")
}

// A "file:" scheme schemaLocation is a LOCAL resource: PermissiveRoot converts
// the file URI to a filesystem path before os.Open, so a valid file:/// include
// LOADS (the type it defines resolves), and a missing one yields os.Open's own
// ENOENT — a demotable resolution miss.
func TestNestedIncludeFileURILoads(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	partPath := filepath.Join(dir, "part.xsd")
	const partXSD = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="PartType">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
</xs:schema>`
	require.NoError(t, os.WriteFile(partPath, []byte(partXSD), 0o600))

	// The main schema references PartType from the file:/// include; if the
	// include is skipped instead of loaded, PartType is unresolved and compile
	// fails — so a successful compile proves the file:/// include loaded.
	mainXSD := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="` + fileURIForTestPath(partPath) + `"/>
  <xs:element name="root" type="PartType"/>
</xs:schema>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(mainXSD))
	require.NoError(t, err)

	_, err = xsd.NewCompiler().FS(helium.PermissiveFS()).BaseDir(dir).Compile(t.Context(), doc)
	require.NoError(t, err, "a valid file:/// include must load (its type must resolve), not be skipped")
}

// A MISSING file:/// include is a demotable resolution miss (os.Open ENOENT): the
// optional include is skipped and the self-sufficient schema still compiles.
func TestNestedIncludeFileURIMissingDemotes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	missingPath := filepath.Join(dir, "missing.xsd")

	mainXSD := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="` + fileURIForTestPath(missingPath) + `"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(mainXSD))
	require.NoError(t, err)

	_, err = xsd.NewCompiler().FS(helium.PermissiveFS()).BaseDir(dir).Compile(t.Context(), doc)
	require.NoError(t, err, "a missing file:/// include must demote to a warning-and-skip, not fail compile")
}

func fileURIForTestPath(path string) string {
	slashPath := filepath.ToSlash(path)
	if !strings.HasPrefix(slashPath, "/") {
		slashPath = "/" + slashPath
	}
	return "file://" + slashPath
}

// TestNestedIncludeAbsoluteURIErrInvalidDemotes verifies the FS-INDEPENDENT
// non-file-URI classification: os.DirFS (used by the W3C harness) rejects a
// non-fs.ValidPath name like "http://foo/foo" with fs.ErrInvalid, NOT
// fs.ErrNotExist. An optional include whose absolute-URI schemaLocation the FS
// reports via fs.ErrInvalid must still DEMOTE (schemaLocation is only a hint),
// exactly as it does when a URI-mapping FS reports fs.ErrNotExist.
func TestNestedIncludeAbsoluteURIErrInvalidDemotes(t *testing.T) {
	t.Parallel()

	const schema = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="http://foo/foo"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
	require.NoError(t, err)

	// openMissSchemaFS returns fs.ErrInvalid for every Open — modeling os.DirFS's
	// response to a non-ValidPath (URI-shaped) name.
	_, err = xsd.NewCompiler().FS(openMissSchemaFS{fs.ErrInvalid}).BaseDir(".").Compile(t.Context(), doc)
	require.NoError(t, err, "a non-file-URI include reported via fs.ErrInvalid must demote, not fail compile")
}

// TestNestedIncludeAbsoluteURIOpaqueOpenIsFatal verifies the scheme-demotion gate
// is NOT too broad: an Open-only URI-AWARE fs.FS that actually FETCHES a non-file
// URI and fails with an OPAQUE error (an "HTTP 500" — neither fs.ErrInvalid nor
// fs.ErrNotExist) reports a REAL fetch failure, not a local-resolution miss, so it
// must stay FATAL. Only fs.ErrInvalid/fs.ErrNotExist (what a plain FS returns for a
// URI it cannot resolve locally) demotes.
func TestNestedIncludeAbsoluteURIOpaqueOpenIsFatal(t *testing.T) {
	t.Parallel()

	const schema = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="http://example.com/inc.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
	require.NoError(t, err)

	// An Open-only FS whose Open returns an OPAQUE fetch error (no fs.ErrInvalid /
	// fs.ErrNotExist in the chain), modeling a URI-aware FS that fetched and got
	// an HTTP 500.
	opaque := errors.New("HTTP 500 for http://example.com/inc.xsd")
	_, err = xsd.NewCompiler().FS(openMissSchemaFS{opaque}).BaseDir(".").Compile(t.Context(), doc)
	require.Error(t, err, "an opaque HTTP-500 open error on a non-file URI must be fatal, not demoted")
}
