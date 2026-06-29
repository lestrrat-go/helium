package xsd

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xmlchar"
	"github.com/lestrrat-go/helium/xpath1"
)

type elementDeclReadOptions struct {
	name                   string
	namespace              string
	minOccurs              int
	maxOccurs              int
	defaultBlock           BlockFlags
	defaultFinal           FinalFlags
	allowAbstract          bool
	allowFinal             bool
	allowSubstitutionGroup bool
}

type attrUseReadOptions struct {
	name       QName
	includeUse bool
}

func (c *compiler) localAttributeNamespace(elem *helium.Element) string {
	if c.version == Version11 && hasAttr(elem, attrTargetNamespace) {
		return getAttr(elem, attrTargetNamespace)
	}
	form := getAttr(elem, attrForm)
	if form == attrValQualified || (form == "" && c.schema.attrFormQualified) {
		return c.schema.targetNamespace
	}
	return ""
}

func parseParticleOccurs(elem *helium.Element) (int, int) {
	minOccurs := 1
	maxOccurs := 1
	if v := getAttr(elem, attrMinOccurs); v != "" {
		minOccurs = parseOccurs(v, 1)
	}
	if v := getAttr(elem, attrMaxOccurs); v != "" {
		maxOccurs = parseOccurs(v, 1)
	}
	return minOccurs, maxOccurs
}

// parseNonNegativeOccurs parses an occurs attribute value as a non-negative
// integer. maxOccurs (allowMax) may additionally be the literal "unbounded",
// represented by the Unbounded sentinel. ok is false when the lexical value is
// not a valid non-negative integer (or "unbounded" when permitted); callers
// report a schema error in that case rather than silently accepting a bogus
// occurrence count.
func parseNonNegativeOccurs(s string, allowMax bool) (int, bool) {
	if allowMax && s == attrValUnbounded {
		return Unbounded, true
	}
	// xs:nonNegativeInteger has no leading sign: a leading '+' or '-' (including
	// "+0"/"-0") is not a valid lexical form. strconv.Atoi would accept these, so
	// reject any non-digit character before converting.
	if !isASCIIDigits(s) {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

// isASCIIDigits reports whether s is a non-empty run of ASCII digits ('0'-'9')
// with no sign, whitespace, or other characters. This matches the lexical space
// of xs:nonNegativeInteger as XSD/libxml2 enforce it for occurrence counts.
func isASCIIDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// validateOccursAttrs validates the minOccurs/maxOccurs attributes of a
// non-element particle (model group, group reference, wildcard). It rejects
// negative, signed, non-integer, and empty occurrence values, applies the
// effective-default minOccurs=1 so a maxOccurs of 0 with an absent or >=1
// minOccurs is rejected with the "maxOccurs >= 1" diagnostic, and enforces
// minOccurs <= maxOccurs. Errors are reported as libxml2-style schema parser
// errors via the compiler's error handler.
//
// xs:element particles are validated by checkLocalElement to preserve the
// libxml2 diagnostic ordering golden tests depend on; this method deliberately
// skips them.
//
// Presence is detected with hasAttr (not value!=""), so an explicitly empty
// minOccurs="" / maxOccurs="" is validated and rejected, matching xmllint.
func (c *compiler) validateOccursAttrs(ctx context.Context, elem *helium.Element) {
	if c.filename == "" {
		return
	}

	line := elem.Line()
	local := elem.LocalName()
	xsdElem := local
	// Attribute to the declaring file (the included/imported schema when inside
	// an xs:include/xs:import/xs:redefine), not the top-level compiler filename.
	src := c.diagSource()

	minPresent := hasAttr(elem, attrMinOccurs)
	maxPresent := hasAttr(elem, attrMaxOccurs)

	minVal, minOK := 1, true
	if minPresent {
		v := getAttr(elem, attrMinOccurs)
		n, ok := parseNonNegativeOccurs(v, false)
		if !ok {
			minOK = false
			c.schemaError(ctx, schemaParserErrorAttr(src, line, local, xsdElem, attrMinOccurs,
				"The value '"+v+"' is not valid. Expected is 'xs:nonNegativeInteger'."))
		} else {
			minVal = n
		}
	}

	maxVal, maxOK := 1, true
	if maxPresent {
		v := getAttr(elem, attrMaxOccurs)
		n, ok := parseNonNegativeOccurs(v, true)
		if !ok {
			maxOK = false
			c.schemaError(ctx, schemaParserErrorAttr(src, line, local, xsdElem, attrMaxOccurs,
				"The value '"+v+"' is not valid. Expected is '(xs:nonNegativeInteger | unbounded)'."))
		} else {
			maxVal = n
		}
	}

	// maxOccurs must be >= 1 unless the effective minOccurs is 0 (a legal
	// prohibited particle, minOccurs=0 maxOccurs=0). The effective minOccurs is 1
	// when minOccurs is absent or invalid, so maxOccurs=0 with an absent/explicit
	// min>=1 is rejected with the ">= 1" diagnostic.
	maxBelowOne := false
	if maxOK && maxVal != Unbounded && maxVal < 1 {
		effMin := 1
		if minPresent && minOK {
			effMin = minVal
		}
		if effMin >= 1 {
			maxBelowOne = true
			c.schemaError(ctx, schemaParserErrorAttr(src, line, local, xsdElem, attrMaxOccurs,
				"The value must be greater than or equal to 1."))
		}
	}

	// minOccurs must not exceed maxOccurs (Unbounded is treated as +inf, so it
	// can never be exceeded). Suppress this when the ">= 1" rule already fired on
	// maxOccurs; libxml2 reports only the maxOccurs error there.
	if minPresent && maxPresent && minOK && maxOK && maxVal != Unbounded && !maxBelowOne && minVal > maxVal {
		c.schemaError(ctx, schemaParserErrorAttr(src, line, local, xsdElem, attrMinOccurs,
			"The value must not be greater than the value of 'maxOccurs'."))
	}
}

func readDefaultOrFixed(elem *helium.Element) (*string, *string) {
	var defaultValue *string
	if hasAttr(elem, attrDefault) {
		v := getAttr(elem, attrDefault)
		defaultValue = &v
	}

	var fixedValue *string
	if hasAttr(elem, attrFixed) {
		v := getAttr(elem, attrFixed)
		fixedValue = &v
	}

	return defaultValue, fixedValue
}

// readProcessContents reads and validates the @processContents attribute of a
// wildcard. An absent attribute defaults to "strict". An invalid value is
// reported as a schema parser error and treated as the "strict" default.
func (c *compiler) readProcessContents(ctx context.Context, elem *helium.Element) ProcessContentsKind {
	if !hasAttr(elem, attrProcessContents) {
		return ProcessStrict
	}
	switch v := normalizeWhiteSpace(getAttr(elem, attrProcessContents), "collapse"); v {
	case attrValStrict:
		return ProcessStrict
	case attrValLax:
		return ProcessLax
	case attrValSkip:
		return ProcessSkip
	default:
		if c.filename != "" {
			msg := fmt.Sprintf("'%s' is not a valid value of the union type '#processContents'.", v)
			c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), elem.Line(),
				elem.LocalName(), elem.LocalName(), attrProcessContents, msg))
		}
		return ProcessStrict
	}
}

