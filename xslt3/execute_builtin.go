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
	if md := ec.stylesheet.modeDefs[mode]; md != nil {
		if md.OnNoMatch != "" {
			onNoMatch = md.OnNoMatch
		}
	} else if mode == "" {
		if md := ec.stylesheet.modeDefs[modeDefault]; md != nil {
			if md.OnNoMatch != "" {
				onNoMatch = md.OnNoMatch
			}
		}
	}
	return ec.applyOnNoMatch(ctx, node, mode, onNoMatch, paramValues...)
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
			srcElem := node.(*helium.Element)
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
		text, err := ec.resultDoc.CreateText(node.Content())
		if err != nil {
			return err
		}
		return ec.addNode(text)
	case helium.AttributeNode:
		attr, ok := node.(*helium.Attribute)
		if !ok {
			return nil
		}
		text, err := ec.resultDoc.CreateText([]byte(attr.Value()))
		if err != nil {
			return err
		}
		return ec.addNode(text)
	default:
		return nil
	}
}

func (ec *execContext) onNoMatchShallowCopy(ctx context.Context, node helium.Node, mode string, paramValues ...map[string]xpath3.Sequence) error {
	switch node.Type() {
	case helium.DocumentNode:
		for child := range helium.Children(node) {
			if err := ec.applyTemplates(ctx, child, mode, paramValues...); err != nil {
				return err
			}
		}
		return nil
	case helium.ElementNode:
		srcElem := node.(*helium.Element)
		newElem, err := ec.resultDoc.CreateElement(srcElem.LocalName())
		if err != nil {
			return err
		}
		for _, ns := range srcElem.Namespaces() {
			_ = newElem.DeclareNamespace(ns.Prefix(), ns.URI())
		}
		if srcElem.URI() != "" {
			_ = newElem.SetActiveNamespace(srcElem.Prefix(), srcElem.URI())
		}
		if err := ec.addNode(newElem); err != nil {
			return err
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
		text, err := ec.resultDoc.CreateText(node.Content())
		if err != nil {
			return err
		}
		return ec.addNode(text)
	case helium.CommentNode:
		comment, err := ec.resultDoc.CreateComment(node.Content())
		if err != nil {
			return err
		}
		return ec.addNode(comment)
	case helium.ProcessingInstructionNode:
		pi, err := ec.resultDoc.CreatePI(node.Name(), string(node.Content()))
		if err != nil {
			return err
		}
		return ec.addNode(pi)
	case helium.AttributeNode:
		attr := node.(*helium.Attribute)
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
		// For document nodes, deep-copy each child.
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
		return ec.addNode(copied)
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
	elem := parent.(*helium.Element)
	if ec.isElementStripped(elem) {
		return true
	}
	// Per XDM 3.1 Section 6.7.1: if the element is validated as having
	// element-only content (from a DTD), whitespace-only text node children
	// are stripped. This ensures documents parsed with DTD element content
	// models behave correctly even without xsl:strip-space.
	return hasElementOnlyContent(elem)
}

// isElementStripped checks if an element matches strip-space rules.
// preserve-space overrides strip-space for the same element.
func (ec *execContext) isElementStripped(elem *helium.Element) bool {
	ss := ec.stylesheet
	if len(ss.stripSpace) == 0 {
		return false
	}

	stripped := false
	stripPriority := -1
	for _, nt := range ss.stripSpace {
		if matchSpaceNameTest(nt, elem, ss.namespaces) {
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
	for _, nt := range ss.preserveSpace {
		if matchSpaceNameTest(nt, elem, ss.namespaces) {
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

// matchSpaceNameTest checks if an element matches a strip/preserve-space NameTest pattern.
func matchSpaceNameTest(nt NameTest, elem *helium.Element, nsBindings map[string]string) bool {
	if nt.Local == "*" && nt.Prefix == "" {
		return true // "*" matches all
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
	// "local" matches elements with that local name (no namespace)
	return elem.LocalName() == nt.Local && elem.URI() == ""
}

// nameTestPriority returns the priority of a NameTest for conflict resolution.
// Specific names > prefix:* > *
func nameTestPriority(nt NameTest) int {
	if nt.Local == "*" && nt.Prefix == "" {
		return 0 // "*"
	}
	if nt.Local == "*" {
		return 1 // "prefix:*"
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
				helium.UnlinkNode(child)
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
