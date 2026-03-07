package shim

import (
	"bytes"
	"context"
	"encoding"
	stdxml "encoding/xml"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"sync"

	helium "github.com/lestrrat-go/helium"
)

type fieldBinding struct {
	fieldName    string
	index        []int
	rawName      string
	tagPath      string // original tag path string for TagPathError (empty if field name derived)
	name         string
	nameSpace    string
	hasNameSpace bool
	path         []string
	isAttr       bool
	isCharData   bool
	isCData      bool
	isInnerXML   bool
	isComment    bool
	isAny        bool
	isXMLName    bool
	omit         bool
	omitEmpty    bool
	fieldType    reflect.Type
	fieldIsPtr   bool
	fieldExport  bool
}

var fieldBindingCache sync.Map

func Unmarshal(data []byte, v any) error {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Kind() != reflect.Pointer {
		return fmt.Errorf("non-pointer passed to Unmarshal")
	}
	if rv.IsNil() {
		return fmt.Errorf("nil pointer passed to Unmarshal")
	}
	trimmed := trimLeadingSpace(data)
	if len(trimmed) == 0 {
		return io.EOF
	}

	// Strip XML declaration — helium's parser is stricter than stdlib about
	// malformed declarations (e.g. charset= instead of encoding=).
	trimmed = stripXMLDecl(trimmed)

	p := helium.NewParser()
	p.SetMaxDepth(maxParseDepth)
	doc, err := p.Parse(context.Background(), trimmed)
	if err != nil {
		return convertParseError(err)
	}
	root := doc.DocumentElement()
	if root == nil {
		return fmt.Errorf("shim: no document element")
	}

	return decodeElementInto(rv.Elem(), root)
}

func trimLeadingSpace(data []byte) []byte {
	for len(data) > 0 {
		c := data[0]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			break
		}
		data = data[1:]
	}
	return data
}