// validateWildcardNamespace validates the namespace-constraint grammar of a
// wildcard's @namespace attribute (XSD 3.10.2). The value is either the keyword
// "##any" or "##other", or a whitespace-separated list whose members are each an
// anyURI, "##targetNamespace", or "##local". The "##any"/"##other" keywords
// must stand alone, and no other "##"-prefixed token is allowed.
func (c *compiler) validateWildcardNamespace(ctx context.Context, elem *helium.Element, raw string) {
	if c.filename == "" {
		return
	}
	reject := func(msg string) {
		c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), elem.Line(),
			elem.LocalName(), elem.LocalName(), attrNamespace, msg))
	}

	// The "##any" / "##other" keywords are EXACT singleton lexical forms: the
	// value must equal the keyword with no surrounding whitespace and no other
	// tokens. libxml2 rejects e.g. "  ##any  ". Compare against the raw value
	// (NOT the whitespace-collapsed value) so padding is caught. True list forms
	// like "##local ##targetNamespace" are still whitespace-collapsed below, so
	// surrounding/inner whitespace around list members remains valid.
	switch raw {
	case WildcardNSAny, WildcardNSOther:
		return
	}

	tokens := splitSpace(normalizeWhiteSpace(raw, "collapse"))
	if len(tokens) == 0 {
		return
	}
	for _, tok := range tokens {
		switch tok {
		case WildcardNSAny, WildcardNSOther:
			// A bare keyword reaching here means raw != keyword (it had padding
			// or extra tokens), so it is never a valid standalone keyword form.
			reject(fmt.Sprintf("The value '%s' is not a valid namespace constraint: '%s' must not be combined with other items.", raw, tok))
			return
		case WildcardNSTargetNamespace, WildcardNSLocal:
			// Valid only as list members.
		default:
			if strings.HasPrefix(tok, "##") {
				reject(fmt.Sprintf("The value '%s' is not a valid namespace constraint: '%s' is not a recognized '##' token.", raw, tok))
				return
			}
			// Otherwise treated as an anyURI namespace name.
		}
	}
}

func (c *compiler) readWildcard(ctx context.Context, elem *helium.Element) *Wildcard {
	hasNS := hasAttr(elem, attrNamespace)
	namespace := getAttr(elem, attrNamespace)
	if !hasNS {
		// ABSENT namespace defaults to ##any. A present-but-empty
		// namespace="" is preserved: it is a (degenerate) namespace list
		// that matches nothing, which is distinct from the ##any default.
		namespace = WildcardNSAny
	} else {
		c.validateWildcardNamespace(ctx, elem, namespace)
	}

	wc := &Wildcard{
		Namespace:       namespace,
		ProcessContents: c.readProcessContents(ctx, elem),
		TargetNS:        c.schema.targetNamespace,
	}

	// XSD 1.1 negated namespace / name constraints. Gated on Version11 so 1.0
	// behavior is byte-identical (the attributes are not recognized in 1.0; if
	// present they are simply ignored, as any foreign attribute would be).
	if c.version != Version11 {
		return wc
	}

	hasNotNS := hasAttr(elem, attrNotNamespace)
	// @namespace and @notNamespace are mutually exclusive (XSD 3.10.2,
	// no-xmlns / src-wildcard): a wildcard may carry at most one of them.
	if hasNS && hasNotNS {
		c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), elem.Line(),
			elem.LocalName(), elem.LocalName(), attrNotNamespace,
			"The attributes 'namespace' and 'notNamespace' are mutually exclusive."))
	}
	if hasNotNS {
		wc.NotNamespace = c.parseNotNamespace(ctx, elem, getAttr(elem, attrNotNamespace))
	}
	if hasAttr(elem, attrNotQName) {
		c.parseNotQName(ctx, elem, wc, getAttr(elem, attrNotQName), isXSDElement(elem, elemAnyAttribute))
	}
	return wc
}

// parseNotNamespace parses an xs:any/xs:anyAttribute @notNamespace value (XSD
// 1.1). The value is an xs:basicNamespaceList whose members are each an anyURI,
// "##targetNamespace", or "##local"; the "##any"/"##other" keywords are NOT
// permitted. It returns the resolved set of EXCLUDED namespace URIs ("" for
// ##local / the absent namespace).
//
// An EMPTY list (e.g. notNamespace="") is VALID: it is a `not` constraint with
// an empty excluded set, which admits every namespace (XSD 1.1 §3.10.1). The
// caller passes a present (possibly empty) attribute value, so a NON-NIL empty
// slice is returned to mark the wildcard as a notNamespace constraint — distinct
// from an ABSENT @notNamespace (nil), which leaves the @namespace constraint in
// effect. wildcardMatches treats a non-nil empty NotNamespace as "excludes
// nothing" (matches all).
func (c *compiler) parseNotNamespace(ctx context.Context, elem *helium.Element, raw string) []string {
	reject := func(msg string) {
		c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), elem.Line(),
			elem.LocalName(), elem.LocalName(), attrNotNamespace, msg))
	}
	tokens := splitSpace(normalizeWhiteSpace(raw, "collapse"))
	seen := make(map[string]struct{}, len(tokens))
	out := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		var ns string
		switch tok {
		case WildcardNSTargetNamespace:
			ns = c.schema.targetNamespace
		case WildcardNSLocal:
			ns = ""
		case WildcardNSAny, WildcardNSOther:
			reject(fmt.Sprintf("The value '%s' is not valid in a 'notNamespace' list.", tok))
			continue
		default:
			if strings.HasPrefix(tok, "##") {
				reject(fmt.Sprintf("The value '%s' is not a recognized '##' token in 'notNamespace'.", tok))
				continue
			}
			ns = tok
		}
		if _, dup := seen[ns]; dup {
			continue
		}
		seen[ns] = struct{}{}
		out = append(out, ns)
	}
	return out
}

