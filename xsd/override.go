package xsd

import (
	"context"
	"fmt"

	helium "github.com/lestrrat-go/helium"
)

// xs:override (XSD 1.1, W3C XML Schema 1.1 Part 1 §4.2.5 / §F) performs WHOLESALE
// REPLACEMENT of top-level components in a referenced schema document. Unlike
// xs:redefine (which restricts/extends the SAME component), an xs:override child
// replaces ANY top-level component of the same (expanded-name, symbol space),
// regardless of the kind of derivation. Components not matched by an override
// child are kept; an override child that matches nothing is dropped.
//
// The transformation is RECURSIVE/transitive: the referenced document's own
// xs:include/xs:override are themselves transformed by the SAME (cascaded) set of
// override children, with the OUTER override winning on a name+symbol-space
// collision. A nested xs:include is treated as an xs:override carrying the active
// override set. Chameleon (no-targetNamespace) referenced documents adopt the
// overriding schema's target namespace, exactly like xs:include.

// overrideSymbol identifies the XSD symbol space an override child / component
// occupies. simpleType and complexType share ONE type-definition symbol space, so
// a simpleType override may replace a complexType of the same name (and vice
// versa) — see W3C Override test over013.
type overrideSymbol int

const (
	overrideSymType overrideSymbol = iota // simpleType + complexType
	overrideSymElement
	overrideSymAttribute
	overrideSymGroup
	overrideSymAttrGroup
	overrideSymNotation
)

// overrideKey is the (symbol space, expanded name) identity used to match an
// override child against a top-level component in the referenced document.
type overrideKey struct {
	sym overrideSymbol
	qn  QName
}

// overrideRun threads the state of one top-level xs:override resolution through
// the (possibly recursive) document closure. children is the accumulated set of
// replacement components, OUTER-precedence (an enclosing override's child is never
// overwritten by a nested one). order preserves declaration order for
// deterministic registration. matched records which keys actually replaced a
// component somewhere in the closure; only matched children are registered, the
// rest are dropped.
type overrideRun struct {
	children map[overrideKey]*helium.Element
	order    []overrideKey
	matched  map[overrideKey]struct{}
}

// overrideChildKey returns the (symbol space, name) identity of a potential
// override child / top-level component, with the name resolved in the current
// schema's target namespace (which a chameleon referenced document adopts). The
// second result is false for an element that is not a named top-level component
// (annotation, include, import, override, redefine, an unnamed declaration).
func (c *compiler) overrideChildKey(elem *helium.Element) (overrideKey, bool) {
	name := getAttr(elem, attrName)
	if name == "" {
		return overrideKey{}, false
	}
	qn := QName{Local: name, NS: c.schema.targetNamespace}
	switch {
	case isXSDElement(elem, elemComplexType), isXSDElement(elem, elemSimpleType):
		return overrideKey{sym: overrideSymType, qn: qn}, true
	case isXSDElement(elem, elemElement):
		return overrideKey{sym: overrideSymElement, qn: qn}, true
	case isXSDElement(elem, elemAttribute):
		return overrideKey{sym: overrideSymAttribute, qn: qn}, true
	case isXSDElement(elem, elemGroup):
		return overrideKey{sym: overrideSymGroup, qn: qn}, true
	case isXSDElement(elem, elemAttributeGroup):
		return overrideKey{sym: overrideSymAttrGroup, qn: qn}, true
	case isXSDElement(elem, elemNotation):
		return overrideKey{sym: overrideSymNotation, qn: qn}, true
	}
	return overrideKey{}, false
}

// loadOverride handles a top-level <xs:override schemaLocation="..."> element. It
// builds the override run from the element's children, loads and transforms the
// referenced document closure (applying and cascading the run), then registers
// the override children that actually replaced a component.
func (c *compiler) loadOverride(ctx context.Context, location string, overrideElem *helium.Element) error {
	run := &overrideRun{
		children: make(map[overrideKey]*helium.Element),
		matched:  make(map[overrideKey]struct{}),
	}
	c.collectOverrideChildren(ctx, overrideElem, run)

	if err := c.overrideLoadTarget(ctx, location, overrideElem, run); err != nil {
		return err
	}

	return c.registerOverrideChildren(ctx, run)
}

