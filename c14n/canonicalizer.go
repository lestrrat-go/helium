package c14n

import (
	"fmt"
	"io"
	"maps"
	"net/url"
	"slices"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

type canonicalizer struct {
	doc               *helium.Document
	mode              Mode
	out               io.Writer
	withComments      bool
	nodeSet           map[helium.Node]struct{} // nil = whole document
	inclusivePrefixes map[string]struct{}
	strictXMLAttrs    bool // strict W3C node-set xml:* handling (default: libxml2)
	nsStack           *visibleNSStack
	// nsNodesByElement indexes NamespaceNodeWrapper nodes by their parent element.
	// Built once during process() when nodeSet is non-nil.
	nsNodesByElement map[helium.Node][]nsSortEntry
	// entityNSContext is a stack of reference-site in-scope namespace bindings
	// (prefix→URI), pushed when the walk descends through an EntityRef. While
	// non-empty, an entity-replacement element resolves the prefixes it does not
	// itself declare against the reference site's bindings rather than the shared
	// Entity declaration node's DTD-context parent chain and its cached
	// active-namespace pointers (both resolved once at parse time against the
	// first reference site).
	entityNSContext []map[string]string
}

func (c *canonicalizer) process() error {
	c.nsStack = newVisibleNSStack()

	// Build namespace node index for node-set mode
	if c.nodeSet != nil {
		c.nsNodesByElement = make(map[helium.Node][]nsSortEntry)
		for n := range c.nodeSet {
			if n.Type() == helium.NamespaceNode {
				parent := n.Parent()
				prefix := n.Name()
				uri := string(n.Content())
				c.nsNodesByElement[parent] = append(c.nsNodesByElement[parent], nsSortEntry{
					prefix: prefix,
					uri:    uri,
				})
			}
		}
	}

	return c.processDocument()
}

// isVisible returns true if the node is in the node set (or if no node set filter is active).
func (c *canonicalizer) isVisible(n helium.Node) bool {
	if c.nodeSet == nil {
		return true
	}
	_, ok := c.nodeSet[n]
	return ok
}

func (c *canonicalizer) processDocument() error {
	// C14N: skip XML declaration, skip DTD
	// Walk document children, output top-level PIs with newlines
	beforeRoot := true
	for child := c.doc.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Type() {
		case helium.DocumentTypeNode, helium.DTDNode:
			continue
		case helium.ElementNode:
			elem, ok := helium.AsNode[*helium.Element](child)
			if !ok {
				continue
			}
			if err := c.processElement(elem); err != nil {
				return err
			}
			beforeRoot = false
		case helium.ProcessingInstructionNode:
			if !c.isVisible(child) {
				continue
			}
			pi, ok := helium.AsNode[*helium.ProcessingInstruction](child)
			if !ok {
				continue
			}
			if err := c.processPI(pi, beforeRoot); err != nil {
				return err
			}
		case helium.CommentNode:
			if !c.withComments || !c.isVisible(child) {
				continue
			}
			cm, ok := helium.AsNode[*helium.Comment](child)
			if !ok {
				continue
			}
			if err := c.processComment(cm, beforeRoot); err != nil {
				return err
			}
		case helium.TextNode:
			// Whitespace-only text between top-level nodes is suppressed in C14N
			continue
		}
	}
	return nil
}

// checkForRelativeNamespaces checks that no namespace on the element has a
// relative URI.  The C14N spec requires implementations to report failure when
// a relative namespace URI is encountered. Both declared namespaces
// (e.Namespaces()) and the element's active namespace (e.Namespace()) are
// inspected, because canonicalization emits in-scope active namespaces too — a
// programmatically built DOM can set an active namespace with a relative URI
// (via SetActiveNamespace) without ever declaring it.
// Mirrors libxml2's xmlC14NCheckForRelativeNamespaces (c14n.c:1338-1373).
func checkForRelativeNamespaces(e *helium.Element) error {
	if err := checkRelativeNamespaceURI(e, e.Namespace()); err != nil {
		return err
	}
	for _, ns := range e.Namespaces() {
		if err := checkRelativeNamespaceURI(e, ns); err != nil {
			return err
		}
	}
	return nil
}

func checkRelativeNamespaceURI(e *helium.Element, ns *helium.Namespace) error {
	if ns == nil {
		return nil
	}
	uri := ns.URI()
	if uri == "" {
		return nil
	}
	// C14N requires an operation failure on a relative namespace URI. A URI is
	// relative unless it carries a scheme, so parse it and require a non-empty
	// scheme rather than testing for a stray ":" (which a relative reference such
	// as "a/b:c" also contains). Mirrors libxml2's xmlC14NCheckForRelativeNamespaces
	// (xmlParseURI + scheme==NULL). url.Parse tolerates a raw space inside an
	// opaque part (e.g. "urn:foo bar") that libxml2's parser rejects, so reject
	// any whitespace/control byte up front — a valid URI never contains one.
	if !validURIReference(uri) {
		return fmt.Errorf("c14n: invalid namespace URI %q on element %s", uri, e.Name())
	}
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme == "" {
		return fmt.Errorf("c14n: relative namespace URI %q on element %s", uri, e.Name())
	}
	return nil
}

