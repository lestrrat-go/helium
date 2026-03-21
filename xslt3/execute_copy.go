package xslt3

import (
	"context"
	"errors"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
)

func (ec *execContext) execCopy(ctx context.Context, inst *CopyInst) error {
	contextNode := normalizeNode(ec.contextNode)

	// Resolve shadow attributes (AVTs) at runtime; fall back to static values.
	copyNS := inst.CopyNamespaces
	inheritNS := inst.InheritNamespaces
	if inst.CopyNamespacesAVT != nil {
		v, err := inst.CopyNamespacesAVT.evaluate(ctx, contextNode)
		if err != nil {
			return err
		}
		if b, ok := parseXSDBool(v); ok {
			copyNS = b
		}
	}
	if inst.InheritNamespacesAVT != nil {
		v, err := inst.InheritNamespacesAVT.evaluate(ctx, contextNode)
		if err != nil {
			return err
		}
		if b, ok := parseXSDBool(v); ok {
			inheritNS = b
		}
	}

	if inst.Select != nil {
		// XSLT 3.0: xsl:copy with select — iterate over selected items
		xpathCtx := ec.newXPathContext(contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, contextNode)
		if err != nil {
			return err
		}
		seq := result.Sequence()
		if len(seq) == 0 {
			return nil // empty sequence: skip body
		}
		// XTTE3180: xsl:copy select must produce at most one item.
		if len(seq) > 1 {
			return dynamicError(errCodeXTTE3180,
				"xsl:copy select produced %d items; at most one is allowed", len(seq))
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

	if contextNode == nil {
		return dynamicError(errCodeXTTE0945, "xsl:copy: no context item")
	}
	if err := ec.execCopyNode(ctx, contextNode, copyNodeOpts{
		body:              inst.Body,
		useAttrSets:       inst.UseAttrSets,
		copyNamespaces:    copyNS,
		inheritNamespaces: inheritNS,
	}); err != nil {
		return err
	}

	// Apply type validation if specified.
	if inst.TypeName != "" {
		// For xsl:copy, the type attribute is only applied to element/attribute
		// context nodes. For text/comment/PI, it is silently ignored.
		isElemOrAttr := contextNode != nil &&
			(contextNode.Type() == helium.ElementNode || contextNode.Type() == helium.AttributeNode)
		isDocument := contextNode != nil && contextNode.Type() == helium.DocumentNode
		// XTTE1535: complex type on non-element, non-document node.
		if !isElemOrAttr && !isDocument && ec.schemaRegistry != nil {
			td, _, found := ec.schemaRegistry.LookupTypeDef(inst.TypeName)
			if found && isComplexTypeDef(td) {
				return dynamicError(errCodeXTTE1535,
					"copy: complex type %s cannot be applied to %s node", inst.TypeName, contextNode.Type())
			}
		}
		// For document nodes, apply the type to the root element.
		if isDocument {
			out := ec.currentOutput()
			// Find the root element in the copied output.
			for child := out.current.FirstChild(); child != nil; child = child.NextSibling() {
				if copiedElem, ok := child.(*helium.Element); ok {
					if err := ec.validateAndNormalizeElementContent(copiedElem, inst.TypeName); err != nil {
						return err
					}
					ec.annotateNode(copiedElem, inst.TypeName)
					ec.annotateAttributesFromType(copiedElem, inst.TypeName)
					break
				}
			}
		}
		if isElemOrAttr {
			out := ec.currentOutput()
			if copied := out.current.LastChild(); copied != nil {
				if copiedElem, ok := copied.(*helium.Element); ok {
					if err := ec.validateAndNormalizeElementContent(copiedElem, inst.TypeName); err != nil {
						if xsltErr, ok := errors.AsType[*XSLTError](err); ok && xsltErr.Code == errCodeXTTE1510 {
							return dynamicError(errCodeXTTE1540,
								"element content does not match declared type %s: %v", inst.TypeName, xsltErr.Message)
						}
						return err
					}
					ec.annotateNode(copiedElem, inst.TypeName)
					ec.annotateAttributesFromType(copiedElem, inst.TypeName)
				}
			}
		}
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

// isComplexTypeDef returns true if the TypeDef represents a complex type (has
// attributes, element content, mixed content, or simpleContent with attributes).
func isComplexTypeDef(td *xsd.TypeDef) bool {
	if len(td.Attributes) > 0 || td.AnyAttribute != nil {
		return true
	}
	switch td.ContentType {
	case xsd.ContentTypeElementOnly, xsd.ContentTypeMixed, xsd.ContentTypeEmpty:
		return true
	}
	return td.ContentModel != nil
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
			// copy-namespaces="yes": copy all in-scope namespace declarations
			// (including those inherited from ancestors).  This matches the
			// XSLT 3.0 spec default behaviour.
			for _, ns := range collectInScopeNamespaces(srcElem) {
				if ns.URI() == "" {
					continue // skip undeclarations
				}
				if err := elem.DeclareNamespace(ns.Prefix(), ns.URI()); err != nil {
					return err
				}
			}
		}
		// When copy-namespaces="no": do not copy source namespace
		// declarations. Only the element's own namespace (required for
		// well-formedness) is declared below.
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

		// Execute body in element context.
		// Temporarily disable sequenceMode so that children are added to
		// this element normally (not captured as separate sequence items).
		out := ec.currentOutput()
		savedCurrent := out.current
		savedPrevAtomic := out.prevWasAtomic
		savedSeqMode := out.sequenceMode
		out.current = elem
		out.prevWasAtomic = false
		out.sequenceMode = false
		defer func() {
			out.current = savedCurrent
			out.prevWasAtomic = savedPrevAtomic
			out.sequenceMode = savedSeqMode
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
			savedSeqMode := out.sequenceMode
			ec.resultDoc = newDoc
			// Temporarily disable sequenceMode so that children are added
			// to the new document node (not captured as separate items).
			out.sequenceMode = false
			docRoot := newDoc.DocumentElement()
			if docRoot == nil {
				// No doc element yet; use the document node itself as output target.
				out.current = newDoc
			}
			if err := ec.executeSequenceConstructor(ctx, opts.body); err != nil {
				ec.resultDoc = savedDoc
				out.current = savedOutput
				out.sequenceMode = savedSeqMode
				return err
			}
			ec.resultDoc = savedDoc
			out.current = savedOutput
			out.sequenceMode = savedSeqMode
			out.pendingItems = append(out.pendingItems, xpath3.NodeItem{Node: newDoc})
			out.noteOutput()
			return nil
		}

		// Not in capture mode — process body in a document node context.
		// XTDE0420: attributes/namespaces in the body are errors.
		savedCurrent := out.current
		out.current = newDoc
		savedDoc := ec.resultDoc
		ec.resultDoc = newDoc
		err := ec.executeSequenceConstructor(ctx, opts.body)
		ec.resultDoc = savedDoc
		out.current = savedCurrent
		if err != nil {
			return err
		}
		// Copy children from the temporary document to the current output.
		for child := newDoc.FirstChild(); child != nil; child = child.NextSibling() {
			if addErr := ec.copyNodeToOutput(child); addErr != nil {
				return addErr
			}
		}
		return nil

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
		copyAttributeToElement(elem, attr)
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
			// Remember the last child before copying so we can identify new nodes.
			lastBefore := out.current.LastChild()
			pendingBefore := len(out.pendingItems)
			if err := ec.copyNodeToOutput(v.Node, copyNS); err != nil {
				return err
			}
			if inst.TypeName != "" {
				// Per XSLT 3.0 spec, the type attribute on copy-of is silently
				// ignored for nodes that are not elements, attributes, or documents.
				if v.Node.Type() != helium.ElementNode && v.Node.Type() != helium.AttributeNode && v.Node.Type() != helium.DocumentNode {
					break
				}
				// Type validation: validate the copied element against the declared type.
				if copied := out.current.LastChild(); copied != nil {
					if copiedElem, ok := copied.(*helium.Element); ok {
						if err := ec.validateAndNormalizeElementContent(copiedElem, inst.TypeName); err != nil {
							if xsltErr, ok := errors.AsType[*XSLTError](err); ok && xsltErr.Code == errCodeXTTE1510 {
								return dynamicError(errCodeXTTE1540,
									"copy-of: element content does not match declared type %s: %v", inst.TypeName, xsltErr.Message)
							}
							return err
						}
						ec.annotateNode(copiedElem, inst.TypeName)
						ec.annotateAttributesFromType(copiedElem, inst.TypeName)
					}
				}
			} else if preserve {
				ec.transferAnnotationsForCopy(v.Node, out.current, lastBefore)
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
			// Transfer accumulator state when copy-accumulators="yes"
			if inst.CopyAccumulators && len(ec.stylesheet.accumulators) > 0 {
				_ = ec.ensureAccumulatorStates(ctx, v.Node)
				// Identify the copied node: check DOM tree first, then pendingItems (sequence mode)
				var copiedNode helium.Node
				if lastBefore == nil {
					copiedNode = out.current.FirstChild()
				} else {
					copiedNode = lastBefore.NextSibling()
				}
				// In sequence mode, the copied node is captured in pendingItems
				// rather than attached to the DOM tree.
				if copiedNode == nil && out.sequenceMode && len(out.pendingItems) > pendingBefore {
					if ni, ok := out.pendingItems[len(out.pendingItems)-1].(xpath3.NodeItem); ok {
						copiedNode = ni.Node
					}
				}
				if copiedNode != nil {
					ec.transferAccumulatorStates(v.Node, copiedNode)
					copyRoot := documentRoot(copiedNode)
					if ec.accumulatorComputedDocs == nil {
						ec.accumulatorComputedDocs = make(map[helium.Node]struct{})
					}
					ec.accumulatorComputedDocs[copyRoot] = struct{}{}
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
		out := ec.currentOutput()
		if out.sequenceMode {
			// In sequence mode, wrap the document copy as a single
			// document-node item so that variables with as="node()"
			// receive exactly 1 item (a document node).
			srcDoc, _ := node.(*helium.Document)
			newDoc := helium.NewDefaultDocument()
			if srcDoc != nil {
				helium.CopyDTDInfo(srcDoc, newDoc)
				newDoc.SetURL(srcDoc.URL())
			}
			for child := node.FirstChild(); child != nil; child = child.NextSibling() {
				copied, err := helium.CopyNode(child, newDoc)
				if err != nil {
					return err
				}
				if err := newDoc.AddChild(copied); err != nil {
					return err
				}
			}
			out.pendingItems = append(out.pendingItems, xpath3.NodeItem{Node: newDoc})
			out.noteOutput()
			return nil
		}
		// Per XSLT spec, xsl:copy-of on document node copies children.
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
		// In sequence mode, capture the attribute as a standalone item.
		if out.sequenceMode {
			var attrNS *helium.Namespace
			if attr.URI() != "" {
				ns, nsErr := out.doc.CreateNamespace(attr.Prefix(), attr.URI())
				if nsErr == nil {
					attrNS = ns
				}
			}
			copiedAttr, err := out.doc.CreateAttribute(attr.Name(), attr.Value(), attrNS)
			if err != nil {
				return err
			}
			out.pendingItems = append(out.pendingItems, xpath3.NodeItem{Node: copiedAttr})
			out.noteOutput()
			return nil
		}
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
		copyAttributeToElement(elem, attr)
		out.noteOutput()
		return nil
	case helium.NamespaceNode:
		// Namespace nodes are copied as namespace declarations on the current element.
		nsw, ok := node.(*helium.NamespaceNodeWrapper)
		if !ok {
			return nil
		}
		out := ec.currentOutput()
		// In sequence mode, capture the namespace node as a standalone item.
		if out.sequenceMode {
			ns, err := out.doc.CreateNamespace(nsw.Name(), string(nsw.Content()))
			if err != nil {
				return err
			}
			copiedNSW := helium.NewNamespaceNodeWrapper(ns, nil)
			out.pendingItems = append(out.pendingItems, xpath3.NodeItem{Node: copiedNSW})
			out.noteOutput()
			return nil
		}
		elem, ok := out.current.(*helium.Element)
		if !ok {
			return nil
		}
		prefix := nsw.Name()
		uri := string(nsw.Content())
		// Check for conflicts (same prefix, different URI) and apply
		// namespace fixup when allowed (e.g. xsl:element auto-generated ns).
		_, fixupOK := ec.nsFixupAllowed[elem]
		for _, ns := range elem.Namespaces() {
			if ns.Prefix() == prefix && ns.URI() != uri {
				if fixupOK && elem.Prefix() == prefix && elem.URI() == ns.URI() {
					// Rename the element's prefix to avoid conflict
					origURI := elem.URI()
					newPrefix := uniqueNSPrefix(elem, prefix+"_0", origURI)
					elem.RemoveNamespaceByPrefix(prefix)
					_ = elem.DeclareNamespace(newPrefix, origURI)
					_ = elem.SetActiveNamespace(newPrefix, origURI)
					break
				}
				return dynamicError(errCodeXTDE0430,
					"namespace prefix %q is already bound to %q; cannot rebind to %q", prefix, ns.URI(), uri)
			}
		}
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

// copyElementNoNamespaces deep-copies an element without copying namespace
// declarations (copy-namespaces="no").  Only the namespaces required for
// well-formedness (element name and attribute names) are preserved.
func (ec *execContext) copyElementNoNamespaces(src *helium.Element) error {
	elem, err := ec.resultDoc.CreateElement(src.LocalName())
	if err != nil {
		return err
	}

	// Declare only the element's own namespace (required for well-formedness).
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
			elem.SetLiteralAttributeNS(a.LocalName(), a.Value(), ns)
		} else {
			elem.SetLiteralAttribute(a.Name(), a.Value())
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
