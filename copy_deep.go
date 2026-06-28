package helium

import (
	"maps"
	"slices"
)

// deepCopyOptions configures the shared deep-copy core (deepCopier). It is the
// single set of "speed/shape primitives" that every deep-copy site in the tree
// layer is expressed through: the leaf-node copy switch, the attribute copy
// loop, the namespace binding, and the child-linking strategy all live in one
// place and are selected by these knobs.
//
// The defaults (the zero value) reproduce the historical helium.CopyDoc/CopyNode
// behavior EXACTLY: over-declared namespaces and preflighted AddChild linking,
// no filtering, no mapping. Callers that want the faster shape opt in.
type deepCopyOptions struct {
	// overDeclareNS selects the namespace-declaration strategy for elements.
	//
	// When true (the historical helium.CopyDoc behavior), every element
	// re-declares its own nsDefs verbatim AND, if its active namespace is not
	// among them, declares that too — i.e. it may emit a redundant declaration
	// already in scope on an ancestor. This OVER-DECLARATION is load-bearing:
	// streaming's fixNamespacesAfterCopy depends on it, so it must be preserved
	// for the general copy path.
	//
	// When false (the strip-space fast path), the element reproduces its own
	// declarations verbatim but binds its active namespace to a declaration
	// already in scope (tracked via inScope) instead of re-declaring it, so the
	// copy is not over-declared. A fallback declaration is emitted only for a
	// degenerate source whose active namespace is not in scope anywhere.
	overDeclareNS bool

	// fastLink links children with AppendChildFast (no cycle/dup-attr preflight)
	// when true. The copy core only ever builds a freshly-constructed,
	// provably-acyclic, duplicate-free tree, so the preflight is pure overhead;
	// the general path keeps AddChild (false) to remain byte-for-byte identical
	// to the historical behavior on its callers.
	fastLink bool

	// omit, when non-nil, decides whether a node is dropped from the copy. It is
	// the generalized node-filter predicate: strip-space passes a
	// whitespace-omit filter; the general copy passes nil (copy everything). It
	// receives the source node, its source parent element (nil at document
	// level), and the per-element user state produced by enterElement.
	omit func(src Node, parent *Element, state any) bool

	// enterElement, when non-nil, derives the child user state for an element
	// from its parent's state. strip-space uses it to thread xml:space="preserve"
	// inheritance down the tree so omit can consult it. nil means "no state".
	enterElement func(src *Element, parentState any) any

	// onCopy, when non-nil, is invoked for every node that IS copied (not
	// omitted), with the source node and its fresh copy. It backs caller-side
	// bookkeeping such as the strip-space node map and ID-table rebuild.
	onCopy func(src, cp Node)

	// afterElementAttrs, when non-nil, is invoked for each copied element AFTER
	// its attributes have been copied (but before its children), with the source
	// and copy elements. strip-space uses it to map attribute correspondence,
	// which requires the copy's attributes to already exist.
	afterElementAttrs func(src, cp *Element)
}

// deepCopier carries the shared deep-copy state.
type deepCopier struct {
	dst  *Document
	opts deepCopyOptions
}

// copyChildren copies the children of src (a document or element) into the
// freshly-built parent, applying the filter and child-linking strategy.
// inScope is the namespace scope on the COPY (used in exact mode); parentState
// is the user state for src.
func (dc *deepCopier) copyChildren(src Node, parent MutableNode, inScope map[string]*Namespace, parentState any) error {
	srcElem, _ := AsNode[*Element](src)
	for c := src.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == DTDNode {
			continue
		}
		child, err := dc.copyNode(c, srcElem, inScope, parentState)
		if err != nil {
			return err
		}
		if child == nil {
			continue
		}
		if dc.opts.fastLink {
			if err := appendFastChild(parent, child); err != nil {
				return err
			}
			continue
		}
		if err := parent.AddChild(child); err != nil {
			return err
		}
	}
	return nil
}

