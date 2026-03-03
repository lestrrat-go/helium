package shim

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

type fieldBinding struct {
	index       int
	name        string
	path        []string
	isAttr      bool
	isCharData  bool
	isInnerXML  bool
	isAny       bool
	isXMLName   bool
	omit        bool
	fieldType   reflect.Type
	fieldIsPtr  bool
	fieldExport bool
}

func Unmarshal(data []byte, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("shim: unmarshal target must be a non-nil pointer")
	}

	doc, err := helium.Parse(context.Background(), data)
	if err != nil {
		return err
	}
	root := doc.DocumentElement()
	if root == nil {
		return fmt.Errorf("shim: no document element")
	}

	return decodeElementInto(rv.Elem(), root)
}

func decodeElementInto(target reflect.Value, elem *helium.Element) error {
	if !target.IsValid() {
		return nil
	}

	if target.Kind() == reflect.Pointer {
		if target.IsNil() {
			target.Set(reflect.New(target.Type().Elem()))
		}
		return decodeElementInto(target.Elem(), elem)
	}

	if target.Kind() != reflect.Struct {
		return assignFromText(target, elementText(elem))
	}

	bindings := buildFieldBindings(target.Type())
	children := childElements(elem)
	consumed := make(map[int]bool)

	for _, binding := range bindings {
		if binding.omit || !binding.fieldExport {
			continue
		}

		field := target.Field(binding.index)

		switch {
		case binding.isXMLName:
			setXMLName(field, elem)
		case binding.isAttr:
			value, ok := attrValue(elem, binding.name)
			if ok {
				if err := assignFromText(field, value); err != nil {
					return err
				}
			}
		case binding.isCharData:
			if err := assignFromText(field, elementText(elem)); err != nil {
				return err
			}
		case binding.isInnerXML:
			if err := assignFromText(field, innerXML(elem)); err != nil {
				return err
			}
		case binding.isAny:
			for idx, anyElem := firstUnconsumed(children, consumed); anyElem != nil; idx, anyElem = firstUnconsumed(children, consumed) {
				consumed[idx] = true
				if err := assignFromElement(field, anyElem); err != nil {
					return err
				}
			}
		default:
			path := binding.path
			if len(path) == 0 {
				path = []string{binding.name}
			}
			idx, matched := findPath(children, consumed, path)
			if matched == nil {
				continue
			}
			consumed[idx] = true
			if err := assignFromElement(field, matched); err != nil {
				return err
			}
		}
	}

	return nil
}

func assignFromElement(field reflect.Value, elem *helium.Element) error {
	if !field.CanSet() {
		return nil
	}

	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return assignFromElement(field.Elem(), elem)
	}

	if field.Kind() == reflect.Struct && !isXMLNameType(field.Type()) {
		return decodeElementInto(field, elem)
	}

	return assignFromText(field, elementText(elem))
}

func assignFromText(field reflect.Value, value string) error {
	if !field.CanSet() {
		return nil
	}

	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return assignFromText(field.Elem(), value)
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
		return nil
	case reflect.Bool:
		b, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return err
		}
		field.SetBool(b)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(i)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(u)
		return nil
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return err
		}
		field.SetFloat(f)
		return nil
	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.Uint8 {
			field.SetBytes([]byte(value))
			return nil
		}
	}

	if isXMLNameType(field.Type()) {
		nameField := field.FieldByName("Local")
		if nameField.IsValid() && nameField.CanSet() {
			nameField.SetString(value)
		}
		return nil
	}

	return nil
}

func buildFieldBindings(t reflect.Type) []fieldBinding {
	bindings := make([]fieldBinding, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		binding := parseFieldBinding(f)
		binding.index = i
		bindings = append(bindings, binding)
	}
	return bindings
}

