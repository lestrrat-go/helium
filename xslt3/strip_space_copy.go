package xslt3

import (
	"maps"
	"slices"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

// copyAndStrip produces a deep copy of src that is ready for an xsl:strip-space
// transform in a SINGLE traversal. In one pre-order walk it:
//
//   - copies every node into a fresh document,
//   - OMITS whitespace-only text/CDATA nodes that strip-space would remove, so
//     no separate strip pass is needed and the copy is smaller,
//   - declares namespaces CORRECTLY without over-declaration (the source tree's
//     own declarations are reproduced verbatim and an element's active namespace
//     is bound to a declaration already in scope), so no prune pass is needed,
//   - links children directly into the parent's child slice via
//     helium.AppendChildFast, bypassing the per-node cycle/duplicate-attribute
//     preflight that AddChild runs (the tree is freshly constructed and provably
//     acyclic and duplicate-free).
//
// The document URI and DTD are preserved exactly as helium.CopyDoc would. In
// addition, document-level state that the copy stands in for the source on is
// carried over: the version/encoding/standalone (via NewDocument), the property
// flags, and the ID-skip state. The latter matters because the copy replaces the
// source for the duration of the transform; without it, a source parsed with
// SkipIDs(true) would lose that flag and id()/GetElementByID on the copy would
// wrongly resolve xml:id/ID attributes that the original source omitted.
//
// The result serializes byte-for-byte identically to copy+prune+strip on the
// same rules, while the caller's source DOM is never mutated.
//
// strip and preserve are the effective xsl:strip-space / xsl:preserve-space
// nameTests. At this point in the transform (before the exec context exists) the
// current package is always the principal stylesheet, so the effective rules are
// exactly the stylesheet's own — matching what stripWhitespaceFromDoc would have
// applied later.
func copyAndStrip(src *helium.Document, strip, preserve []nameTest, buildNodeMap bool) (*helium.Document, map[helium.Node]helium.Node, error) {
	dst := helium.NewDocument(src.Version(), src.Encoding(), src.Standalone())

	// Carry over the remaining document-level state that the copy stands in for
	// the source on. Properties record how the source was produced (e.g.
	// DocHTML, well-formedness); idsSkip preserves ID semantics so id() and
	// GetElementByID on the copy behave exactly as on the original source. The
	// ID table itself is intentionally NOT copied: it points at the original
	// source's elements, and a SkipIDs source has none anyway.
	dst.SetProperties(src.Properties())
	dst.SetSkipIDs(src.SkipIDs())

	// Deep-copy the internal DTD subset first (metadata + entities/elements/
	// attributes/notations), matching helium.CopyDoc's ordering so the copy
	// round-trips identically.
	helium.CopyDTDInfo(src, dst)

	// Carry over the source's EXTERNAL DTD subset too. CopyDTDInfo (like
	// helium.CopyDoc) only handles the internal subset, but GetElementByID's lazy
	// tree walk consults BOTH subsets for ID-typed attribute declarations. The
	// copy drops the source ID table (it points at the source's elements), so id()
	// on the copy falls back to that walk; without the external subset, IDs
	// declared in an external DTD would resolve under no-strip (which transforms
	// the source directly) but not under strip-space (which transforms this copy).
	// The external subset is only read by ID resolution, so sharing the pointer is
	// safe and keeps the two paths in agreement.
	if ext := src.ExtSubset(); ext != nil {
		dst.SetExtSubset(ext)
	}

	sc := &stripCopier{dst: dst, strip: strip, preserve: preserve}
	// When the initial match selection must be remapped onto the copy, record the
	// original->copy correspondence here. Whitespace omission changes the child
	// shape, so a parallel post-hoc walk would misalign; building the map during
	// the copy is both correct and cheaper. The document node itself maps so that
	// "/" from a remapped context node resolves to the copy.
	if buildNodeMap {
		sc.nodeMap = make(map[helium.Node]helium.Node)
		sc.nodeMap[src] = dst
	}

	// Copy document-level children in source order, skipping the DTD (already
	// handled above). The document root has no element parent and no inherited
	// namespace scope.
	for c := src.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.DTDNode {
			continue
		}
		child, err := sc.copyNode(c, nil, nil, false)
		if err != nil {
			return nil, nil, err
		}
		if child == nil {
			continue
		}
		if err := helium.AppendChildFast(dst, child); err != nil {
			return nil, nil, err
		}
	}

	dst.SetURL(src.URL())
	return dst, sc.nodeMap, nil
}

