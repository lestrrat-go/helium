package xsd

import (
	"context"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

// checkSchemaNamespaceAttrs rejects any attribute in the XML Schema namespace
// (http://www.w3.org/2001/XMLSchema) appearing on an XSD-namespace schema
// element. The normative schema-for-schemas declares every schema element's
// complex type with `<xs:anyAttribute namespace="##other"/>`, so the ONLY
// attributes permitted are the explicitly-declared unqualified (no-namespace)
// ones plus foreign attributes from a namespace OTHER than the XSD namespace.
// An attribute in the XSD namespace itself (e.g. `xsd:type`, `xsd:targetNamespace`)
// is neither a recognized schema attribute (those are unqualified) nor an
// admissible foreign attribute, so it is a representation error.
//
// This is a version-independent XSD rule, enforced in both 1.0 and 1.1.
func (c *compiler) checkSchemaNamespaceAttrs(ctx context.Context, root *helium.Element) {
	if c.filename == "" {
		return
	}
	c.walkSchemaNamespaceAttrs(ctx, root)
}

func (c *compiler) walkSchemaNamespaceAttrs(ctx context.Context, elem *helium.Element) {
	if elem.URI() == lexicon.NamespaceXSD {
		for _, a := range elem.Attributes() {
			if a.URI() != lexicon.NamespaceXSD {
				continue
			}
			file := c.diagSource()
			c.schemaError(ctx, schemaParserErrorAttr(file, elem.Line(), elem.LocalName(), elem.LocalName(), a.LocalName(),
				"Attributes from the schema namespace ('"+lexicon.NamespaceXSD+"') are not allowed on schema components; only unqualified schema attributes and foreign-namespace attributes are permitted."))
		}
	}

	for child := range helium.Children(elem) {
		ce, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		// Do NOT descend into xs:appinfo / xs:documentation payload: their content
		// is arbitrary application/human data, not schema components, so an
		// XSD-namespace attribute embedded there is not governed by the
		// schema-for-schemas.
		if ce.URI() == lexicon.NamespaceXSD && (ce.LocalName() == elemAppinfo || ce.LocalName() == elemDocumentation) {
			continue
		}
		c.walkSchemaNamespaceAttrs(ctx, ce)
	}
}
