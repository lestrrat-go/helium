package xslt3

import (
	"bytes"
	"fmt"
	"io"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/xpath3"
)

// serializeAdaptiveItems serializes a sequence of items using the adaptive
// serialization method. Each item is serialized according to its type.
func serializeAdaptiveItems(w io.Writer, items xpath3.Sequence, doc *helium.Document, itemSep *string, charMaps ...map[rune]string) error {
	if (items == nil || sequence.Len(items) == 0) && doc != nil {
		var cm map[rune]string
		if len(charMaps) > 0 {
			cm = charMaps[0]
		}
		return serializeXML(w, doc, defaultOutputDef(), cm)
	}
	var cm map[rune]string
	if len(charMaps) > 0 {
		cm = charMaps[0]
	}
	sep := "\n"
	if itemSep != nil {
		sep = *itemSep
	}
	// Per XSLT 3.0 §20: when adaptive output contains a single document or
	// element node, serialize it using the XML method (which includes the
	// XML declaration by default). When there are multiple items, serialize
	// each without a declaration.
	singleNodeItem := items != nil && sequence.Len(items) == 1
	adaptIdx := 0
	for item := range sequence.Items(items) {
		if adaptIdx > 0 && sep != "" {
			if _, err := io.WriteString(w, sep); err != nil {
				return err
			}
		}
		if singleNodeItem {
			if ni, ok := item.(xpath3.NodeItem); ok {
				if elem, ok := ni.Node.(*helium.Element); ok {
					// Wrap in a temp document to get the XML declaration.
					tmpDoc := helium.NewDefaultDocument()
					clone, _ := helium.CopyNode(elem, tmpDoc)
					if clone != nil {
						_ = tmpDoc.AddChild(clone)
					}
					xmlOutDef := defaultOutputDef()
					if err := serializeXML(w, tmpDoc, xmlOutDef, cm); err != nil {
						return err
					}
					continue
				}
			}
		}
		s := serializeItemAdaptive(item, cm)
		if _, err := io.WriteString(w, s); err != nil {
			return err
		}
		adaptIdx++
	}
	return nil
}

// adaptiveQuoteString wraps a string in double quotes for adaptive output.
// Unlike JSON escaping, adaptive serialization only escapes embedded double
// quotes by doubling them (XSLT 3.0 §26.3.5). Other characters such as
// backslashes are emitted literally.
func adaptiveQuoteString(s string) string {
	var buf bytes.Buffer
	buf.WriteByte('"')
	for _, r := range s {
		if r == '"' {
			buf.WriteString("\"\"")
		} else {
			buf.WriteRune(r)
		}
	}
	buf.WriteByte('"')
	return buf.String()
}

// isAdaptiveQuotedType returns true when an atomic type should be serialized
// with double-quote wrapping in adaptive output (XSLT 3.0 §26.3.5).
// String-like types use JSON quoting; numeric types and xs:boolean do not.
func isAdaptiveQuotedType(typeName string) bool {
	switch typeName {
	case "xs:boolean",
		"xs:integer", "xs:decimal", "xs:float", "xs:double",
		"xs:long", "xs:int", "xs:short", "xs:byte",
		"xs:unsignedLong", "xs:unsignedInt", "xs:unsignedShort", "xs:unsignedByte",
		"xs:nonNegativeInteger", "xs:nonPositiveInteger",
		"xs:positiveInteger", "xs:negativeInteger":
		return false
	}
	return true
}

// serializeItemAdaptive serializes a single item using the adaptive method.
func serializeItemAdaptive(item xpath3.Item, charMap map[rune]string) string {
	maybeApply := func(s string) string {
		if len(charMap) > 0 {
			return applyCharMap(s, charMap)
		}
		return s
	}
	switch v := item.(type) {
	case xpath3.MapItem:
		return serializeMapAdaptive(v, charMap)
	case xpath3.ArrayItem:
		return serializeArrayAdaptive(v, charMap)
	case xpath3.NodeItem:
		var buf bytes.Buffer
		switch v.Node.(type) {
		case *helium.Element, *helium.Document:
			_ = helium.NewWriter().XMLDeclaration(false).WriteTo(&buf, v.Node)
		case *helium.Attribute:
			attr := v.Node.(*helium.Attribute) //nolint:forcetypeassert
			buf.WriteString(attr.Name())
			buf.WriteString("=\"")
			buf.Write(attr.Content())
			buf.WriteString("\"")
		default:
			buf.Write(v.Node.Content())
		}
		return maybeApply(buf.String())
	case xpath3.AtomicValue:
		s, _ := xpath3.AtomicToString(v)
		s = maybeApply(s)
		if isAdaptiveQuotedType(v.TypeName) {
			return adaptiveQuoteString(s)
		}
		return s
	default:
		if av, ok := item.(xpath3.AtomicValue); ok {
			s, _ := xpath3.AtomicToString(av)
			s = maybeApply(s)
			if isAdaptiveQuotedType(av.TypeName) {
				return adaptiveQuoteString(s)
			}
			return s
		}
		return fmt.Sprintf("%v", item)
	}
}

// serializeMapAdaptive serializes a map using adaptive serialization.
func serializeMapAdaptive(m xpath3.MapItem, charMap map[rune]string) string {
	var buf bytes.Buffer
	buf.WriteString("map{")
	first := true
	_ = m.ForEach(func(k xpath3.AtomicValue, v xpath3.Sequence) error {
		if !first {
			buf.WriteByte(',')
		}
		first = false
		ks, _ := xpath3.AtomicToString(k)
		buf.WriteString(jsonEscapeString(ks))
		buf.WriteByte(':')
		vLen2 := 0
		if v != nil {
			vLen2 = sequence.Len(v)
		}
		switch vLen2 {
		case 1:
			buf.WriteString(serializeItemAdaptive(v.Get(0), charMap))
		case 0:
			buf.WriteString("()")
		default:
			buf.WriteByte('(')
			for i := range vLen2 {
				if i > 0 {
					buf.WriteByte(',')
				}
				buf.WriteString(serializeItemAdaptive(v.Get(i), charMap))
			}
			buf.WriteByte(')')
		}
		return nil
	})
	buf.WriteByte('}')
	return buf.String()
}

// serializeArrayAdaptive serializes an array using adaptive serialization.
func serializeArrayAdaptive(a xpath3.ArrayItem, charMap map[rune]string) string {
	var buf bytes.Buffer
	buf.WriteByte('[')
	members := a.Members()
	for i, member := range members {
		if i > 0 {
			buf.WriteByte(',')
		}
		mLen2 := 0
		if member != nil {
			mLen2 = sequence.Len(member)
		}
		switch mLen2 {
		case 1:
			buf.WriteString(serializeItemAdaptive(member.Get(0), charMap))
		case 0:
			buf.WriteString("()")
		default:
			buf.WriteByte('(')
			for j := range mLen2 {
				if j > 0 {
					buf.WriteByte(',')
				}
				buf.WriteString(serializeItemAdaptive(member.Get(j), charMap))
			}
			buf.WriteByte(')')
		}
	}
	buf.WriteByte(']')
	return buf.String()
}
