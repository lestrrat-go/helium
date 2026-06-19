package xsd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
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

	minPresent := hasAttr(elem, attrMinOccurs)
	maxPresent := hasAttr(elem, attrMaxOccurs)

	minVal, minOK := 1, true
	if minPresent {
		v := getAttr(elem, attrMinOccurs)
		n, ok := parseNonNegativeOccurs(v, false)
		if !ok {
			minOK = false
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, xsdElem, attrMinOccurs,
				"'"+v+"' is not a valid value of the atomic type 'xs:nonNegativeInteger'."), helium.ErrorLevelFatal))
			c.errorCount++
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
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, xsdElem, attrMaxOccurs,
				"'"+v+"' is not a valid value of the union type 'xs:allNNI'."), helium.ErrorLevelFatal))
			c.errorCount++
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
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, xsdElem, attrMaxOccurs,
				"The value must be greater than or equal to 1."), helium.ErrorLevelFatal))
			c.errorCount++
		}
	}

	// minOccurs must not exceed maxOccurs (Unbounded is treated as +inf, so it
	// can never be exceeded). Suppress this when the ">= 1" rule already fired on
	// maxOccurs; libxml2 reports only the maxOccurs error there.
	if minPresent && maxPresent && minOK && maxOK && maxVal != Unbounded && !maxBelowOne && minVal > maxVal {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserErrorAttr(c.filename, line, local, xsdElem, attrMinOccurs,
			"The value must not be greater than the value of 'maxOccurs'."), helium.ErrorLevelFatal))
		c.errorCount++
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

func readProcessContents(elem *helium.Element) ProcessContentsKind {
	switch getAttr(elem, attrProcessContents) {
	case attrValLax:
		return ProcessLax
	case attrValSkip:
		return ProcessSkip
	default:
		return ProcessStrict
	}
}

func (c *compiler) readWildcard(_ context.Context, elem *helium.Element) *Wildcard {
	namespace := getAttr(elem, attrNamespace)
	if namespace == "" {
		namespace = WildcardNSAny
	}

	return &Wildcard{
		Namespace:       namespace,
		ProcessContents: readProcessContents(elem),
		TargetNS:        c.schema.targetNamespace,
	}
}

func (c *compiler) readElementDecl(ctx context.Context, elem *helium.Element, opts elementDeclReadOptions) (*ElementDecl, error) {
	decl := &ElementDecl{
		Name:      QName{Local: opts.name, NS: opts.namespace},
		MinOccurs: opts.minOccurs,
		MaxOccurs: opts.maxOccurs,
		Nillable:  c.readBooleanAttr(ctx, elem, attrNillable),
	}

	if opts.allowAbstract {
		decl.Abstract = getAttr(elem, attrAbstract) == attrValTrue
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
	return decl, nil
}

// readBooleanAttr reads a schema-side xs:boolean attribute (e.g. nillable),
// applying whitespace-collapse lexical rules (true/false/1/0). An absent
// attribute is false. An invalid lexical is reported as a schema parser error
// and treated as false.
func (c *compiler) readBooleanAttr(ctx context.Context, elem *helium.Element, attr string) bool {
	if !hasAttr(elem, attr) {
		return false
	}
	v := normalizeWhiteSpace(getAttr(elem, attr), "collapse")
	switch v {
	case "true", "1":
		return true
	case "false", "0":
		return false
	}
	msg := fmt.Sprintf("'%s' is not a valid value of the atomic type 'xs:boolean'.", v)
	c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserErrorAttr(c.filename, elem.Line(),
		elem.LocalName(), elemElement, attr, msg), helium.ErrorLevelFatal))
	c.errorCount++
	return false
}

