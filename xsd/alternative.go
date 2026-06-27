package xsd

import (
	"context"
	"fmt"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
)

// altTypeRef is a deferred xs:alternative @type reference, resolved in
// resolveRefs once all named types are registered.
type altTypeRef struct {
	alt      *TypeAlternative
	qn       QName
	line     int
	source   string
	elemName string
}

// parseTypeAlternatives reads the XSD 1.1 <xs:alternative> children of an
// xs:element declaration (conditional type assignment), in document order. Each
// alternative carries an optional @test (absent on the final, default
// alternative) and a @type reference; the test is pre-compiled as XPath 3.1, and
// the @type reference is registered for deferred resolution in resolveRefs.
//
// Callers must only invoke this in XSD 1.1 mode.
func (c *compiler) parseTypeAlternatives(ctx context.Context, elem *helium.Element) []*TypeAlternative {
	var alts []*TypeAlternative
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok || !isXSDElement(ce, elemAlternative) {
			continue
		}
		alt := c.parseTypeAlternative(ctx, ce)
		if alt != nil {
			alts = append(alts, alt)
		}
	}
	return alts
}

// parseTypeAlternative reads a single <xs:alternative>. A missing @type or a
// malformed @test is a fatal schema error (returns nil); an inline anonymous
// type is not yet supported.
func (c *compiler) parseTypeAlternative(ctx context.Context, elem *helium.Element) *TypeAlternative {
	typeRef := getAttr(elem, attrType)
	if typeRef == "" {
		if c.filename != "" {
			c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), elemAlternative,
				"An inline type or a 'type' attribute is required on xs:alternative (inline types are not yet supported)."))
		}
		return nil
	}
	alt := &TypeAlternative{
		Namespaces: collectNSContext(elem),
		Line:       elem.Line(),
		Source:     c.diagSource(),
		TypeName:   c.resolveQName(ctx, elem, typeRef),
	}
	c.altTypeRefs = append(c.altTypeRefs, altTypeRef{
		alt:      alt,
		qn:       alt.TypeName,
		line:     elem.Line(),
		source:   c.diagSource(),
		elemName: elem.LocalName(),
	})

	// A testless alternative is the unconditional default; it carries no compiled
	// expression and always matches.
	if test := getAttr(elem, attrTest); test != "" {
		alt.Test = test
		compiled, err := xpath3.NewCompiler().Compile(test)
		if err != nil {
			if c.filename != "" {
				c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), elemAlternative,
					fmt.Sprintf("The XPath expression '%s' of the type alternative is not valid: %v.", test, err)))
			}
			return nil
		}
		if err := compiled.Validate(alt.Namespaces); err != nil {
			if c.filename != "" {
				c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), elemAlternative,
					fmt.Sprintf("The XPath expression '%s' of the type alternative has an unbound namespace prefix: %v.", test, err)))
			}
			return nil
		}
		alt.compiled = compiled
	}
	return alt
}

// resolveAltTypeRefs resolves each xs:alternative's @type reference against the
// registered named types, reporting a fatal schema error for an unresolved type.
// Run from resolveRefs after the named-type table is populated.
func (c *compiler) resolveAltTypeRefs(ctx context.Context) {
	for _, ref := range c.altTypeRefs {
		td, ok := c.schema.types[ref.qn]
		if !ok {
			if c.filename != "" {
				msg := fmt.Sprintf("The QName value '{%s}%s' does not resolve to a(n) type definition.", ref.qn.NS, ref.qn.Local)
				c.schemaError(ctx, schemaElemDeclErrorAttr(c.diagSourceOrRecorded(ref.source), ref.line, ref.elemName, attrType, msg))
			}
			td = &TypeDef{Name: ref.qn, ContentType: ContentTypeSimple}
		}
		ref.alt.Type = td
	}
}

// applyTypeAlternatives implements XSD 1.1 conditional type assignment for an
// element: when the declaration has a {type table} and no xsi:type is present,
// the governing type is the one chosen by the alternatives; otherwise declType is
// returned unchanged. xsi:type takes precedence over CTA, so this is a no-op when
// an xsi:type attribute is present.
func (vc *validationContext) applyTypeAlternatives(ctx context.Context, elem *helium.Element, edecl *ElementDecl, declType *TypeDef) *TypeDef {
	if vc.version != Version11 || edecl == nil || len(edecl.Alternatives) == 0 {
		return declType
	}
	if elementHasXsiType(elem) {
		return declType
	}
	for _, alt := range edecl.Alternatives {
		// A testless alternative is the unconditional default.
		if alt.compiled == nil {
			if alt.Test == "" && alt.Type != nil {
				return alt.Type
			}
			continue
		}
		ev := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Namespaces(alt.Namespaces)
		res, err := ev.Evaluate(ctx, alt.compiled, elem)
		if err != nil {
			// A failing alternative test does not select the type (treated as not
			// matched); evaluation continues to the next alternative.
			continue
		}
		ok, err := xpath3.EBV(res.Sequence())
		if err == nil && ok && alt.Type != nil {
			return alt.Type
		}
	}
	return declType
}

// elementHasXsiType reports whether the element carries an xsi:type attribute.
func elementHasXsiType(elem *helium.Element) bool {
	for _, a := range elem.Attributes() {
		if a.URI() == lexicon.NamespaceXSI && a.LocalName() == attrType {
			return true
		}
	}
	return false
}
