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

// parseTypeAlternative reads a single <xs:alternative>. The governing type is
// supplied either by the @type attribute (a named reference resolved in
// resolveRefs) or by an inline anonymous <xs:complexType>/<xs:simpleType> child.
// A missing type (neither @type nor inline) or a malformed @test is a fatal
// schema error (returns nil).
func (c *compiler) parseTypeAlternative(ctx context.Context, elem *helium.Element) *TypeAlternative {
	alt := &TypeAlternative{
		Namespaces: collectNSContext(elem),
		Line:       elem.Line(),
		Source:     c.diagSource(),
		// fn:static-base-uri() exposes the SCHEMA document URI, never the diagnostic
		// label/source (which may be a caller-supplied error-message string).
		BaseURI: c.schemaBaseURI,
	}

	// XSD 1.1 requires EXACTLY ONE governing-type source: either a @type attribute
	// or a single inline <xs:complexType>/<xs:simpleType> child, never both and never
	// two inline types.
	typeRef := getAttr(elem, attrType)
	inlineCount := countInlineAlternativeTypes(elem)
	switch {
	case typeRef != "" && inlineCount > 0:
		if c.filename != "" {
			c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), elemAlternative,
				"An xs:alternative must not have both a 'type' attribute and an inline type definition."))
		}
		return nil
	case inlineCount > 1:
		if c.filename != "" {
			c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), elemAlternative,
				"An xs:alternative must not have more than one inline type definition."))
		}
		return nil
	}

	if typeRef != "" {
		alt.TypeName = c.resolveQName(ctx, elem, typeRef)
		c.altTypeRefs = append(c.altTypeRefs, altTypeRef{
			alt:      alt,
			qn:       alt.TypeName,
			line:     elem.Line(),
			source:   c.diagSource(),
			elemName: elem.LocalName(),
		})
	} else {
		// No @type: an inline anonymous <xs:complexType> or <xs:simpleType> child
		// supplies the governing type. It is compiled through the same path as an
		// inline type on an element/attribute declaration, so any refs it carries
		// (e.g. an extension @base) resolve later in resolveRefs.
		td, found, err := c.parseInlineAlternativeType(ctx, elem)
		switch {
		case err != nil:
			if c.filename != "" {
				c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), elemAlternative,
					err.Error()))
			}
			return nil
		case !found:
			if c.filename != "" {
				c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), elemAlternative,
					"An inline type or a 'type' attribute is required on xs:alternative."))
			}
			return nil
		}
		alt.Type = td
	}

	// XSD 1.1 xpathDefaultNamespace: the effective value (local on the alternative,
	// else inherited from the <xs:schema> element) supplies the XPath default
	// element namespace, exposed to xpath3 as the "" (default-namespace) binding so
	// an unprefixed name test in @test matches that namespace.
	if xdn, ok := c.effectiveXPathDefaultNS(elem); ok {
		alt.Namespaces[""] = xdn
	}

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

// countInlineAlternativeTypes returns the number of inline anonymous
// <xs:complexType>/<xs:simpleType> children of an <xs:alternative>, so the caller
// can enforce the exactly-one-governing-type-source rule.
func countInlineAlternativeTypes(elem *helium.Element) int {
	n := 0
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if isXSDElement(ce, elemComplexType) || isXSDElement(ce, elemSimpleType) {
			n++
		}
	}
	return n
}

// parseInlineAlternativeType compiles the inline anonymous <xs:complexType> or
// <xs:simpleType> child of an <xs:alternative>. found reports whether such a child
// was present; when found is false the caller reports the missing-type error. A
// malformed inline type returns a non-nil error.
func (c *compiler) parseInlineAlternativeType(ctx context.Context, elem *helium.Element) (*TypeDef, bool, error) {
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXSDElement(ce, elemComplexType):
			td, err := c.parseComplexType(ctx, ce)
			return td, true, err
		case isXSDElement(ce, elemSimpleType):
			td, err := c.parseSimpleType(ctx, ce)
			return td, true, err
		}
	}
	return nil, false, nil
}

// effectiveXPathDefaultNS resolves the XSD 1.1 xpathDefaultNamespace in effect for
// an XPath-bearing schema element (here an xs:alternative): the value declared on
// the element itself wins, otherwise the schema-level value is inherited. The
// returned string is the resolved namespace URI (empty for ##local); ok is false
// when no xpathDefaultNamespace is in effect, in which case the caller leaves the
// default-element-namespace binding untouched. The ##targetNamespace/
// ##defaultNamespace/##local keywords are resolved against the element's context.
func (c *compiler) effectiveXPathDefaultNS(elem *helium.Element) (string, bool) {
	raw := ""
	switch {
	case hasAttr(elem, attrXPathDefaultNamespace):
		raw = getAttr(elem, attrXPathDefaultNamespace)
	case c.xpathDefaultNSSet:
		raw = c.xpathDefaultNS
	default:
		return "", false
	}
	// xpathDefaultNamespace is whitespace-collapse, so the ##keyword forms (and a
	// literal URI) must be matched after collapsing surrounding/internal whitespace.
	raw = normalizeWhiteSpace(raw, "collapse")
	switch raw {
	case xpathDefaultNSTargetNamespace:
		return c.schema.targetNamespace, true
	case xpathDefaultNSLocal:
		return "", true
	case xpathDefaultNSDefaultNamespace:
		// The in-scope default namespace (xmlns="…") at the alternative element.
		return collectNSContext(elem)[""], true
	default:
		return raw, true
	}
}

