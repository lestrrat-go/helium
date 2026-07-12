package helium_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/iofs"
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
	p := helium.NewParser().BlockXXE(false).
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
	p := helium.NewParser().BlockXXE(false).SubstituteEntities(true).FS(closingFS{data: "<e>ok</e>", closed: &closed})
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
	p := helium.NewParser().BlockXXE(false).SubstituteEntities(true).FS(orderingFS{
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
		// A PE reference inside an entity value in the internal subset violates
		// the PEs in Internal Subset WFC (XML §2.8) and is rejected outright,
		// before any general-reference well-formedness of the (would-be)
		// replacement text is considered. This holds whether the PE would expand
		// to a well-formed or a malformed general reference. The malformed
		// general-reference-via-PE path itself is exercised in the external
		// subset subtest, where PE references in entity values are permitted.
		wouldBeGood := `<!DOCTYPE r [<!ENTITY % p "&#38;amp;"><!ENTITY e "%p; ok">]><r/>`
		_, errGood := helium.NewParser().Parse(t.Context(), []byte(wouldBeGood))
		require.Error(t, errGood,
			"a PE reference in an internal-subset entity value is not well formed")
		require.Contains(t, errGood.Error(), "PEReferences forbidden in internal subset")

		wouldBeBad := `<!DOCTYPE r [<!ENTITY % amp "&#38;"><!ENTITY e "%amp;broken">]><r/>`
		_, errBad := helium.NewParser().Parse(t.Context(), []byte(wouldBeBad))
		require.Error(t, errBad,
			"a PE reference in an internal-subset entity value is not well formed")
		require.Contains(t, errBad.Error(), "PEReferences forbidden in internal subset")
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
			dtdSystemID: {Data: []byte(
				`<!ENTITY c "control">` + "\n" +
					`<!ENTITY % amp "&#38;">` + "\n" +
					`<!ENTITY e "%amp;broken">`)},
		}
		const input = `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

		_, err := helium.NewParser().BlockXXE(false).
			LoadExternalDTD(true).
			FS(fsys).
			Parse(t.Context(), []byte(input))
		require.Error(t, err, "malformed entity reference in an external subset declaration must surface as a parse error")
		require.Contains(t, err.Error(), "malformed entity reference in entity value")
	})
}

// TestExternalSystemParameterEntityCaptured proves that an external SYSTEM
// parameter entity declared in the external subset is registered. parseExternalID
// returns (systemURI, publicID); the external-PE declaration path must guard on
// the systemURI and record it. Guarding on the publicID instead drops a SYSTEM PE
// entirely (a SYSTEM declaration has no public ID), so it would never be stored.
//
// A control general entity declared on the line before proves the external subset
// is loaded; only the SYSTEM external PE was being dropped.
func TestExternalSystemParameterEntityCaptured(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		dtdSystemID: {Data: []byte(
			`<!ENTITY ctrl "control">` + "\n" +
				`<!ENTITY % pe SYSTEM "pe.ent">`)},
	}
	const input = `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

	doc, err := helium.NewParser().BlockXXE(false).
		LoadExternalDTD(true).
		FS(fsys).
		Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	require.NotNil(t, doc)

	_, ctrlOK := doc.GetEntity("ctrl")
	require.True(t, ctrlOK, "control general entity must be stored, proving the external subset loaded")

	ent, ok := doc.GetParameterEntity("pe")
	require.True(t, ok, "external SYSTEM parameter entity must be registered")
	require.Equal(t, enum.ExternalParameterEntity, ent.EntityType())
	require.Equal(t, peSystemID, ent.SystemID(), "the SYSTEM literal must be recorded as the system ID")
	require.Empty(t, ent.ExternalID(), "a SYSTEM declaration has no public ID")
}

