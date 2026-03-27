package xslt3

import (
	"bytes"
	"fmt"
	"io"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/xpath3"
)

// serializeJSONItems serializes a sequence of items as JSON output.
// Per XSLT 3.0 §26: the sequence is serialized as a single JSON value.
func serializeJSONItems(w io.Writer, items xpath3.Sequence, doc *helium.Document, outDef *OutputDef) error {
	itemsLen := 0
	if items != nil {
		itemsLen = sequence.Len(items)
	}
	if itemsLen == 0 && doc != nil {
		// No captured items: serialize DOM content as text for JSON
		return serializeAdaptiveItems(w, items, doc, nil)
	}
	nodeMethod := ""
	if outDef != nil {
		nodeMethod = outDef.JSONNodeOutputMethod
	}
	if itemsLen == 1 {
		s, err := serializeItemJSON(items.Get(0), nodeMethod)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, s)
		return err
	}
	// Multiple items: serialize as JSON array
	members := make([]xpath3.Sequence, itemsLen)
	for i := range itemsLen {
		members[i] = xpath3.ItemSlice{items.Get(i)}
	}
	s, err := serializeItemJSON(xpath3.NewArray(members), nodeMethod)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, s)
	return err
}

// serializeItemJSON serializes a single XDM item as JSON.
// nodeMethod specifies the json-node-output-method for serializing nodes.
func serializeItemJSON(item xpath3.Item, nodeMethod string) (string, error) {
	switch v := item.(type) {
	case xpath3.MapItem:
		return serializeMapJSON(v, nodeMethod)
	case xpath3.ArrayItem:
		return serializeArrayJSON(v, nodeMethod)
	case xpath3.NodeItem:
		if nodeMethod != "" && nodeMethod != methodText {
			return jsonEscapeString(serializeNodeWithMethod(v.Node, nodeMethod)), nil
		}
		return jsonEscapeString(nodeStringValue(v.Node)), nil
	case xpath3.AtomicValue:
		return serializeAtomicJSON(v), nil
	default:
		if av, ok := item.(xpath3.AtomicValue); ok {
			return serializeAtomicJSON(av), nil
		}
		return jsonEscapeString(fmt.Sprintf("%v", item)), nil
	}
}

// serializeMapJSON serializes a map as a JSON object.
func serializeMapJSON(m xpath3.MapItem, nodeMethod string) (string, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	first := true
	var serErr error
	_ = m.ForEach(func(k xpath3.AtomicValue, v xpath3.Sequence) error {
		if serErr != nil {
			return serErr
		}
		if !first {
			buf.WriteByte(',')
		}
		first = false
		ks, _ := xpath3.AtomicToString(k)
		buf.WriteString(jsonEscapeString(ks))
		buf.WriteByte(':')
		vLen := 0
		if v != nil {
			vLen = sequence.Len(v)
		}
		switch vLen {
		case 1:
			s, err := serializeItemJSON(v.Get(0), nodeMethod)
			if err != nil {
				serErr = err
				return err
			}
			buf.WriteString(s)
		case 0:
			buf.WriteString("null")
		default:
			members := make([]xpath3.Sequence, vLen)
			for i := range vLen {
				members[i] = xpath3.ItemSlice{v.Get(i)}
			}
			s, err := serializeItemJSON(xpath3.NewArray(members), nodeMethod)
			if err != nil {
				serErr = err
				return err
			}
			buf.WriteString(s)
		}
		return nil
	})
	if serErr != nil {
		return "", serErr
	}
	buf.WriteByte('}')
	return buf.String(), nil
}

// serializeArrayJSON serializes an array as a JSON array.
func serializeArrayJSON(a xpath3.ArrayItem, nodeMethod string) (string, error) {
	var buf bytes.Buffer
	buf.WriteByte('[')
	members := a.Members()
	for i, member := range members {
		if i > 0 {
			buf.WriteByte(',')
		}
		mLen := 0
		if member != nil {
			mLen = sequence.Len(member)
		}
		switch mLen {
		case 1:
			s, err := serializeItemJSON(member.Get(0), nodeMethod)
			if err != nil {
				return "", err
			}
			buf.WriteString(s)
		case 0:
			buf.WriteString("null")
		default:
			// Multi-item member: serialize as array
			submembers := make([]xpath3.Sequence, mLen)
			for j := range mLen {
				submembers[j] = xpath3.ItemSlice{member.Get(j)}
			}
			s, err := serializeItemJSON(xpath3.NewArray(submembers), nodeMethod)
			if err != nil {
				return "", err
			}
			buf.WriteString(s)
		}
	}
	buf.WriteByte(']')
	return buf.String(), nil
}

// serializeAtomicJSON serializes an atomic value as JSON.
func serializeAtomicJSON(v xpath3.AtomicValue) string {
	typeName := v.TypeName
	switch typeName {
	case "xs:boolean":
		b, _ := xpath3.AtomicToString(v)
		return b
	case "xs:integer", "xs:int", "xs:long", "xs:short", "xs:byte",
		"xs:unsignedInt", "xs:unsignedLong", "xs:unsignedShort", "xs:unsignedByte",
		"xs:positiveInteger", "xs:nonNegativeInteger", "xs:negativeInteger", "xs:nonPositiveInteger":
		s, _ := xpath3.AtomicToString(v)
		return s
	case "xs:double", "xs:float", "xs:decimal":
		s, _ := xpath3.AtomicToString(v)
		if s == "NaN" || s == "INF" || s == "-INF" || s == "+INF" {
			return jsonEscapeString(s)
		}
		return s
	default:
		s, _ := xpath3.AtomicToString(v)
		return jsonEscapeString(s)
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
