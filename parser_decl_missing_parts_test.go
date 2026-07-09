package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestDeclMissingMandatoryPart covers three DTD/XML declarations that are
// missing a component the grammar makes mandatory. Each production has a
// malformed form that must now be a fatal well-formedness error AND a
// well-formed near-miss (including a present-but-empty literal) that must still
// parse, guarding against over-rejection.
func TestDeclMissingMandatoryPart(t *testing.T) {
	t.Parallel()

	// NotationDecl [82]: '<!NOTATION' S Name S (ExternalID | PublicID) S? '>'.
	// The ExternalID/PublicID is mandatory (W3C ibm-not-wf-P82-ibm82n03).
	t.Run("notation", func(t *testing.T) {
		t.Parallel()

		t.Run("missing ExternalID/PublicID rejected", func(t *testing.T) {
			t.Parallel()
			_, err := helium.NewParser().Parse(t.Context(),
				[]byte(`<?xml version="1.0"?><!DOCTYPE root [<!NOTATION n >]><root/>`))
			require.ErrorIs(t, err, helium.ErrNotationExternalIDRequired)
		})

		t.Run("SYSTEM form parses", func(t *testing.T) {
			t.Parallel()
			_, err := helium.NewParser().Parse(t.Context(),
				[]byte(`<?xml version="1.0"?><!DOCTYPE root [<!NOTATION n SYSTEM "n.dtd">]><root/>`))
			require.NoError(t, err)
		})

		t.Run("PUBLIC-only form parses", func(t *testing.T) {
			t.Parallel()
			_, err := helium.NewParser().Parse(t.Context(),
				[]byte(`<?xml version="1.0"?><!DOCTYPE root [<!NOTATION n PUBLIC "pub-id">]><root/>`))
			require.NoError(t, err)
		})

		t.Run("SYSTEM empty literal parses", func(t *testing.T) {
			t.Parallel()
			// A present-but-empty SystemLiteral is well formed; found=true so it
			// is not mistaken for a missing ExternalID.
			_, err := helium.NewParser().Parse(t.Context(),
				[]byte(`<?xml version="1.0"?><!DOCTYPE root [<!NOTATION n SYSTEM "">]><root/>`))
			require.NoError(t, err)
		})
	})

	// EntityDecl [73] EntityDef / [74] PEDef: EntityValue | ExternalID(...).
	// A declaration with neither is fatal (W3C o-p73fail4).
	t.Run("entity", func(t *testing.T) {
		t.Parallel()

		t.Run("general missing value/ExternalID rejected", func(t *testing.T) {
			t.Parallel()
			_, err := helium.NewParser().Parse(t.Context(),
				[]byte(`<?xml version="1.0"?><!DOCTYPE root [<!ENTITY ge >]><root/>`))
			require.ErrorIs(t, err, helium.ErrValueRequired)
		})

		t.Run("parameter missing value/ExternalID rejected", func(t *testing.T) {
			t.Parallel()
			_, err := helium.NewParser().Parse(t.Context(),
				[]byte(`<?xml version="1.0"?><!DOCTYPE root [<!ENTITY % pe >]><root/>`))
			require.ErrorIs(t, err, helium.ErrValueRequired)
		})

		t.Run("general EntityValue parses", func(t *testing.T) {
			t.Parallel()
			_, err := helium.NewParser().Parse(t.Context(),
				[]byte(`<?xml version="1.0"?><!DOCTYPE root [<!ENTITY ge "value">]><root/>`))
			require.NoError(t, err)
		})

		t.Run("general ExternalID parses", func(t *testing.T) {
			t.Parallel()
			// Declared but not referenced, so the external resource is never
			// loaded; the declaration alone must be accepted.
			_, err := helium.NewParser().Parse(t.Context(),
				[]byte(`<?xml version="1.0"?><!DOCTYPE root [<!ENTITY ge SYSTEM "ge.ent">]><root/>`))
			require.NoError(t, err)
		})

		t.Run("general SYSTEM empty literal parses", func(t *testing.T) {
			t.Parallel()
			_, err := helium.NewParser().Parse(t.Context(),
				[]byte(`<?xml version="1.0"?><!DOCTYPE root [<!ENTITY ge SYSTEM "">]><root/>`))
			require.NoError(t, err)
		})
	})

	// EncodingDecl [80]: S 'encoding' Eq ('"' EncName '"' | "'" EncName "'").
	// A present "encoding" keyword with no EncName is fatal (W3C
	// ibm-not-wf-P80-ibm80n03); an absent keyword is benign.
	t.Run("encoding", func(t *testing.T) {
		t.Parallel()

		t.Run("missing EncName (no quote) rejected", func(t *testing.T) {
			t.Parallel()
			_, err := helium.NewParser().Parse(t.Context(),
				[]byte(`<?xml version="1.0" encoding= ?><root/>`))
			require.Error(t, err)
		})

		t.Run("empty EncName rejected", func(t *testing.T) {
			t.Parallel()
			_, err := helium.NewParser().Parse(t.Context(),
				[]byte(`<?xml version="1.0" encoding=""?><root/>`))
			require.Error(t, err)
		})

		t.Run("valid EncName parses", func(t *testing.T) {
			t.Parallel()
			_, err := helium.NewParser().Parse(t.Context(),
				[]byte(`<?xml version="1.0" encoding="UTF-8"?><root/>`))
			require.NoError(t, err)
		})

		t.Run("absent encoding with standalone parses", func(t *testing.T) {
			t.Parallel()
			// No encoding keyword at all: the benign AttrNotFoundError must fall
			// through to the optional StandaloneDecl, not become fatal.
			_, err := helium.NewParser().Parse(t.Context(),
				[]byte(`<?xml version="1.0" standalone="yes"?><root/>`))
			require.NoError(t, err)
		})
	})
}