// TestParameterEntityDeclFirstExternalSubset covers the case where a
// parameter-entity DECLARATION (<!ENTITY % ...>) is the FIRST declaration of an
// external subset. The '%' marker following <!ENTITY must not be mis-parsed as a
// parameter-entity REFERENCE, which previously produced a spurious
// "space required at line 1, column 2" (the psDTD-only marker guard did not
// apply because parseMarkupDecl sets psDTD only AFTER the first declaration).
func TestParameterEntityDeclFirstExternalSubset(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE doc SYSTEM "d.dtd"><doc/>`

	testcases := []struct {
		name string
		dtd  string
	}{
		{
			// Internal PE declared first, then referenced to supply the element decl.
			name: "internal-pe-decl-first",
			dtd:  `<!ENTITY % pe "<!ELEMENT doc EMPTY>">` + "\n" + `%pe;`,
		},
		{
			// External PE declared first (never referenced): the declaration alone
			// must not trip the marker guard.
			name: "external-pe-decl-first",
			dtd:  `<!ENTITY % bad SYSTEM "bad.ent">` + "\n" + `<!ELEMENT doc EMPTY>`,
		},
		{
			// PE declaration is the first declaration INSIDE an INCLUDE section.
			name: "pe-decl-first-in-include",
			dtd: `<![INCLUDE[` + "\n" + `<!ENTITY % rootel "<!ELEMENT doc EMPTY>">` + "\n" +
				`]]>` + "\n" + `%rootel;`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fsys := fstest.MapFS{dtdSystemID: {Data: []byte(tc.dtd)}}
			doc, err := helium.NewParser().BlockXXE(false).
				LoadExternalDTD(true).SubstituteEntities(true).FS(fsys).
				Parse(t.Context(), []byte(input))
			require.NoError(t, err, "a parameter-entity declaration as the first external-subset declaration must parse")
			require.NotNil(t, doc)
		})
	}
}

// TestConditionalSectionKeywordFromParameterEntity covers an external-subset
// conditional section whose INCLUDE keyword is supplied by a parameter entity
// (`<![ %e; ... ]]>` with %e; -> "INCLUDE["). The blank skip after "<![" must
// leave the "%" for expansion (not consume it unexpanded), and the spent PE
// cursor must be popped before the body floor is captured, so the body
// declarations (here a defaulting <!ATTLIST>) are parsed and applied.
func TestConditionalSectionKeywordFromParameterEntity(t *testing.T) {
	t.Parallel()

	const input = `<!DOCTYPE doc SYSTEM "d.dtd"><doc></doc>`
	dtd := "<!ENTITY % e \"INCLUDE[\">\n" +
		"<!ELEMENT doc (#PCDATA)>\n" +
		"<![ %e; <!ATTLIST doc a1 CDATA \"v1\"> ]]>\n"
	fsys := fstest.MapFS{dtdSystemID: {Data: []byte(dtd)}}

	doc, err := helium.NewParser().BlockXXE(false).
		LoadExternalDTD(true).DefaultDTDAttributes(true).SubstituteEntities(true).
		FS(fsys).Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	require.NotNil(t, doc)

	str, werr := helium.WriteString(doc.DocumentElement())
	require.NoError(t, werr)
	require.Contains(t, str, wantAttrA1V1, "the <!ATTLIST> inside the PE-supplied INCLUDE section must supply the default attribute")
}

// wantAttrA1V1 is the serialized default attribute the PE-in-markup fixtures
// assemble; shared to keep the repeated literal in one place.
const wantAttrA1V1 = `a1="v1"`

// TestParameterEntityInMarkupDecl covers XML §4.4.8 "Included as PE": in the
// EXTERNAL subset a parameter-entity reference is recognized and included
// ANYWHERE a markup declaration occurs — INSIDE or ADJACENT to the declaration,
// not only between declarations — and its replacement text is padded with one
// leading and one trailing space. Each DTD here is a valid external subset that
// must parse AND apply the resulting declaration (W3C xmlconf valid/not-sa
// 019/020/021 and the japanese/spec.dtd content-model & common-attribute
// patterns).
func TestParameterEntityInMarkupDecl(t *testing.T) {
	t.Parallel()

	const input = `<!DOCTYPE doc SYSTEM "d.dtd"><doc></doc>`

	testcases := []struct {
		name string
		dtd  string
		want string // substring the serialized <doc> must contain
	}{
		{
			// PE adjacent to an attribute type: `CDATA%e;` with %e; -> "'v1'". The
			// §4.4.8 leading space separates the type from the default value.
			name: "pe-adjacent-to-attribute-type",
			dtd:  "<!ELEMENT doc (#PCDATA)>\n<!ENTITY % e \"'v1'\">\n<!ATTLIST doc a1 CDATA%e;>\n",
			want: wantAttrA1V1,
		},
		{
			// PE supplies the element name immediately after <!ATTLIST (no space):
			// `<!ATTLIST%e;a1` with %e; -> "doc".
			name: "pe-supplies-element-name",
			dtd:  "<!ENTITY % e \"doc\">\n<!ELEMENT doc (#PCDATA)>\n<!ATTLIST%e;a1 CDATA \"v1\">\n",
			want: wantAttrA1V1,
		},
		{
			// PE supplies most of the ATTLIST body: %e; -> "doc a1 CDATA".
			name: "pe-supplies-attlist-body-head",
			dtd:  "<!ENTITY % e \"doc a1 CDATA\">\n<!ELEMENT doc (#PCDATA)>\n<!ATTLIST %e; \"v1\">\n",
			want: wantAttrA1V1,
		},
		{
			// PE supplies the whole attribute definition list.
			name: "pe-supplies-full-attlist-body",
			dtd:  "<!ENTITY % att \"a1 CDATA 'v1'\">\n<!ELEMENT doc (#PCDATA)>\n<!ATTLIST doc %att;>\n",
			want: wantAttrA1V1,
		},
		{
			// PE inside an element content model, recursively nested through an
			// empty PE (the japanese/spec.dtd `%class;` idiom).
			name: "pe-in-content-model",
			dtd: "<!ENTITY % local \"\">\n<!ENTITY % kids \"a %local;\">\n" +
				"<!ELEMENT a (#PCDATA)>\n<!ELEMENT head (#PCDATA)>\n" +
				"<!ELEMENT doc (head?, (%kids;)*)>\n<!ATTLIST doc a1 CDATA 'v1'>\n",
			want: wantAttrA1V1,
		},
		{
			// PE supplies an ATTLIST enumeration name list: `(%vals;)`.
			name: "pe-in-attribute-enumeration",
			dtd:  "<!ENTITY % vals \"red|green|blue\">\n<!ELEMENT doc (#PCDATA)>\n<!ATTLIST doc a1 (%vals;) \"red\">\n",
			want: `a1="red"`,
		},
		{
			// PE supplies a #FIXED default value.
			name: "pe-in-fixed-default",
			dtd:  "<!ENTITY % v \"'red'\">\n<!ELEMENT doc (#PCDATA)>\n<!ATTLIST doc a1 CDATA #FIXED %v;>\n",
			want: `a1="red"`,
		},
		{
			// PE supplies a NOTATION type name list, plus the notation declarations.
			name: "pe-in-notation-type-list",
			dtd: "<!ENTITY % ns \"gif|jpg\">\n<!ELEMENT doc (#PCDATA)>\n" +
				"<!NOTATION gif SYSTEM \"gif\">\n<!NOTATION jpg SYSTEM \"jpg\">\n" +
				"<!ATTLIST doc t NOTATION (%ns;) #IMPLIED a1 CDATA 'v1'>\n",
			want: wantAttrA1V1,
		},
		{
			// PE supplies a NOTATION declaration's SYSTEM literal: `SYSTEM %sid;`.
			name: "pe-in-notation-decl-system-id",
			dtd: "<!ENTITY % sid \"'g.dtd'\">\n<!ELEMENT doc (#PCDATA)>\n" +
				"<!NOTATION gif SYSTEM %sid;>\n<!ATTLIST doc a1 CDATA 'v1'>\n",
			want: wantAttrA1V1,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fsys := fstest.MapFS{dtdSystemID: {Data: []byte(tc.dtd)}}
			doc, err := helium.NewParser().BlockXXE(false).
				LoadExternalDTD(true).DefaultDTDAttributes(true).SubstituteEntities(true).
				ValidateDTD(true).FS(fsys).Parse(t.Context(), []byte(input))
			require.NoError(t, err, "a valid external subset with a PE in/adjacent to a markup declaration must parse")
			require.NotNil(t, doc)

			str, werr := helium.WriteString(doc.DocumentElement())
			require.NoError(t, werr)
			require.Contains(t, str, tc.want, "the declaration assembled across the PE boundary must be applied")
		})
	}
}

// TestParameterEntitySuppliesEntityValue covers a parameter entity supplying an
// internal general entity's value in the external subset:
// `<!ENTITY greet %pub;>` with %pub; -> "'hello'" declares greet as an INTERNAL
// entity whose value is `hello` (not an empty external entity), so a later
// `&greet;` reference expands to `hello`.
func TestParameterEntitySuppliesEntityValue(t *testing.T) {
	t.Parallel()

	dtd := "<!ENTITY % pub \"'hello'\">\n<!ELEMENT doc (#PCDATA)>\n<!ENTITY greet %pub;>\n"
	const input = `<!DOCTYPE doc SYSTEM "d.dtd"><doc>&greet;</doc>`
	fsys := fstest.MapFS{dtdSystemID: {Data: []byte(dtd)}}

	doc, err := helium.NewParser().BlockXXE(false).
		LoadExternalDTD(true).SubstituteEntities(true).
		FS(fsys).Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	require.NotNil(t, doc)

	str, werr := helium.WriteString(doc.DocumentElement())
	require.NoError(t, werr)
	require.Equal(t, "<doc>hello</doc>", str, "the PE-supplied internal entity value must expand, not become an empty external entity")
}

// TestInternalSubsetPEInMarkupRejected asserts the INTERNAL subset stays
// byte-identical to origin: a parameter entity must NOT supply part of (or be
// adjacent to) a markup declaration there (WFC: PEs in Internal Subset). PE
// expansion inside markup is EXTERNAL-subset-only; a '%' where an "S", token, or
// '>' is required is rejected exactly as before, never silently accepted.
func TestInternalSubsetPEInMarkupRejected(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name    string
		doctype string
	}{
		{
			// A '%' where an "S" or '>' is required in an <!ENTITY> declaration.
			name:    "stray-percent-in-entity-decl",
			doctype: `<!DOCTYPE doc [<!ELEMENT doc EMPTY><!ENTITY e SYSTEM "x"%p;>]><doc/>`,
		},
		{
			// A PE supplying the <!ATTLIST> body in the internal subset.
			name:    "pe-supplies-attlist-body",
			doctype: `<!DOCTYPE doc [<!ELEMENT doc EMPTY><!ENTITY % att "a1 CDATA 'v1'"><!ATTLIST doc %att;>]><doc/>`,
		},
		{
			// A PE supplying an <!ELEMENT> content model in the internal subset.
			name:    "pe-supplies-content-model",
			doctype: `<!DOCTYPE doc [<!ENTITY % m "#PCDATA"><!ELEMENT doc (%m;)>]><doc/>`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := helium.NewParser().LoadExternalDTD(true).SubstituteEntities(true).
				Parse(t.Context(), []byte(tc.doctype))
			require.Error(t, err, "a parameter entity inside a markup declaration in the internal subset must be rejected")
		})
	}
}

// TestParameterEntityMarkupBoundaryViolation covers the XML validity constraints
// that a markup declaration (and a parenthesized content-model group) must start
// and stop in the SAME entity: a closing '>' or ')' supplied by a DIFFERENT
// parameter-entity replacement text than the one that opened the declaration/group
// is a boundary violation and must be rejected (W3C xmlconf invalid cases E14,
// invalid/002, ibm P49/P50/P51 invalid). Rejecting these must NOT regress the
// valid PE-in-markup documents above.
func TestParameterEntityMarkupBoundaryViolation(t *testing.T) {
	t.Parallel()

	const input = `<!DOCTYPE doc SYSTEM "d.dtd"><doc></doc>`

	testcases := []struct {
		name string
		dtd  string
	}{
		{
			// The ATTLIST closing '>' comes from inside the PE (W3C errata E14).
			name: "attlist-close-in-pe",
			dtd:  "<!ELEMENT doc ANY>\n<!ENTITY % e \"a1 CDATA #IMPLIED>\">\n<!ATTLIST doc %e;\n",
		},
		{
			// Content-model '(' in a PE, ')' in the containing DTD (W3C invalid/002).
			name: "content-model-open-in-pe-close-in-dtd",
			dtd:  "<!ENTITY % e \"(#PCDATA\">\n<!ELEMENT doc %e;)>\n",
		},
		{
			// Content-model '(' in one PE, ')' in another (W3C ibm P49 invalid).
			name: "content-model-group-split-across-pes",
			dtd: "<!ELEMENT a EMPTY>\n<!ELEMENT b (#PCDATA)>\n" +
				"<!ENTITY % choice1 \"(a|b\">\n<!ENTITY % choice2 \"|c)\">\n" +
				"<!ELEMENT c ANY>\n<!ELEMENT child1 %choice1;%choice2; >\n",
		},
		{
			// The <!ENTITY> closing '>' comes from a PE (%close; -> ">").
			name: "entity-close-in-pe",
			dtd:  "<!ELEMENT doc (#PCDATA)>\n<!ENTITY % close \">\">\n<!ENTITY greet 'hi'%close;\n",
		},
		{
			// The <!NOTATION> closing '>' comes from a PE (%close; -> ">").
			name: "notation-close-in-pe",
			dtd:  "<!ELEMENT doc (#PCDATA)>\n<!ENTITY % close \">\">\n<!NOTATION gif SYSTEM 'gif'%close;\n",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fsys := fstest.MapFS{dtdSystemID: {Data: []byte(tc.dtd)}}
			_, err := helium.NewParser().BlockXXE(false).
				LoadExternalDTD(true).DefaultDTDAttributes(true).SubstituteEntities(true).
				ValidateDTD(true).FS(fsys).Parse(t.Context(), []byte(input))
			require.Error(t, err, "a markup declaration / group that crosses a PE boundary must be rejected")
		})
	}
}

// TestExternalPublicParameterEntityCaptured proves the PUBLIC external parameter
// entity path records the public and system IDs in the correct fields rather than
// swapping them.
func TestExternalPublicParameterEntityCaptured(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		dtdSystemID: {Data: []byte(
			`<!ENTITY ctrl "control">` + "\n" +
				`<!ENTITY % pe PUBLIC "-//x//pe" "pe.ent">`)},
	}
	const input = `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

	doc, err := helium.NewParser().BlockXXE(false).
		LoadExternalDTD(true).
		FS(fsys).
		Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	require.NotNil(t, doc)

	ent, ok := doc.GetParameterEntity("pe")
	require.True(t, ok, "external PUBLIC parameter entity must be registered")
	require.Equal(t, enum.ExternalParameterEntity, ent.EntityType())
	require.Equal(t, peSystemID, ent.SystemID(), "the system literal must be the system ID")
	require.Equal(t, "-//x//pe", ent.ExternalID(), "the public ID must be the external ID")
}

