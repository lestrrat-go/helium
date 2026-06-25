package helium_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestMaxNameLength(t *testing.T) {
	t.Parallel()

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
}

func TestMaxContentModelDepth(t *testing.T) {
	t.Parallel()

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
}
