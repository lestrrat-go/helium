package xsd

import (
	"context"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

// checkIDConstraintPlacement enforces the schema-for-schemas placement and
// content-model rules for identity-constraint components that helium's
// element-declaration-scoped parser (parseIDConstraint) cannot see. Because
// parseIDConstraint only scans the children of xs:element declarations, an
// xs:key/xs:unique/xs:keyref declared anywhere else — or an xs:selector/xs:field
// with stray content — is silently dropped and the invalid schema would compile.
//
// Rules enforced (XSD Structures 3.11.1, the schema for schemas):
//
//   - xs:key / xs:unique / xs:keyref may appear ONLY as a child of an xs:element
//     declaration. A constraint placed at the top level, or inside xs:attribute /
//     xs:group / xs:attributeGroup / xs:complexType / xs:simpleType / a model
//     group / an xs:selector / an xs:field, is a schema error.
//   - xs:selector / xs:field may appear ONLY as a child of xs:key / xs:unique /
//     xs:keyref, with content model (annotation?) and only the {id?, xpath,
//     xpathDefaultNamespace?} attributes (any other unqualified attribute, e.g.
//     name, or any non-annotation child element is a schema error).
//
// These are version-independent structural constraints, so the walk runs in both
// XSD 1.0 and 1.1.
func (c *compiler) checkIDConstraintPlacement(ctx context.Context, root *helium.Element) {
	if c.filename == "" {
		return
	}
	c.walkIDConstraintPlacement(ctx, root, "")
}

// walkIDConstraintPlacement recurses the schema document, tracking the XSD local
// name of each element's parent so the placement of identity-constraint elements
// (and their selector/field children) can be checked against the parent kind.
func (c *compiler) walkIDConstraintPlacement(ctx context.Context, elem *helium.Element, parentLocal string) {
	if elem.URI() == lexicon.NamespaceXSD {
		switch elem.LocalName() {
		case elemUnique, elemKey, elemKeyRef:
			if parentLocal != elemElement {
				c.reportIDCPlacementError(ctx, elem,
					"An identity constraint ('"+elem.LocalName()+"') is only allowed as a child of an element declaration.")
			}
		case elemSelector, elemField:
			switch parentLocal {
			case elemUnique, elemKey, elemKeyRef:
				c.checkSelectorFieldContent(ctx, elem)
			default:
				c.reportIDCPlacementError(ctx, elem,
					"A '"+elem.LocalName()+"' is only allowed as a child of an identity constraint (key, keyref or unique).")
			}
		}
	}

	var childParent string
	if elem.URI() == lexicon.NamespaceXSD {
		childParent = elem.LocalName()
	}
	for child := range helium.Children(elem) {
		ce, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		// Do not descend into annotation payload: xs:appinfo/xs:documentation may
		// legitimately embed XSD-namespace elements (e.g. a documentation example),
		// which are application data, not schema components, and must not be
		// mistaken for misplaced identity-constraint declarations.
		if ce.URI() == lexicon.NamespaceXSD && (ce.LocalName() == elemAppinfo || ce.LocalName() == elemDocumentation) {
			continue
		}
		c.walkIDConstraintPlacement(ctx, ce, childParent)
	}
}

// checkSelectorFieldContent enforces the (annotation?) content model and the
// allowed-attribute set of a correctly-placed xs:selector/xs:field element.
func (c *compiler) checkSelectorFieldContent(ctx context.Context, elem *helium.Element) {
	local := elem.LocalName()

	// Only the unqualified attributes id, xpath and xpathDefaultNamespace are
	// permitted; foreign-namespaced (prefixed) attributes are allowed. An
	// unexpected unqualified attribute (e.g. name) is a schema error.
	for _, attr := range elem.Attributes() {
		if attr.Prefix() != "" {
			continue
		}
		switch attr.LocalName() {
		case "id", attrXPath, attrXPathDefaultNS:
		default:
			c.reportIDCPlacementError(ctx, elem,
				"The attribute '"+attr.LocalName()+"' is not allowed on '"+local+"'.")
		}
	}

	// Content model (annotation?): at most one xs:annotation, and no other
	// element children.
	annotationSeen := false
	strayChild := false
	for child := range helium.Children(elem) {
		ce, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if ce.URI() == lexicon.NamespaceXSD && ce.LocalName() == elemAnnotation {
			if annotationSeen {
				strayChild = true
			}
			annotationSeen = true
			continue
		}
		strayChild = true
	}
	if strayChild {
		c.reportIDCPlacementError(ctx, elem,
			"The content of '"+local+"' is not valid. Expected is (annotation?).")
	}
}

// reportIDCPlacementError reports an identity-constraint placement/content-model
// violation as a fatal schema compilation error, cited against the declaring
// document (diagSource) so a violation in an included/redefined document is
// attributed to that document, matching elem.Line().
func (c *compiler) reportIDCPlacementError(ctx context.Context, elem *helium.Element, msg string) {
	src := c.diagSource()
	if src == "" {
		return
	}
	local := elem.LocalName()
	c.schemaError(ctx, schemaParserError(src, elem.Line(), local, local, msg))
}