// TestExternalParameterEntityContentLoaded proves that referencing an external
// SYSTEM parameter entity in the external subset actually loads its content and
// applies the declarations it contains. The external PE pe.ent declares a general
// entity; with external DTD loading enabled that entity must be registered,
// proving the external PE content was pulled in and parsed (not silently dropped).
func TestExternalParameterEntityContentLoaded(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		dtdSystemID: {Data: []byte(
			`<!ENTITY ctrl "control">` + "\n" +
				`<!ENTITY % pe SYSTEM "pe.ent">` + "\n" +
				`%pe;`)},
		peSystemID: {Data: []byte(`<!ENTITY fromPE "loaded-from-external-pe">`)},
	}
	const input = `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

	doc, err := helium.NewParser().BlockXXE(false).
		LoadExternalDTD(true).
		FS(fsys).
		Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	require.NotNil(t, doc)

	_, ctrlOK := doc.GetEntity("ctrl")
	require.True(t, ctrlOK, "control general entity must be stored, proving the external subset loaded")

	ent, ok := doc.GetEntity("fromPE")
	require.True(t, ok, "the general entity declared inside the external PE must be registered")
	require.Equal(t, "loaded-from-external-pe", string(ent.Content()))
}

// TestExternalParameterEntityNotLoadedSecureDefault proves the secure default
// (XXE blocked) loads no external parameter entity content: with the default
// parser the external subset is not loaded at all, so the general entity declared
// inside the external PE is absent. Behavior is unchanged from before the fix.
func TestExternalParameterEntityNotLoadedSecureDefault(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		dtdSystemID: {Data: []byte(
			`<!ENTITY ctrl "control">` + "\n" +
				`<!ENTITY % pe SYSTEM "pe.ent">` + "\n" +
				`%pe;`)},
		peSystemID: {Data: []byte(`<!ENTITY fromPE "loaded-from-external-pe">`)},
	}
	const input = `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

	doc, err := helium.NewParser().FS(fsys).Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	require.NotNil(t, doc)

	_, ok := doc.GetEntity("fromPE")
	require.False(t, ok, "secure default must not load external parameter entity content")
}

