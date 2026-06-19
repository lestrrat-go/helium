package helium_test

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"

	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// externalEntityMaxBytes mirrors the unexported cap in parserctx.go. The size
// guard is what this test exercises; keep the two in sync.
const externalEntityMaxBytes int64 = 10 * 1024 * 1024 // 10 MiB

// entityAllowedExpansionBytes and entityFixedCostBytes mirror the unexported
// amplification constants in parserctx.go (entityAllowedExpansion and
// entityFixedCost). They let the single-reference accounting test sit exactly
// at the boundary where charging a second fixed cost would cross the baseline.
// Keep them in sync with parserctx.go.
const (
	entityAllowedExpansionBytes int64 = 1_000_000 // 1 MB baseline before ratio check
	entityFixedCostBytes        int64 = 20        // fixed byte cost per entity reference
)

// finiteFile is an fs.File that yields exactly n bytes of 'A' and then io.EOF.
// Unlike an unbounded reader, it cannot hang or OOM if the size guard ever
// regresses: a finite (cap+1) source still trips the cap deterministically. It
// records whether Close was called.
type finiteFile struct {
	remaining int64
	closed    *atomic.Bool
}

func (f *finiteFile) Stat() (fs.FileInfo, error) { return nil, fs.ErrInvalid }

func (f *finiteFile) Read(p []byte) (int, error) {
	if f.remaining <= 0 {
		return 0, io.EOF
	}
	n := min(int64(len(p)), f.remaining)
	for i := range n {
		p[i] = 'A'
	}
	f.remaining -= n
	return int(n), nil
}

func (f *finiteFile) Close() error {
	f.closed.Store(true)
	return nil
}

// finiteFS hands out a single finiteFile of the configured size, recording
// closure.
type finiteFS struct {
	size   int64
	closed *atomic.Bool
}

func (s finiteFS) Open(string) (fs.File, error) {
	return &finiteFile{remaining: s.size, closed: s.closed}, nil
}

// TestExternalEntitySizeCap ensures that an external parsed entity whose content
// exceeds the size cap is rejected with the specific size-cap error (rather than
// read fully via io.ReadAll), and that the resolved input is closed. The source
// is finite (cap+1 bytes) so a regression of the guard cannot hang or OOM the
// test; it would instead fail the specific-error assertion.
func TestExternalEntitySizeCap(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE r [<!ENTITY x SYSTEM "big">]><r>&x;</r>`

	var closed atomic.Bool
	p := helium.NewParser().
		SubstituteEntities(true).
		FS(finiteFS{size: externalEntityMaxBytes + 1, closed: &closed})

	_, err := p.Parse(t.Context(), []byte(input))
	require.Error(t, err, "oversized external entity must be rejected")
	require.Contains(t, err.Error(), "exceeds maximum size",
		"error must explain the size cap, got: %v", err)
	require.True(t, closed.Load(), "resolved external entity input must be closed")
}

// readCloserFile wraps a string and records Close, used to verify external
// entity inputs are closed even on the success path.
type readCloserFile struct {
	r      *io.SectionReader
	closed *atomic.Bool
}

func (f *readCloserFile) Stat() (fs.FileInfo, error) { return nil, fs.ErrInvalid }
func (f *readCloserFile) Read(p []byte) (int, error) { return f.r.Read(p) }
func (f *readCloserFile) Close() error {
	f.closed.Store(true)
	return nil
}

type closingFS struct {
	data   string
	closed *atomic.Bool
}

func (s closingFS) Open(string) (fs.File, error) {
	return &readCloserFile{
		r:      io.NewSectionReader(strings.NewReader(s.data), 0, int64(len(s.data))),
		closed: s.closed,
	}, nil
}

func TestExternalEntityInputClosed(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE r [<!ENTITY x SYSTEM "ext">]><r>&x;</r>`

	var closed atomic.Bool
	p := helium.NewParser().SubstituteEntities(true).FS(closingFS{data: "<e>ok</e>", closed: &closed})
	_, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	require.True(t, closed.Load(), "resolved external entity input must be closed on success")
}

