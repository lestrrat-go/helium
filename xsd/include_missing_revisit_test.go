package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestMissingIncludeRevisitedWarnsConsistently verifies that a missing
// xs:include/xs:redefine schemaLocation demoted to a warning does not pollute
// the loaded-set: a SECOND include/redefine of the SAME missing location is
// handled consistently (warned about / skipped again), not silently swallowed
// nor — for xs:redefine — turned into a spurious "does already exist"
// duplicate-redefine error against an empty Phase-A set. The load failure must
// roll back the includeVisited marker that was added before the read attempt.
func TestMissingIncludeRevisitedWarnsConsistently(t *testing.T) {
	t.Parallel()

	const mainXSD = "main.xsd"

	compileWith := func(t *testing.T, src string) (warnings, errStr string) {
		t.Helper()
		fsys := fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(src)},
			// missing.xsd is intentionally absent so the loader gets
			// fs.ErrNotExist (a non-fatal hint miss demoted to a warning).
		}
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label(mainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		requireCompileResultErr(t, err)
		require.NoError(t, collector.Close())
		w, e := partitionCompileErrors(collector.Errors())
		return w, e
	}

	t.Run("two missing includes both warn", func(t *testing.T) {
		t.Parallel()
		warnings, errStr := compileWith(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="missing.xsd"/>
  <xs:include schemaLocation="missing.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)
		require.Empty(t, errStr, "a missing include hint must not be a fatal error")
		// Each include element must independently warn; the second must not be
		// silently skipped as "already loaded".
		require.Equal(t, 2, strings.Count(warnings, "Skipping the include."),
			"both missing includes must warn; got: %q", warnings)
	})

	t.Run("two missing redefines both warn without spurious duplicate error", func(t *testing.T) {
		t.Parallel()
		warnings, errStr := compileWith(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="missing.xsd">
    <xs:simpleType name="codeType">
      <xs:restriction base="codeType"><xs:maxLength value="5"/></xs:restriction>
    </xs:simpleType>
  </xs:redefine>
  <xs:redefine schemaLocation="missing.xsd">
    <xs:simpleType name="codeType">
      <xs:restriction base="codeType"><xs:maxLength value="3"/></xs:restriction>
    </xs:simpleType>
  </xs:redefine>
</xs:schema>`)
		// The second redefine of the same missing document must NOT be turned
		// into a duplicate-component error against an empty Phase-A set.
		require.NotContains(t, errStr, "does already exist",
			"a missing redefine hint must not produce a spurious duplicate-redefine error; got: %q", errStr)
		require.Empty(t, errStr, "a missing redefine hint must not be a fatal error; got: %q", errStr)
		require.Equal(t, 2, strings.Count(warnings, "Skipping the redefine."),
			"both missing redefines must warn; got: %q", warnings)
	})
}
