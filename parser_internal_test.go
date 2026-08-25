package helium

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/strcursor"
	"github.com/lestrrat-go/helium/sax"
	"github.com/stretchr/testify/require"
)

// nilPEHandler embeds a TreeBuilder but reports every parameter entity as
// "not declared but not an error" — i.e. returns (nil, nil) — which drives
// parseStringPEReference down the branch that clears pctx.valid, in place of
// erroring on a missing PE. This is exactly the live-state mutation the
// validation pass must snapshot and restore.
type nilPEHandler struct {
	*TreeBuilder
}

func (h nilPEHandler) GetParameterEntity(context.Context, string) (sax.Entity, error) {
	return nil, nil //nolint:nilnil
}

// validateEntityValueRefs must leave the live parser state it touches UNCHANGED,
// on both the unresolved- and resolved-PE paths.
func TestValidateEntityValueRefs(t *testing.T) {
	t.Parallel()

	// The PE-expansion path resolves parameter-entity references through
	// parseStringPEReference, which mutates pctx.hasPERefs (always set true) and,
	// for an unresolved PE in a document with an external subset, clears pctx.valid.
	// Those mutations belong to the real parse, not to this throwaway syntax check.
	// validateEntityValueRefs must snapshot and restore hasPERefs and valid (and
	// sizeentcopy) so a failed PE-expanded validation does not perturb them.
	t.Run("state restored after a failed validation", func(t *testing.T) {
		t.Parallel()

		tb := NewTreeBuilder()
		handler := nilPEHandler{TreeBuilder: tb}
		doc := NewDocument("1.0", "", StandaloneImplicitNo)

		pctx := &parserCtx{}
		require.NoError(t, pctx.init(nil, bytes.NewReader(nil)))
		pctx.doc = doc
		pctx.sax = handler
		pctx.treeBuilder = tb

		// An external subset lets an UNRESOLVED PE reference take the path that clears
		// pctx.valid, in place of erroring out as a missing PE, while still always
		// setting pctx.hasPERefs. This is exactly the live-state mutation the
		// validation pass must not leak.
		pctx.hasExternalSubset = true

		ctx := withParserCtx(t.Context(), pctx)

		// Sanity: confirm the PE path actually mutates the fields, so the restore is
		// genuinely doing work. Run parseStringPEReference directly on an unresolved
		// PE and observe hasPERefs flip true and valid flip false.
		probe := &parserCtx{}
		require.NoError(t, probe.init(nil, bytes.NewReader(nil)))
		probe.doc = doc
		probe.sax = handler
		probe.treeBuilder = tb
		probe.hasExternalSubset = true
		probe.valid = true
		probe.hasPERefs = false
		probeCtx := withParserCtx(t.Context(), probe)
		_, _, perr := probe.parseStringPEReference(probeCtx, []byte("%missing;"))
		require.NoError(t, perr, "an unresolved PE with an external subset must not error")
		require.True(t, probe.hasPERefs, "parseStringPEReference must set hasPERefs (mutation under test)")
		require.False(t, probe.valid, "parseStringPEReference must clear valid (mutation under test)")

		// Known pre-state for the field-invariance assertion.
		pctx.valid = true
		pctx.hasPERefs = false
		pctx.sizeentcopy = 0

		// A literal with an unresolved PE reference followed by a malformed general
		// reference: the PE path mutates hasPERefs/valid, then the general-reference
		// scan fails on "&broken" (missing semicolon). The validation therefore
		// returns an error AND has touched the live state — exactly the case the
		// restore must cover.
		err := pctx.validateEntityValueRefs(ctx, []byte("%missing;&broken"))
		require.Error(t, err, "a malformed general reference must make validation fail")

		require.False(t, pctx.hasPERefs,
			"hasPERefs must be restored after a failed PE-expanded validation")
		require.True(t, pctx.valid,
			"valid must be restored after a failed PE-expanded validation")
		require.Equal(t, int64(0), pctx.sizeentcopy,
			"sizeentcopy must be restored after a failed PE-expanded validation")
	})

	// The restore also covers the resolved-PE path: a real parameter entity is
	// expanded during validation (setting hasPERefs and charging the amplification
	// counter), yet hasPERefs and sizeentcopy are restored to their pre-validation
	// values.
	t.Run("state restored after a resolved PE", func(t *testing.T) {
		t.Parallel()

		pctx := &parserCtx{}
		require.NoError(t, pctx.init(nil, bytes.NewReader(nil)))

		doc := NewDocument("1.0", "", StandaloneImplicitNo)
		dtd, err := doc.CreateInternalSubset("r", "", "")
		require.NoError(t, err)
		_, err = dtd.AddEntity("p", enum.InternalParameterEntity, "", "", "expansion")
		require.NoError(t, err)
		pctx.doc = doc

		tb := NewTreeBuilder()
		pctx.sax = tb
		pctx.treeBuilder = tb

		// A PE reference in an entity value is only expanded when the parser is
		// effectively external (external subset or external parameter entity); in the
		// internal subset it is a fatal WFC error. Put the context in the external
		// subset so the resolved-PE EXPANSION path — the subject of this test — runs.
		pctx.inSubset = inExternalSubset

		ctx := withParserCtx(t.Context(), pctx)

		pctx.valid = true
		pctx.hasPERefs = false
		pctx.sizeentcopy = 0

		// "%p;" resolves to "expansion" (no general reference), so validation
		// succeeds; the resolved-PE path still set hasPERefs and charged the counter.
		err = pctx.validateEntityValueRefs(ctx, []byte("%p;"))
		require.NoError(t, err, "a resolved PE with no general reference must validate cleanly")

		require.False(t, pctx.hasPERefs,
			"hasPERefs must be restored after a resolved-PE validation")
		require.True(t, pctx.valid, "valid must be restored after a resolved-PE validation")
		require.Equal(t, int64(0), pctx.sizeentcopy,
			"sizeentcopy must be restored after a resolved-PE validation")
	})
}