// orderingFS resolves two external entities, "ext" and "ext2", and records the
// closed state of "ext" at the moment "ext2" is opened. "ext"'s buffered content
// references "ext2" (&y;), so "ext2" is opened only while "ext"'s already-read
// content is being parsed. If the parser closes "ext" promptly — at the read
// boundary, before parsing the buffered content — then "ext" is already closed
// by the time "ext2" is opened. If it deferred the close to function return,
// "ext" would still be open here.
type orderingFS struct {
	extClosed           *atomic.Bool
	extClosedAtNestOpen *atomic.Bool
	nestOpened          *atomic.Bool
}

func (s orderingFS) Open(name string) (fs.File, error) {
	if name == "ext2" {
		// "ext2" is opened only during the parse of "ext"'s buffered content.
		// Capture whether "ext" was already closed at this instant.
		s.nestOpened.Store(true)
		s.extClosedAtNestOpen.Store(s.extClosed.Load())
		const data = "<inner/>"
		return &readCloserFile{
			r:      io.NewSectionReader(strings.NewReader(data), 0, int64(len(data))),
			closed: &atomic.Bool{},
		}, nil
	}
	// "ext": its buffered content references the nested external entity &y;.
	const data = "<e>&y;</e>"
	return &readCloserFile{
		r:      io.NewSectionReader(strings.NewReader(data), 0, int64(len(data))),
		closed: s.extClosed,
	}, nil
}