func (c *compiler) readElementType(ctx context.Context, elem *helium.Element, decl *ElementDecl, sourceName string) error {
	typeRef := getAttr(elem, attrType)
	if typeRef != "" {
		qn := c.resolveQName(ctx, elem, typeRef)
		c.elemRefs[decl] = qn
		c.markChameleonEligible(decl, elem, typeRef)
		c.elemRefSources[decl] = elemRefSource{elemName: sourceName, line: elem.Line()}
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
		decl.Type = c.schema.types[QName{Local: "anyType", NS: lexicon.NamespaceXSD}]
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
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, ce.Line(),
					ce.LocalName(), "attribute", err.Error()), helium.ErrorLevelFatal))
				c.errorCount++
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
	c.attrUseSources[au] = attrConstraintSource{
		line:  elem.Line(),
		local: opts.name.Local,
	}
	au.Default, au.Fixed = readDefaultOrFixed(elem)
	if au.Fixed != nil {
		au.FixedNS = collectNSContext(elem)
	}
	// Record source info so the default/fixed constraint value can be validated
	// against the attribute's declared simple type once type refs are resolved.
	if au.Default != nil || au.Fixed != nil {
		c.attrUseConstraintSources[au] = attrConstraintSource{
			line:  elem.Line(),
			local: opts.name.Local,
			nsMap: collectNSContext(elem),
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

	c.globalElemSources[decl] = elemRefSource{elemName: name, line: elem.Line()}
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
	c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(source, elem.Line(),
		elem.LocalName(), component,
		kind+" "+qnDisplay+" does already exist."), helium.ErrorLevelFatal))
	c.errorCount++
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
		c.elemRefSources[edecl] = elemRefSource{elemName: elem.LocalName(), line: elem.Line()}
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
		if getAttr(elem, attrUse) == attrValRequired {
			au.Required = true
		}
		if hasAttr(elem, attrDefault) {
			v := getAttr(elem, attrDefault)
			au.Default = &v
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
				line:  elem.Line(),
				local: qn.Local,
				nsMap: collectNSContext(elem),
			}
		}
		c.attrRefs[au] = qn
		return au
	}

	name := getAttr(elem, attrName)
	// Determine attribute namespace based on form and attributeFormDefault.
	attrNS := ""
	form := getAttr(elem, attrForm)
	if form == attrValQualified || (form == "" && c.schema.attrFormQualified) {
		attrNS = c.schema.targetNamespace
	}
	return c.readAttributeUseDecl(ctx, elem, attrUseReadOptions{
		name:       QName{Local: name, NS: attrNS},
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
	if name == "" {
		return nil
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
			selectorLine = ce.Line()
		case isXSDElement(ce, elemField):
			idc.Fields = append(idc.Fields, getAttr(ce, attrXPath))
			fieldLines = append(fieldLines, ce.Line())
		}
	}

	// Pre-compile selector XPath expression. A malformed selector XPath is a
	// fatal schema error: leaving SelectorExpr nil would silently disable the
	// whole constraint (the field-level uniqueness/keyref checks would never
	// run), so an invalid schema must fail to compile rather than validate
	// documents as if no constraint were present.
	if idc.Selector != "" {
		compiled, err := xpath1.Compile(idc.Selector)
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
		compiled, err := xpath1.Compile(f)
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
	if prefix, local, found := strings.Cut(refer, ":"); found {
		ns := lookupNS(elem, prefix)
		if ns == "" && prefix != "" {
			msg := fmt.Sprintf("The keyref identity-constraint '%s' has a 'refer' attribute '%s' whose namespace prefix '%s' is not bound.", idc.Name, refer, prefix)
			if c.filename != "" {
				c.errorHandler.Handle(ctx, helium.NewLeveledError(
					schemaParserErrorAttr(c.filename, idc.Line, elemKeyRef, elemKeyRef, attrRefer, msg),
					helium.ErrorLevelFatal))
				c.errorCount++
			}
			return QName{}, true
		}
		return QName{Local: local, NS: ns}, false
	}
	// Unprefixed: use the in-scope default namespace, else the target namespace.
	ns := c.schema.targetNamespace
	if defNS := lookupNS(elem, ""); defNS != "" {
		ns = defNS
	}
	return QName{Local: refer, NS: ns}, false
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
	c.errorHandler.Handle(ctx, helium.NewLeveledError(
		schemaParserErrorAttr(c.filename, line, kind, kind, attrXPath, msg),
		helium.ErrorLevelFatal))
	c.errorCount++
}
