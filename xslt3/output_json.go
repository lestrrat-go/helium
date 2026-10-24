package xslt3

import (
	"bytes"
	"fmt"
	"io"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/xpath3"
)

// serializeJSONItems serializes a sequence of items as JSON output.
// Per XSLT 3.0 §26: the sequence is serialized as a single JSON value.
func serializeJSONItems(w io.Writer, items xpath3.Sequence, doc *helium.Document, outDef *OutputDef) error {
	if sequence.Len(items) == 0 && doc != nil && doc.FirstChild() != nil {
		items = xpath3.ItemSlice{xpath3.NodeItem{Node: doc}}
	}
	if sequence.Len(items) > 1 {
		members := make([]xpath3.Sequence, sequence.Len(items))
		for i := range members {
			members[i] = xpath3.ItemSlice{items.Get(i)}
		}
		items = xpath3.ItemSlice{xpath3.NewArray(members)}
	}
	s, err := serializeSequenceJSON(items, outDef)
	if err != nil {
		return err
	}
	return writeFullString(w, s)
}

// serializeSequenceJSON applies the same cardinality rule to the result
// sequence, map values, and array members. An explicit array is one item.
func serializeSequenceJSON(items xpath3.Sequence, outDef *OutputDef) (string, error) {
	switch sequence.Len(items) {
	case 0:
		return "null", nil
	case 1:
		return serializeItemJSON(items.Get(0), outDef)
	default:
		return "", dynamicError(lexicon.ErrSERE0023, "JSON output requires at most one item per value")
	}
}

// serializeItemJSON serializes a single XDM item as JSON. The json-node-output
// method and its serialization parameters (e.g. html-version) are taken from
// outDef, which may be nil.
func serializeItemJSON(item xpath3.Item, outDef *OutputDef) (string, error) {
	nodeMethod := ""
	if outDef != nil {
		nodeMethod = outDef.JSONNodeOutputMethod
	}
	switch v := item.(type) {
	case xpath3.MapItem:
		return serializeMapJSON(v, outDef)
	case xpath3.ArrayItem:
		return serializeArrayJSON(v, outDef)
	case xpath3.NodeItem:
		if nodeMethod != "" && nodeMethod != methodText {
			htmlVersion := ""
			if outDef != nil {
				htmlVersion = outDef.HTMLVersion
			}
			s, err := serializeNodeWithMethod(v.Node, nodeMethod, htmlVersion)
			if err != nil {
				return "", err
			}
			return jsonEscapeString(s), nil
		}
		return jsonEscapeString(nodeStringValue(v.Node)), nil
	case xpath3.AtomicValue:
		return serializeAtomicJSON(v)
	default:
		return "", dynamicError(lexicon.ErrSERE0021, "item of type %T cannot be serialized as JSON", item)
	}
}

// serializeMapJSON serializes a map as a JSON object.
func serializeMapJSON(m xpath3.MapItem, outDef *OutputDef) (string, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	first := true
	err := m.ForEach(func(k xpath3.AtomicValue, v xpath3.Sequence) error {
		if !first {
			buf.WriteByte(',')
		}
		first = false
		ks, _ := xpath3.AtomicToString(k)
		buf.WriteString(jsonEscapeString(ks))
		buf.WriteByte(':')
		s, err := serializeSequenceJSON(v, outDef)
		if err != nil {
			return err
		}
		buf.WriteString(s)
		return nil
	})
	if err != nil {
		return "", err
	}
	buf.WriteByte('}')
	return buf.String(), nil
}

// serializeArrayJSON serializes an array as a JSON array.
func serializeArrayJSON(a xpath3.ArrayItem, outDef *OutputDef) (string, error) {
	var buf bytes.Buffer
	buf.WriteByte('[')
	members := a.Members()
	for i, member := range members {
		if i > 0 {
			buf.WriteByte(',')
		}
		s, err := serializeSequenceJSON(member, outDef)
		if err != nil {
			return "", err
		}
		buf.WriteString(s)
	}
	buf.WriteByte(']')
	return buf.String(), nil
}

// serializeAtomicJSON serializes an atomic value as JSON.
func serializeAtomicJSON(v xpath3.AtomicValue) (string, error) {
	typeName := v.TypeName
	switch typeName {
	case "xs:boolean":
		b, _ := xpath3.AtomicToString(v)
		return b, nil
	case "xs:integer", "xs:int", "xs:long", "xs:short", "xs:byte",
		"xs:unsignedInt", "xs:unsignedLong", "xs:unsignedShort", "xs:unsignedByte",
		"xs:positiveInteger", "xs:nonNegativeInteger", "xs:negativeInteger", "xs:nonPositiveInteger":
		s, _ := xpath3.AtomicToString(v)
		return s, nil
	case "xs:double", "xs:float", "xs:decimal":
		s, _ := xpath3.AtomicToString(v)
		if s == "NaN" || s == "INF" || s == "-INF" || s == "+INF" {
			return "", dynamicError(lexicon.ErrSERE0020, "numeric value %s cannot be represented in JSON", s)
		}
		return s, nil
	default:
		s, _ := xpath3.AtomicToString(v)
		return jsonEscapeString(s), nil
	}
}

// jsonEscapeString returns a JSON-escaped double-quoted string.
func jsonEscapeString(s string) string {
	var buf bytes.Buffer
	buf.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString("\\\"")
		case '\\':
			buf.WriteString("\\\\")
		case '\n':
			buf.WriteString("\\n")
		case '\r':
			buf.WriteString("\\r")
		case '\t':
			buf.WriteString("\\t")
		case '\b':
			buf.WriteString("\\b")
		case '\f':
			buf.WriteString("\\f")
		case '/':
			buf.WriteString("\\/")
		default:
			if r < 0x20 {
				fmt.Fprintf(&buf, "\\u%04x", r)
			} else {
				buf.WriteRune(r)
			}
		}
	}
	buf.WriteByte('"')
	return buf.String()
}