// parseNotQName parses an xs:any/xs:anyAttribute @notQName value (XSD 1.1) into
// the wildcard's disallowed-name set. Members are each a QName, "##defined", or
// (for xs:any only) "##definedSibling". Each QName must be lexically valid, its
// prefix bound, and its namespace admitted by the wildcard's namespace
// constraint (otherwise listing it would be pointless and is a schema error).
// isAttr is true for an xs:anyAttribute wildcard, for which "##definedSibling"
// is NOT a permitted token (XSD 1.1 restricts it to ELEMENT wildcards).
func (c *compiler) parseNotQName(ctx context.Context, elem *helium.Element, wc *Wildcard, raw string, isAttr bool) {
	reject := func(msg string) {
		c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), elem.Line(),
			elem.LocalName(), elem.LocalName(), attrNotQName, msg))
	}
	tokens := splitSpace(normalizeWhiteSpace(raw, "collapse"))
	for _, tok := range tokens {
		switch tok {
		case WildcardQNameDefined:
			wc.NotQNameDefined = true
			continue
		case WildcardQNameDefinedSibling:
			if isAttr {
				reject("The value '##definedSibling' is only allowed on an element wildcard, not on 'anyAttribute'.")
				continue
			}
			wc.NotQNameDefinedSibling = true
			continue
		}
		if !xmlchar.IsValidQName(tok) {
			reject(fmt.Sprintf("The value '%s' is not a valid QName.", tok))
			continue
		}
		qn := c.resolveNotQName(ctx, elem, tok)
		// The name's namespace must be admitted by the wildcard's namespace
		// constraint; excluding a name the wildcard could never match is an error
		// (cvc/wildcard: notQName names must be in an allowed namespace).
		if !wildcardMatches(wc, qn.NS) {
			reject(fmt.Sprintf("The QName '%s' in 'notQName' is not in a namespace allowed by the wildcard.", tok))
			continue
		}
		wc.NotQName = append(wc.NotQName, qn)
	}
}

// resolveNotQName resolves a @notQName QName token using resolve-QName ACTUAL
// VALUE semantics (XSD 1.1), which differ from schema-component reference
// resolution (c.resolveQName): an UNPREFIXED token resolves through the in-scope
// DEFAULT namespace, or to the ABSENT namespace when there is no default — it
// must NEVER fall back to the schema's targetNamespace. A prefixed token uses
// the in-scope binding (and the predeclared xml prefix), with the same
// unbound-prefix diagnostic as c.resolveQName.
func (c *compiler) resolveNotQName(ctx context.Context, elem *helium.Element, ref string) QName {
	if prefix, local, found := strings.Cut(ref, ":"); found {
		ns := lookupNS(elem, prefix)
		if ns == "" && prefix != "" {
			c.reportUnboundQNamePrefix(ctx, elem, ref, prefix)
		}
		return QName{Local: local, NS: ns}
	}
	// Unprefixed: the in-scope default namespace, else absent. No targetNamespace
	// fallback (lookupNS returns "" when no default xmlns is in scope).
	return QName{Local: ref, NS: lookupNS(elem, "")}
}

func (c *compiler) readElementDecl(ctx context.Context, elem *helium.Element, opts elementDeclReadOptions) (*ElementDecl, error) {
	decl := &ElementDecl{
		Name:      QName{Local: opts.name, NS: opts.namespace},
		MinOccurs: opts.minOccurs,
		MaxOccurs: opts.maxOccurs,
		Nillable:  c.readBooleanAttr(ctx, elem, attrNillable),
	}

	if opts.allowAbstract {
		decl.Abstract = c.readBooleanAttr(ctx, elem, attrAbstract)
	}

	if opts.allowSubstitutionGroup {
		if sg := getAttr(elem, attrSubstitutionGroup); sg != "" {
			decl.SubstitutionGroup = c.resolveQName(ctx, elem, sg)
		}
	}

	decl.Default, decl.Fixed = readDefaultOrFixed(elem)
	if decl.Fixed != nil {
		decl.FixedNS = collectNSContext(elem)
	}
	if decl.Default != nil {
		decl.DefaultNS = collectNSContext(elem)
	}

	if hasAttr(elem, attrBlock) {
		decl.Block = parseBlockFlags(getAttr(elem, attrBlock))
		decl.BlockSet = true
	} else {
		decl.Block = opts.defaultBlock
	}

	if opts.allowFinal {
		if hasAttr(elem, attrFinal) {
			decl.Final = parseElemFinalFlags(getAttr(elem, attrFinal))
			decl.FinalSet = true
		} else {
			decl.Final = opts.defaultFinal
		}
	}

	if err := c.readElementType(ctx, elem, decl, opts.name); err != nil {
		return nil, err
	}
	decl.IDCs = c.parseIDConstraints(ctx, elem)
	if c.version == Version11 {
		decl.Alternatives = c.parseTypeAlternatives(ctx, elem)
		if len(decl.Alternatives) > 0 {
			c.ctaElems = append(c.ctaElems, decl)
		}
	}
	return decl, nil
}

// readBooleanAttr reads a schema-side xs:boolean attribute (e.g. nillable,
// abstract, mixed) applying whitespace-collapse lexical rules. It accepts the
// four canonical xs:boolean lexical forms (true/false/1/0); an absent attribute
// is false. An invalid lexical form is reported as a schema parser error and
// treated as false. The owning element's local name is used in the diagnostic
// so the same helper serves every boolean schema attribute.
func (c *compiler) readBooleanAttr(ctx context.Context, elem *helium.Element, attr string) bool {
	if !hasAttr(elem, attr) {
		return false
	}
	v, ok := parseSchemaBool(getAttr(elem, attr))
	if ok {
		return v
	}
	msg := fmt.Sprintf("'%s' is not a valid value of the atomic type 'xs:boolean'.", normalizeWhiteSpace(getAttr(elem, attr), "collapse"))
	c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), elem.Line(),
		elem.LocalName(), elem.LocalName(), attr, msg))
	return false
}

// parseSchemaBool parses an xs:boolean lexical value, applying the
// whitespace-collapse rule and accepting the four canonical forms
// "true"/"false"/"1"/"0". The second return value is false when the value is
// not a valid xs:boolean lexical form.
func parseSchemaBool(raw string) (bool, bool) {
	switch normalizeWhiteSpace(raw, "collapse") {
	case "true", "1":
		return true, true
	case "false", "0":
		return false, true
	default:
		return false, false
	}
}