// collectOverrideChildren records the override children of overrideElem into run.
// A child whose key duplicates another child of the SAME xs:override is a fatal
// schema error (W3C over021). A child whose key is already present in run (an
// OUTER override already provided it) is shadowed and dropped — the outer
// component wins (W3C over009).
func (c *compiler) collectOverrideChildren(ctx context.Context, overrideElem *helium.Element, run *overrideRun) {
	localSeen := make(map[overrideKey]struct{})
	for child := range helium.Children(overrideElem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		elem, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if isXSDElement(elem, elemAnnotation) {
			continue
		}
		key, ok := c.overrideChildKey(elem)
		if !ok {
			continue
		}
		if _, dup := localSeen[key]; dup {
			c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), elemOverride,
				fmt.Sprintf("An xs:override must not contain two children that override the same component '%s'.", key.qn.Local)))
			continue
		}
		localSeen[key] = struct{}{}
		if _, exists := run.children[key]; exists {
			// Shadowed by an enclosing (outer) override; the outer child wins.
			continue
		}
		run.children[key] = elem
		run.order = append(run.order, key)
	}
}

// recordOverrideTarget enforces that a single schema document does not contain
// two xs:override elements targeting the SAME document (W3C over022). seen is the
// per-document set of already-targeted resolved paths. It returns true if the
// override may proceed, false (after reporting a fatal error) on a duplicate. A
// path that fails to resolve is allowed through so the load attempt surfaces the
// I/O error.
func (c *compiler) recordOverrideTarget(ctx context.Context, srcElem *helium.Element, location string, seen map[string]struct{}) bool {
	path, err := validateSchemaPath(c.baseDir, location)
	if err != nil {
		return true
	}
	if _, dup := seen[path]; dup {
		displayLoc := location
		if c.filename != "" {
			displayLoc = schemaDisplayLoc(c.filename, location)
		}
		c.schemaError(ctx, schemaParserError(c.diagSource(), srcElem.Line(), srcElem.LocalName(), elemOverride,
			"The schema document '"+displayLoc+"' must not be overridden more than once."))
		return false
	}
	seen[path] = struct{}{}
	return true
}