func (c *canonicalizer) processElement(e *helium.Element) error {
	if err := checkForRelativeNamespaces(e); err != nil {
		return err
	}

	visible := c.isVisible(e)

	// Push a namespace frame for this element (visible or not)
	c.nsStack.save()
	defer c.nsStack.restore()

	if visible {
		if _, err := io.WriteString(c.out, "<"); err != nil {
			return err
		}
		if err := c.writeQualifiedName(e); err != nil {
			return err
		}

		// Render namespace axis
		if err := c.renderNamespaces(e); err != nil {
			return err
		}

		// Render attribute axis
		if err := c.renderAttributes(e); err != nil {
			return err
		}

		if _, err := io.WriteString(c.out, ">"); err != nil {
			return err
		}
	} else if c.nodeSet != nil {
		// Non-visible element with node-set members: libxml2 still processes its
		// namespace axis and then its attribute axis (xmlC14NProcessElementNode
		// calls both with visible=0), rendering in-node-set nodes as text.
		// In exclusive mode, only namespace nodes whose prefix is in the inclusive
		// list are output.
		if c.mode == ExclusiveC14N10 {
			if len(c.inclusivePrefixes) > 0 {
				if err := c.renderNSNodesAsText(e, func(prefix string) bool {
					_, ok := c.inclusivePrefixes[prefix]
					return ok
				}); err != nil {
					return err
				}
			}
		} else if err := c.renderNSNodesAsText(e, nil); err != nil {
			return err
		}
		if err := c.renderOmittedAttributes(e); err != nil {
			return err
		}
	}

	// Recurse children
	for child := range helium.Children(e) {
		if err := c.processNode(child); err != nil {
			return err
		}
	}

	if visible {
		if _, err := io.WriteString(c.out, "</"); err != nil {
			return err
		}
		if err := c.writeQualifiedName(e); err != nil {
			return err
		}
		if _, err := io.WriteString(c.out, ">"); err != nil {
			return err
		}
	}
	return nil
}

func (c *canonicalizer) processNode(n helium.Node) error {
	switch n.Type() {
	case helium.ElementNode:
		elem, ok := helium.AsNode[*helium.Element](n)
		if !ok {
			return nil
		}
		return c.processElement(elem)
	case helium.TextNode:
		return c.processText(n)
	case helium.CDATASectionNode:
		return c.processText(n)
	case helium.ProcessingInstructionNode:
		if !c.isVisible(n) {
			return nil
		}
		// Inside elements, PIs are inline (no position-dependent newlines)
		pi, ok := helium.AsNode[*helium.ProcessingInstruction](n)
		if !ok {
			return nil
		}
		return c.writePI(pi)
	case helium.CommentNode:
		if !c.withComments || !c.isVisible(n) {
			return nil
		}
		// Inside elements, comments are inline (no position-dependent newlines)
		cm, ok := helium.AsNode[*helium.Comment](n)
		if !ok {
			return nil
		}
		return c.writeComment(cm)
	case helium.EntityRefNode:
		// Expand entity ref children. For an unexpanded reference the child is
		// the shared Entity declaration node, handled by the EntityNode case.
		// Canonicalize the replacement subtree in the namespace context of THIS
		// reference site (per W3C Canonical XML the replacement is included as if
		// textually substituted here), so a prefix in the replacement resolves
		// against the site's in-scope bindings.
		c.pushEntityContext(n)
		err := c.processChildren(n)
		c.popEntityContext()
		return err
	case helium.EntityNode:
		// The Entity declaration node holds the parsed replacement content as
		// its children; emit it so the reference contributes its replacement
		// text (and any markup) to the canonical output.
		return c.processChildren(n)
	}
	return nil
}

func (c *canonicalizer) processChildren(n helium.Node) error {
	for child := range helium.Children(n) {
		if err := c.processNode(child); err != nil {
			return err
		}
	}
	return nil
}

// currentEntityContext returns the reference-site namespace bindings for the
// entity expansion currently being walked, or nil when the walk is not inside
// one.
func (c *canonicalizer) currentEntityContext() map[string]string {
	n := len(c.entityNSContext)
	if n == 0 {
		return nil
	}
	return c.entityNSContext[n-1]
}

// pushEntityContext records the in-scope namespace bindings at an entity
// reference site so the replacement subtree canonicalizes as if the text were
// inserted there. In ordinary document content the reference site's ancestors
// carry reliable bindings. For a reference nested inside another entity, inherit
// the enclosing entity's reference-site context and overlay the xmlns
// declarations physically present in the enclosing entity subtree (the cached
// active-namespace pointers there are unreliable).
func (c *canonicalizer) pushEntityContext(entityRef helium.Node) {
	ctx := make(map[string]string)
	parent, ok := helium.AsNode[*helium.Element](entityRef.Parent())
	outer := c.currentEntityContext()
	switch {
	case outer == nil && ok:
		for prefix, ns := range domutil.InScopeNamespaces(parent, c.nodeSet == nil) {
			ctx[prefix] = ns.URI()
		}
	case outer != nil:
		maps.Copy(ctx, outer)
		if ok {
			c.overlayEntityNSDecls(ctx, parent)
		}
	}
	c.entityNSContext = append(c.entityNSContext, ctx)
}