func parseFieldBinding(f reflect.StructField) fieldBinding {
	b := fieldBinding{
		name:        f.Name,
		fieldType:   f.Type,
		fieldIsPtr:  f.Type.Kind() == reflect.Pointer,
		fieldExport: f.PkgPath == "",
	}

	if f.Name == "XMLName" && isXMLNameType(derefType(f.Type)) {
		b.isXMLName = true
	}

	tag := f.Tag.Get("xml")
	if tag == "-" {
		b.omit = true
		return b
	}

	if tag == "" {
		b.name = f.Name
		return b
	}

	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])
	if name != "" {
		b.name = name
	}

	if strings.Contains(b.name, ">") {
		segments := strings.Split(b.name, ">")
		b.path = make([]string, 0, len(segments))
		for _, segment := range segments {
			segment = strings.TrimSpace(segment)
			if segment != "" {
				b.path = append(b.path, segment)
			}
		}
	}

	for _, flag := range parts[1:] {
		switch strings.TrimSpace(flag) {
		case "attr":
			b.isAttr = true
		case "chardata":
			b.isCharData = true
		case "innerxml":
			b.isInnerXML = true
		case "any":
			b.isAny = true
		}
	}

	return b
}

func elementText(elem *helium.Element) string {
	if elem == nil {
		return ""
	}
	var b strings.Builder
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		switch v := child.(type) {
		case *helium.Text:
			b.Write(v.Content())
		case *helium.CDATASection:
			b.Write(v.Content())
		}
	}
	return b.String()
}

func innerXML(elem *helium.Element) string {
	if elem == nil {
		return ""
	}
	var b bytes.Buffer
	w := helium.NewWriter(helium.WithNoDecl())
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		_ = w.WriteNode(&b, child)
	}
	return b.String()
}

func childElements(elem *helium.Element) []*helium.Element {
	result := make([]*helium.Element, 0)
	if elem == nil {
		return result
	}
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() == helium.ElementNode {
			result = append(result, child.(*helium.Element))
		}
	}
	return result
}

func firstUnconsumed(children []*helium.Element, consumed map[int]bool) (int, *helium.Element) {
	for i, child := range children {
		if !consumed[i] {
			return i, child
		}
	}
	return -1, nil
}

func findPath(children []*helium.Element, consumed map[int]bool, path []string) (int, *helium.Element) {
	if len(path) == 0 {
		return -1, nil
	}

	for i, child := range children {
		if consumed[i] {
			continue
		}
		if localName(child.Name()) != path[0] {
			continue
		}
		if len(path) == 1 {
			return i, child
		}

		cur := child
		ok := true
		for _, name := range path[1:] {
			next := firstChildByName(cur, name)
			if next == nil {
				ok = false
				break
			}
			cur = next
		}
		if ok {
			return i, cur
		}
	}

	return -1, nil
}

func firstChildByName(elem *helium.Element, name string) *helium.Element {
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce := child.(*helium.Element)
		if localName(ce.Name()) == name {
			return ce
		}
	}
	return nil
}

func localName(name string) string {
	if i := strings.IndexByte(name, ':'); i >= 0 {
		return name[i+1:]
	}
	return name
}

func attrValue(elem *helium.Element, name string) (string, bool) {
	for _, attr := range elem.Attributes() {
		if localName(attr.Name()) == name {
			return attr.Value(), true
		}
	}
	return "", false
}

func setXMLName(field reflect.Value, elem *helium.Element) {
	if !field.CanSet() {
		return
	}
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}
	if !isXMLNameType(field.Type()) {
		return
	}
	local := field.FieldByName("Local")
	if local.IsValid() && local.CanSet() {
		local.SetString(localName(elem.Name()))
	}
	space := field.FieldByName("Space")
	if space.IsValid() && space.CanSet() {
		space.SetString(elem.URI())
	}
}

func isXMLNameType(t reflect.Type) bool {
	if t.Kind() != reflect.Struct {
		return false
	}
	_, okLocal := t.FieldByName("Local")
	_, okSpace := t.FieldByName("Space")
	return okLocal && okSpace
}

func derefType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}