// TestExternalParameterEntityNestedRelativeBaseURI proves that a relative system
// ID in a declaration INSIDE an external parameter entity resolves against the
// PE's OWN location, not the containing DTD. d.dtd declares a PE at "sub/pe.ent"
// and references it; sub/pe.ent declares a general entity "e" with a relative
// SYSTEM id "leaf.ent". With baseURI scoped to the PE while its replacement text
// is parsed, e must resolve to "sub/leaf.ent" (sibling of pe.ent), NOT "leaf.ent"
// (sibling of d.dtd).
//
// The leading control general entity sidesteps a SEPARATE pre-existing parser
// bug (present on main, orthogonal to base-URI scoping): a parameter-entity
// declaration as the VERY FIRST declaration of an external subset fails with
// "space required". The control entity keeps this regression focused on scoping.
func TestExternalParameterEntityNestedRelativeBaseURI(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		dtdSystemID: {Data: []byte(
			`<!ENTITY ctrl "control">` + "\n" +
				`<!ENTITY % pe SYSTEM "sub/pe.ent">` + "\n" +
				`%pe;`)},
		"sub/pe.ent": {Data: []byte(`<!ENTITY e SYSTEM "leaf.ent">`)},
	}
	const input = `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

	doc, err := helium.NewParser().BlockXXE(false).
		LoadExternalDTD(true).
		FS(fsys).
		Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	require.NotNil(t, doc)

	ent, ok := doc.GetEntity("e")
	require.True(t, ok, "general entity declared inside the external PE must be registered")
	require.Equal(t, "sub/leaf.ent", ent.URI(),
		"relative system ID inside the external PE must resolve against the PE's own location")
}

// TestExternalParameterEntityInEntityValueLoaded proves that an external
// parameter entity referenced inside a general entity's VALUE is loaded and
// expanded regardless of whether the PE was ever referenced at the top level of
// the DTD first. The external subset declares "%p;" (SYSTEM "value.ent") and a
// general entity g whose value is "%p;" — WITHOUT any top-level "%p;" reference
// that would otherwise be what first caches p's content. g's expanded value must
// equal value.ent's content, proving the load happens through the centralized
// PE-replacement path rather than depending on reference order.
//
// The leading control general entity sidesteps the same separate pre-existing
// first-declaration bug noted on TestExternalParameterEntityNestedRelativeBaseURI.
func TestExternalParameterEntityInEntityValueLoaded(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		dtdSystemID: {Data: []byte(
			`<!ENTITY ctrl "control">` + "\n" +
				`<!ENTITY % p SYSTEM "value.ent">` + "\n" +
				`<!ENTITY g "%p;">`)},
		"value.ent": {Data: []byte(`expanded-from-external-pe`)},
	}
	const input = `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

	doc, err := helium.NewParser().BlockXXE(false).
		LoadExternalDTD(true).
		FS(fsys).
		Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	require.NotNil(t, doc)

	ent, ok := doc.GetEntity("g")
	require.True(t, ok, "general entity g must be registered")
	require.Equal(t, "expanded-from-external-pe", string(ent.Content()),
		"external PE referenced in an entity value must be loaded and expanded regardless of reference order")
}

// TestExternalParameterEntityInEntityValueSecureDefault proves the secure
// default does NOT load an external PE referenced inside an entity value: with
// the default parser the external subset is not loaded at all, so g is absent.
func TestExternalParameterEntityInEntityValueSecureDefault(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		dtdSystemID: {Data: []byte(
			`<!ENTITY ctrl "control">` + "\n" +
				`<!ENTITY % p SYSTEM "value.ent">` + "\n" +
				`<!ENTITY g "%p;">`)},
		"value.ent": {Data: []byte(`expanded-from-external-pe`)},
	}
	const input = `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

	doc, err := helium.NewParser().FS(fsys).Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	require.NotNil(t, doc)

	_, ok := doc.GetEntity("g")
	require.False(t, ok, "secure default must not load the external subset or its PE-referencing entity value")
}

// TestExternalParameterEntitySelfRecursionRejected proves that a self-recursive
// external parameter entity is rejected with a recursion error rather than
// pushing cursors until the entity-amplification ceiling trips (or OOM): pe.ent's
// replacement text references the very PE that loaded it, so the active-PE guard
// must fail the parse the moment the nested "%pe;" is seen.
func TestExternalParameterEntitySelfRecursionRejected(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		dtdSystemID: {Data: []byte(
			`<!ENTITY ctrl "control">` + "\n" +
				`<!ENTITY % pe SYSTEM "pe.ent">` + "\n" +
				`%pe;`)},
		peSystemID: {Data: []byte(`%pe;`)},
	}
	const input = `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

	_, err := helium.NewParser().BlockXXE(false).
		LoadExternalDTD(true).
		FS(fsys).
		Parse(t.Context(), []byte(input))
	require.Error(t, err, "self-recursive external parameter entity must be rejected")
	require.Contains(t, err.Error(), "references itself",
		"rejection must be the recursion guard, not an amplification/ceiling trip")
}

// peCatalog resolves the external PE's system ID to a DIFFERENT URI than the
// entity's declared one, modeling the catalog/custom-resolver case where
// input.URI() (the URI actually opened) differs from the entity's URI().
type peCatalog struct{ from, to string }

func (c peCatalog) Resolve(_ context.Context, _, sysID string) string {
	if sysID == c.from {
		return c.to
	}
	return ""
}
func (c peCatalog) ResolveURI(_ context.Context, _ string) string { return "" }