// TestExternalEntityClosedBeforeContentParsed proves the resolved external input
// is closed at the read boundary — BEFORE its already-buffered content is parsed
// — not merely before Parse returns. The first entity's content references a
// second external entity, so the second entity's Open happens mid-parse of the
// first entity's buffered bytes; at that point the first input must already be
// closed.
func TestExternalEntityClosedBeforeContentParsed(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE r [
  <!ENTITY x SYSTEM "ext">
  <!ENTITY y SYSTEM "ext2">
]><r>&x;</r>`

	var extClosed, extClosedAtNestOpen, nestOpened atomic.Bool
	p := helium.NewParser().SubstituteEntities(true).FS(orderingFS{
		extClosed:           &extClosed,
		extClosedAtNestOpen: &extClosedAtNestOpen,
		nestOpened:          &nestOpened,
	})
	_, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	require.True(t, nestOpened.Load(),
		"nested external entity must be opened while the first entity's content is parsed")
	require.True(t, extClosedAtNestOpen.Load(),
		"the first external input must be closed BEFORE its buffered content is parsed")
}

// countingFS hands out the same byte content on every Open and records how many
// times Open was called, so a test can prove that repeated references to one
// external entity read the source only once (the rest hit the cached
// expandedSize accounting).
type countingFS struct {
	data  string
	opens *atomic.Int64
}

func (s countingFS) Open(string) (fs.File, error) {
	s.opens.Add(1)
	return &readCloserFile{
		r:      io.NewSectionReader(strings.NewReader(s.data), 0, int64(len(s.data))),
		closed: &atomic.Bool{},
	}, nil
}

// TestExternalEntityAmplification proves that an external parsed entity's bytes
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
func TestExternalEntityAmplification(t *testing.T) {
	t.Parallel()

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

		doc, err := helium.NewParser().
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

		_, err := helium.NewParser().
			SubstituteEntities(true).
			FS(countingFS{data: body, opens: &opens}).
			Parse(t.Context(), []byte(input))
		require.Error(t, err, "repeated references to a large external entity must trip the guard")
		require.Contains(t, err.Error(), "amplification",
			"error must explain the amplification limit, got: %v", err)
		require.Equal(t, int64(1), opens.Load(),
			"the external source must be read exactly once; repeats rely on cached accounting")
	})
}

// TestEntityValueMalformedGeneralRef ensures that a general reference inside an
// EntityValue is syntax-checked: a missing semicolon must be rejected even
// though the general reference itself is not expanded.
func TestEntityValueMalformedGeneralRef(t *testing.T) {
	t.Parallel()

	const input = `<!DOCTYPE r [<!ENTITY e "&broken">]><r/>`

	p := helium.NewParser()
	_, err := p.Parse(t.Context(), []byte(input))
	require.Error(t, err, "malformed general reference in entity value must be rejected")
}

// TestEntityValueDirectCharRefMalformedGeneralRef ensures that a DIRECT
// character reference adjacent to a bare '&' or a name does not synthesize a
// well-formed general reference. A direct char ref is character data; it must
// never combine with surrounding text to manufacture a "&Name;". Both repros
// would be wrongly accepted if direct char refs were resolved into the
// validation stream rather than treated as inert character data.
func TestEntityValueDirectCharRefMalformedGeneralRef(t *testing.T) {
	t.Parallel()

	t.Run("char ref completes a bare ampersand name", func(t *testing.T) {
		t.Parallel()
		// "&&#97;;" must NOT be read as "&a;": the first '&' is a bare ampersand
		// (malformed) and "&#97;" is character data.
		const input = `<!DOCTYPE r [<!ENTITY e "&&#97;;">]><r/>`
		_, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.Error(t, err,
			"a char ref must not complete a bare '&' into a general reference")
	})

	t.Run("char ref supplies a trailing semicolon", func(t *testing.T) {
		t.Parallel()
		// "&a&#59;" must NOT be read as "&a;": the trailing ';' is character data
		// (&#59;), not the terminator of a general reference.
		const input = `<!DOCTYPE r [<!ENTITY e "&a&#59;">]><r/>`
		_, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.Error(t, err,
			"a char ref must not supply the ';' that completes a general reference")
	})
}

// TestEntityValueDirectCharRefAccepted ensures that a legitimately valid
// EntityValue containing a direct character reference is still accepted and
// stored literally (not expanded). The inert-placeholder treatment in the
// reference-validation pass must not reject valid char refs.
func TestEntityValueDirectCharRefAccepted(t *testing.T) {
	t.Parallel()

	t.Run("standalone char ref", func(t *testing.T) {
		t.Parallel()
		const input = `<!DOCTYPE r [<!ENTITY e "x&#97;y">]><r/>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.NoError(t, err, "a standalone direct char ref must be accepted")
		require.NotNil(t, doc)

		ent, ok := doc.GetEntity("e")
		require.True(t, ok, "entity e must be declared")
		// A direct character reference is character data: it is resolved to its
		// character in the stored value ("&#97;" -> "a"), unlike a general
		// reference such as "&amp;" which is stored verbatim.
		require.Equal(t, "xay", string(ent.Content()),
			"direct char refs are character data, resolved in the stored value")
	})

	t.Run("predefined amp entity", func(t *testing.T) {
		t.Parallel()
		const input = `<!DOCTYPE r [<!ENTITY e "&amp;">]><r/>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.NoError(t, err, "a well-formed &amp; must be accepted")
		require.NotNil(t, doc)

		ent, ok := doc.GetEntity("e")
		require.True(t, ok, "entity e must be declared")
		require.Equal(t, "&amp;", string(ent.Content()),
			"general references must be stored literally, not expanded")
	})

	t.Run("direct ampersand char ref is inert", func(t *testing.T) {
		t.Parallel()
		// A direct "&#38;" is character data: it resolves to a literal '&' in
		// the stored value but does NOT combine with the following NameChars
		// into a synthesized general reference. The reference-validation pass
		// must therefore treat the direct char ref as inert and accept the
		// declaration, unlike "&broken" written directly (a malformed ref) or a
		// '&' re-introduced through a parameter entity.
		const input = `<!DOCTYPE r [<!ENTITY e "&#38;broken">]><r/>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.NoError(t, err,
			"a direct &#38; must be inert and not synthesize a malformed &broken ref")
		require.NotNil(t, doc)

		ent, ok := doc.GetEntity("e")
		require.True(t, ok, "entity e must be declared")
		require.Equal(t, "&broken", string(ent.Content()),
			"direct char refs are resolved in the stored value (&#38; -> &), not expanded as a general ref")
	})
}

