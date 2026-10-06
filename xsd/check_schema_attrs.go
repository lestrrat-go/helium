package xsd

import (
	"context"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

// checkSchemaElementAttrs walks every XSD-namespace element of a schema document
// and applies two attribute rules from the schema-for-schemas:
//
//   - Any attribute in the XML Schema namespace
//     (http://www.w3.org/2001/XMLSchema) is rejected. The normative
//     schema-for-schemas declares every schema element's complex type with
//     `<xs:anyAttribute namespace="##other"/>`, so the ONLY attributes permitted
//     are the explicitly-declared unqualified (no-namespace) ones plus foreign
//     attributes from a namespace OTHER than the XSD namespace. An attribute in
//     the XSD namespace itself (e.g. `xsd:type`, `xsd:targetNamespace`) is neither
//     a recognized schema attribute (those are unqualified) nor an admissible
//     foreign attribute. This rule is version-independent (1.0 and 1.1).
//   - In XSD 1.0 mode, an unqualified attribute that only the XSD 1.1
//     schema-for-schemas declares (xsd11OnlyAttr) is rejected: it is outside the
//     1.0 XML representation of its element (Part 1 §3).
func (c *compiler) checkSchemaElementAttrs(ctx context.Context, root *helium.Element) {
	if c.filename == "" {
		return
	}
	c.walkSchemaElementAttrs(ctx, root)
}

func (c *compiler) walkSchemaElementAttrs(ctx context.Context, elem *helium.Element) {
	if elem.URI() == lexicon.NamespaceXSD {
		local := elem.LocalName()
		for _, a := range elem.Attributes() {
			switch a.URI() {
			case lexicon.NamespaceXSD:
				c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), elem.Line(), local, local, a.LocalName(),
					"Attributes from the schema namespace ('"+lexicon.NamespaceXSD+"') are not allowed on schema components; only unqualified schema attributes and foreign-namespace attributes are permitted."))
			case "":
				if c.version != Version11 && xsd11OnlyAttr(local, a.LocalName()) {
					c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), local, local,
						"The attribute '"+a.LocalName()+"' is not allowed."))
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
		// is arbitrary application/human data, not schema components, so an
		// attribute embedded there is not governed by the schema-for-schemas.
		if ce.URI() == lexicon.NamespaceXSD && (ce.LocalName() == elemAppinfo || ce.LocalName() == elemDocumentation) {
			continue
		}
		c.walkSchemaElementAttrs(ctx, ce)
	}
}

// xsd11OnlyAttr reports whether attr is an unqualified attribute that the XSD
// 1.1 schema-for-schemas adds to elemLocal, an element that also exists in XSD
// 1.0. The set is the difference between the 1.0 and 1.1 schema-for-schemas
// (http://www.w3.org/2001/XMLSchema.xsd and
// http://www.w3.org/2009/XMLSchema/XMLSchema.xsd). xs:attribute's 1.1 additions
// (inheritable, targetNamespace) are absent here because checkAttrVocabulary
// already rejects them in 1.0.
func xsd11OnlyAttr(elemLocal, attr string) bool {
	switch elemLocal {
	case elemAny, elemAnyAttribute:
		return attr == attrNotNamespace || attr == attrNotQName
	case elemComplexType:
		return attr == attrDefaultAttrsApply
	case elemElement:
		return attr == attrTargetNamespace
	case elemSelector, elemField:
		return attr == attrXPathDefaultNamespace
	case elemKey, elemKeyRef, elemUnique:
		return attr == attrRef
	case elemSchema:
		return attr == attrDefaultAttributes || attr == attrXPathDefaultNamespace
	}
	return false
}
