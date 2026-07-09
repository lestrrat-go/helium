package helium

import (
	"bytes"
	"context"
	"testing"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/sax"
	"github.com/stretchr/testify/require"
)

// nilPEHandler embeds a TreeBuilder but reports every parameter entity as
// "not declared but not an error" — i.e. returns (nil, nil) — which drives
// parseStringPEReference down the branch that clears pctx.valid (rather than
// erroring on a missing PE). This is exactly the live-state mutation the
// validation pass must snapshot and restore.
type nilPEHandler struct {
	*TreeBuilder
}

func (h nilPEHandler) GetParameterEntity(context.Context, string) (sax.Entity, error) {
	return nil, nil //nolint:nilnil
}

// TestValidateEntityValueRefsSideEffectFree proves that validateEntityValueRefs
// leaves the live parser state it touches UNCHANGED, even when validation fails.
//
// The PE-expansion path resolves parameter-entity references through
// parseStringPEReference, which mutates pctx.hasPERefs (always set true) and,
// for an unresolved PE in a document with an external subset, clears pctx.valid.
// Those mutations belong to the real parse, not to this throwaway syntax check.
// validateEntityValueRefs must snapshot and restore hasPERefs and valid (and
// sizeentcopy) so a failed PE-expanded validation does not perturb them.
func TestValidateEntityValueRefsSideEffectFree(t *testing.T) {
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
	// pctx.valid (rather than erroring out as a missing PE), while still always
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
}

// TestValidateEntityValueRefsRestoresOnResolvedPE proves the restore also covers
// the resolved-PE path: a real parameter entity is expanded during validation
// (setting hasPERefs and charging the amplification counter), yet hasPERefs and
// sizeentcopy are restored to their pre-validation values.
func TestValidateEntityValueRefsRestoresOnResolvedPE(t *testing.T) {
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
}

// TestEntityHardCeiling verifies the absolute entity-expansion ceiling trips
// even when the amplification ratio check is disabled
// (MaxEntityAmplification(-1)). It lowers entityHardCeiling for the duration of
// the test so the ceiling can be exercised with a modest document rather than
// expanding toward the production 1 GB cap (which risked CI OOM).
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
