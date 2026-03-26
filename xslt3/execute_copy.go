package xslt3

import (
	"context"
	"errors"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
)

func (ec *execContext) execCopy(ctx context.Context, inst *copyInst) error {
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
		result, err := ec.evalXPath(inst.Select, contextNode)
		if err != nil {
			return err
		}
		seq := result.Sequence()
		if seq == nil || sequence.Len(seq) == 0 {
			return nil // empty sequence: skip body
		}
		// XTTE3180: xsl:copy select must produce at most one item.
		if sequence.Len(seq) > 1 {
			return dynamicError(errCodeXTTE3180,
				"xsl:copy select produced %d items; at most one is allowed", sequence.Len(seq))
		}
		for item := range sequence.Items(seq) {
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
				out := ec.currentOutput()
				lastBefore := out.current.LastChild()
				pendingBefore := len(out.pendingItems)
				err := ec.execCopyNode(ctx, v.Node, copyNodeOpts{
					body:              inst.Body,
					copyNamespaces:    copyNS,
					inheritNamespaces: inheritNS,
					useAttrSets:       inst.UseAttrSets,
				})
				ec.contextNode = savedCtx
				ec.currentNode = savedCur
				ec.position = savedPos
				ec.size = savedSize
				if err != nil {
					return err
				}
				// Apply schema validation to the copied node.
				if err := ec.applyCopyValidation(ctx, inst, out, lastBefore, pendingBefore, v.Node); err != nil {
					return err
				}
			case xpath3.AtomicValue:
				// Atomic values: output as text, body is not evaluated.
				// Adjacent atomic values are space-separated per XSLT 3.0 §5.7.2.
				s, err := xpath3.AtomicToString(v)
				if err != nil {
					return err
				}
				out := ec.currentOutput()
				if out.prevWasAtomic {
					sep := ec.resultDoc.CreateText([]byte(" "))
					if err := ec.addNode(sep); err != nil {
						return err
					}
				}
				text := ec.resultDoc.CreateText([]byte(s))
				if err := ec.addNode(text); err != nil {
					return err
				}
				out.prevWasAtomic = true
			}
		}
		return nil
	}

	if contextNode == nil {
		// XSLT 3.0: context item may be an atomic value (e.g. inside
		// xsl:for-each over a sequence of atomics). Copy it as text.
		if av, ok := ec.contextItem.(xpath3.AtomicValue); ok {
			s, err := xpath3.AtomicToString(av)
			if err != nil {
				return err
			}
			out := ec.currentOutput()
			if out.prevWasAtomic {
				sep := ec.resultDoc.CreateText([]byte(" "))
				if err := ec.addNode(sep); err != nil {
					return err
				}
			}
			text := ec.resultDoc.CreateText([]byte(s))
			if err := ec.addNode(text); err != nil {
				return err
			}
			out.prevWasAtomic = true
			return nil
		}
		return dynamicError(errCodeXTTE0945, "xsl:copy: no context item")
	}
	// XTTE1535: if type or validation attribute is specified on xsl:copy and the
	// context item is not an element or document node, signal a type error.
	// For type=, only complex types trigger this error; simple types can annotate attributes.
	if inst.TypeName != "" || inst.Validation == validationStrict || inst.Validation == validationLax {
		nt := contextNode.Type()
		if nt != helium.ElementNode && nt != helium.DocumentNode {
			if inst.Validation == validationStrict || inst.Validation == validationLax {
				return dynamicError(errCodeXTTE1535,
					"xsl:copy validation=%q: context node is %s, not an element or document node",
					inst.Validation, nt)
			}
			if inst.TypeName != "" && ec.isComplexType(inst.TypeName) {
				return dynamicError(errCodeXTTE1535,
					"xsl:copy type=%q: context node is %s, not an element or document node",
					inst.TypeName, nt)
			}
		}
	}

	out := ec.currentOutput()
	lastBefore := out.current.LastChild()
	pendingBefore := len(out.pendingItems)
	if err := ec.execCopyNode(ctx, contextNode, copyNodeOpts{
		body:              inst.Body,
		useAttrSets:       inst.UseAttrSets,
		copyNamespaces:    copyNS,
		inheritNamespaces: inheritNS,
	}); err != nil {
		return err
	}

	return ec.applyCopyValidation(ctx, inst, out, lastBefore, pendingBefore, contextNode)
}

