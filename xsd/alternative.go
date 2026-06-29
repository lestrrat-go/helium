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
	// chameleonEligible is true when the lexical @type ref was unprefixed with no
	// in-scope default namespace, so resolveAltTypeRefs may retry the no-namespace
	// ({}) key — mirroring the ordinary element @type chameleon/no-TNS fallback.
	chameleonEligible bool
}

// parseTypeAlternatives reads the XSD 1.1 <xs:alternative> children of an
// xs:element declaration (conditional type assignment), in document order. Each
// alternative carries an optional @test (absent on the final, default
// alternative) and a @type reference; the test is pre-compiled as XPath 3.1, and
// the @type reference is registered for deferred resolution in resolveRefs.
//
// Callers must only invoke this in XSD 1.1 mode.
func (c *compiler) parseTypeAlternatives(ctx context.Context, elem *helium.Element) []*TypeAlternative {
	// Gather the xs:alternative child elements in document order first, so the
	// testless-default ordering constraint can be checked positionally (only the
	// LAST alternative may omit @test).
	var altElems []*helium.Element
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok || !isXSDElement(ce, elemAlternative) {
			continue
		}
		altElems = append(altElems, ce)
	}

	var alts []*TypeAlternative
	for i, ce := range altElems {
		// A testless alternative (no @test) is the unconditional default and must be
		// LAST in document order; an earlier testless alternative would render every
		// following alternative unreachable, which is a schema error (cta9001err).
		if !hasAttr(ce, attrTest) && i != len(altElems)-1 {
			if c.filename != "" {
				c.schemaError(ctx, schemaParserError(c.diagSource(), ce.Line(), ce.LocalName(), elemAlternative,
					"Only the last xs:alternative may omit the 'test' attribute."))
			}
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
	// The XSD 1.1 default xpath-default-namespace is ##local (no default element
	// namespace), so the schema document's XML default namespace (xmlns="…") must
	// NOT seed the "" binding — only an effective xpathDefaultNamespace may. Drop it
	// here; effectiveXPathDefaultNS below re-adds "" when xpathDefaultNamespace applies.
	nsCtx := collectNSContext(elem)
	delete(nsCtx, "")
	alt := &TypeAlternative{
		Namespaces: nsCtx,
		Line:       elem.Line(),
		Source:     c.diagSource(),
		// fn:static-base-uri() exposes the SCHEMA document URI, never the diagnostic
		// label/source (which may be a caller-supplied error-message string).
		BaseURI: c.schemaBaseURI,
	}

	// XSD 1.1 requires EXACTLY ONE governing-type source: either a @type attribute
	// or a single inline <xs:complexType>/<xs:simpleType> child, never both and never
	// two inline types. Presence is tested with hasAttr so a present-but-empty
	// @type="" is treated as a (malformed) type source, not as "inline only".
	hasType := hasAttr(elem, attrType)
	typeRef := getAttr(elem, attrType)
	inlineCount := countInlineAlternativeTypes(elem)
	switch {
	case hasType && inlineCount > 0:
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
	case hasType && normalizeWhiteSpace(typeRef, "collapse") == "":
		if c.filename != "" {
			c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), elem.Line(), elem.LocalName(), elemAlternative, attrType,
				"The value '"+typeRef+"' is not a valid QName."))
		}
		return nil
	}

	if hasType {
		// xs:QName uses whitespace collapse, so resolve the COLLAPSED lexical value
		// (a valid type=" xs:int " must not be rejected). Use it for the resolved
		// QName, the deferred ref, the chameleon-eligibility test, and diagnostics.
		typeRef = normalizeWhiteSpace(typeRef, "collapse")
		// Validate the lexical QName BEFORE resolveQName, mirroring idc's @ref/@refer
		// handling: a malformed value like ":T" (leading colon) is not a valid QName
		// and must be a fatal schema error, not silently resolved to {}T.
		if err := validateQName(typeRef); err != nil {
			if c.filename != "" {
				c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), elem.Line(), elem.LocalName(), elemAlternative, attrType,
					"The value '"+typeRef+"' is not a valid QName."))
			}
			return nil
		}
		alt.TypeName = c.resolveQName(ctx, elem, typeRef)
		c.altTypeRefs = append(c.altTypeRefs, altTypeRef{
			alt:               alt,
			qn:                alt.TypeName,
			line:              elem.Line(),
			source:            c.diagSource(),
			elemName:          elem.LocalName(),
			chameleonEligible: refChameleonEligible(elem, typeRef),
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

	// A testless alternative (no @test attribute at all) is the unconditional
	// default; it carries no compiled expression and always matches. Presence is
	// tested with hasAttr so a present-but-empty test="" is compiled (and fails) as
	// an invalid XPath rather than being silently treated as the default.
	if hasAttr(elem, attrTest) {
		test := getAttr(elem, attrTest)
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
		// The CTA @test runs in a static context with NO in-scope variables and whose
		// in-scope schema types are the built-in (xs:) types only; a free variable
		// reference (cta9002err) or a user-defined type named in cast/castable/instance
		// of/treat as (cta9003err / s3_12ii06) is a schema error.
		if err := c.checkCTATestStaticContext(elem, compiled, alt.Namespaces); err != nil {
			if c.filename != "" {
				c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), elemAlternative,
					fmt.Sprintf("The XPath expression '%s' of the type alternative is not valid: %v.", test, err)))
			}
			return nil
		}
		alt.compiled = compiled
	}
	return alt
}

