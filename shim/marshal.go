package shim

import (
	"bytes"
	"encoding"
	stdxml "encoding/xml"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// marshalValue is the main marshal dispatch. It encodes val as XML,
// optionally using the provided start element for the outer tag.
func (enc *Encoder) marshalValue(v any, start *StartElement) error {
	val := reflect.ValueOf(v)
	return enc.marshalReflectValue(val, start)
}

func (enc *Encoder) marshalReflectValue(val reflect.Value, start *StartElement) error {
	if !val.IsValid() {
		return nil
	}

	// Dereference pointers and interfaces
	for val.Kind() == reflect.Pointer || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	// Check Marshaler interface
	if val.CanInterface() {
		if m, ok := val.Interface().(Marshaler); ok {
			return enc.marshalViaHook(m, val, start)
		}
	}
	if val.CanAddr() {
		if m, ok := val.Addr().Interface().(Marshaler); ok {
			return enc.marshalViaHook(m, val, start)
		}
	}

	// Check TextMarshaler interface
	if val.CanInterface() {
		if m, ok := val.Interface().(encoding.TextMarshaler); ok {
			return enc.marshalTextMarshaler(m, val, start)
		}
	}
	if val.CanAddr() {
		if m, ok := val.Addr().Interface().(encoding.TextMarshaler); ok {
			return enc.marshalTextMarshaler(m, val, start)
		}
	}

	switch val.Kind() {
	case reflect.Struct:
		return enc.marshalStruct(val, start)
	case reflect.Slice, reflect.Array:
		if val.Type().Elem().Kind() == reflect.Uint8 {
			// []byte / [N]byte — marshal as text
			return enc.marshalSimpleValue(val, start)
		}
		// Slice/array of elements
		for i := 0; i < val.Len(); i++ {
			if err := enc.marshalReflectValue(val.Index(i), start); err != nil {
				return err
			}
		}
		return nil
	default:
		return enc.marshalSimpleValue(val, start)
	}
}

// marshalViaHook calls the Marshaler's MarshalXML through a stdlib encoder bridge.
func (enc *Encoder) marshalViaHook(m Marshaler, val reflect.Value, start *StartElement) error {
	se := enc.defaultStart(val, start)

	var buf bytes.Buffer
	stdEnc := stdxml.NewEncoder(&buf)
	if enc.indent != "" || enc.prefix != "" {
		stdEnc.Indent(enc.prefix, enc.indent)
	}
	if err := m.MarshalXML(stdEnc, se); err != nil {
		return err
	}
	if err := stdEnc.Flush(); err != nil {
		return err
	}

	// Write the hook's output directly
	_, err := enc.w.Write(buf.Bytes())
	return err
}

// marshalTextMarshaler handles encoding.TextMarshaler values.
func (enc *Encoder) marshalTextMarshaler(m encoding.TextMarshaler, val reflect.Value, start *StartElement) error {
	text, err := m.MarshalText()
	if err != nil {
		return err
	}

	se := enc.defaultStart(val, start)
	if err := enc.EncodeToken(se); err != nil {
		return err
	}
	if err := enc.EncodeToken(CharData(text)); err != nil {
		return err
	}
	return enc.EncodeToken(se.End())
}

// marshalStruct encodes a struct value as XML.
func (enc *Encoder) marshalStruct(val reflect.Value, start *StartElement) error {
	bindings, err := buildFieldBindings(val.Type())
	if err != nil {
		return err
	}

	se := enc.buildStructStart(val, bindings, start)

	// Collect attr fields
	for _, b := range bindings {
		if !b.isAttr || b.omit || !b.fieldExport {
			continue
		}
		field, ok := fieldByIndexAlloc(val, b.index)
		if !ok {
			continue
		}
		attr, err := enc.marshalAttr(b, field)
		if err != nil {
			return err
		}
		if attr != nil {
			se.Attr = append(se.Attr, *attr)
		}
	}

	if err := enc.EncodeToken(se); err != nil {
		return err
	}

	// Encode content fields
	for _, b := range bindings {
		if b.omit || !b.fieldExport || b.isAttr || b.isXMLName {
			continue
		}
		field, ok := fieldByIndexAlloc(val, b.index)
		if !ok {
			continue
		}

		if b.omitEmpty && isEmptyValue(field) {
			continue
		}

		switch {
		case b.isCharData:
			text := textValue(field)
			if text != "" {
				if err := enc.EncodeToken(CharData([]byte(text))); err != nil {
					return err
				}
			}
		case b.isCData:
			text := textValue(field)
			if text != "" {
				// Write raw CDATA — bypass EncodeToken
				enc.w.WriteString("<![CDATA[")
				enc.w.WriteString(text)
				enc.w.WriteString("]]>")
				enc.lastWasStart = false
				enc.lastWasText = true
			}
		case b.isInnerXML:
			raw := textValue(field)
			if raw != "" {
				// Write raw XML — bypass escaping
				enc.w.WriteString(raw)
				enc.lastWasStart = false
				enc.lastWasText = false
			}
		case b.isComment:
			text := textValue(field)
			if text != "" {
				if err := enc.EncodeToken(Comment([]byte(text))); err != nil {
					return err
				}
			}
		case b.isAny:
			if err := enc.marshalReflectValue(field, nil); err != nil {
				return err
			}
		default:
			// Element field
			if err := enc.marshalField(b, field); err != nil {
				return err
			}
		}
	}

	return enc.EncodeToken(se.End())
}

// marshalField encodes a struct field as a child element.
func (enc *Encoder) marshalField(b fieldBinding, field reflect.Value) error {
	// Handle path tags (a>b>c)
	path := b.path
	if len(path) > 0 {
		return enc.marshalPathField(path, field)
	}

	name := b.name
	if name == "" {
		name = b.fieldName
	}

	// For slices of non-byte elements, marshal each element
	if field.Kind() == reflect.Slice && field.Type().Elem().Kind() != reflect.Uint8 {
		for i := 0; i < field.Len(); i++ {
			elemStart := StartElement{Name: Name{Local: name}}
			if b.hasNameSpace {
				elemStart.Name.Space = b.nameSpace
			}
			if err := enc.marshalReflectValue(field.Index(i), &elemStart); err != nil {
				return err
			}
		}
		return nil
	}

	elemStart := StartElement{Name: Name{Local: name}}
	if b.hasNameSpace {
		elemStart.Name.Space = b.nameSpace
	}
	return enc.marshalReflectValue(field, &elemStart)
}

// marshalPathField encodes a field with a path tag (e.g., "a>b>c").
func (enc *Encoder) marshalPathField(path []string, field reflect.Value) error {
	// Open nested elements
	for _, name := range path[:len(path)-1] {
		if err := enc.EncodeToken(StartElement{Name: Name{Local: name}}); err != nil {
			return err
		}
	}

	// Marshal the leaf
	leafName := path[len(path)-1]
	leafStart := StartElement{Name: Name{Local: leafName}}

	if field.Kind() == reflect.Slice && field.Type().Elem().Kind() != reflect.Uint8 {
		for i := 0; i < field.Len(); i++ {
			if err := enc.marshalReflectValue(field.Index(i), &leafStart); err != nil {
				return err
			}
		}
	} else {
		if err := enc.marshalReflectValue(field, &leafStart); err != nil {
			return err
		}
	}

	// Close nested elements in reverse
	for i := len(path) - 2; i >= 0; i-- {
		if err := enc.EncodeToken(EndElement{Name: Name{Local: path[i]}}); err != nil {
			return err
		}
	}

	return nil
}

// marshalSimpleValue encodes a simple (non-struct, non-slice) value.
func (enc *Encoder) marshalSimpleValue(val reflect.Value, start *StartElement) error {
	text, err := simpleText(val)
	if err != nil {
		return err
	}

	se := enc.defaultStart(val, start)
	if err := enc.EncodeToken(se); err != nil {
		return err
	}
	if text != "" {
		if err := enc.EncodeToken(CharData([]byte(text))); err != nil {
			return err
		}
	}
	return enc.EncodeToken(se.End())
}

// marshalAttr encodes a field binding as an XML attribute.
func (enc *Encoder) marshalAttr(b fieldBinding, field reflect.Value) (*Attr, error) {
	if b.omitEmpty && isEmptyValue(field) {
		return nil, nil
	}

	for field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return nil, nil
		}
		field = field.Elem()
	}

	name := Name{Local: b.name}
	if b.hasNameSpace {
		name.Space = b.nameSpace
	}

	// MarshalerAttr check
	if field.CanInterface() {
		if m, ok := field.Interface().(MarshalerAttr); ok {
			attr, err := m.MarshalXMLAttr(name)
			if err != nil {
				return nil, err
			}
			if attr.Name.Local == "" {
				return nil, nil
			}
			return &attr, nil
		}
	}
	if field.CanAddr() {
		if m, ok := field.Addr().Interface().(MarshalerAttr); ok {
			attr, err := m.MarshalXMLAttr(name)
			if err != nil {
				return nil, err
			}
			if attr.Name.Local == "" {
				return nil, nil
			}
			return &attr, nil
		}
	}

	// TextMarshaler check
	if field.CanInterface() {
		if m, ok := field.Interface().(encoding.TextMarshaler); ok {
			text, err := m.MarshalText()
			if err != nil {
				return nil, err
			}
			return &Attr{Name: name, Value: string(text)}, nil
		}
	}
	if field.CanAddr() {
		if m, ok := field.Addr().Interface().(encoding.TextMarshaler); ok {
			text, err := m.MarshalText()
			if err != nil {
				return nil, err
			}
			return &Attr{Name: name, Value: string(text)}, nil
		}
	}

	text, err := simpleText(field)
	if err != nil {
		return nil, err
	}
	return &Attr{Name: name, Value: text}, nil
}

