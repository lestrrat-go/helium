package relaxng

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestResolveHrefWindowsBase locks include/externalRef href resolution when the
// schema was loaded from a native Windows path: CompileFile derives baseDir via
// filepath.Dir and SetURL stores the native path, so on Windows both carry
// backslashes. A sibling href ("inline2.rng") must resolve INSIDE the base
// directory rather than tripping the ".."-escape guard — the tutor9_* RelaxNG
// golden regression. The inputs are plain strings, so the Windows behavior is
// exercised on Linux.
func TestResolveHrefWindowsBase(t *testing.T) {
	t.Parallel()

	newCompiler := func(baseDir string) *compiler {
		return &compiler{baseDir: baseDir}
	}

	// Build a tiny document with a backslash URL and an <include href=...> elem.
	mkElem := func(docURL, href string) *helium.Element {
		doc := helium.NewDefaultDocument()
		doc.SetURL(docURL)
		elem := doc.CreateElement("include")
		err := elem.SetAttribute("href", href)
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(elem))
		return elem
	}

	t.Run("sibling href against windows base resolves inside base", func(t *testing.T) {
		c := newCompiler(`..\testdata\relaxng\test`)
		elem := mkElem(`..\testdata\relaxng\test\tutor9_7.rng`, "inline2.rng")
		got, err := c.resolveHref(t.Context(), elem, "inline2.rng")
		require.NoError(t, err)
		require.Equal(t, "../testdata/relaxng/test/inline2.rng", got)
	})

	t.Run("escaping href against windows base still rejected", func(t *testing.T) {
		c := newCompiler(`..\testdata\relaxng\test`)
		elem := mkElem(`..\testdata\relaxng\test\tutor9_7.rng`, `..\..\..\etc\passwd`)
		_, err := c.resolveHref(t.Context(), elem, `..\..\..\etc\passwd`)
		require.Error(t, err)
		require.ErrorContains(t, err, "escapes base directory")
	})
}