// TestEntityValueValidGeneralRefLiteral ensures that a well-formed general
// reference in an EntityValue is accepted AND stored literally (not expanded):
// the stored entity content must still contain "&amp; &good;" verbatim.
func TestEntityValueValidGeneralRefLiteral(t *testing.T) {
	t.Parallel()

	const input = `<!DOCTYPE r [<!ENTITY e "&amp; &good;">]><r/>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
	require.NoError(t, err, "well-formed general references in entity value must be accepted")
	require.NotNil(t, doc)

	ent, ok := doc.GetEntity("e")
	require.True(t, ok, "entity e must be declared")
	require.Equal(t, "&amp; &good;", string(ent.Content()),
		"general references must be stored literally, not expanded")
}

// TestEntityValueMalformedGeneralRefViaPE ensures that a malformed general
// reference re-introduced through a parameter-entity reference is rejected, even
// though the parameter entity itself only contributes a character reference. The
// EntityValue "%amp;broken" with "%amp;" -> "&#38;" -> "&" expands to "&broken",
// which is malformed and must be rejected (matching libxml2/xmllint), whereas a
// direct "&#38;" in an EntityValue is character data and is accepted.
func TestEntityValueMalformedGeneralRefViaPE(t *testing.T) {
	t.Parallel()

	t.Run("internal subset", func(t *testing.T) {
		t.Parallel()
		// helium recognizes PE references in the internal subset (more permissive
		// than the XML WFC), which lets us drive PE expansion through a path that
		// propagates the validation error rather than swallowing it.
		good := `<!DOCTYPE r [<!ENTITY % p "&#38;amp;"><!ENTITY e "%p; ok">]><r/>`
		_, errGood := helium.NewParser().Parse(t.Context(), []byte(good))
		require.NoError(t, errGood,
			"a well-formed reference produced via a PE must be accepted")

		bad := `<!DOCTYPE r [<!ENTITY % amp "&#38;"><!ENTITY e "%amp;broken">]><r/>`
		_, errBad := helium.NewParser().Parse(t.Context(), []byte(bad))
		require.Error(t, errBad,
			"a malformed reference produced via a PE must be rejected")
		require.Contains(t, errBad.Error(), "malformed entity reference in entity value")
	})

	t.Run("external subset", func(t *testing.T) {
		t.Parallel()
		// External DTD repro from the issue: a PE expands to "&" which combines
		// with following text into the malformed reference "&broken". The
		// malformed entity (e) must not be stored, while a control entity (c)
		// declared before it is stored, proving the parse reaches the entities and
		// the rejection is specific to the malformed declaration.
		//
		// The malformed per-declaration error in the external subset must now
		// surface as a top-level parse error rather than being swallowed.
		fsys := fstest.MapFS{
			"d.dtd": {Data: []byte(
				`<!ENTITY c "control">` + "\n" +
					`<!ENTITY % amp "&#38;">` + "\n" +
					`<!ENTITY e "%amp;broken">`)},
		}
		const input = `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

		_, err := helium.NewParser().
			LoadExternalDTD(true).
			FS(fsys).
			Parse(t.Context(), []byte(input))
		require.Error(t, err, "malformed entity reference in an external subset declaration must surface as a parse error")
		require.Contains(t, err.Error(), "malformed entity reference in entity value")
	})
}