func (c *canonicalizer) popEntityContext() {
	c.entityNSContext = c.entityNSContext[:len(c.entityNSContext)-1]
}

// overlayEntityNSDecls applies the xmlns declarations physically present in the
// entity subtree onto ctx: the element and its ancestors up to the entity
// boundary (the first non-element parent — the shared Entity declaration node),
// outermost first so an inner declaration wins. Only real declarations
// (nsDefs) are consulted; the elements' cached active-namespace pointers are
// not, since those were resolved at parse time against a specific reference
// site.
func (c *canonicalizer) overlayEntityNSDecls(ctx map[string]string, e *helium.Element) {
	var chain []*helium.Element
	for n := helium.Node(e); ; n = n.Parent() {
		el, ok := helium.AsNode[*helium.Element](n)
		if !ok {
			break
		}
		chain = append(chain, el)
	}
	for _, el := range slices.Backward(chain) {
		for _, ns := range el.Namespaces() {
			ctx[ns.Prefix()] = ns.URI()
		}
	}
}

func (c *canonicalizer) processText(n helium.Node) error {
	if c.nodeSet != nil && !c.isVisible(n) {
		return nil
	}
	// Both Text and CDATASection are output as escaped text in C14N
	return escapeText(c.out, n.Content())
}

func (c *canonicalizer) processPI(pi *helium.ProcessingInstruction, beforeRoot bool) error {
	if beforeRoot {
		// PI before root: output then newline
		if err := c.writePI(pi); err != nil {
			return err
		}
		_, err := io.WriteString(c.out, "\n")
		return err
	}
	// PI after root: newline then output
	if _, err := io.WriteString(c.out, "\n"); err != nil {
		return err
	}
	return c.writePI(pi)
}

func (c *canonicalizer) processComment(cm *helium.Comment, beforeRoot bool) error {
	if beforeRoot {
		if err := c.writeComment(cm); err != nil {
			return err
		}
		_, err := io.WriteString(c.out, "\n")
		return err
	}
	if _, err := io.WriteString(c.out, "\n"); err != nil {
		return err
	}
	return c.writeComment(cm)
}

func (c *canonicalizer) writePI(pi *helium.ProcessingInstruction) error {
	if _, err := io.WriteString(c.out, "<?"); err != nil {
		return err
	}
	if _, err := io.WriteString(c.out, pi.Name()); err != nil {
		return err
	}
	data := pi.Content()
	if len(data) > 0 {
		if _, err := io.WriteString(c.out, " "); err != nil {
			return err
		}
		if err := escapePIOrComment(c.out, data); err != nil {
			return err
		}
	}
	_, err := io.WriteString(c.out, "?>")
	return err
}

func (c *canonicalizer) writeComment(cm *helium.Comment) error {
	if _, err := io.WriteString(c.out, "<!--"); err != nil {
		return err
	}
	if err := escapePIOrComment(c.out, cm.Content()); err != nil {
		return err
	}
	_, err := io.WriteString(c.out, "-->")
	return err
}

func (c *canonicalizer) writeQualifiedName(e *helium.Element) error {
	ns := e.Namespace()
	if ns != nil && ns.Prefix() != "" {
		if _, err := io.WriteString(c.out, ns.Prefix()); err != nil {
			return err
		}
		if _, err := io.WriteString(c.out, ":"); err != nil {
			return err
		}
	}
	_, err := io.WriteString(c.out, e.LocalName())
	return err
}

// renderNamespaces outputs the namespace axis for the element.
func (c *canonicalizer) renderNamespaces(e *helium.Element) error {
	if c.mode == ExclusiveC14N10 {
		return c.renderNamespacesExclusive(e)
	}

	if c.nodeSet != nil {
		return c.renderNamespacesNodeSet(e)
	}

	// Whole-document mode: collect in-scope namespaces
	nsMap := c.collectInScopeNamespaces(e)

	// Determine which need to be output (not yet on the rendered stack)
	var toOutput []nsSortEntry
	for prefix, uri := range nsMap {
		if c.nsStack.needsOutput(prefix, uri) {
			toOutput = append(toOutput, nsSortEntry{prefix: prefix, uri: uri})
			c.nsStack.add(prefix, uri)
		}
	}

	// Sort and output
	sortNamespaces(toOutput)
	for _, ns := range toOutput {
		if err := c.writeNSDecl(ns.prefix, ns.uri); err != nil {
			return err
		}
	}
	return nil
}