// checkCTATestStaticContext enforces the XSD 1.1 static-context restrictions on an
// xs:alternative @test XPath beyond namespace-prefix validity: the expression may
// reference no variables (CTA exposes none) and may name only built-in (xs:) atomic
// types in cast/castable/instance of/treat as. A user-defined type reference or a
// free variable returns a non-nil error. The type-name namespace is resolved
// against the alternative's in-scope namespaces (unprefixed → the default element
// namespace binding, "").
func (c *compiler) checkCTATestStaticContext(_ *helium.Element, compiled *xpath3.Expression, namespaces map[string]string) error {
	refs := compiled.StaticReferences()
	if len(refs.FreeVariables) > 0 {
		return fmt.Errorf("undefined variable $%s", refs.FreeVariables[0])
	}
	for _, tn := range refs.TypeNames {
		ns := namespaces[tn.Prefix]
		if ns == lexicon.NamespaceXSD {
			continue
		}
		ref := tn.Name
		if tn.Prefix != "" {
			ref = tn.Prefix + ":" + tn.Name
		}
		return fmt.Errorf("the type %q is not a built-in type and is not available in a type alternative", ref)
	}
	return nil
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
	// A locally-present xpathDefaultNamespace on the alternative wins and is resolved
	// against the alternative's OWN namespace context (so a local ##defaultNamespace
	// uses the alternative's in-scope default namespace). resolveXPathDefaultNSToken
	// whitespace-collapses the raw value (xs:anyURI) before matching the ##keywords.
	if hasAttr(elem, attrXPathDefaultNamespace) {
		return resolveXPathDefaultNSToken(elem, getAttr(elem, attrXPathDefaultNamespace), c.schema.targetNamespace), true
	}
	// Otherwise inherit the schema-level token, but resolve it against the HOST
	// element (the alternative), NOT the schema root. Per the XSD 1.1 {xpath default
	// namespace} mapping the ##defaultNamespace keyword always uses the in-scope
	// default namespace of the element bearing the XPath, even when the
	// xpathDefaultNamespace attribute is inherited from <schema> — so an
	// alternative's own xmlns redeclaration governs (cta0005). The pre-resolved
	// root value (schemaXPathDefaultNS) is therefore NOT used here; only the raw
	// token is inherited and re-resolved at elem. ##targetNamespace/##local/literal
	// URI tokens resolve identically at either element, so this is a strict
	// refinement of the ##defaultNamespace case.
	if c.xpathDefaultNSSet {
		return resolveXPathDefaultNSToken(elem, c.schemaXPathDefaultNSToken, c.schema.targetNamespace), true
	}
	return "", false
}

// elementAlternatives returns the effective {type table} alternatives for an
// element declaration: its own, or — for an <xs:element ref="g"> particle that
// does not carry the table — the referenced global declaration's. Mirrors
// effectiveAlternatives (validation) / declAlternatives (EDC) for the restriction
// and consistency checks that run at compile time.
func elementAlternatives(decl *ElementDecl, schema *Schema) []*TypeAlternative {
	if decl == nil {
		return nil
	}
	if len(decl.Alternatives) > 0 {
		return decl.Alternatives
	}
	if decl.IsRef && schema != nil {
		if g, ok := schema.LookupElement(decl.Name.Local, decl.Name.NS); ok && g != decl {
			return g.Alternatives
		}
	}
	return nil
}

