package helium_test

import (
	"errors"
	"runtime"
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

	t.Run("cap boundary is strict-greater for the fast path", func(t *testing.T) {
		t.Parallel()
		// The fast path's scan budget is cap+utf8.UTFMax, so a value of
		// cap+1..cap+UTFMax bytes is still settled by ScanSimpleAttrValue;
		// the explicit post-scan re-check must reject it. Exactly cap bytes
		// is accepted; one more fails.
		atCap, err := helium.NewParser().
			MaxNodeContentSize(64).
			Parse(t.Context(), []byte(bodies["fast"](64)))
		require.NoError(t, err)
		require.NotNil(t, atCap)

		_, err = helium.NewParser().
			MaxNodeContentSize(64).
			Parse(t.Context(), []byte(bodies["fast"](65)))
		require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
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

// TestMaxNodeContentSizeAttrEntityReplacement covers the entity-replacement
// branch of parseAttributeValueInternal (decodeEntities → rep written into the
// accumulating buffer), which the &amp;-based "slow" test above does NOT reach
// because &amp; is a predefined entity handled on a separate branch. An over-cap
// replacement must fail with ErrNodeContentTooLarge while the running total is
// bounded by the cap, not after the whole rep is copied in.
func TestMaxNodeContentSizeAttrEntityReplacement(t *testing.T) {
	t.Parallel()

	// big is far larger than the 64-byte cap; a single reference stays well
	// under the entity-amplification baseline so the node-content cap is what
	// trips, not the amplification guard.
	doc := func(attr string) []byte {
		big := strings.Repeat("a", 4096)
		return []byte(`<!DOCTYPE r [<!ENTITY big "` + big + `">]>` +
			`<r ` + attr + `/>`)
	}

	t.Run("substituted general entity over cap fails", func(t *testing.T) {
		t.Parallel()
		// SubstituteEntities(true) forces &big; to expand inline into the
		// attribute value, exercising the replaceEntities replacement loop.
		_, err := helium.NewParser().
			SubstituteEntities(true).
			MaxNodeContentSize(64).
			Parse(t.Context(), doc(`a="&big;"`))
		require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
	})

	t.Run("namespace attribute over cap fails", func(t *testing.T) {
		t.Parallel()
		// A namespace attribute forces entity replacement even without
		// SubstituteEntities, so xmlns:x="&big;" hits the same branch.
		_, err := helium.NewParser().
			MaxNodeContentSize(64).
			Parse(t.Context(), doc(`xmlns:x="&big;"`))
		require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
	})

	t.Run("within-cap substituted entity parses fine", func(t *testing.T) {
		t.Parallel()
		small := strings.Repeat("a", 32)
		in := []byte(`<!DOCTYPE r [<!ENTITY small "` + small + `">]>` +
			`<r a="&small;"/>`)
		d, err := helium.NewParser().
			SubstituteEntities(true).
			MaxNodeContentSize(64).
			Parse(t.Context(), in)
		require.NoError(t, err)
		require.NotNil(t, d)
	})

	t.Run("unresolved entity reference with long name over cap fails", func(t *testing.T) {
		t.Parallel()
		// Non-substituted general-entity branch: with SubstituteEntities(false)
		// (the default) a declared entity reference is copied literally as
		// "&"+ent.name+";" into the attribute buffer. A very long entity name
		// under MaxNameLength(-1) must trip ErrNodeContentTooLarge before the
		// whole reference is copied, not after.
		longName := strings.Repeat("e", 4096)
		in := []byte(`<!DOCTYPE r [<!ENTITY ` + longName + ` "x">]>` +
			`<r a="&` + longName + `;"/>`)
		_, err := helium.NewParser().
			MaxNameLength(-1).
			MaxNodeContentSize(64).
			Parse(t.Context(), in)
		require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
	})
}

// TestMaxNodeContentSizeAttrEntityNotMaterialized proves the substituted
// entity-replacement path enforces the cap DURING decode rather than after
// fully materializing the expansion. The entity nests so its stored literal is
// tiny (~tens of KiB) but its full expansion is ~64 MiB. The amplification guard
// is disabled so only the node-content cap can stop it. If the decoder still
// built the whole replacement string before checking the cap (the old
// decodeEntities → rep behavior), this parse would allocate at least the full
// 64 MiB expansion; streaming the decode through the cap keeps total allocation
// orders of magnitude smaller. The bound is checked via runtime.MemStats
// TotalAlloc, so this test must NOT run in parallel (TotalAlloc is process-wide
// and a concurrent test's allocations would pollute the delta).
func TestMaxNodeContentSizeAttrEntityNotMaterialized(t *testing.T) {
	// no t.Parallel(): isolated so the TotalAlloc delta reflects only this parse.

	// inner: 4 KiB; outer references inner 16384 times => ~64 MiB expansion,
	// but the stored literal of outer is only ~3*16384 = ~48 KiB.
	inner := strings.Repeat("a", 4096)
	var refs strings.Builder
	for range 16384 {
		refs.WriteString("&inner;")
	}
	in := []byte(`<!DOCTYPE r [` +
		`<!ENTITY inner "` + inner + `">` +
		`<!ENTITY outer "` + refs.String() + `">` +
		`]><r a="&outer;"/>`)

	const expansion = 16384 * 4096 // ~64 MiB the old path would have materialized
	// Generous bound: far below the full expansion, far above the parse's real
	// working set (input + entity-table literals + cursor buffers ≈ a few MiB).
	const allocBound = 16 << 20 // 16 MiB

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	_, err := helium.NewParser().
		SubstituteEntities(true).
		MaxEntityAmplification(-1). // disable amplification guard: cap is the only brake
		MaxNodeContentSize(64).
		Parse(t.Context(), in)

	runtime.ReadMemStats(&after)

	require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)

	delta := after.TotalAlloc - before.TotalAlloc
	require.Less(t, delta, uint64(allocBound),
		"entity expansion was materialized: parse allocated %d bytes (full expansion is %d bytes); the cap must stop the decode incrementally",
		delta, uint64(expansion))
}