// renderNamespacesNodeSet outputs namespaces for node-set mode.
// A namespace node is output if it's in the node set for this element and
// the nearest visible ancestor that also has this prefix in its node set
// either doesn't exist or has a different URI.
func (c *canonicalizer) renderNamespacesNodeSet(e *helium.Element) error {
	nsNodes := c.nsNodesByElement[e]

	var toOutput []nsSortEntry
	hasDefaultNS := false
	for _, nsn := range nsNodes {
		// Skip the xml namespace — it's never explicitly declared in C14N
		if nsn.prefix == lexicon.PrefixXML {
			continue
		}
		if nsn.prefix == "" {
			hasDefaultNS = true
		}
		if !c.nsRenderedByAncestor(e, nsn.prefix, nsn.uri) {
			toOutput = append(toOutput, nsn)
		}
	}

	// Special case: default namespace undeclaration.
	// If the element has no default namespace node in its node set
	// but the nearest visible ancestor with a default namespace node
	// rendered a non-empty URI, we must emit xmlns="" to "reset"
	// the default namespace so it doesn't leak through from the ancestor.
	if !hasDefaultNS {
		ancURI := c.findNearestRenderedDefaultNS(e)
		if ancURI != "" {
			toOutput = append(toOutput, nsSortEntry{prefix: "", uri: ""})
		}
	}

	sortNamespaces(toOutput)
	for _, ns := range toOutput {
		if err := c.writeNSDecl(ns.prefix, ns.uri); err != nil {
			return err
		}
	}
	return nil
}

// renderNSNodesAsText outputs namespace nodes on non-visible elements as text.
// When a namespace node is in the node set but its parent element is not visible,
// the namespace declaration is rendered as text content (e.g. " xmlns:foo=\"uri\"").
// In exclusive mode only prefixes accepted by include are output. In inclusive
// mode (include == nil) a namespace node already rendered by the nearest visible
// ancestor is suppressed, matching the inclusive-C14N rule that such a namespace
// node is ignored.
func (c *canonicalizer) renderNSNodesAsText(e *helium.Element, include func(string) bool) error {
	nsNodes := c.nsNodesByElement[e]
	if len(nsNodes) == 0 {
		return nil
	}

	var toOutput []nsSortEntry
	for _, nsn := range nsNodes {
		if nsn.prefix == lexicon.PrefixXML {
			continue
		}
		if include != nil {
			// Exclusive mode: only inclusive prefixes, and only when not already
			// rendered on the rendered-namespace stack.
			if !include(nsn.prefix) || !c.nsStack.needsOutput(nsn.prefix, nsn.uri) {
				continue
			}
		} else if c.nsRenderedByAncestor(e, nsn.prefix, nsn.uri) {
			// Inclusive mode: ignore a namespace already rendered by the nearest
			// visible ancestor.
			continue
		}
		toOutput = append(toOutput, nsn)
	}
	sortNamespaces(toOutput)

	// A non-visible element only consults the rendered-namespace stack for
	// suppression; it never adds to it (libxml2 calls xmlC14NVisibleNsStackAdd
	// for visible elements only). Adding here would wrongly suppress the same
	// declaration on sibling omitted elements.
	for _, ns := range toOutput {
		if err := c.writeNSDecl(ns.prefix, ns.uri); err != nil {
			return err
		}
	}
	return nil
}

// renderOmittedAttributes outputs, as text, the in-node-set attributes of a
// non-visible element (libxml2's xmlC14NProcessAttrsAxis with visible=0). No
// xml:* inheritance or xml:base fixup is performed — that applies to visible
// elements only; an omitted element simply emits its node-set attributes.
func (c *canonicalizer) renderOmittedAttributes(e *helium.Element) error {
	attrs := e.Attributes()
	entries := make([]attrSortEntry, 0, len(attrs))
	for _, attr := range attrs {
		if !c.isVisible(attr) {
			continue
		}
		entries = append(entries, attrSortEntry{
			attr:      attr,
			localName: attr.LocalName(),
			nsURI:     attr.URI(),
		})
	}
	sortAttributes(entries)
	for _, entry := range entries {
		if err := c.writeAttribute(entry); err != nil {
			return err
		}
	}
	return nil
}

// nsRenderedByAncestor checks if the namespace (prefix, uri) is already
// effectively rendered by walking up through the nearest visible ancestor.
// The check compares against the nearest visible parent's full namespace
// node set — if the parent has the same (prefix, uri), suppress.
// If the parent has the prefix with a different URI, or doesn't have the prefix
// at all, emit.
func (c *canonicalizer) nsRenderedByAncestor(e *helium.Element, prefix, uri string) bool {
	// Find the nearest visible ancestor
	for n := e.Parent(); n != nil; n = n.Parent() {
		if n.Type() != helium.ElementNode {
			continue
		}
		anc, ok := helium.AsNode[*helium.Element](n)
		if !ok {
			continue
		}
		if !c.isVisible(anc) {
			continue
		}
		// Check this ancestor's namespace node set
		ancNS := c.nsNodesByElement[anc]
		for _, ans := range ancNS {
			if ans.prefix == prefix {
				return ans.uri == uri
			}
		}
		// Nearest visible ancestor doesn't have this prefix → need to emit
		return false
	}
	// No visible ancestor at all
	// For default namespace with empty URI, suppress (implicit)
	if prefix == "" && uri == "" {
		return true
	}
	return false
}

