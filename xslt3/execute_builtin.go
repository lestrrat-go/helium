package xslt3

import (
	"context"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/xpath3"
)

// applyBuiltinRules applies the built-in template rules per XSLT spec.
func (ec *execContext) applyBuiltinRules(ctx context.Context, node helium.Node, mode string, paramValues ...map[string]xpath3.Sequence) error {
	// Check for xsl:mode on-no-match behavior
	onNoMatch := onNoMatchTextOnlyCopy // XSLT 3.0 default
	modeDefs := ec.effectiveModeDefs()
	if md := modeDefs[mode]; md != nil {
		if md.OnNoMatch != "" {
			onNoMatch = md.OnNoMatch
		}
	} else if mode == "" {
		if md := modeDefs[modeDefault]; md != nil {
			if md.OnNoMatch != "" {
				onNoMatch = md.OnNoMatch
			}
		}
	}
	// Built-in template rules apply templates in the same mode. When the
	// current mode is "" (the unnamed mode), use "#unnamed" so that recursive
	// applyTemplates calls don't re-resolve "" to the stylesheet's default-mode.
	builtinMode := mode
	if builtinMode == "" {
		builtinMode = modeUnnamed
	}
	return ec.applyOnNoMatch(ctx, node, builtinMode, onNoMatch, paramValues...)
}

func (ec *execContext) applyOnNoMatch(ctx context.Context, node helium.Node, mode, behavior string, paramValues ...map[string]xpath3.Sequence) error {
	switch behavior {
	case onNoMatchShallowCopy:
		return ec.onNoMatchShallowCopy(ctx, node, mode, paramValues...)
	case onNoMatchDeepCopy:
		return ec.onNoMatchDeepCopy(node)
	case onNoMatchShallowSkip:
		if node.Type() == helium.ElementNode {
			// XSLT 3.0: shallow-skip for elements applies templates to
			// attributes and children (but does not copy the element).
			srcElem := node.(*helium.Element) //nolint:forcetypeassert
			for _, attr := range srcElem.Attributes() {
				if err := ec.applyTemplates(ctx, attr, mode, paramValues...); err != nil {
					return err
				}
			}
			for child := range helium.Children(node) {
				if err := ec.applyTemplates(ctx, child, mode, paramValues...); err != nil {
					return err
				}
			}
		} else if node.Type() == helium.DocumentNode {
			for child := range helium.Children(node) {
				if err := ec.applyTemplates(ctx, child, mode, paramValues...); err != nil {
					return err
				}
			}
		}
		return nil
	case onNoMatchDeepSkip:
		// Per XSLT 3.0 spec (bug #30219): the built-in template rule for
		// document nodes always processes children, even with deep-skip.
		if node.Type() == helium.DocumentNode {
			for child := range helium.Children(node) {
				if err := ec.applyTemplates(ctx, child, mode, paramValues...); err != nil {
					return err
				}
			}
		}
		return nil
	case onNoMatchFail:
		// Per XSLT 3.0 spec (bug #30219): the built-in template rule for
		// document nodes always processes children, even with on-no-match=fail.
		if node.Type() == helium.DocumentNode {
			for child := range helium.Children(node) {
				if err := ec.applyTemplates(ctx, child, mode, paramValues...); err != nil {
					return err
				}
			}
			return nil
		}
		return dynamicError(errCodeXTDE0555, "no matching template in mode %q (on-no-match=fail)", mode)
	default: // "text-only-copy"
		return ec.onNoMatchTextOnlyCopy(ctx, node, mode, paramValues...)
	}
}

func (ec *execContext) onNoMatchTextOnlyCopy(ctx context.Context, node helium.Node, mode string, paramValues ...map[string]xpath3.Sequence) error {
	switch node.Type() {
	case helium.DocumentNode, helium.ElementNode:
		for child := range helium.Children(node) {
			if err := ec.applyTemplates(ctx, child, mode, paramValues...); err != nil {
				return err
			}
		}
		return nil
	case helium.TextNode, helium.CDATASectionNode:
		if ec.shouldStripWhitespace(node) {
			return nil
		}
		text := ec.resultDoc.CreateText(node.Content())
		return ec.addNode(text)
	case helium.AttributeNode:
		attr, ok := node.(*helium.Attribute)
		if !ok {
			return nil
		}
		text := ec.resultDoc.CreateText([]byte(attr.Value()))
		return ec.addNode(text)
	default:
		return nil
	}
}

