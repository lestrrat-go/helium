package xslt3

import (
	"context"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// checkAtomizable returns FOTY0013 if the sequence contains any map or
// function items, which cannot be atomized.
func checkAtomizable(seq xpath3.Sequence) error {
	for _, item := range seq {
		switch item.(type) {
		case xpath3.MapItem:
			return dynamicError(errCodeFOTY0013, "cannot atomize a map item")
		case xpath3.FunctionItem:
			return dynamicError(errCodeFOTY0013, "cannot atomize a function item")
		}
	}
	return nil
}

func (ec *execContext) execValueOf(ctx context.Context, inst *ValueOfInst) error {
	var value string

	emptySequence := false
	if inst.Select != nil {
		// Default separator for select is " "
		separator := " "
		if inst.Separator != nil {
			var err error
			separator, err = inst.Separator.evaluate(ctx, ec.contextNode)
			if err != nil {
				return err
			}
		}
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		seq := result.Sequence()
		if len(seq) == 0 {
			emptySequence = true
		}
		// FOTY0013: maps and functions cannot be atomized
		if err := checkAtomizable(seq); err != nil {
			return err
		}
		// XSLT spec §11.3: zero-length text nodes in the result sequence
		// are discarded before stringification.
		filtered := filterZeroLengthTextNodes(seq)
		value = stringifySequenceWithSep(filtered, separator)
	} else if len(inst.Body) > 0 {
		// Default separator for body content is "" (zero-length string)
		separator := ""
		if inst.Separator != nil {
			var err error
			separator, err = inst.Separator.evaluate(ctx, ec.contextNode)
			if err != nil {
				return err
			}
		}
		// XSLT 2.0: value-of body is temporary output state (XTDE1480).
		// XSLT 3.0 relaxes this restriction.
		isV2TempOutput := ec.stylesheet.version != "" && ec.stylesheet.version < "3.0"
		if isV2TempOutput {
			ec.temporaryOutputDepth++
		}
		val, err := ec.evaluateBodySeparateText(ctx, inst.Body)
		if isV2TempOutput {
			ec.temporaryOutputDepth--
		}
		if err != nil {
			return err
		}
		// XSLT spec §11.3: adjacent text nodes in the result are merged
		// before the separator is applied.
		val = mergeAdjacentTextNodes(val)
		value = stringifySequenceWithSep(val, separator)
	}
	// Skip text node only when select evaluates to an empty sequence;
	// an empty string from select="''" should still produce a zero-length
	// text node (important for as="xs:string" variables).
	if emptySequence {
		return nil
	}
	// XSLT 3.0 §20.1: disable-output-escaping is ignored when writing
	// to a temporary tree (variable/parameter body) or sequence mode.
	doeEffective := inst.DisableOutputEscaping && ec.temporaryOutputDepth == 0
	if doeEffective {
		// Insert DOE marker PI so the serializer writes this text raw.
		pi, err := ec.resultDoc.CreatePI("disable-output-escaping", "")
		if err != nil {
			return err
		}
		if err := ec.addNode(pi); err != nil {
			return err
		}
		text, err := ec.resultDoc.CreateText([]byte(value))
		if err != nil {
			return err
		}
		if err := ec.addNode(text); err != nil {
			return err
		}
		piEnd, err := ec.resultDoc.CreatePI("enable-output-escaping", "")
		if err != nil {
			return err
		}
		return ec.addNode(piEnd)
	}
	text, err := ec.resultDoc.CreateText([]byte(value))
	if err != nil {
		return err
	}
	return ec.addNode(text)
}

// filterZeroLengthTextNodes removes zero-length text nodes from a sequence.
// Per XSLT spec §11.3, these are discarded during xsl:value-of processing.
func filterZeroLengthTextNodes(seq xpath3.Sequence) xpath3.Sequence {
	result := make(xpath3.Sequence, 0, len(seq))
	for _, item := range seq {
		if ni, ok := item.(xpath3.NodeItem); ok {
			if ni.Node.Type() == helium.TextNode && len(ni.Node.Content()) == 0 {
				continue
			}
		}
		result = append(result, item)
	}
	return result
}

// removeZeroLengthTextNodes removes text nodes with zero-length string
// value from the sequence. Per XSLT 3.0 §5.7.2, zero-length text nodes
// in the result sequence are removed before separator insertion.
func removeZeroLengthTextNodes(seq xpath3.Sequence) xpath3.Sequence {
	result := seq[:0:0]
	changed := false
	for _, item := range seq {
		ni, ok := item.(xpath3.NodeItem)
		if ok && ni.Node.Type() == helium.TextNode && len(ni.Node.Content()) == 0 {
			changed = true
			continue
		}
		result = append(result, item)
	}
	if !changed {
		return seq
	}
	return result
}

// mergeAdjacentTextNodes merges consecutive text node items in a sequence
// into single text nodes. Per XSLT spec §11.3, adjacent text nodes are
// merged before separator insertion in xsl:value-of.
func mergeAdjacentTextNodes(seq xpath3.Sequence) xpath3.Sequence {
	if len(seq) <= 1 {
		return seq
	}
	result := make(xpath3.Sequence, 0, len(seq))
	for i := 0; i < len(seq); i++ {
		ni, ok := seq[i].(xpath3.NodeItem)
		if !ok || ni.Node.Type() != helium.TextNode {
			result = append(result, seq[i])
			continue
		}
		// Merge consecutive text nodes
		merged := string(ni.Node.Content())
		j := i + 1
		for j < len(seq) {
			nj, ok := seq[j].(xpath3.NodeItem)
			if !ok || nj.Node.Type() != helium.TextNode {
				break
			}
			merged += string(nj.Node.Content())
			j++
		}
		if j > i+1 {
			// Create a merged text node
			doc := ni.Node.OwnerDocument()
			text, err := doc.CreateText([]byte(merged))
			if err == nil {
				result = append(result, xpath3.NodeItem{Node: text})
			} else {
				result = append(result, seq[i])
			}
			i = j - 1
		} else {
			result = append(result, seq[i])
		}
	}
	return result
}

func (ec *execContext) execText(inst *TextInst) error {
	value := inst.Value
	if inst.TVT != nil {
		ctx := ec.transformCtx
		if ctx == nil {
			ctx = context.Background()
		}
		ctx = withExecContext(ctx, ec)
		var err error
		value, err = inst.TVT.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		// For TVTs that evaluate to empty, skip the text node but
		// break atomic adjacency chains, consistent with static
		// empty xsl:text behaviour.
		if value == "" {
			out := ec.currentOutput()
			if !out.wherePopulated {
				out.prevWasAtomic = false
			}
			return nil
		}
	}
	// xsl:text with empty literal content produces a zero-length text node
	// only in sequenceMode (needed for as="text()" variables to receive
	// exactly 1 item). Outside sequenceMode, empty literal text is not
	// added to the result tree but it DOES break atomic adjacency chains:
	// <xsl:text/> between two xsl:sequence instructions prevents the
	// inter-atomic space separator from being inserted (XSLT 3.0 §5.7.2).
	if value == "" && inst.TVT == nil {
		out := ec.currentOutput()
		if out.sequenceMode {
			text, err := ec.resultDoc.CreateText(nil)
			if err != nil {
				return err
			}
			return ec.addNode(text)
		}
		// Inside xsl:where-populated, zero-length text nodes are removed
		// before the constructing complex content rules apply (XSLT 3.0
		// §8.4), so they must not break atomic adjacency chains.
		if !out.wherePopulated {
			out.prevWasAtomic = false
		}
		return nil
	}
	// XSLT 3.0 §20.1: disable-output-escaping is ignored when writing
	// to a temporary tree (variable/parameter body) or sequence mode.
	textDOEEffective := inst.DisableOutputEscaping && ec.temporaryOutputDepth == 0
	if textDOEEffective {
		pi, err := ec.resultDoc.CreatePI("disable-output-escaping", "")
		if err != nil {
			return err
		}
		if err := ec.addNode(pi); err != nil {
			return err
		}
		text, err := ec.resultDoc.CreateText([]byte(value))
		if err != nil {
			return err
		}
		if err := ec.addNode(text); err != nil {
			return err
		}
		piEnd, err := ec.resultDoc.CreatePI("enable-output-escaping", "")
		if err != nil {
			return err
		}
		return ec.addNode(piEnd)
	}
	text, err := ec.resultDoc.CreateText([]byte(value))
	if err != nil {
		return err
	}
	return ec.addNode(text)
}

func (ec *execContext) execLiteralText(inst *LiteralTextInst) error {
	value := inst.Value
	if inst.TVT != nil {
		// Text value template: evaluate like an AVT
		ctx := ec.transformCtx
		if ctx == nil {
			ctx = context.Background()
		}
		ctx = withExecContext(ctx, ec)
		var err error
		value, err = inst.TVT.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
	}
	if value == "" {
		return nil
	}
	text, err := ec.resultDoc.CreateText([]byte(value))
	if err != nil {
		return err
	}
	return ec.addNode(text)
}

func (ec *execContext) execXSLSequence(ctx context.Context, inst *XSLSequenceInst) error {
	xpathCtx := ec.newXPathContext(ec.contextNode)
	result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
	if err != nil {
		return err
	}
	out := ec.currentOutput()

	// In capture mode, accumulate items directly instead of writing to DOM.
	// Only capture when we are at the function's root output level (the
	// document element wrapper). When nested inside a result element
	// (e.g. an LRE or xsl:copy body), items must be written to the DOM
	// as children of that element.
	// For the principal output with json/adaptive method, the current node
	// is the Document itself (not a wrapper element), so also match that case.
	atRootLevel := out.doc != nil && (out.current == out.doc.DocumentElement() || out.current == out.doc)
	if out.captureItems && atRootLevel {
		out.pendingItems = append(out.pendingItems, result.Sequence()...)
		if len(result.Sequence()) > 0 {
			out.noteOutput()
		}
		return nil
	}

	prevWasAtomic := out.prevWasAtomic
	prevHadOutput := out.prevHadOutput
	hadAtomic := false // tracks whether any atomic (including empty) was seen
	seq := flattenArraysInSequence(result.Sequence())
	for _, item := range seq {
		switch v := item.(type) {
		case xpath3.NodeItem:
			prevWasAtomic = false
			hadAtomic = false
			if normalizeNode(v.Node) == nil {
				continue
			}
			if v.Node.Type() == helium.AttributeNode {
				// Attribute nodes: add as attribute to current element
				attr := v.Node.(*helium.Attribute)
				elem, ok := out.current.(*helium.Element)
				if ok {
					copyAttributeToElement(elem, attr)
					out.noteOutput()
				}
				continue
			}
			if v.Node.Type() == helium.NamespaceNode {
				// Namespace nodes: add as namespace declaration to current element.
				// This supports xsl:sequence select="namespace::*[...]" patterns
				// where namespace nodes must be added before attributes/children.
				nsw, ok := v.Node.(*helium.NamespaceNodeWrapper)
				if ok {
					elem, eok := out.current.(*helium.Element)
					if eok {
						prefix := nsw.Name()
						uri := string(nsw.Content())
						if !hasNSDecl(elem, prefix, uri) {
							_ = elem.DeclareNamespace(prefix, uri)
						}
						out.noteOutput()
					}
				}
				continue
			}
			if v.Node.Type() == helium.DocumentNode {
				// Document nodes: output their children (per XSLT spec,
				// a document node in a sequence constructor is replaced
				// by its children).
				for child := v.Node.FirstChild(); child != nil; child = child.NextSibling() {
					copied, copyErr := helium.CopyNode(child, ec.resultDoc)
					if copyErr != nil {
						return copyErr
					}
					if err := ec.addNode(copied); err != nil {
						return err
					}
				}
				prevHadOutput = true
				continue
			}
			copied, copyErr := helium.CopyNode(v.Node, ec.resultDoc)
			if copyErr != nil {
				return copyErr
			}
			if err := ec.addNode(copied); err != nil {
				return err
			}
			// Fix namespace declarations for copied elements relative to
			// their new position in the result tree (e.g., add xmlns=""
			// on no-namespace elements under a default-namespace parent).
			if elem, ok := copied.(*helium.Element); ok {
				ec.fixNamespacesAfterCopy(elem)
				fixDescendantDefaultNS(elem)
			}
			prevHadOutput = true
		case xpath3.AtomicValue:
			s, sErr := xpath3.AtomicToString(v)
			if sErr != nil {
				return sErr
			}
			// Zero-length atomic strings produce no text node (XSLT 3.0
			// §11.4.1). Within the same sequence constructor scope, skip
			// separator for empty strings. Across scope boundaries (e.g.,
			// different for-each iterations), insert separator so that
			// consecutive atomic values get proper inter-item spacing.
			// addNodeUntracked is used so the separator does NOT count as
			// output for on-empty detection.
			if s == "" {
				if prevWasAtomic && !out.wherePopulated && out.emptyAtomicGen != out.seqConstructorGen {
					sepStr := " "
					if out.itemSeparator != nil {
						sepStr = *out.itemSeparator
					}
					if sepStr != "" {
						sep, tErr := ec.resultDoc.CreateText([]byte(sepStr))
						if tErr != nil {
							return tErr
						}
						if err := ec.addNodeUntracked(sep); err != nil {
							return err
						}
					}
				}
				out.emptyAtomicGen = out.seqConstructorGen
				hadAtomic = true
				continue
			}
			// Insert separator between consecutive items.
			// When item-separator is explicitly set, insert between any
			// adjacent items (node→atomic or atomic→atomic). Otherwise,
			// only insert the default space between adjacent atomics.
			if prevWasAtomic || (prevHadOutput && out.itemSeparator != nil) {
				sepStr := " "
				if out.itemSeparator != nil {
					sepStr = *out.itemSeparator
				}
				if sepStr != "" {
					sep, tErr := ec.resultDoc.CreateText([]byte(sepStr))
					if tErr != nil {
						return tErr
					}
					if err := ec.addNode(sep); err != nil {
						return err
					}
				}
			}
			text, tErr := ec.resultDoc.CreateText([]byte(s))
			if tErr != nil {
				return tErr
			}
			ec.markAtomicTextNode(text)
			if err := ec.addNode(text); err != nil {
				return err
			}
			prevWasAtomic = true
			hadAtomic = true
			prevHadOutput = true
		case xpath3.FunctionItem, xpath3.MapItem, *xpath3.ArrayItem:
			// XTDE0450: function items (including maps and arrays) cannot
			// appear in result tree content — unless the output method
			// supports items (json/adaptive).
			if !out.captureItems && !ec.isItemOutputMethod() {
				return dynamicError(errCodeXTDE0450,
					"cannot add a %T to the result tree", item)
			}
			out.pendingItems = append(out.pendingItems, item)
		default:
			if out.captureItems {
				out.pendingItems = append(out.pendingItems, item)
			}
		}
	}
	// Set prevWasAtomic based on whether any atomic was encountered
	// (including empty strings). This allows separate xsl:sequence
	// instructions producing empty strings to generate inter-atomic
	// separators, while empty strings within a single expression don't.
	if hadAtomic {
		out.prevWasAtomic = true
	} else {
		out.prevWasAtomic = prevWasAtomic
	}
	out.prevHadOutput = prevHadOutput
	return nil
}

// outputSequence writes a sequence of items to the current output.
func (ec *execContext) outputSequence(seq xpath3.Sequence) error {
	out := ec.currentOutput()

	// In capture mode, accumulate items directly (handles maps, arrays,
	// functions, and other non-DOM items that cannot be serialized to a tree).
	if out.captureItems {
		out.pendingItems = append(out.pendingItems, seq...)
		if len(seq) > 0 {
			out.noteOutput()
		}
		return nil
	}

	seq = flattenArraysInSequence(seq)
	prevWasAtomic := out.prevWasAtomic
	for _, item := range seq {
		switch v := item.(type) {
		case xpath3.NodeItem:
			prevWasAtomic = false
			if v.Node.Type() == helium.DocumentNode {
				// Document nodes: output their children.
				for child := v.Node.FirstChild(); child != nil; child = child.NextSibling() {
					copied, copyErr := helium.CopyNode(child, ec.resultDoc)
					if copyErr != nil {
						return copyErr
					}
					if err := ec.addNode(copied); err != nil {
						return err
					}
				}
				continue
			}
			if v.Node.Type() == helium.AttributeNode {
				// Attribute nodes: add as attribute on the current output element.
				attr, ok := v.Node.(*helium.Attribute)
				if ok {
					if elem, eOK := out.current.(*helium.Element); eOK {
						if attr.URI() != "" {
							ns, nsErr := ec.resultDoc.CreateNamespace(attr.Prefix(), attr.URI())
							if nsErr != nil {
								return nsErr
							}
							elem.SetLiteralAttributeNS(attr.LocalName(), attr.Value(), ns)
						} else {
							elem.SetLiteralAttribute(attr.Name(), attr.Value())
						}
						continue
					}
				}
			}
			copied, copyErr := helium.CopyNode(v.Node, ec.resultDoc)
			if copyErr != nil {
				return copyErr
			}
			if err := ec.addNode(copied); err != nil {
				return err
			}
		case xpath3.AtomicValue:
			s, sErr := xpath3.AtomicToString(v)
			if sErr != nil {
				return sErr
			}
			if s == "" {
				prevWasAtomic = true
				continue
			}
			// Insert separator between consecutive items.
			// When item-separator is explicitly set, insert between any
			// adjacent items (node→atomic or atomic→atomic). Otherwise,
			// only insert the default space between adjacent atomics.
			if prevWasAtomic || (out.prevHadOutput && out.itemSeparator != nil) {
				sepStr := " "
				if out.itemSeparator != nil {
					sepStr = *out.itemSeparator
				}
				if sepStr != "" {
					sep, tErr := ec.resultDoc.CreateText([]byte(sepStr))
					if tErr != nil {
						return tErr
					}
					if err := ec.addNode(sep); err != nil {
						return err
					}
				}
			}
			text, tErr := ec.resultDoc.CreateText([]byte(s))
			if tErr != nil {
				return tErr
			}
			ec.markAtomicTextNode(text)
			if err := ec.addNode(text); err != nil {
				return err
			}
			prevWasAtomic = true
		case xpath3.FunctionItem, xpath3.MapItem:
			if out.captureItems || ec.isItemOutputMethod() {
				out.pendingItems = append(out.pendingItems, item)
			} else {
				return dynamicError(errCodeXTDE0450,
					"cannot add non-node item to the result tree")
			}
		}
	}
	out.prevWasAtomic = prevWasAtomic
	return nil
}

// atomizeSequence converts each item in the sequence to an atomic value.
// Arrays are flattened before atomization.
func atomizeSequence(seq xpath3.Sequence) xpath3.Sequence {
	seq = flattenArraysInSequence(seq)
	result := make(xpath3.Sequence, 0, len(seq))
	for _, item := range seq {
		av, err := xpath3.AtomizeItem(item)
		if err != nil {
			result = append(result, xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: stringifyItem(item)})
		} else {
			result = append(result, av)
		}
	}
	return result
}