// findNearestRenderedDefaultNS walks up through visible ancestors to find
// the URI of the default namespace node in the nearest ancestor's node set.
func (c *canonicalizer) findNearestRenderedDefaultNS(e *helium.Element) string {
	for n := e.Parent(); n != nil; n = n.Parent() {
		if n.Type() != helium.ElementNode {
			continue
		}
		anc, ok := helium.AsNode[*helium.Element](n)
		if !ok {
			continue
		}
		if !c.isVisible(anc) {
			continue
		}
		// The nearest visible ancestor determines the default namespace in
		// scope for e in the canonical output, so this ancestor is
		// authoritative — never walk further up. The XPath namespace axis only
		// yields a default-namespace node when the in-scope default URI is
		// non-empty; its absence here means the default namespace is empty (it
		// was reset via xmlns=""). Returning "" in that case stops us from
		// reaching past a reset to a more distant, no-longer-in-scope default.
		for _, ans := range c.nsNodesByElement[anc] {
			if ans.prefix == "" {
				return ans.uri
			}
		}
		return ""
	}
	return ""
}

func (c *canonicalizer) renderNamespacesExclusive(e *helium.Element) error {
	if c.nodeSet != nil {
		return c.renderNamespacesExclusiveNodeSet(e)
	}

	// Whole-document mode: output "visibly utilized" namespaces
	// plus any in the inclusive prefixes list.
	utilized := make(map[string]string)

	// Element's own namespace
	if ns := e.Namespace(); ns != nil {
		utilized[ns.Prefix()] = c.resolvedNSURI(e, ns)
	} else {
		// Check if default namespace needs to be undeclared
		if existingURI, found := c.nsStack.lookup(""); found && existingURI != "" {
			utilized[""] = ""
		}
	}

	// Attribute namespaces
	for _, attr := range e.Attributes() {
		if p := attr.Prefix(); p != "" {
			utilized[p] = attr.URI()
		}
	}

	// Inclusive prefixes
	if c.inclusivePrefixes != nil {
		nsMap := c.collectInScopeNamespaces(e)
		for prefix := range c.inclusivePrefixes {
			if uri, ok := nsMap[prefix]; ok {
				utilized[prefix] = uri
			}
		}
	}

	var toOutput []nsSortEntry
	for prefix, uri := range utilized {
		if c.nsStack.needsOutput(prefix, uri) {
			toOutput = append(toOutput, nsSortEntry{prefix: prefix, uri: uri})
			c.nsStack.add(prefix, uri)
		}
	}

	sortNamespaces(toOutput)
	for _, ns := range toOutput {
		if err := c.writeNSDecl(ns.prefix, ns.uri); err != nil {
			return err
		}
	}
	return nil
}

// renderNamespacesExclusiveNodeSet handles exclusive C14N namespace rendering
// when a node set is active. A namespace is output only if:
//  1. Its prefix is "visibly utilized" (element's own ns or attribute ns) OR
//     in the inclusive prefix list
//  2. AND the corresponding namespace node is in the node set for this element
//  3. AND it differs from what the nsStack already has (not already rendered)
//
// Uses nsStack (not nsRenderedByAncestor) to track what was actually rendered,
// since exclusive mode only renders a subset of ns nodes in the node set.
func (c *canonicalizer) renderNamespacesExclusiveNodeSet(e *helium.Element) error {
	nsNodes := c.nsNodesByElement[e]

	// Build a map of ns nodes in the node set for this element
	nsNodeMap := make(map[string]string, len(nsNodes))
	for _, nsn := range nsNodes {
		nsNodeMap[nsn.prefix] = nsn.uri
	}

	// Determine "candidate" prefixes: visibly utilized ∪ inclusive
	candidates := make(map[string]bool)

	// Element's own namespace prefix. An element in no namespace (e.g. after an
	// xmlns="" reset) has a nil active namespace; it still visibly utilizes the
	// empty default-namespace prefix, so make "" a candidate. The undeclaration
	// check below then emits xmlns="" iff an output ancestor left a non-empty
	// default namespace in scope (mirrors the whole-document exclusive path).
	if ns := e.Namespace(); ns != nil {
		candidates[ns.Prefix()] = true
	} else {
		candidates[""] = true
	}

	// Attribute namespace prefixes (only visible attributes)
	for _, attr := range e.Attributes() {
		if !c.isVisible(attr) {
			continue
		}
		if p := attr.Prefix(); p != "" {
			candidates[p] = true
		}
	}

	// Inclusive prefixes
	for prefix := range c.inclusivePrefixes {
		candidates[prefix] = true
	}

	var toOutput []nsSortEntry

	for prefix := range candidates {
		if prefix == lexicon.PrefixXML {
			continue
		}

		// Only output if the ns node is in the node set for this element
		uri, inNodeSet := nsNodeMap[prefix]
		if !inNodeSet {
			// Special case: default namespace undeclaration.
			// If "" is a candidate but its ns node is NOT in the node set,
			// check if a visible ancestor rendered a non-empty default ns.
			if prefix == "" {
				if existingURI, found := c.nsStack.lookup(""); found && existingURI != "" {
					toOutput = append(toOutput, nsSortEntry{prefix: "", uri: ""})
					c.nsStack.add("", "")
				}
			}
			continue
		}

		if c.nsStack.needsOutput(prefix, uri) {
			toOutput = append(toOutput, nsSortEntry{prefix: prefix, uri: uri})
			c.nsStack.add(prefix, uri)
		}
	}

	sortNamespaces(toOutput)
	for _, ns := range toOutput {
		if err := c.writeNSDecl(ns.prefix, ns.uri); err != nil {
			return err
		}
	}
	return nil
}