func (c *compiler) readElementType(ctx context.Context, elem *helium.Element, decl *ElementDecl, sourceName string) error {
	typeRef := getAttr(elem, attrType)
	if typeRef != "" {
		qn := c.resolveQName(ctx, elem, typeRef)
		c.elemRefs[decl] = qn
		c.markChameleonEligible(decl, elem, typeRef)
		c.elemRefSources[decl] = elemRefSource{elemName: sourceName, line: elem.Line(), source: c.diagSource()}
		return nil
	}

	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXSDElement(ce, elemAnnotation):
			c.checkAnnotation(ctx, ce)
		case isXSDElement(ce, elemComplexType):
			td, err := c.parseComplexType(ctx, ce)
			if err != nil {
				return err
			}
			decl.Type = td
		case isXSDElement(ce, elemSimpleType):
			td, err := c.parseSimpleType(ctx, ce)
			if err != nil {
				return err
			}
			decl.Type = td
		}
	}

	// An element declaration with no explicit type, no inline type, and no
	// substitution-group head to inherit from defaults to the built-in
	// xs:anyType (XSD 3.3.2: {type definition} defaults to xs:anyType). This
	// ensures xsi:nil lexical validation and nilled-empty enforcement run for
	// no-type declarations the same as for typed ones. Substitution-group
	// members are left untyped so they can inherit the head's type at validation.
	if decl.Type == nil && decl.SubstitutionGroup == (QName{}) {
		decl.Type = c.schema.types[QName{Local: typeAnyType, NS: lexicon.NamespaceXSD}]
	}

	return nil
}

func (c *compiler) readAttributeUseDecl(ctx context.Context, elem *helium.Element, opts attrUseReadOptions) *AttrUse {
	au := &AttrUse{Name: opts.name}
	if typeRef := getAttr(elem, attrType); typeRef != "" {
		au.TypeName = c.resolveQName(ctx, elem, typeRef)
	} else {
		// No type attribute: look for an inline anonymous <xs:simpleType>.
		// (type and inline simpleType are mutually exclusive, enforced by
		// checkAttributeUse.)
		for child := range helium.Children(elem) {
			if child.Type() != helium.ElementNode {
				continue
			}
			ce, ok := helium.AsNode[*helium.Element](child)
			if !ok {
				continue
			}
			if !isXSDElement(ce, elemSimpleType) {
				continue
			}
			td, err := c.parseSimpleType(ctx, ce)
			if err != nil {
				c.schemaError(ctx, schemaParserError(c.filename, ce.Line(),
					ce.LocalName(), "attribute", err.Error()))
				break
			}
			au.Type = td
			break
		}
	}
	if opts.includeUse {
		switch getAttr(elem, attrUse) {
		case attrValRequired:
			au.Required = true
		case attrValProhibited:
			au.Prohibited = true
		}
	}
	// XSD 1.1 inheritable: a non-boolean lexical (e.g. "" or "2") is a schema
	// error (reported by readBooleanAttr); whitespace is collapsed (" 1 " → 1).
	if c.version == Version11 && hasAttr(elem, attrInheritable) {
		au.Inheritable = c.readBooleanAttr(ctx, elem, attrInheritable)
		au.InheritableSet = true
	}
	c.attrUseSources[au] = attrConstraintSource{
		line:   elem.Line(),
		local:  opts.name.Local,
		source: c.includeFile,
	}
	au.Default, au.Fixed = readDefaultOrFixed(elem)
	if au.Fixed != nil {
		au.FixedNS = collectNSContext(elem)
	}
	if au.Default != nil {
		au.DefaultNS = collectNSContext(elem)
	}
	// Record source info so the default/fixed constraint value can be validated
	// against the attribute's declared simple type once type refs are resolved.
	if au.Default != nil || au.Fixed != nil {
		c.attrUseConstraintSources[au] = attrConstraintSource{
			line:   elem.Line(),
			local:  opts.name.Local,
			nsMap:  collectNSContext(elem),
			source: c.includeFile,
		}
	}
	return au
}

func (c *compiler) parseGlobalElement(ctx context.Context, elem *helium.Element) error {
	c.checkGlobalElement(ctx, elem)
	name := getAttr(elem, attrName)
	if name == "" {
		// Still register with a placeholder name to continue parsing.
		return nil
	}

	// Check for a duplicate global element declaration BEFORE reading the decl
	// body, so a rejected declaration records no type/element refs that would
	// produce unrelated follow-on errors.
	declName := QName{Local: name, NS: c.schema.targetNamespace}
	if _, exists := c.schema.elements[declName]; exists {
		c.reportDuplicateComponent(ctx, elem, "element", "A global element declaration", declName)
		return nil
	}

	decl, err := c.readElementDecl(ctx, elem, elementDeclReadOptions{
		name:                   name,
		namespace:              c.schema.targetNamespace,
		minOccurs:              1,
		maxOccurs:              1,
		defaultBlock:           c.schema.blockDefault,
		defaultFinal:           c.schema.finalDefault & (FinalExtension | FinalRestriction),
		allowAbstract:          true,
		allowFinal:             true,
		allowSubstitutionGroup: true,
	})
	if err != nil {
		return err
	}

	c.globalElemSources[decl] = elemRefSource{elemName: name, line: elem.Line(), source: c.diagSource()}
	c.schema.elements[decl.Name] = decl
	return nil
}

// reportDuplicateComponent emits the schema-parser error for a redeclared
// global component (element, type, model group, or attribute group). component
// is the XSD element name used in the error prefix (e.g. "element", "type");
// kind is the descriptive phrase (e.g. "A global element declaration").
func (c *compiler) reportDuplicateComponent(ctx context.Context, elem *helium.Element, component, kind string, name QName) {
	qnDisplay := "'" + name.NS + "'" + name.Local
	if name.NS != "" {
		qnDisplay = "'{" + name.NS + "}" + name.Local + "'"
	}
	// A duplicate inside an xs:include/xs:redefine reports against that file
	// (c.includeFile); a top-level (main-schema) duplicate has no includeFile,
	// so fall back to the compiler's own filename label so the diagnostic keeps
	// its path prefix instead of starting with ":line:".
	source := c.includeFile
	if source == "" {
		source = c.filename
	}
	c.schemaError(ctx, schemaParserError(source, elem.Line(),
		elem.LocalName(), component,
		kind+" "+qnDisplay+" does already exist."))
}