// copyNode copies a single source node. parent is the SOURCE parent element (nil
// at document level), inScope is the copy-side namespace scope, and parentState
// is the user state for parent. Returns (nil, nil) when the node is filtered out.
func (dc *deepCopier) copyNode(src Node, parent *Element, inScope map[string]*Namespace, parentState any) (Node, error) {
	switch src.Type() {
	case ElementNode:
		elem, ok := AsNode[*Element](src)
		if !ok {
			return CopyNode(src, dc.dst)
		}
		return dc.copyElement(elem, inScope, parentState)
	case TextNode:
		if dc.filtered(src, parent, parentState) {
			return nil, nil //nolint:nilnil // omitted by filter
		}
		return dc.recorded(src, dc.dst.CreateText(slices.Clone(src.Content()))), nil
	case CDATASectionNode:
		if dc.filtered(src, parent, parentState) {
			return nil, nil //nolint:nilnil // omitted by filter
		}
		return dc.recorded(src, dc.dst.CreateCDATASection(slices.Clone(src.Content()))), nil
	case CommentNode:
		if dc.filtered(src, parent, parentState) {
			return nil, nil //nolint:nilnil // omitted by filter
		}
		return dc.recorded(src, dc.dst.CreateComment(slices.Clone(src.Content()))), nil
	case ProcessingInstructionNode:
		if dc.filtered(src, parent, parentState) {
			return nil, nil //nolint:nilnil // omitted by filter
		}
		return dc.recorded(src, dc.dst.CreatePI(src.Name(), string(src.Content()))), nil
	case EntityRefNode:
		if dc.filtered(src, parent, parentState) {
			return nil, nil //nolint:nilnil // omitted by filter
		}
		cp, err := dc.dst.CreateCharRef(src.Name())
		if err != nil {
			return nil, err
		}
		return dc.recorded(src, cp), nil
	default:
		// Mirror CopyNode for any other node type (e.g. NamespaceNode) so the
		// copy stays faithful. These are never filtered.
		cp, err := CopyNode(src, dc.dst)
		if err != nil {
			return nil, err
		}
		if cp != nil {
			dc.record(src, cp)
		}
		return cp, nil
	}
}

func (dc *deepCopier) filtered(src Node, parent *Element, state any) bool {
	return dc.opts.omit != nil && dc.opts.omit(src, parent, state)
}

func (dc *deepCopier) record(src, cp Node) {
	// A faithful deep copy preserves source line numbers so diagnostics emitted
	// against a copied tree (e.g. xsd's conditional-inclusion clone) keep their
	// original locations instead of reporting line 0. record runs for every node
	// the copier produces (elements via copyElement, text/comment/PI/CDATA/
	// entity-ref via recorded, and the default CopyNode branch), so this is the
	// single chokepoint that covers them all.
	copyLine(src, cp)
	if dc.opts.onCopy != nil {
		dc.opts.onCopy(src, cp)
	}
}

// copyLine carries src's source line number onto its copy cp. Every helium node
// type embeds docnode (reachable via baseDocNode), including the virtual
// NamespaceNodeWrapper, so this works uniformly; the nil guards are belt-and-
// suspenders.
func copyLine(src, cp Node) {
	if src == nil || cp == nil {
		return
	}
	sdn, cdn := src.baseDocNode(), cp.baseDocNode()
	if sdn == nil || cdn == nil {
		return
	}
	cdn.SetLine(sdn.Line())
}

// recorded records the mapping and returns cp for convenient inline use.
func (dc *deepCopier) recorded(src, cp Node) Node {
	dc.record(src, cp)
	return cp
}

// copyElement copies an element. inScope maps each namespace prefix to the
// *Namespace declaration in scope on the COPY. parentState is the user state of
// the source parent element.
func (dc *deepCopier) copyElement(src *Element, inScope map[string]*Namespace, parentState any) (Node, error) {
	elem := dc.dst.CreateElement(src.LocalName())
	dc.record(src, elem)

	childScope, err := dc.bindNamespaces(src, elem, inScope)
	if err != nil {
		return nil, err
	}

	if err := dc.copyAttributes(src, elem); err != nil {
		return nil, err
	}

	if dc.opts.afterElementAttrs != nil {
		dc.opts.afterElementAttrs(src, elem)
	}

	state := parentState
	if dc.opts.enterElement != nil {
		state = dc.opts.enterElement(src, parentState)
	}

	if err := dc.copyChildren(src, elem, childScope, state); err != nil {
		return nil, err
	}
	return elem, nil
}

// bindNamespaces reproduces src's namespace declarations on elem and sets elem's
// active namespace, honoring the over-declare vs exact mode. It returns the
// child namespace scope (only meaningful in exact mode; nil in over-declare
// mode, which does not track scope).
func (dc *deepCopier) bindNamespaces(src, elem *Element, inScope map[string]*Namespace) (map[string]*Namespace, error) {
	if dc.opts.overDeclareNS {
		return nil, dc.bindNamespacesOverDeclare(src, elem)
	}
	return dc.bindNamespacesExact(src, elem, inScope)
}

