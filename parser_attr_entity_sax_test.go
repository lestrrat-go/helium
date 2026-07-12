package helium_test

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/sax"
	"github.com/stretchr/testify/require"
)

// TestIndirectEntityRefInAttributeValueSAXMode proves the attribute-value WFC
// walk resolves nested references through the SAX getEntity callback, so a pure
// SAX-event parse — a custom handler that REPLACES the tree builder, leaving no
// document being built and thus an empty document entity table — still catches
// an INDIRECT external reference in an attribute value.
//
// The document references the outer entity in element CONTENT before the
// attribute. Under SubstituteEntities(false) that content reference marks the
// entity checked, so the attribute-value path cannot lean on a one-shot
// replacement-text re-validation gated on the checked bit; the memoized
// attribute-value WFC walk must run regardless, resolving the nested external
// entity — reachable ONLY through the custom handler's GetEntity — and rejecting
// it ("No External Entity References"). A document-table-only walk misses this.
func TestIndirectEntityRefInAttributeValueSAXMode(t *testing.T) {
	t.Parallel()

	// An external parsed entity is legal in element content but a WFC violation
	// when reached (directly or indirectly) from an attribute value, so it is the
	// clean way to check an entity in content first and still violate in an
	// attribute. It is known ONLY to the custom GetEntity, never declared in the
	// parsed DTD nor present in any document table.
	extDoc := helium.NewDefaultDocument()
	extDTD, err := extDoc.CreateInternalSubset("r", "", "")
	require.NoError(t, err)
	ext, err := extDTD.AddEntity("ext", enum.ExternalGeneralParsedEntity, "", "nul", "")
	require.NoError(t, err)

	// A pure SAX-event handler (NOT a TreeBuilder): no document tree is built, so
	// pctx.doc is absent and the document entity table is empty. Declared entities
	// are captured from the entity-declaration callback; the nested "ext" is
	// pre-seeded so only GetEntity can resolve it.
	h := sax.New()
	entities := map[string]*helium.Entity{"ext": ext}
	h.SetOnEntityDecl(sax.EntityDeclFunc(func(_ context.Context, name string, typ enum.EntityType, publicID, systemID, notation string) error {
		d := helium.NewDefaultDocument()
		dtd, derr := d.CreateInternalSubset("r", "", "")
		if derr != nil {
			return derr
		}
		e, derr := dtd.AddEntity(name, typ, publicID, systemID, notation)
		if derr != nil {
			return derr
		}
		entities[name] = e
		return nil
	}))
	h.SetOnGetEntity(sax.GetEntityFunc(func(_ context.Context, name string) (sax.Entity, error) {
		if e, ok := entities[name]; ok {
			return e, nil
		}
		return nil, nil //nolint:nilnil
	}))

	const doc = `<!DOCTYPE r [
<!ELEMENT r ANY>
<!ELEMENT e EMPTY>
<!ATTLIST e a CDATA #IMPLIED>
<!ENTITY outer "&ext;">
]>
<r>&outer;<e a="&outer;"/></r>`

	_, perr := helium.NewParser().SubstituteEntities(false).SAXHandler(h).
		Parse(t.Context(), []byte(doc))
	require.Error(t, perr, "an indirect external reference resolvable only via SAX GetEntity must violate the attribute-value WFC even after the entity is used in content")
	require.Contains(t, perr.Error(), "attribute references external entity")
}