func stripXMLDecl(data []byte) []byte {
	if len(data) < 5 || string(data[:5]) != "<?xml" {
		return data
	}
	end := bytes.Index(data, []byte("?>"))
	if end < 0 {
		return data
	}
	return trimLeadingSpace(data[end+2:])
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

	handled, err := tryUnmarshalXMLHook(target, elem)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	handled, err = tryUnmarshalTextHook(target, elementText(elem))
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	if target.Kind() != reflect.Struct {
		return assignFromText(target, elementText(elem))
	}

	bindings, err := buildFieldBindings(target.Type())
	if err != nil {
		return err
	}
	if err := validateXMLNameExpectation(bindings, elem); err != nil {
		return err
	}
	children := childElements(elem)
	consumed := make(map[int]bool)
	consumedLeaves := make(map[*helium.Element]bool)
	consumedAttr := make(map[int]bool)

	// Process bindings in two passes: non-any first (to consume specific elements),
	// then any bindings (to pick up remaining unconsumed elements).
	var anyBindings []fieldBinding
	for _, binding := range bindings {
		if binding.omit || !binding.fieldExport {
			continue
		}
		if binding.isAny && !binding.isAttr {
			anyBindings = append(anyBindings, binding)
			continue
		}

		switch {
		case binding.isXMLName:
			// Only set XMLName for the top-level struct's own field.
			// Embedded struct XMLName fields should remain zero.
			if len(binding.index) > 1 {
				continue
			}
			field, ok := fieldByIndexAlloc(target, binding.index)
			if !ok {
				continue
			}
			setXMLName(field, elem)
		case binding.isAttr:
			// Defer any,attr bindings to after specific attrs are consumed
			if binding.isAny {
				continue
			}
			field, ok := fieldByIndexAlloc(target, binding.index)
			if !ok {
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
			field, ok := fieldByIndexAlloc(target, binding.index)
			if !ok {
				continue
			}
			if err := assignFromText(field, elementText(elem)); err != nil {
				return err
			}
		case binding.isCData:
			field, ok := fieldByIndexAlloc(target, binding.index)
			if !ok {
				continue
			}
			if err := assignFromText(field, elementText(elem)); err != nil {
				return err
			}
		case binding.isInnerXML:
			field, ok := fieldByIndexNoAlloc(target, binding.index)
			if !ok {
				continue
			}
			if field.Kind() == reflect.Interface || field.Kind() == reflect.Pointer {
				continue
			}
			ft := field.Type()
			for ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			switch ft.Kind() {
			case reflect.String:
				if err := assignFromText(field, innerXML(elem)); err != nil {
					return err
				}
			case reflect.Slice:
				if ft.Elem().Kind() == reflect.Uint8 {
					if err := assignFromText(field, innerXML(elem)); err != nil {
						return err
					}
				}
			}
		case binding.isComment:
			field, ok := fieldByIndexNoAlloc(target, binding.index)
			if !ok {
				continue
			}
			if field.Kind() == reflect.Interface || field.Kind() == reflect.Pointer {
				continue
			}
			commentText := elementComment(elem)
			if commentText != "" || (field.Kind() != reflect.Slice || field.Type().Elem().Kind() != reflect.Uint8) {
				if err := assignFromText(field, commentText); err != nil {
					return err
				}
			}
		default:
			// Element binding: use non-allocating accessor first to check if
			// matching children exist, to avoid allocating nil embedded pointers
			// when no data is present.
			isPath := len(binding.path) > 1

			if isPath {
				// Multi-segment path (e.g., "A>B"): find leaves without
				// consuming wrapper elements so multiple bindings can share
				// the same wrapper. Also mark wrappers in consumed so the
				// any-field pass skips them.
				wrapperIdx, leaf := findPathLeaf(children, binding.path, binding.nameSpace, binding.hasNameSpace, consumedLeaves)
				if leaf == nil {
					continue
				}

				field, ok := fieldByIndexAlloc(target, binding.index)
				if !ok {
					continue
				}

				ft := field.Type()
				for ft.Kind() == reflect.Pointer {
					ft = ft.Elem()
				}
				if isXMLNameType(ft) {
					consumedLeaves[leaf] = true
					consumed[wrapperIdx] = true
					setXMLName(field, leaf)
					continue
				}

				if field.Kind() == reflect.Interface {
					if field.IsNil() {
						continue
					}
					field = field.Elem()
				}

				if field.Kind() == reflect.Slice && field.Type().Elem().Kind() != reflect.Uint8 {
					for wi, leaf := findPathLeaf(children, binding.path, binding.nameSpace, binding.hasNameSpace, consumedLeaves); leaf != nil; wi, leaf = findPathLeaf(children, binding.path, binding.nameSpace, binding.hasNameSpace, consumedLeaves) {
						consumedLeaves[leaf] = true
						consumed[wi] = true
						item := reflect.New(field.Type().Elem()).Elem()
						if err := assignFromElement(item, leaf); err != nil {
							return err
						}
						field.Set(reflect.Append(field, item))
					}
					continue
				}

				// Scalar field: consume all matches, last one wins (stdlib behavior).
				for wi, leaf := findPathLeaf(children, binding.path, binding.nameSpace, binding.hasNameSpace, consumedLeaves); leaf != nil; wi, leaf = findPathLeaf(children, binding.path, binding.nameSpace, binding.hasNameSpace, consumedLeaves) {
					consumedLeaves[leaf] = true
					consumed[wi] = true
					if err := assignFromElement(field, leaf); err != nil {
						return err
					}
				}
			} else {
				// Single-segment (direct child match): use consumed set on children.
				// Use rawName to preserve namespace info for matchElementByTag.
				path := []string{binding.rawName}

				_, matched := findPath(children, consumed, path)
				if matched == nil {
					continue
				}

				field, ok := fieldByIndexAlloc(target, binding.index)
				if !ok {
					continue
				}

				ft := field.Type()
				for ft.Kind() == reflect.Pointer {
					ft = ft.Elem()
				}
				if isXMLNameType(ft) {
					idx, m := findPath(children, consumed, path)
					if m == nil {
						continue
					}
					consumed[idx] = true
					setXMLName(field, m)
					continue
				}

				if field.Kind() == reflect.Interface {
					if field.IsNil() {
						continue
					}
					field = field.Elem()
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

				// Scalar field: consume all matches, last one wins (stdlib behavior).
				for {
					idx, matched2 := findPath(children, consumed, path)
					if matched2 == nil {
						break
					}
					consumed[idx] = true
					if err := assignFromElement(field, matched2); err != nil {
						return err
					}
				}
			}
		}
	}

	// Second pass: process any-tagged bindings on remaining unconsumed elements
	for _, binding := range anyBindings {
		field, ok := fieldByIndexAlloc(target, binding.index)
		if !ok {
			continue
		}
		if field.Kind() == reflect.Interface {
			continue
		}
		if field.Kind() == reflect.Slice && field.Type().Elem().Kind() != reflect.Uint8 {
			for idx, anyElem := firstUnconsumed(children, consumed); anyElem != nil; idx, anyElem = firstUnconsumed(children, consumed) {
				consumed[idx] = true
				item := reflect.New(field.Type().Elem()).Elem()
				if err := assignFromElement(item, anyElem); err != nil {
					return err
				}
				field.Set(reflect.Append(field, item))
			}
			continue
		}

		for idx, anyElem := firstUnconsumed(children, consumed); anyElem != nil; idx, anyElem = firstUnconsumed(children, consumed) {
			consumed[idx] = true
			if err := assignFromElement(field, anyElem); err != nil {
				return err
			}
		}
	}

	// Third pass: process any,attr bindings on remaining unconsumed attributes
	for _, binding := range bindings {
		if !binding.isAttr || !binding.isAny || binding.omit || !binding.fieldExport {
			continue
		}
		field, ok := fieldByIndexAlloc(target, binding.index)
		if !ok {
			continue
		}
		// Handle []xml.Attr field
		if field.Type() == attrSliceType {
			for i, attr := range elem.Attributes() {
				if consumedAttr[i] {
					continue
				}
				a := Attr{
					Name:  Name{Space: attr.URI(), Local: localName(attr.Name())},
					Value: attr.Value(),
				}
				field.Set(reflect.Append(field, reflect.ValueOf(a)))
			}
			continue
		}
		for i, attr := range elem.Attributes() {
			if consumedAttr[i] {
				continue
			}
			consumedAttr[i] = true
			if err := assignFromAttr(field, attr); err != nil {
				return err
			}
		}
	}

	return nil
}

func assignFromElement(field reflect.Value, elem *helium.Element) error {
	if field.Kind() == reflect.Pointer {
		if !field.IsNil() {
			return assignFromElement(field.Elem(), elem)
		}
		if !field.CanSet() {
			return nil
		}
		field.Set(reflect.New(field.Type().Elem()))
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
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			field.SetBool(false)
			return nil
		}
		b, err := strconv.ParseBool(trimmed)
		if err != nil {
			return err
		}
		field.SetBool(b)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			field.SetInt(0)
			return nil
		}
		i, err := strconv.ParseInt(trimmed, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(i)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			field.SetUint(0)
			return nil
		}
		u, err := strconv.ParseUint(trimmed, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(u)
		return nil
	case reflect.Float32, reflect.Float64:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			field.SetFloat(0)
			return nil
		}
		f, err := strconv.ParseFloat(trimmed, field.Type().Bits())
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
	switch t.Kind() {
	case reflect.Interface:
		return UnmarshalError("cannot unmarshal into " + t.String())
	default:
		return UnmarshalError("unknown type " + t.String())
	}
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
	for _, candidate := range interfaceCandidates(field) {
		hook, ok := candidate.(Unmarshaler)
		if !ok {
			continue
		}

		tr := &elementTokenReader{elem: elem}
		dec := stdxml.NewTokenDecoder(tr)
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

// elementTokenReader walks a helium DOM subtree and emits stdxml.Token values.
type elementTokenReader struct {
	elem    *helium.Element
	tokens  []stdxml.Token
	pos     int
	built   bool
}

func (r *elementTokenReader) Token() (stdxml.Token, error) {
	if !r.built {
		r.buildTokens()
		r.built = true
	}
	if r.pos >= len(r.tokens) {
		return nil, io.EOF
	}
	tok := r.tokens[r.pos]
	r.pos++
	return tok, nil
}

func (r *elementTokenReader) buildTokens() {
	r.tokens = make([]stdxml.Token, 0, 8)
	r.emitElement(r.elem)
}

func (r *elementTokenReader) emitElement(elem *helium.Element) {
	se := stdxml.StartElement{
		Name: stdxml.Name{Space: elem.URI(), Local: localName(elem.Name())},
	}
	for _, attr := range elem.Attributes() {
		se.Attr = append(se.Attr, toStdAttr(attr))
	}
	r.tokens = append(r.tokens, se)

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		switch v := child.(type) {
		case *helium.Element:
			r.emitElement(v)
		case *helium.Text:
			r.tokens = append(r.tokens, stdxml.CharData(v.Content()))
		case *helium.CDATASection:
			r.tokens = append(r.tokens, stdxml.CharData(v.Content()))
		case *helium.Comment:
			r.tokens = append(r.tokens, stdxml.Comment(v.Content()))
		case *helium.ProcessingInstruction:
			r.tokens = append(r.tokens, stdxml.ProcInst{
				Target: v.Name(),
				Inst:   v.Content(),
			})
		}
	}

	r.tokens = append(r.tokens, stdxml.EndElement{
		Name: stdxml.Name{Space: elem.URI(), Local: localName(elem.Name())},
	})
}

func buildFieldBindings(t reflect.Type) ([]fieldBinding, error) {
	if cached, ok := fieldBindingCache.Load(t); ok {
		return cached.([]fieldBinding), nil
	}

	bindings := make([]fieldBinding, 0, t.NumField())
	collectFieldBindings(t, nil, &bindings, map[reflect.Type]bool{})
	for _, b := range bindings {
		if b.isXMLName && b.isAttr {
			return nil, fmt.Errorf("xml: invalid tag in field %s of type %s: \"xml:%s\"", b.fieldName, t, b.rawName+",attr")
		}
	}
	if err := validateTagPathConflicts(t, bindings); err != nil {
		return nil, err
	}
	bindings = resolveBindingConflicts(bindings)

	fieldBindingCache.Store(t, bindings)
	return bindings, nil
}

func validateTagPathConflicts(t reflect.Type, bindings []fieldBinding) error {
	paths := make([]fieldBinding, 0, len(bindings))

	for _, binding := range bindings {
		if binding.omit || !binding.fieldExport {
			continue
		}
		if binding.isAttr || binding.isCharData || binding.isCData || binding.isInnerXML || binding.isComment || binding.isAny || binding.isXMLName {
			continue
		}
		path := binding.path
		if len(path) == 0 {
			path = []string{binding.rawName}
		}

		for _, prev := range paths {
			prevPath := prev.path
			if len(prevPath) == 0 {
				prevPath = []string{prev.rawName}
			}
			if len(prev.index) != len(binding.index) {
				continue
			}
			// Different namespaces never conflict (matches stdlib addFieldInfo).
			if prev.hasNameSpace && binding.hasNameSpace && prev.nameSpace != binding.nameSpace {
				continue
			}
			if pathConflicts(prevPath, path) {
				return &TagPathError{
					Struct: t,
					Field1: prev.fieldName,
					Tag1:   prev.tagPath,
					Field2: binding.fieldName,
					Tag2:   binding.tagPath,
				}
			}
		}

		paths = append(paths, binding)
	}

	return nil
}

func pathConflicts(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}

	n := len(a)
	if len(b) < n {
		n = len(b)
	}

	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
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

	// Second pass: resolve cross-depth shadowing for element bindings.
	// A plain field "FieldA" at depth 1 shadows path fields "FieldA>A1" at depth 2
	// because they share the same top-level XML element name.
	type elemInfo struct {
		topName string
		depth   int
		keptIdx int
	}
	var elems []elemInfo
	for i, b := range kept {
		if b.omit || !b.fieldExport || b.isAttr || b.isCharData || b.isCData || b.isInnerXML || b.isComment || b.isAny || b.isXMLName {
			continue
		}
		topName := b.rawName
		if len(b.path) > 0 {
			topName = b.path[0]
		}
		elems = append(elems, elemInfo{topName: topName, depth: len(b.index), keptIdx: i})
	}

	shadowed := make(map[int]bool)
	for i, a := range elems {
		if shadowed[a.keptIdx] {
			continue
		}
		for j, b := range elems {
			if i == j || shadowed[b.keptIdx] {
				continue
			}
			if a.topName != b.topName {
				continue
			}
			// Same top-level name at different depths: shallower wins
			if a.depth < b.depth {
				shadowed[b.keptIdx] = true
			} else if b.depth < a.depth {
				shadowed[a.keptIdx] = true
				break
			}
		}
	}

	if len(shadowed) > 0 {
		filtered := make([]fieldBinding, 0, len(kept))
		for i, b := range kept {
			if !shadowed[i] {
				filtered = append(filtered, b)
			}
		}
		kept = filtered
	}

	return kept
}

func validateXMLNameExpectation(bindings []fieldBinding, elem *helium.Element) error {
	for _, binding := range bindings {
		if !binding.isXMLName || binding.omit {
			continue
		}
		// Only validate against top-level XMLName, not embedded structs' XMLName.
		if len(binding.index) > 1 {
			continue
		}

		spec := strings.TrimSpace(binding.rawName)
		if spec == "" || spec == "XMLName" {
			return nil
		}

		space, local, hasSpace := parseTagNameSpec(spec)
		if local == "" {
			return nil
		}

		if localName(elem.Name()) != local {
			return UnmarshalError(fmt.Sprintf("expected element type <%s> but have <%s>", local, localName(elem.Name())))
		}
		if hasSpace && elem.URI() != space {
			return UnmarshalError(fmt.Sprintf("expected element <%s> in name space %s but have %s", local, space, elem.URI()))
		}
		return nil
	}

	return nil
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

	// Include namespace to distinguish fields with same local path
	// but different namespaces (e.g., "space x>b" vs "space1 x>b").
	if binding.hasNameSpace {
		return kind + "|" + binding.nameSpace + " " + name
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

// fieldByIndexNoAlloc traverses the struct field index chain without
// allocating nil pointers. Returns false if a nil pointer is encountered.
func fieldByIndexNoAlloc(v reflect.Value, index []int) (reflect.Value, bool) {
	cur := v
	for _, i := range index {
		if cur.Kind() == reflect.Pointer {
			if cur.IsNil() {
				return reflect.Value{}, false
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
		fieldName:   f.Name,
		rawName:     f.Name,
		name:        f.Name,
		fieldType:   f.Type,
		fieldIsPtr:  f.Type.Kind() == reflect.Pointer,
		fieldExport: f.PkgPath == "",
	}

	if f.Name == "XMLName" {
		b.isXMLName = true
	}

	tag, hasTag := f.Tag.Lookup("xml")
	if tag == "-" {
		b.omit = true
		return b
	}

	if tag == "" {
		if hasTag && b.isXMLName {
			// Explicit xml:"" on XMLName — empty namespace and name
			b.name = ""
			b.rawName = ""
			return b
		}
		// If the field type is a struct with an XMLName tag, use that tag
		// name for element matching (stdlib precedence: XMLName tag > field name).
		if xmlNameTag := structXMLNameTag(derefType(f.Type)); xmlNameTag != "" {
			b.rawName = xmlNameTag
			b.nameSpace, b.name, b.hasNameSpace = parseTagNameSpec(xmlNameTag)
		} else {
			b.name = f.Name
		}
		return b
	}

	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])
	if name != "" {
		b.rawName = name
		b.tagPath = name

		// Extract namespace first (e.g., "space x>b" → ns="space", local="x>b")
		var localPart string
		b.nameSpace, localPart, b.hasNameSpace = parseTagNameSpec(name)

		// Split by ">" for path tags (e.g., "A>B>C" or ">item" shorthand)
		if strings.Contains(localPart, ">") {
			segments := strings.Split(localPart, ">")
			if segments[0] == "" {
				segments[0] = f.Name // ">item" shorthand: wrapper = field name
			}
			b.path = segments
			b.name = segments[len(segments)-1]
		} else {
			b.name = localPart
		}
	}

	for _, flag := range parts[1:] {
		switch strings.TrimSpace(flag) {
		case "attr":
			b.isAttr = true
		case "chardata":
			b.isCharData = true
		case "cdata":
			b.isCData = true
		case "innerxml":
			b.isInnerXML = true
		case "comment":
			b.isComment = true
		case "any":
			b.isAny = true
		case "omitempty":
			b.omitEmpty = true
		}
	}

	return b
}

// structXMLNameTag returns the XMLName field's tag name for a struct type,
// or "" if the type is not a struct or has no XMLName tag.
func structXMLNameTag(t reflect.Type) string {
	if t.Kind() != reflect.Struct {
		return ""
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Name != "XMLName" {
			continue
		}
		if !isXMLNameType(derefType(f.Type)) {
			continue
		}
		tag := f.Tag.Get("xml")
		if tag == "" || tag == "-" {
			return ""
		}
		parts := strings.Split(tag, ",")
		name := strings.TrimSpace(parts[0])
		if name != "" && name != "XMLName" {
			return name
		}
		return ""
	}
	return ""
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

func elementComment(elem *helium.Element) string {
	if elem == nil {
		return ""
	}
	var b strings.Builder
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if comment, ok := child.(*helium.Comment); ok {
			b.Write(comment.Content())
		}
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

// findPathLeaf walks a multi-segment path (e.g., ["A","B","C"]) through the
// children without consuming wrapper elements. It returns the first unconsumed
// leaf element matching the full path and the index of the direct child (wrapper)
// that was traversed. Returns (-1, nil) if no match found.
// The ns/hasNS parameters apply to the leaf element only (matching stdlib behavior).
func findPathLeaf(children []*helium.Element, path []string, ns string, hasNS bool, consumedLeaves map[*helium.Element]bool) (int, *helium.Element) {
	if len(path) == 0 {
		return -1, nil
	}

	wrapperName := path[0]
	for i, child := range children {
		if localName(child.Name()) != wrapperName {
			continue
		}
		if len(path) == 1 {
			// This is the leaf level
			if consumedLeaves[child] {
				continue
			}
			if hasNS && child.URI() != ns {
				continue
			}
			return i, child
		}
		// Descend into wrapper — find leaf in grandchildren
		grandchildren := childElements(child)
		_, leaf := findPathLeafInner(grandchildren, path[1:], ns, hasNS, consumedLeaves)
		if leaf != nil {
			return i, leaf
		}
	}

	return -1, nil
}

// findPathLeafInner is the recursive helper for findPathLeaf.
// It doesn't need to track the wrapper index (only the top level does).
func findPathLeafInner(children []*helium.Element, path []string, ns string, hasNS bool, consumedLeaves map[*helium.Element]bool) (int, *helium.Element) {
	if len(path) == 0 {
		return -1, nil
	}

	name := path[0]
	for i, child := range children {
		if localName(child.Name()) != name {
			continue
		}
		if len(path) == 1 {
			if consumedLeaves[child] {
				continue
			}
			if hasNS && child.URI() != ns {
				continue
			}
			return i, child
		}
		grandchildren := childElements(child)
		_, leaf := findPathLeafInner(grandchildren, path[1:], ns, hasNS, consumedLeaves)
		if leaf != nil {
			return i, leaf
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

var xmlNameType = reflect.TypeOf(stdxml.Name{})
var attrSliceType = reflect.TypeOf([]stdxml.Attr{})

func isXMLNameType(t reflect.Type) bool {
	return t == xmlNameType
}

func derefType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}