// stripCopier carries the per-copy state for copyAndStrip.
type stripCopier struct {
	dst      *helium.Document
	strip    []nameTest
	preserve []nameTest
	// nodeMap, when non-nil, records original->copy node correspondence (elements,
	// text/comment/PI leaves, and element attributes) for initial-match-selection
	// remapping. Omitted whitespace nodes have no entry.
	nodeMap map[helium.Node]helium.Node
}

// record stores the original->copy mapping when a node map is being built and
// returns cp unchanged for convenient inline use.
func (sc *stripCopier) record(src, cp helium.Node) helium.Node {
	if sc.nodeMap != nil {
		sc.nodeMap[src] = cp
	}
	return cp
}

// copyNode copies a single source node into sc.dst. For elements, inScope maps
// each namespace prefix to the *Namespace declaration in scope on the COPY, so
// an element's active namespace can be bound without re-declaring it (avoiding
// over-declaration). xmlSpacePreserve reports whether an ancestor carried
// xml:space="preserve", which suppresses stripping in this subtree.
//
// Returns (nil, nil) when the node is a whitespace-only text/CDATA node that
// strip-space removes.
func (sc *stripCopier) copyNode(src helium.Node, parent *helium.Element, inScope map[string]*helium.Namespace, xmlSpacePreserve bool) (helium.Node, error) {
	switch src.Type() {
	case helium.ElementNode:
		elem, ok := helium.AsNode[*helium.Element](src)
		if !ok {
			return nil, nil //nolint:nilnil // a nil node with no error means "omit from copy"
		}
		return sc.copyElement(elem, inScope, xmlSpacePreserve)
	case helium.TextNode:
		if !xmlSpacePreserve && sc.stripText(src, parent) {
			return nil, nil //nolint:nilnil // omitted whitespace-only node
		}
		return sc.record(src, sc.dst.CreateText(slices.Clone(src.Content()))), nil
	case helium.CDATASectionNode:
		if !xmlSpacePreserve && sc.stripText(src, parent) {
			return nil, nil //nolint:nilnil // omitted whitespace-only node
		}
		return sc.record(src, sc.dst.CreateCDATASection(slices.Clone(src.Content()))), nil
	case helium.CommentNode:
		return sc.record(src, sc.dst.CreateComment(slices.Clone(src.Content()))), nil
	case helium.ProcessingInstructionNode:
		return sc.record(src, sc.dst.CreatePI(src.Name(), string(src.Content()))), nil
	case helium.EntityRefNode:
		cp, err := sc.dst.CreateCharRef(src.Name())
		if err != nil {
			return nil, err
		}
		return sc.record(src, cp), nil
	default:
		// Mirror helium.CopyNode's behavior for any other node type so the copy
		// stays faithful (e.g. NamespaceNode handled the same way).
		return helium.CopyNode(src, sc.dst)
	}
}

