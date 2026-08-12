package helium_test

import (
	"bytes"
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
	"unicode/utf8"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/sax"
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

func TestParserLimits(t *testing.T) {
	t.Parallel()

	t.Run("max name length", func(t *testing.T) {
		longName := strings.Repeat("a", 200)
		doc := "<" + longName + "></" + longName + ">"

		t.Run("default accepts a moderately long name", func(t *testing.T) {
			t.Parallel()
			d, err := helium.NewParser().Parse(t.Context(), []byte(doc))
			require.NoError(t, err)
			require.NotNil(t, d)
		})

		t.Run("positive limit rejects an over-length name", func(t *testing.T) {
			t.Parallel()
			_, err := helium.NewParser().MaxNameLength(64).Parse(t.Context(), []byte(doc))
			require.Error(t, err)
			require.ErrorContains(t, err, "name is too long")
		})

		t.Run("limit is in bytes, not runes", func(t *testing.T) {
			t.Parallel()
			// "a界界" is 3 runes but 7 bytes (each 界 is 3 UTF-8 bytes). With a
			// byte limit of 4 it must be rejected; a rune-based check (the bug this
			// guards) would wrongly accept it (3 <= 4). The over-long name is
			// surfaced as a name-parse failure.
			mb := "<a界界></a界界>"
			_, err := helium.NewParser().MaxNameLength(4).Parse(t.Context(), []byte(mb))
			require.Error(t, err, "a 7-byte name must be rejected at a 4-byte limit")
		})

		t.Run("negative limit removes the cap", func(t *testing.T) {
			t.Parallel()
			// A name far past the 50000-char default still parses when the limit
			// is disabled.
			huge := strings.Repeat("a", 60000)
			d, err := helium.NewParser().
				MaxNameLength(-1).
				Parse(t.Context(), []byte("<"+huge+"></"+huge+">"))
			require.NoError(t, err)
			require.NotNil(t, d)
		})

		t.Run("limit bounds the full prefixed QName", func(t *testing.T) {
			t.Parallel()
			// The element QName "aaaa:bbbbb" is 10 bytes. A per-part check would
			// wrongly accept it under MaxNameLength(5) (each NCName part is <= 5);
			// the limit must bound the whole QName.
			src := `<aaaa:bbbbb xmlns:aaaa="u"/>`
			_, err := helium.NewParser().MaxNameLength(5).Parse(t.Context(), []byte(src))
			require.Error(t, err, "a 10-byte prefixed QName must be rejected at a 5-byte limit")
		})

		t.Run("limit bounds entity-reference names in entity values", func(t *testing.T) {
			t.Parallel()
			// A general-entity reference name inside an entity's replacement value
			// (here 8 bytes) must be rejected at declaration time under a 4-byte
			// cap, not silently stored and only caught if the entity is expanded.
			src := `<?xml version="1.0"?>` + "\n" +
				`<!DOCTYPE r [<!ENTITY e "&aaaaaaaa;">]>` + "\n" +
				`<r/>`
			_, err := helium.NewParser().MaxNameLength(4).Parse(t.Context(), []byte(src))
			require.Error(t, err, "over-limit entity-reference name in an entity value must be rejected")
		})

		t.Run("limit applies inside entity expansion", func(t *testing.T) {
			t.Parallel()
			// The entity replacement text contains an element whose name (8 bytes)
			// exceeds the limit. The cap must be enforced during entity expansion,
			// not just on the top-level document (the nested parser inherits it).
			src := `<?xml version="1.0"?>` + "\n" +
				`<!DOCTYPE r [<!ENTITY e "<longname/>">]>` + "\n" +
				`<r>&e;</r>`
			_, err := helium.NewParser().
				SubstituteEntities(true).
				MaxNameLength(4).
				Parse(t.Context(), []byte(src))
			require.Error(t, err, "MaxNameLength must be enforced inside entity expansion")
		})
	})

	t.Run("max content-model depth", func(t *testing.T) {
		// A DTD whose root content model nests parenthesized groups several levels
		// deep. The default (128) accepts it; a tiny limit rejects it.
		doc := `<?xml version="1.0"?>
<!DOCTYPE root [
<!ELEMENT root (((((((a)))))))>
<!ELEMENT a (#PCDATA)>
]>
<root><a/></root>`

		t.Run("default accepts a shallow content model", func(t *testing.T) {
			t.Parallel()
			d, err := helium.NewParser().Parse(t.Context(), []byte(doc))
			require.NoError(t, err)
			require.NotNil(t, d)
		})

		t.Run("tiny limit rejects a nested content model", func(t *testing.T) {
			t.Parallel()
			_, err := helium.NewParser().MaxContentModelDepth(2).Parse(t.Context(), []byte(doc))
			require.Error(t, err)
			require.ErrorContains(t, err, "too deep")
		})

		t.Run("negative limit removes the cap", func(t *testing.T) {
			t.Parallel()
			d, err := helium.NewParser().MaxContentModelDepth(-1).Parse(t.Context(), []byte(doc))
			require.NoError(t, err)
			require.NotNil(t, d)
		})
	})

	t.Run("max depth", func(t *testing.T) {
		t.Run("exceeded", func(t *testing.T) {
			t.Parallel()

			input := []byte(strings.Repeat("<a>", 10) + strings.Repeat("</a>", 10))
			p := helium.NewParser().MaxDepth(5)

			_, err := p.Parse(t.Context(), input)
			require.Error(t, err)
			require.Contains(t, err.Error(), "exceeded max depth")
		})

		t.Run("within limit", func(t *testing.T) {
			t.Parallel()

			input := []byte(strings.Repeat("<a>", 5) + "hello" + strings.Repeat("</a>", 5))
			p := helium.NewParser().MaxDepth(10)

			doc, err := p.Parse(t.Context(), input)
			require.NoError(t, err)
			require.NotNil(t, doc)
		})

		t.Run("exact limit", func(t *testing.T) {
			t.Parallel()

			input := []byte(strings.Repeat("<a>", 5) + "hello" + strings.Repeat("</a>", 5))
			p := helium.NewParser().MaxDepth(5)

			doc, err := p.Parse(t.Context(), input)
			require.NoError(t, err)
			require.NotNil(t, doc)
		})

		t.Run("zero is unlimited", func(t *testing.T) {
			t.Parallel()

			input := []byte(strings.Repeat("<a>", 100) + "hello" + strings.Repeat("</a>", 100))
			p := helium.NewParser()

			doc, err := p.Parse(t.Context(), input)
			require.NoError(t, err)
			require.NotNil(t, doc)
		})

		t.Run("via ParseReader", func(t *testing.T) {
			t.Parallel()

			input := strings.Repeat("<a>", 10) + strings.Repeat("</a>", 10)
			p := helium.NewParser().MaxDepth(5)

			_, err := p.ParseReader(t.Context(), bytes.NewReader([]byte(input)))
			require.Error(t, err)
			require.Contains(t, err.Error(), "exceeded max depth")
		})

		t.Run("enforced within substituted entity", func(t *testing.T) {
			t.Parallel()

			// The replacement text expands to two nested elements. With entity
			// substitution enabled the depth check must still apply to the chunk
			// parsed for the entity, so MaxDepth(1) rejects the inner <b/>.
			input := []byte(`<!DOCTYPE r [<!ENTITY e "<a><b/></a>">]><r>&e;</r>`)
			p := helium.NewParser().SubstituteEntities(true).MaxDepth(1)

			_, err := p.Parse(t.Context(), input)
			require.Error(t, err)
			require.Contains(t, err.Error(), "exceeded max depth")
		})

		t.Run("within limit inside substituted entity", func(t *testing.T) {
			t.Parallel()

			input := []byte(`<!DOCTYPE r [<!ENTITY e "<a><b/></a>">]><r>&e;</r>`)
			p := helium.NewParser().SubstituteEntities(true).MaxDepth(10)

			doc, err := p.Parse(t.Context(), input)
			require.NoError(t, err)
			require.NotNil(t, doc)
		})

		t.Run("single level via substituted entity counts parent depth", func(t *testing.T) {
			t.Parallel()

			// The entity replacement text is a single element, but it is substituted
			// inside <r>, so the literal document is <r><a/></r> (depth 2). The nested
			// chunk parse must continue counting from the parent's current element
			// depth (1) instead of restarting at 0, so MaxDepth(1) must reject it.
			input := []byte(`<!DOCTYPE r [<!ENTITY e "<a/>">]><r>&e;</r>`)
			p := helium.NewParser().SubstituteEntities(true).MaxDepth(1)

			_, err := p.Parse(t.Context(), input)
			require.Error(t, err)
			require.Contains(t, err.Error(), "exceeded max depth")
		})

		t.Run("enforced within external substituted entity", func(t *testing.T) {
			t.Parallel()

			// The external entity replacement text adds element nesting. With the
			// parent's current element depth carried into the external chunk parse,
			// MaxDepth(1) must reject the element delivered by the external entity.
			fsys := fstest.MapFS{
				"nested.xml": &fstest.MapFile{Data: []byte(`<a/>`)},
			}
			input := []byte(`<!DOCTYPE r [<!ENTITY e SYSTEM "nested.xml">]><r>&e;</r>`)
			p := helium.NewParser().BlockXXE(false).SubstituteEntities(true).MaxDepth(1).FS(fsys)

			_, err := p.Parse(t.Context(), input)
			require.Error(t, err)
			require.Contains(t, err.Error(), "exceeded max depth")
		})

		t.Run("cached entity replay under deeper element exceeds limit", func(t *testing.T) {
			t.Parallel()

			// The first &e; expands as a direct child of <r> (depth 2) and caches
			// its subtree. The second &e; is referenced inside <x> (depth 2), so its
			// element reaches depth 3. The cached subtree must still be charged
			// against MaxDepth on replay, otherwise the deeper reuse is wrongly
			// accepted.
			input := []byte(`<!DOCTYPE r [<!ENTITY e "<a/>">]><r>&e;<x>&e;</x></r>`)
			p := helium.NewParser().SubstituteEntities(true).MaxDepth(2)

			_, err := p.Parse(t.Context(), input)
			require.Error(t, err)
			require.Contains(t, err.Error(), "exceeded max depth")
		})

		t.Run("cached entity replay within limit succeeds", func(t *testing.T) {
			t.Parallel()

			// Same shape as above, but MaxDepth(3) accommodates the deeper reuse, so
			// both expansions parse cleanly.
			input := []byte(`<!DOCTYPE r [<!ENTITY e "<a/>">]><r>&e;<x>&e;</x></r>`)
			p := helium.NewParser().SubstituteEntities(true).MaxDepth(3)

			doc, err := p.Parse(t.Context(), input)
			require.NoError(t, err)
			require.NotNil(t, doc)
		})
	})
}

func TestMaxNodeContentSize(t *testing.T) {
	t.Parallel()

	t.Run("character data", func(t *testing.T) {
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
	})

	t.Run("an attribute value", func(t *testing.T) {
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
	})

	// the entity-replacement
	// branch of parseAttributeValueInternal (decodeEntities → rep written into the
	// accumulating buffer), which the &amp;-based "slow" test above does NOT reach
	// because &amp; is a predefined entity handled on a separate branch. An over-cap
	// replacement must fail with ErrNodeContentTooLarge while the running total is
	// bounded by the cap, not after the whole rep is copied in.
	t.Run("an attribute entity replacement", func(t *testing.T) {
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
	})

	// the substituted
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
	t.Run("an attribute entity is not materialized", func(t *testing.T) {
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
	})
}

func TestCharBufferSize(t *testing.T) {
	t.Parallel()

	// confirms a tiny char buffer (which forces
	// repeated cursor refills) still parses a larger document correctly.
	t.Run("affects the parse", func(t *testing.T) {
		var b []byte
		b = append(b, []byte(`<root>`)...)
		for range 200 {
			b = append(b, []byte(`<item>x</item>`)...)
		}
		b = append(b, []byte(`</root>`)...)

		doc, err := helium.NewParser().CharBufferSize(16).Parse(t.Context(), b)
		require.NoError(t, err)
		require.Equal(t, "root", doc.DocumentElement().Name())
	})

	// a large
	// delimiter-free character-data run delivered to a streaming SAX consumer is
	// scanned and delivered in bounded chunks rather than materialized whole. Before
	// the fix the entire run was buffered (in charBuf and the cursor's internal
	// buffer) before the first chunk was delivered.
	//
	// Rather than sampling a global heap signal (non-deterministic, especially
	// under t.Parallel), this instruments the reader: the parser pulls bytes from r
	// on demand, so the count of bytes read at the first SAX callback shows whether
	// delivery began before the whole payload was drained. A streaming parser fires
	// the first callback after reading only a bounded prefix (one cursor buffer ≈
	// 8 KiB), far short of the multi-megabyte run; the pre-fix whole-run buffering
	// would have drained nearly all of it first.
	t.Run("bounds char-data memory", func(t *testing.T) {
		const fillBytes = 32 << 20 // 32 MiB delimiter-free run
		const bufSize = 8192

		r := &genCharDataReader{
			prefix: []byte("<root>"),
			fill:   'a',
			remain: fillBytes,
			suffix: []byte("</root>"),
		}

		var chunkCount, total, maxChunk, readAtFirstChunk int

		handler := sax.New()
		record := func(_ context.Context, ch []byte) error {
			if chunkCount == 0 {
				readAtFirstChunk = r.nread
			}
			chunkCount++
			total += len(ch)
			maxChunk = max(maxChunk, len(ch))
			return nil
		}
		handler.SetOnCharacters(sax.CharactersFunc(record))
		handler.SetOnIgnorableWhitespace(sax.IgnorableWhitespaceFunc(record))

		p := helium.NewParser().SAXHandler(handler).CharBufferSize(bufSize)
		_, err := p.ParseReader(context.Background(), r)
		require.NoError(t, err)

		require.Equal(t, fillBytes, total, "every character byte is delivered")
		require.LessOrEqual(t, maxChunk, bufSize, "no chunk exceeds the configured buffer size")
		require.Greater(t, chunkCount, 1, "the run is delivered in multiple chunks")

		// The first callback must fire before the whole run has been read; a bound
		// well below the payload (yet generously above one cursor buffer) proves the
		// scan buffer stays bounded instead of materializing the whole run first.
		require.Less(t, readAtFirstChunk, 1<<20,
			"streaming must deliver the first chunk after reading only a bounded prefix; read %d bytes first", readAtFirstChunk)
	})

	// a large
	// delimiter-free all-whitespace run delivered to a streaming SAX consumer is
	// bounded in memory. A blank run cannot be proven ignorable whitespace until
	// its end is in view, so a naive implementation buffers it whole before the
	// first callback. The bounded-whitespace policy downgrades an over-budget blank
	// prefix to character data and streams the rest in fixed-size chunks, so the
	// first callback fires after reading only a bounded prefix. This mirrors
	// TestParserCharBufferSizeBoundsCharDataMemory but uses a whitespace fill byte.
	t.Run("bounds whitespace memory", func(t *testing.T) {
		const fillBytes = 32 << 20 // 32 MiB delimiter-free whitespace run
		const bufSize = 8192

		r := &genCharDataReader{
			prefix: []byte("<root>"),
			fill:   ' ',
			remain: fillBytes,
			suffix: []byte("</root>"),
		}

		var chunkCount, total, maxChunk, readAtFirstChunk int
		sawIgnorable := false

		handler := sax.New()
		record := func(ignorable bool) func(context.Context, []byte) error {
			return func(_ context.Context, ch []byte) error {
				if chunkCount == 0 {
					readAtFirstChunk = r.nread
				}
				if ignorable {
					sawIgnorable = true
				}
				chunkCount++
				total += len(ch)
				maxChunk = max(maxChunk, len(ch))
				return nil
			}
		}
		handler.SetOnCharacters(sax.CharactersFunc(record(false)))
		handler.SetOnIgnorableWhitespace(sax.IgnorableWhitespaceFunc(record(true)))

		p := helium.NewParser().SAXHandler(handler).CharBufferSize(bufSize)
		_, err := p.ParseReader(context.Background(), r)
		require.NoError(t, err)

		require.Equal(t, fillBytes, total, "every whitespace byte is delivered")
		require.LessOrEqual(t, maxChunk, bufSize, "no chunk exceeds the configured buffer size")
		require.Greater(t, chunkCount, 1, "the run is delivered in multiple chunks")
		require.False(t, sawIgnorable,
			"an over-budget blank run is downgraded to character data, not buffered whole as ignorable whitespace")

		// The first callback must fire well before the whole run has been read,
		// proving the blank prefix is bounded rather than materialized whole.
		require.Less(t, readAtFirstChunk, 1<<20,
			"bounded-whitespace policy must deliver the first chunk after reading only a bounded prefix; read %d bytes first", readAtFirstChunk)
	})

	// pins the chunked
	// streaming-SAX path's whitespace classification against the single-shot path,
	// which classifies the WHOLE run as one unit. The single-shot path looks over
	// the whole run to decide ignorable-whitespace vs. character data; the chunked
	// path delivers in bounded pieces and cannot, so it must not emit any
	// IgnorableWhitespace event until the whole run is proven blank: a fully-blank
	// run is ignorable whitespace, but a run containing any text is character data
	// in its entirety, leading blanks included.
	//
	// This guards against two earlier bugs: per-chunk classification, where a blank
	// run could arrive as Characters then IgnorableWhitespace under a tiny buffer;
	// and "sticky downgrade", where the leading blanks of <root>  text</root> were
	// emitted as an early IgnorableWhitespace chunk before the text was seen.
	t.Run("classifies whitespace consistently", func(t *testing.T) {
		type event struct {
			ignorable bool
			content   string
		}

		run := func(src string, bufSize int) []event {
			var events []event
			handler := sax.New()
			handler.SetOnCharacters(sax.CharactersFunc(func(_ context.Context, ch []byte) error {
				events = append(events, event{ignorable: false, content: string(ch)})
				return nil
			}))
			handler.SetOnIgnorableWhitespace(sax.IgnorableWhitespaceFunc(func(_ context.Context, ch []byte) error {
				events = append(events, event{ignorable: true, content: string(ch)})
				return nil
			}))
			_, err := helium.NewParser().SAXHandler(handler).CharBufferSize(bufSize).
				Parse(context.Background(), []byte(src))
			require.NoError(t, err)
			return events
		}

		concat := func(events []event) string {
			var b strings.Builder
			for _, e := range events {
				b.WriteString(e.content)
			}
			return b.String()
		}

		t.Run("all-whitespace run stays ignorable across chunks", func(t *testing.T) {
			t.Parallel()
			events := run("<root>        </root>", 2)
			require.Greater(t, len(events), 1, "the tiny buffer must split the run")
			require.Equal(t, "        ", concat(events), "every whitespace byte is delivered")
			for _, e := range events {
				require.True(t, e.ignorable, "a fully-blank run must classify every chunk as ignorable whitespace")
			}
		})

		t.Run("run containing text is all characters, no leading ignorable", func(t *testing.T) {
			t.Parallel()
			events := run("<root>      text</root>", 2)
			require.Equal(t, "      text", concat(events), "every character byte is delivered")
			require.NotEmpty(t, events, "a run containing text must deliver characters")
			for _, e := range events {
				require.False(t, e.ignorable,
					"a run containing text is character data in its entirety; leading blanks must not be emitted as ignorable whitespace")
			}
		})
	})

	// the chunked streaming-SAX
	// path never splits a UTF-8 rune, even when CharBufferSize is smaller than a
	// single multi-byte rune. ScanCharDataSlice returns a lone over-budget rune
	// whole (to guarantee progress), and deliverCharacters must then deliver it
	// whole rather than slicing it into invalid UTF-8 fragments.
	t.Run("never splits a rune", func(t *testing.T) {
		// Mix 2-, 3-, and 4-byte runes so a 1-byte buffer is narrower than every
		// rune in the run.
		const content = "héllo—世界🌍ok"

		var chunks []string
		handler := sax.New()
		collect := func(_ context.Context, ch []byte) error {
			require.True(t, utf8.Valid(ch), "no chunk may contain a split (invalid) UTF-8 rune: %q", ch)
			chunks = append(chunks, string(ch))
			return nil
		}
		handler.SetOnCharacters(sax.CharactersFunc(collect))
		handler.SetOnIgnorableWhitespace(sax.IgnorableWhitespaceFunc(collect))

		_, err := helium.NewParser().SAXHandler(handler).CharBufferSize(1).
			Parse(context.Background(), []byte("<root>"+content+"</root>"))
		require.NoError(t, err)

		require.Equal(t, content, strings.Join(chunks, ""), "every byte is delivered intact")
	})

	// pins the chunked streaming-SAX
	// path's classification of an all-whitespace run that ends at an entity
	// reference ('&') rather than a start/end tag ('<'). The single-shot path
	// (areBlanksBytes) treats such a run as character data — it is ignorable
	// whitespace only when the delimiter that ends it is '<' or CR. An earlier
	// chunked implementation dropped that trailing-delimiter check and misreported
	// the leading blanks as IgnorableWhitespace. Both paths must agree.
	t.Run("whitespace before an entity", func(t *testing.T) {
		type event struct {
			ignorable bool
			content   string
		}

		run := func(src string, bufSize int) []event {
			var events []event
			handler := sax.New()
			handler.SetOnCharacters(sax.CharactersFunc(func(_ context.Context, ch []byte) error {
				events = append(events, event{ignorable: false, content: string(ch)})
				return nil
			}))
			handler.SetOnIgnorableWhitespace(sax.IgnorableWhitespaceFunc(func(_ context.Context, ch []byte) error {
				events = append(events, event{ignorable: true, content: string(ch)})
				return nil
			}))
			_, err := helium.NewParser().SAXHandler(handler).CharBufferSize(bufSize).
				Parse(context.Background(), []byte(src))
			require.NoError(t, err)
			return events
		}

		// Leading whitespace, then an entity reference. The run "   " ends at '&',
		// so it is character data, not ignorable whitespace.
		const src = `<root>   &amp;</root>`

		// Single-shot path (no chunking) is the reference classification.
		single := run(src, 0)
		require.NotEmpty(t, single)
		for _, e := range single {
			require.False(t, e.ignorable,
				"single-shot: a whitespace run ending at '&' is character data, not ignorable whitespace")
		}

		// Chunked path must match: no IgnorableWhitespace event, leading blanks
		// delivered as characters.
		chunked := run(src, 2)
		require.NotEmpty(t, chunked)
		var b strings.Builder
		for _, e := range chunked {
			require.False(t, e.ignorable,
				"chunked: a whitespace run ending at '&' must match the single-shot path (character data)")
			b.WriteString(e.content)
		}
		require.Equal(t, "   &", b.String(), "every character byte (including leading blanks) is delivered")
	})
}