func (c *compiler) parseLocalElement(ctx context.Context, elem *helium.Element) (*Particle, error) {
	c.checkLocalElement(ctx, elem)
	minOcc, maxOcc := parseParticleOccurs(elem)

	// Handle element references (ref="...").
	if ref := getAttr(elem, attrRef); ref != "" {
		qn := c.resolveQName(ctx, elem, ref)
		edecl := &ElementDecl{
			Name:      qn,
			MinOccurs: minOcc,
			MaxOccurs: maxOcc,
			IsRef:     true,
		}
		c.elemRefs[edecl] = qn
		c.elemRefSources[edecl] = elemRefSource{elemName: elem.LocalName(), line: elem.Line(), source: c.diagSource()}
		return &Particle{
			MinOccurs: minOcc,
			MaxOccurs: maxOcc,
			Term:      edecl,
		}, nil
	}

	name := getAttr(elem, attrName)
	if name == "" {
		return nil, fmt.Errorf("xsd: local element missing name")
	}

	// Determine element namespace based on form and elementFormDefault.
	elemNS := ""
	form := getAttr(elem, attrForm)
	if form == attrValQualified || (form == "" && c.schema.elemFormQualified) {
		elemNS = c.schema.targetNamespace
	}

	edecl, err := c.readElementDecl(ctx, elem, elementDeclReadOptions{
		name:         name,
		namespace:    elemNS,
		minOccurs:    minOcc,
		maxOccurs:    maxOcc,
		defaultBlock: c.schema.blockDefault,
	})
	if err != nil {
		return nil, err
	}

	return &Particle{
		MinOccurs: minOcc,
		MaxOccurs: maxOcc,
		Term:      edecl,
	}, nil
}

func (c *compiler) parseWildcard(ctx context.Context, elem *helium.Element) *Particle {
	c.validateOccursAttrs(ctx, elem)
	minOcc, maxOcc := parseParticleOccurs(elem)
	wc := c.readWildcard(ctx, elem)
	return &Particle{
		MinOccurs: minOcc,
		MaxOccurs: maxOcc,
		Term:      wc,
	}
}

func (c *compiler) parseAnyAttribute(ctx context.Context, elem *helium.Element) *Wildcard {
	return c.readWildcard(ctx, elem)
}

func (c *compiler) parseGlobalAttribute(ctx context.Context, elem *helium.Element) {
	c.checkAttributeUse(ctx, elem)
	name := getAttr(elem, attrName)
	if name == "" {
		return
	}
	// Global attributes are always in the target namespace (per spec).
	qn := QName{Local: name, NS: c.schema.targetNamespace}

	// A GLOBAL attribute declaration must not be in the XSI namespace (XSD 1.1
	// Schema Component Constraint "no-xsi" / xs:attribute representation): the XSI
	// namespace is reserved for the four processor attributes and a schema may not
	// add to it. This is gated on Version11: it is NEW in this PR and the opt-in
	// contract requires 1.0 to stay byte-identical to origin/feat-xsd11, which has
	// no global-attribute no-xsi check (the pre-existing check in check_elements.go
	// covers only LOCAL qualified attributes and is left unchanged for 1.0).
	if c.version == Version11 && c.filename != "" && qn.NS == lexicon.NamespaceXSI {
		c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), elem.Line(),
			elem.LocalName(), elem.LocalName(), attrName,
			"An attribute declaration must not be in the XSI namespace."))
		return
	}

	// Check for a duplicate global attribute declaration BEFORE parsing the body,
	// so a rejected declaration records no type/constraint refs that would
	// produce unrelated follow-on errors. xs:redefine never targets global
	// attributes, so (mirroring the other named components) only suppress the
	// report when processing redefine overrides.
	if _, exists := c.schema.globalAttrs[qn]; exists && c.redefine == nil {
		c.reportDuplicateComponent(ctx, elem, "attribute", "A global attribute declaration", qn)
		return
	}

	au := c.readAttributeUseDecl(ctx, elem, attrUseReadOptions{
		name: qn,
	})

	c.schema.globalAttrs[qn] = au
}

func (c *compiler) parseAttributeUse(ctx context.Context, elem *helium.Element) *AttrUse {
	c.checkAttributeUse(ctx, elem)
	// Handle attribute references (ref="...").
	if ref := getAttr(elem, attrRef); ref != "" {
		qn := c.resolveQName(ctx, elem, ref)
		au := &AttrUse{Name: qn}
		switch getAttr(elem, attrUse) {
		case attrValRequired:
			au.Required = true
		case attrValProhibited:
			au.Prohibited = true
		}
		// XSD 1.1: an explicit inheritable on the ref USE wins over the referenced
		// global declaration's value (resolved in resolveRefs when unset here).
		if c.version == Version11 && hasAttr(elem, attrInheritable) {
			au.Inheritable = c.readBooleanAttr(ctx, elem, attrInheritable)
			au.InheritableSet = true
		}
		// Record the source for a prohibited ref'd use so the pointless-prohibition
		// warning can cite its line and declaring file. Only prohibited uses need
		// this here; a non-prohibited ref'd use carries no inline type to feed the
		// other attrUseSources consumers (e.g. the NOTATION enumeration check).
		if au.Prohibited {
			c.attrUseSources[au] = attrConstraintSource{
				line:   elem.Line(),
				local:  qn.Local,
				source: c.includeFile,
			}
		}
		if hasAttr(elem, attrDefault) {
			v := getAttr(elem, attrDefault)
			au.Default = &v
			au.DefaultNS = collectNSContext(elem)
		}
		if hasAttr(elem, attrFixed) {
			v := getAttr(elem, attrFixed)
			au.Fixed = &v
			au.FixedNS = collectNSContext(elem)
		}
		// Record source info so a local default/fixed constraint on a ref'd
		// attribute use is validated against the resolved (global) attribute's
		// simple type once resolveRefs copies the type in.
		if au.Default != nil || au.Fixed != nil {
			c.attrUseConstraintSources[au] = attrConstraintSource{
				line:   elem.Line(),
				local:  qn.Local,
				nsMap:  collectNSContext(elem),
				source: c.includeFile,
			}
		}
		c.attrRefs[au] = qn
		return au
	}

	name := getAttr(elem, attrName)
	return c.readAttributeUseDecl(ctx, elem, attrUseReadOptions{
		name:       QName{Local: name, NS: c.localAttributeNamespace(elem)},
		includeUse: true,
	})
}

