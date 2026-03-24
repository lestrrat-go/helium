package xslt3

import (
	"context"
	"errors"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

func (ec *execContext) execLiteralResultElement(ctx context.Context, inst *literalResultElement) error {
	// Use LocalName so that SetActiveNamespace doesn't double the prefix
	elemName := inst.LocalName
	if elemName == "" {
		elemName = inst.Name
	}
	// Create the element in the current output frame's document to avoid
	// cross-document node identity issues with type annotations.
	doc := ec.currentOutput().doc
	elem, err := doc.CreateElement(elemName)
	if err != nil {
		return err
	}

	// Declare namespaces (skip if parent already has the same declaration).
	// Declare the default namespace first for deterministic serialization order.
	if uri, ok := inst.Namespaces[""]; ok {
		if !ec.isNSDeclaredInScope("", uri) {
			if err := elem.DeclareNamespace("", uri); err != nil {
				return err
			}
		}
	}
	for prefix, uri := range inst.Namespaces {
		if prefix == "" {
			continue // already handled above
		}
		if !ec.isNSDeclaredInScope(prefix, uri) {
			if err := elem.DeclareNamespace(prefix, uri); err != nil {
				return err
			}
		}
	}

	// Set the element's own namespace
	prefixExcluded := false
	if inst.Namespace != "" {
		if err := elem.SetActiveNamespace(inst.Prefix, inst.Namespace); err != nil {
			return err
		}
		// Ensure the namespace declaration is present for serialization.
		// SetActiveNamespace only sets n.ns; we also need it in nsDefs.
		// However, if the prefix was excluded (not in inst.Namespaces), defer
		// declaration so that xsl:namespace instructions can claim the prefix.
		// Namespace fixup after body execution will assign a new prefix if needed.
		_, prefixInNS := inst.Namespaces[inst.Prefix]
		if inst.Prefix == "" {
			prefixInNS = true // default namespace is never "excluded" in this sense
		}
		if prefixInNS {
			if !hasNSDecl(elem, inst.Prefix, inst.Namespace) && !ec.isNSDeclaredInScope(inst.Prefix, inst.Namespace) {
				if err := elem.DeclareNamespace(inst.Prefix, inst.Namespace); err != nil {
					return err
				}
			}
		} else {
			prefixExcluded = true
		}
	} else if inst.Prefix == "" && ec.hasDefaultNSInScope() {
		// No namespace on this LRE but default namespace in scope — undeclare it
		if err := elem.DeclareNamespace("", ""); err != nil {
			return err
		}
	}

	out := ec.currentOutput()

	// Always add element to the DOM tree.
	if err := ec.addNode(elem); err != nil {
		return err
	}

	// For item-output-method (json/adaptive) at the principal output level,
	// also capture as a pending item for serialization. Use isItemOutputMethod()
	// which checks the current result-document method (handles both principal
	// and secondary output). temporaryOutputDepth ensures we don't capture
	// inside variable evaluation where captureItems is also set.
	// For adaptive output at the principal level, also capture the element
	// as a pending item so serializeAdaptiveItems can enumerate it.
	// Don't do this for json output (elements stay in DOM for json-node-output-method).
	// Don't double-capture when sequenceMode is true (addNode already captured).
	captureAsItem := ec.currentResultDocMethod == methodAdaptive && ec.temporaryOutputDepth == 0 && !out.sequenceMode

	// Execute body in element context with a new variable scope.
	// Temporarily disable sequenceMode so that children are added to this
	// element normally (not captured as separate items in the sequence).
	savedCurrent := out.current
	savedPrevAtomic := out.prevWasAtomic
	savedSeqMode := out.sequenceMode
	savedCapture := out.captureItems
	savedWherePop := out.wherePopulated
	out.current = elem
	out.prevWasAtomic = false
	out.sequenceMode = false
	out.captureItems = false
	// Clear wherePopulated inside the LRE body so that xsl:document
	// unwraps its children normally. The LRE element itself is the node
	// that xsl:where-populated will check for emptiness — preserving
	// document nodes inside it would leave orphaned document children
	// that are invisible to isPopulated and the serializer.
	out.wherePopulated = false
	ec.pushVarScope()

	// Override static base URI when the LRE carries xml:base
	savedBaseOverride := ec.staticBaseURIOverride
	if inst.StaticBaseURI != "" {
		ec.staticBaseURIOverride = inst.StaticBaseURI
	}

	// Propagate xpath-default-namespace from the LRE
	savedXPathDefaultNS := ec.xpathDefaultNS
	savedHasXPathDefaultNS := ec.hasXPathDefaultNS
	if inst.HasXPathDefaultNS {
		ec.xpathDefaultNS = inst.XPathDefaultNS
		ec.hasXPathDefaultNS = true
	}

	// Override default-validation scope (XSLT 3.0 §3.6)
	savedDefaultValidation := ec.defaultValidation
	if inst.DefaultValidation != "" {
		ec.defaultValidation = inst.DefaultValidation
	}

	defer func() {
		ec.defaultValidation = savedDefaultValidation
		ec.xpathDefaultNS = savedXPathDefaultNS
		ec.hasXPathDefaultNS = savedHasXPathDefaultNS
		ec.staticBaseURIOverride = savedBaseOverride
		ec.popVarScope()
		out.current = savedCurrent
		out.prevWasAtomic = savedPrevAtomic
		out.sequenceMode = savedSeqMode
		out.captureItems = savedCapture
		out.wherePopulated = savedWherePop
	}()

	// Apply attribute sets first, then LRE attributes override them
	// (per XSLT spec: LRE attributes take precedence over attribute sets).
	if len(inst.UseAttrSets) > 0 {
		if err := ec.applyAttributeSets(ctx, inst.UseAttrSets); err != nil {
			return err
		}
	}

	// Evaluate and set LRE attributes (after attribute sets so they override)
	for _, attr := range inst.Attrs {
		val, err := attr.Value.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		if attr.Namespace != "" {
			// Ensure the attribute's namespace is declared on the element
			if !hasNSDecl(elem, attr.Prefix, attr.Namespace) && !ec.isNSDeclaredInScope(attr.Prefix, attr.Namespace) {
				if err := elem.DeclareNamespace(attr.Prefix, attr.Namespace); err != nil {
					return err
				}
			}
			ns, err := ec.resultDoc.CreateNamespace(attr.Prefix, attr.Namespace)
			if err != nil {
				return err
			}
			// Use literal mode: avt evaluation produces plain text that
			// may contain & from resolved entities (e.g. &amp; -> &).
			// SetAttributeNS would re-parse those as entity references.
			elem.SetLiteralAttributeNS(attr.LocalName, val, ns)
		} else {
			// Use literal mode for the same reason as above.
			elem.SetLiteralAttribute(attr.Name, val)
		}
	}

	if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
		return err
	}

	// Namespace fixup: when the element's prefix was excluded (via
	// exclude-result-prefixes), an xsl:namespace instruction may have
	// claimed that prefix for a different URI. Ensure the element's own
	// namespace is declared, renaming its prefix if needed.
	if prefixExcluded && inst.Namespace != "" {
		ec.fixupExcludedElementNS(elem, inst.Prefix, inst.Namespace)
	}

	// inherit-namespaces="no": undeclare parent namespaces on direct
	// child elements so they do not inherit them via the DOM tree.
	if !inst.InheritNamespaces {
		undeclareInheritedNamespaces(elem)
	}

	// Type-based content validation (xsl:type on LRE).
	// XTTE1540: invalid content for the declared type.
	if inst.TypeName != "" {
		if err := ec.validateAndNormalizeElementContent(elem, inst.TypeName); err != nil {
			// Convert XTTE1510 to XTTE1540 for xsl:type on LRE.
			if xsltErr, ok := errors.AsType[*XSLTError](err); ok && xsltErr.Code == errCodeXTTE1510 {
				return dynamicError(errCodeXTTE1540,
					"element content does not match declared type %s: %v", inst.TypeName, xsltErr.Message)
			}
			return err
		}
		ec.annotateNode(elem, inst.TypeName)
		ec.annotateAttributesFromType(elem, inst.TypeName)
	}

	// Schema validation (xsl:validation on LRE, or default-validation).
	// Per XSLT spec, type and validation are mutually exclusive: when type
	// is specified, do not also apply default-validation.
	if inst.TypeName == "" {
		if v := ec.effectiveValidation(inst.Validation); v != "" {
			if err := ec.validateConstructedElement(ctx, elem, v); err != nil {
				return err
			}
		}
	}

	// Capture the fully-built element as a pending item for item serialization.
	if captureAsItem {
		out.pendingItems = append(out.pendingItems, xpath3.NodeItem{Node: elem})
		out.noteOutput()
	}

	return nil
}