// TestExternalParameterEntityValueFirstResolvedURICached proves that when an
// external PE is loaded FIRST through a general entity's value (caching its
// content), a later top-level "%pe;" parses the cached bytes against the URI the
// bytes were ACTUALLY loaded from — the catalog-resolved "sub/pe.ent" — not the
// entity's declared "pe.ent". A relative SYSTEM id inside the PE ("leaf.ent")
// must therefore resolve to "sub/leaf.ent". Before the resolved-URI cache fix the
// cached path returned e.URI() ("pe.ent"), wrongly resolving leaf to "leaf.ent".
func TestExternalParameterEntityValueFirstResolvedURICached(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		dtdSystemID: {Data: []byte(
			`<!ENTITY ctrl "control">` + "\n" +
				`<!ENTITY % pe SYSTEM "pe.ent">` + "\n" +
				`<!ENTITY g "%pe;">` + "\n" +
				`%pe;`)},
		"sub/pe.ent": {Data: []byte(`<!ENTITY leaf SYSTEM "leaf.ent">`)},
	}
	const input = `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

	doc, err := helium.NewParser().BlockXXE(false).
		LoadExternalDTD(true).
		Catalog(peCatalog{from: peSystemID, to: "sub/pe.ent"}).
		FS(fsys).
		Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	require.NotNil(t, doc)

	ent, ok := doc.GetEntity("leaf")
	require.True(t, ok, "general entity declared inside the cached external PE must be registered")
	require.Equal(t, "sub/leaf.ent", ent.URI(),
		"cached external PE must resolve relative IDs against the URI it was actually loaded from")
}

// TestExternalParameterEntityWithTextDecl proves that an external parameter
// entity whose replacement text begins with a TextDecl
// ("<?xml ... encoding=...?>") is parsed: the TextDecl is consumed before the
// declaration loop instead of being rejected as a processing instruction.
func TestExternalParameterEntityWithTextDecl(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		dtdSystemID: {Data: []byte(
			`<!ENTITY ctrl "control">` + "\n" +
				`<!ENTITY % pe SYSTEM "pe.ent">` + "\n" +
				`%pe;`)},
		peSystemID: {Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
			`<!ENTITY td "from-textdecl-pe">`)},
	}
	const input = `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

	doc, err := helium.NewParser().BlockXXE(false).
		LoadExternalDTD(true).
		FS(fsys).
		Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	require.NotNil(t, doc)

	ent, ok := doc.GetEntity("td")
	require.True(t, ok, "entity declared after a TextDecl in an external PE must be registered")
	require.Equal(t, "from-textdecl-pe", string(ent.Content()),
		"external PE beginning with a TextDecl must parse its declarations")
}

// TestExternalParameterEntityInEntityValueStripsTextDecl proves that when an
// external parameter entity whose replacement text begins with a TextDecl is
// referenced inside a GENERAL entity's value ("<!ENTITY g "%p;">"), the stored
// value of g is the PE's POST-TextDecl bytes only — the leading
// "<?xml ... encoding=...?>" must NOT be embedded into g. The decode is
// centralized at the shared load/cache chokepoint, so the entity-value
// expansion path and the top-level "%pe;" path both see the stripped bytes.
func TestExternalParameterEntityInEntityValueStripsTextDecl(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		dtdSystemID: {Data: []byte(
			`<!ENTITY ctrl "control">` + "\n" +
				`<!ENTITY % p SYSTEM "value.ent">` + "\n" +
				`<!ENTITY g "%p;">`)},
		"value.ent": {Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>` +
			`post-textdecl-value`)},
	}
	const input = `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

	doc, err := helium.NewParser().BlockXXE(false).
		LoadExternalDTD(true).
		FS(fsys).
		Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	require.NotNil(t, doc)

	ent, ok := doc.GetEntity("g")
	require.True(t, ok, "general entity g must be registered")
	require.Equal(t, "post-textdecl-value", string(ent.Content()),
		"external PE referenced in an entity value must contribute only its post-TextDecl bytes, not the TextDecl itself")
}

// TestExternalParameterEntityVersionOnlyTextDeclRejected proves that an external
// parameter entity whose replacement text begins with a version-only
// declaration ("<?xml version="1.0"?>") is rejected: a TextDecl REQUIRES an
// EncodingDecl, so a version-only declaration is not a valid TextDecl and must
// not be leniently accepted.
func TestExternalParameterEntityVersionOnlyTextDeclRejected(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		dtdSystemID: {Data: []byte(
			`<!ENTITY ctrl "control">` + "\n" +
				`<!ENTITY % pe SYSTEM "pe.ent">` + "\n" +
				`%pe;`)},
		peSystemID: {Data: []byte(`<?xml version="1.0"?>` + "\n" +
			`<!ENTITY td "x">`)},
	}
	const input = `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

	_, err := helium.NewParser().BlockXXE(false).
		LoadExternalDTD(true).
		FS(fsys).
		Parse(t.Context(), []byte(input))
	require.Error(t, err,
		"a version-only declaration is not a valid TextDecl (encoding is required) and must be rejected")
}

// TestExternalParameterEntityStandaloneTextDeclRejected proves that an external
// parameter entity whose replacement text begins with a declaration carrying a
// 'standalone' pseudo-attribute is rejected: a TextDecl does not permit a
// StandaloneDecl, so such a declaration is malformed and must not be leniently
// accepted.
func TestExternalParameterEntityStandaloneTextDeclRejected(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		dtdSystemID: {Data: []byte(
			`<!ENTITY ctrl "control">` + "\n" +
				`<!ENTITY % pe SYSTEM "pe.ent">` + "\n" +
				`%pe;`)},
		peSystemID: {Data: []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" +
			`<!ENTITY td "x">`)},
	}
	const input = `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

	_, err := helium.NewParser().BlockXXE(false).
		LoadExternalDTD(true).
		FS(fsys).
		Parse(t.Context(), []byte(input))
	require.Error(t, err,
		"a TextDecl carrying a standalone pseudo-attribute is malformed and must be rejected")
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
	fsys := fstest.MapFS{dtdSystemID: {Data: []byte(dtd)}}
	input := `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

	doc, err := helium.NewParser().BlockXXE(false).
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
}

// TestParseFileLargeEntityNotFalselyAmplified guards against a regression where
// ParseFile delegated to the streaming reader path, leaving inputSize == 0. A
// large internal entity referenced exactly once is legitimate (no
// amplification) and must parse, but a zero inputSize made the
// amplification-ratio guard fall back to a divisor of 1 and falsely reject it.
// ParseFile knows the file size, so it must seed inputSize like Parse([]byte)
// does and produce the same result.
func TestParseFileLargeEntityNotFalselyAmplified(t *testing.T) {
	t.Parallel()

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

	p := helium.NewParser().SubstituteEntities(true).MaxEntityAmplification(-1) // disable amplification guard to test depth only
	_, err := p.Parse(t.Context(), []byte(dtd.String()))
	require.Error(t, err, "depth > 40 should still error")
	require.Contains(t, err.Error(), "entity loop")
}

// TestParameterEntities exercises parameter-entity declaration and reference in
// the internal subset. A PE reference between declarations (supplying a whole
// markup declaration) is well formed; a PE reference WITHIN a declaration — here
// inside another entity's value — violates the "PEs in Internal Subset" WFC
// (XML §2.8) and is fatal, matching libxml2.
func TestParameterEntities(t *testing.T) {
	t.Parallel()

	// A parameter entity referenced BETWEEN declarations, supplying a complete
	// markup declaration. This is where PE references may occur in the internal
	// subset, so it must parse.
	const good = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ENTITY % decls "<!ELEMENT doc (#PCDATA)>">
%decls;
<!ENTITY greeting "Hello World">
]>
<doc>&greeting;</doc>`

	doc, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(good))
	require.NoError(t, err, "a PE reference between declarations is well formed")
	require.NotNil(t, doc.DocumentElement())

	// A parameter entity referenced WITHIN a markup declaration (inside an
	// entity value) in the internal subset violates the PEs in Internal Subset
	// WFC and is a fatal error.
	const bad = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ENTITY % name "World">
