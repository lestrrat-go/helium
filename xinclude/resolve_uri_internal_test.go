package xinclude

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
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

// TestComputeFixupBasesWindowsShaped locks the xml:base fixup relativization for
// a native Windows target base mixed with a forward-slash include source URI
// (the exact shape on a Windows runner: p.baseURI is an OS path while the
// resolved include URI uses '/'). The values are plain strings, so the Windows
// branch is exercised on every OS. Before the fix, computeFixupBases used
// filepath.IsAbs / commonAncestorDir / filepath.Rel, which split on '\' and
// failed to find the common ancestor of the mixed-separator inputs, producing
// wrong relative bases (the libxml2 base.xml golden failure: "two2" instead of
// "../../ents/one/two2"). The relativization is now pure forward-slash.
func TestComputeFixupBasesWindowsShaped(t *testing.T) {
	t.Parallel()

	const (
		winBase   = `D:\proj\xinclude\docs\base.xml`
		sourceURI = "D:/proj/xinclude/ents/base-inc.xml"
	)

	// An xi:include element with no xml:base of its own, so the effective target
	// base reduces to the relativized target document path.
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	inc, err := doc.CreateElement("include")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(inc))

	p := &processor{baseURI: winBase}
	relSource, target := p.computeFixupBases(inc, sourceURI)
	require.Equal(t, "ents/base-inc.xml", relSource,
		"include source must relativize against the common ancestor with forward-slash semantics")
	require.Equal(t, "docs/base.xml", target,
		"target document base must relativize against the common ancestor")

	t.Run("posix unchanged", func(t *testing.T) {
		pdoc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		pinc, err := pdoc.CreateElement("include")
		require.NoError(t, err)
		require.NoError(t, pdoc.AddChild(pinc))
		pp := &processor{baseURI: "/proj/xinclude/docs/base.xml"}
		rs, tgt := pp.computeFixupBases(pinc, "/proj/xinclude/ents/base-inc.xml")
		require.Equal(t, "ents/base-inc.xml", rs)
		require.Equal(t, "docs/base.xml", tgt)
	})
}
