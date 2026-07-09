package helium_test

import (
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestStandaloneExternalEntityReference exercises the WFC: Entity Declared
// (XML §4.1) as constrained by the Standalone Document Declaration (§2.9): in a
// standalone="yes" document a reference to a general entity declared ONLY in the
// external subset is a fatal well-formedness error, because a standalone="yes"
// document asserts that no external markup declarations affect its content. The
// same reference in a standalone="no" document, or a reference to an
// internally-declared entity in a standalone="yes" document, is well-formed and
// must still parse. This mirrors libxml2's xmlSAX2GetEntity XML_ERR_NOT_STANDALONE
// check and closes W3C not-wf cases ibm-not-wf-P32-ibm32n09, ibm-not-wf-P68-ibm68n06
// and not-wf-sa03.
func TestStandaloneExternalEntityReference(t *testing.T) {
	t.Parallel()

	const extDTD = "ext.dtd"

	newParser := func(fsys fstest.MapFS) helium.Parser {
		return helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			DefaultDTDAttributes(true).
			SubstituteEntities(true).
			FS(fsys)
	}

	// The external subset declares a general entity referenced by the document.
	extEntityFS := fstest.MapFS{
		extDTD: &fstest.MapFile{Data: []byte(`<!ENTITY ext "from external subset">` + "\n")},
	}

	t.Run("rejected: standalone yes, entity declared only externally, in content", func(t *testing.T) {
		t.Parallel()

		// ibm-not-wf-P32-ibm32n09 shape: reference in element content.
		doc := `<?xml version="1.0" standalone="yes"?>` + "\n" +
			`<!DOCTYPE root SYSTEM "` + extDTD + `" [` + "\n" +
			`<!ELEMENT root (#PCDATA)>` + "\n" +
			`]>` + "\n" +
			`<root>&ext;</root>`
		_, err := newParser(extEntityFS).Parse(t.Context(), []byte(doc))
		require.Error(t, err, "standalone=yes referencing an externally-declared entity must be a WF error")
		require.ErrorIs(t, err, helium.ErrNotStandalone)
	})

	t.Run("rejected: standalone yes, entity declared only externally, in attribute value", func(t *testing.T) {
		t.Parallel()

		// ibm-not-wf-P68-ibm68n06 / not-wf-sa03 shape: reference in an
		// attribute value.
		doc := `<?xml version="1.0" standalone="yes"?>` + "\n" +
			`<!DOCTYPE root SYSTEM "` + extDTD + `" [` + "\n" +
			`<!ELEMENT root (#PCDATA)>` + "\n" +
			`<!ATTLIST root att CDATA #IMPLIED>` + "\n" +
			`]>` + "\n" +
			`<root att="x-&ext;-y">content</root>`
		_, err := newParser(extEntityFS).Parse(t.Context(), []byte(doc))
		require.Error(t, err, "standalone=yes referencing an externally-declared entity in an attribute value must be a WF error")
		require.ErrorIs(t, err, helium.ErrNotStandalone)
	})

	t.Run("accepted: standalone no, entity declared externally", func(t *testing.T) {
		t.Parallel()

		// The identical document with standalone="no" is well-formed: external
		// declarations are permitted to supply entities.
		doc := `<?xml version="1.0" standalone="no"?>` + "\n" +
			`<!DOCTYPE root SYSTEM "` + extDTD + `" [` + "\n" +
			`<!ELEMENT root (#PCDATA)>` + "\n" +
			`]>` + "\n" +
			`<root>&ext;</root>`
		parsed, err := newParser(extEntityFS).Parse(t.Context(), []byte(doc))
		require.NoError(t, err, "standalone=no referencing an externally-declared entity must still parse")
		require.Equal(t, "from external subset", string(parsed.DocumentElement().Content()))
	})

	t.Run("accepted: standalone yes, entity declared internally", func(t *testing.T) {
		t.Parallel()

		// A standalone="yes" document referencing an entity declared in the
		// INTERNAL subset is well-formed even when an external subset is present.
		doc := `<?xml version="1.0" standalone="yes"?>` + "\n" +
			`<!DOCTYPE root SYSTEM "` + extDTD + `" [` + "\n" +
			`<!ELEMENT root (#PCDATA)>` + "\n" +
			`<!ENTITY ext "from internal subset">` + "\n" +
			`]>` + "\n" +
			`<root>&ext;</root>`
		parsed, err := newParser(extEntityFS).Parse(t.Context(), []byte(doc))
		require.NoError(t, err, "an internally-declared entity must resolve under standalone=yes")
		require.Equal(t, "from internal subset", string(parsed.DocumentElement().Content()))
	})
}
