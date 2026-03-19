package xslt3

import (
	"context"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

func (ec *execContext) execCopy(ctx context.Context, inst *CopyInst) error {
	// Resolve shadow attributes (AVTs) at runtime; fall back to static values.
	copyNS := inst.CopyNamespaces
	inheritNS := inst.InheritNamespaces
	if inst.CopyNamespacesAVT != nil {
		v, err := inst.CopyNamespacesAVT.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		if b, ok := parseXSDBool(v); ok {
			copyNS = b
		}
	}
	if inst.InheritNamespacesAVT != nil {
		v, err := inst.InheritNamespacesAVT.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		if b, ok := parseXSDBool(v); ok {
			inheritNS = b
		}
	}

	if inst.Select != nil {
		// XSLT 3.0: xsl:copy with select — iterate over selected items
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		seq := result.Sequence()
		if len(seq) == 0 {
			return nil // empty sequence: skip body
		}
		for _, item := range seq {
			switch v := item.(type) {
			case xpath3.NodeItem:
				// Set focus to the selected node (singleton focus)
				savedCtx := ec.contextNode
				savedCur := ec.currentNode
				savedPos := ec.position
				savedSize := ec.size
				ec.contextNode = v.Node
				ec.currentNode = v.Node
				ec.position = 1
				ec.size = 1
				err := ec.execCopyNode(ctx, v.Node, copyNodeOpts{
					body:              inst.Body,
					copyNamespaces:    copyNS,
					inheritNamespaces: inheritNS,
				})
				ec.contextNode = savedCtx
				ec.currentNode = savedCur
				ec.position = savedPos
				ec.size = savedSize
				if err != nil {
					return err
				}
			case xpath3.AtomicValue:
				// Atomic values: output as text, body is not evaluated
				s, err := xpath3.AtomicToString(v)
				if err != nil {
					return err
				}
				text, err := ec.resultDoc.CreateText([]byte(s))
				if err != nil {
					return err
				}
				if err := ec.addNode(text); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if ec.contextNode == nil {
		return dynamicError(errCodeXTTE0945, "xsl:copy: no context item")
	}
	if err := ec.execCopyNode(ctx, ec.contextNode, copyNodeOpts{
		body:              inst.Body,
		useAttrSets:       inst.UseAttrSets,
		copyNamespaces:    copyNS,
		inheritNamespaces: inheritNS,
	}); err != nil {
		return err
	}

	// Apply validation if specified and context node is an element.
	if v := ec.effectiveValidation(inst.Validation); v != "" && v != "preserve" {
		out := ec.currentOutput()
		// The most recently added child of the current output is the copy.
		if copied := out.current.LastChild(); copied != nil {
			if copiedElem, ok := copied.(*helium.Element); ok {
				if err := ec.validateConstructedElement(ctx, copiedElem, v); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// effectiveValidation returns the validation mode for a copy/copy-of instruction,
// falling back to the stylesheet default when the instruction has none.
func (ec *execContext) effectiveValidation(instValidation string) string {
	if instValidation != "" {
		return instValidation
	}
	return ec.stylesheet.defaultValidation
}

type copyNodeOpts struct {
	body              []Instruction
	useAttrSets       []string
	copyNamespaces    bool
	inheritNamespaces bool
}

func (ec *execContext) execCopyNode(ctx context.Context, node helium.Node, opts copyNodeOpts) error {
	if node == nil {
		return nil
	}

	switch node.Type() {
	case helium.ElementNode:
		srcElem := node.(*helium.Element)
		// Use LocalName to avoid prefix doubling with SetActiveNamespace
		elem, err := ec.resultDoc.CreateElement(srcElem.LocalName())
		if err != nil {
			return err
		}

		if opts.copyNamespaces {
			// Copy all in-scope namespace declarations (not just those
			// directly declared on the element).  This matches the XSLT 3.0
			// spec for xsl:copy with copy-namespaces="yes".
			for _, ns := range collectInScopeNamespaces(srcElem) {
				if ns.URI() == "" {
					continue // skip undeclarations
				}
				if err := elem.DeclareNamespace(ns.Prefix(), ns.URI()); err != nil {
					return err
				}
			}
		}
		if srcElem.URI() != "" {
			// Always declare the element's own namespace
			if !hasNSDecl(elem, srcElem.Prefix(), srcElem.URI()) {
				if err := elem.DeclareNamespace(srcElem.Prefix(), srcElem.URI()); err != nil {
					return err
				}
			}
			if err := elem.SetActiveNamespace(srcElem.Prefix(), srcElem.URI()); err != nil {
				return err
			}
		}

		if err := ec.addNode(elem); err != nil {
			return err
		}

		// Execute body in element context
		out := ec.currentOutput()
		savedCurrent := out.current
		savedPrevAtomic := out.prevWasAtomic
		out.current = elem
		out.prevWasAtomic = false
		defer func() {
			out.current = savedCurrent
			out.prevWasAtomic = savedPrevAtomic
		}()

		// Apply attribute sets if specified
		if len(opts.useAttrSets) > 0 {
			if err := ec.applyAttributeSets(ctx, opts.useAttrSets); err != nil {
				return err
			}
		}

		if err := ec.executeSequenceConstructor(ctx, opts.body); err != nil {
			return err
		}

		// inherit-namespaces="no": undeclare parent namespaces on direct
		// child elements so they do not inherit them via the DOM tree.
		if !opts.inheritNamespaces {
			undeclareInheritedNamespaces(elem)
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
		pi := node.(*helium.ProcessingInstruction)
		newPI, err := ec.resultDoc.CreatePI(pi.Name(), string(pi.Content()))
		if err != nil {
			return err
		}
		return ec.addNode(newPI)

	case helium.DocumentNode:
		// xsl:copy of a document node creates a new document node.
		// DTD information (including unparsed entities) is preserved.
		srcDoc, _ := node.(*helium.Document)
		newDoc := helium.NewDefaultDocument()
		if srcDoc != nil {
			// Copy DTD information to preserve unparsed entities.
			helium.CopyDTDInfo(srcDoc, newDoc)
			newDoc.SetURL(srcDoc.URL())
		}

		out := ec.currentOutput()
		if out.captureItems {
			// We're inside a variable or function body — capture the
			// document node as an item.
			savedDoc := ec.resultDoc
			savedOutput := out.current
			ec.resultDoc = newDoc
			docRoot := newDoc.DocumentElement()
			if docRoot == nil {
				// No doc element yet; use the document node itself as output target.
				out.current = newDoc
			}
			if err := ec.executeSequenceConstructor(ctx, opts.body); err != nil {
				ec.resultDoc = savedDoc
				out.current = savedOutput
				return err
			}
			ec.resultDoc = savedDoc
			out.current = savedOutput
			out.pendingItems = append(out.pendingItems, xpath3.NodeItem{Node: newDoc})
			out.noteOutput()
			return nil
		}

		// Not in capture mode — process body in current context.
		return ec.executeSequenceConstructor(ctx, opts.body)

	case helium.AttributeNode:
		attr := node.(*helium.Attribute)
		out := ec.currentOutput()
		elem, ok := out.current.(*helium.Element)
		if !ok {
			// XTDE0410: adding attribute to non-element
			return dynamicError(errCodeXTDE0410,
				"cannot add attribute %s to a non-element node", attr.Name())
		}
		if elem.FirstChild() != nil {
			// XTDE0410: adding attribute after child content
			return dynamicError(errCodeXTDE0410,
				"cannot add attribute %s after child nodes have been added", attr.Name())
		}
		if err := copyAttributeToElement(elem, attr); err != nil {
			return err
		}
		out.noteOutput()
		return nil
	}

	return nil
}

func (ec *execContext) execCopyOf(ctx context.Context, inst *CopyOfInst) error {
	xpathCtx := ec.newXPathContext(ec.contextNode)
	result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
	if err != nil {
		return err
	}

	// Resolve shadow attribute at runtime
	copyNS := inst.CopyNamespaces
	if inst.CopyNamespacesAVT != nil {
		v, err := inst.CopyNamespacesAVT.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		if b, ok := parseXSDBool(v); ok {
			copyNS = b
		}
	}

	effectiveVal := ec.effectiveValidation(inst.Validation)
	preserve := effectiveVal == "preserve"

	out := ec.currentOutput()
	prevWasAtomic := out.prevWasAtomic
	seq := flattenArraysInSequence(result.Sequence())
	for _, item := range seq {
		switch v := item.(type) {
		case xpath3.NodeItem:
			prevWasAtomic = false
			if err := ec.copyNodeToOutput(v.Node, copyNS); err != nil {
				return err
			}
			if preserve {
				last := out.current.LastChild()
				if last != nil {
					ec.deepTransferAnnotations(v.Node, last)
				}
			} else if effectiveVal == "strict" || effectiveVal == "lax" || effectiveVal == "strip" {
				// Apply validation/strip to the most recently added node in output.
				if copied := out.current.LastChild(); copied != nil {
					if copiedElem, ok := copied.(*helium.Element); ok {
						if err := ec.validateConstructedElement(ctx, copiedElem, effectiveVal); err != nil {
							return err
						}
					}
				}
			}
		case xpath3.AtomicValue:
			s, err := xpath3.AtomicToString(v)
			if err != nil {
				return err
			}
			if prevWasAtomic {
				sep, tErr := ec.resultDoc.CreateText([]byte(" "))
				if tErr != nil {
					return tErr
				}
				if err := ec.addNode(sep); err != nil {
					return err
				}
			}
			text, err := ec.resultDoc.CreateText([]byte(s))
			if err != nil {
				return err
			}
			if err := ec.addNode(text); err != nil {
				return err
			}
			prevWasAtomic = true
		}
	}
	out.prevWasAtomic = prevWasAtomic
	return nil
}

// copyNodeToOutput copies a node to the current output, handling document
// and attribute nodes specially. When copyNamespaces is false, namespace
// declarations are not copied onto element nodes (only those required by
// the element name and attribute names are preserved).
func (ec *execContext) copyNodeToOutput(node helium.Node, copyNamespaces ...bool) error {
	copyNS := true
	if len(copyNamespaces) > 0 {
		copyNS = copyNamespaces[0]
	}
	switch node.Type() {
	case helium.DocumentNode:
		// Copy children of the document node
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if err := ec.copyNodeToOutput(child, copyNS); err != nil {
				return err
			}
		}
		return nil
	case helium.AttributeNode:
		attr, ok := node.(*helium.Attribute)
		if !ok {
			return nil
		}
		out := ec.currentOutput()
		elem, ok := out.current.(*helium.Element)
		if !ok {
			// XTDE0410: adding attribute to non-element
			return dynamicError(errCodeXTDE0410,
				"cannot add attribute %s to a non-element node", attr.Name())
		}
		if elem.FirstChild() != nil {
			// XTDE0410: adding attribute after child content
			return dynamicError(errCodeXTDE0410,
				"cannot add attribute %s after child nodes have been added", attr.Name())
		}
		if err := copyAttributeToElement(elem, attr); err != nil {
			return err
		}
		out.noteOutput()
		return nil
	case helium.NamespaceNode:
		// Namespace nodes are copied as namespace declarations on the current element.
		nsw, ok := node.(*helium.NamespaceNodeWrapper)
		if !ok {
			return nil
		}
		out := ec.currentOutput()
		elem, ok := out.current.(*helium.Element)
		if !ok {
			return nil
		}
		prefix := nsw.Name()
		uri := string(nsw.Content())
		return elem.DeclareNamespace(prefix, uri)
	case helium.DTDNode:
		// DTDs are not copied to the result tree in XSLT
		return nil
	default:
		if !copyNS && node.Type() == helium.ElementNode {
			return ec.copyElementNoNamespaces(node.(*helium.Element))
		}
		copied, err := helium.CopyNode(node, ec.resultDoc)
		if err != nil {
			return err
		}
		if err := ec.addNode(copied); err != nil {
			return err
		}
		// After inserting a copied element into a new context, ensure
		// namespace declarations are correct relative to the new parent.
		if elem, ok := copied.(*helium.Element); ok {
			ec.fixNamespacesAfterCopy(elem)
		}
		return nil
	}
}

// copyElementNoNamespaces deep-copies an element but omits namespace
// declarations that are not required by the element or attribute names.
func (ec *execContext) copyElementNoNamespaces(src *helium.Element) error {
	elem, err := ec.resultDoc.CreateElement(src.LocalName())
	if err != nil {
		return err
	}

	// Only declare namespace for the element's own name
	if src.URI() != "" {
		if err := elem.DeclareNamespace(src.Prefix(), src.URI()); err != nil {
			return err
		}
		if err := elem.SetActiveNamespace(src.Prefix(), src.URI()); err != nil {
			return err
		}
	}

	// Copy attributes, declaring their namespaces as needed
	for _, a := range src.Attributes() {
		if a.URI() != "" {
			if !hasNSDecl(elem, a.Prefix(), a.URI()) {
				if err := elem.DeclareNamespace(a.Prefix(), a.URI()); err != nil {
					return err
				}
			}
			ns, nsErr := ec.resultDoc.CreateNamespace(a.Prefix(), a.URI())
			if nsErr != nil {
				return nsErr
			}
			if err := elem.SetAttributeNS(a.LocalName(), a.Value(), ns); err != nil {
				return err
			}
		} else {
			if err := elem.SetAttribute(a.Name(), a.Value()); err != nil {
				return err
			}
		}
	}

	// Recursively copy children (also without namespaces).
	// Temporarily disable sequenceMode so that child nodes are added to the
	// element's DOM tree instead of being appended as separate sequence items.
	out := ec.currentOutput()
	savedCurrent := out.current
	savedSeqMode := out.sequenceMode
	out.current = elem
	out.sequenceMode = false
	for child := src.FirstChild(); child != nil; child = child.NextSibling() {
		if err := ec.copyNodeToOutput(child, false); err != nil {
			out.current = savedCurrent
			out.sequenceMode = savedSeqMode
			return err
		}
	}
	out.current = savedCurrent
	out.sequenceMode = savedSeqMode

	if err := ec.addNode(elem); err != nil {
		return err
	}
	return nil
}
