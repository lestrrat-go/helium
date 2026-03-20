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
			return dynamicError("FOTY0013", "cannot atomize a map item")
		case xpath3.FunctionItem:
			return dynamicError("FOTY0013", "cannot atomize a function item")
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
		if inst.HasSeparator {
			if inst.Separator != nil {
				var err error
				separator, err = inst.Separator.evaluate(ctx, ec.contextNode)
				if err != nil {
					return err
				}
			} else {
				separator = ""
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
		if inst.HasSeparator && inst.Separator != nil {
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
		// For TVTs that evaluate to empty, skip the text node
		if value == "" {
			return nil
		}
	}
	// Note: xsl:text with empty literal content still produces a zero-length
	// text node. This is needed for variables with as="text()" to receive
	// exactly 1 item. Only TVT-expanded empty values are skipped (above).
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
	if out.captureItems && out.doc != nil && out.current == out.doc.DocumentElement() {
		out.pendingItems = append(out.pendingItems, result.Sequence()...)
		if len(result.Sequence()) > 0 {
			out.noteOutput()
		}
		return nil
	}

	prevWasAtomic := out.prevWasAtomic
	seq := flattenArraysInSequence(result.Sequence())
	for _, item := range seq {
		switch v := item.(type) {
		case xpath3.NodeItem:
			prevWasAtomic = false
			if normalizeNode(v.Node) == nil {
				continue
			}
			if v.Node.Type() == helium.AttributeNode {
				// Attribute nodes: add as attribute to current element
				attr := v.Node.(*helium.Attribute)
				elem, ok := out.current.(*helium.Element)
				if ok {
					if err := copyAttributeToElement(elem, attr); err != nil {
						return err
					}
					out.noteOutput()
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
				continue
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
			// Insert separator between consecutive atomic values.
			// Use item-separator from the output frame if set, otherwise default space.
			if prevWasAtomic {
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
		case xpath3.FunctionItem:
			// XTDE0450: function items cannot appear in result tree content.
			if !out.captureItems {
				return dynamicError(errCodeXTDE0450,
					"cannot add function item to result tree content")
			}
			out.pendingItems = append(out.pendingItems, item)
		default:
			if out.captureItems {
				out.pendingItems = append(out.pendingItems, item)
			}
			// Maps/arrays are silently skipped from non-capture output.
			// A future enhancement may serialize them with item-separator.
		}
	}
	out.prevWasAtomic = prevWasAtomic
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
							if err := elem.SetAttributeNS(attr.LocalName(), attr.Value(), ns); err != nil {
								return err
							}
						} else {
							if err := elem.SetAttribute(attr.Name(), attr.Value()); err != nil {
								return err
							}
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
			if prevWasAtomic {
				// Use the item-separator from the output frame if set,
				// otherwise default to a single space between adjacent atomics.
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
			if out.captureItems {
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