// TestEntityValueRefValidationIsSideEffectFree proves that the reference
// validation in parseEntityValue does NOT perturb the entity-amplification
// accounting. validateEntityValueRefs PE-expands the literal to scan for
// malformed general references, and the real PE substitution that follows
// expands the same literal again. Each expansion charges the amplification
// counters; validateEntityValueRefs must snapshot and restore sizeentcopy so
// the same parameter entity is not charged twice.
//
// The repro is an external DTD that declares a parameter entity just under the
// 1 MiB baseline and a general entity referencing it via %p;. A single charge
// stays under the baseline (no ratio check); a double charge crosses the
// baseline and trips the amplification-ratio guard against the tiny main-
// document input, which would reject the declaration. The entity therefore
// being stored successfully is what proves the validation pass is side-effect
// free.
func TestEntityValueRefValidationIsSideEffectFree(t *testing.T) {
	t.Parallel()

	// Just under the 1 MiB amplification baseline (entityAllowedExpansion):
	// one expansion stays below it, a double charge crosses it.
	big := strings.Repeat("A", 1_000_000-100)
	dtd := `<!ENTITY c "control">` + "\n" +
		`<!ENTITY % p "` + big + `">` + "\n" +
		`<!ENTITY e "%p;">`
	fsys := fstest.MapFS{"d.dtd": {Data: []byte(dtd)}}
	input := `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

	doc, err := helium.NewParser().
		LoadExternalDTD(true).
		FS(fsys).
		Parse(t.Context(), []byte(input))
	require.NoError(t, err, "a near-baseline parameter entity must not trip the amplification guard")
	require.NotNil(t, doc)

	_, cOK := doc.GetEntity("c")
	require.True(t, cOK, "control entity must be stored, proving the external DTD was loaded")

	ent, eOK := doc.GetEntity("e")
	require.True(t, eOK,
		"a PE-expanding entity just under the baseline must be stored; double-charging would reject it")
	require.Equal(t, len(big), len(ent.Content()),
		"the stored value is the single PE expansion, not a doubled one")
}

// TestDirectPEReferenceAmplification exercises the direct parameter-entity
// replacement path in parsePEReference: a large PE declared in the internal DTD
// subset and referenced directly (%p;) as markup many times. Each reference
// decodes the replacement text and pushes it as new input; the PE's OWN expanded
// size must be charged to the amplification counters on every use, otherwise a
// small DTD can drive unbounded expansion past the limit. Each PE expands to a
// large comment (valid DTD markup), so the only growth is the replacement text
// itself — no nested entity refs (which decodeEntities already charges) are
// involved, isolating the direct-PE charge being verified here.
func TestDirectPEReferenceAmplification(t *testing.T) {
	t.Parallel()

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
}

// TestNestedPEReferenceNoDoubleCount is the regression for the PE-accounting
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
func TestNestedPEReferenceNoDoubleCount(t *testing.T) {
	t.Parallel()

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
}

func TestEntityAmplification(t *testing.T) {
	t.Parallel()

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

	t.Run("RelaxLimits disables guard", func(t *testing.T) {
		t.Parallel()
		// With RelaxLimits, billion laughs should be allowed (guard disabled).
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

		p := helium.NewParser().SubstituteEntities(true).RelaxLimits(true)
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
		require.NotNil(t, doc)
	})

	t.Run("RelaxLimits still capped by absolute ceiling", func(t *testing.T) {
		// Intentionally NOT t.Parallel: this subtest drives expansion up
		// to entityHardCeiling (1 GiB). Running it alongside the parallel
		// subtests above amplified peak memory under loaded CI runners.
		// The ceiling does eventually trip, but the parser still
		// materializes nontrivial intermediate state, so we serialize it.
		// A bigger billion-laughs that would expand to many GB even with
		// the ratio check disabled. The absolute ceiling (entityHardCeiling
		// in parserctx.go) must still trip and abort the parse.
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

		p := helium.NewParser().SubstituteEntities(true).RelaxLimits(true)
		_, err := p.Parse(t.Context(), []byte(xml))
		require.Error(t, err, "absolute ceiling must trip even with RelaxLimits")
		require.Contains(t, err.Error(), "maximum entity expansion size",
			"error must explain the ceiling, got: %v", err)
		require.Regexp(t, `\(\d+ > \d+\)`, err.Error(),
			"error must include observed and configured sizes for diagnosis, got: %v", err)
	})
}

func TestPredefinedEntities(t *testing.T) {
	// Predefined entities (&lt; &gt; &amp; &apos; &quot;) must never trigger the guard.
	xml := `<?xml version="1.0"?>
<root>&lt;&gt;&amp;&apos;&quot;</root>`

	p := helium.NewParser()
	doc, err := p.Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	require.NotNil(t, doc)
}

func TestPredefinedEntityRedeclaration(t *testing.T) {
	t.Run("valid redeclaration accepted", func(t *testing.T) {
		// §4.6: redeclaring lt with content "<" (via &#60;) is allowed.
		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY lt "&#60;">
]>
<root>&lt;</root>`
		_, err := helium.NewParser().Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
	})

	t.Run("invalid redeclaration rejected", func(t *testing.T) {
		// §4.6: redeclaring lt with wrong content is a hard error.
		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY lt "X">
]>
<root>&lt;</root>`
		_, err := helium.NewParser().Parse(t.Context(), []byte(xml))
		require.Error(t, err)
		require.Contains(t, err.Error(), "redeclared")
	})

	t.Run("valid redeclaration with char ref accepted", func(t *testing.T) {
		// Content is &#60; (char ref for <), which resolves to <
		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY lt "&#60;">
  <!ENTITY gt "&#62;">
  <!ENTITY amp "&#38;">
  <!ENTITY apos "&#39;">
  <!ENTITY quot "&#34;">
]>
<root/>`
		_, err := helium.NewParser().Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
	})

	t.Run("DTD.AddEntity rejects wrong content", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		dtd, err := doc.CreateDTD()
		require.NoError(t, err)
		_, err = dtd.AddEntity("amp", enum.InternalGeneralEntity, "", "", "wrong")
		require.Error(t, err)
		require.Contains(t, err.Error(), "redeclared")
	})

	t.Run("DTD.AddEntity accepts correct content", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		dtd, err := doc.CreateDTD()
		require.NoError(t, err)
		_, err = dtd.AddEntity("amp", enum.InternalGeneralEntity, "", "", "&")
		require.NoError(t, err)
	})

	t.Run("DTD.AddEntity accepts char ref content", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		dtd, err := doc.CreateDTD()
		require.NoError(t, err)
		// &#60; resolves to <
		_, err = dtd.AddEntity("lt", enum.InternalGeneralEntity, "", "", "&#60;")
		require.NoError(t, err)
	})
}