// buildStructStart determines the StartElement for a struct value.
func (enc *Encoder) buildStructStart(val reflect.Value, bindings []fieldBinding, override *StartElement) StartElement {
	if override != nil && override.Name.Local != "" {
		return *override
	}

	// Check XMLName field value
	for _, b := range bindings {
		if !b.isXMLName {
			continue
		}
		field, ok := fieldByIndexAlloc(val, b.index)
		if !ok {
			continue
		}
		for field.Kind() == reflect.Pointer {
			if field.IsNil() {
				break
			}
			field = field.Elem()
		}
		if field.Kind() == reflect.Struct && isXMLNameType(field.Type()) {
			local := field.FieldByName("Local")
			space := field.FieldByName("Space")
			if local.IsValid() && local.String() != "" {
				name := Name{Local: local.String()}
				if space.IsValid() {
					name.Space = space.String()
				}
				return StartElement{Name: name}
			}
		}
		// Check tag on XMLName
		if b.rawName != "" && b.rawName != "XMLName" && b.name != "" {
			name := Name{Local: b.name}
			if b.hasNameSpace {
				name.Space = b.nameSpace
			}
			return StartElement{Name: name}
		}
	}

	// Fall back to type name
	t := val.Type()
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return StartElement{Name: Name{Local: typeName(t)}}
}