// parseIDConstraints scans element children for xs:key, xs:keyref, xs:unique declarations.
func (c *compiler) parseIDConstraints(ctx context.Context, elem *helium.Element) []*IDConstraint {
	var idcs []*IDConstraint
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		var kind IDCKind
		switch {
		case isXSDElement(ce, elemUnique):
			kind = IDCUnique
		case isXSDElement(ce, elemKey):
			kind = IDCKey
		case isXSDElement(ce, elemKeyRef):
			kind = IDCKeyRef
		default:
			continue
		}
		idc := c.parseIDConstraint(ctx, ce, kind)
		if idc != nil {
			idcs = append(idcs, idc)
		}
	}
	return idcs
}

// parseIDConstraint parses a single xs:key, xs:keyref, or xs:unique declaration.
func (c *compiler) parseIDConstraint(ctx context.Context, elem *helium.Element, kind IDCKind) *IDConstraint {
	name := getAttr(elem, attrName)
	// Detect the @ref form by PRESENCE, not value: getAttr cannot tell an absent
	// attribute from an empty one, so a literal ref="" must be recognized as the
	// (invalid) ref form rather than silently treated as absent and dropped.
	hasRef := c.version == Version11 && hasAttr(elem, attrRef)
	if name == "" && !hasRef {
		return nil
	}

	// XSD 1.1 identity-constraint @ref: the constraint reuses a referenced
	// constraint's name/selector/field. The ref form may carry only annotation/id
	// metadata, so name/selector/field/refer MUST be absent; the referenced
	// constraint is resolved (and its selector/fields copied in) at compile time
	// by resolveConstraintRefs.
	if hasRef {
		source := c.includeFile
		if source == "" {
			source = c.filename
		}
		xsdElem := idcKindName(kind)
		ref := getAttr(elem, attrRef)
		// An empty @ref names no constraint and is a fatal schema error; drop the
		// constraint so resolveConstraintRefs does not also report it as unknown.
		if ref == "" {
			if source != "" {
				c.schemaError(ctx, schemaParserErrorAttr(source, elem.Line(), xsdElem, xsdElem, attrRef,
					"An identity-constraint 'ref' attribute must not be empty."))
			}
			return nil
		}
		// A @ref constraint must not also declare its own name/selector/field/refer
		// (the ref form is mutually exclusive with the full form). Companions are
		// detected by PRESENCE (hasAttr), not value, so an empty-but-present
		// name=""/refer="" is still rejected, consistent with the ref-form detection.
		if hasAttr(elem, attrName) {
			c.reportIDCRefConflict(ctx, source, elem.Line(), xsdElem, attrName)
		}
		if hasAttr(elem, attrRefer) {
			c.reportIDCRefConflict(ctx, source, elem.Line(), xsdElem, attrRefer)
		}
		for child := range helium.Children(elem) {
			ce, ok := helium.AsNode[*helium.Element](child)
			if !ok {
				continue
			}
			if isXSDElement(ce, elemSelector) {
				c.reportIDCRefConflict(ctx, source, elem.Line(), xsdElem, elemSelector)
			}
			if isXSDElement(ce, elemField) {
				c.reportIDCRefConflict(ctx, source, elem.Line(), xsdElem, elemField)
			}
		}
		idc := &IDConstraint{
			Kind:            kind,
			Namespaces:      collectNSContext(elem),
			Line:            elem.Line(),
			Source:          source,
			IsConstraintRef: true,
			ConstraintRef:   ref,
		}
		idc.ConstraintRefQName, idc.constraintRefUnbound = c.resolveIDCNameQName(ctx, elem, ref)
		return idc
	}
	// Source pins the filename of the schema document that declares this
	// constraint, paired with Line. A constraint parsed inside an
	// xs:include/xs:redefine carries that file (c.includeFile); a constraint
	// parsed by an import sub-compiler carries that sub-compiler's filename. So a
	// deferred @refer check run later by the (top-level) compiler cites the
	// declaring file, not the importing compiler's filename, matching Line.
	source := c.includeFile
	if source == "" {
		source = c.filename
	}
	idc := &IDConstraint{
		Name: name,
		// The name attribute is an NCName; the constraint's identity is the
		// QName {targetNamespace}name (XSD identity-constraints live in the
		// schema's target namespace).
		QName:      QName{Local: name, NS: c.schema.targetNamespace},
		Kind:       kind,
		Namespaces: collectNSContext(elem),
		Line:       elem.Line(),
		Source:     source,
	}
	if kind == IDCKeyRef {
		idc.Refer = getAttr(elem, attrRefer)
		// @refer is a QName; resolve it namespace-aware against the constraint
		// element's in-scope namespaces. An empty refer or an unbound prefix is a
		// fatal schema error (reported later by checkKeyRefRefers, which also
		// verifies the referenced constraint exists). Store the resolved QName so
		// validation can look the target up by full identity rather than by local
		// name only.
		idc.ReferQName, idc.referUnbound = c.resolveIDCReferQName(ctx, elem, idc)
	}
	// fieldLines tracks the source line of each <field>, parallel to idc.Fields,
	// so a malformed field XPath is reported against the right element.
	var selectorLine int
	var fieldLines []int
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXSDElement(ce, elemSelector):
			idc.Selector = getAttr(ce, attrXPath)
			idc.SelectorDefaultNS = c.resolveXPathDefaultNS(ce)
			selectorLine = ce.Line()
		case isXSDElement(ce, elemField):
			idc.Fields = append(idc.Fields, getAttr(ce, attrXPath))
			idc.FieldDefaultNS = append(idc.FieldDefaultNS, c.resolveXPathDefaultNS(ce))
			fieldLines = append(fieldLines, ce.Line())
		}
	}

	// Pre-compile selector XPath expression. A malformed selector XPath is a
	// fatal schema error: leaving SelectorExpr nil would silently disable the
	// whole constraint (the field-level uniqueness/keyref checks would never
	// run), so an invalid schema must fail to compile rather than validate
	// documents as if no constraint were present.
	if idc.Selector != "" {
		compiled, err := compileIDCXPath(idc.Selector, false)
		if err != nil {
			c.reportIDCXPathError(ctx, elemSelector, selectorLine, idc.Selector, err)
		} else {
			idc.SelectorExpr = compiled
		}
	}

	// Pre-compile field XPath expressions. A malformed field XPath is likewise
	// fatal: with FieldExprs[i] nil the field would fall back to a per-validation
	// recompile that also fails and is currently swallowed, again disabling the
	// constraint.
	idc.FieldExprs = make([]*xpath1.Expression, len(idc.Fields))
	for i, f := range idc.Fields {
		compiled, err := compileIDCXPath(f, true)
		if err != nil {
			line := 0
			if i < len(fieldLines) {
				line = fieldLines[i]
			}
			c.reportIDCXPathError(ctx, elemField, line, f, err)
			continue
		}
		idc.FieldExprs[i] = compiled
	}

	return idc
}

