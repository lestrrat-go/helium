package xslt3

import (
	"context"
	"errors"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// validateDocumentStructure checks that a document node has exactly one element
// child, no text nodes (non-whitespace), and only comments/PIs otherwise.
// Returns XTTE1550 on violation.
func validateDocumentStructure(doc *helium.Document) error {
	elemCount := 0
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Type() {
		case helium.ElementNode:
			elemCount++
		case helium.TextNode, helium.CDATASectionNode:
			// Any text content at the document level (including whitespace
			// that is not whitespace-only) fails validation.
			text := strings.TrimSpace(string(child.Content()))
			if text != "" {
				return dynamicError(errCodeXTTE1550,
					"validated document has text nodes at the top level")
			}
		case helium.CommentNode, helium.ProcessingInstructionNode, helium.DTDNode:
			// Allowed at document level.
		}
	}
	if elemCount != 1 {
		return dynamicError(errCodeXTTE1550,
			"validated document must have exactly one root element, found %d", elemCount)
	}
	return nil
}

// execDocument implements xsl:document: creates a document node wrapping
// the result of executing the body.
func (ec *execContext) execDocument(ctx context.Context, inst *DocumentInst) error {
	tmpDoc := helium.NewDefaultDocument()
	frame := &outputFrame{doc: tmpDoc, current: tmpDoc}
	ec.outputStack = append(ec.outputStack, frame)
	if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
		ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
		return err
	}
	ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]

	// Apply type validation (xsl:document type="...").
	if inst.TypeName != "" {
		// XTTE1550: document must have exactly one element child.
		if err := validateDocumentStructure(tmpDoc); err != nil {
			return err
		}
		if ec.schemaRegistry != nil {
			root := findDocumentElement(tmpDoc)
			if root != nil {
				if err := ec.validateAndNormalizeElementContent(root, inst.TypeName); err != nil {
					var xsltErr *XSLTError
					if errors.As(err, &xsltErr) && xsltErr.Code == errCodeXTTE1510 {
						return dynamicError(errCodeXTTE1540,
							"document content does not match declared type %s: %v", inst.TypeName, xsltErr.Message)
					}
					return err
				}
			}
		}
	}

	// Apply validation if requested (xsl:document validation="strict"|"lax").
	if v := inst.Validation; v == "strict" || v == "lax" {
		if v == "strict" {
			if err := validateDocumentStructure(tmpDoc); err != nil {
				return err
			}
		}
		if ec.schemaRegistry != nil {
			ann, valErr := ec.schemaRegistry.ValidateDoc(ctx, tmpDoc)
			if valErr != nil && v == "strict" {
				return dynamicError(errCodeXTTE1540, "validation of document node failed: %v", valErr)
			}
			if valErr == nil && v == "strict" {
				// XTTE1555: check xs:ID uniqueness and xs:IDREF resolution.
				if err := ValidateDocIDConstraints(tmpDoc, ann); err != nil {
					return err
				}
			}
			for node, typeName := range ann {
				ec.annotateNode(node, typeName)
			}
		}
	}

	// Emit the document node as an item in the parent output frame.
	// sequenceMode means we are in evaluateBodyAsSequence — emit as item.
	// captureItems with a non-document insertion point means simple content
	// construction (e.g., inside xsl:comment) — emit the document as a
	// single item so that atomization yields the correct string value
	// (excluding comment nodes). wherePopulated means we are inside
	// xsl:where-populated — emit the document node so emptiness can be
	// checked. Otherwise copy children directly.
	out := ec.currentOutput()
	if out.sequenceMode || (out.captureItems && out.current != nil && out.current.Type() != helium.DocumentNode) {
		out.pendingItems = append(out.pendingItems, xpath3.NodeItem{Node: tmpDoc})
		out.noteOutput()
	} else if out.wherePopulated {
		if err := ec.addNode(tmpDoc); err != nil {
			return err
		}
	} else {
		for child := tmpDoc.FirstChild(); child != nil; child = child.NextSibling() {
			if err := ec.copyNodeToOutput(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func (ec *execContext) execResultDocument(ctx context.Context, inst *ResultDocumentInst) error {
	// XTDE1480: xsl:result-document is not allowed in a temporary output state.
	if ec.temporaryOutputDepth > 0 {
		return dynamicError(errCodeXTDE1480, "xsl:result-document is not allowed while in temporary output state")
	}

	// Evaluate the href AVT to determine the output URI.
	href := ""
	if inst.Href != nil {
		var err error
		href, err = inst.Href.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
	}

	// Check for duplicate URI (XTDE1490).
	if _, used := ec.usedResultURIs[href]; used {
		return dynamicError(errCodeXTDE1490, "two result documents written to same URI: %q", href)
	}

	isPrimary := href == ""

	if isPrimary && ec.primaryClaimedImplicitly {
		return dynamicError(errCodeXTRE1495, "primary output URI already has implicit content")
	}

	ec.usedResultURIs[href] = struct{}{}

	// Resolve effective item-separator: xsl:result-document attribute takes
	// priority (including #absent which blocks format inheritance),
	// then the named xsl:output (format), then nil (default).
	var itemSep *string
	if inst.ItemSeparatorSet {
		// Attribute was present on xsl:result-document; use its value
		// (nil for #absent, or the specified string).
		itemSep = inst.ItemSeparator
	} else if inst.Format != "" {
		if outDef, ok := ec.stylesheet.outputs[inst.Format]; ok && outDef.ItemSeparator != nil {
			itemSep = outDef.ItemSeparator
		}
	} else if outDef, ok := ec.stylesheet.outputs[""]; ok && outDef.ItemSeparator != nil {
		itemSep = outDef.ItemSeparator
	}

	// Track the current output URI for current-output-uri()
	savedOutputURI := ec.currentOutputURI
	ec.currentOutputURI = href
	defer func() { ec.currentOutputURI = savedOutputURI }()

	if isPrimary {
		v := inst.Validation
		if inst.TypeName != "" && v == "" {
			// type attribute without explicit validation: build into temp doc, validate type, copy.
			tmpDoc := helium.NewDefaultDocument()
			ec.outputStack = append(ec.outputStack, &outputFrame{doc: tmpDoc, current: tmpDoc, itemSeparator: itemSep})
			if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
				ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
				return err
			}
			ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
			root := findDocumentElement(tmpDoc)
			if root != nil && ec.schemaRegistry != nil {
				if err := ec.validateAndNormalizeElementContent(root, inst.TypeName); err != nil {
					var xsltErr *XSLTError
					if errors.As(err, &xsltErr) && xsltErr.Code == errCodeXTTE1510 {
						return dynamicError(errCodeXTTE1540,
							"result document content does not match declared type %s: %v", inst.TypeName, xsltErr.Message)
					}
					return err
				}
			}
			primaryFrame := ec.outputStack[0]
			for child := tmpDoc.FirstChild(); child != nil; child = child.NextSibling() {
				if err := primaryFrame.doc.AddChild(child); err != nil {
					return err
				}
			}
			return nil
		}
		if v == "strict" || v == "lax" {
			// When validation is requested for the primary output, build into a
			// temporary document, validate it, then copy children to the primary
			// output. This is the only way we can inspect the complete document
			// structure before emitting it.
			tmpDoc := helium.NewDefaultDocument()
			ec.outputStack = append(ec.outputStack, &outputFrame{doc: tmpDoc, current: tmpDoc, itemSeparator: itemSep})
			if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
				ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
				return err
			}
			ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
			// XTTE1550: validate document structure.
			if v == "strict" {
				if err := validateDocumentStructure(tmpDoc); err != nil {
					return err
				}
			}
			if ec.schemaRegistry != nil {
				ann, valErr := ec.schemaRegistry.ValidateDoc(ctx, tmpDoc)
				if valErr != nil && v == "strict" {
					return dynamicError(errCodeXTTE1540, "validation of primary result document failed: %v", valErr)
				}
				if valErr == nil && v == "strict" {
					// XTTE1555: check xs:ID uniqueness and xs:IDREF resolution.
					if err := ValidateDocIDConstraints(tmpDoc, ann); err != nil {
						return err
					}
				}
				for node, typeName := range ann {
					ec.annotateNode(node, typeName)
				}
			}
			// Copy validated children into the primary output.
			primaryFrame := ec.outputStack[0]
			for child := tmpDoc.FirstChild(); child != nil; child = child.NextSibling() {
				if err := primaryFrame.doc.AddChild(child); err != nil {
					return err
				}
			}
			return nil
		}
		// Write directly to the primary output (base frame).
		savedStack := ec.outputStack
		ec.outputStack = ec.outputStack[:1] // keep only the base frame
		ec.insideResultDocPrimary = true
		savedSep := ec.outputStack[0].itemSeparator
		ec.outputStack[0].itemSeparator = itemSep
		if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
			ec.insideResultDocPrimary = false
			ec.outputStack[0].itemSeparator = savedSep
			ec.outputStack = savedStack
			return err
		}
		ec.insideResultDocPrimary = false
		ec.outputStack[0].itemSeparator = savedSep
		ec.outputStack = savedStack
		return nil
	}

	// Secondary output: execute body into a temporary document.
	tmpDoc := helium.NewDefaultDocument()

	// Set the document URL so that base-uri() returns the correct value.
	// Resolve relative href against the stylesheet base URI.
	resolvedHref := href
	if ec.stylesheet.baseURI != "" {
		resolved := helium.BuildURI(href, ec.stylesheet.baseURI)
		if resolved != "" {
			resolvedHref = resolved
		}
	}
	tmpDoc.SetURL(resolvedHref)

	ec.outputStack = append(ec.outputStack, &outputFrame{doc: tmpDoc, current: tmpDoc, itemSeparator: itemSep})
	if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
		ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
		return err
	}
	ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]

	// Validate the result document if requested.
	if v := inst.Validation; v == "strict" || v == "lax" {
		// XTTE1550: when validating a document node, the children must comprise
		// exactly one element node, no text nodes, and zero or more comment and
		// processing instruction nodes, in any order.
		if v == "strict" {
			if err := validateDocumentStructure(tmpDoc); err != nil {
				return err
			}
		}
		if ec.schemaRegistry != nil {
			ann, valErr := ec.schemaRegistry.ValidateDoc(ctx, tmpDoc)
			if valErr != nil && v == "strict" {
				return dynamicError(errCodeXTTE1540, "validation of result document failed: %v", valErr)
			}
			if valErr == nil && v == "strict" {
				// XTTE1555: check xs:ID uniqueness and xs:IDREF resolution.
				if err := ValidateDocIDConstraints(tmpDoc, ann); err != nil {
					return err
				}
			}
			for node, typeName := range ann {
				ec.annotateNode(node, typeName)
			}
		}
	} else if inst.Validation == "strip" {
		root := findDocumentElement(tmpDoc)
		if root != nil {
			ec.stripAnnotations(root)
		}
	}

	// Store the secondary result document.
	ec.resultDocuments[href] = tmpDoc
	return nil
}