// An undeclared parameter entity must not panic anywhere in the decode path.
//
// For an undeclared parameter entity in a context with an external subset (or
// after a prior PE reference), parseStringPEReference deliberately returns a nil
// entity with NO error after clearing pctx.valid — the libxml2-faithful "PE not
// declared, validity error, keep going" convention that the nilPEHandler models.
// The '&' (general-entity) branch and expandEntityValueForRefCheck both guard
// that nil; the '%' branch must too, or it dereferences ent.Content() and panics
// the whole parse.
func TestUndefinedParameterEntity(t *testing.T) {
	// Drive the full Parse path with a SAX handler that uses the (nil, nil)
	// convention and assert no panic.
	t.Run("full parse does not panic", func(t *testing.T) {
		cases := []string{
			`<!DOCTYPE r SYSTEM "x" [<!ENTITY e "%missing;">]><r/>`,
			`<!DOCTYPE r SYSTEM "x" [<!ENTITY % p "%missing;">]><r/>`,
			`<!DOCTYPE r SYSTEM "x" [<!ENTITY e "a%missing;b">]><r/>`,
		}

		for _, input := range cases {
			t.Run(input, func(t *testing.T) {
				handler := nilPEHandler{TreeBuilder: NewTreeBuilder()}
				require.NotPanics(t, func() {
					_, _ = NewParser().SAXHandler(handler).Parse(t.Context(), []byte(input))
				})
			})
		}
	})

	// The decode branch directly: an undeclared parameter entity resolved through
	// the (nil, nil) convention must expand to nothing (consistent with
	// expandEntityValueForRefCheck), never panic, and must not surface an error.
	t.Run("decodeEntities expands it to nothing", func(t *testing.T) {
		pctx := &parserCtx{}
		require.NoError(t, pctx.init(nil, bytes.NewReader(nil)))
		doc := NewDocument("1.0", "", StandaloneImplicitNo)
		tb := NewTreeBuilder()
		pctx.doc = doc
		pctx.sax = nilPEHandler{TreeBuilder: tb}
		pctx.treeBuilder = tb
		pctx.hasExternalSubset = true

		ctx := withParserCtx(t.Context(), pctx)

		var out string
		var err error
		require.NotPanics(t, func() {
			out, err = pctx.decodeEntities(ctx, []byte("a%missing;b"), SubstitutePERef)
		})
		require.NoError(t, err, "an undeclared PE in an external-subset context is a validity error, not fatal")
		require.Equal(t, "ab", out, "the undeclared PE must expand to nothing")
		// The unresolved PE reference must still be charged against entity-expansion
		// accounting (reference width + per-reference fixed cost), otherwise an
		// undeclared PE becomes a free way to dodge the amplification/ceiling limits.
		require.Equal(t, int64(len("%missing;"))+entityFixedCost, pctx.sizeentcopy,
			"unresolved PE reference must be charged against sizeentcopy")
	})
}