<!ENTITY greeting "Hello %name;">
<!ELEMENT doc (#PCDATA)>
]>
<doc>&greeting;</doc>`

	_, err = helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(bad))
	require.Error(t, err, "a PE reference within a declaration in the internal subset is not well formed")
	require.Contains(t, err.Error(), "PEReferences forbidden in internal subset")
}

// TestEntitySubstitution exercises entity expansion in content and attributes.
func TestEntitySubstitution(t *testing.T) {
	t.Parallel()

	const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ENTITY greeting "hello world">
]>
<doc attr="&greeting;">&greeting;</doc>`

	doc, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(src))
	require.NoError(t, err)
	require.Contains(t, string(doc.DocumentElement().Content()), "hello world")
}

// TestExternalDTDConditionalSections exercises INCLUDE/IGNORE conditional
// sections, which only appear in the external subset.
func TestExternalDTDConditionalSections(t *testing.T) {
	t.Parallel()

	const dtd = `<!ELEMENT root (#PCDATA)>
<![INCLUDE[
<!ENTITY included "in">
]]>
<![IGNORE[
<!ENTITY ignored "out">
]]>`
	path := writeDTD(t, dtd)

	xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "` + path + `">
<root/>`

	doc, err := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).FS(helium.PermissiveFS()).Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	_, found := doc.GetEntity("included")
	require.True(t, found, "entity inside INCLUDE section must be declared")

	_, found = doc.GetEntity("ignored")
	require.False(t, found, "entity inside IGNORE section must be skipped")
}

// TestExternalDTDNotationsAndEntities exercises notation and external entity
// declarations resolved from the external subset.
func TestExternalDTDNotationsAndEntities(t *testing.T) {
	t.Parallel()

	const dtd = `<!ELEMENT root (#PCDATA)>
<!NOTATION gif SYSTEM "viewer.exe">
<!NOTATION png PUBLIC "-//N//EN" "png.exe">
<!ENTITY img SYSTEM "img.gif" NDATA gif>
<!ENTITY ext SYSTEM "data.xml">
<!ENTITY pub PUBLIC "-//P//EN" "pub.xml">`
	path := writeDTD(t, dtd)

	xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "` + path + `">
<root/>`

	doc, err := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).FS(helium.PermissiveFS()).Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	require.NotNil(t, doc.ExtSubset(), "external subset must be present")
	// The external general entities are resolvable from the document.
	_, ok := doc.GetEntity("ext")
	require.True(t, ok, "external SYSTEM entity declared in ext subset")
	_, ok = doc.GetEntity("pub")
	require.True(t, ok, "external PUBLIC entity declared in ext subset")
}

// TestExternalDTDPublicIdentifier exercises a DOCTYPE that declares a PUBLIC
// external identifier (with both public and system IDs).
func TestExternalDTDPublicIdentifier(t *testing.T) {
	t.Parallel()

	const dtd = `<!ELEMENT root (#PCDATA)>
<!ENTITY who "world">`
	path := writeDTD(t, dtd)

	xml := `<?xml version="1.0"?>
<!DOCTYPE root PUBLIC "-//Example//DTD root//EN" "` + path + `">
<root/>`

	doc, err := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).FS(helium.PermissiveFS()).Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	_, found := doc.GetEntity("who")
	require.True(t, found)
}

// TestExternalDTDMissingFile exercises the not-found branch of external DTD
// resolution: a SYSTEM id pointing at a non-existent file must not crash.
func TestExternalDTDMissingFile(t *testing.T) {
	t.Parallel()

	xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "/nonexistent/path/to.dtd">
<root>content</root>`

	// The document body is still well-formed; the external DTD simply cannot be
	// loaded. Parsing should not panic.
	doc, _ := helium.NewParser().LoadExternalDTD(true).Parse(t.Context(), []byte(xml))
	if doc != nil {
		require.NotNil(t, doc.DocumentElement())
	}
}

// TestInternalSubsetParameterEntityInclusion exercises a parameter entity used
// inside the internal subset to pull in further declarations.
func TestInternalSubsetParameterEntityInclusion(t *testing.T) {
	t.Parallel()

	const xml = `<?xml version="1.0"?>
<!DOCTYPE root [
<!ENTITY % decls "<!ELEMENT root (#PCDATA)><!ENTITY inner 'inner-value'>">
%decls;
]>
<root/>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	_, found := doc.GetEntity("inner")
	require.True(t, found, "entity declared via internal-subset PE inclusion must be present")
}

// writeDTD writes a DTD file into a fresh temp dir and returns its path.
func writeDTD(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "ext.dtd")
	require.NoError(t, os.WriteFile(p, []byte(body), 0600))
	return p
}

