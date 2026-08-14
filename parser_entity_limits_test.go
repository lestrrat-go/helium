package helium_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"
)

// stoppableEBCDICEntityReader serves a finite EBCDIC head ending in an open
// entity-value literal (<!ENTITY e "), then an endless run of an EBCDIC filler
// byte that never reaches EOF, modeling a hostile never-ending internal-DTD
// entity value over the streaming EBCDIC path. The parser must terminate it via
// the per-node content cap WITHOUT buffering the tail.
type stoppableEBCDICEntityReader struct {
	head    []byte
	fill    byte
	pos     int
	stopped atomic.Bool
}

func (r *stoppableEBCDICEntityReader) Stop() { r.stopped.Store(true) }

func (r *stoppableEBCDICEntityReader) Read(p []byte) (int, error) {
	if r.stopped.Load() {
		return 0, io.EOF
	}
	if r.pos < len(r.head) {
		n := copy(p, r.head[r.pos:])
		r.pos += n
		return n, nil
	}
	for i := range p {
		p[i] = r.fill
	}
	return len(p), nil
}

func TestEntityAmplification(t *testing.T) {
	t.Parallel()

	t.Run("general entities", func(t *testing.T) {
		t.Run("billion laughs", func(t *testing.T) {
			t.Parallel()
			// Classic billion-laughs: 10 nested entities, each referencing 10 copies
			// of the previous. Total expansion: 10^10 bytes.
			xml := `<?xml version="1.0"?>
<!DOCTYPE lolz [
  <!ENTITY lol "lol">
  <!ENTITY lol2 "&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;">
  <!ENTITY lol3 "&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;">
  <!ENTITY lol4 "&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;">
  <!ENTITY lol5 "&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;">
  <!ENTITY lol6 "&lol5;&lol5;&lol5;&lol5;&lol5;&lol5;&lol5;&lol5;&lol5;&lol5;">
  <!ENTITY lol7 "&lol6;&lol6;&lol6;&lol6;&lol6;&lol6;&lol6;&lol6;&lol6;&lol6;">
  <!ENTITY lol8 "&lol7;&lol7;&lol7;&lol7;&lol7;&lol7;&lol7;&lol7;&lol7;&lol7;">
  <!ENTITY lol9 "&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;&lol8;">
]>
<root>&lol9;</root>`

			p := helium.NewParser().SubstituteEntities(true)
			_, err := p.Parse(t.Context(), []byte(xml))
			require.Error(t, err, "billion laughs should be rejected")
			require.Contains(t, err.Error(), "amplification")
		})

		t.Run("quadratic blowup", func(t *testing.T) {
			t.Parallel()
			// Large entity repeated many times: quadratic blowup.
			// helium.Entity content is 100KB, referenced 100 times → 10MB expansion from ~110KB input.
			bigContent := strings.Repeat("A", 100_000)
			refs := strings.Repeat("&big;", 100)
			xml := fmt.Sprintf(`<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY big "%s">
]>
<root>%s</root>`, bigContent, refs)

			p := helium.NewParser().SubstituteEntities(true)
			_, err := p.Parse(t.Context(), []byte(xml))
			require.Error(t, err, "quadratic blowup should be rejected")
			require.Contains(t, err.Error(), "amplification")
		})

		t.Run("normal entities", func(t *testing.T) {
			t.Parallel()
			// Small expansion well within limits — must succeed.
			xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY greeting "Hello, World!">
]>
<root>&greeting;</root>`

			p := helium.NewParser().SubstituteEntities(true)
			doc, err := p.Parse(t.Context(), []byte(xml))
			require.NoError(t, err)
			require.NotNil(t, doc)
		})

		t.Run("MaxEntityAmplification(-1) disables guard", func(t *testing.T) {
			t.Parallel()
			// With the ratio check disabled, billion laughs should be allowed.
			// Use a smaller version to avoid actual memory exhaustion.
			xml := `<?xml version="1.0"?>
<!DOCTYPE lolz [
  <!ENTITY lol "lol">
  <!ENTITY lol2 "&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;">
  <!ENTITY lol3 "&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;">
  <!ENTITY lol4 "&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;">
  <!ENTITY lol5 "&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;">
]>
<root>&lol5;</root>`

			p := helium.NewParser().SubstituteEntities(true).MaxEntityAmplification(-1)
			doc, err := p.Parse(t.Context(), []byte(xml))
			require.NoError(t, err)
			require.NotNil(t, doc)
		})

		// The absolute hard-ceiling behavior (it trips even with the ratio check
		// disabled) is covered by TestEntityHardCeiling in the internal test, which
		// lowers entityHardCeiling so it need not actually expand toward 1 GB.
	})

	// an external parsed entity's bytes
	// are charged to the amplification counters on EVERY reference via the cached
	// expandedSize, not just on the first read, AND that a single reference is
	// charged the per-reference fixed cost exactly once.
	//
	// The single-reference body is sized so that CORRECT accounting (one
	// entityFixedCost via entityCheck plus the raw bytes via entityCheckBytes) lands
	// just at/under the free baseline and succeeds, while a SECOND fixed cost — the
	// regression where the external content is charged via entityCheck instead of
	// entityCheckBytes — would cross the baseline and trip the amplification guard.
	// The subtest therefore fails if the accounting regresses to a double fixed cost.
	//
	// The repeated-references subtest proves the cached expandedSize is charged on
	// every reference: only when the entity is referenced many times does the
	// accumulated size trip the guard, and the FS Open count confirms the source is
	// read exactly once.
	t.Run("external entities", func(t *testing.T) {
		t.Run("single reference succeeds, fixed cost charged once", func(t *testing.T) {
			t.Parallel()
			// Size the body so a single reference's correct charge —
			// len(body) bytes (entityCheckBytes) + one entityFixedCost (entityCheck)
			// — sits just under the baseline. A regression that charged the external
			// content through entityCheck would add a SECOND fixed cost, pushing the
			// total over the baseline and tripping the ratio guard against this tiny
			// input. The 10-byte slack keeps the correct total strictly under the
			// baseline; a single extra fixed cost (20 bytes) crosses it.
			bodyLen := entityAllowedExpansionBytes - entityFixedCostBytes - 10
			body := strings.Repeat("A", int(bodyLen))

			var opens atomic.Int64
			const input = `<?xml version="1.0"?>
<!DOCTYPE r [<!ENTITY x SYSTEM "big.txt">]><r>&x;</r>`

			doc, err := helium.NewParser().BlockXXE(false).
				SubstituteEntities(true).
				FS(countingFS{data: body, opens: &opens}).
				Parse(t.Context(), []byte(input))
			require.NoError(t, err,
				"a single reference at the baseline must succeed; a second fixed cost would reject it")
			require.NotNil(t, doc)
			require.Equal(t, int64(1), opens.Load(), "the external source must be read exactly once")
		})

		t.Run("repeated references trip guard, source opened once", func(t *testing.T) {
			t.Parallel()
			// 800 KiB: comfortably under the 1 MB free baseline so one reference alone
			// never trips the ratio check.
			body := strings.Repeat("A", 800*1024)
			// Inert padding inside a comment so the input is "large", keeping the
			// amplification ratio from tripping on a single expansion while
			// contributing nothing to entity expansion.
			padding := strings.Repeat(" ", 200*1024)

			var opens atomic.Int64

			var refs strings.Builder
			for range 10 {
				refs.WriteString("&x;")
			}
			input := fmt.Sprintf(`<?xml version="1.0"?>
<!DOCTYPE r [<!ENTITY x SYSTEM "big.txt">]><r><!--%s-->%s</r>`, padding, refs.String())

			_, err := helium.NewParser().BlockXXE(false).
				SubstituteEntities(true).
				FS(countingFS{data: body, opens: &opens}).
				Parse(t.Context(), []byte(input))
			require.Error(t, err, "repeated references to a large external entity must trip the guard")
			require.Contains(t, err.Error(), "amplification",
				"error must explain the amplification limit, got: %v", err)
			require.Equal(t, int64(1), opens.Load(),
				"the external source must be read exactly once; repeats rely on cached accounting")
		})
	})

	// the direct parameter-entity
	// replacement path in parsePEReference: a large PE declared in the internal DTD
	// subset and referenced directly (%p;) as markup many times. Each reference
	// decodes the replacement text and pushes it as new input; the PE's OWN expanded
	// size must be charged to the amplification counters on every use, otherwise a
	// small DTD can drive unbounded expansion past the limit. Each PE expands to a
	// large comment (valid DTD markup), so the only growth is the replacement text
	// itself — no nested entity refs (which decodeEntities already charges) are
	// involved, isolating the direct-PE charge being verified here.
	t.Run("a direct PE reference", func(t *testing.T) {
		// ~100 KiB per expansion, referenced 200 times → ~20 MB of expansion from a
		// ~100 KiB subset. This crosses the 1 MiB baseline and trips the
		// amplification ratio guard relative to the tiny main document.
		big := strings.Repeat("A", 100_000)
		pe := "<!-- " + big + " -->"
		refs := strings.Repeat("%p;\n", 200)
		xml := `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE r [` + "\n" +
			`<!ENTITY % p "` + pe + `">` + "\n" +
			refs +
			`]>` + "\n" + `<r/>`

		_, err := helium.NewParser().Parse(t.Context(), []byte(xml))
		require.Error(t, err, "repeated direct PE expansion must trip the entity-expansion limit")
		require.Contains(t, err.Error(), "amplification",
			"error must explain the amplification limit, got: %v", err)
	})

	// is the regression for the PE-accounting
	// double-count bug. The PE %p; has a TINY literal replacement text ("<!-- &g;
	// -->") that expands ALMOST ENTIRELY through a nested GENERAL entity reference
	// &g; pointing at a large value. When %p; is referenced, parsePEReference
	// decodes its replacement via decodeEntities(SubstituteBoth), which ALREADY
	// charges the &g; expansion against the amplification counters. The PE-direct
	// charge must therefore account ONLY p's own literal replacement bytes
	// (len(entity.Content()), here ~12 bytes), NOT the full decoded length
	// (~100 KiB). Charging the decoded length double-counts g's expansion and
	// would falsely reject this legitimate DTD.
	//
	// Sizing keeps the CORRECT total (one charge of g per %p;, ~8*100 KiB ≈ 800 KiB)
	// below the 1 MiB amplification baseline so it must NOT be rejected, while the
	// OLD double-counting total (~1.6 MiB) crosses the baseline and trips the ratio
	// guard against the ~100 KiB input. A regression to the old accounting
	// (entityCheck on len(decodedContent)) brings this test back as a spurious
	// "amplification" rejection.
	t.Run("a nested PE reference is not double counted", func(t *testing.T) {
		big := strings.Repeat("A", 100_000)
		xml := `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE r [` + "\n" +
			`<!ENTITY g "` + big + `">` + "\n" +
			`<!ENTITY % p "<!-- &g; -->">` + "\n" +
			strings.Repeat("%p;\n", 8) +
			`]>` + "\n" + `<r/>`

		doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
		require.NoError(t, err,
			"a PE that expands mostly through a nested general entity must stay under the amplification limit; double-counting the nested expansion would falsely reject it")
		require.NotNil(t, doc)
	})

	// against a regression where
	// ParseFile delegated to the streaming reader path, leaving inputSize == 0. A
	// large internal entity referenced exactly once is legitimate (no
	// amplification) and must parse, but a zero inputSize made the
	// amplification-ratio guard fall back to a divisor of 1 and falsely reject it.
	// ParseFile knows the file size, so it must seed inputSize like Parse([]byte)
	// does and produce the same result.
	t.Run("a large parsed file is not falsely amplified", func(t *testing.T) {
		// Entity content just over the 1 MiB ratio-check baseline
		// (entityAllowedExpansion), referenced exactly once. The expansion is far
		// below the file size, so the amplification ratio is well under the limit.
		bigContent := strings.Repeat("A", 1_500_000)
		xml := fmt.Sprintf(`<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY big "%s">
]>
<root>&big;</root>`, bigContent)

		// Baseline: Parse([]byte) must accept this (inputSize == len(xml)).
		doc, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(xml))
		require.NoError(t, err, "Parse([]byte) must accept a large entity referenced once")
		require.NotNil(t, doc)

		dir := t.TempDir()
		path := filepath.Join(dir, "big-entity.xml")
		require.NoError(t, os.WriteFile(path, []byte(xml), 0o600))

		// ParseFile must match Parse([]byte): the guard must not falsely trip.
		fileDoc, err := helium.NewParser().SubstituteEntities(true).ParseFile(t.Context(), path)
		require.NoError(t, err, "ParseFile must accept the same large-entity document as Parse([]byte)")
		require.NotNil(t, fileDoc)
	})
}

func TestEntityDepthLimit(t *testing.T) {
	t.Parallel()

	t.Run("nesting depth", func(t *testing.T) {
		// Build deeply nested entity references (depth > 40).
		var dtd strings.Builder
		dtd.WriteString(`<?xml version="1.0"?>` + "\n" + `<!DOCTYPE root [` + "\n")
		dtd.WriteString(`  <!ENTITY e0 "x">` + "\n")
		for i := 1; i <= 45; i++ {
			fmt.Fprintf(&dtd, "  <!ENTITY e%d \"&e%d;\">\n", i, i-1)
		}
		dtd.WriteString("]>\n")
		dtd.WriteString("<root>&e45;</root>")

		p := helium.NewParser().SubstituteEntities(true).MaxEntityAmplification(-1) // disable amplification guard to test depth only
		_, err := p.Parse(t.Context(), []byte(dtd.String()))
		require.Error(t, err, "depth > 40 should still error")
		require.Contains(t, err.Error(), "entity loop")
	})

	// confirms the attribute-value WFC
	// walk traverses a long acyclic chain of nested internal entities without native
	// call-stack recursion (the walker uses an explicit work stack) and does not
	// false-reject a harmless plain-text terminus.
	t.Run("a deep chain in an attribute value is bounded", func(t *testing.T) {
		var b strings.Builder
		b.WriteString("<!DOCTYPE r [\n<!ELEMENT r EMPTY>\n")
		const depth = 30 // stays under the entity-expansion depth guard
		for i := range depth {
			fmt.Fprintf(&b, "<!ENTITY e%d \"&e%d;\">\n", i, i+1)
		}
		fmt.Fprintf(&b, "<!ENTITY e%d \"end\">\n<!ATTLIST r a CDATA \"&e0;\">\n]>\n<r/>", depth)

		doc, err := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).
			DefaultDTDAttributes(true).SubstituteEntities(false).ValidateDTD(false).
			Parse(t.Context(), []byte(b.String()))
		require.NoError(t, err, "a harmless deep entity chain must be accepted")
		require.NotNil(t, doc)
	})
}

func TestEntitySizeCap(t *testing.T) {
	t.Parallel()

	// ensures that an external parsed entity whose content
	// exceeds the size cap is rejected with the specific size-cap error, and never
	// read fully via io.ReadAll, and that the resolved input is closed. The source
	// is finite (cap+1 bytes) so a regression of the guard cannot hang or OOM the
	// test; it would instead fail the specific-error assertion.
	t.Run("an external entity is capped", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE r [<!ENTITY x SYSTEM "big">]><r>&x;</r>`

		var closed atomic.Bool
		p := helium.NewParser().BlockXXE(false).
			SubstituteEntities(true).
			FS(finiteFS{size: externalEntityMaxBytes + 1, closed: &closed})

		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "oversized external entity must be rejected")
		require.Contains(t, err.Error(), "exceeds maximum size",
			"error must explain the size cap, got: %v", err)
		require.True(t, closed.Load(), "resolved external entity input must be closed")
	})

	// the internal-DTD entity-value
	// literal scanner (parseEntityValueInternal): a giant entity value is an
	// indivisible content run that must be bounded by the per-node content cap, both
	// when the literal is properly terminated and when it runs unterminated toward
	// EOF. Before the fix this scanner had no cap and peeked an ever-growing offset,
	// so an internal DTD (which parses by default) could grow memory without bound.
	t.Run("an entity-value literal", func(t *testing.T) {
		const limit = 64

		t.Run("terminated over-cap entity value fails closed", func(t *testing.T) {
			t.Parallel()
			body := strings.Repeat("a", 200)
			doc := `<!DOCTYPE r [<!ENTITY e "` + body + `">]><r/>`
			_, err := helium.NewParser().
				MaxNodeContentSize(limit).
				Parse(t.Context(), []byte(doc))
			require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
		})

		t.Run("unterminated over-cap entity value fails closed (Parse)", func(t *testing.T) {
			t.Parallel()
			// No closing quote: the scanner runs to EOF. The cap must trip before the
			// whole run is buffered, so nothing grows unbounded.
			body := strings.Repeat("a", 200)
			doc := `<!DOCTYPE r [<!ENTITY e "` + body
			_, err := helium.NewParser().
				MaxNodeContentSize(limit).
				Parse(t.Context(), []byte(doc))
			require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
		})

		t.Run("unterminated over-cap entity value fails closed (ParseReader)", func(t *testing.T) {
			t.Parallel()
			body := strings.Repeat("a", 200)
			doc := `<!DOCTYPE r [<!ENTITY e "` + body
			_, err := helium.NewParser().
				MaxNodeContentSize(limit).
				ParseReader(t.Context(), strings.NewReader(doc))
			require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
		})

		t.Run("within-cap entity value parses fine", func(t *testing.T) {
			t.Parallel()
			doc := `<!DOCTYPE r [<!ENTITY e "` + strings.Repeat("a", 32) + `">]><r>&e;</r>`
			d, err := helium.NewParser().
				MaxNodeContentSize(limit).
				Parse(t.Context(), []byte(doc))
			require.NoError(t, err)
			require.NotNil(t, d)
		})
	})

	// the SYSTEM/PUBLIC
	// literal scanners reached through an external ENTITY declaration in the
	// internal subset (parseEntityDecl -> parseExternalID). A giant system or public
	// literal in an external general or parameter entity declaration must fail closed
	// with the per-node content cap, buffering nothing unbounded; the generic
	// "value required" message must not mask the resource-limit error.
	t.Run("an external entity declaration literal", func(t *testing.T) {
		const limit = 64
		body := strings.Repeat("a", 200)

		cases := []struct {
			name string
			doc  string
		}{
			{
				name: "external general entity SYSTEM over-cap",
				doc:  `<!DOCTYPE r [<!ENTITY e SYSTEM "` + body + `">]><r/>`,
			},
			{
				name: "external parameter entity SYSTEM over-cap",
				doc:  `<!DOCTYPE r [<!ENTITY % e SYSTEM "` + body + `">]><r/>`,
			},
			{
				name: "external parameter entity PUBLIC over-cap system literal",
				doc:  `<!DOCTYPE r [<!ENTITY % e PUBLIC "pub" "` + body + `">]><r/>`,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name+" (Parse)", func(t *testing.T) {
				t.Parallel()
				_, err := helium.NewParser().
					MaxNodeContentSize(limit).
					Parse(t.Context(), []byte(tc.doc))
				require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
			})

			t.Run(tc.name+" (ParseReader)", func(t *testing.T) {
				t.Parallel()
				_, err := helium.NewParser().
					MaxNodeContentSize(limit).
					ParseReader(t.Context(), strings.NewReader(tc.doc))
				require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
			})
		}
	})

	// the SYSTEM/PUBLIC literal
	// scanners (parseSystemLiteral/parsePubidLiteral) reached through the DOCTYPE
	// external ID. A giant system or public literal must fail closed with the
	// per-node content cap, buffering nothing unbounded; the generic "URI
	// required" message must not mask the resource-limit error.
	t.Run("an external ID literal", func(t *testing.T) {
		const limit = 64

		t.Run("over-cap system literal fails closed", func(t *testing.T) {
			t.Parallel()
			body := strings.Repeat("a", 200)
			doc := `<!DOCTYPE r SYSTEM "` + body + `"><r/>`
			_, err := helium.NewParser().
				MaxNodeContentSize(limit).
				Parse(t.Context(), []byte(doc))
			require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
		})

		t.Run("over-cap public literal fails closed", func(t *testing.T) {
			t.Parallel()
			body := strings.Repeat("a", 200)
			doc := `<!DOCTYPE r PUBLIC "` + body + `" "sys"><r/>`
			_, err := helium.NewParser().
				MaxNodeContentSize(limit).
				Parse(t.Context(), []byte(doc))
			require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
		})

		t.Run("within-cap external id parses fine", func(t *testing.T) {
			t.Parallel()
			doc := `<!DOCTYPE r PUBLIC "` + strings.Repeat("a", 16) + `" "` + strings.Repeat("b", 16) + `"><r/>`
			d, err := helium.NewParser().
				MaxNodeContentSize(limit).
				Parse(t.Context(), []byte(doc))
			require.NoError(t, err)
			require.NotNil(t, d)
		})
	})

	// a finite EBCDIC
	// document with a (within-cap) entity value parses identically via ParseReader
	// and Parse([]byte), so the bounded literal scanner did not change normal output.
	t.Run("an EBCDIC entity value matches Parse", func(t *testing.T) {
		const xml = `<?xml version="1.0" encoding="IBM037"?><!DOCTYPE r [<!ENTITY e "hello">]><r>&e;</r>`
		ebcdic, err := charmap.CodePage037.NewEncoder().Bytes([]byte(xml))
		require.NoError(t, err)

		serialize := func(doc *helium.Document) string {
			var buf bytes.Buffer
			require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
			return buf.String()
		}

		bytesDoc, err := helium.NewParser().Parse(t.Context(), ebcdic)
		require.NoError(t, err)
		want := serialize(bytesDoc)

		readerDoc, err := helium.NewParser().ParseReader(t.Context(), bytes.NewReader(ebcdic))
		require.NoError(t, err)
		require.Equal(t, want, serialize(readerDoc))
	})

	// the parser.go
	// "unbounded EBCDIC streams are bounded by parser caps" claim holds for the
	// internal-DTD entity-value literal scanner: an EBCDIC stream whose entity value
	// never terminates is bounded by MaxNodeContentSize and fails with
	// ErrNodeContentTooLarge, never buffered whole into memory.
	t.Run("an unbounded EBCDIC entity value is bounded by the node cap", func(t *testing.T) {
		const decl = `<?xml version="1.0" encoding="IBM037"?><!DOCTYPE r [<!ENTITY e "`
		head, err := charmap.CodePage037.NewEncoder().Bytes([]byte(decl))
		require.NoError(t, err)
		require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, head[:4],
			"encoded head must start with the EBCDIC invariant prefix")

		fill, err := charmap.CodePage037.NewEncoder().Bytes([]byte("a"))
		require.NoError(t, err)

		r := &stoppableEBCDICEntityReader{head: head, fill: fill[0]}
		defer r.Stop()

		errCh := make(chan error, 1)
		go func() {
			_, perr := helium.NewParser().MaxNodeContentSize(4096).ParseReader(t.Context(), r)
			errCh <- perr
		}()

		select {
		case perr := <-errCh:
			require.ErrorIs(t, perr, helium.ErrNodeContentTooLarge,
				"an unbounded EBCDIC entity value must be bounded by the per-node content cap")
		case <-time.After(5 * time.Second):
			r.Stop()
			t.Fatal("ParseReader did not terminate an unbounded EBCDIC entity value within the timeout")
		}
	})
}