// TestEntityHardCeiling verifies the absolute entity-expansion ceiling trips
// even when the amplification ratio check is disabled
// (MaxEntityAmplification(-1)). It lowers entityHardCeiling for the duration of
// the test so the ceiling can be exercised with a modest document, well short of
// the production 1 GB cap (which risked CI OOM).
func TestEntityHardCeiling(t *testing.T) {
	orig := entityHardCeiling
	entityHardCeiling = 50_000 // tiny ceiling: trips well under any real memory
	defer func() { entityHardCeiling = orig }()

	// A billion-laughs document whose expansion comfortably exceeds the lowered
	// ceiling but stays small in absolute terms.
	xml := `<?xml version="1.0"?>
<!DOCTYPE lolz [
  <!ENTITY lol "lol">
  <!ENTITY lol2 "&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;">
  <!ENTITY lol3 "&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;">
  <!ENTITY lol4 "&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;">
  <!ENTITY lol5 "&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;&lol4;">
]>
<root>&lol5;</root>`

	p := NewParser().SubstituteEntities(true).MaxEntityAmplification(-1)
	_, err := p.Parse(context.Background(), []byte(xml))
	require.Error(t, err, "the absolute ceiling must trip even with the ratio check disabled")
	require.Contains(t, err.Error(), "maximum entity expansion size",
		"error must explain the ceiling, got: %v", err)
}

// blankThenReadErrReader yields blanks blank bytes and then fails every
// subsequent Read with err. It models the push parser's stream, whose blocking
// Read returns context.Canceled when cancellation unblocks a pending wait: the
// ByteCursor records that as a sticky Err() while PeekAt reports 0 (the same 0 a
// genuine non-blank byte / clean EOF yields).
type blankThenReadErrReader struct {
	blanks int
	served int
	err    error
}

func (r *blankThenReadErrReader) Read(p []byte) (int, error) {
	if r.served < r.blanks {
		n := min(len(p), r.blanks-r.served)
		for i := range n {
			p[i] = ' '
		}
		r.served += n
		return n, nil
	}
	return 0, r.err
}

func TestSkipBlankBytes(t *testing.T) {
	t.Run("a read error surfaces", testSkipBlankBytesReadError)
}

// testSkipBlankBytesReadError pins the cancellation contract at the blank-scan
// layer: a sticky cursor read error (a push-stream Read returning
// context.Canceled) must be surfaced through pctx.blankRunErr so callers such as
// parseXMLDecl propagate context.Canceled instead of synthesizing a syntax error
// like "blank needed after '<?xml'".
//
// ctx is context.Background() (never cancelled) on purpose: the ONLY signal of
// the cancellation is the cursor's sticky Err(). Treating PeekAt==0 as "no blank
// / EOF" and returning a nil error would leave blankRunErr unset and mask the
// read failure. ctx.Err() at the top of the scan loop cannot rescue this case
// here because ctx itself is not cancelled.
func testSkipBlankBytesReadError(t *testing.T) {
	cases := map[string]struct {
		blanks       int  // blanks buffered before the read error
		wantAdvanced bool // whether any whitespace was consumed first
	}{
		// First peek already hits the read error (no buffered blank): the
		// i==0 branch must consult the sticky Err() and never report "no blank".
		"read error on first peek": {blanks: 0, wantAdvanced: false},
		// Some blanks are consumed, then the read error stops the run short of a
		// full chunk: the partial-chunk branch must surface the sticky Err() too.
		"read error after some blanks": {blanks: 3, wantAdvanced: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := &blankThenReadErrReader{blanks: tc.blanks, err: context.Canceled}
			cur := strcursor.NewByteCursor(r)

			pctx := &parserCtx{}
			advanced := pctx.skipBlankBytes(context.Background(), cur)

			require.Equal(t, tc.wantAdvanced, advanced,
				"skipBlankBytes should report whether any whitespace was consumed")
			require.ErrorIs(t, pctx.blankRunErr, context.Canceled,
				"a sticky cursor read error must be surfaced through blankRunErr, not masked as no-blank")
		})
	}
}

