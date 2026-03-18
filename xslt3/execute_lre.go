package xslt3

import (
	"context"

	"github.com/lestrrat-go/helium"
)

func (ec *execContext) execLiteralResultElement(ctx context.Context, inst *LiteralResultElement) error {
	// Use LocalName so that SetActiveNamespace doesn't double the prefix
	elemName := inst.LocalName
	if elemName == "" {
		elemName = inst.Name
	}
	elem, err := ec.resultDoc.CreateElement(elemName)
	if err != nil {
		return err
	}

	// Declare namespaces (skip if parent already has the same declaration)
	for prefix, uri := range inst.Namespaces {
		if !ec.isNSDeclaredInScope(prefix, uri) {
			if err := elem.DeclareNamespace(prefix, uri); err != nil {
				return err
			}
		}
	}

	// Set the element's own namespace
	if inst.Namespace != "" {
		if err := elem.SetActiveNamespace(inst.Prefix, inst.Namespace); err != nil {
			return err
		}
		// Ensure the namespace declaration is present for serialization.
		// SetActiveNamespace only sets n.ns; we also need it in nsDefs.
		if !hasNSDecl(elem, inst.Prefix, inst.Namespace) && !ec.isNSDeclaredInScope(inst.Prefix, inst.Namespace) {
			if err := elem.DeclareNamespace(inst.Prefix, inst.Namespace); err != nil {
				return err
			}
		}
	} else if inst.Prefix == "" && ec.hasDefaultNSInScope() {
		// No namespace on this LRE but default namespace in scope — undeclare it
		if err := elem.DeclareNamespace("", ""); err != nil {
			return err
		}
	}

	// Add element to output first so attribute sets can attach attributes.
	if err := ec.addNode(elem); err != nil {
		return err
	}

	// Execute body in element context with a new variable scope.
	// Temporarily disable sequenceMode so that children are added to this
	// element normally (not captured as separate items in the sequence).
	out := ec.currentOutput()
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

	defer func() {
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
	if len(inst.UseAttributeSets) > 0 {
		if err := ec.applyAttributeSets(ctx, inst.UseAttributeSets); err != nil {
			return err
		}
	}
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
			if err := elem.SetAttributeNS(attr.LocalName, val, ns); err != nil {
				return err
			}
		} else {
			if err := elem.SetAttribute(attr.Name, val); err != nil {
				return err
			}
		}
	}

	return ec.executeSequenceConstructor(ctx, inst.Body)
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
		// Apply referenced attribute sets first (use-attribute-sets on the set itself)
		if len(asDef.UseAttrSets) > 0 {
			if err := ec.applyAttributeSetsGuarded(ctx, asDef.UseAttrSets, active); err != nil {
				delete(active, name)
				return err
			}
		}
		// Execute the attribute instructions
		for _, inst := range asDef.Attrs {
			if err := ec.executeInstruction(ctx, inst); err != nil {
				delete(active, name)
				return err
			}
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
