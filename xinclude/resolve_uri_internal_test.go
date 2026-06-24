package xinclude

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestResolveBaseWindowsShaped locks the resolveBase/resolveURI chain for a
// native Windows base. The values are plain strings, so the Windows behavior is
// exercised on every OS. Before the fix, url.Parse read the drive letter as a
// scheme and resolveBase("D:\\..\\base.xml", "one/two") returned the garbage
// "d:///one/two", which then dropped the include directory entirely — the
// libxml2 base.xml golden case.
func TestResolveBaseWindowsShaped(t *testing.T) {
	t.Parallel()

	t.Run("drive-letter base keeps directory through xml:base + relative href", func(t *testing.T) {
		const winBase = `D:\a\helium\helium\testdata\xinclude\docs\base.xml`
		b := resolveBase(winBase, "one/two")
		require.Equal(t, "D:/a/helium/helium/testdata/xinclude/docs/one/two", b)

		u, err := resolveURI("../../ents/base-inc.xml", b)
		require.NoError(t, err)
		require.Equal(t, "D:/a/helium/helium/testdata/xinclude/ents/base-inc.xml", u)
	})

	t.Run("posix base unchanged", func(t *testing.T) {
		b := resolveBase("/a/b/docs/base.xml", "one/two")
		require.Equal(t, "/a/b/docs/one/two", b)
	})

	t.Run("absolute xml:base replaces base verbatim", func(t *testing.T) {
		require.Equal(t, "http://x/y", resolveBase(`D:\dir\doc.xml`, "http://x/y"))
	})
}
