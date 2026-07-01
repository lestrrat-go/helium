package xsd

import (
	"context"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xmlchar"
)

// checkSchemaComponentIDs enforces the xs:ID typing of the `id` attribute that
// every XSD schema component element may carry (xsd.org §3.*: the {id} property
// is of type xs:ID). Within a single schema document each such id must be a
// valid xs:ID — i.e. a valid NCName after whitespace collapse (xs:ID derives
// from xs:token, whose whitespace facet is "collapse") — and must be unique.
//
// Gated to Version11: enforcing it under XSD 1.0 could change the compile
// outcome of an existing 1.0 schema that carries a duplicate/invalid id which
// libxml2 tolerated, breaking the byte-identical 1.0 golden contract. The
// constraint itself is not 1.1-specific, but the opt-in version toggle keeps
// 1.0 behavior frozen.
func (c *compiler) checkSchemaComponentIDs(ctx context.Context, root *helium.Element) {
	// Self-guard the Version11 gating (every caller already gates too) so the
	// "Gated to Version11" contract is self-contained: enforcing @id uniqueness
	// under XSD 1.0 would change the compile outcome of a 1.0 schema libxml2
	// tolerated and break the byte-identical 1.0 golden contract. Mirrors the
	// adjacent readDefaultOpenContent, which self-guards the same way.
	if c.version != Version11 {
		return
	}
	if c.filename == "" {
		return
	}
	seen := make(map[string]struct{})
	c.walkComponentIDs(ctx, root, seen)
}

func (c *compiler) walkComponentIDs(ctx context.Context, elem *helium.Element, seen map[string]struct{}) {
	// The `id` attribute is an unqualified attribute on an XSD-namespace
	// element. Foreign-namespace elements (e.g. inside xs:appinfo) are not
	// schema components, so their `id` attributes are not xs:ID and are ignored.
	if elem.URI() == lexicon.NamespaceXSD {
		if a, ok := elem.FindAttribute(helium.NSPredicate{Local: "id", NamespaceURI: ""}); ok {
			id := normalizeWhiteSpace(a.Value(), "collapse")
			// Attribute the diagnostic to the document that actually declares this
			// component: for an xs:include/xs:redefine/xs:override the walked root is
			// the nested document (whose line numbers elem.Line() carries), so
			// c.filename (the including schema) would mis-cite the file. c.diagSource()
			// returns c.includeFile when a nested document is active, else c.filename.
			file := c.diagSource()
			switch {
			case !xmlchar.IsValidNCName(id):
				c.schemaError(ctx, schemaParserErrorAttr(file, elem.Line(), elem.LocalName(), elem.LocalName(), "id",
					"The value '"+a.Value()+"' of attribute 'id' is not a valid 'xs:ID'."))
			default:
				if _, dup := seen[id]; dup {
					c.schemaError(ctx, schemaParserErrorAttr(file, elem.Line(), elem.LocalName(), elem.LocalName(), "id",
						"The value '"+id+"' of attribute 'id' is not unique within the schema document."))
				} else {
					seen[id] = struct{}{}
				}
			}
		}
	}

	for child := range helium.Children(elem) {
		ce, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		// Do NOT descend into xs:appinfo / xs:documentation payload: their content
		// is arbitrary application/human data, NOT schema components, so an `id`
		// attribute on an element embedded there is not a schema-component xs:ID
		// and must not be collected (an xs:appinfo MAY itself contain XSD-namespace
		// elements). The xs:annotation element's OWN @id was already collected
		// above before this descent; only its annotation-payload children are
		// skipped here.
		if ce.URI() == lexicon.NamespaceXSD && (ce.LocalName() == elemAppinfo || ce.LocalName() == elemDocumentation) {
			continue
		}
		c.walkComponentIDs(ctx, ce, seen)
	}
}
