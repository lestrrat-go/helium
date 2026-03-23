package xslt3

import (
	"context"
	"errors"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/internal/sequence"
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
				sep, tErr := ec.resultDoc.CreateText([]byte(" "))
				if tErr != nil {
					return tErr
				}
				if err := ec.addNode(sep); err != nil {
					return err
				}
			}
			text, tErr := ec.resultDoc.CreateText([]byte(s))
			if tErr != nil {
				return tErr
			}
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
func (ec *execContext) applyCopyValidation(ctx context.Context, inst *CopyInst, out *outputFrame, lastBefore helium.Node, pendingBefore int, sourceNode helium.Node) error {
	// Apply type validation if specified.
	if inst.TypeName != "" {
		// XTTE1535: if the type attribute refers to a complex type and
		// the item being copied is an attribute node, that is a type error.
		if sourceNode != nil && sourceNode.Type() == helium.AttributeNode {
			if ec.isComplexTypeName(inst.TypeName) {
				return dynamicError(errCodeXTTE1535,
					"xsl:copy type=%q refers to a complex type, but copied item is an attribute node", inst.TypeName)
			}
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
	if v := ec.effectiveValidation(inst.Validation); v != "" && v != validationPreserve {
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

// isComplexType returns true if typeName refers to a complex type in the
// imported schemas. Returns false for built-in types and unknown types.
func (ec *execContext) isComplexType(typeName string) bool {
	if ec.schemaRegistry == nil {
		return false
	}
	return ec.schemaRegistry.IsComplexType(typeName)
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

func (ec *execContext) execCopyOf(ctx context.Context, inst *CopyOfInst) error {
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
				// XTTE1535: type attribute on copy-of with a complex type requires
				// element or document nodes. Simple types may annotate attributes too.
				nt := v.Node.Type()
				if nt != helium.ElementNode && nt != helium.DocumentNode && ec.isComplexType(inst.TypeName) {
					return dynamicError(errCodeXTTE1535,
						"xsl:copy-of type=%q: copied node is %s, not an element or document node",
						inst.TypeName, nt)
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
				ec.transferAnnotationsForCopy(v.Node, out.current, lastBefore)
			} else if effectiveVal == validationStrict || effectiveVal == validationLax || effectiveVal == validationStrip {
				// Apply validation/strip to the most recently added node in output.
				// In sequence mode the copied node lives in pendingItems, not
				// as a child of out.current.
				copiedElem := findCopiedElement(out, lastBefore, pendingBefore)
				if copiedElem != nil {
					if err := ec.validateConstructedElementWithIDCheck(ctx, copiedElem, effectiveVal); err != nil {
						return err
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
	for child := range helium.Children(src) {
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