// overrideLoadTarget loads the referenced schema document, applies the override
// run to its top-level components (suppressing replaced ones and recording
// matches), then recurses into the document's own include/override closure with
// the cascaded run. It mirrors loadInclude's per-document form/default/chameleon
// handling.
func (c *compiler) overrideLoadTarget(ctx context.Context, location string, srcElem *helium.Element, run *overrideRun) error {
	path, err := validateSchemaPath(c.baseDir, location)
	if err != nil {
		return fmt.Errorf("xsd: failed to load override %q: %w", location, err)
	}

	// A document already pulled in (by a prior include/override, OR a back-edge in
	// a circular override pointing at an ancestor) is not re-loaded: its components
	// were already contributed and the cascade terminates here (W3C over023/024).
	if _, seen := c.includeVisited[path]; seen {
		return nil
	}
	c.includeVisited[path] = struct{}{}

	data, err := c.readNestedSchema(path)
	if err != nil {
		return fmt.Errorf("xsd: failed to load override %q: %w", location, err)
	}

	doc, err := c.parse(ctx, data)
	if err != nil {
		return fmt.Errorf("xsd: failed to parse override %q: %w", location, err)
	}

	incRoot := findDocumentElement(doc)
	if incRoot == nil || !isXSDElement(incRoot, elemSchema) {
		return fmt.Errorf("xsd: overridden document %q is not an xs:schema", location)
	}

	// Conditional inclusion runs per document, before the targetNamespace check.
	savedIncludeFileVC := c.includeFile
	if c.filename != "" {
		c.includeFile = schemaDisplayLoc(c.filename, location)
	}
	rootExcluded := c.applyConditionalInclusion(ctx, incRoot)
	c.includeFile = savedIncludeFileVC
	if rootExcluded {
		return nil
	}

	// Target namespace compatibility: same rule as xs:include (W3C over016/017). A
	// referenced document with no targetNamespace is a chameleon and adopts the
	// overriding schema's namespace.
	incTargetNS := getAttr(incRoot, attrTargetNamespace)
	if incTargetNS != "" && incTargetNS != c.schema.targetNamespace {
		displayLoc := location
		if c.filename != "" {
			displayLoc = schemaDisplayLoc(c.filename, location)
		}
		c.schemaError(ctx, schemaParserError(c.filename, srcElem.Line(), srcElem.LocalName(), elemOverride,
			"The target namespace '"+incTargetNS+"' of the overridden schema '"+displayLoc+"' differs from '"+c.schema.targetNamespace+"' of the overriding schema."))
		return nil
	}

	// Per-document form/default/chameleon settings, mirroring loadInclude: the
	// referenced document's own elementFormDefault/attributeFormDefault/
	// blockDefault/finalDefault/xpathDefaultNamespace govern its declarations and
	// must NOT be inherited from the overriding schema; reset to spec defaults plus
	// this document's own values, then restore on exit.
	savedElemForm := c.schema.elemFormQualified
	savedAttrForm := c.schema.attrFormQualified
	savedBlockDefault := c.schema.blockDefault
	savedFinalDefault := c.schema.finalDefault
	savedIncludeFile := c.includeFile
	savedXPathDefaultNS := c.schemaXPathDefaultNS
	savedSchemaBaseURI := c.schemaBaseURI
	savedCTAXPathDefaultNSSet := c.xpathDefaultNSSet
	c.schema.elemFormQualified = getAttr(incRoot, attrElementFormDefault) == attrValQualified
	c.schema.attrFormQualified = getAttr(incRoot, attrAttributeFormDefault) == attrValQualified
	c.schema.blockDefault = parseBlockFlags(getAttr(incRoot, attrBlockDefault))
	c.schema.finalDefault = parseFinalFlags(getAttr(incRoot, attrFinalDefault))
	c.schemaXPathDefaultNS = ""
	if c.version == Version11 {
		c.schemaXPathDefaultNS = resolveXPathDefaultNSToken(incRoot, getAttr(incRoot, attrXPathDefaultNS), c.schema.targetNamespace)
	}
	c.schemaBaseURI = path
	c.xpathDefaultNSSet = hasAttr(incRoot, attrXPathDefaultNamespace)
	if c.filename != "" {
		c.includeFile = schemaDisplayLoc(c.filename, location)
	}

	restore := func() {
		c.schema.elemFormQualified = savedElemForm
		c.schema.attrFormQualified = savedAttrForm
		c.schema.blockDefault = savedBlockDefault
		c.schema.finalDefault = savedFinalDefault
		c.schemaXPathDefaultNS = savedXPathDefaultNS
		c.schemaBaseURI = savedSchemaBaseURI
		c.xpathDefaultNSSet = savedCTAXPathDefaultNSSet
		c.includeFile = savedIncludeFile
	}

	// Parse the surviving (non-replaced) top-level components.
	if err := c.overrideParseTargetChildren(ctx, incRoot, run); err != nil {
		restore()
		return err
	}

	// Recurse into the referenced document's own include/override/import/redefine,
	// cascading the override run.
	err = c.overrideProcessNested(ctx, incRoot, path, location, run)
	restore()
	return err
}

// overrideParseTargetChildren parses the top-level children of an overridden
// document, dispatching exactly like parseSchemaChildren EXCEPT that a named
// component whose key is overridden by the active run is SUPPRESSED (skipped, and
// recorded in run.matched so the corresponding override child will be
// registered). xs:include/xs:import/xs:override/xs:redefine/xs:annotation are
// skipped here; they are handled by overrideProcessNested.
func (c *compiler) overrideParseTargetChildren(ctx context.Context, root *helium.Element, run *overrideRun) error {
	for child := range helium.Children(root) {
		if child.Type() != helium.ElementNode {
			continue
		}
		elem, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if key, named := c.overrideChildKey(elem); named {
			if _, overridden := run.children[key]; overridden {
				run.matched[key] = struct{}{}
				continue
			}
		}
		switch {
		case isXSDElement(elem, elemElement):
			if err := c.parseGlobalElement(ctx, elem); err != nil {
				return err
			}
		case isXSDElement(elem, elemComplexType):
			if err := c.parseNamedComplexType(ctx, elem); err != nil {
				return err
			}
		case isXSDElement(elem, elemSimpleType):
			if err := c.parseNamedSimpleType(ctx, elem); err != nil {
				return err
			}
		case isXSDElement(elem, elemGroup):
			if err := c.parseNamedGroup(ctx, elem); err != nil {
				return err
			}
		case isXSDElement(elem, elemAttributeGroup):
			if err := c.parseNamedAttributeGroup(ctx, elem); err != nil {
				return err
			}
		case isXSDElement(elem, elemAttribute):
			c.parseGlobalAttribute(ctx, elem)
		}
	}
	return nil
}

