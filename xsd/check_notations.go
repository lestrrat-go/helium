package xsd

import (
	"context"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xmlchar"
)

// checkNotations enforces the schema-for-schemas structural rules for
// <xs:notation> declarations (XSD Structures §3.14) that helium's declaration
// collector (parseSchemaChildren) does not verify — it only records the @name of
// each notation that is a direct child of xs:schema and ignores everything else,
// so a misplaced notation, one with a bad/missing name, one lacking both
// identifiers, one with disallowed content, or a duplicate name would otherwise
// compile clean.
//
// Rules enforced (the schema for schemas, §3.14.2):
//
//   - an xs:notation is valid ONLY as a child of xs:schema;
//   - it must carry a @name that is a valid xs:NCName (after whitespace collapse);
//   - it must carry at least one of @public / @system (either may be empty);
//   - its content model is (annotation?): at most one xs:annotation child, no
//     other element children, and no non-whitespace character content;
//   - its @name must be unique among the notation declarations of the document.
//
// These are version-independent structural constraints, so the walk runs in both
// XSD 1.0 and 1.1. It mirrors checkSchemaComponentIDs / checkIDConstraintPlacement
// and is invoked on the entry document and every included / imported / redefined
// / overridden document, each with a fresh per-document seen set.
func (c *compiler) checkNotations(ctx context.Context, root *helium.Element) {
	if c.filename == "" {
		return
	}
	seen := make(map[string]struct{})
	c.walkNotations(ctx, root, "", seen)
}

// walkNotations recurses the schema document, tracking the XSD local name of each
// element's parent so a notation's placement can be checked against the parent
// kind. It does not descend into annotation payload (xs:appinfo / xs:documentation),
// which may legitimately embed XSD-namespace elements that are application data,
// not schema components.
func (c *compiler) walkNotations(ctx context.Context, elem *helium.Element, parentLocal string, seen map[string]struct{}) {
	if elem.URI() == lexicon.NamespaceXSD && elem.LocalName() == elemNotation {
		c.checkNotationDecl(ctx, elem, parentLocal, seen)
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
		if ce.URI() == lexicon.NamespaceXSD && (ce.LocalName() == elemAppinfo || ce.LocalName() == elemDocumentation) {
			continue
		}
		c.walkNotations(ctx, ce, childParent, seen)
	}
}

// checkNotationDecl validates a single xs:notation element against the §3.14.2
// structural rules. A misplaced notation is reported and the remaining checks are
// skipped (its @name never enters the uniqueness set, since only a top-level
// notation is a real declaration).
func (c *compiler) checkNotationDecl(ctx context.Context, elem *helium.Element, parentLocal string, seen map[string]struct{}) {
	src := c.diagSource()
	if src == "" {
		return
	}
	line := elem.Line()

	// Placement: xs:notation is a top-level schema declaration only.
	if parentLocal != elemSchema {
		c.schemaError(ctx, schemaParserError(src, line, elemNotation, elemNotation,
			"A notation declaration is only allowed as a child of xs:schema."))
		return
	}

	// Only the unqualified attributes id, name, public and system are permitted;
	// foreign-namespaced attributes are allowed, but an attribute qualified with
	// the XSD namespace, or an unexpected unqualified attribute, is a schema error.
	for _, attr := range elem.Attributes() {
		switch attr.URI() {
		case "":
			switch attr.LocalName() {
			case "id", attrName, attrPublic, attrSystem:
			default:
				c.schemaError(ctx, schemaParserError(src, line, elemNotation, elemNotation,
					"The attribute '"+attr.LocalName()+"' is not allowed."))
			}
		case lexicon.NamespaceXSD:
			c.schemaError(ctx, schemaParserError(src, line, elemNotation, elemNotation,
				"The attribute '"+attr.LocalName()+"' is not allowed."))
		}
	}

	// @name must be present and a valid NCName (xs:NCName has whiteSpace=collapse).
	nameOK := false
	if hasAttr(elem, attrName) {
		name := normalizeWhiteSpace(getAttr(elem, attrName), "collapse")
		if xmlchar.IsValidNCName(name) {
			nameOK = true
			if _, dup := seen[name]; dup {
				c.schemaError(ctx, schemaParserError(src, line, elemNotation, elemNotation,
					"A notation named '"+name+"' is already declared."))
			} else {
				seen[name] = struct{}{}
			}
		}
	}
	if !nameOK {
		c.schemaError(ctx, schemaParserError(src, line, elemNotation, elemNotation,
			"A notation declaration must have a 'name' attribute that is a valid NCName."))
	}

	// At least one of @public / @system must be present (either may be empty).
	if !hasAttr(elem, attrPublic) && !hasAttr(elem, attrSystem) {
		c.schemaError(ctx, schemaParserError(src, line, elemNotation, elemNotation,
			"A notation declaration must have at least one of the 'public' and 'system' attributes."))
	}

	c.checkNotationContent(ctx, elem, src)
}

// checkNotationContent enforces the (annotation?) content model of an xs:notation:
// at most one xs:annotation child, no other element children, and no
// non-whitespace character content.
func (c *compiler) checkNotationContent(ctx context.Context, elem *helium.Element, src string) {
	annotationSeen := false
	invalid := false
	for child := range helium.Children(elem) {
		switch child.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			if !xmlchar.IsAllSpace(child.Content()) {
				invalid = true
			}
		case helium.ElementNode:
			ce, ok := child.(*helium.Element)
			if !ok {
				continue
			}
			if ce.URI() == lexicon.NamespaceXSD && ce.LocalName() == elemAnnotation {
				if annotationSeen {
					invalid = true
				}
				annotationSeen = true
				continue
			}
			invalid = true
		}
	}
	if invalid {
		c.schemaError(ctx, schemaParserError(src, elem.Line(), elemNotation, elemNotation,
			"The content is not valid. Expected is (annotation?)."))
	}
}