// applyCopyValidation applies type or validation-mode validation to the element
// produced by xsl:copy. It handles elements in both the DOM tree and in
// pendingItems (sequence/capture mode). sourceNode is the original node being copied.
func (ec *execContext) applyCopyValidation(ctx context.Context, inst *copyInst, out *outputFrame, lastBefore helium.Node, pendingBefore int, sourceNode helium.Node) error {
	// Apply type validation if specified.
	if inst.TypeName != "" {
		// XTTE1535: if the type attribute refers to a complex type and
		// the item being copied is an attribute node, that is a type error.
		if sourceNode != nil && sourceNode.Type() == helium.AttributeNode {
			if ec.isComplexTypeName(inst.TypeName) {
				return dynamicError(errCodeXTTE1535,
					"xsl:copy type=%q refers to a complex type, but copied item is an attribute node", inst.TypeName)
			}
			// For simple types on attribute nodes, validate the attribute value
			// against the declared type.
			attr, ok := sourceNode.(*helium.Attribute)
			if ok {
				typeName := normalizeTypeName(inst.TypeName, ec)
				content := string(attr.Content())
				if castErr := ec.validateAttributeValueForType(content, typeName); castErr != nil {
					return dynamicError(errCodeXTTE1510,
						"xsl:copy: attribute value %q does not match type %s: %v", content, inst.TypeName, castErr)
				}
			}
			return nil
		}
		copiedElem := findCopiedElement(out, lastBefore, pendingBefore)
		if copiedElem != nil {
			if err := ec.validateAndNormalizeElementContent(copiedElem, inst.TypeName); err != nil {
				if xsltErr, ok := errors.AsType[*XSLTError](err); ok && xsltErr.Code == errCodeXTTE1510 {
					return dynamicError(errCodeXTTE1540,
						"element content does not match declared type %s: %v", inst.TypeName, xsltErr.Message)
				}
				return err
			}
			ec.annotateNode(copiedElem, inst.TypeName)
			ec.annotateAttributesFromType(copiedElem, inst.TypeName)
			propagateAnnotationToPending(out, pendingBefore, copiedElem, inst.TypeName)
		}
		return nil
	}
	// Apply validation if specified and the copied node is an element.
	// Per XSLT spec, type and validation are mutually exclusive.
	if inst.TypeName == "" {
		if v := ec.effectiveValidation(inst.Validation); v != "" {
			copiedElem := findCopiedElement(out, lastBefore, pendingBefore)
			if copiedElem != nil {
				if err := ec.validateConstructedElement(ctx, copiedElem, v); err != nil {
					return err
				}
				// After validation, propagate type annotations to pending items
				// so that instance-of checks on variable references work correctly.
				if out.sequenceMode && len(out.pendingItems) > pendingBefore {
					ec.propagateValidationAnnotationsToPending(out, pendingBefore)
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
	return ec.defaultValidation
}

// isComplexType returns true if typeName refers to a complex type in the
// imported schemas. Returns false for built-in types and unknown types.
func (ec *execContext) isComplexType(typeName string) bool {
	if ec.schemaRegistry == nil {
		return false
	}
	return ec.schemaRegistry.IsComplexType(typeName)
}

type copyNodeOpts struct {
	body              []instruction
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
		elem := ec.resultDoc.CreateElement(srcElem.LocalName())

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
		// Propagate source element's base URI to orphan copies (spec §11.9.4).
		if elem.Parent() == nil {
			if srcBase := helium.NodeGetBase(srcElem.OwnerDocument(), srcElem); srcBase != "" {
				helium.SetNodeBaseURI(elem, srcBase)
			}
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
		text := ec.resultDoc.CreateText(node.Content())
		return ec.addNode(text)

	case helium.CommentNode:
		comment := ec.resultDoc.CreateComment(node.Content())
		return ec.addNode(comment)

	case helium.ProcessingInstructionNode:
		pi := node.(*helium.ProcessingInstruction)
		newPI := ec.resultDoc.CreatePI(pi.Name(), string(pi.Content()))
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
		for child := range helium.Children(newDoc) {
			if addErr := ec.copyNodeToOutput(child); addErr != nil {
				return addErr
			}
		}
		return nil

	case helium.AttributeNode:
		attr := node.(*helium.Attribute)
		out := ec.currentOutput()
		// In sequence mode (e.g. variable with as="attribute(*)*"),
		// capture the attribute as a standalone item.
		if out.sequenceMode {
			var attrNS *helium.Namespace
			if attr.URI() != "" {
				ns, nsErr := out.doc.CreateNamespace(attr.Prefix(), attr.URI())
				if nsErr == nil {
					attrNS = ns
				}
			}
			copiedAttr, cErr := out.doc.CreateAttribute(attr.Name(), attr.Value(), attrNS)
			if cErr != nil {
				return cErr
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
	}

	return nil
}

func (ec *execContext) execCopyOf(ctx context.Context, inst *copyOfInst) error {
	result, err := ec.evalXPath(inst.Select, ec.contextNode)
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
	preserve := effectiveVal == validationPreserve

	out := ec.currentOutput()
	prevWasAtomic := out.prevWasAtomic
	seq := flattenArraysInSequence(result.Sequence())
	for item := range sequence.Items(seq) {
		switch v := item.(type) {
		case xpath3.NodeItem:
			prevWasAtomic = false
			// XTTE0950: copying namespace-sensitive content with
			// copy-namespaces="no" and validation="preserve", or copying
			// a standalone attribute with namespace-sensitive content and
			// validation="preserve" (without parent element).
			if preserve {
				if !copyNS && ec.nodeHasNamespaceSensitiveContent(v.Node) {
					return dynamicError(errCodeXTTE0950,
						"copy-of: cannot copy namespace-sensitive content with copy-namespaces=\"no\" and validation=\"preserve\"")
				}
				if v.Node.Type() == helium.AttributeNode && ec.nodeHasNamespaceSensitiveContent(v.Node) {
					return dynamicError(errCodeXTTE0950,
						"copy-of: cannot copy attribute with namespace-sensitive content without parent element (validation=\"preserve\")")
				}
			}
			// Remember the last child before copying so we can identify new nodes.
			lastBefore := out.current.LastChild()
			pendingBefore := len(out.pendingItems)
			if err := ec.copyNodeToOutput(v.Node, copyNS); err != nil {
				return err
			}
			// When copy-namespaces="yes" and the source is an element,
			// propagate in-scope namespace declarations from ancestors
			// onto the copied element. This must happen at the top level
			// of xsl:copy-of (not recursively in copyNodeToOutput) so
			// that inner copies and other instructions are not affected.
			if copyNS && v.Node.Type() == helium.ElementNode {
				copiedNSElem := findCopiedElement(out, lastBefore, pendingBefore)
				if copiedNSElem != nil {
					propagateAncestorNamespaces(v.Node.(*helium.Element), copiedNSElem)
				}
			}
			if inst.TypeName != "" {
				// Per XSLT 3.0 spec Section 11.9.4, the type attribute on copy-of
				// is silently ignored for nodes that are not elements, attributes,
				// or document nodes.
				nt := v.Node.Type()
				if nt != helium.ElementNode && nt != helium.AttributeNode && nt != helium.DocumentNode {
					break
				}
				// Attribute type validation: check the attribute value against the declared type.
				if v.Node.Type() == helium.AttributeNode {
					// XTTE1535: complex type on an attribute node is an error.
					if ec.isComplexTypeName(inst.TypeName) {
						return dynamicError(errCodeXTTE1535,
							"copy-of type=%q refers to a complex type, but copied item is an attribute node", inst.TypeName)
					}
					if attr, ok := v.Node.(*helium.Attribute); ok {
						typeName := normalizeTypeName(inst.TypeName, ec)
						if castErr := ec.validateAttributeValueForType(attr.Value(), typeName); castErr != nil {
							return dynamicError(errCodeXTTE1510,
								"copy-of: attribute value %q is not valid for type %s: %v", attr.Value(), inst.TypeName, castErr)
						}
					}
				}
				// Type validation: validate the copied element against the declared type.
				copiedElem := findCopiedElement(out, lastBefore, pendingBefore)
				if copiedElem != nil {
					if err := ec.validateAndNormalizeElementContent(copiedElem, inst.TypeName); err != nil {
						if xsltErr, ok := errors.AsType[*XSLTError](err); ok && xsltErr.Code == errCodeXTTE1510 {
							return dynamicError(errCodeXTTE1540,
								"copy-of: element content does not match declared type %s: %v", inst.TypeName, xsltErr.Message)
						}
						return err
					}
					ec.annotateNode(copiedElem, inst.TypeName)
					ec.annotateAttributesFromType(copiedElem, inst.TypeName)
					// Propagate type annotation to pending NodeItem so that
					// variable references carry the annotation for instance-of checks.
					propagateAnnotationToPending(out, pendingBefore, copiedElem, inst.TypeName)
				}
			} else if preserve {
				if out.sequenceMode && len(out.pendingItems) > pendingBefore {
					// In sequence mode, the copied node lives in pendingItems,
					// not as a child of out.current. Transfer annotations from
					// the source node to the copy, then propagate to pending
					// NodeItems so instance-of checks work.
					for i := pendingBefore; i < len(out.pendingItems); i++ {
						if ni, ok := out.pendingItems[i].(xpath3.NodeItem); ok {
							ec.deepTransferAnnotations(v.Node, ni.Node)
						}
					}
					ec.propagateValidationAnnotationsToPending(out, pendingBefore)
				} else {
					ec.transferAnnotationsForCopy(v.Node, out.current, lastBefore)
				}
			} else if effectiveVal == validationStrict || effectiveVal == validationLax || effectiveVal == validationStrip {
				// Apply validation/strip to the most recently added node in output.
				// In sequence mode the copied node lives in pendingItems, not
				// as a child of out.current.
				copiedElem := findCopiedElement(out, lastBefore, pendingBefore)
				if copiedElem != nil {
					if err := ec.validateConstructedElementWithIDCheck(ctx, copiedElem, effectiveVal); err != nil {
						return err
					}
					// XTTE1555: when copying a complete document node with
					// validation="strict"|"lax", check ID uniqueness and
					// IDREF resolution. Subtree copies (non-document sources)
					// skip this because IDREFs may reference IDs outside the
					// copied fragment.
					if v.Node.Type() == helium.DocumentNode && (effectiveVal == validationStrict || effectiveVal == validationLax) {
						if err := ec.checkIDConstraintsForCopiedDoc(out, pendingBefore, copiedElem); err != nil {
							return err
						}
					}
					// After validation, propagate any type annotations acquired
					// during schema validation to the pending NodeItem so that
					// instance-of checks on variable references work correctly.
					if out.sequenceMode && len(out.pendingItems) > pendingBefore {
						ec.propagateValidationAnnotationsToPending(out, pendingBefore)
					}
				}
			}
			// Transfer accumulator state when copy-accumulators="yes"
			if inst.CopyAccumulators && len(ec.effectiveAccumulators()) > 0 {
				// XTDE3362: check that accumulators are applicable
				// to the source document. This requires explicit
				// use-accumulators on the processing mode.
				if err := ec.checkCopyAccumulators(v.Node); err != nil {
					return err
				}
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
				sep := ec.resultDoc.CreateText([]byte(" "))
				if err := ec.addNode(sep); err != nil {
					return err
				}
			}
			text := ec.resultDoc.CreateText([]byte(s))
			if err := ec.addNode(text); err != nil {
				return err
			}
			prevWasAtomic = true
		}
	}
	out.prevWasAtomic = prevWasAtomic
	return nil
}

// findCopiedElement locates the element that was most recently added to the
// output by a copy-of operation. It checks the DOM tree first, then pending
// items in sequence mode. When the copied node is a document, the document
// element is returned so that validation can be applied.
func findCopiedElement(out *outputFrame, lastBefore helium.Node, pendingBefore int) *helium.Element {
	if copied := out.current.LastChild(); copied != nil {
		if elem, ok := copied.(*helium.Element); ok {
			return elem
		}
	}
	if (out.sequenceMode || out.captureItems) && len(out.pendingItems) > pendingBefore {
		ni, ok := out.pendingItems[len(out.pendingItems)-1].(xpath3.NodeItem)
		if !ok {
			return nil
		}
		if elem, ok := ni.Node.(*helium.Element); ok {
			return elem
		}
		// When the copied node is a document, extract its document element
		// so that validation can be applied to the root element.
		if doc, ok := ni.Node.(*helium.Document); ok {
			return doc.DocumentElement()
		}
	}
	return nil
}

// findPendingDocument returns the most recently added pending document node,
// or nil if the last pending item is not a document.
func findPendingDocument(out *outputFrame, pendingBefore int) *helium.Document {
	if !out.sequenceMode || len(out.pendingItems) <= pendingBefore {
		return nil
	}
	ni, ok := out.pendingItems[len(out.pendingItems)-1].(xpath3.NodeItem)
	if !ok {
		return nil
	}
	doc, _ := ni.Node.(*helium.Document)
	return doc
}

// checkIDConstraintsForCopiedDoc checks xs:ID uniqueness and xs:IDREF
// resolution for a copied document node. It finds the owning document of
// copiedElem (either a pending document in sequence mode or the result
// document) and validates ID constraints using the current type annotations.
func (ec *execContext) checkIDConstraintsForCopiedDoc(out *outputFrame, pendingBefore int, copiedElem *helium.Element) error {
	// In sequence mode, the document lives in pendingItems.
	if pendingDoc := findPendingDocument(out, pendingBefore); pendingDoc != nil {
		return validateDocIDConstraints(pendingDoc, ec.collectAnnotations(pendingDoc))
	}
	// In non-sequence mode, copiedElem was added directly to the output.
	// Build a temporary document containing just this element for ID checking.
	tmpDoc := helium.NewDefaultDocument()
	copied, err := helium.CopyNode(copiedElem, tmpDoc)
	if err != nil {
		return nil // best effort
	}
	if err := tmpDoc.AddChild(copied); err != nil {
		return nil
	}
	// Transfer annotations from the live tree to the copy.
	ann := make(xsd.TypeAnnotations)
	ec.transferAnnotationsToDoc(copiedElem, copied, ann)
	return validateDocIDConstraints(tmpDoc, ann)
}

// transferAnnotationsToDoc recursively copies type annotations from src tree
// to dst tree (which must have the same structure) into the annotation map.
func (ec *execContext) transferAnnotationsToDoc(src, dst helium.Node, ann xsd.TypeAnnotations) {
	if typeName, ok := ec.typeAnnotations[src]; ok {
		ann[dst] = typeName
	}
	// Transfer attribute annotations
	if srcElem, ok := src.(*helium.Element); ok {
		if dstElem, ok := dst.(*helium.Element); ok {
			srcAttrs := srcElem.Attributes()
			dstAttrs := dstElem.Attributes()
			for i, srcAttr := range srcAttrs {
				if typeName, ok := ec.typeAnnotations[srcAttr]; ok && i < len(dstAttrs) {
					ann[dstAttrs[i]] = typeName
				}
			}
		}
	}
	srcChild := src.FirstChild()
	dstChild := dst.FirstChild()
	for srcChild != nil && dstChild != nil {
		ec.transferAnnotationsToDoc(srcChild, dstChild, ann)
		srcChild = srcChild.NextSibling()
		dstChild = dstChild.NextSibling()
	}
}

// collectAnnotations builds a TypeAnnotations map for nodes within the given
// document by filtering ec.typeAnnotations to nodes belonging to that tree.
func (ec *execContext) collectAnnotations(doc *helium.Document) xsd.TypeAnnotations {
	ann := make(xsd.TypeAnnotations)
	for node, typeName := range ec.typeAnnotations {
		if nodeInDoc(node, doc) {
			ann[node] = typeName
		}
	}
	return ann
}

// nodeInDoc returns true when node belongs to the tree rooted at doc.
func nodeInDoc(node helium.Node, doc *helium.Document) bool {
	for n := node; n != nil; n = n.Parent() {
		if n == doc {
			return true
		}
	}
	return false
}

// propagateAnnotationToPending sets the TypeAnnotation on the most recent
// pending NodeItem (or the document element within a pending document node)
// so that variable references carry type annotations for instance-of checks.
func propagateAnnotationToPending(out *outputFrame, pendingBefore int, elem *helium.Element, typeName string) {
	if !out.sequenceMode || len(out.pendingItems) <= pendingBefore {
		return
	}
	idx := len(out.pendingItems) - 1
	ni, ok := out.pendingItems[idx].(xpath3.NodeItem)
	if !ok {
		return
	}
	if ni.Node == elem {
		ni.TypeAnnotation = typeName
		out.pendingItems[idx] = ni
	}
}

// propagateValidationAnnotationsToPending updates the TypeAnnotation on pending
// NodeItems based on the execContext's typeAnnotations map. This ensures that
// variable references carry schema type annotations acquired during validation.
func (ec *execContext) propagateValidationAnnotationsToPending(out *outputFrame, pendingBefore int) {
	if ec.typeAnnotations == nil {
		return
	}
	for i := pendingBefore; i < len(out.pendingItems); i++ {
		ni, ok := out.pendingItems[i].(xpath3.NodeItem)
		if !ok {
			continue
		}
		// For element nodes, check the typeAnnotations map directly.
		if ann, found := ec.typeAnnotations[ni.Node]; found && ni.TypeAnnotation == "" {
			ni.TypeAnnotation = ann
			out.pendingItems[i] = ni
			continue
		}
		// For document nodes, check the document element's annotation.
		if doc, ok := ni.Node.(*helium.Document); ok {
			if docElem := doc.DocumentElement(); docElem != nil {
				if ann, found := ec.typeAnnotations[docElem]; found {
					// The document itself doesn't get a type annotation,
					// but we record it so that path navigation from the
					// document node can find it via nodeItemFor.
					_ = ann
				}
			}
		}
	}
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
			for child := range helium.Children(node) {
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
		// Also preserve DTD information (unparsed entities, notations)
		// in the target document so that unparsed-entity-uri() etc.
		// continue to work on the result tree.
		if srcDoc, ok := node.(*helium.Document); ok && srcDoc.IntSubset() != nil {
			if targetDoc := out.doc; targetDoc != nil && targetDoc.IntSubset() == nil {
				helium.CopyDTDInfo(srcDoc, targetDoc)
			}
		}
		for child := range helium.Children(node) {
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

// propagateAncestorNamespaces copies in-scope namespace declarations from
// ancestors of src onto dst. This ensures that copy-namespaces="yes" includes
// all inherited namespace bindings, not just those directly declared on the
// element. The xml namespace is excluded since it is always in scope.
func propagateAncestorNamespaces(src, dst *helium.Element) {
	// Collect prefixes already declared on the destination.
	declared := make(map[string]struct{})
	for _, ns := range dst.Namespaces() {
		declared[ns.Prefix()] = struct{}{}
	}

	// Walk up ancestors and add any undeclared namespace bindings.
	for cur := src.Parent(); cur != nil; {
		pElem, ok := cur.(*helium.Element)
		if !ok {
			break
		}
		for _, ns := range pElem.Namespaces() {
			prefix := ns.Prefix()
			if prefix == "xml" {
				continue
			}
			if _, exists := declared[prefix]; exists {
				continue
			}
			_ = dst.DeclareNamespace(prefix, ns.URI())
			declared[prefix] = struct{}{}
		}
		cur = pElem.Parent()
	}
}

// copyElementNoNamespaces deep-copies an element without copying namespace
// declarations (copy-namespaces="no").  Only the namespaces required for
// well-formedness (element name and attribute names) are preserved.
func (ec *execContext) copyElementNoNamespaces(src *helium.Element) error {
	elem := ec.resultDoc.CreateElement(src.LocalName())

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
	for child := range helium.Children(src) {
		if err := ec.copyNodeToOutput(child, false); err != nil {
			out.current = savedCurrent
			out.sequenceMode = savedSeqMode
			return err
		}
	}
	out.current = savedCurrent
	out.sequenceMode = savedSeqMode

	// copy-namespaces="no": children within the copy subtree must not
	// inherit namespace declarations that were added to this element for
	// its own well-formedness (element name / attribute names). Undeclare
	// such bindings on each direct child element that does not itself
	// require them.
	undeclareParentCopyNS(elem)

	if err := ec.addNode(elem); err != nil {
		return err
	}
	// After inserting the element into the result tree, fix namespace
	// declarations relative to the new parent (e.g. add xmlns=""
	// undeclarations to prevent inheriting the parent's default namespace
	// when the element is not in that namespace).
	ec.fixNamespacesAfterCopy(elem)
	return nil
}

// undeclareParentCopyNS adds namespace undeclarations on each direct child
// element of parent for every namespace declared on parent that the child
// does not itself require. This prevents copy-internal namespace leakage:
// when copy-namespaces="no", a parent element may need a namespace for its
// name/attributes, but children should not inherit those bindings.
func undeclareParentCopyNS(parent *helium.Element) {
	// Collect namespaces declared on the parent.
	parentNS := parent.Namespaces()
	if len(parentNS) == 0 {
		return
	}

	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		// Collect prefixes that the child needs.
		childNeeded := make(map[string]struct{})
		if childElem.URI() != "" {
			childNeeded[childElem.Prefix()] = struct{}{}
		}
		for _, a := range childElem.Attributes() {
			if a.URI() != "" {
				childNeeded[a.Prefix()] = struct{}{}
			}
		}
		// Collect prefixes already declared on the child.
		childDeclared := make(map[string]struct{})
		for _, ns := range childElem.Namespaces() {
			childDeclared[ns.Prefix()] = struct{}{}
		}

		for _, ns := range parentNS {
			prefix := ns.Prefix()
			if prefix == "xml" {
				continue
			}
			// Skip if the child already declares this prefix.
			if _, ok := childDeclared[prefix]; ok {
				continue
			}
			// Skip if the child needs this prefix.
			if _, ok := childNeeded[prefix]; ok {
				continue
			}
			// Undeclare: the child doesn't need this binding.
			_ = childElem.DeclareNamespace(prefix, "")
			childDeclared[prefix] = struct{}{}
		}
	}
}

// checkCopyAccumulators verifies that all accumulators declared in the
// stylesheet are applicable to the tree containing node. This implements
// the XTDE3362 check for xsl:copy-of/@copy-accumulators="yes".
// An accumulator is applicable to a source document only if the initial
// mode explicitly lists it via use-accumulators.
func (ec *execContext) checkCopyAccumulators(node helium.Node) error {
	accumulators := ec.effectiveAccumulators()
	if len(accumulators) == 0 {
		return nil
	}

	// When there is an activeAccumulators set (e.g. from xsl:source-document),
	// the document-level applicability already restricts access.
	if ec.activeAccumulators != nil {
		for name := range accumulators {
			if _, ok := ec.activeAccumulators[name]; !ok {
				return dynamicError(errCodeXTDE3362,
					"accumulator %q is not applicable to the source document (copy-accumulators)", name)
			}
		}
		return nil
	}

	// Only check mode-level use-accumulators for nodes from the initial
	// source document. variable/RTF trees have accumulators computed
	// lazily and are always accessible.
	root := documentRoot(node)
	if root != ec.sourceDoc {
		return nil
	}

	// Determine the mode definition for the initial mode.
	modeDefs := ec.effectiveModeDefs()
	md := modeDefs[ec.currentMode]
	if md == nil {
		md = modeDefs[modeDefault]
	}

	// When no explicit xsl:mode declaration exists, the mode has no
	// use-accumulators attribute. For the initial source document,
	// accumulators are only applicable if the mode declares them.
	if md == nil {
		return dynamicError(errCodeXTDE3362,
			"copy-accumulators requires use-accumulators on the processing mode")
	}
	// When use-accumulators attribute is absent (nil), the default is
	// "#all" per XSLT 3.0 spec — all accumulators are applicable.
	if md.UseAccumulators == nil {
		return nil
	}
	ua := *md.UseAccumulators
	if ua == "#all" {
		return nil
	}
	allowed := make(map[string]struct{})
	for _, n := range strings.Fields(ua) {
		allowed[n] = struct{}{}
	}
	for name := range accumulators {
		if _, ok := allowed[name]; !ok {
			return dynamicError(errCodeXTDE3362,
				"accumulator %q is not applicable to the source document (copy-accumulators)", name)
		}
	}
	return nil
}

// nodeHasNamespaceSensitiveContent checks if a node (or any of its
// descendants/attributes) has a type annotation that is namespace-sensitive
// (xs:QName, xs:NOTATION, or a type derived from them). This is used for
// XTTE0950 checking.
func (ec *execContext) nodeHasNamespaceSensitiveContent(node helium.Node) bool {
	if ec.typeAnnotations == nil {
		return false
	}
	return ec.checkNodeNSSensitive(node)
}

func (ec *execContext) checkNodeNSSensitive(node helium.Node) bool {
	ann := ec.typeAnnotations[node]
	if isNamespaceSensitiveType(ann) {
		return true
	}
	if ec.schemaRegistry != nil && ann != "" {
		if ec.schemaRegistry.IsSubtypeOf(ann, "xs:QName") || ec.schemaRegistry.IsSubtypeOf(ann, "xs:NOTATION") {
			return true
		}
	}
	// Check attributes
	if elem, ok := node.(*helium.Element); ok {
		for _, attr := range elem.Attributes() {
			attrAnn := ec.typeAnnotations[attr]
			if isNamespaceSensitiveType(attrAnn) {
				return true
			}
			if ec.schemaRegistry != nil && attrAnn != "" {
				if ec.schemaRegistry.IsSubtypeOf(attrAnn, "xs:QName") || ec.schemaRegistry.IsSubtypeOf(attrAnn, "xs:NOTATION") {
					return true
				}
			}
		}
		// Check child elements recursively
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			if ec.checkNodeNSSensitive(child) {
				return true
			}
		}
	}
	return false
}

// isNamespaceSensitiveType returns true if the given type name is xs:QName,
// xs:NOTATION, or common derived forms.
func isNamespaceSensitiveType(typeName string) bool {
	switch typeName {
	case "xs:QName", "xs:NOTATION",
		"Q{http://www.w3.org/2001/XMLSchema}QName",
		"Q{http://www.w3.org/2001/XMLSchema}NOTATION":
		return true
	}
	return false
}


// isComplexTypeName returns true if the given type name refers to a complex
// type definition in the imported schemas.
func (ec *execContext) isComplexTypeName(typeName string) bool {
	if ec.schemaRegistry == nil {
		return false
	}
	td, _, found := ec.schemaRegistry.LookupTypeDef(typeName)
	if !found {
		return false
	}
	// A complex type has a content model, attributes, or non-simple content type.
	return td.ContentModel != nil || len(td.Attributes) > 0 || td.AnyAttribute != nil
}

// validateAttributeValueForType validates an attribute string value against
// a type name. For built-in XSD types it uses xpath3.CastFromString; for
// user-defined types (list, union, restriction) it falls back to the schema
// registry's ValidateCast. Returns nil when the value is valid.
func (ec *execContext) validateAttributeValueForType(value, typeName string) error {
	_, castErr := xpath3.CastFromString(value, typeName)
	if castErr == nil {
		return nil
	}
	// CastFromString only knows built-in types. For user-defined simple types
	// (lists, unions, restrictions), delegate to the schema registry.
	// Only fall back when CastFromString signals an unknown type (XPTY0004);
	// if CastFromString recognised the type but the value is invalid
	// (FORG0001 etc.), trust that verdict.
	if ec.schemaRegistry == nil {
		return castErr
	}
	var xe *xpath3.XPathError
	if errors.As(castErr, &xe) && xe.Code != "XPTY0004" {
		return castErr
	}
	return ec.schemaRegistry.ValidateCast(value, typeName)
}