// applyAttributeSets applies named attribute sets to the current output element.
// Each call starts a fresh cycle-detection scope so that body instructions
// (e.g., an LRE inside the attribute set that itself uses an attribute set)
// are allowed to re-enter without being flagged as cycles.
func (ec *execContext) applyAttributeSets(ctx context.Context, names []string) error {
	return ec.applyAttributeSetsGuarded(ctx, names, make(map[string]struct{}))
}

// applyAttributeSetsGuarded is the recursive core that tracks which attribute
// sets are currently being expanded via use-attribute-sets (defense in depth).
func (ec *execContext) applyAttributeSetsGuarded(ctx context.Context, names []string, active map[string]struct{}) error {
	for _, name := range names {
		asDef := ec.stylesheet.attributeSets[name]
		if asDef == nil {
			continue
		}
		if _, ok := active[name]; ok {
			return dynamicError(errCodeXTSE0720,
				"attribute-set %q has a circular use-attribute-sets reference (runtime)", name)
		}
		active[name] = struct{}{}
		// Process each part (same-named declaration) in document order.
		// Each part applies its own use-attribute-sets first, then its own attrs.
		for _, part := range asDef.Parts {
			// Set static base URI from this part's xml:base declaration
			savedBaseOverride := ec.staticBaseURIOverride
			if part.StaticBaseURI != "" {
				ec.staticBaseURIOverride = part.StaticBaseURI
			}
			if len(part.UseAttrSets) > 0 {
				if err := ec.applyAttributeSetsGuarded(ctx, part.UseAttrSets, active); err != nil {
					ec.staticBaseURIOverride = savedBaseOverride
					delete(active, name)
					return err
				}
			}
			// Execute the attribute instructions.
			// XSLT spec: only global variables/params are visible in
			// attribute set bodies, not template-local variables.
			savedLocalVars := ec.localVars
			ec.localVars = nil
			for _, inst := range part.Attrs {
				if err := ec.executeInstruction(ctx, inst); err != nil {
					ec.localVars = savedLocalVars
					ec.staticBaseURIOverride = savedBaseOverride
					delete(active, name)
					return err
				}
			}
			ec.localVars = savedLocalVars
			ec.staticBaseURIOverride = savedBaseOverride
		}
		delete(active, name)
	}
	return nil
}

