package helium_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/sax"
	"github.com/stretchr/testify/require"
)

// TestSafeDefaults locks in the secure-by-default behavior of NewParser. Each
// subtest asserts a guarantee that untrusted input relies on; a regression that
// re-enables external loading, the host filesystem, or unbounded depth by
// default must fail here.
func TestSafeDefaults(t *testing.T) {
	t.Parallel()

	// extDTD is an external subset that would default an attribute if loaded.
	// Observing the attribute on the root element proves the DTD was read.
	const extDTDName = "ext.dtd"
	const extDTD = `<!ELEMENT doc EMPTY>` + "\n" + `<!ATTLIST doc x CDATA "default">`
	const dtdDoc = `<?xml version="1.0"?>` + "\n" + `<!DOCTYPE doc SYSTEM "ext.dtd">` + "\n" + `<doc/>`

	t.Run("external entity blocked by default", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY ext SYSTEM "ext.xml">
]>
<doc>&ext;</doc>`

		resolved := false
		s := sax.New()
		s.SetOnResolveEntity(sax.ResolveEntityFunc(func(_ context.Context, _, systemID string) (sax.ParseInput, error) {
			resolved = true
			return newStringParseInput("<inner/>", systemID), nil
		}))

		// SubstituteEntities is requested but BlockXXE is NOT cleared, so the
		// default XXE guard must keep the external entity from being fetched.
		_, _ = helium.NewParser().SAXHandler(s).SubstituteEntities(true).Parse(t.Context(), []byte(input))
		require.False(t, resolved, "external entity must not be resolved by default")
	})

	t.Run("external DTD blocked by default", func(t *testing.T) {
		t.Parallel()

		resolved := false
		s := sax.New()
		s.SetOnResolveEntity(sax.ResolveEntityFunc(func(_ context.Context, _, systemID string) (sax.ParseInput, error) {
			resolved = true
			return newStringParseInput(extDTD, systemID), nil
		}))

		// LoadExternalDTD is requested but BlockXXE is NOT cleared.
		_, _ = helium.NewParser().SAXHandler(s).LoadExternalDTD(true).DefaultDTDAttributes(true).Parse(t.Context(), []byte(dtdDoc))
		require.False(t, resolved, "external DTD must not be loaded by default")
	})

	t.Run("default FS denies even with XXE lifted", func(t *testing.T) {
		t.Parallel()

		// XXE is lifted and external-DTD loading requested, but no FS is
		// supplied. The default deny-all FS must still prevent the load, so the
		// ATTLIST default never materializes.
		doc, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			DefaultDTDAttributes(true).
			Parse(t.Context(), []byte(dtdDoc))
		require.NoError(t, err)
		root := doc.DocumentElement()
		require.NotNil(t, root)
		_, ok := root.GetAttribute("x")
		require.False(t, ok, "default deny-all FS must block external DTD loading")
	})

	t.Run("explicit FS re-enables loading", func(t *testing.T) {
		t.Parallel()

		// Same parser, but an explicit FS is supplied: the DTD now loads and the
		// defaulted attribute appears. This isolates the FS as the gate.
		fsys := fstest.MapFS{extDTDName: &fstest.MapFile{Data: []byte(extDTD)}}
		doc, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			DefaultDTDAttributes(true).
			FS(fsys).
			Parse(t.Context(), []byte(dtdDoc))
		require.NoError(t, err)
		root := doc.DocumentElement()
		require.NotNil(t, root)
		x, ok := root.GetAttribute("x")
		require.True(t, ok, "an explicit FS must re-enable external DTD loading")
		require.Equal(t, "default", x)
	})

	t.Run("FS(nil) restores deny-all", func(t *testing.T) {
		t.Parallel()

		fsys := fstest.MapFS{extDTDName: &fstest.MapFile{Data: []byte(extDTD)}}
		// Supply a permissive FS, then reset to nil: nil must restore the
		// deny-all default, not the historical permissive root.
		doc, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			DefaultDTDAttributes(true).
			FS(fsys).
			FS(nil).
			Parse(t.Context(), []byte(dtdDoc))
		require.NoError(t, err)
		root := doc.DocumentElement()
		require.NotNil(t, root)
		_, ok := root.GetAttribute("x")
		require.False(t, ok, "FS(nil) must restore the deny-all default")
	})

	t.Run("element depth capped at 256 by default", func(t *testing.T) {
		t.Parallel()

		atLimit := strings.Repeat("<a>", 256) + strings.Repeat("</a>", 256)
		_, err := helium.NewParser().Parse(t.Context(), []byte(atLimit))
		require.NoError(t, err, "nesting at the 256 default limit must parse")

		overLimit := strings.Repeat("<a>", 257) + strings.Repeat("</a>", 257)
		_, err = helium.NewParser().Parse(t.Context(), []byte(overLimit))
		require.Error(t, err, "nesting past the 256 default limit must be rejected")
		require.Contains(t, err.Error(), "exceeded max depth")
	})

	t.Run("MaxDepth(0) keeps the 256 default cap", func(t *testing.T) {
		t.Parallel()

		deep := strings.Repeat("<a>", 300) + strings.Repeat("</a>", 300)
		_, err := helium.NewParser().MaxDepth(0).Parse(t.Context(), []byte(deep))
		require.Error(t, err, "MaxDepth(0) must select the 256 default, not disable the cap")
		require.Contains(t, err.Error(), "exceeded max depth")
	})

	t.Run("MaxDepth(-1) disables the cap", func(t *testing.T) {
		t.Parallel()

		deep := strings.Repeat("<a>", 512) + strings.Repeat("</a>", 512)
		_, err := helium.NewParser().MaxDepth(-1).Parse(t.Context(), []byte(deep))
		require.NoError(t, err, "a negative MaxDepth must opt out of the depth cap")
	})
}

// TestPermissiveFS verifies the public escape hatch opens host paths via
// os.Open — including absolute paths that os.DirFS would reject — so callers
// that deliberately need the historical behavior can restore it.
func TestPermissiveFS(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "x.txt")
	require.NoError(t, os.WriteFile(path, []byte("hi"), 0o600))

	f, err := helium.PermissiveFS().Open(path)
	require.NoError(t, err, "PermissiveFS must open an absolute host path")
	require.NoError(t, f.Close())

	_, err = helium.PermissiveFS().Open(filepath.Join(dir, "does-not-exist"))
	require.Error(t, err, "a missing path must still error")
}