// defaultStart returns the start element to use for a non-struct value.
func (enc *Encoder) defaultStart(val reflect.Value, start *StartElement) StartElement {
	if start != nil && start.Name.Local != "" {
		return *start
	}
	t := val.Type()
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return StartElement{Name: Name{Local: typeName(t)}}
}

// typeName returns the type name with generic type parameters stripped.
func typeName(t reflect.Type) string {
	name := t.Name()
	if i := strings.IndexByte(name, '['); i >= 0 {
		return name[:i]
	}
	return name
}

func simpleText(val reflect.Value) (string, error) {
	for val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return "", nil
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.String:
		return val.String(), nil
	case reflect.Bool:
		return strconv.FormatBool(val.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(val.Uint(), 10), nil
	case reflect.Float32:
		return strconv.FormatFloat(val.Float(), 'g', -1, 32), nil
	case reflect.Float64:
		return strconv.FormatFloat(val.Float(), 'g', -1, 64), nil
	case reflect.Slice:
		if val.Type().Elem().Kind() == reflect.Uint8 {
			return string(val.Bytes()), nil
		}
		return "", &UnsupportedTypeError{Type: val.Type()}
	case reflect.Array:
		if val.Type().Elem().Kind() == reflect.Uint8 {
			b := make([]byte, val.Len())
			for i := range b {
				b[i] = byte(val.Index(i).Uint())
			}
			return string(b), nil
		}
		return "", &UnsupportedTypeError{Type: val.Type()}
	case reflect.Map:
		return "", &UnsupportedTypeError{Type: val.Type()}
	}

	return fmt.Sprintf("%v", val.Interface()), nil
}

func textValue(field reflect.Value) string {
	for field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return ""
		}
		field = field.Elem()
	}

	// Check TextMarshaler interface
	if field.CanInterface() {
		if m, ok := field.Interface().(encoding.TextMarshaler); ok {
			text, err := m.MarshalText()
			if err == nil {
				return string(text)
			}
		}
	}
	if field.CanAddr() {
		if m, ok := field.Addr().Interface().(encoding.TextMarshaler); ok {
			text, err := m.MarshalText()
			if err == nil {
				return string(text)
			}
		}
	}

	switch field.Kind() {
	case reflect.String:
		return field.String()
	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.Uint8 {
			return string(field.Bytes())
		}
	}
	return fmt.Sprintf("%v", field.Interface())
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice:
		return v.Len() == 0
	case reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Pointer, reflect.Interface:
		return v.IsNil()
	}
	return false
}
