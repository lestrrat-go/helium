package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestEntityAttrExternalSubset exercises the VC: Entity Name for ENTITY/ENTITIES
// attributes whose referenced unparsed entity is declared in the EXTERNAL
// subset. A validating processor loads the external subset even for a
// standalone="yes" document, so the entity lookup must search both subsets
// regardless of standalone (W3C xml sun/valid/sa03, sa04). Parsed and
// undeclared references must still be rejected.
func TestEntityAttrExternalSubset(t *testing.T) {
	t.Parallel()

	// External subset declaring an unparsed entity plus a parsed one, and the
	// ENTITY/ENTITIES attribute declarations that reference them.
	const extDTD = `<!ELEMENT doc EMPTY>
<!NOTATION nonce SYSTEM "nonce.exe">
<!ENTITY unparsed-1 SYSTEM "u1.dat" NDATA nonce>
<!ENTITY unparsed-2 PUBLIC "pub-u2" "u2.dat" NDATA nonce>
<!ENTITY parsed-1 SYSTEM "p1.xml">
<!ATTLIST doc one ENTITY #IMPLIED
              many ENTITIES #IMPLIED>`

	// The core fix: an ENTITY attribute referencing an externally-declared
	// unparsed entity validates even under standalone="yes".
	t.Run("ENTITY referencing external unparsed entity accepted", func(t *testing.T) {
		t.Parallel()
		errs, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE doc SYSTEM "ext.dtd">
<doc one="unparsed-1"/>`, extDTD)
		require.NoError(t, err)
		require.Empty(t, errs)
	})

	t.Run("ENTITIES referencing external unparsed entities accepted", func(t *testing.T) {
		t.Parallel()
		errs, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE doc SYSTEM "ext.dtd">
<doc many="unparsed-1 unparsed-2"/>`, extDTD)
		require.NoError(t, err)
		require.Empty(t, errs)
	})

	// Guard: an undeclared entity reference is still a validity error.
	t.Run("ENTITY referencing undeclared entity rejected", func(t *testing.T) {
		t.Parallel()
		errs, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE doc SYSTEM "ext.dtd">
<doc one="nonexistent"/>`, extDTD)
		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		require.True(t, containsError(errs, `references undeclared entity "nonexistent"`))
	})

	// Guard: a PARSED entity (external, no NDATA) is not a valid ENTITY value.
	t.Run("ENTITY referencing external parsed entity rejected", func(t *testing.T) {
		t.Parallel()
		errs, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE doc SYSTEM "ext.dtd">
<doc one="parsed-1"/>`, extDTD)
		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		require.True(t, containsError(errs, `references entity "parsed-1" which is not unparsed`))
	})

	// Guard: a PARSED entity declared in the INTERNAL subset is likewise not a
	// valid ENTITY value (the not-unparsed check spans both subsets).
	t.Run("ENTITY referencing internal parsed entity rejected", func(t *testing.T) {
		t.Parallel()
		errs, err := parseStandalone(t, `<?xml version="1.0" standalone="no"?>
<!DOCTYPE doc SYSTEM "ext.dtd" [
  <!ENTITY internal-parsed "text">
]>
<doc one="internal-parsed"/>`, extDTD)
		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		require.True(t, containsError(errs, `references entity "internal-parsed" which is not unparsed`))
	})
}
