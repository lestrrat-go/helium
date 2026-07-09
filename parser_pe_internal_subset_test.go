package helium_test

import (
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestPEReferenceInInternalSubsetEntityValue exercises the "PEs in Internal
// Subset" WFC (XML §2.8): in the internal DTD subset a parameter-entity
// reference must not occur WITHIN a markup declaration, only where a markup
// declaration can occur. A PE reference inside an EntityValue literal is a fatal
// well-formedness error in the internal subset, while the same construct is
// permitted in the external subset (and within an external parameter entity).
// This matches libxml2's xmlExpandPEsInEntityValue PARSER_EXTERNAL gate and
// closes W3C not-wf cases not-wf-sa-160, not-wf-sa-162, ibm-not-wf-P29-ibm29n04,
// ibm-not-wf-P69-ibm69n06 and ibm-not-wf-P69-ibm69n07.
func TestPEReferenceInInternalSubsetEntityValue(t *testing.T) {
	t.Parallel()

	newParser := func() helium.Parser {
		return helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			DefaultDTDAttributes(true).
			SubstituteEntities(true).
			FS(helium.PermissiveFS())
	}

	t.Run("rejected in internal subset", func(t *testing.T) {
		t.Parallel()

		notWF := map[string]string{
			// not-wf-sa-160: general entity value contains a PE reference.
			"general entity value": "<!DOCTYPE doc [\n" +
				"<!ELEMENT doc (#PCDATA)>\n" +
				"<!ENTITY % e \"\">\n" +
				"<!ENTITY foo \"%e;\">\n" +
				"]>\n<doc></doc>",
			// not-wf-sa-162: parameter entity value contains a PE reference.
			"parameter entity value": "<!DOCTYPE doc [\n" +
				"<!ELEMENT doc (#PCDATA)>\n" +
				"<!ENTITY % e1 \"\">\n" +
				"<!ENTITY % e2 \"%e1;\">\n" +
				"]>\n<doc></doc>",
			// ibm-not-wf-P29-ibm29n04: PE reference inside an entity declaration.
			"non-empty PE in entity value": "<!DOCTYPE animal [\n" +
				"<!ELEMENT animal ANY>\n" +
				"<!ENTITY % parameterE \"A leopard\">\n" +
				"<!ENTITY content \"%parameterE;\">\n" +
				"]>\n<animal>stuff</animal>",
			// ibm-not-wf-P69-ibm69n06: the recursive PE cycle is unreachable
			// because the PE reference inside <!ENTITY bbb "%paaa;"> is rejected
			// first.
			"recursive PE via general entity": "<!DOCTYPE root [\n" +
				"<!ELEMENT root (#PCDATA)>\n" +
				"<!ENTITY % paaa \"&bbb;\">\n" +
				"<!ENTITY bbb \"%paaa;\">\n" +
				"]>\n<root/>",
		}

		for name, src := range notWF {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				_, err := newParser().Parse(t.Context(), []byte(src))
				require.Error(t, err, "a PE reference within an internal-subset declaration is not well formed")
				require.Contains(t, err.Error(), "PEReferences forbidden in internal subset")
			})
		}
	})

	t.Run("permitted in external subset", func(t *testing.T) {
		t.Parallel()

		// The very same construct (a PE reference inside an entity value) is
		// legal in the external subset, where it must still expand normally.
		fsys := fstest.MapFS{
			"pe-ext.dtd": &fstest.MapFile{Data: []byte(
				"<!ENTITY % e \"expanded\">\n<!ENTITY foo \"%e;\">\n")},
		}
		const src = "<!DOCTYPE doc SYSTEM \"pe-ext.dtd\" [\n<!ELEMENT doc (#PCDATA)>\n]>\n<doc>&foo;</doc>"

		p := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			DefaultDTDAttributes(true).
			SubstituteEntities(true).
			FS(fsys)
		doc, err := p.Parse(t.Context(), []byte(src))
		require.NoError(t, err, "a PE reference in an external-subset entity value is well formed")
		require.Equal(t, "expanded", string(doc.DocumentElement().Content()),
			"the external-subset PE must still expand inside the entity value")
	})

	t.Run("PE between declarations in internal subset is well formed", func(t *testing.T) {
		t.Parallel()

		// A PE reference BETWEEN declarations (supplying a complete markup
		// declaration) is where PE references may occur in the internal subset,
		// so it must parse.
		const src = "<!DOCTYPE doc [\n" +
			"<!ENTITY % decls \"<!ELEMENT doc (#PCDATA)>\">\n" +
			"%decls;\n" +
			"<!ENTITY greeting \"hi\">\n" +
			"]>\n<doc>&greeting;</doc>"

		doc, err := newParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err, "a PE reference between declarations is well formed in the internal subset")
		require.Equal(t, "hi", string(doc.DocumentElement().Content()))
	})
}
