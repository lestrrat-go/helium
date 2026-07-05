package xsd_test

import (
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