// competingNonFillerAttrs is the number of non-filler attributes
// buildCompetingErrorStartTag puts on the tag (q:b, p:a, r:a). The two xmlns
// declarations are namespace declarations, not attributes, so they never
// count toward the len(attrs) that attrDupSetThreshold is compared against.
const competingNonFillerAttrs = 3

// buildCompetingErrorStartTag builds a start tag carrying BOTH an
// unbound-prefix attribute (q:b) and, at higher indices, an expanded-name
// duplicate pair (p:a and r:a, whose prefixes both resolve to urn:x).
// fillerCount padN attributes sit between the namespace declarations and the
// three, so the caller can place len(attrs) on either side of
// attrDupSetThreshold while keeping the logical input identical.
func buildCompetingErrorStartTag(fillerCount int) string {
	var b strings.Builder
	b.WriteString(`<root xmlns:p="urn:x" xmlns:r="urn:x"`)
	for i := range fillerCount {
		fmt.Fprintf(&b, " pad%d=\"v%d\"", i, i)
	}
	b.WriteString(` q:b="1" p:a="2" r:a="3"/>`)
	return b.String()
}

// TestStartTagDuplicateAttributeAboveThreshold pins the start-tag
// duplicate-detection behaviour of a tag whose attribute count crosses
// attrDupSetThreshold, where duplicate detection switches from a linear
// scan to a map-backed set. Attribute counts are derived from
// attrDupSetThreshold, not hardcoded, so the tests keep exercising the set
// path if the threshold constant ever changes.
func TestStartTagDuplicateAttributeAboveThreshold(t *testing.T) {
	t.Parallel()

	t.Run("qualified name duplicate rejected", func(t *testing.T) {
		t.Parallel()

		// attrDupSetThreshold+8 distinct local attributes, then one more
		// repeating the very first name: the duplicate is only discovered
		// once the map-backed set (built once len(attrs) crosses the
		// threshold) is consulted.
		n := attrDupSetThreshold + 8
		var b strings.Builder
		b.WriteString("<root")
		for i := range n {
			fmt.Fprintf(&b, " a%d=\"v%d\"", i, i)
		}
		fmt.Fprintf(&b, " a0=\"dup\"/>")

		_, err := NewParser().Parse(t.Context(), []byte(b.String()))
		require.Error(t, err, "a start tag with a duplicate qualified name past the threshold must be rejected")
		require.Contains(t, err.Error(), "duplicate attribute is not allowed")
	})

	t.Run("all distinct attributes accepted in order", func(t *testing.T) {
		t.Parallel()

		n := attrDupSetThreshold + 8
		var b strings.Builder
		b.WriteString("<root")
		for i := range n {
			fmt.Fprintf(&b, " a%d=\"v%d\"", i, i)
		}
		b.WriteString("/>")

		doc, err := NewParser().Parse(t.Context(), []byte(b.String()))
		require.NoError(t, err, "a start tag with N>threshold distinct attributes must be accepted")

		root := doc.DocumentElement()
		require.NotNil(t, root)
		attrs := root.Attributes()
		require.Len(t, attrs, n, "all attributes must survive onto the DOM element")
		for i, a := range attrs {
			require.Equal(t, fmt.Sprintf("a%d", i), a.Name(), "attribute order must be preserved")
			require.Equal(t, fmt.Sprintf("v%d", i), a.Value())
		}
	})

	t.Run("expanded name duplicate across distinct prefixes rejected", func(t *testing.T) {
		t.Parallel()

		// p:a and q:a, both bound to the same URI, are the same expanded
		// name (Namespaces in XML §6.3) even though their qualified names
		// differ. Filler attributes push the tag past attrDupSetThreshold
		// so the expandedLast map path is exercised.
		fillerCount := attrDupSetThreshold + 6
		var b strings.Builder
		b.WriteString(`<root xmlns:p="urn:x" xmlns:q="urn:x" p:a="1"`)
		for i := range fillerCount {
			fmt.Fprintf(&b, " pad%d=\"v%d\"", i, i)
		}
		b.WriteString(` q:a="2"/>`)

		_, err := NewParser().Parse(t.Context(), []byte(b.String()))
		require.Error(t, err, "distinct prefixes bound to the same URI on the same local name must be rejected")
		require.Contains(t, err.Error(), "duplicate attribute is not allowed")

		var pe ErrParseError
		require.ErrorAs(t, err, &pe)
		require.NotEqual(t, ErrorDomainNamespace, pe.Domain,
			"the expanded-name duplicate must be reported as a duplicate-attribute error, not a namespace error")
	})

	t.Run("unbound prefix reported before a later expanded name duplicate", func(t *testing.T) {
		t.Parallel()

		// q:b's prefix is never declared. It appears before both p:a and
		// r:a (which share p's URI and so collide on expanded name) so
		// that, scanning attributes in document order, the unbound-prefix
		// namespace error is what today's parser reaches first — the
		// duplicate collision between p:a and r:a sits at higher indices
		// and is never reached.
		//
		// Crossing attrDupSetThreshold must change neither WHICH of the two
		// competing errors wins, nor its message, nor its position. So the
		// same logical input is parsed on both sides of the threshold and
		// every case is pinned all the way down to the reported line and
		// column.
		cases := []struct {
			name        string
			fillerCount int
			wantSetPath bool
		}{
			// One attribute short of the threshold: expandedLast stays nil
			// and the original quadratic inner loop decides the outcome.
			{"below threshold, linear scan", attrDupSetThreshold - competingNonFillerAttrs - 1, false},
			// Exactly at the threshold: the lowest count at which
			// expandedLast is built.
			{"at threshold, expandedLast built", attrDupSetThreshold - competingNonFillerAttrs, true},
			// Comfortably past the threshold.
			{"above threshold, expandedLast built", attrDupSetThreshold, true},
		}

		var first ErrParseError
		for i, tc := range cases {
			src := buildCompetingErrorStartTag(tc.fillerCount)
			require.Equal(t, tc.wantSetPath, tc.fillerCount+competingNonFillerAttrs >= attrDupSetThreshold,
				"%s: the case must sit on the intended side of attrDupSetThreshold", tc.name)

			// The namespace check runs only after the whole start tag has
			// been consumed, so the error is reported at the tag's "/>"
			// terminator. Deriving the expected column from the input pins
			// it exactly without baking in a literal that would shift every
			// time the filler count changes.
			term := strings.Index(src, "/>")
			require.Positive(t, term)
			wantColumn := term + 1
			wantSnippet := src[:term]

			_, err := NewParser().Parse(t.Context(), []byte(src))
			require.Error(t, err, "%s: an unbound attribute prefix must be rejected", tc.name)

			var pe ErrParseError
			require.ErrorAs(t, err, &pe, "%s: the unbound-prefix error must be a helium.ErrParseError", tc.name)
			require.Equal(t, ErrorDomainNamespace, pe.Domain,
				"%s: an unbound prefix reached before the expanded-name duplicate must report the namespace error, not the duplicate error", tc.name)
			require.Contains(t, err.Error(), "namespace 'q' not found",
				"%s: the unbound prefix q, not p or r, must be the one reported", tc.name)
			require.NotNil(t, pe.Err, "%s: the parse error must wrap its underlying cause", tc.name)
			require.Equal(t, "namespace 'q' not found", pe.Err.Error(),
				"%s: the wrapped cause pins the message independently of the position suffix", tc.name)
			require.Equal(t, 1, pe.LineNumber, "%s: the whole document is a single line", tc.name)
			require.Equal(t, wantColumn, pe.Column,
				"%s: the error must be reported at the start tag's '/>' terminator", tc.name)
			require.Equal(t, wantSnippet, pe.Line,
				"%s: the reported context snippet must be the start tag up to its terminator", tc.name)

			if i == 0 {
				first = pe
				continue
			}
			// The cross-threshold invariant. Only the absolute column moves
			// between cases, and only because the filler attributes shift the
			// terminator's byte offset, which each case already pins exactly
			// above. Everything else must be identical to the below-threshold
			// parse.
			require.Equal(t, first.Domain, pe.Domain, "%s: the threshold must not change the error domain", tc.name)
			require.Equal(t, first.Level, pe.Level, "%s: the threshold must not change the error level", tc.name)
			require.Equal(t, first.Err.Error(), pe.Err.Error(), "%s: the threshold must not change the error message", tc.name)
			require.Equal(t, first.LineNumber, pe.LineNumber, "%s: the threshold must not change the reported line", tc.name)
		}
	})
}