// copyElement copies an element, reproducing its own namespace declarations
// verbatim and binding its active namespace to an in-scope declaration without
// over-declaring it. It then copies attributes and recurses into children using
// helium.AppendChildFast for direct linkage.
func (sc *stripCopier) copyElement(src *helium.Element, inScope map[string]*helium.Namespace, xmlSpacePreserve bool) (helium.Node, error) {
	elem := sc.dst.CreateElement(src.LocalName())
	sc.record(src, elem)

	// childScope = inScope plus this element's own declarations. Build it lazily:
	// only allocate a fresh map when this element declares namespaces, otherwise
	// reuse the parent scope directly (no per-element allocation in the common
	// no-declaration case).
	childScope := inScope
	ownDecls := src.Namespaces()
	if len(ownDecls) > 0 {
		childScope = make(map[string]*helium.Namespace, len(inScope)+len(ownDecls))
		maps.Copy(childScope, inScope)
		for _, ns := range ownDecls {
			decl, err := sc.dst.CreateNamespace(ns.Prefix(), ns.URI())
			if err != nil {
				return nil, err
			}
			elem.AddNamespaceDecl(decl)
			childScope[ns.Prefix()] = decl
		}
	}

	// Bind the active namespace to a declaration in scope on the copy. The source
	// tree is well-formed, so a matching (prefix, URI) declaration always exists
	// either on this element or an ancestor; reuse it instead of declaring anew.
	if ns := src.Namespace(); ns != nil && ns.URI() != "" {
		active := childScope[ns.Prefix()]
		if active == nil || active.URI() != ns.URI() {
			// No in-scope declaration matches (degenerate source); declare it here
			// to keep the copy serializable, mirroring helium.CopyDoc's fallback.
			decl, err := sc.dst.CreateNamespace(ns.Prefix(), ns.URI())
			if err != nil {
				return nil, err
			}
			elem.AddNamespaceDecl(decl)
			if childScope == nil || len(ownDecls) == 0 {
				childScope = make(map[string]*helium.Namespace, len(inScope)+1)
				maps.Copy(childScope, inScope)
			}
			childScope[ns.Prefix()] = decl
			active = decl
		}
		elem.SetNs(active)
	}

	// Copy attributes, preserving namespace information. Use the same value-
	// parsing setters as helium.CopyDoc (SetAttribute/SetAttributeNS) so the copy
	// is byte-for-byte identical, including any entity-reference handling.
	for _, a := range src.Attributes() {
		if a.URI() != "" {
			ns, nsErr := sc.dst.CreateNamespace(a.Prefix(), a.URI())
			if nsErr != nil {
				return nil, nsErr
			}
			if _, err := elem.SetAttributeNS(a.LocalName(), a.Value(), ns); err != nil {
				return nil, err
			}
			continue
		}
		if _, err := elem.SetAttribute(a.Name(), a.Value()); err != nil {
			return nil, err
		}
	}

	// Map attributes by expanded (URI, local) name when remapping is needed.
	// Attributes hang off the property list, not the child spine, and their
	// expanded name is unique per element.
	if sc.nodeMap != nil {
		mapElementAttributes(sc.nodeMap, src, elem)
	}

	// xml:space="preserve" on this element (or inherited) suppresses stripping in
	// this subtree, matching shouldStripWhitespace's inheritedXMLSpace check.
	childPreserve := xmlSpacePreserve
	if v, ok := src.GetAttribute("xml:space"); ok {
		childPreserve = v == lexicon.SpacePreserve
	}

	for c := src.FirstChild(); c != nil; c = c.NextSibling() {
		child, err := sc.copyNode(c, src, childScope, childPreserve)
		if err != nil {
			return nil, err
		}
		if child == nil {
			continue
		}
		if err := helium.AppendChildFast(elem, child); err != nil {
			return nil, err
		}
	}

	return elem, nil
}

// stripText reports whether a whitespace-only text/CDATA node would be removed by
// strip-space given parent (the ORIGINAL source parent element). It mirrors
// shouldStripWhitespace without an exec context: the effective rules are passed
// in, and xml:space inheritance is already handled by the caller via the
// xmlSpacePreserve flag, so only the element's strip/preserve match and DTD
// element-only content are evaluated here.
func (sc *stripCopier) stripText(src helium.Node, parent *helium.Element) bool {
	if parent == nil {
		return false
	}
	for _, b := range src.Content() {
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			return false
		}
	}
	if isElementStrippedBy(parent, sc.strip, sc.preserve) {
		return true
	}
	return hasElementOnlyContent(parent)
}

// isElementStrippedBy is the exec-context-free form of execContext.isElementStripped:
// it resolves the most authoritative matching strip rule and checks whether a
// preserve rule of at least equal authority overrides it.
func isElementStrippedBy(elem *helium.Element, strip, preserve []nameTest) bool {
	if len(strip) == 0 {
		return false
	}
	stripPrec, stripPriority, stripped := -1, -1, false
	for _, nt := range strip {
		if !matchSpaceNameTest(nt, elem) {
			continue
		}
		if !stripped || rankSpaceRule(nt) > packSpaceRank(stripPrec, stripPriority) {
			stripPrec = nt.ImportPrec
			stripPriority = nameTestPriority(nt)
			stripped = true
		}
	}
	if !stripped {
		return false
	}
	stripRank := packSpaceRank(stripPrec, stripPriority)
	for _, nt := range preserve {
		if matchSpaceNameTest(nt, elem) && rankSpaceRule(nt) >= stripRank {
			return false
		}
	}
	return true
}
