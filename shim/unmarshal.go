package shim

import (
	"bytes"
	"context"
	"encoding"
	stdxml "encoding/xml"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	helium "github.com/lestrrat-go/helium"
)

type fieldBinding struct {
	index        []int
	rawName      string
	name         string
	nameSpace    string
	hasNameSpace bool
	path         []string
	isAttr       bool
	isCharData   bool
	isInnerXML   bool
	isAny        bool
	isXMLName    bool
	omit         bool
	fieldType    reflect.Type
	fieldIsPtr   bool
	fieldExport  bool
}

var fieldBindingCache sync.Map

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
	consumedAttr := make(map[int]bool)

	for _, binding := range bindings {
		if binding.omit || !binding.fieldExport {
			continue
		}

		field, ok := fieldByIndexAlloc(target, binding.index)
		if !ok {
			continue
		}

		switch {
		case binding.isXMLName:
			setXMLName(field, elem)
		case binding.isAttr:
			if binding.isAny {
				for i, attr := range elem.Attributes() {
					if consumedAttr[i] {
						continue
					}
					consumedAttr[i] = true
					if err := assignFromAttr(field, attr); err != nil {
						return err
					}
				}
				continue
			}

			idx, attr, ok := lookupAttr(elem, binding.name, binding.nameSpace, binding.hasNameSpace)
			if ok {
				consumedAttr[idx] = true
				if err := assignFromAttr(field, attr); err != nil {
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
				path = []string{binding.rawName}
			}

			if field.Kind() == reflect.Slice && field.Type().Elem().Kind() != reflect.Uint8 {
				for {
					idx, matched := findPath(children, consumed, path)
					if matched == nil {
						break
					}
					consumed[idx] = true

					item := reflect.New(field.Type().Elem()).Elem()
					if err := assignFromElement(item, matched); err != nil {
						return err
					}
					field.Set(reflect.Append(field, item))
				}
				continue
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

	handled, err := tryUnmarshalXMLHook(field, elem)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	if field.Kind() == reflect.Struct && !isXMLNameType(field.Type()) {
		return decodeElementInto(field, elem)
	}

	return assignFromText(field, elementText(elem))
}

func assignFromAttr(field reflect.Value, attr *helium.Attribute) error {
	if !field.CanSet() {
		return nil
	}

	if field.Kind() == reflect.Slice && field.Type().Elem() == reflect.TypeOf(stdxml.Attr{}) {
		field.Set(reflect.Append(field, reflect.ValueOf(toStdAttr(attr))))
		return nil
	}

	if field.Type() == reflect.TypeOf(stdxml.Attr{}) {
		field.Set(reflect.ValueOf(toStdAttr(attr)))
		return nil
	}

	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return assignFromAttr(field.Elem(), attr)
	}

	handled, err := tryUnmarshalXMLAttrHook(field, attr)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	return assignFromText(field, attr.Value())
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

	handled, err := tryUnmarshalTextHook(field, value)
	if err != nil {
		return err
	}
	if handled {
		return nil
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
		i, err := strconv.ParseInt(strings.TrimSpace(value), 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(i)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(strings.TrimSpace(value), 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(u)
		return nil
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(strings.TrimSpace(value), field.Type().Bits())
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
		return unsupportedUnmarshalTypeError(field.Type())
	case reflect.Map, reflect.Interface, reflect.Func, reflect.Chan:
		return unsupportedUnmarshalTypeError(field.Type())
	}

	if isXMLNameType(field.Type()) {
		nameField := field.FieldByName("Local")
		if nameField.IsValid() && nameField.CanSet() {
			nameField.SetString(value)
		}
		return nil
	}

	return unsupportedUnmarshalTypeError(field.Type())
}

func unsupportedUnmarshalTypeError(t reflect.Type) error {
	return UnmarshalError("cannot unmarshal into " + t.String())
}

func interfaceCandidates(v reflect.Value) []any {
	candidates := make([]any, 0, 2)
	if v.IsValid() && v.CanInterface() {
		candidates = append(candidates, v.Interface())
	}
	if v.IsValid() && v.CanAddr() && v.Addr().CanInterface() {
		candidates = append(candidates, v.Addr().Interface())
	}
	return candidates
}

func tryUnmarshalTextHook(field reflect.Value, value string) (bool, error) {
	for _, candidate := range interfaceCandidates(field) {
		if hook, ok := candidate.(encoding.TextUnmarshaler); ok {
			if err := hook.UnmarshalText([]byte(value)); err != nil {
				return true, err
			}
			return true, nil
		}
	}
	return false, nil
}

func tryUnmarshalXMLAttrHook(field reflect.Value, attr *helium.Attribute) (bool, error) {
	xa := toStdAttr(attr)

	for _, candidate := range interfaceCandidates(field) {
		if hook, ok := candidate.(UnmarshalerAttr); ok {
			if err := hook.UnmarshalXMLAttr(xa); err != nil {
				return true, err
			}
			return true, nil
		}
	}

	return false, nil
}

func tryUnmarshalXMLHook(field reflect.Value, elem *helium.Element) (bool, error) {
	xmlBytes, err := outerXML(elem)
	if err != nil {
		return false, err
	}

	for _, candidate := range interfaceCandidates(field) {
		hook, ok := candidate.(Unmarshaler)
		if !ok {
			continue
		}

		dec := stdxml.NewDecoder(bytes.NewReader(xmlBytes))
		tok, err := dec.Token()
		if err != nil {
			return true, err
		}
		start, ok := tok.(stdxml.StartElement)
		if !ok {
			return true, fmt.Errorf("shim: expected start element token, got %T", tok)
		}
		if err := hook.UnmarshalXML(dec, start); err != nil {
			return true, err
		}
		return true, nil
	}

	return false, nil
}

func buildFieldBindings(t reflect.Type) []fieldBinding {
	if cached, ok := fieldBindingCache.Load(t); ok {
		return cached.([]fieldBinding)
	}

	bindings := make([]fieldBinding, 0, t.NumField())
	collectFieldBindings(t, nil, &bindings, map[reflect.Type]bool{})
	bindings = resolveBindingConflicts(bindings)

	fieldBindingCache.Store(t, bindings)
	return bindings
}

func resolveBindingConflicts(bindings []fieldBinding) []fieldBinding {
	kept := make([]fieldBinding, 0, len(bindings))
	seen := make(map[string]int, len(bindings))

	for _, binding := range bindings {
		if binding.omit || !binding.fieldExport {
			kept = append(kept, binding)
			continue
		}

		key := bindingKey(binding)
		if prevIdx, ok := seen[key]; ok {
			if preferBinding(binding, kept[prevIdx]) {
				kept[prevIdx] = binding
			}
			continue
		}

		seen[key] = len(kept)
		kept = append(kept, binding)
	}

	return kept
}

func bindingKey(binding fieldBinding) string {
	kind := "elem"
	if binding.isAttr {
		kind = "attr"
	}
	if binding.isCharData {
		kind = "chardata"
	}
	if binding.isInnerXML {
		kind = "innerxml"
	}
	if binding.isAny {
		kind = "any"
	}

	name := binding.rawName
	if len(binding.path) > 0 {
		name = strings.Join(binding.path, ">")
	}

	return kind + "|" + name
}

func preferBinding(candidate, current fieldBinding) bool {
	if len(candidate.index) != len(current.index) {
		return len(candidate.index) < len(current.index)
	}
	return false
}

func collectFieldBindings(t reflect.Type, parentIndex []int, out *[]fieldBinding, seen map[reflect.Type]bool) {
	if seen[t] {
		return
	}
	seen[t] = true
	defer func() {
		delete(seen, t)
	}()

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		idx := append(append([]int(nil), parentIndex...), i)

		if shouldFlattenEmbeddedField(f) {
			embeddedType := derefType(f.Type)
			if !seen[embeddedType] {
				collectFieldBindings(embeddedType, idx, out, seen)
				continue
			}
		}

		binding := parseFieldBinding(f)
		binding.index = idx
		*out = append(*out, binding)
	}
}

func shouldFlattenEmbeddedField(f reflect.StructField) bool {
	if !f.Anonymous {
		return false
	}
	if f.PkgPath != "" {
		return false
	}
	tag := f.Tag.Get("xml")
	if tag == "-" {
		return false
	}
	if tag != "" {
		return false
	}
	ft := derefType(f.Type)
	if ft.Kind() != reflect.Struct {
		return false
	}
	if isXMLNameType(ft) {
		return false
	}
	return true
}

func fieldByIndexAlloc(v reflect.Value, index []int) (reflect.Value, bool) {
	cur := v
	for _, i := range index {
		if cur.Kind() == reflect.Pointer {
			if cur.IsNil() {
				if !cur.CanSet() {
					return reflect.Value{}, false
				}
				cur.Set(reflect.New(cur.Type().Elem()))
			}
			cur = cur.Elem()
		}
		if cur.Kind() != reflect.Struct {
			return reflect.Value{}, false
		}
		cur = cur.Field(i)
	}
	if !cur.IsValid() {
		return reflect.Value{}, false
	}
	return cur, true
}

func parseFieldBinding(f reflect.StructField) fieldBinding {
	b := fieldBinding{
		rawName:     f.Name,
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
		b.rawName = name
		b.nameSpace, b.name, b.hasNameSpace = parseTagNameSpec(name)
	}

	if strings.Contains(b.rawName, ">") {
		segments := strings.Split(b.rawName, ">")
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

func outerXML(elem *helium.Element) ([]byte, error) {
	if elem == nil {
		return nil, nil
	}
	var b bytes.Buffer
	w := helium.NewWriter(helium.WithNoDecl())
	if err := w.WriteNode(&b, elem); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
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
		if !matchElementByTag(child, path[0]) {
			continue
		}
		if len(path) == 1 {
			return i, child
		}

		cur := child
		ok := true
		for _, name := range path[1:] {
			next := firstChildByTag(cur, name)
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

func firstChildByTag(elem *helium.Element, tag string) *helium.Element {
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce := child.(*helium.Element)
		if matchElementByTag(ce, tag) {
			return ce
		}
	}
	return nil
}

func matchElementByTag(elem *helium.Element, tag string) bool {
	space, local, hasSpace := parseTagNameSpec(tag)
	if localName(elem.Name()) != local {
		return false
	}
	if hasSpace {
		return elem.URI() == space
	}
	return true
}

func parseTagNameSpec(tag string) (space, local string, hasSpace bool) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return "", "", false
	}
	parts := strings.SplitN(tag, " ", 2)
	if len(parts) == 2 {
		space = strings.TrimSpace(parts[0])
		local = strings.TrimSpace(parts[1])
		if local != "" {
			return space, local, true
		}
	}
	return "", tag, false
}

func localName(name string) string {
	if i := strings.IndexByte(name, ':'); i >= 0 {
		return name[i+1:]
	}
	return name
}

func lookupAttr(elem *helium.Element, name, space string, hasSpace bool) (int, *helium.Attribute, bool) {
	for i, attr := range elem.Attributes() {
		if localName(attr.Name()) != name {
			continue
		}
		if hasSpace && attr.URI() != space {
			continue
		}
		return i, attr, true
	}
	return -1, nil, false
}

func toStdAttr(attr *helium.Attribute) stdxml.Attr {
	return stdxml.Attr{
		Name:  stdxml.Name{Space: attr.URI(), Local: localName(attr.Name())},
		Value: attr.Value(),
	}
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