// typeTablesEquivalent reports whether two {type table}s are equivalent for the XSD
// 1.1 constraints that require it (Element Declarations Consistent; element
// restriction Particle Valid (Restriction) clause 4.6). Per the spec a Type Table
// T1 is equivalent to T2 iff their {alternatives} have the same length with
// equivalent corresponding entries (and equivalent default type definitions). The
// comparison is the spec-sanctioned CONSERVATIVE one — a processor "may treat two
// type alternatives as non-equivalent" unless it detects they are the same: two
// alternatives are equivalent only when their {test}s are identical and their
// {type definition}s are the same type definition. The default (testless)
// alternative is the final slice entry (Test == ""), so comparing the whole slice
// covers both {alternatives} and {default type definition}. Two empty tables (both
// absent) are equivalent.
func typeTablesEquivalent(a, b []*TypeAlternative) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Test != b[i].Test {
			return false
		}
		if !elementTypesConsistent(a[i].Type, b[i].Type) {
			return false
		}
	}
	return true
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
		if !ok && ref.chameleonEligible {
			// The type may come from an imported schema with no targetNamespace
			// (chameleon include); retry the no-namespace ({}) key, mirroring the
			// ordinary element @type fallback in resolveRefs.
			td, ok = c.schema.types[QName{Local: ref.qn.Local, NS: ""}]
		}
		if !ok {
			if c.filename != "" && !c.deprecatedDatatypeQName(ref.qn) {
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
// effectiveAlternatives returns the XSD 1.1 conditional-type-assignment {type
// table} in effect for an element declaration: its own alternatives, or — for an
// <xs:element ref="g"> particle whose ElementDecl is a ref that does not carry the
// table — the referenced GLOBAL declaration's alternatives (mirroring idcHostDecl's
// ref handling). Returns nil outside XSD 1.1 or when no alternatives apply.
func (vc *validationContext) effectiveAlternatives(edecl *ElementDecl) []*TypeAlternative {
	if vc.version != Version11 || edecl == nil {
		return nil
	}
	if len(edecl.Alternatives) > 0 {
		return edecl.Alternatives
	}
	if edecl.IsRef {
		if g, ok := vc.schema.LookupElement(edecl.Name.Local, edecl.Name.NS); ok && g != edecl {
			return g.Alternatives
		}
	}
	return nil
}

// hasTypeTable reports whether edecl has an effective conditional-type-assignment
// {type table}. Used to scope xsi:type handling: a present-but-empty xsi:type may
// only hard-error where it would otherwise suppress a CTA-selected type.
func (vc *validationContext) hasTypeTable(edecl *ElementDecl) bool {
	return len(vc.effectiveAlternatives(edecl)) > 0
}

func (vc *validationContext) applyTypeAlternatives(ctx context.Context, elem *helium.Element, edecl *ElementDecl, declType *TypeDef) *TypeDef {
	alts := vc.effectiveAlternatives(edecl)
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
// an element whose declared type is decl. An alternative type T is validly
// substitutable for the declared type D iff ANY of:
//
//  1. T is xs:error (always permitted — it makes the element invalid by design);
//  2. strictBuiltinAwareDerivedFrom(T, D) — genuine Type Derivation OK, covering
//     identity (T == D), user restriction/extension chains (pointer-linked via
//     isDerivedFrom), the built-in simple-type hierarchy (built-ins are not
//     BaseType-linked), and the xs:anySimpleType simple-content rule;
//  3. D is a union and T is validly substitutable for some member type, applying
//     this same predicate RECURSIVELY (so nested union members are handled).
//
// There is deliberately NO permissive "any two simple types" fallback: a genuine
// USER derivation IS caught by isDerivedFrom (user BaseType chains are linked by
// resolveRefs), so the former fallback only ever masked NON-derivations (e.g.
// xs:string for a user SmallInt, or for union(SmallInt, xs:boolean)). The W3C
// suite is the safety net — if a real derivation is missed, isDerivedFrom must be
// extended to cover it rather than reinstating the fallback.
func isValidlySubstitutable(alt, decl *TypeDef) bool {
	return isValidlySubstitutableForDeclaredType(alt, decl, true)
}

func isXsiTypeDerivedFromDeclared(actual, declared *TypeDef) bool {
	return isValidlySubstitutableForDeclaredType(actual, declared, false)
}

func isValidlySubstitutableForDeclaredType(sub, decl *TypeDef, allowError bool) bool {
	if sub == nil || decl == nil {
		return true
	}
	if allowError && isErrorType(sub) {
		return true
	}
	if strictBuiltinAwareDerivedFrom(sub, decl) {
		return true
	}
	// Use the variety/members RESOLVED through the base chain: a named type that
	// restricts an inline union keeps union variety only via its base (its direct
	// Variety/MemberTypes are empty), so reading decl.Variety/decl.MemberTypes
	// directly would skip the union branch and false-reject a member-derived
	// alternative. The member check stays strict and recursive (no fallback).
	if resolveVariety(decl) == TypeVarietyUnion && (allowError || unionDerivationHasNoFacets(decl)) {
		for _, m := range resolveUnionMembers(decl) {
			if isValidlySubstitutableForDeclaredType(sub, m, allowError) {
				return true
			}
		}
	}
	return false
}

func unionDerivationHasNoFacets(td *TypeDef) bool {
	for cur := td; cur != nil && resolveVariety(cur) == TypeVarietyUnion; cur = cur.BaseType {
		if !facetSetEmpty(cur.Facets) {
			return false
		}
	}
	return true
}

func facetSetEmpty(fs *FacetSet) bool {
	if fs == nil {
		return true
	}
	return len(fs.Enumeration) == 0 &&
		fs.MinInclusive == nil &&
		fs.MaxInclusive == nil &&
		fs.MinExclusive == nil &&
		fs.MaxExclusive == nil &&
		fs.TotalDigits == nil &&
		fs.FractionDigits == nil &&
		fs.Length == nil &&
		fs.MinLength == nil &&
		fs.MaxLength == nil &&
		fs.ExplicitTimezone == nil &&
		len(fs.Patterns) == 0 &&
		fs.WhiteSpace == nil &&
		len(fs.Assertions) == 0
}