func (ec *execContext) onNoMatchShallowCopy(ctx context.Context, node helium.Node, mode string, paramValues ...map[string]xpath3.Sequence) error {
	switch node.Type() {
	case helium.DocumentNode:
		// The built-in template for document nodes in shallow-copy mode
		// creates a new document node and applies templates to the children.
		// When in sequence mode (e.g., inside a function returning
		// document-node()), wrap the result in a document node so the
		// return type matches. Otherwise, emit children directly.
		out := ec.currentOutput()
		if out.sequenceMode || (out.captureItems && out.current != nil && out.current.Type() != helium.DocumentNode) {
			tmpDoc := helium.NewDefaultDocument()
			// Preserve source document URL for fn:base-uri on the copy.
			if srcDoc, ok := node.(*helium.Document); ok {
				tmpDoc.SetURL(srcDoc.URL())
			}
			frame := &outputFrame{doc: tmpDoc, current: tmpDoc}
			ec.outputStack = append(ec.outputStack, frame)
			for child := range helium.Children(node) {
				if err := ec.applyTemplates(ctx, child, mode, paramValues...); err != nil {
					ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
					return err
				}
			}
			ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
			// Always capture the document (even empty) so fn:base-uri works.
			out.pendingItems = append(out.pendingItems, xpath3.NodeItem{Node: tmpDoc})
			out.noteOutput()
			return nil
		}
		for child := range helium.Children(node) {
			if err := ec.applyTemplates(ctx, child, mode, paramValues...); err != nil {
				return err
			}
		}
		return nil
	case helium.ElementNode:
		srcElem := node.(*helium.Element) //nolint:forcetypeassert
		newElem := ec.resultDoc.CreateElement(srcElem.LocalName())
		for _, ns := range srcElem.Namespaces() {
			_ = newElem.DeclareNamespace(ns.Prefix(), ns.URI())
		}
		if srcElem.URI() != "" {
			_ = newElem.SetActiveNamespace(srcElem.Prefix(), srcElem.URI())
		}
		if err := ec.addNode(newElem); err != nil {
			return err
		}
		if newElem.Parent() == nil {
			if srcBase := helium.NodeGetBase(srcElem.OwnerDocument(), srcElem); srcBase != "" {
				helium.SetNodeBaseURI(newElem, srcBase)
			}
		}
		out := ec.currentOutput()
		savedCurrent := out.current
		savedSeqMode := out.sequenceMode
		savedCapture := out.captureItems
		out.current = newElem
		// Temporarily disable sequenceMode and captureItems so that
		// children are added to this element normally (not captured as
		// separate items in the sequence). This mirrors execElement.
		out.sequenceMode = false
		out.captureItems = false
		defer func() {
			out.current = savedCurrent
			out.sequenceMode = savedSeqMode
			out.captureItems = savedCapture
		}()
		// Apply templates to attributes first (so user templates can
		// intercept attribute nodes, e.g. match="w/@id"), then children.
		for _, attr := range srcElem.Attributes() {
			if err := ec.applyTemplates(ctx, attr, mode, paramValues...); err != nil {
				return err
			}
		}
		for child := range helium.Children(srcElem) {
			if err := ec.applyTemplates(ctx, child, mode, paramValues...); err != nil {
				return err
			}
		}
		return nil
	case helium.TextNode, helium.CDATASectionNode:
		text := ec.resultDoc.CreateText(node.Content())
		return ec.addNode(text)
	case helium.CommentNode:
		comment := ec.resultDoc.CreateComment(node.Content())
		return ec.addNode(comment)
	case helium.ProcessingInstructionNode:
		pi := ec.resultDoc.CreatePI(node.Name(), string(node.Content()))
		return ec.addNode(pi)
	case helium.AttributeNode:
		attr := node.(*helium.Attribute) //nolint:forcetypeassert
		out := ec.currentOutput()
		if outElem, ok := out.current.(*helium.Element); ok {
			copyAttributeToElement(outElem, attr)
			out.noteOutput()
		}
		return nil
	default:
		return nil
	}
}