// resolvedNSURI returns the URI of the element's name namespace. Inside an
// entity expansion the element's cached active-namespace pointer was resolved
// once at parse time against the first reference site, so re-resolve the
// element's prefix against the current reference-site context; outside an entity
// the cached pointer is authoritative.
func (c *canonicalizer) resolvedNSURI(e *helium.Element, ns *helium.Namespace) string {
	if c.currentEntityContext() == nil {
		return ns.URI()
	}
	if uri, ok := c.collectInScopeNamespaces(e)[ns.Prefix()]; ok {
		return uri
	}
	return ns.URI()
}

// collectInScopeNamespaces collects all in-scope namespace bindings for an element
// by walking up the ancestor chain.
func (c *canonicalizer) collectInScopeNamespaces(e *helium.Element) map[string]string {
	// Inside an entity expansion the element's parent chain stops at the shared
	// Entity declaration node, so its own parent-chain walk (and its cached
	// active-namespace pointer) can only see the entity declaration's context.
	// Resolve against the reference site instead: start from the reference-site
	// bindings and overlay the xmlns declarations physically present in the
	// entity subtree.
	if ctx := c.currentEntityContext(); ctx != nil {
		nsMap := make(map[string]string, len(ctx))
		maps.Copy(nsMap, ctx)
		c.overlayEntityNSDecls(nsMap, e)
		if c.nodeSet == nil {
			delete(nsMap, lexicon.PrefixXML)
		}
		return nsMap
	}

	// Remove the xml namespace (never explicitly output per C14N spec) unless
	// it's inherited via xml:* attributes in node-set mode.
	byPrefix := domutil.InScopeNamespaces(e, c.nodeSet == nil)
	nsMap := make(map[string]string, len(byPrefix))
	for prefix, ns := range byPrefix {
		nsMap[prefix] = ns.URI()
	}
	return nsMap
}

