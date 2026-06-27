package xsd

import (
	"context"
	"fmt"

	helium "github.com/lestrrat-go/helium"
)

// parseOpenContent reads an XSD 1.1 <xs:openContent> element. mode defaults to
// "interleave"; "suffix" restricts open elements to a trailing position; "none"
// disables open content and returns nil. The wildcard is taken from the child
// <xs:any>. Callers must only invoke this in XSD 1.1 mode.
func (c *compiler) parseOpenContent(ctx context.Context, elem *helium.Element) *OpenContent {
	mode := OpenContentInterleave
	switch getAttr(elem, attrMode) {
	case "", "interleave":
		mode = OpenContentInterleave
	case "suffix":
		mode = OpenContentSuffix
	case "none":
		// Explicitly no open content (used to override a default open content).
		return nil
	default:
		if c.filename != "" {
			c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), elem.Line(), elem.LocalName(), elemOpenContent, attrMode,
				"The value of 'mode' must be one of 'interleave', 'suffix', or 'none'."))
		}
	}

	var wc *Wildcard
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if isXSDElement(ce, elemAny) {
			wc = c.readWildcard(ctx, ce)
			break
		}
	}
	if wc == nil {
		// An xs:openContent with mode != none requires an xs:any wildcard.
		if c.filename != "" {
			c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), elemOpenContent,
				"An 'openContent' with mode other than 'none' must contain an 'any' wildcard."))
		}
		return nil
	}
	return &OpenContent{Mode: mode, Wildcard: wc}
}

// collectModelElementNames returns the set of element expanded names declared
// anywhere in a content model (recursing through nested model groups, including
// substitution-group members). It backs the open-content "interleave" rule:
// weak wildcards never claim an element whose name is declared in the model, so
// such elements must go through the normal content-model match.
func collectModelElementNames(mg *ModelGroup, schema *Schema) map[QName]bool {
	names := make(map[QName]bool)
	var walk func(g *ModelGroup)
	walk = func(g *ModelGroup) {
		if g == nil {
			return
		}
		for _, p := range g.Particles {
			switch term := p.Term.(type) {
			case *ElementDecl:
				names[term.Name] = true
				for _, m := range schema.substGroups[term.Name] {
					names[m.Name] = true
				}
			case *ModelGroup:
				walk(term)
			}
		}
	}
	walk(mg)
	return names
}

// validateContentModelOpen validates an element's children against a content
// model carrying XSD 1.1 open content.
//
//   - suffix: the declared content is matched from the start; every remaining
//     trailing child must match the open wildcard.
//   - interleave: children whose expanded name is NOT declared in the model and
//     which match the open wildcard are removed (they are the open content);
//     the rest must satisfy the declared model. An element whose name IS declared
//     always goes through the model (weak-wildcard precedence), so a misplaced or
//     excess declared element is still a violation rather than open content.
func (vc *validationContext) validateContentModelOpen(ctx context.Context, elem *helium.Element, mg *ModelGroup, oc *OpenContent) error {
	children := collectChildElements(elem)

	if oc.Mode == OpenContentSuffix {
		consumed, err := vc.matchContentModel(ctx, elem, mg, children)
		if err != nil {
			return err
		}
		leftover := children[consumed:]
		// A trailing child whose name is declared in the model is a misplaced
		// declared element, not open content (weak-wildcard precedence): the model
		// already had its chance to consume it, so it is unexpected.
		declaredNames := collectModelElementNames(mg, vc.schema)
		for _, ch := range leftover {
			if declaredNames[QName{Local: ch.name, NS: ch.ns}] {
				vc.reportValidityError(ctx, vc.filename, ch.elem.Line(), ch.displayName, "This element is not expected.")
				return fmt.Errorf("unexpected element")
			}
		}
		return vc.validateOpenChildren(ctx, elem, oc.Wildcard, leftover)
	}

	// interleave
	declaredNames := collectModelElementNames(mg, vc.schema)
	var declared, open []childElem
	for _, ch := range children {
		qn := QName{Local: ch.name, NS: ch.ns}
		if !declaredNames[qn] && wildcardMatches(oc.Wildcard, ch.ns) {
			open = append(open, ch)
			continue
		}
		declared = append(declared, ch)
	}
	if err := vc.validateContentModelTop(ctx, elem, mg, declared); err != nil {
		return err
	}
	return vc.validateOpenChildren(ctx, elem, oc.Wildcard, open)
}

// validateOpenChildren validates a set of open-content child elements against the
// open wildcard (processContents lax/strict/skip). Any child that does not match
// the wildcard's namespace constraint is reported as unexpected.
func (vc *validationContext) validateOpenChildren(ctx context.Context, parent *helium.Element, wc *Wildcard, open []childElem) error {
	if len(open) == 0 {
		return nil
	}
	p := &Particle{MinOccurs: 0, MaxOccurs: Unbounded, Term: wc}
	consumed, err := vc.matchWildcardParticle(ctx, parent, p, wc, open, 0)
	if err != nil {
		return err
	}
	if consumed < len(open) {
		ce := open[consumed]
		vc.reportValidityError(ctx, vc.filename, ce.elem.Line(), ce.displayName, "This element is not expected.")
		return fmt.Errorf("unexpected element")
	}
	return nil
}