// overrideProcessNested processes the include/override/import/redefine children of
// an overridden document, cascading the override run. A nested xs:include is
// treated as an xs:override carrying the active run (the override transformation
// is transitive through includes). A nested xs:override merges its own children
// into the run (outer-precedence) and recurses. xs:import and xs:redefine are
// processed normally — imports are not transformed by override (W3C over025/029),
// and no Override test combines override with redefine. References are resolved
// relative to the overridden document, so baseDir/filename are switched as in
// processNestedIncludes, behind the same include-depth guard.
func (c *compiler) overrideProcessNested(ctx context.Context, incRoot *helium.Element, path, location string, run *overrideRun) error {
	if c.includeDepth >= c.maxIncludeDepth {
		return fmt.Errorf("%w (limit=%d, location=%q)", errIncludeDepthExceeded, c.maxIncludeDepth, location)
	}
	savedBaseDir := c.baseDir
	savedFilename := c.filename
	c.baseDir = schemaBaseDir(path)
	if savedFilename != "" {
		c.filename = schemaDisplayLoc(savedFilename, location)
	}
	c.includeDepth++

	// Per-document override-target set (W3C over022): two xs:override elements in
	// the overridden document targeting the same document are rejected.
	overrideSeen := make(map[string]struct{})
	var err error
	for child := range helium.Children(incRoot) {
		if err != nil {
			break
		}
		if child.Type() != helium.ElementNode {
			continue
		}
		elem, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXSDElement(elem, elemInclude):
			loc := getAttr(elem, attrSchemaLocation)
			if loc == "" {
				continue
			}
			// An xs:include inside an overridden document is itself transformed by
			// the override: load it as an override target carrying the same run.
			err = c.overrideLoadTarget(ctx, loc, elem, run)
		case isXSDElement(elem, elemOverride):
			loc := getAttr(elem, attrSchemaLocation)
			if loc == "" {
				continue
			}
			if !c.recordOverrideTarget(ctx, elem, loc, overrideSeen) {
				continue
			}
			// A nested xs:override contributes its own children (outer-precedence)
			// and its target is loaded under the merged run.
			c.collectOverrideChildren(ctx, elem, run)
			err = c.overrideLoadTarget(ctx, loc, elem, run)
		case isXSDElement(elem, elemImport):
			loc := getAttr(elem, attrSchemaLocation)
			if loc == "" {
				continue
			}
			ns := getAttr(elem, attrNamespace)
			if prevLoc, seen := c.importedNS[ns]; seen {
				_ = prevLoc
				continue
			}
			if ierr := c.loadImport(ctx, loc, ns, elem); ierr != nil {
				if IsFatalSchemaLoad(ierr) {
					err = ierr
					break
				}
				continue
			}
			c.importedNS[ns] = loc
		case isXSDElement(elem, elemRedefine):
			loc := getAttr(elem, attrSchemaLocation)
			if loc == "" {
				continue
			}
			err = c.loadRedefine(ctx, loc, elem)
		}
	}

	c.includeDepth--
	c.baseDir = savedBaseDir
	c.filename = savedFilename
	return err
}

// registerOverrideChildren registers the override children that actually replaced
// a component in the referenced document closure (run.matched), in declaration
// order. An override child that matched nothing is dropped (W3C over003/over007/
// over026) — leaving a dangling reference to it a schema error, as intended.
// A replaced component was suppressed during loading, so its slot is free and the
// normal named-component parser registers the override child without a spurious
// duplicate report. Notation overrides are not modeled (notations are not
// compiled), so a matched notation child is a no-op.
func (c *compiler) registerOverrideChildren(ctx context.Context, run *overrideRun) error {
	for _, key := range run.order {
		if _, matched := run.matched[key]; !matched {
			continue
		}
		elem := run.children[key]
		switch key.sym {
		case overrideSymType:
			if isXSDElement(elem, elemComplexType) {
				if err := c.parseNamedComplexType(ctx, elem); err != nil {
					return err
				}
				continue
			}
			if err := c.parseNamedSimpleType(ctx, elem); err != nil {
				return err
			}
		case overrideSymElement:
			if err := c.parseGlobalElement(ctx, elem); err != nil {
				return err
			}
		case overrideSymAttribute:
			c.parseGlobalAttribute(ctx, elem)
		case overrideSymGroup:
			if err := c.parseNamedGroup(ctx, elem); err != nil {
				return err
			}
		case overrideSymAttrGroup:
			if err := c.parseNamedAttributeGroup(ctx, elem); err != nil {
				return err
			}
		case overrideSymNotation:
			// notations are not modeled by the compiler; nothing to register.
		}
	}
	return nil
}