func (c *canonicalizer) writeNSDecl(prefix, uri string) error {
	if _, err := io.WriteString(c.out, " xmlns"); err != nil {
		return err
	}
	if prefix != "" {
		if _, err := io.WriteString(c.out, ":"); err != nil {
			return err
		}
		if _, err := io.WriteString(c.out, prefix); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(c.out, `="`); err != nil {
		return err
	}
	if err := escapeAttrValue(c.out, []byte(uri)); err != nil {
		return err
	}
	_, err := io.WriteString(c.out, `"`)
	return err
}

// renderAttributes outputs the attribute axis for the element.
func (c *canonicalizer) renderAttributes(e *helium.Element) error {
	attrs := e.Attributes()

	entries := make([]attrSortEntry, 0, len(attrs))
	for _, attr := range attrs {
		// C14N 1.1 handles xml:lang, xml:space and xml:base specially (below), so
		// keep them out of the ordinary visible-attribute pass.
		if c.mode == C14N11 && isInheritableXMLAttr(attr) {
			continue
		}
		if c.nodeSet != nil && !c.isVisible(attr) {
			continue
		}
		entries = append(entries, attrSortEntry{
			attr:      attr,
			localName: attr.LocalName(),
			nsURI:     attr.URI(),
		})
	}

	switch {
	case c.nodeSet != nil && c.mode == C14N10:
		c.inheritXMLAttrs10(e, &entries)
	case c.mode == C14N11:
		c.processSimpleInheritable11(e, &entries, "lang")
		c.processSimpleInheritable11(e, &entries, "space")
		if err := c.processXMLBase11(e, &entries); err != nil {
			return err
		}
	}

	sortAttributes(entries)

	for _, entry := range entries {
		if err := c.writeAttribute(entry); err != nil {
			return err
		}
	}
	return nil
}

// isInheritableXMLAttr reports whether attr is one of the xml-namespace
// attributes that C14N 1.1 processes specially (xml:lang, xml:space, xml:base).
func isInheritableXMLAttr(attr *helium.Attribute) bool {
	if attr.URI() != lexicon.NamespaceXML {
		return false
	}
	switch attr.LocalName() {
	case "lang", "space", xmlBaseLocalName:
		return true
	}
	return false
}

// xmlAttrOf returns the element's attribute with the given local name in the
// xml namespace, whether or not it is in the node set.
func xmlAttrOf(e *helium.Element, localName string) (*helium.Attribute, bool) {
	for _, attr := range e.Attributes() {
		if attr.LocalName() == localName && attr.URI() == lexicon.NamespaceXML {
			return attr, true
		}
	}
	return nil, false
}

// hasGap reports whether the element's parent element is omitted from the node
// set (so inherited xml:* attributes must be re-rendered on the element). A
// non-element parent (the document) counts as a gap.
func (c *canonicalizer) hasGap(e *helium.Element) bool {
	parent, ok := helium.AsNode[*helium.Element](e.Parent())
	if !ok {
		return true
	}
	return !c.isVisible(parent)
}

// strict reports whether strict W3C node-set xml:* handling applies. The toggle
// governs node-set processing only, so it has no effect in whole-document mode.
func (c *canonicalizer) strict() bool {
	return c.strictXMLAttrs && c.nodeSet != nil
}

// inheritXMLAttrs10 imports xml:* attributes from omitted ancestors for C14N 1.0
// node-set processing. Inheritance happens only across a gap; the nearest
// ancestor value for each xml:* name is imported unless that name is blocked.
//
// libxml2 blocks only on the element's already-rendered (visible) xml:*
// attributes. The strict W3C reading blocks on the element's entire attribute
// axis — any xml:* attribute it carries, whether or not it is in the node set.
func (c *canonicalizer) inheritXMLAttrs10(e *helium.Element, entries *[]attrSortEntry) {
	if !c.hasGap(e) {
		return
	}

	blocked := make(map[string]struct{})
	if c.strict() {
		for _, attr := range e.Attributes() {
			if attr.URI() == lexicon.NamespaceXML {
				blocked[attr.LocalName()] = struct{}{}
			}
		}
	} else {
		for _, entry := range *entries {
			if entry.nsURI == lexicon.NamespaceXML {
				blocked[entry.localName] = struct{}{}
			}
		}
	}

	for n := e.Parent(); n != nil; n = n.Parent() {
		anc, ok := helium.AsNode[*helium.Element](n)
		if !ok {
			continue
		}
		for _, attr := range anc.Attributes() {
			if attr.URI() != lexicon.NamespaceXML {
				continue
			}
			ln := attr.LocalName()
			if _, ok := blocked[ln]; ok {
				continue
			}
			*entries = append(*entries, attrSortEntry{
				attr:      attr,
				nsURI:     lexicon.NamespaceXML,
				localName: ln,
			})
			blocked[ln] = struct{}{}
		}
	}
}

// processSimpleInheritable11 handles a C14N 1.1 simple inheritable attribute
// (xml:lang or xml:space). The element's own value blocks inheritance and is
// emitted — unless strict mode suppresses an own value that is excluded from the
// node set. With no own value, the nearest omitted-ancestor value is inherited
// across a gap.
func (c *canonicalizer) processSimpleInheritable11(e *helium.Element, entries *[]attrSortEntry, localName string) {
	if own, ok := xmlAttrOf(e, localName); ok {
		if !c.strict() || c.isVisible(own) {
			*entries = append(*entries, attrSortEntry{attr: own, nsURI: lexicon.NamespaceXML, localName: localName})
		}
		return // own attribute blocks inheritance regardless of mode
	}

	for n := e.Parent(); n != nil; n = n.Parent() {
		anc, ok := helium.AsNode[*helium.Element](n)
		if !ok {
			return
		}
		if c.isVisible(anc) {
			return // reached a rendered ancestor: it carries the value normally
		}
		if a, ok := xmlAttrOf(anc, localName); ok {
			*entries = append(*entries, attrSortEntry{attr: a, nsURI: lexicon.NamespaceXML, localName: localName})
			return
		}
	}
}

// processXMLBase11 computes the C14N 1.1 xml:base value, following libxml2's
// xmlC14NFixupBaseAttr: the in-document xml:base values of the element and its
// contiguous omitted ancestors are joined lexically (join-URI-References). No
// external/retrieval base URI participates.
func (c *canonicalizer) processXMLBase11(e *helium.Element, entries *[]attrSortEntry) error {
	ownAttr, hasOwn := xmlAttrOf(e, xmlBaseLocalName)

	// xml:base values of contiguous omitted ancestors (inner→outer), stopping at
	// the first rendered ancestor.
	var innerToOuter []string
	hiddenHasBase := false
	for n := e.Parent(); n != nil; n = n.Parent() {
		anc, ok := helium.AsNode[*helium.Element](n)
		if !ok {
			break
		}
		if c.isVisible(anc) {
			break
		}
		if a, ok := xmlAttrOf(anc, xmlBaseLocalName); ok {
			innerToOuter = append(innerToOuter, a.Value())
			hiddenHasBase = true
		}
	}

	// Strict mode performs the fixup only when an omitted ancestor actually
	// carries xml:base; an excluded own xml:base then renders as an ordinary
	// attribute (if visible) but never seeds a fixup. An empty value is dropped,
	// matching the join's empty result.
	if c.strict() && !hiddenHasBase {
		// No omitted-ancestor base: own xml:base renders as an ordinary attribute
		// (validated at the writeAttribute chokepoint), never seeding a fixup.
		if hasOwn && c.isVisible(ownAttr) {
			if v := ownAttr.Value(); v != "" {
				c.setXMLBaseEntry(entries, v)
			}
		}
		return nil
	}

	// Join chain, outermost→innermost: omitted-ancestor bases then the element's
	// own base. The own value is the innermost term of the join sequence and is
	// included whether or not the attribute node is itself in the node set.
	chain := make([]string, 0, len(innerToOuter)+1)
	for _, v := range slices.Backward(innerToOuter) {
		chain = append(chain, v)
	}
	if hasOwn {
		chain = append(chain, ownAttr.Value())
	}

	if len(chain) == 0 {
		return nil
	}
	res, faithful := reduceXMLBase(chain)
	if !faithful && c.strict() {
		return fmt.Errorf("c14n: xml:base on element %s cannot be canonicalized faithfully", e.Name())
	}
	if res != "" {
		c.setXMLBaseEntry(entries, res)
	}
	return nil
}

// xmlBaseLocalName is the local name of the xml:base attribute.
const xmlBaseLocalName = "base"

// setXMLBaseEntry sets or adds xml:base in the attr list with the given value.
func (c *canonicalizer) setXMLBaseEntry(entries *[]attrSortEntry, value string) {
	// Replace existing xml:base or add new
	for i, e := range *entries {
		if e.nsURI == lexicon.NamespaceXML && e.localName == xmlBaseLocalName {
			(*entries)[i].fixupValue = value
			(*entries)[i].hasFixup = true
			(*entries)[i].attr = nil
			return
		}
	}
	// Add new entry with fixupValue (attr is nil for synthetic entries)
	*entries = append(*entries, attrSortEntry{
		nsURI:      lexicon.NamespaceXML,
		localName:  xmlBaseLocalName,
		fixupValue: value,
		hasFixup:   true,
	})
}

func (c *canonicalizer) writeAttribute(entry attrSortEntry) error {
	// Strict mode is fail-closed on xml:base: every emitted value — an ordinary
	// or inherited attribute (C14N 1.0 and exclusive), an omitted-element
	// attribute, or a synthetic 1.1 fixup result — must be canonicalizable
	// faithfully. This is the single emission chokepoint; the chain-term check in
	// reduceXMLBase additionally catches a degenerate input that joins into a
	// faithful-looking result.
	if c.strict() && entry.nsURI == lexicon.NamespaceXML && entry.localName == xmlBaseLocalName {
		v := entry.fixupValue
		if !entry.hasFixup && entry.attr != nil {
			v = entry.attr.Value()
		}
		if !faithfulXMLBaseValue(v) {
			return fmt.Errorf("c14n: xml:base %q cannot be canonicalized faithfully", v)
		}
	}

	if _, err := io.WriteString(c.out, " "); err != nil {
		return err
	}

	// Write qualified name
	if entry.nsURI != "" && entry.attr != nil {
		// Namespaced attribute: use the prefix
		p := entry.attr.Prefix()
		if p != "" {
			if _, err := io.WriteString(c.out, p); err != nil {
				return err
			}
			if _, err := io.WriteString(c.out, ":"); err != nil {
				return err
			}
		}
	} else if entry.nsURI != "" && entry.attr == nil {
		// Synthetic attribute (e.g., xml:base fixup): write xml: prefix
		if _, err := io.WriteString(c.out, "xml:"); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(c.out, entry.localName); err != nil {
		return err
	}
	if _, err := io.WriteString(c.out, `="`); err != nil {
		return err
	}

	// Write value: check for fixup value first (C14N 1.1 xml:base). A fixup may
	// legitimately be the empty string (an empty relative reference resolving
	// back to the base), so rely on hasFixup rather than fixupValue != "" to
	// avoid falling through to writeAttrValue(nil) for synthetic entries.
	if entry.hasFixup {
		if err := escapeAttrValue(c.out, []byte(entry.fixupValue)); err != nil {
			return err
		}
	} else {
		if err := c.writeAttrValue(entry.attr); err != nil {
			return err
		}
	}

	_, err := io.WriteString(c.out, `"`)
	return err
}

// writeAttrValue writes the canonical attribute value by walking child nodes.
func (c *canonicalizer) writeAttrValue(attr *helium.Attribute) error {
	for child := range helium.Children(attr) {
		switch child.Type() {
		case helium.TextNode:
			if err := escapeAttrValue(c.out, child.Content()); err != nil {
				return err
			}
		case helium.EntityRefNode:
			// Expand entity reference children recursively
			for entChild := range helium.Children(child) {
				if err := escapeAttrValue(c.out, entChild.Content()); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