// resolveXPathDefaultNSToken resolves a raw @xpathDefaultNamespace value to a
// default ELEMENT namespace URI against elem's namespace context: empty/##local →
// no default (""), ##targetNamespace → targetNS, ##defaultNamespace → elem's
// in-scope default namespace, any other value → the literal URI. ##defaultNamespace
// is the namespace-context-SENSITIVE case: a schema-level value must be resolved
// against the SCHEMA ROOT (where the attribute appears), NOT later against a
// selector/field that may redeclare the default namespace — so schema-level values
// are pre-resolved at root-read time (see compiler.schemaXPathDefaultNS) and
// inherited as the already-resolved URI.
//
// @xpathDefaultNamespace is xs:anyURI (whiteSpace=collapse), so the raw lexical
// value is whitespace-collapsed BEFORE matching the ##keyword forms — a padded
// "  ##targetNamespace  " must resolve to the target namespace, not be mistaken for
// a literal URI. Every caller (schema-level root/include/redefine/import and the
// idc/CTA local selector/field paths) routes through here, so the collapse is
// centralized here rather than duplicated per call site.
func resolveXPathDefaultNSToken(elem *helium.Element, raw, targetNS string) string {
	collapsed := normalizeWhiteSpace(raw, "collapse")
	switch collapsed {
	case "", xpathDefaultNSLocal:
		return ""
	case xpathDefaultNSTargetNamespace:
		return targetNS
	case xpathDefaultNSDefaultNamespace:
		return lookupNS(elem, "")
	default:
		return collapsed
	}
}

// resolveXPathDefaultNS resolves the effective default element namespace for an
// identity-constraint selector/field XPath (XSD 1.1). A LOCALLY-PRESENT value on
// the selector/field element is resolved against THAT element's context. The
// schema-level @xpathDefaultNamespace is inherited ONLY when the attribute is
// ABSENT on the element — detected by PRESENCE (hasAttr), since xs:anyURI admits
// the empty value and getAttr cannot tell an explicit @xpathDefaultNamespace=""
// from an absent one, so an explicit empty value means "no default element
// namespace" and does NOT inherit. The inherited value is the schema-level URI
// ALREADY RESOLVED against the schema root when the root was read, so an inherited
// ##defaultNamespace uses the root's default namespace, not this selector/field's.
// An absent value (and 1.0 mode) yields no default. Returns "" for "no default".
func (c *compiler) resolveXPathDefaultNS(elem *helium.Element) string {
	if c.version != Version11 {
		return ""
	}
	if !hasAttr(elem, attrXPathDefaultNS) {
		return c.schemaXPathDefaultNS
	}
	return resolveXPathDefaultNSToken(elem, getAttr(elem, attrXPathDefaultNS), c.schema.targetNamespace)
}

// resolveIDCReferQName resolves an xs:keyref/@refer QName against the constraint
// element's in-scope namespaces. An unprefixed refer resolves to the in-scope
// default namespace (falling back to the schema's target namespace), matching how
// other XSD QName-valued attributes (@type, @ref) are resolved. A prefixed refer
// whose prefix is not bound in scope is a fatal schema error; the returned bool
// reports that so checkKeyRefRefers can suppress its own "unknown key" diagnostic.
func (c *compiler) resolveIDCReferQName(ctx context.Context, elem *helium.Element, idc *IDConstraint) (QName, bool) {
	refer := idc.Refer
	if refer == "" {
		// An empty @refer is reported by checkKeyRefRefers.
		return QName{}, false
	}
	// Report against the constraint's DECLARING file (idc.Source, pinned in
	// parseIDConstraint to c.includeFile/c.filename), paired with idc.Line, so a
	// malformed/unbound @refer in an INCLUDED or REDEFINED schema cites the
	// included file — not the including schema — matching the line number used.
	source := idc.Source
	if source == "" {
		source = c.diagSource()
	}
	// A malformed @refer (e.g. ":k") is a fatal error, not a silently-resolved
	// unprefixed reference. The returned bool suppresses checkKeyRefRefers's own
	// "unknown key" diagnostic, mirroring the unbound-prefix path.
	if err := validateQName(refer); err != nil {
		if source != "" {
			msg := fmt.Sprintf("The keyref identity-constraint '%s' has a 'refer' attribute '%s' that is not a valid QName.", idc.Name, refer)
			c.schemaError(ctx,
				schemaParserErrorAttr(source, idc.Line, elemKeyRef, elemKeyRef, attrRefer, msg))
		}
		return QName{}, true
	}
	if prefix, local, found := strings.Cut(refer, ":"); found {
		ns := lookupNS(elem, prefix)
		if ns == "" && prefix != "" {
			msg := fmt.Sprintf("The keyref identity-constraint '%s' has a 'refer' attribute '%s' whose namespace prefix '%s' is not bound.", idc.Name, refer, prefix)
			if source != "" {
				c.schemaError(ctx,
					schemaParserErrorAttr(source, idc.Line, elemKeyRef, elemKeyRef, attrRefer, msg))
			}
			return QName{}, true
		}
		if c.rejectDeprecatedDatatypeNamespace(ctx, elem, refer, ns) {
			return QName{}, true
		}
		return QName{Local: local, NS: ns}, false
	}
	// Unprefixed: use the in-scope default namespace, else the target namespace.
	ns := c.schema.targetNamespace
	if defNS := lookupNS(elem, ""); defNS != "" {
		ns = defNS
	}
	if c.rejectDeprecatedDatatypeNamespace(ctx, elem, refer, ns) {
		return QName{}, true
	}
	return QName{Local: refer, NS: ns}, false
}

