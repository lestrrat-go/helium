package relaxng_test

import (
	"os"
	"path/filepath"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

const rngNS = "http://relaxng.org/ns/structure/1.0"

// compileInterleave compiles schema and returns the concatenated fatal
// compile-error text.
func compileInterleave(t *testing.T, schema string) string {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
	require.NoError(t, err)

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, err = relaxng.NewCompiler().Label("test.rng").ErrorHandler(collector).Compile(t.Context(), doc)
	require.NoError(t, err)
	_ = collector.Close()
	_, compileErrors := partitionCompileErrors(collector.Errors())
	return compileErrors
}

// TestInterleaveNameClassConflict covers the compile-time RELAX NG §7.4
// interleave conflict check: overlapping element/text name classes and
// overlapping attribute name classes across interleave branches are fatal
// compile errors, reported in libxml2's exact wording.
func TestInterleaveNameClassConflict(t *testing.T) {
	t.Parallel()

	const elemConflictMsg = "element interleave: Relax-NG parser error : Element or text conflicts in interleave"
	const attrConflictMsg = "element interleave: Relax-NG parser error : Attributes conflicts in interleave"

	t.Run("two same-name element branches conflict", func(t *testing.T) {
		t.Parallel()
		errs := compileInterleave(t, `<element name="root" xmlns="`+rngNS+`">
  <interleave>
    <element name="a"><empty/></element>
    <element name="a"><text/></element>
  </interleave>
</element>`)
		require.Contains(t, errs, elemConflictMsg)
	})

	t.Run("text in two branches conflicts", func(t *testing.T) {
		t.Parallel()
		errs := compileInterleave(t, `<element name="root" xmlns="`+rngNS+`">
  <interleave>
    <text/>
    <text/>
  </interleave>
</element>`)
		require.Contains(t, errs, elemConflictMsg)
	})

	t.Run("mixed whose group contains text conflicts", func(t *testing.T) {
		t.Parallel()
		errs := compileInterleave(t, `<element name="root" xmlns="`+rngNS+`">
  <mixed>
    <element name="a"><empty/></element>
    <text/>
  </mixed>
</element>`)
		require.Contains(t, errs, elemConflictMsg)
	})

	t.Run("anyName element vs named element conflicts", func(t *testing.T) {
		t.Parallel()
		errs := compileInterleave(t, `<element name="root" xmlns="`+rngNS+`">
  <interleave>
    <element><anyName/><empty/></element>
    <element name="a"><empty/></element>
  </interleave>
</element>`)
		require.Contains(t, errs, elemConflictMsg)
	})

	t.Run("conflict reached through ref", func(t *testing.T) {
		t.Parallel()
		errs := compileInterleave(t, `<grammar xmlns="`+rngNS+`">
  <start>
    <element name="root">
      <interleave>
        <element name="a"><empty/></element>
        <ref name="optA"/>
      </interleave>
    </element>
  </start>
  <define name="optA">
    <optional><element name="a"><text/></element></optional>
  </define>
</grammar>`)
		require.Contains(t, errs, elemConflictMsg)
	})

	t.Run("combine interleave defines flags second define's line", func(t *testing.T) {
		t.Parallel()
		// The second <define> below starts on line 6.
		errs := compileInterleave(t, `<grammar xmlns="`+rngNS+`">
  <start><ref name="foo"/></start>
  <define name="foo" combine="interleave">
    <element name="a"><empty/></element>
  </define>
  <define name="foo" combine="interleave">
    <element name="a"><text/></element>
  </define>
</grammar>`)
		require.Contains(t, errs, elemConflictMsg)
		require.Contains(t, errs, ":6: "+elemConflictMsg)
	})

	t.Run("two attribute branches through refs conflict", func(t *testing.T) {
		t.Parallel()
		errs := compileInterleave(t, `<grammar xmlns="`+rngNS+`">
  <start>
    <element name="root">
      <interleave>
        <ref name="attrA"/>
        <ref name="attrB"/>
      </interleave>
    </element>
  </start>
  <define name="attrA"><attribute name="x"/></define>
  <define name="attrB"><attribute name="x"/></define>
</grammar>`)
		require.Contains(t, errs, attrConflictMsg)
	})

	t.Run("no conflict for anyName-except vs the excluded name", func(t *testing.T) {
		t.Parallel()
		errs := compileInterleave(t, `<element name="root" xmlns="`+rngNS+`">
  <interleave>
    <element><anyName><except><name>a</name></except></anyName><empty/></element>
    <element name="a"><empty/></element>
  </interleave>
</element>`)
		require.Empty(t, errs)
	})

	t.Run("no conflict for nsName in a different namespace than a plain name", func(t *testing.T) {
		t.Parallel()
		errs := compileInterleave(t, `<element name="root" xmlns="`+rngNS+`">
  <interleave>
    <element><nsName ns="urn:x"/><empty/></element>
    <element name="a"><empty/></element>
  </interleave>
</element>`)
		require.Empty(t, errs)
	})

	t.Run("no conflict between text and elements", func(t *testing.T) {
		t.Parallel()
		errs := compileInterleave(t, `<element name="root" xmlns="`+rngNS+`">
  <interleave>
    <text/>
    <element name="a"><empty/></element>
  </interleave>
</element>`)
		require.Empty(t, errs)
	})

	t.Run("mixed(a, b) has no conflict", func(t *testing.T) {
		t.Parallel()
		errs := compileInterleave(t, `<element name="root" xmlns="`+rngNS+`">
  <mixed>
    <element name="a"><empty/></element>
    <element name="b"><empty/></element>
  </mixed>
</element>`)
		require.Empty(t, errs)
	})
}

// TestGoldenSchemasInterleaveConflicts checks every committed RELAX NG golden
// schema for §7.4 interleave conflicts: every schema except interleave0_0.rng
// and tutor14_1.rng must compile without an interleave-conflict diagnostic,
// and those two must produce exactly the "Element or text conflicts in
// interleave" line at the line their <interleave> starts on (libxml2 parity;
// see the design's section 2.2).
func TestGoldenSchemasInterleaveConflicts(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(testdataBase, "test")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	expected := map[string]string{
		"interleave0_0.rng": "interleave0_0.rng:4: element interleave: Relax-NG parser error : " +
			"Element or text conflicts in interleave\n",
		"tutor14_1.rng": "tutor14_1.rng:21: element interleave: Relax-NG parser error : " +
			"Element or text conflicts in interleave\n",
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".rng" {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(filepath.Join(dir, name))
			require.NoError(t, err)
			doc, err := helium.NewParser().Parse(t.Context(), data)
			if err != nil {
				// Not well-formed XML (e.g. broken-xml.rng), irrelevant to the
				// §7.4 interleave-conflict check this test covers.
				return
			}

			collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
			_, err = relaxng.NewCompiler().Label(name).ErrorHandler(collector).Compile(t.Context(), doc)
			require.NoError(t, err)
			_ = collector.Close()
			_, compileErrors := partitionCompileErrors(collector.Errors())

			if want, ok := expected[name]; ok {
				require.Equal(t, want, compileErrors)
				return
			}
			require.NotContains(t, compileErrors, "conflicts in interleave")
		})
	}
}