// TestParseDTDEntityValuesAndParamRefs parses internal-subset DTDs that exercise
// the entity-value and parameter-entity-reference parser paths: entity values
// containing character and general-entity references, a parameter entity declared
// and then referenced inside the subset, and a mixed-content (#PCDATA|x|y)*
// declaration with several alternatives.
func TestParseDTDEntityValuesAndParamRefs(t *testing.T) {
	t.Parallel()

	t.Run("entity value with char and general refs", func(t *testing.T) {
		t.Parallel()
		const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ELEMENT doc (#PCDATA)>
<!ENTITY base "base">
<!ENTITY composed "prefix-&base;-&#65;-suffix">
]>
<doc>&composed;</doc>`
		doc, err := helium.NewParser().SubstituteEntities(true).Parse(context.Background(), []byte(src))
		require.NoError(t, err)
		require.NotNil(t, doc.DocumentElement())
	})

	t.Run("parameter entity expands to a markup declaration", func(t *testing.T) {
		t.Parallel()
		// A parameter entity whose replacement text is an entire markup
		// declaration, referenced via %e; inside the internal subset, drives
		// the PE-reference expansion path in the subset parser.
		const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ENTITY % e "<!ELEMENT doc (#PCDATA)>">
%e;
]>
<doc>text</doc>`
		doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
		require.NoError(t, err)
		dtd := doc.IntSubset()
		require.NotNil(t, dtd)
		_, ok := dtd.LookupElement("doc", "")
		require.True(t, ok, "the PE-supplied element declaration was registered")
	})

	t.Run("mixed content with several alternatives", func(t *testing.T) {
		t.Parallel()
		const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ELEMENT doc (#PCDATA | a | b | c)*>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b (#PCDATA)>
<!ELEMENT c (#PCDATA)>
]>
<doc>t <a/> u <b/> v <c/> w</doc>`
		doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
		require.NoError(t, err)
		dtd := doc.IntSubset()
		require.NotNil(t, dtd)
		edecl, ok := dtd.LookupElement("doc", "")
		require.True(t, ok)
		require.Equal(t, enum.MixedElementType, edecl.DeclType())
	})

	t.Run("element children content with nested groups and occurrences", func(t *testing.T) {
		t.Parallel()
		const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ELEMENT doc (head, (para | list)*, foot?)>
<!ELEMENT head (#PCDATA)>
<!ELEMENT para (#PCDATA)>
<!ELEMENT list (#PCDATA)>
<!ELEMENT foot (#PCDATA)>
]>
<doc><head/><para/><list/><para/><foot/></doc>`
		doc, err := helium.NewParser().ValidateDTD(true).Parse(context.Background(), []byte(src))
		require.NoError(t, err)
		require.NotNil(t, doc.DocumentElement())
	})
}

// TestCreateReferenceWithDeclaredEntity exercises CreateReference both for a
// predefined entity (resolvable) and an undeclared name (no entity attached).
func TestCreateReferenceWithDeclaredEntity(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("doc", "", "")
	require.NoError(t, err)

	_, err = dtd.AddEntity("greet", enum.InternalGeneralEntity, "", "", "Hello")
	require.NoError(t, err)

	// Reference to a declared general entity: the entity content is attached.
	ref, err := doc.CreateReference("greet")
	require.NoError(t, err)
	require.Equal(t, helium.EntityRefNode, ref.Type())
	require.Equal(t, []byte("Hello"), ref.Content())

	// Reference to an undeclared name: still produces an EntityRef, but with no
	// resolved content.
	ref2, err := doc.CreateReference("undeclared")
	require.NoError(t, err)
	require.Equal(t, "undeclared", ref2.Name())

	// "&name;" form is accepted and stripped.
	ref3, err := doc.CreateReference("&greet;")
	require.NoError(t, err)
	require.Equal(t, "greet", ref3.Name())

	// Empty name is rejected.
	_, err = doc.CreateReference("")
	require.Error(t, err)
}

// TestGetEntityExternalSubset exercises GetEntity's external-subset lookup and
// the standalone short-circuit, plus GetParameterEntity.
func TestGetEntityFromInternalSubset(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("doc", "", "")
	require.NoError(t, err)

	_, err = dtd.AddEntity("ge", enum.InternalGeneralEntity, "", "", "general")
	require.NoError(t, err)
	_, err = dtd.AddEntity("pe", enum.InternalParameterEntity, "", "", "param")
	require.NoError(t, err)

	ent, ok := doc.GetEntity("ge")
	require.True(t, ok)
	require.Equal(t, []byte("general"), ent.Content())

	_, ok = doc.GetEntity("missing")
	require.False(t, ok)

	pe, ok := doc.GetParameterEntity("pe")
	require.True(t, ok)
	require.Equal(t, []byte("param"), pe.Content())

	_, ok = doc.GetParameterEntity("missing")
	require.False(t, ok)

	// A document with no internal subset finds nothing.
	bare := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
	_, ok = bare.GetEntity("ge")
	require.False(t, ok)
	_, ok = bare.GetParameterEntity("pe")
	require.False(t, ok)
}

// TestEntityURIFallback covers Entity.URI's fallback to SystemID.
func TestEntityURIFallback(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("doc", "", "")
	require.NoError(t, err)

	ext, err := dtd.AddEntity("e", enum.ExternalGeneralParsedEntity, "pub", "sys.ent", "")
	require.NoError(t, err)
	// No resolved URI set => falls back to SystemID.
	require.Equal(t, "sys.ent", ext.URI())
	require.Equal(t, "sys.ent", ext.SystemID())
	require.Equal(t, "pub", ext.ExternalID())
}

// TestEntityRefToUnparsedEntity drives the "entity reference to unparsed entity"
// error branch of parseEntityRef.
func TestEntityRefToUnparsedEntity(t *testing.T) {
	t.Parallel()

	const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!NOTATION gif SYSTEM "viewer">
  <!ENTITY img SYSTEM "img.gif" NDATA gif>
]>
<doc>&img;</doc>`

	_, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.Error(t, err)
}

// trimSlashFS adapts an fs.FS so a leading-slash absolute name (such as the
// "/C:/..." path FileURIToPath yields on a POSIX host) is accepted by an
// fs.ValidPath-enforcing FS like fstest.MapFS.
type trimSlashFS struct{ inner fs.FS }

func (f trimSlashFS) Open(name string) (fs.File, error) {
	// Normalize to forward slashes (the native open name is backslash-separated
	// on Windows: "C:\\win\\dir\\ext.dtd") and drop the leading slash the POSIX
	// "/C:/..." form carries, so fstest.MapFS (keyed "C:/win/dir/ext.dtd")
	// serves it on every OS.
	return f.inner.Open(strings.TrimPrefix(filepath.ToSlash(name), "/")) //nolint:wrapcheck // test helper
}

// TestExternalSubsetResolvesAgainstWindowsDriveFileURIBase is the string-shaped
// (GOOS-independent) regression for the Windows nested-external-DTD failure: a
// document parsed with a Windows-drive "file:" base URI
// ("file:///C:/win/dir/doc.xml") that declares a RELATIVE external DTD
// ("ext.dtd"). The resolver must combine them into a proper "file:" URI (via
// BuildURI) and convert it to a local path before Open, NOT mangle it with
// filepath.Dir/Join — on Windows that cleared the directory and dropped the DTD.
// The base is a plain string, so this exercises the Windows branch on every OS.
// The resolved open name is whatever FileURIToPath yields for the combined
// "file:///C:/win/dir/ext.dtd": "/C:/win/dir/ext.dtd" on a POSIX host,
// "C:\\win\\dir\\ext.dtd" on Windows. Derive it the same way so the assertion
// is correct on both, and let trimSlashFS normalize either form to the MapFS key.
func TestExternalSubsetResolvesAgainstWindowsDriveFileURIBase(t *testing.T) {
	t.Parallel()

	const dtd = `<!ELEMENT chapter (#PCDATA)>
<!ENTITY greet "hello from nested dtd">`

	openName, err := iofs.FileURIToPath("file:///C:/win/dir/ext.dtd")
	require.NoError(t, err)
	fsys := &recordingFS{inner: trimSlashFS{fstest.MapFS{"C:/win/dir/ext.dtd": {Data: []byte(dtd)}}}}

	xml := `<?xml version="1.0"?>` +
		`<!DOCTYPE chapter SYSTEM "ext.dtd">` +
		`<chapter>text</chapter>`

	doc, err := helium.NewParser().BlockXXE(false).
		LoadExternalDTD(true).
		BaseURI("file:///C:/win/dir/doc.xml").
		FS(fsys).
		Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	// The relative SYSTEM id resolved into the base directory (never dropped to a
	// bare "ext.dtd", the Windows filepath.Join failure mode), so the DTD was
	// found and its general entity declared.
	require.True(t, fsys.wasOpened(openName),
		"relative SYSTEM id must resolve against the windows-drive file: base")
	_, found := doc.GetEntity("greet")
	require.True(t, found, "entity from external DTD must be declared, proving the file: DTD URI was resolved")
}

// TestIndirectEntityRefInAttributeValue exercises the XML 1.0 attribute-value
// well-formedness constraints ("No External Entity References", "No < in
// Attribute Values") against an INDIRECT reference — a general entity whose OWN
// replacement text is harmless but which transitively references an external or
// unparsed entity, or reaches a literal '<'. Under SubstituteEntities(false) the
// stricter attribute-value WFCs must fire on EVERY attribute occurrence,
// independent of whether the entity was already expanded (and its weaker
// element-content check recorded) in element content first.
func TestIndirectEntityRefInAttributeValue(t *testing.T) {
	t.Parallel()

	const head = "<!DOCTYPE r [\n" +
		"<!ELEMENT r ANY>\n" +
		"<!ELEMENT e EMPTY>\n" +
		"<!ATTLIST e a CDATA #IMPLIED>\n"

	testcases := []struct {
		name    string
		src     string
		wantMsg string
	}{
		{
			// The entity is expanded in element content FIRST (setting its
			// checked bit), then referenced from an attribute. The checked bit
			// must NOT suppress the attribute-value WFC re-validation.
			name: "external-indirect-after-content",
			src: head +
				"<!ENTITY ext SYSTEM \"nul\">\n" +
				"<!ENTITY outer \"&ext;\">\n" +
				"]>\n<r>&outer;<e a=\"&outer;\"/></r>",
			wantMsg: "attribute references external entity",
		},
		{
			name: "external-indirect-attr-only",
			src: head +
				"<!ENTITY ext SYSTEM \"nul\">\n" +
				"<!ENTITY outer \"&ext;\">\n" +
				"]>\n<r><e a=\"&outer;\"/></r>",
			wantMsg: "attribute references external entity",
		},
		{
			// An unparsed (NDATA) entity reached only through an attribute value.
			name: "unparsed-indirect-attr-only",
			src: head +
				"<!NOTATION gif PUBLIC \"gif\">\n" +
				"<!ENTITY pic SYSTEM \"nul\" NDATA gif>\n" +
				"<!ENTITY outer \"&pic;\">\n" +
				"]>\n<r><e a=\"&outer;\"/></r>",
			wantMsg: "entity reference to unparsed entity",
		},
		{
			// A nested entity whose replacement text contains a '<' (via a char
			// reference resolved into its stored content).
			name: "lessthan-indirect-attr-only",
			src: head +
				"<!ENTITY inner \"x&#60;y\">\n" +
				"<!ENTITY outer \"&inner;\">\n" +
				"]>\n<r><e a=\"&outer;\"/></r>",
			wantMsg: "'<' in entity is not allowed in attribute values",
		},
		{
			// The nested external entity is declared AFTER the ATTLIST default
			// value that transitively references it (forward reference). The WFC
			// classification must NOT be memoized against the incomplete entity
			// tables seen while the default value is parsed — a cached result
			// would let the document be accepted once the entity is declared.
			name: "external-indirect-attlist-default-forward",
			src: "<!DOCTYPE r [\n" +
				"<!ELEMENT r EMPTY>\n" +
				"<!ENTITY outer \"&ext;\">\n" +
				"<!ATTLIST r a CDATA \"&outer;\">\n" +
				"<!ENTITY ext SYSTEM \"nul\">\n" +
				"]>\n<r/>",
			wantMsg: "not defined",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).
				DefaultDTDAttributes(true).SubstituteEntities(false).ValidateDTD(false).
				Parse(t.Context(), []byte(tc.src))
			require.Error(t, err, "indirect entity reference must violate the attribute-value WFC")
			require.Nil(t, doc, "no document on a fatal well-formedness error")
			require.Contains(t, err.Error(), tc.wantMsg)
		})
	}
}

// TestIndirectHarmlessEntityRefInAttributeValue confirms the attribute-value WFC
// walk does not over-reject: an indirect general entity whose transitive
// replacement text is plain character data (no external/unparsed reference, no
// '<') is accepted, including when referenced multiple times.
func TestIndirectHarmlessEntityRefInAttributeValue(t *testing.T) {
	t.Parallel()

	src := "<!DOCTYPE r [\n" +
		"<!ELEMENT r ANY>\n" +
		"<!ELEMENT e EMPTY>\n" +
		"<!ATTLIST e a CDATA #IMPLIED>\n" +
		"<!ENTITY inner \"value\">\n" +
		"<!ENTITY outer \"a&inner;b\">\n" +
		"]>\n<r>&outer;<e a=\"&outer; &outer;\"/></r>"

	doc, err := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).
		DefaultDTDAttributes(true).SubstituteEntities(false).ValidateDTD(false).
		Parse(t.Context(), []byte(src))
	require.NoError(t, err, "a harmless indirect entity must be accepted in an attribute value")
	require.NotNil(t, doc)
}

// TestForwardReferencedEntityInAttributeDefault covers a DTD attribute default
// value that transitively references an external entity declared AFTER it (a
// forward reference). The parse-time check cannot see the entity yet, so the
// well-formedness violation is caught by the post-DTD re-validation once the
// entity tables are complete. An external subset makes the early undefined
// reference non-fatal, reproducing the case that would otherwise slip through.
func TestForwardReferencedEntityInAttributeDefault(t *testing.T) {
	t.Parallel()

	const doc = "<!DOCTYPE r SYSTEM \"d.dtd\" [\n" +
		"<!ELEMENT r EMPTY>\n" +
		"<!ENTITY outer \"&ext;\">\n" +
		"<!ATTLIST r a CDATA \"&outer;\">\n" +
		"<!ENTITY ext SYSTEM \"x\">\n" +
		"]>\n<r/>"
	fsys := fstest.MapFS{"d.dtd": {Data: []byte("<!-- external subset -->")}}

	got, err := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).
		DefaultDTDAttributes(true).SubstituteEntities(false).ValidateDTD(false).
		FS(fsys).Parse(t.Context(), []byte(doc))
	require.Error(t, err, "a forward-referenced external entity in a default value must be rejected")
	require.Nil(t, got)
	require.Contains(t, err.Error(), "attribute references external entity")
}

// TestDeepEntityChainInAttributeValueBounded confirms the attribute-value WFC
// walk traverses a long acyclic chain of nested internal entities without native
// call-stack recursion (the walker uses an explicit work stack) and does not
// false-reject a harmless plain-text terminus.
func TestDeepEntityChainInAttributeValueBounded(t *testing.T) {
	t.Parallel()

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
}
