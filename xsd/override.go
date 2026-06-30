package xsd

import (
	"context"
	"fmt"
	"maps"
	"sort"
	"strings"

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
//
// State threading (fixes for the OVR-001/OVR-002 review findings): the "active
// override set" is passed DOWN by value as a fresh map at each nested xs:override
// (BRANCH-LOCAL), so a nested override's own children can never leak into the
// suppression set of a LATER SIBLING include/override target. Each xs:override's
// own matched children are REGISTERED IMMEDIATELY, while the document that DECLARES
// them is still the active compiler context (form/block/final defaults,
// xpathDefaultNamespace, base URI, diagnostic file) — never deferred to an outer
// context. overrideLoadTarget returns the subset of the active set that matched
// somewhere in its closure so the OWNER of each key (the level that declared it)
// can decide registration in its own context.

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

// overrideChildEntry is one replacement component declared by an xs:override,
// retaining declaration order for deterministic, in-context registration.
type overrideChildEntry struct {
	key  overrideKey
	elem *helium.Element
}

// overrideMatchCtx records the per-DOCUMENT default state of the document where a
// matched (suppressed) component was DECLARED, so its replacement is registered
// under THAT document's @defaultAttributes AND <xs:defaultOpenContent> (§4.2.5 — a
// replacement is governed by the document where the matched component was declared,
// which for a nested-include target is the included document, NOT the direct
// override target).
type overrideMatchCtx struct {
	defAttrs schemaDefaultAttrsState
	oc       *OpenContent
}

// schemaDefaultAttrsState captures a schema document's XSD 1.1
// @defaultAttributes resolution: the referenced attribute-group QName, whether
// the attribute was present and well-formed, and its source. An xs:override
// replacement component is, per §4.2.5, copied into the TARGET (overridden)
// document, so the TARGET document's @defaultAttributes — not the overriding
// document's — governs it; overrideLoadTarget captures the target's state and the
// owner level reapplies it when registering the replacement.
type schemaDefaultAttrsState struct {
	qn  QName
	set bool
	src attrGroupRefUseSource
}

// schemaDefaultAttrs reads the compiler's current schema-level default-attribute
// state.
func (c *compiler) schemaDefaultAttrs() schemaDefaultAttrsState {
	return schemaDefaultAttrsState{
		qn:  c.schema.defaultAttributes,
		set: c.schema.defaultAttrsSet,
		src: c.schema.defaultAttrsSrc,
	}
}

// applySchemaDefaultAttrs sets the compiler's current schema-level
// default-attribute state, so a subsequently parsed complex type records the
// implicit default attribute-group reference (or none) of the governing document.
func (c *compiler) applySchemaDefaultAttrs(s schemaDefaultAttrsState) {
	c.schema.defaultAttributes = s.qn
	c.schema.defaultAttrsSet = s.set
	c.schema.defaultAttrsSrc = s.src
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

// loadOverride handles a top-level <xs:override schemaLocation="..."> element. The
// override children belong to the CURRENT (overriding) schema document, so they
// are collected and — once the closure determines which of them matched — REGISTERED
// HERE, while the overriding document's context is active.
func (c *compiler) loadOverride(ctx context.Context, location string, overrideElem *helium.Element) error {
	locals := c.collectOverrideChildren(ctx, overrideElem, nil)
	active := activeFromEntries(nil, locals)

	matched, err := c.overrideLoadTarget(ctx, location, overrideElem, active)
	if err != nil {
		return err
	}

	// Each matched key carries the per-document @defaultAttributes / default open
	// content of the document where its component was DECLARED (§4.2.5); the
	// overriding document's defaults must not govern the replacements, and a
	// replacement matched in a nested INCLUDED document uses that document's defaults
	// — registerMatchedChildren applies each key's recorded context.
	return c.registerMatchedChildren(ctx, locals, matched)
}

// collectOverrideChildren returns the replacement components declared directly on
// overrideElem, in declaration order. A child whose key duplicates another child
// of the SAME xs:override is a fatal schema error (W3C over021). A child whose key
// is already present in the inherited active set (an OUTER override provided it) is
// SHADOWED and dropped — the outer component wins (W3C over009). It does NOT mutate
// active; the caller derives the branch-local active set via activeFromEntries.
func (c *compiler) collectOverrideChildren(ctx context.Context, overrideElem *helium.Element, active map[overrideKey]*helium.Element) []overrideChildEntry {
	localSeen := make(map[overrideKey]struct{})
	var entries []overrideChildEntry
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
		if _, shadowed := active[key]; shadowed {
			// Shadowed by an enclosing (outer) override; the outer child wins.
			continue
		}
		entries = append(entries, overrideChildEntry{key: key, elem: elem})
	}
	return entries
}

// activeFromEntries returns a NEW active override set = inherited (outer) keys plus
// this level's own entries. The fresh map is the branch-local set handed to the
// target's closure; sibling traversal keeps its own inherited map untouched, so a
// nested override's children cannot leak into a sibling target (OVR-001).
func activeFromEntries(inherited map[overrideKey]*helium.Element, entries []overrideChildEntry) map[overrideKey]*helium.Element {
	out := make(map[overrideKey]*helium.Element, len(inherited)+len(entries))
	maps.Copy(out, inherited)
	for _, e := range entries {
		out[e.key] = e.elem
	}
	return out
}

// overrideActiveFingerprint builds a deterministic key for the active override
// set: the sorted (symbol space, namespace, local name, replacement-element
// identity) of every entry. Element identity (`%p`) is included so two active
// sets sharing the same names but DIFFERENT replacement definitions are
// distinguished — otherwise two distinct overrides of one document with different
// child definitions would wrongly dedup. Stable within a single compilation,
// which is all the visitation key needs.
func overrideActiveFingerprint(active map[overrideKey]*helium.Element) string {
	if len(active) == 0 {
		return ""
	}
	parts := make([]string, 0, len(active))
	for k, e := range active {
		parts = append(parts, fmt.Sprintf("%d|%s|%s|%p", k.sym, k.qn.NS, k.qn.Local, e))
	}
	sort.Strings(parts)
	return strings.Join(parts, ";")
}

// registerMatchedChildren registers each entry whose key actually matched a
// component in the closure; an UNMATCHED override child is DROPPED, never added as
// a new component. This is DELIBERATE and conformance-verified, NOT the literal
// §4.2.5 reading (which appears to add all override children unconditionally): W3C
// test over026 declares an override child `zonedDate` that matches nothing AND a
// sibling `para` that references it, and expects the schema to be INVALID precisely
// because the unmatched `zonedDate` is "ignored", making the reference dangling.
// over003/over007 annotate their unmatched children as "ignored" too. Registering
// unmatched children instead makes over026 (and only over026) FAIL — verified
// empirically — so do NOT "fix" this to register them. A dangling reference to a
// dropped child is therefore a schema error, as intended. Called while the
// document that DECLARES these children is the active compiler context.
func (c *compiler) registerMatchedChildren(ctx context.Context, entries []overrideChildEntry, matched map[overrideKey]overrideMatchCtx) error {
	for _, e := range entries {
		mctx, ok := matched[e.key]
		if !ok {
			continue
		}
		// Register under the DECLARING document's per-document @defaultAttributes AND
		// default open content (§4.2.5) — which for a replacement matched in a nested
		// included document is that included document's, not the direct target's. Only
		// these two defaults are swapped; the rest of the context (form/block/final
		// defaults, xpathDefaultNamespace, base URI, includeFile) stays the declaring
		// OVERRIDE document's, which is the caller's active context.
		savedDefAttrs := c.schemaDefaultAttrs()
		savedOC := c.defaultOpenContent
		c.applySchemaDefaultAttrs(mctx.defAttrs)
		c.defaultOpenContent = mctx.oc
		err := c.registerOverrideChild(ctx, e.key, e.elem)
		c.applySchemaDefaultAttrs(savedDefAttrs)
		c.defaultOpenContent = savedOC
		if err != nil {
			return err
		}
	}
	return nil
}

// registerOverrideChild registers a single matched override child via the normal
// named-component parser. The replaced component was suppressed during loading, so
// the slot is free and no spurious duplicate is reported. Notation overrides are
// not modeled (notations are not compiled), so a matched notation child is a no-op.
func (c *compiler) registerOverrideChild(ctx context.Context, key overrideKey, elem *helium.Element) error {
	switch key.sym {
	case overrideSymType:
		if isXSDElement(elem, elemComplexType) {
			return c.parseNamedComplexType(ctx, elem)
		}
		return c.parseNamedSimpleType(ctx, elem)
	case overrideSymElement:
		return c.parseGlobalElement(ctx, elem)
	case overrideSymAttribute:
		c.parseGlobalAttribute(ctx, elem)
	case overrideSymGroup:
		return c.parseNamedGroup(ctx, elem)
	case overrideSymAttrGroup:
		return c.parseNamedAttributeGroup(ctx, elem)
	case overrideSymNotation:
		// Notations have no full component model, but their NAMES must be collected
		// (mirroring parseSchemaChildren) so an xs:NOTATION-restriction enumeration
		// can resolve against a declared notation (W3C over015): the override child
		// notation replaces the suppressed target notation in c.notations.
		c.collectNotation(elem)
	}
	return nil
}

// collectNotation records an <xs:notation>'s name in c.notations (in the current
// document's target namespace), matching parseSchemaChildren's notation handling.
func (c *compiler) collectNotation(elem *helium.Element) {
	if name := getAttr(elem, attrName); name != "" {
		c.notations[QName{Local: name, NS: c.schema.targetNamespace}] = struct{}{}
	}
}

// reportOverrideIncludeConflict emits the fatal error for a schema document that
// is BOTH pulled in by a plain xs:include/xs:redefine AND transformed by an
// xs:override in the same assembly. kind is the XSD element local name of the
// element being processed (override/include/redefine) so the diagnostic cites the
// right source construct regardless of which side discovered the conflict.
func (c *compiler) reportOverrideIncludeConflict(ctx context.Context, srcElem *helium.Element, location, kind string) {
	displayLoc := location
	if c.filename != "" {
		displayLoc = schemaDisplayLoc(c.filename, location)
	}
	c.schemaError(ctx, schemaParserError(c.diagSource(), srcElem.Line(), srcElem.LocalName(), kind,
		"The schema document '"+displayLoc+"' is both included/redefined and overridden; a document cannot be pulled in by xs:include/xs:redefine and transformed by xs:override in the same schema."))
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

// overrideLoadTarget loads the referenced schema document and applies the active
// override set to its top-level components (suppressing replaced ones), then
// recurses into the document's own include/override closure with the SAME active
// set. It returns the subset of active keys that matched a component anywhere in
// this closure (this document plus its nested include/override targets). It does
// NOT register override children — that is the OWNER level's job, in its own
// context. Mirrors loadInclude's per-document form/default/chameleon handling.
func (c *compiler) overrideLoadTarget(ctx context.Context, location string, srcElem *helium.Element, active map[overrideKey]*helium.Element) (map[overrideKey]overrideMatchCtx, error) {
	matched := make(map[overrideKey]overrideMatchCtx)
	var targetDefAttrs schemaDefaultAttrsState
	// §4.2.5: the override children are copied INTO the transformed target document,
	// so the TARGET document's <xs:defaultOpenContent> — not the overriding
	// document's — governs the replacement components. Captured below and returned
	// for the owner to apply when it registers them.
	var targetOC *OpenContent

	path, err := validateSchemaPath(c.baseDir, location)
	if err != nil {
		return matched, fmt.Errorf("xsd: failed to load override %q: %w", location, err)
	}

	// Back-edge to the overriding ROOT: terminate WITHOUT re-loading/re-registering
	// it (the root's own components are owned by the top-level compile). Checked
	// before everything else so a circular override pointing at the root (over023)
	// never re-parses it.
	if path == c.rootKey {
		return matched, nil
	}

	// include+override conflict: the target was already pulled in by a PLAIN
	// xs:include/xs:redefine. The override transform produces a DISTINCT constituent
	// whose components would collide with the plain-included originals (§4.2.5/§F;
	// duplicate top-level components). This is a fatal schema error — not a silent
	// no-op — so report it and stop the override.
	if _, included := c.includeVisited[path]; included {
		c.reportOverrideIncludeConflict(ctx, srcElem, location, elemOverride)
		return matched, nil
	}

	// Override-cycle / diamond termination is keyed by (path, active-override-set):
	// the SAME document reached with the SAME active set is a true cycle/diamond and
	// terminates here, but the same document reached with a DIFFERENT active set is a
	// DISTINCT transformed document and must be loaded again — keying by path alone
	// would OVER-terminate and silently drop a sibling override's replacements (a
	// genuine collision then surfaces via the duplicate-component check, as it
	// should). The path-level overridePaths set still records the load for the
	// symmetric include+override conflict check in loadInclude/loadRedefine.
	vkey := path + "\x00" + overrideActiveFingerprint(active)
	if _, seen := c.overrideVisited[vkey]; seen {
		return matched, nil
	}
	if c.overrideVisited == nil {
		c.overrideVisited = make(map[string]struct{})
	}
	if c.overridePaths == nil {
		c.overridePaths = make(map[string]struct{})
	}
	c.overrideVisited[vkey] = struct{}{}
	c.overridePaths[path] = struct{}{}

	data, err := c.readNestedSchema(path)
	if err != nil {
		return matched, fmt.Errorf("xsd: failed to load override %q: %w", location, err)
	}

	doc, err := c.parse(ctx, data)
	if err != nil {
		return matched, fmt.Errorf("xsd: failed to parse override %q: %w", location, err)
	}

	incRoot := findDocumentElement(doc)
	if incRoot == nil || !isXSDElement(incRoot, elemSchema) {
		return matched, fmt.Errorf("xsd: overridden document %q is not an xs:schema", location)
	}

	// Conditional inclusion runs per document, before the targetNamespace check.
	savedIncludeFileVC := c.includeFile
	if c.filename != "" {
		c.includeFile = schemaDisplayLoc(c.filename, location)
	}
	rootExcluded := c.applyConditionalInclusion(ctx, incRoot)
	c.includeFile = savedIncludeFileVC
	if rootExcluded {
		return matched, nil
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
		return matched, nil
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
	savedDefAttrs := c.schemaDefaultAttrs()
	savedDefaultOpenContent := c.defaultOpenContent
	c.schema.elemFormQualified = getAttr(incRoot, attrElementFormDefault) == attrValQualified
	c.schema.attrFormQualified = getAttr(incRoot, attrAttributeFormDefault) == attrValQualified
	c.schema.blockDefault = parseBlockFlags(getAttr(incRoot, attrBlockDefault))
	c.schema.finalDefault = parseFinalFlags(getAttr(incRoot, attrFinalDefault))
	c.schemaXPathDefaultNS = ""
	// The overridden document's SURVIVING components AND its override REPLACEMENT
	// children both use the TARGET document's own default open content (per-document,
	// §4.2.5 — the replacements are copied INTO the target). targetOC is returned so
	// the owner applies it when registering the replacements.
	targetOC = c.readDefaultOpenContent(ctx, incRoot)
	c.defaultOpenContent = targetOC
	if c.version == Version11 {
		c.schemaXPathDefaultNS = resolveXPathDefaultNSToken(incRoot, getAttr(incRoot, attrXPathDefaultNS), c.schema.targetNamespace)
	}
	c.schemaBaseURI = path
	c.xpathDefaultNSSet = hasAttr(incRoot, attrXPathDefaultNamespace)
	if c.filename != "" {
		c.includeFile = schemaDisplayLoc(c.filename, location)
	}
	// §4.2.5: the override children are copied INTO the transformed target
	// document, so the TARGET document's @defaultAttributes — not the overriding
	// document's — governs both the surviving and the replacement complex types.
	// Apply it for the duration of this document's parse; the owner level reapplies
	// targetDefAttrs when it registers the replacement children (which happens after
	// restore()).
	qn, set, src := c.resolveSchemaDefaultAttributes(ctx, incRoot)
	targetDefAttrs = schemaDefaultAttrsState{qn: qn, set: set, src: src}
	c.applySchemaDefaultAttrs(targetDefAttrs)
	if set {
		// Queue the unresolved-attribute-group check for the TARGET document's
		// @defaultAttributes, matching readSchemaDefaultAttributes. The per-type
		// implicit ref recorded by parseComplexType silently skips an unresolved
		// group, so without this an override target declaring
		// defaultAttributes="t:missing" would compile clean. The source was captured
		// while c.includeFile points at the target, so checkSchemaDefaultAttributes
		// attributes the diagnostic to the target document.
		c.schemaDefaultAttrRefs = append(c.schemaDefaultAttrRefs, schemaDefaultAttrRef{qn: qn, src: src})
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
		c.applySchemaDefaultAttrs(savedDefAttrs)
		c.defaultOpenContent = savedDefaultOpenContent
	}

	// Parse the surviving (non-replaced) top-level components, recording which
	// active keys this document's own components matched.
	if err := c.overrideParseTargetChildren(ctx, incRoot, active, matched); err != nil {
		restore()
		return matched, err
	}

	// Recurse into the referenced document's own include/override/import/redefine,
	// cascading the active set; merge nested matches (always a subset of active).
	nestedMatched, nerr := c.overrideProcessNested(ctx, incRoot, path, location, active)
	maps.Copy(matched, nestedMatched)
	restore()
	return matched, nerr
}

// overrideParseTargetChildren parses the top-level children of an overridden
// document, dispatching exactly like parseSchemaChildren EXCEPT that a named
// component whose key is in the active override set is SUPPRESSED (skipped, and
// recorded in matched so the owner can register the replacement). xs:include/
// xs:import/xs:override/xs:redefine/xs:annotation are skipped here; they are
// handled by overrideProcessNested.
func (c *compiler) overrideParseTargetChildren(ctx context.Context, root *helium.Element, active map[overrideKey]*helium.Element, matched map[overrideKey]overrideMatchCtx) error {
	for child := range helium.Children(root) {
		if child.Type() != helium.ElementNode {
			continue
		}
		elem, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if key, named := c.overrideChildKey(elem); named {
			if _, overridden := active[key]; overridden {
				// Record the DECLARING (this) document's per-document defaults so the
				// replacement is registered under THEM (§4.2.5). c.defaultOpenContent /
				// c.schemaDefaultAttrs() are this target document's at suppression time.
				matched[key] = overrideMatchCtx{defAttrs: c.schemaDefaultAttrs(), oc: c.defaultOpenContent}
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
		case isXSDElement(elem, elemNotation):
			// A target notation NOT overridden is kept; collect its name so an
			// xs:NOTATION-restriction enumeration can resolve against it (mirroring
			// parseSchemaChildren). An overridden one was suppressed above.
			c.collectNotation(elem)
		}
	}
	return nil
}

// overrideProcessNested processes the include/override/import/redefine children of
// an overridden document, cascading the active override set. A nested xs:include
// is treated as an xs:override carrying the SAME active set (the override
// transformation is transitive through includes). A nested xs:override derives a
// BRANCH-LOCAL active set (inherited ∪ its own children, outer-precedence) for its
// target, registers ITS OWN matched children HERE — while this (the declaring)
// document's context is active — and never mutates the inherited set, so siblings
// are unaffected. xs:import and xs:redefine are processed normally — imports are
// not transformed by override (W3C over025/029), and no Override test combines
// override with redefine. References resolve relative to the overridden document,
// so baseDir/filename are switched as in processNestedIncludes, behind the same
// include-depth guard. Returns the subset of the INHERITED active set that matched
// somewhere in the nested closure, for the caller to propagate upward.
func (c *compiler) overrideProcessNested(ctx context.Context, incRoot *helium.Element, path, location string, active map[overrideKey]*helium.Element) (map[overrideKey]overrideMatchCtx, error) {
	nestedMatched := make(map[overrideKey]overrideMatchCtx)
	if c.includeDepth >= c.maxIncludeDepth {
		return nestedMatched, fmt.Errorf("%w (limit=%d, location=%q)", errIncludeDepthExceeded, c.maxIncludeDepth, location)
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
			// the override: load it as an override target carrying the same active
			// set. Every matched key is (by construction) in the inherited set.
			var m map[overrideKey]overrideMatchCtx
			m, err = c.overrideLoadTarget(ctx, loc, elem, active)
			maps.Copy(nestedMatched, m)
		case isXSDElement(elem, elemOverride):
			loc := getAttr(elem, attrSchemaLocation)
			if loc == "" {
				continue
			}
			if !c.recordOverrideTarget(ctx, elem, loc, overrideSeen) {
				continue
			}
			// A nested xs:override contributes its OWN children. Build a branch-local
			// active set (outer-precedence) so these children cannot leak into a
			// sibling target's suppression set.
			locals := c.collectOverrideChildren(ctx, elem, active)
			childActive := activeFromEntries(active, locals)
			var m map[overrideKey]overrideMatchCtx
			m, err = c.overrideLoadTarget(ctx, loc, elem, childActive)
			if err != nil {
				break
			}
			// Register THIS override's own matched children NOW, while the declaring
			// document's context (form/block/final defaults, xpathDefaultNamespace,
			// base URI, includeFile) is still active — registerMatchedChildren applies
			// each matched key's recorded @defaultAttributes AND default open content
			// (§4.2.5: the replacement uses the document where its component was
			// declared, which for a nested-include match is the included document).
			rerr := c.registerMatchedChildren(ctx, locals, m)
			if rerr != nil {
				err = rerr
				break
			}
			// Propagate only the INHERITED keys that matched up to the caller; the
			// local children were just registered and must not leak further.
			for k, v := range m {
				if _, inherited := active[k]; inherited {
					nestedMatched[k] = v
				}
			}
		case isXSDElement(elem, elemImport):
			// Shared with processIncludes so an import inside an overridden document
			// emits the same already-imported / load-failure warnings; only a fatal
			// load condition aborts.
			err = c.processImport(ctx, elem)
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
	return nestedMatched, err
}
