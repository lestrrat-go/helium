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
func copyAndStrip(src *helium.Document, strip, preserve []nameTest, buildNodeMap bool, schemaWS *schemaWSClassifier) (*helium.Document, map[helium.Node]helium.Node, error) {
	// Use RawEncoding(), not Encoding(): the latter synthesizes "utf8" when the
	// source XML declaration omitted an encoding, which would make the copy
	// serialize a spurious encoding="utf8" the source never had. The copy must
	// reproduce the source's encoding state EXACTLY (empty stays empty), matching
	// what helium.CopyDoc does (it reads the raw encoding field directly).
	// Version() and Standalone() already return the raw, unsynthesized values.
	dst := helium.NewDocument(src.Version(), src.RawEncoding(), src.Standalone())

	// Carry over the remaining document-level state that the copy stands in for
	// the source on. Properties record how the source was produced (e.g.
	// DocHTML, well-formedness); idsSkip preserves ID semantics so id() and
	// GetElementByID on the copy behave exactly as on the original source. The
	// ID table itself is intentionally NOT copied: it points at the original
	// source's elements, and a SkipIDs source has none anyway.
	dst.SetProperties(src.Properties())
	dst.SetSkipIDs(src.SkipIDs())

	// Decide whether to rebuild the copy's ID table from the source's. The source
	// ID table (populated during parse) is authoritative for id()/GetElementByID on
	// the source; the copy must reproduce it so both the no-strip and strip-space
	// paths resolve ids identically. Translating the table reproduces the source's
	// interned ID-table identity (and resolves at O(1)) rather than re-deriving ids
	// through the lazy O(n) GetElementByID fallback walk, which — though it now looks
	// up DTD ATTLIST decls by their raw qualified name (prefix+local), correctly
	// handling a prefixed element's ATTLIST (e.g. <!ATTLIST a:item eid ID>) — would
	// still yield a table with different element identities. Skip the rebuild when the source skips ids
	// (no resolution either way) or has no interned table (API-built source — its
	// own id() already relies on the lazy fallback, which the copy reproduces via
	// the carried-over DTD subsets).
	srcIDs := src.IDTable()
	rebuildIDs := !src.SkipIDs() && len(srcIDs) > 0

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
	//
	// Deep-copy the external subset (rather than sharing the pointer): the copy
	// document can be exposed to user code via raw-result capture, and *DTD has
	// mutators, so an aliased external subset would let a handler mutating the
	// copy's ExtSubset corrupt the source. CopyExtSubset gives the copy its own
	// independent external subset while keeping ID resolution identical.
	helium.CopyExtSubset(src, dst)

	sc := &stripCopier{dst: dst, strip: strip, preserve: preserve, schemaWS: schemaWS}
	// When rebuilding the ID table, record source-element->copy-element so the
	// source's ID entries can be translated onto the copy after the walk.
	if rebuildIDs {
		sc.elemMap = make(map[*helium.Element]*helium.Element)
	}
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

	// Rebuild the copy's ID table by translating each source entry's element
	// through the source->copy element correspondence. Any source element that has
	// no copy (it cannot occur for an ID-bearing element, which is never a
	// whitespace-only omitted node) is skipped defensively.
	if rebuildIDs {
		for id, srcElem := range srcIDs {
			cp := sc.elemMap[srcElem]
			if cp == nil {
				continue
			}
			dst.RegisterID(id, cp)
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
	// schemaWS, when non-nil, applies the XSLT 3.0 §4.4.2 schema-aware whitespace
	// verdicts (which override xsl:strip-space / xsl:preserve-space) using the type
	// annotations gathered during source validation. It is keyed on the ORIGINAL
	// source nodes, which is exactly what stripText's parent argument carries.
	schemaWS *schemaWSClassifier
	// nodeMap, when non-nil, records original->copy node correspondence (elements,
	// text/comment/PI leaves, and element attributes) for initial-match-selection
	// remapping. Omitted whitespace nodes have no entry.
	nodeMap map[helium.Node]helium.Node
	// elemMap, when non-nil, records source-element->copy-element correspondence so
	// the copy's ID table can be rebuilt by translating the source's ID entries.
	elemMap map[*helium.Element]*helium.Element
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
	if sc.elemMap != nil {
		sc.elemMap[src] = elem
	}

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

	// Copy attributes, preserving namespace information. Use the LITERAL setters
	// (SetLiteralAttribute/SetLiteralAttributeNS): a.Value() is the parser's
	// already-resolved value, so re-parsing it (SetAttribute/SetAttributeNS runs
	// CreateAttribute, which interprets entity references) would choke on a bare
	// '&' or '<' that was originally an entity (e.g. an href value carrying
	// '&amp;'), and silently double-resolve a value like '&amp;amp;'. Storing the
	// resolved value literally serializes byte-for-byte identically (the serializer
	// re-escapes '&'/'<') while never re-interpreting it.
	for _, a := range src.Attributes() {
		if a.URI() != "" {
			ns, nsErr := sc.dst.CreateNamespace(a.Prefix(), a.URI())
			if nsErr != nil {
				return nil, nsErr
			}
			if err := elem.SetLiteralAttributeNS(a.LocalName(), a.Value(), ns); err != nil {
				return nil, err
			}
			continue
		}
		if err := elem.SetLiteralAttribute(a.Name(), a.Value()); err != nil {
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
	if !isWhitespaceOnly(src.Content()) {
		return false
	}
	// A schema type annotation overrides xsl:strip-space / xsl:preserve-space:
	// element-only content strips regardless of preserve-space, while simple or
	// mixed content (or an assertion-bearing ancestor) preserves regardless of
	// strip-space (XSLT 3.0 §4.4.2).
	switch sc.schemaWS.mode(parent) {
	case schemaWSStrip:
		return true
	case schemaWSPreserve:
		return false
	}
	if isElementStrippedBy(parent, sc.strip, sc.preserve) {
		return true
	}
	// A copyAndStrip call with no strip rules AND no schema classifier is a PURE
	// copy (used to make the private validation copy). It must stay byte-faithful
	// and NOT drop DTD element-only whitespace, so the fallback below is skipped.
	// A genuine strip invocation always carries strip rules or a classifier (the
	// non-schema strip path always has non-empty strip rules), so this guard never
	// affects it.
	if len(sc.strip) == 0 && sc.schemaWS == nil {
		return false
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