func TestUndeclaredEntityFatal(t *testing.T) {
	t.Parallel()

	// An undeclared general entity reference, with no DTD/external subset
	// and no parameter-entity references, is a fatal well-formedness error.
	xml := `<r>&bogus;</r>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.Error(t, err, "undeclared entity with no DTD must be fatal")
	require.Nil(t, doc, "no document should be returned on a fatal error")
	require.ErrorIs(t, err, helium.ErrUndeclaredEntity, "error must be the undeclared-entity sentinel")

	var pe helium.ErrParseError
	require.True(t, errors.As(err, &pe), "error must be an ErrParseError")
	require.Equal(t, helium.ErrorLevelFatal, pe.Level, "undeclared entity must be a fatal-level error")
}

func TestEntityDepthLimit(t *testing.T) {
	// Build deeply nested entity references (depth > 40).
	var dtd strings.Builder
	dtd.WriteString(`<?xml version="1.0"?>` + "\n" + `<!DOCTYPE root [` + "\n")
	dtd.WriteString(`  <!ENTITY e0 "x">` + "\n")
	for i := 1; i <= 45; i++ {
		fmt.Fprintf(&dtd, "  <!ENTITY e%d \"&e%d;\">\n", i, i-1)
	}
	dtd.WriteString("]>\n")
	dtd.WriteString("<root>&e45;</root>")

	p := helium.NewParser().SubstituteEntities(true).RelaxLimits(true) // disable amplification guard to test depth only
	_, err := p.Parse(t.Context(), []byte(dtd.String()))
	require.Error(t, err, "depth > 40 should still error")
	require.Contains(t, err.Error(), "entity loop")
}
