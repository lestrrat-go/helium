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
			switch {
			case !xmlchar.IsValidNCName(id):
				c.schemaError(ctx, schemaParserErrorAttr(c.filename, elem.Line(), elem.LocalName(), elem.LocalName(), "id",
					"The value '"+a.Value()+"' of attribute 'id' is not a valid 'xs:ID'."))
			default:
				if _, dup := seen[id]; dup {
					c.schemaError(ctx, schemaParserErrorAttr(c.filename, elem.Line(), elem.LocalName(), elem.LocalName(), "id",
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
		c.walkComponentIDs(ctx, ce, seen)
	}
}