func (ec *execContext) onNoMatchDeepCopy(node helium.Node) error {
	// Deep copy: copy the entire subtree to the output without template matching.
	switch node.Type() {
	case helium.DocumentNode:
		out := ec.currentOutput()
		if out.sequenceMode || (out.captureItems && out.current != nil && out.current.Type() != helium.DocumentNode) {
			srcDoc, _ := node.(*helium.Document)
			copied, err := helium.CopyNode(node, ec.resultDoc)
			if err != nil {
				return err
			}
			if srcDoc != nil {
				if copiedDoc, ok := copied.(*helium.Document); ok {
					copiedDoc.SetURL(srcDoc.URL())
				}
			}
			out.pendingItems = append(out.pendingItems, xpath3.NodeItem{Node: copied})
			out.noteOutput()
			return nil
		}
		for child := range helium.Children(node) {
			if err := ec.onNoMatchDeepCopy(child); err != nil {
				return err
			}
		}
		return nil
	case helium.ElementNode, helium.TextNode, helium.CDATASectionNode,
		helium.CommentNode, helium.ProcessingInstructionNode:
		copied, err := helium.CopyNode(node, ec.resultDoc)
		if err != nil {
			return err
		}
		if err := ec.addNode(copied); err != nil {
			return err
		}
		if node.Type() == helium.ElementNode && copied.Parent() == nil {
			srcElem := node.(*helium.Element) //nolint:forcetypeassert
			if srcBase := helium.NodeGetBase(srcElem.OwnerDocument(), srcElem); srcBase != "" {
				helium.SetNodeBaseURI(copied, srcBase)
			}
		}
		return nil
	case helium.AttributeNode:
		attr, ok := node.(*helium.Attribute)
		if !ok {
			return nil
		}
		out := ec.currentOutput()
		if outElem, ok := out.current.(*helium.Element); ok {
			copyAttributeToElement(outElem, attr)
			out.noteOutput()
			return nil
		}
		return nil
	default:
		return nil
	}
}

// shouldStripWhitespace returns true if a text node is whitespace-only
// and its parent element matches a strip-space pattern or has element-only
// content per its DTD declaration (XDM 3.1 Section 6.7.1).
func (ec *execContext) shouldStripWhitespace(node helium.Node) bool {
	if normalizeNode(node) == nil {
		return false
	}
	// Only strip text/CDATA nodes, not elements or other node types
	if node.Type() != helium.TextNode && node.Type() != helium.CDATASectionNode {
		return false
	}
	content := node.Content()
	// Check if whitespace-only
	for _, b := range content {
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			return false
		}
	}
	// Check parent element against strip/preserve space rules
	parent := node.Parent()
	if parent == nil || parent.Type() != helium.ElementNode {
		return false
	}
	elem := parent.(*helium.Element) //nolint:forcetypeassert
	// xml:space="preserve" on the element or an ancestor overrides strip-space.
	// Walk up to find the nearest xml:space declaration.
	if inheritedXMLSpace(elem) == "preserve" {
		return false
	}
	if ec.isElementStripped(elem) {
		return true
	}
	// Per XDM 3.1 Section 6.7.1: if the element is validated as having
	// element-only content (from a DTD), whitespace-only text node children
	// are stripped. This ensures documents parsed with DTD element content
	// models behave correctly even without xsl:strip-space.
	return hasElementOnlyContent(elem)
}

// inheritedXMLSpace walks up the ancestor chain to find the nearest
// xml:space attribute and returns its value ("preserve" or "default").
// Returns "" if no xml:space is declared on any ancestor.
func inheritedXMLSpace(elem *helium.Element) string {
	for n := helium.Node(elem); n != nil; n = n.Parent() {
		e, ok := n.(*helium.Element)
		if !ok {
			continue
		}
		if v, ok := e.GetAttribute("xml:space"); ok {
			return v
		}
	}
	return ""
}

