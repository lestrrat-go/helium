package xslt3

import (
	"context"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

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

	if isPrimary {
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

	// Store the secondary result document.
	ec.resultDocuments[href] = tmpDoc
	return nil
}