// bindNamespacesOverDeclare reproduces the historical helium.copyElement
// namespace behavior verbatim: declare every nsDefs entry, then declare the
// active namespace too if it was not already declared, then set it active. This
// can emit a declaration already in scope on an ancestor (over-declaration) and
// is REQUIRED by streaming's fixNamespacesAfterCopy.
func (dc *deepCopier) bindNamespacesOverDeclare(src, elem *Element) error {
	declaredPrefixes := make(map[string]struct{})

	if nc, ok := Node(src).(NamespaceContainer); ok {
		for _, ns := range nc.Namespaces() {
			if err := elem.DeclareNamespace(ns.Prefix(), ns.URI()); err != nil {
				return err
			}
			declaredPrefixes[ns.Prefix()] = struct{}{}
		}
	}

	if nsr, ok := Node(src).(Namespacer); ok {
		if ns := nsr.Namespace(); ns != nil {
			if _, declared := declaredPrefixes[ns.Prefix()]; ns.URI() != "" && !declared {
				if err := elem.DeclareNamespace(ns.Prefix(), ns.URI()); err != nil {
					return err
				}
			}
			if err := elem.SetActiveNamespace(ns.Prefix(), ns.URI()); err != nil {
				return err
			}
		}
	}
	return nil
}

// bindNamespacesExact reproduces the strip-space fast-path namespace behavior:
// declarations are reproduced verbatim, and the active namespace is bound to a
// declaration already in scope instead of re-declaring it (no over-declaration).
// A fallback declaration is emitted only when no in-scope declaration matches.
func (dc *deepCopier) bindNamespacesExact(src, elem *Element, inScope map[string]*Namespace) (map[string]*Namespace, error) {
	childScope := inScope
	ownDecls := src.Namespaces()
	if len(ownDecls) > 0 {
		childScope = make(map[string]*Namespace, len(inScope)+len(ownDecls))
		maps.Copy(childScope, inScope)
		for _, ns := range ownDecls {
			decl, err := dc.dst.CreateNamespace(ns.Prefix(), ns.URI())
			if err != nil {
				return nil, err
			}
			elem.AddNamespaceDecl(decl)
			childScope[ns.Prefix()] = decl
		}
	}

	if ns := src.Namespace(); ns != nil && ns.URI() != "" {
		active := childScope[ns.Prefix()]
		if active == nil || active.URI() != ns.URI() {
			decl, err := dc.dst.CreateNamespace(ns.Prefix(), ns.URI())
			if err != nil {
				return nil, err
			}
			elem.AddNamespaceDecl(decl)
			if len(ownDecls) == 0 {
				childScope = make(map[string]*Namespace, len(inScope)+1)
				maps.Copy(childScope, inScope)
			}
			childScope[ns.Prefix()] = decl
			active = decl
		}
		elem.SetNs(active)
	}
	return childScope, nil
}

// copyAttributes copies src's attributes onto elem using the value-parsing
// setters (SetAttribute/SetAttributeNS) so the copy is byte-for-byte identical,
// including entity-reference handling. This loop is identical across all copy
// sites.
func (dc *deepCopier) copyAttributes(src, elem *Element) error {
	for _, a := range src.Attributes() {
		if a.URI() != "" {
			ns, nsErr := dc.dst.CreateNamespace(a.Prefix(), a.URI())
			if nsErr != nil {
				return nsErr
			}
			if _, err := elem.SetAttributeNS(a.LocalName(), a.Value(), ns); err != nil {
				return err
			}
		} else if _, err := elem.SetAttribute(a.Name(), a.Value()); err != nil {
			return err
		}
		// Preserve the source attribute's line. SetAttribute* returns the element,
		// so look the copy back up by expanded name. This is the attribute analogue
		// of record()'s line preservation, but done HERE (not via dc.record) so it
		// stays metadata-only and does not start surfacing attributes to onCopy
		// callbacks, which only expect element/text/etc. nodes.
		if cp, ok := elem.FindAttribute(NSPredicate{Local: a.LocalName(), NamespaceURI: a.URI()}); ok {
			copyLine(a, cp)
		}
	}
	return nil
}