// isErrorType reports whether td is the XSD 1.1 built-in xs:error type, whose
// value space and lexical space are empty so any element it governs is invalid.
func isErrorType(td *TypeDef) bool {
	return td != nil && td.Name.NS == lexicon.NamespaceXSD && td.Name.Local == lexicon.TypeError
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
	if vc.version != Version11 || edecl == nil {
		return declType
	}
	// Conditional type assignment is a property of the referenced GLOBAL element
	// declaration, so an <xs:element ref="g"> particle does not carry the type
	// table (its ElementDecl is a ref, like IDCs). Fall back to the global
	// declaration's alternatives, mirroring idcHostDecl's ref handling.
	alts := edecl.Alternatives
	if len(alts) == 0 && edecl.IsRef {
		if g, ok := vc.schema.LookupElement(edecl.Name.Local, edecl.Name.NS); ok && g != edecl {
			alts = g.Alternatives
		}
	}
	if len(alts) == 0 {
		return declType
	}
	if elementHasXsiType(elem) {
		return declType
	}
	// The XPath @test runs against the XSD 1.1 CTA context node: a detached copy of
	// the element with its own + inherited attributes and namespaces but no children
	// or parent (§3.13). Built once per element and shared by every alternative.
	var cta *helium.Element
	for _, alt := range alts {
		// A testless alternative is the unconditional default.
		if alt.compiled == nil {
			if alt.Test == "" && alt.Type != nil {
				return alt.Type
			}
			continue
		}
		if cta == nil {
			cta = vc.ctaContextNode(elem)
		}
		ev := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Namespaces(alt.Namespaces).
			Position(1).
			Size(1).
			CollectionResolver(emptyCollectionResolver{})
		if alt.BaseURI != "" {
			ev = ev.BaseURI(alt.BaseURI)
		}
		res, err := ev.Evaluate(ctx, alt.compiled, cta)
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

// checkAltSubstitutability enforces the XSD 1.1 constraint that each alternative's
// {type definition} be validly substitutable for the element's declared type
// (cvc-alt / "Type Alternative"). It runs after resolveRefs so both the element's
// declared type and every alternative type (named or inline) are linked.
func (c *compiler) checkAltSubstitutability(ctx context.Context) {
	if c.filename == "" || c.version != Version11 {
		return
	}
	for _, edecl := range c.ctaElems {
		// A substitution-group member with no explicit type has edecl.Type == nil but
		// inherits the head's type; check against the EFFECTIVE declared type so a
		// simple alternative cannot bypass the simple-vs-complex rejection via a nil
		// declared type.
		declType := effectiveDeclType(edecl, c.schema)
		if declType == nil {
			continue
		}
		for _, alt := range edecl.Alternatives {
			if alt.Type == nil {
				continue
			}
			if isValidlySubstitutable(alt.Type, declType) {
				continue
			}
			msg := fmt.Sprintf("The type alternative's type '{%s}%s' is not validly substitutable for the declared type '{%s}%s' of the element declaration.",
				alt.Type.Name.NS, alt.Type.Name.Local, declType.Name.NS, declType.Name.Local)
			c.schemaError(ctx, schemaElemDeclErrorAttr(c.diagSourceOrRecorded(alt.Source), alt.Line, elemAlternative, attrType, msg))
		}
	}
}

// isValidlySubstitutable reports whether a type alternative's type alt may govern
// an element whose declared type is decl. xs:error is always permitted (it makes
// the element invalid by design). Otherwise alt must be validly derived from decl
// (Type Derivation OK), which — for a union declared type — includes alt being
// derived from one of the union's member types.
//
// The check is deliberately under-strict for SIMPLE alternative types against a
// SIMPLE-or-union declared type: the XSD built-in simple-type hierarchy (e.g.
// xs:nonNegativeInteger ⊂ xs:integer) is not linked via BaseType pointers, so
// isDerivedFrom cannot confirm those legitimate derivations. Rather than risk
// false-rejecting a valid schema, an unconfirmed simple-vs-simple derivation is
// accepted. A simple alternative against a COMPLEX declared type (or an unrelated
// complex alternative type) is firmly rejected — a simple type can never be derived
// from a complex one, so that is always a real violation, not a hierarchy-linking
// gap.
func isValidlySubstitutable(alt, decl *TypeDef) bool {
	if alt == nil || decl == nil {
		return true
	}
	if isErrorType(alt) {
		return true
	}
	if isDerivedFrom(alt, decl) {
		return true
	}
	if decl.Variety == TypeVarietyUnion {
		for _, m := range decl.MemberTypes {
			if isValidlySubstitutable(alt, m) {
				return true
			}
		}
	}
	// Only fall back to the under-strict acceptance when BOTH types are simple
	// (or the declared type is a union): a simple alternative against a complex
	// declared type is a definite violation.
	declSimple := decl.ContentType == ContentTypeSimple || decl.Variety == TypeVarietyUnion
	return alt.ContentType == ContentTypeSimple && declSimple
}
