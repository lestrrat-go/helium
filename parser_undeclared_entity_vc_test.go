package helium_test

import (
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// An internal parameter-entity reference downgrades an undeclared general entity
// from a fatal well-formedness error to the "Entity Declared" VALIDITY
// constraint. In a fully-internal DTD a validating processor must report it (W3C
// xmlconf rmt-e3e-13).
func TestUndeclaredEntityValidityErrorWhenValidating(t *testing.T) {
	t.Parallel()

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
}

// The SAME document is accepted by a non-validating processor: with an internal
// PE reference present the undeclared entity is only a validity constraint, not a
// well-formedness error, so it is a warning and the parse succeeds.
func TestUndeclaredEntityAcceptedWhenNotValidating(t *testing.T) {
	t.Parallel()

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
}

// When an EXTERNAL parameter entity is involved, helium cannot be certain the
// entity is not declared in unread/incompletely-resolved external markup, so it
// stays lenient even when validating — a still-undeclared entity is NOT promoted
// to a fatal error (guards against over-rejecting a valid document; W3C
// rmt-e2e-18).
func TestUndeclaredEntityLenientWithExternalPE(t *testing.T) {
	t.Parallel()

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
}