// isNSDeclaredInScope checks if a namespace prefix→URI binding is already
// declared on an ancestor element in the current output tree.
func (ec *execContext) isNSDeclaredInScope(prefix, uri string) bool {
	out := ec.currentOutput()
	for node := out.current; node != nil; node = node.Parent() {
		elem, ok := node.(*helium.Element)
		if !ok {
			continue
		}
		for _, ns := range elem.Namespaces() {
			if ns.Prefix() == prefix {
				// Found a declaration for this prefix — it's in scope
				// only if the URI matches. If a different URI is bound,
				// the desired binding is shadowed.
				return ns.URI() == uri
			}
		}
		// Also check the element's own active namespace for unprefixed elements
		if prefix == "" && elem.Prefix() == "" && elem.URI() != "" {
			return elem.URI() == uri
		}
	}
	return false
}

// hasDefaultNSInScope returns true if there is a default namespace (xmlns="...")
// with a non-empty URI declared on any ancestor in the result tree.
func (ec *execContext) hasDefaultNSInScope() bool {
	out := ec.currentOutput()
	for node := out.current; node != nil; node = node.Parent() {
		elem, ok := node.(*helium.Element)
		if !ok {
			continue
		}
		// Check the element's own namespace (if default, i.e. no prefix)
		if elem.Prefix() == "" && elem.URI() != "" {
			return true
		}
		// Check namespace declarations
		for _, ns := range elem.Namespaces() {
			if ns.Prefix() == "" && ns.URI() != "" {
				return true
			}
		}
	}
	return false
}

