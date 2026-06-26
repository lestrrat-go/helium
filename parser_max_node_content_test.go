package helium_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// wrapConstruct returns an XML document whose single root element contains an
// indivisible content run of the given kind, with a body of n 'a' bytes.
func nodeContentDoc(kind string, n int) string {
	body := strings.Repeat("a", n)
	switch kind {
	case "cdata":
		return "<r><![CDATA[" + body + "]]></r>"
	case "comment":
		return "<r><!--" + body + "--></r>"
	case "pi":
		return "<r><?pi " + body + "?></r>"
	case "chardata":
		return "<r>" + body + "</r>"
	default:
		panic("unknown kind " + kind)
	}
}

func TestMaxNodeContentSize(t *testing.T) {
	t.Parallel()

	kinds := []string{"cdata", "comment", "pi", "chardata"}

	t.Run("oversized run fails with ErrNodeContentTooLarge", func(t *testing.T) {
		t.Parallel()
		for _, kind := range kinds {
			t.Run(kind, func(t *testing.T) {
				t.Parallel()
				doc := nodeContentDoc(kind, 200)
				_, err := helium.NewParser().
					MaxNodeContentSize(64).
					Parse(t.Context(), []byte(doc))
				require.Error(t, err)
				require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
			})
		}
	})

	t.Run("within-cap run parses fine", func(t *testing.T) {
		t.Parallel()
		for _, kind := range kinds {
			t.Run(kind, func(t *testing.T) {
				t.Parallel()
				doc := nodeContentDoc(kind, 32)
				d, err := helium.NewParser().
					MaxNodeContentSize(64).
					Parse(t.Context(), []byte(doc))
				require.NoError(t, err)
				require.NotNil(t, d)
			})
		}
	})

	t.Run("negative limit disables the cap", func(t *testing.T) {
		t.Parallel()
		for _, kind := range kinds {
			t.Run(kind, func(t *testing.T) {
				t.Parallel()
				// A run far past the 10 MiB default still parses when the cap
				// is disabled with a negative value.
				doc := nodeContentDoc(kind, 12<<20)
				d, err := helium.NewParser().
					MaxNodeContentSize(-1).
					Parse(t.Context(), []byte(doc))
				require.NoError(t, err)
				require.NotNil(t, d)
			})
		}
	})

	t.Run("a raised cap admits a larger run", func(t *testing.T) {
		t.Parallel()
		for _, kind := range kinds {
			t.Run(kind, func(t *testing.T) {
				t.Parallel()
				doc := nodeContentDoc(kind, 4096)
				// Fails under a small cap...
				_, err := helium.NewParser().
					MaxNodeContentSize(1024).
					Parse(t.Context(), []byte(doc))
				require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
				// ...and parses under a cap large enough for it.
				d, err := helium.NewParser().
					MaxNodeContentSize(8192).
					Parse(t.Context(), []byte(doc))
				require.NoError(t, err)
				require.NotNil(t, d)
			})
		}
	})

	t.Run("secure default rejects a run over 10 MiB", func(t *testing.T) {
		t.Parallel()
		// NewParser applies DefaultMaxNodeContentSize (10 MiB); an 11 MiB
		// char-data run fails by default without any explicit cap.
		doc := nodeContentDoc("chardata", 11<<20)
		_, err := helium.NewParser().Parse(t.Context(), []byte(doc))
		require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
	})

	t.Run("secure default admits a run under 10 MiB", func(t *testing.T) {
		t.Parallel()
		doc := nodeContentDoc("chardata", 1<<20)
		d, err := helium.NewParser().Parse(t.Context(), []byte(doc))
		require.NoError(t, err)
		require.NotNil(t, d)
	})

	t.Run("cap boundary is strict-greater for char data", func(t *testing.T) {
		t.Parallel()
		// Exactly cap bytes is accepted; one more byte fails.
		atCap, err := helium.NewParser().
			MaxNodeContentSize(64).
			Parse(t.Context(), []byte(nodeContentDoc("chardata", 64)))
		require.NoError(t, err)
		require.NotNil(t, atCap)

		_, err = helium.NewParser().
			MaxNodeContentSize(64).
			Parse(t.Context(), []byte(nodeContentDoc("chardata", 65)))
		require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
	})

	t.Run("errors.Is matches the sentinel", func(t *testing.T) {
		t.Parallel()
		_, err := helium.NewParser().
			MaxNodeContentSize(16).
			Parse(t.Context(), []byte(nodeContentDoc("comment", 128)))
		require.True(t, errors.Is(err, helium.ErrNodeContentTooLarge))
	})
}

func TestMaxNodeContentSizeAttrValue(t *testing.T) {
	t.Parallel()

	// fast: a simple value (no entities/special whitespace) taking the
	// ScanSimpleAttrValue fast path. slow: a value containing an entity
	// reference, forcing the buffer-accumulating slow path.
	bodies := map[string]func(n int) string{
		"fast": func(n int) string {
			return `<r a="` + strings.Repeat("a", n) + `"/>`
		},
		"slow": func(n int) string {
			return `<r a="` + strings.Repeat("a", n) + `&amp;"/>`
		},
	}

	t.Run("oversized attribute value fails with ErrNodeContentTooLarge", func(t *testing.T) {
		t.Parallel()
		for kind, mk := range bodies {
			t.Run(kind, func(t *testing.T) {
				t.Parallel()
				_, err := helium.NewParser().
					MaxNodeContentSize(64).
					Parse(t.Context(), []byte(mk(200)))
				require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
			})
		}
	})

	t.Run("within-cap attribute value parses fine", func(t *testing.T) {
		t.Parallel()
		for kind, mk := range bodies {
			t.Run(kind, func(t *testing.T) {
				t.Parallel()
				d, err := helium.NewParser().
					MaxNodeContentSize(64).
					Parse(t.Context(), []byte(mk(32)))
				require.NoError(t, err)
				require.NotNil(t, d)
			})
		}
	})

	t.Run("negative limit disables the cap", func(t *testing.T) {
		t.Parallel()
		// A value far past the 10 MiB default still parses when the cap is
		// disabled with a negative value.
		d, err := helium.NewParser().
			MaxNodeContentSize(-1).
			Parse(t.Context(), []byte(bodies["fast"](12<<20)))
		require.NoError(t, err)
		require.NotNil(t, d)
	})

	t.Run("secure default rejects an attribute value over 10 MiB", func(t *testing.T) {
		t.Parallel()
		_, err := helium.NewParser().Parse(t.Context(), []byte(bodies["fast"](11<<20)))
		require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
	})
}