// TestAddAttributeDefaultAboveThreshold pins addAttributeDefault's
// duplicate-suppression behaviour once an element type accumulates enough
// <!ATTLIST> defaults to cross attrDupSetThreshold worth of declarations:
// each default is applied exactly once, and a repeated declaration for an
// already-declared attribute keeps the first declaration's value.
func TestAddAttributeDefaultAboveThreshold(t *testing.T) {
	t.Parallel()

	n := attrDupSetThreshold + 8
	var dtd strings.Builder
	dtd.WriteString("<!DOCTYPE root [\n<!ELEMENT root EMPTY>\n<!ATTLIST root\n")
	for i := range n {
		fmt.Fprintf(&dtd, "  a%d CDATA \"v%d\"\n", i, i)
	}
	// A repeated declaration for a0 with a different default value: the
	// first declaration (XML 1.0 §3.3) must win.
	dtd.WriteString("  a0 CDATA \"ignored\">\n")
	dtd.WriteString("]>\n<root/>")

	doc, err := NewParser().DefaultDTDAttributes(true).Parse(t.Context(), []byte(dtd.String()))
	require.NoError(t, err)

	root := doc.DocumentElement()
	require.NotNil(t, root)
	attrs := root.Attributes()
	require.Len(t, attrs, n, "each default must be applied exactly once")

	seen := map[string]struct{}{}
	for _, a := range attrs {
		_, dup := seen[a.Name()]
		require.False(t, dup, "attribute %q applied more than once", a.Name())
		seen[a.Name()] = struct{}{}
	}

	a0, ok := findAttribute(attrs, "a0")
	require.True(t, ok, "a0 must be present")
	require.Equal(t, "v0", a0.Value(), "the first <!ATTLIST> declaration for a repeated attribute must win")
}