// resolveIDCNameQName resolves an identity-constraint @ref QName against the
// element's in-scope namespaces. A prefixed ref resolves its prefix; a prefixed
// ref whose prefix is not bound in scope is a fatal schema error (reported via
// reportUnboundQNamePrefix, mirroring every other QName-valued schema attribute)
// rather than silently mapping to the empty namespace — the returned bool reports
// that so resolveConstraintRefs can suppress its own "unknown constraint"
// diagnostic. An unprefixed ref uses the in-scope default namespace, falling back
// to the schema's target namespace (identity-constraints live in the target
// namespace).
func (c *compiler) resolveIDCNameQName(ctx context.Context, elem *helium.Element, ref string) (QName, bool) {
	// A malformed value (e.g. ":u") must be a fatal error, not silently resolved
	// as an unprefixed/default-namespace reference (strings.Cut would yield an
	// empty prefix that bypasses the unbound-prefix check below).
	if err := validateQName(ref); err != nil {
		c.reportInvalidQNameValue(ctx, elem, ref)
		return QName{}, true
	}
	if prefix, local, found := strings.Cut(ref, ":"); found {
		ns := lookupNS(elem, prefix)
		if ns == "" && prefix != "" {
			c.reportUnboundQNamePrefix(ctx, elem, ref, prefix)
			return QName{}, true
		}
		if c.rejectDeprecatedDatatypeNamespace(ctx, elem, ref, ns) {
			return QName{}, true
		}
		return QName{Local: local, NS: ns}, false
	}
	ns := c.schema.targetNamespace
	if defNS := lookupNS(elem, ""); defNS != "" {
		ns = defNS
	}
	if c.rejectDeprecatedDatatypeNamespace(ctx, elem, ref, ns) {
		return QName{}, true
	}
	return QName{Local: ref, NS: ns}, false
}

// reportIDCRefConflict reports a fatal schema error for an identity-constraint
// that uses @ref together with a companion (name/selector/field/refer) that the
// ref form forbids.
func (c *compiler) reportIDCRefConflict(ctx context.Context, source string, line int, xsdElem, companion string) {
	if source == "" {
		return
	}
	msg := fmt.Sprintf("An identity-constraint with a 'ref' attribute must not also specify '%s'.", companion)
	c.schemaError(ctx, schemaParserErrorAttr(source, line, xsdElem, xsdElem, attrRef, msg))
}

// reportIDCXPathError reports a malformed identity-constraint selector/field
// XPath as a fatal schema compilation error. kind is elemSelector or elemField.
func (c *compiler) reportIDCXPathError(ctx context.Context, kind string, line int, xpath string, cause error) {
	if c.filename == "" {
		return
	}
	noun := "selector"
	if kind == elemField {
		noun = "field"
	}
	msg := fmt.Sprintf("The %s XPath '%s' is not a valid %s expression: %s.", noun, xpath, noun, cause)
	c.schemaError(ctx,
		schemaParserErrorAttr(c.filename, line, kind, kind, attrXPath, msg))
}

// compileIDCXPath compiles an identity-constraint selector/field XPath and
// additionally verifies it conforms to the restricted XPath subset that XSD
// (Structures 3.11.6) permits for selectors and fields. The full XPath 1.0
// grammar accepted by xpath1.Compile is broader than that subset, so a syntax
// check alone would wrongly accept expressions such as string/number literal
// steps, function calls, variable references, operators, or predicates that the
// subset forbids. allowAttribute is true for <field> (which may end in an
// attribute step) and false for <selector> (where the attribute axis is not
// permitted).
func compileIDCXPath(expr string, allowAttribute bool) (*xpath1.Expression, error) {
	compiled, err := xpath1.Compile(expr)
	if err != nil {
		return nil, err
	}
	// Re-parse to inspect the AST; the expression already compiled, so this
	// cannot fail. The subset gate only runs at schema-compile time.
	ast, err := xpath1.Parse(expr)
	if err != nil {
		return nil, err
	}
	if err := validateIDCXPathSubset(ast, allowAttribute); err != nil {
		return nil, err
	}
	return compiled, nil
}

// validateIDCXPathSubset reports an error if expr falls outside the XSD
// identity-constraint XPath subset. The subset is a union ('|') of relative
// location paths whose steps use only the child axis (with a name test), the
// self axis (the abbreviated '.'), the descendant-or-self step of the './/'
// prefix, and — for fields only — a trailing attribute step. Anything else
// (literals, function calls, variables, arithmetic/boolean operators, filter
// expressions, predicates, absolute paths) is rejected.
func validateIDCXPathSubset(expr xpath1.Expr, allowAttribute bool) error {
	switch e := expr.(type) {
	case xpath1.UnionExpr:
		if err := validateIDCXPathSubset(e.Left, allowAttribute); err != nil {
			return err
		}
		return validateIDCXPathSubset(e.Right, allowAttribute)
	case *xpath1.LocationPath:
		return validateIDCLocationPath(e, allowAttribute)
	case xpath1.LocationPath:
		return validateIDCLocationPath(&e, allowAttribute)
	default:
		return errors.New("the expression is outside the identity-constraint XPath subset")
	}
}

// validateIDCLocationPath checks a single location path against the IDC subset.
func validateIDCLocationPath(lp *xpath1.LocationPath, allowAttribute bool) error {
	if lp.Absolute {
		return errors.New("absolute location paths are not permitted")
	}
	last := len(lp.Steps) - 1
	for i, step := range lp.Steps {
		if len(step.Predicates) > 0 {
			return errors.New("predicates are not permitted")
		}
		switch step.Axis {
		case xpath1.AxisSelf, xpath1.AxisDescendantOrSelf:
			// Only the abbreviated '.' and './/' steps (axis::node()) are
			// allowed; a name or other node-type test on these axes is not.
			tt, ok := step.NodeTest.(xpath1.TypeTest)
			if !ok || tt.Type != xpath1.NodeTestNode {
				return fmt.Errorf("the %s axis is only permitted as the abbreviated '.' or './/' step", step.Axis)
			}
		case xpath1.AxisChild:
			if _, ok := step.NodeTest.(xpath1.NameTest); !ok {
				return errors.New("only name tests are permitted on the child axis")
			}
		case xpath1.AxisAttribute:
			if !allowAttribute {
				return errors.New("the attribute axis is not permitted in a selector")
			}
			if i != last {
				return errors.New("the attribute axis is only permitted in the final step")
			}
			if _, ok := step.NodeTest.(xpath1.NameTest); !ok {
				return errors.New("only name tests are permitted on the attribute axis")
			}
		default:
			return fmt.Errorf("the %s axis is not permitted", step.Axis)
		}
	}
	return nil
}
