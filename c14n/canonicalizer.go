package c14n

import (
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"slices"
	"strings"

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
	baseURI           string // document base URI for C14N 1.1 xml:base fixup
	nsStack           *visibleNSStack
	// nsNodesByElement indexes NamespaceNodeWrapper nodes by their parent element.
	// Built once during process() when nodeSet is non-nil.
	nsNodesByElement map[helium.Node][]nsSortEntry
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
	if uri != "" && !strings.Contains(uri, ":") {
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
		// Non-visible element: output namespace nodes in the node set as text.
		// In exclusive mode, only output ns nodes whose prefix is in the inclusive list.
		if c.mode == ExclusiveC14N10 {
			if len(c.inclusivePrefixes) > 0 {
				if err := c.renderNSNodesAsText(e, func(prefix string) bool {
					_, ok := c.inclusivePrefixes[prefix]
					return ok
				}); err != nil {
					return err
				}
			}
		} else {
			if err := c.renderNSNodesAsText(e, nil); err != nil {
				return err
			}
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
		// Expand entity ref children
		for child := range helium.Children(n) {
			if err := c.processNode(child); err != nil {
				return err
			}
		}
		return nil
	}
	return nil
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
// Only namespace nodes whose prefix satisfies include are output; pass nil to
// output all (non-xml) prefixes.
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
		if include != nil && !include(nsn.prefix) {
			continue
		}
		toOutput = append(toOutput, nsn)
	}
	sortNamespaces(toOutput)

	for _, ns := range toOutput {
		if err := c.writeNSDecl(ns.prefix, ns.uri); err != nil {
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
		ancNS := c.nsNodesByElement[anc]
		for _, ans := range ancNS {
			if ans.prefix == "" {
				return ans.uri
			}
		}
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
		utilized[ns.Prefix()] = ns.URI()
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

	// Element's own namespace prefix
	if ns := e.Namespace(); ns != nil {
		candidates[ns.Prefix()] = true
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

// collectInScopeNamespaces collects all in-scope namespace bindings for an element
// by walking up the ancestor chain.
func (c *canonicalizer) collectInScopeNamespaces(e *helium.Element) map[string]string {
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

	// Build sort entries
	entries := make([]attrSortEntry, 0, len(attrs))
	for _, attr := range attrs {
		// In node-set mode, skip non-visible attributes
		if c.nodeSet != nil && !c.isVisible(attr) {
			continue
		}
		entry := attrSortEntry{
			attr:      attr,
			localName: attr.LocalName(),
			nsURI:     attr.URI(),
		}
		entries = append(entries, entry)
	}

	// Handle xml:* attribute inheritance from hidden ancestors in node-set mode
	if c.nodeSet != nil && c.mode == C14N10 {
		c.inheritXMLAttrsFiltered(e, &entries, nil)
	} else if c.nodeSet != nil && c.mode == C14N11 {
		// C14N 1.1: only inherit xml:lang and xml:space (not xml:id or xml:base).
		c.inheritXMLAttrsFiltered(e, &entries, func(ln string) bool {
			return ln == "lang" || ln == "space"
		})
		c.fixupXMLBase(e, &entries)
	}

	sortAttributes(entries)

	for _, entry := range entries {
		if err := c.writeAttribute(entry); err != nil {
			return err
		}
	}
	return nil
}

// inheritXMLAttrsFiltered adds xml:* attributes inherited from ancestors in
// node-set mode. The accept predicate selects which xml:* local names to
// inherit; pass nil to inherit all of them (C14N 1.0).
//
// The rule: an element E needs to re-render inherited xml:* attributes only
// when there is a "non-visible gap" — i.e., E's immediate parent element is
// NOT in the node set. If the parent is visible, its output carries the xml:*
// attributes through normal XML scoping, so no re-rendering is needed.
//
// When there IS a gap, walk ALL ancestors to find the nearest one with each
// xml:* attribute and inherit from it.
func (c *canonicalizer) inheritXMLAttrsFiltered(e *helium.Element, entries *[]attrSortEntry, accept func(string) bool) {
	// Only inherit if there's a non-visible gap
	if parentNode := e.Parent(); parentNode != nil {
		if parentElem, ok := helium.AsNode[*helium.Element](parentNode); ok {
			if c.isVisible(parentElem) {
				// Parent is visible — no gap, xml:* attrs flow through normally
				return
			}
		}
	}

	// Parent is NOT visible (gap exists).
	// Walk ALL ancestors to find xml:* attrs. Take the nearest value for each.
	present := make(map[string]bool)
	for _, entry := range *entries {
		if entry.nsURI == lexicon.NamespaceXML {
			present[entry.localName] = true
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
			if accept != nil && !accept(ln) {
				continue
			}
			if present[ln] {
				continue
			}
			*entries = append(*entries, attrSortEntry{
				attr:      attr,
				nsURI:     lexicon.NamespaceXML,
				localName: ln,
			})
			present[ln] = true
		}
	}
}

// fixupXMLBase computes the xml:base fixup for C14N 1.1.
// When there's a non-visible gap, the element's xml:base must be adjusted
// to account for non-visible ancestors' xml:base contributions.
func (c *canonicalizer) fixupXMLBase(e *helium.Element, entries *[]attrSortEntry) {
	if parentNode := e.Parent(); parentNode != nil {
		if parentElem, ok := helium.AsNode[*helium.Element](parentNode); ok {
			if c.isVisible(parentElem) {
				return // Parent visible, no fixup needed
			}
		}
	}

	// Compute E's effective base URI
	eBase := c.effectiveBaseURI(e)

	// Find nearest visible ancestor's effective base URI
	vaBase := ""
	for n := e.Parent(); n != nil; n = n.Parent() {
		if anc, ok := helium.AsNode[*helium.Element](n); ok {
			if c.isVisible(anc) {
				vaBase = c.effectiveBaseURI(anc)
				break
			}
		}
	}
	if vaBase == "" {
		// No visible ancestor: use document base URI
		vaBase = c.documentBaseURI()
	}

	if eBase == vaBase {
		// Remove xml:base from entries if present (same base, no need)
		c.removeXMLBaseEntry(entries)
		return
	}

	// Compute the relative xml:base value
	xmlBaseValue := relativizeURI(vaBase, eBase)

	// Replace or add xml:base in entries
	c.setXMLBaseEntry(entries, xmlBaseValue)
}

// effectiveBaseURI computes the effective base URI for an element
// by resolving xml:base attributes from the document root down.
func (c *canonicalizer) effectiveBaseURI(e *helium.Element) string {
	// Collect ancestor chain
	var chain []*helium.Element
	for n := helium.Node(e); n != nil; n = n.Parent() {
		if elem, ok := helium.AsNode[*helium.Element](n); ok {
			chain = append(chain, elem)
		}
	}

	// Start with document base URI
	base := c.documentBaseURI()

	// Process from outermost to innermost
	for _, v := range slices.Backward(chain) {
		elem := v
		xmlBase := getXMLBaseAttr(elem)
		if xmlBase == "" {
			continue
		}

		baseURL, err := url.Parse(base)
		if err != nil {
			base = xmlBase
			continue
		}
		ref, err := url.Parse(xmlBase)
		if err != nil {
			continue
		}
		base = baseURL.ResolveReference(ref).String()
	}
	return base
}

// documentBaseURI returns the document's base URI suitable for RFC 3986
// relative-reference resolution. If the configured base URI is already an
// absolute URI (it has a scheme and authority, e.g. "http://example.com/..."),
// it is preserved as-is so that xml:base fixup uses proper URI semantics. Only
// plain filesystem paths are converted to a file:// URL.
func (c *canonicalizer) documentBaseURI() string {
	if c.baseURI == "" {
		return ""
	}
	// An absolute URI (one with a scheme, e.g. "http://example.com/...",
	// "file:/tmp/doc.xml", or "urn:...") must not be rewritten as a filesystem
	// path. url.Parse treats a single-letter Windows drive prefix (e.g.
	// "c:\dir") as a scheme, so exclude single-letter schemes to keep treating
	// drive-letter paths as filesystem paths.
	if u, err := url.Parse(c.baseURI); err == nil && u.IsAbs() && len(u.Scheme) > 1 {
		return c.baseURI
	}
	// Convert file path to URL for proper URI resolution.
	absPath, err := filepath.Abs(c.baseURI)
	if err != nil {
		return c.baseURI
	}
	return "file://" + absPath
}

// xmlBaseLocalName is the local name of the xml:base attribute.
const xmlBaseLocalName = "base"

// getXMLBaseAttr returns the xml:base attribute value of an element, or "".
func getXMLBaseAttr(e *helium.Element) string {
	for _, attr := range e.Attributes() {
		if attr.LocalName() == xmlBaseLocalName && attr.URI() == lexicon.NamespaceXML {
			return attr.Value()
		}
	}
	return ""
}

// relativizeURI computes a relative URI from base to target.
// If the URIs have different schemes or hosts, returns the absolute target.
func relativizeURI(base, target string) string {
	baseURL, err := url.Parse(base)
	if err != nil {
		return target
	}
	targetURL, err := url.Parse(target)
	if err != nil {
		return target
	}

	// Different scheme or authority: return absolute
	if baseURL.Scheme != targetURL.Scheme || baseURL.Host != targetURL.Host {
		return target
	}

	// Opaque / non-hierarchical URIs (e.g. "urn:target") carry their data in
	// Opaque rather than Path, so there is no path to relativize. Path-based
	// relativization would yield a meaningless (and possibly empty) result, so
	// return the target absolutely instead.
	if baseURL.Opaque != "" || targetURL.Opaque != "" {
		return target
	}

	basePath := baseURL.Path
	targetPath := targetURL.Path

	// Find common directory prefix
	baseDir := basePath[:strings.LastIndex(basePath, "/")+1]

	// Find longest common directory prefix
	common := ""
	bi, ti := 0, 0
	for bi < len(baseDir) && ti < len(targetPath) {
		if baseDir[bi] != targetPath[ti] {
			break
		}
		if baseDir[bi] == '/' {
			common = baseDir[:bi+1]
		}
		bi++
		ti++
	}
	// Count remaining directories in base after common prefix
	remaining := baseDir[len(common):]
	ups := 0
	for _, ch := range remaining {
		if ch == '/' {
			ups++
		}
	}

	// Build the path-relative part and the query/fragment suffix separately so
	// the relative-reference candidates below can be assembled and tested
	// independently.
	pathRelative := strings.Repeat("../", ups) + targetPath[len(common):]

	suffix := ""
	if targetURL.RawQuery != "" || targetURL.ForceQuery {
		suffix += "?" + targetURL.RawQuery
	}
	if targetURL.Fragment != "" {
		suffix += "#" + targetURL.EscapedFragment()
	}

	// roundTrips reports whether resolving the candidate reference against the
	// base reproduces the exact target. A relative reference is only correct if
	// it round-trips; otherwise it would silently change the URI.
	roundTrips := func(ref string) bool {
		candidate, err := url.Parse(ref)
		if err != nil {
			return false
		}
		return baseURL.ResolveReference(candidate).String() == targetURL.String()
	}

	// Primary candidate: the relativized path plus the carried query/fragment.
	candidate := pathRelative + suffix
	if roundTrips(candidate) {
		return candidate
	}

	// When the path part is empty the target lives in the base document's own
	// directory. The bare suffix (e.g. "?q=1#frag") resolves against the base
	// *document* (re-using its filename), so it points at the base document with
	// the suffix attached rather than the directory. If that does not round-trip
	// to the target, a leading "." selects the directory itself: "./?q=1#frag"
	// (or ".?q=1#frag" when no path remains) resolves to the directory plus the
	// suffix, which is the correct minimal relative reference for a same-directory
	// target carrying only a query/fragment.
	if pathRelative == "" {
		dotCandidate := "." + suffix
		if roundTrips(dotCandidate) {
			return dotCandidate
		}
	}

	// No relative candidate round-trips; emit the absolute target so the
	// canonical xml:base resolves to the exact target.
	return targetURL.String()
}

// removeXMLBaseEntry removes any xml:base entry from the attr list.
func (c *canonicalizer) removeXMLBaseEntry(entries *[]attrSortEntry) {
	result := (*entries)[:0]
	for _, entry := range *entries {
		if entry.nsURI == lexicon.NamespaceXML && entry.localName == xmlBaseLocalName {
			continue
		}
		result = append(result, entry)
	}
	*entries = result
}

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
