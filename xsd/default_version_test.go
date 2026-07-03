package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestDefaultVersion verifies Compiler.DefaultVersion: it is the fallback used
// only when no version is forced via Version() and the schema declares no
// vc:minVersion hint. A forced Version() and a vc:minVersion hint both take
// precedence. The selected version is probed via the 1.1-only "+INF" lexical
// for xs:double (accepted only under 1.1).
func TestDefaultVersion(t *testing.T) {
	t.Parallel()

	const versioningNS = `xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning"`

	compile := func(t *testing.T, c xsd.Compiler, schemaXML string) (*xsd.Schema, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		return c.Compile(t.Context(), doc)
	}
	validate := func(t *testing.T, schema *xsd.Schema, instance string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), doc)
	}

	// A schema silent on version (no vc:minVersion, no forced Version()).
	plain := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="v" type="xs:double"/>
</xs:schema>`
	// A schema declaring vc:minVersion="1.1".
	hinted := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" ` + versioningNS + ` vc:minVersion="1.1">
  <xs:element name="v" type="xs:double"/>
</xs:schema>`

	t.Run("DefaultVersion(Version11) upgrades a schema silent on version", func(t *testing.T) {
		t.Parallel()
		s, err := compile(t, xsd.NewCompiler().DefaultVersion(xsd.Version11), plain)
		require.NoError(t, err)
		require.NoError(t, validate(t, s, `<v>+INF</v>`))
	})

	t.Run("standalone default stays Version10 (byte-identical)", func(t *testing.T) {
		t.Parallel()
		s, err := compile(t, xsd.NewCompiler(), plain)
		require.NoError(t, err)
		require.Error(t, validate(t, s, `<v>+INF</v>`))
	})

	t.Run("DefaultVersion(Version10) keeps a silent schema at 1.0", func(t *testing.T) {
		t.Parallel()
		s, err := compile(t, xsd.NewCompiler().DefaultVersion(xsd.Version10), plain)
		require.NoError(t, err)
		require.Error(t, validate(t, s, `<v>+INF</v>`))
	})

	t.Run("forced Version(Version10) wins over DefaultVersion(Version11)", func(t *testing.T) {
		t.Parallel()
		s, err := compile(t, xsd.NewCompiler().DefaultVersion(xsd.Version11).Version(xsd.Version10), plain)
		require.NoError(t, err)
		require.Error(t, validate(t, s, `<v>+INF</v>`))
	})

	t.Run("vc:minVersion hint wins over DefaultVersion(Version10)", func(t *testing.T) {
		t.Parallel()
		s, err := compile(t, xsd.NewCompiler().DefaultVersion(xsd.Version10), hinted)
		require.NoError(t, err)
		require.NoError(t, validate(t, s, `<v>+INF</v>`))
	})
}