// isElementStripped checks if an element matches strip-space rules.
// preserve-space overrides strip-space for the same element.
// Uses the effective (package-scoped) strip/preserve rules.
func (ec *execContext) isElementStripped(elem *helium.Element) bool {
	stripRules := ec.effectiveStripSpace()
	if len(stripRules) == 0 {
		return false
	}

	nsBindings := ec.effectiveStripNamespaces()

	stripped := false
	stripPriority := -1
	for _, nt := range stripRules {
		if matchSpaceNameTest(nt, elem, nsBindings) {
			p := nameTestPriority(nt)
			if p > stripPriority {
				stripPriority = p
				stripped = true
			}
		}
	}

	if !stripped {
		return false
	}

	// Check if preserve-space overrides
	for _, nt := range ec.effectivePreserveSpace() {
		if matchSpaceNameTest(nt, elem, nsBindings) {
			p := nameTestPriority(nt)
			if p >= stripPriority {
				return false
			}
		}
	}
	return true
}

// hasElementOnlyContent returns true if the element has element-only content
// per its DTD declaration. This is used to strip whitespace-only text nodes
// in elements with element-only content, per XDM 3.1 Section 6.7.1.
func hasElementOnlyContent(elem *helium.Element) bool {
	doc := elem.OwnerDocument()
	if doc == nil {
		return false
	}
	name := elem.LocalName()
	prefix := elem.Prefix()
	// Check both internal and external DTD subsets
	for _, dtd := range []*helium.DTD{doc.IntSubset(), doc.ExtSubset()} {
		if dtd == nil {
			continue
		}
		if edecl, ok := dtd.LookupElement(name, prefix); ok {
			return edecl.DeclType() == enum.ElementElementType
		}
	}
	return false
}

// matchSpaceNameTest checks if an element matches a strip/preserve-space nameTest pattern.
func matchSpaceNameTest(nt nameTest, elem *helium.Element, nsBindings map[string]string) bool {
	if nt.Local == "*" && nt.Prefix == "" {
		return true // "*" matches all
	}
	if nt.Prefix == "*" {
		// "*:NCName" matches elements with given local name in any namespace
		return elem.LocalName() == nt.Local
	}
	if nt.Local == "*" && nt.Prefix != "" {
		// "prefix:*" matches elements in that namespace
		nsURI := nsBindings[nt.Prefix]
		return elem.URI() == nsURI
	}
	if nt.Prefix != "" {
		// "prefix:local" matches specific element in namespace
		nsURI := nsBindings[nt.Prefix]
		return elem.LocalName() == nt.Local && elem.URI() == nsURI
	}
	// Unprefixed name: use resolved URI if xpath-default-namespace was applied
	if nt.HasURI {
		return elem.LocalName() == nt.Local && elem.URI() == nt.URI
	}
	// "local" matches elements with that local name (no namespace)
	return elem.LocalName() == nt.Local && elem.URI() == ""
}

// nameTestPriority returns the priority of a nameTest for conflict resolution.
// Specific names > prefix:* or *:NCName > *
func nameTestPriority(nt nameTest) int {
	if nt.Local == "*" && nt.Prefix == "" {
		return 0 // "*"
	}
	if nt.Local == "*" || nt.Prefix == "*" {
		return 1 // "prefix:*" or "*:NCName"
	}
	return 2 // specific name
}

// stripWhitespaceFromDoc removes whitespace-only text nodes from a document
// tree according to the stylesheet's xsl:strip-space and xsl:preserve-space rules.
// This is called when loading documents so that XPath evaluation sees the
// correctly stripped tree.
func (ec *execContext) stripWhitespaceFromDoc(doc *helium.Document) {
	ec.stripWhitespaceFromNode(doc)
}

func (ec *execContext) stripWhitespaceFromNode(root helium.Node) {
	// Use an explicit stack to avoid deep recursion on large documents.
	stack := make([]helium.Node, 0, 32)
	stack = append(stack, root)
	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		child := node.FirstChild()
		for child != nil {
			next := child.NextSibling()
			if ec.shouldStripWhitespace(child) {
				helium.UnlinkNode(child.(helium.MutableNode)) //nolint:forcetypeassert
			} else if child.FirstChild() != nil {
				stack = append(stack, child)
			}
			child = next
		}
	}
}

// selectDefaultNodes returns the default node-set for apply-templates
// (child::node()).
func selectDefaultNodes(node helium.Node) []helium.Node {
	if normalizeNode(node) == nil {
		return nil
	}
	var nodes []helium.Node
	for child := range helium.Children(node) {
		nodes = append(nodes, child)
	}
	return nodes
}
