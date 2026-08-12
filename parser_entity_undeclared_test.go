package helium_test

import (
	"errors"
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestUndeclaredEntity(t *testing.T) {
	t.Parallel()

	// The SAME document is accepted by a non-validating processor: with an internal
	// PE reference present the undeclared entity is only a validity constraint, not a
	// well-formedness error, so it is a warning and the parse succeeds.
	t.Run("accepted when not validating", func(t *testing.T) {
		const src = `<!DOCTYPE foo [
<!ENTITY % pe "<!ENTITY ent1 'text'>">
%pe;
<!ELEMENT foo ANY>
]>
<foo>&ent2;</foo>`
		doc, err := helium.NewParser().
			SubstituteEntities(true).
			Parse(t.Context(), []byte(src))
		require.NoError(t, err, "a non-validating parse only warns on the undeclared entity")
		require.NotNil(t, doc)
	})

	// An internal parameter-entity reference downgrades an undeclared general entity
	// from a fatal well-formedness error to the "Entity Declared" VALIDITY
	// constraint. In a fully-internal DTD a validating processor must report it (W3C
	// xmlconf rmt-e3e-13).
	t.Run("a validity error when validating", func(t *testing.T) {
		const src = `<!DOCTYPE foo [
<!ENTITY % pe "<!ENTITY ent1 'text'>">
%pe;
<!ELEMENT foo ANY>
]>
<foo>&ent2;</foo>`
		_, err := helium.NewParser().
			SubstituteEntities(true).
			ValidateDTD(true).
			Parse(t.Context(), []byte(src))
		require.Error(t, err, "an undeclared entity must be a validity error when validating")
		require.Contains(t, err.Error(), "undeclared entity")
	})

	// When an EXTERNAL parameter entity is involved, helium cannot be certain the
	// entity is not declared in unread/incompletely-resolved external markup, so it
	// stays lenient even when validating — a still-undeclared entity is NOT promoted
	// to a fatal error (guards against over-rejecting a valid document; W3C
	// rmt-e2e-18).
	t.Run("lenient with an external PE", func(t *testing.T) {
		const src = `<!DOCTYPE foo [
<!ENTITY % pe SYSTEM "pe.ent">
%pe;
<!ELEMENT foo ANY>
]>
<foo>&ent2;</foo>`
		fsys := fstest.MapFS{"pe.ent": &fstest.MapFile{Data: []byte("<!-- external PE, declares nothing -->")}}
		doc, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			SubstituteEntities(true).
			ValidateDTD(true).
			FS(fsys).
			Parse(t.Context(), []byte(src))
		require.NoError(t, err, "an external PE keeps the undeclared entity a non-fatal warning")
		require.NotNil(t, doc)
	})

	// A present-but-empty external ID (`SYSTEM ""`) still marks an external subset,
	// so the fully-internal precondition of the undeclared-entity VC is not met and
	// the undeclared entity stays a non-fatal warning even when validating.
	t.Run("lenient with an empty external ID", func(t *testing.T) {
		const src = `<!DOCTYPE foo SYSTEM "" [
<!ELEMENT foo ANY>
]>
<foo>&ent2;</foo>`
		doc, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			SubstituteEntities(true).
			ValidateDTD(true).
			FS(fstest.MapFS{}).
			Parse(t.Context(), []byte(src))
		require.NoError(t, err, "a present-but-empty external ID keeps the undeclared entity a non-fatal warning")
		require.NotNil(t, doc)
	})

	// The same attribute-value case parses (only warns) when NOT validating.
	t.Run("in an attribute, accepted when not validating", func(t *testing.T) {
		const src = `<!DOCTYPE foo [
<!ENTITY % pe "<!ENTITY ent1 'text'>">
%pe;
<!ELEMENT foo EMPTY>
<!ATTLIST foo a CDATA #IMPLIED>
<!ENTITY bad "&ent2;">
]>
<foo a="&bad;"/>`
		doc, err := helium.NewParser().
			SubstituteEntities(true).
			Parse(t.Context(), []byte(src))
		require.NoError(t, err, "a non-validating parse only warns on the undeclared entity")
		require.NotNil(t, doc)
	})

	// The external-PE leniency holds for the attribute-value path too: with an
	// external PE involved, a still-undeclared entity referenced from an attribute
	// value stays a non-fatal warning even when validating.
	t.Run("in an attribute, lenient with an external PE", func(t *testing.T) {
		const src = `<!DOCTYPE foo [
<!ENTITY % pe SYSTEM "pe.ent">
%pe;
<!ELEMENT foo EMPTY>
<!ATTLIST foo a CDATA #IMPLIED>
<!ENTITY bad "&ent2;">
]>
<foo a="&bad;"/>`
		fsys := fstest.MapFS{"pe.ent": &fstest.MapFile{Data: []byte("<!-- external PE, declares nothing -->")}}
		doc, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			SubstituteEntities(true).
			ValidateDTD(true).
			FS(fsys).
			Parse(t.Context(), []byte(src))
		require.NoError(t, err, "an external PE keeps the attribute-value undeclared entity a warning")
		require.NotNil(t, doc)
	})

	// The "Entity Declared" VC applies to attribute values too, not only element
	// content. An undeclared general entity nested inside an entity referenced from
	// an attribute value is a validity error under a fully-internal DTD when
	// validating — through the substitute-entities string path.
	t.Run("in an attribute, a validity error on the substitute path", func(t *testing.T) {
		const src = `<!DOCTYPE foo [
<!ENTITY % pe "<!ENTITY ent1 'text'>">
%pe;
<!ELEMENT foo EMPTY>
<!ATTLIST foo a CDATA #IMPLIED>
<!ENTITY bad "&ent2;">
]>
<foo a="&bad;"/>`
		_, err := helium.NewParser().
			SubstituteEntities(true).
			ValidateDTD(true).
			Parse(t.Context(), []byte(src))
		require.Error(t, err, "an undeclared entity in an attribute value must be a validity error")
		require.Contains(t, err.Error(), "undeclared entity")
	})

	// The same attribute-value case through the NON-substituting attribute-value WFC
	// walk (handleUndeclaredEntity) is equally a validity error when validating.
	t.Run("in an attribute, a validity error on the well-formedness walk", func(t *testing.T) {
		const src = `<!DOCTYPE foo [
<!ENTITY % pe "<!ENTITY ent1 'text'>">
%pe;
<!ELEMENT foo EMPTY>
<!ATTLIST foo a CDATA #IMPLIED>
<!ENTITY bad "&ent2;">
]>
<foo a="&bad;"/>`
		_, err := helium.NewParser().
			ValidateDTD(true).
			Parse(t.Context(), []byte(src))
		require.Error(t, err, "an undeclared entity in an attribute value must be a validity error")
		require.Contains(t, err.Error(), "undeclared entity")
	})

	t.Run("fatal without a DTD", func(t *testing.T) {
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
	})

	// An internal general entity whose replacement text references an undefined
	// entity, referenced from an attribute value, must be reported as a
	// well-formedness error — not crash the parser. getEntity returns a typed-nil
	// *Entity for the undefined inner entity, which a naive interface nil-check
	// misses. (W3C not-wf-sa-077.)
	t.Run("a nested undefined entity in an attribute value does not panic", func(t *testing.T) {
		const doc = `<!DOCTYPE doc [
<!ENTITY foo "&bar;">
]>
<doc a="&foo;"></doc>`

		require.NotPanics(t, func() {
			_, err := helium.NewParser().Parse(t.Context(), []byte(doc))
			require.Error(t, err, "a reference to an undefined entity must be a well-formedness error")
		})
	})
}