func findAttribute(attrs []*Attribute, name string) (*Attribute, bool) {
	for _, a := range attrs {
		if a.Name() == name {
			return a, true
		}
	}
	return nil, false
}

// nsDeclTag builds a start tag carrying n distinct xmlns:pN declarations
// and, if dup, one more xmlns:p0 repeating the first prefix — a same-element
// duplicate that must be rejected regardless of parseNsClean.
func nsDeclTag(n int, dup bool) string {
	var b strings.Builder
	b.WriteString("<root")
	for i := range n {
		fmt.Fprintf(&b, ` xmlns:p%d="urn:x%d"`, i, i)
	}
	if dup {
		b.WriteString(` xmlns:p0="urn:dup"`)
	}
	b.WriteString("/>")
	return b.String()
}

// TestNamespaceDeclarationDuplicateAboveThreshold pins the same-element
// namespace-declaration duplicate check (nsDeclared) once a tag's
// declaration count crosses attrDupSetThreshold, where the check switches
// from a linear nsDeclared scan to a map-backed set. CleanNamespaces(true)
// (parseNsClean) only suppresses a redundant ANCESTOR redeclaration; it must
// not affect a same-element duplicate either way.
func TestNamespaceDeclarationDuplicateAboveThreshold(t *testing.T) {
	t.Parallel()

	n := attrDupSetThreshold + 8

	t.Run("distinct declarations accepted", func(t *testing.T) {
		t.Parallel()

		doc, err := NewParser().Parse(t.Context(), []byte(nsDeclTag(n, false)))
		require.NoError(t, err, "n>threshold distinct xmlns declarations must be accepted")
		root := doc.DocumentElement()
		require.NotNil(t, root)
	})

	t.Run("duplicate rejected", func(t *testing.T) {
		t.Parallel()

		_, err := NewParser().Parse(t.Context(), []byte(nsDeclTag(n, true)))
		require.Error(t, err, "a same-element duplicate xmlns declaration past the threshold must be rejected")
		require.Contains(t, err.Error(), "duplicate attribute is not allowed")
	})

	t.Run("duplicate rejected with CleanNamespaces", func(t *testing.T) {
		t.Parallel()

		_, err := NewParser().CleanNamespaces(true).Parse(t.Context(), []byte(nsDeclTag(n, true)))
		require.Error(t, err, "CleanNamespaces must not suppress a same-element duplicate xmlns declaration past the threshold")
		require.Contains(t, err.Error(), "duplicate attribute is not allowed")
	})
}