// fixNamespacesAfterCopy ensures a copied element has the correct namespace
// declarations relative to its new parent in the result tree.  When a
// no-namespace element is placed under a parent with a default namespace,
// it needs xmlns="" to prevent inheriting the parent's namespace.
// Similarly, redundant namespace declarations that match the parent are removed.
func (ec *execContext) fixNamespacesAfterCopy(elem *helium.Element) {
	parentDefaultNS := ""
	for node := elem.Parent(); node != nil; node = node.Parent() {
		if pe, ok := node.(*helium.Element); ok {
			if pe.Prefix() == "" && pe.URI() != "" {
				parentDefaultNS = pe.URI()
				break
			}
			for _, ns := range pe.Namespaces() {
				if ns.Prefix() == "" && ns.URI() != "" {
					parentDefaultNS = ns.URI()
					break
				}
			}
			if parentDefaultNS != "" {
				break
			}
		}
	}

	if parentDefaultNS == "" {
		return
	}

	// Element has no namespace but parent has default — add xmlns=""
	if elem.URI() == "" && elem.Prefix() == "" {
		hasUndecl := false
		for _, ns := range elem.Namespaces() {
			if ns.Prefix() == "" {
				hasUndecl = true
				break
			}
		}
		if !hasUndecl {
			_ = elem.DeclareNamespace("", "")
		}
	}

	// Element IS in the parent's namespace — remove redundant xmlns declaration
	if elem.URI() == parentDefaultNS && elem.Prefix() == "" {
		for _, ns := range elem.Namespaces() {
			if ns.Prefix() == "" && ns.URI() == parentDefaultNS {
				elem.RemoveNamespaceByPrefix("")
				break
			}
		}
	}
}

// fixDescendantDefaultNS recursively walks the descendant elements of elem
// and adds xmlns="" undeclarations on any no-namespace descendant that would
// otherwise inherit a default namespace from an ancestor in the result tree.
// This is needed when copying subtrees into a context with a default namespace.
func fixDescendantDefaultNS(elem *helium.Element) {
	// Determine the effective default namespace from this element.
	defaultNS := ""
	if elem.Prefix() == "" && elem.URI() != "" {
		defaultNS = elem.URI()
	}
	for _, ns := range elem.Namespaces() {
		if ns.Prefix() == "" {
			defaultNS = ns.URI()
			break
		}
	}
	// If no default namespace is active, check ancestors.
	if defaultNS == "" {
		for node := elem.Parent(); node != nil; node = node.Parent() {
			pe, ok := node.(*helium.Element)
			if !ok {
				continue
			}
			if pe.Prefix() == "" && pe.URI() != "" {
				defaultNS = pe.URI()
				break
			}
			for _, ns := range pe.Namespaces() {
				if ns.Prefix() == "" {
					defaultNS = ns.URI()
					break
				}
			}
			if defaultNS != "" {
				break
			}
		}
	}
	if defaultNS == "" {
		return
	}
	fixDescendantDefaultNSWalk(elem, defaultNS)
}

func fixDescendantDefaultNSWalk(parent *helium.Element, activeDefaultNS string) {
	for child := range helium.Children(parent) {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		// Check if this child overrides the default namespace.
		childDefaultNS := activeDefaultNS
		for _, ns := range childElem.Namespaces() {
			if ns.Prefix() == "" {
				childDefaultNS = ns.URI()
				break
			}
		}
		if childElem.Prefix() == "" && childElem.URI() == "" && activeDefaultNS != "" {
			// No-namespace element under a default namespace — needs xmlns=""
			hasUndecl := false
			for _, ns := range childElem.Namespaces() {
				if ns.Prefix() == "" {
					hasUndecl = true
					break
				}
			}
			if !hasUndecl {
				_ = childElem.DeclareNamespace("", "")
				childDefaultNS = ""
			}
		}
		fixDescendantDefaultNSWalk(childElem, childDefaultNS)
	}
}

// fixupExcludedElementNS handles namespace fixup for a literal result element
// whose own prefix was excluded via exclude-result-prefixes. After the body
// (including xsl:namespace instructions) has been executed, the element's
// prefix may now be bound to a different URI. In that case, invent a new
// prefix for the element's original namespace. If the prefix is still free,
// just declare it normally.
func (ec *execContext) fixupExcludedElementNS(elem *helium.Element, origPrefix, origURI string) {
	// Check if the prefix is now bound to a different URI
	for _, ns := range elem.Namespaces() {
		if ns.Prefix() == origPrefix {
			if ns.URI() == origURI {
				return // already correct
			}
			// Prefix is claimed by a different URI — invent a new prefix
			newPrefix := uniqueNSPrefix(elem, origPrefix+"_0", origURI)
			_ = elem.DeclareNamespace(newPrefix, origURI)
			_ = elem.SetActiveNamespace(newPrefix, origURI)
			return
		}
	}
	// Prefix is free — declare it for the element
	if !ec.isNSDeclaredInScope(origPrefix, origURI) {
		_ = elem.DeclareNamespace(origPrefix, origURI)
	}
}
