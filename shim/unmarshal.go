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

const xmlNameField = "XMLName"

// heliumNoEncoding is the sentinel helium's [helium.Document.Encoding] returns
// for a document whose declaration named no encoding. It must be treated as
// "no encoding declared", not as an encoding to honor.
const heliumNoEncoding = "utf8"

type fieldBinding struct {
	fieldName    string
	index        []int
	rawName      string
	tagPath      string // original tag path string for TagPathError (empty if field name derived)
	name         string
	nameSpace    string
	hasNameSpace bool
	// matchLocal, matchSpace, and matchHasSpace are parseTagNameSpec(rawName),
	// precomputed once in parseFieldBinding so the single-segment element
	// scan (findFrom) never re-parses rawName per candidate child. Derived
	// from rawName rather than name/nameSpace, which are populated
	// inconsistently across the tag-parsing branches above.
	matchLocal    string
	matchSpace    string
	matchHasSpace bool
	path          []string
	isAttr        bool
	isCharData    bool
	isCData       bool
	isInnerXML    bool
	isComment     bool
	isAny         bool
	isXMLName     bool
	omit          bool
	omitEmpty     bool
	fieldType     reflect.Type
	fieldIsPtr    bool
	fieldExport   bool
}

var fieldBindingCache sync.Map

// validateUnmarshalTarget checks that v is a non-nil pointer, matching
// encoding/xml's Unmarshal/Decode/DecodeElement error behavior.
func validateUnmarshalTarget(v any) (reflect.Value, error) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Kind() != reflect.Pointer {
		return reflect.Value{}, fmt.Errorf("non-pointer passed to Unmarshal")
	}
	if rv.IsNil() {
		return reflect.Value{}, fmt.Errorf("nil pointer passed to Unmarshal")
	}
	return rv.Elem(), nil
}

func Unmarshal(data []byte, v any) error {
	rv, err := validateUnmarshalTarget(v)
	if err != nil {
		return err
	}
	// A document that is empty or only whitespace has no root element; match
	// encoding/xml and report io.EOF. The trim serves only this check — helium
	// itself sees the original, untrimmed bytes below.
	if len(trimLeadingSpace(data)) == 0 {
		return io.EOF
	}

	// helium sees the whole document verbatim, including any leading whitespace
	// and any XML declaration, and is the single authority for the XMLDecl
	// grammar (XML 1.0 §2.8), the version rule (1.0 and 1.1 are supported; a
	// version outside the 1.x family is rejected), and declaration placement.
	// Its parse verdict is the answer: a declaration it rejects — a charset=
	// pseudo-attribute, a missing/empty version, an empty encoding,
	// pseudo-attributes out of order, an unsupported version, or a misplaced,
	// reserved-cased, or preceded-by-whitespace declaration (a declaration is
	// legal only at document position 0, so any whitespace ahead of it makes it
	// misplaced) — surfaces as a parse error. The bytes are NOT trimmed before
	// this parse: trimming would hide leading whitespace from helium and make
	// shim accept a misplaced declaration helium rejects. encoding/xml accepts
	// many of these; this shim is backed by a spec-conforming parser and does
	// not.
	p := helium.NewParser().MaxDepth(maxParseDepth)
	doc, err := p.Parse(context.Background(), data)
	if err != nil {
		return convertParseError(err)
	}

	// The only declaration rule left shim-side is the encoding one: Unmarshal has
	// no CharsetReader, so a non-UTF-8 encoding cannot be honored.
	if err := validateXMLDeclFields(doc); err != nil {
		return err
	}
	root := doc.DocumentElement()
	if root == nil {
		return fmt.Errorf("shim: no document element")
	}

	return decodeElementInto(rv, root)
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

// validateXMLDeclFields applies the shim's encoding rule to a parsed document's
// declared encoding: [Unmarshal] has no CharsetReader, so an explicitly declared
// non-UTF-8 encoding cannot be honored and is rejected. The version and grammar
// were already judged by helium's parse. doc.Encoding() returns the
// heliumNoEncoding sentinel when no encoding was declared; that must be treated
// as absent, so only an explicitly declared non-UTF-8 encoding is rejected.
func validateXMLDeclFields(doc *helium.Document) error {
	enc := doc.Encoding()
	if !encodingNeedsCharsetReader(enc) {
		return nil
	}
	return errCharsetReaderNil(enc)
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
	consumed := make([]bool, len(children))
	// consumedLeaves stays nil until the first multi-segment path binding
	// needs it — reads of a nil map are legal, and a childless or path-free
	// element never allocates it.
	var consumedLeaves map[*helium.Element]struct{}
	consumedAttr := make(map[int]struct{})

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
				consumedAttr[idx] = struct{}{}
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
				// Multi-segment path (e.g., "A>B"): a resumable DFS scanner
				// walks the path without consuming wrapper elements, so
				// multiple bindings can share the same wrapper. It also
				// marks the top-level wrapper in consumed so the any-field
				// pass skips it. One scanner per binding: created just
				// before the first probe, discarded when this binding's
				// drain finishes — nothing outside this branch keeps it
				// alive, and nothing during the drain un-marks a consumed
				// leaf or mutates the tree, so resuming it never misses a
				// leaf a from-scratch walk would have found.
				if consumedLeaves == nil {
					consumedLeaves = make(map[*helium.Element]struct{})
				}
				scanner := newPathScanner(children, binding.path, binding.nameSpace, binding.hasNameSpace)
				wrapperIdx, leaf := scanner.next(consumedLeaves)
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
					consumedLeaves[leaf] = struct{}{}
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
					for leaf != nil {
						consumedLeaves[leaf] = struct{}{}
						consumed[wrapperIdx] = true
						item := reflect.New(field.Type().Elem()).Elem()
						if err := assignFromElement(item, leaf); err != nil {
							return err
						}
						field.Set(reflect.Append(field, item))

						wrapperIdx, leaf = scanner.next(consumedLeaves)
					}
					continue
				}

				// Scalar field: consume all matches, last one wins (stdlib behavior).
				for leaf != nil {
					consumedLeaves[leaf] = struct{}{}
					consumed[wrapperIdx] = true
					if err := assignFromElement(field, leaf); err != nil {
						return err
					}
					wrapperIdx, leaf = scanner.next(consumedLeaves)
				}
			} else {
				// Single-segment (direct child match): scan with a
				// per-binding cursor that only ever moves forward. Every
				// match this binding claims is drained before the next
				// binding runs, so resuming the scan at the last claimed
				// index plus one sees exactly what a from-0 rescan would.
				// The consumed check inside findFrom still applies, since a
				// child at or after the cursor may already have been
				// claimed by an earlier binding.
				cur := 0
				idx, matched := findFrom(children, consumed, cur, binding.matchLocal, binding.matchSpace, binding.matchHasSpace)
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
					consumed[idx] = true
					setXMLName(field, matched)
					continue
				}

				if field.Kind() == reflect.Interface {
					if field.IsNil() {
						continue
					}
					field = field.Elem()
				}

				if field.Kind() == reflect.Slice && field.Type().Elem().Kind() != reflect.Uint8 {
					for matched != nil {
						consumed[idx] = true
						cur = idx + 1

						item := reflect.New(field.Type().Elem()).Elem()
						if err := assignFromElement(item, matched); err != nil {
							return err
						}
						field.Set(reflect.Append(field, item))

						idx, matched = findFrom(children, consumed, cur, binding.matchLocal, binding.matchSpace, binding.matchHasSpace)
					}
					continue
				}

				// Scalar field: consume all matches, last one wins (stdlib behavior).
				for matched != nil {
					consumed[idx] = true
					cur = idx + 1
					if err := assignFromElement(field, matched); err != nil {
						return err
					}
					idx, matched = findFrom(children, consumed, cur, binding.matchLocal, binding.matchSpace, binding.matchHasSpace)
				}
			}
		}
	}

	// Second pass: process any-tagged bindings on remaining unconsumed
	// elements. A single cursor spans the whole pass, not just one binding:
	// only this pass writes consumed at this point, so the next unconsumed
	// index is non-decreasing across every any-binding's drain, exactly as
	// it is within one binding's own drain.
	anyCursor := 0
	for _, binding := range anyBindings {
		field, ok := fieldByIndexAlloc(target, binding.index)
		if !ok {
			continue
		}
		if field.Kind() == reflect.Interface {
			continue
		}
		if field.Kind() == reflect.Slice && field.Type().Elem().Kind() != reflect.Uint8 {
			idx, anyElem := nextUnconsumed(children, consumed, anyCursor)
			for anyElem != nil {
				consumed[idx] = true
				anyCursor = idx + 1
				item := reflect.New(field.Type().Elem()).Elem()
				if err := assignFromElement(item, anyElem); err != nil {
					return err
				}
				field.Set(reflect.Append(field, item))
				idx, anyElem = nextUnconsumed(children, consumed, anyCursor)
			}
			continue
		}

		idx, anyElem := nextUnconsumed(children, consumed, anyCursor)
		for anyElem != nil {
			consumed[idx] = true
			anyCursor = idx + 1
			if err := assignFromElement(field, anyElem); err != nil {
				return err
			}
			idx, anyElem = nextUnconsumed(children, consumed, anyCursor)
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
				if _, ok := consumedAttr[i]; ok {
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
			if _, ok := consumedAttr[i]; ok {
				continue
			}
			consumedAttr[i] = struct{}{}
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

	if field.Kind() == reflect.Slice && field.Type().Elem() == reflect.TypeFor[stdxml.Attr]() {
		field.Set(reflect.Append(field, reflect.ValueOf(toStdAttr(attr))))
		return nil
	}

	if field.Type() == reflect.TypeFor[stdxml.Attr]() {
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
	elem   *helium.Element
	tokens []stdxml.Token
	pos    int
	built  bool
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

	for child := range helium.Children(elem) {
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
		return cached.([]fieldBinding), nil //nolint:forcetypeassert
	}

	bindings := make([]fieldBinding, 0, t.NumField())
	collectFieldBindings(t, nil, &bindings, map[reflect.Type]struct{}{})
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

	n := min(len(a), len(b))

	for i := range n {
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

	shadowed := make(map[int]struct{})
	for i, a := range elems {
		if _, ok := shadowed[a.keptIdx]; ok {
			continue
		}
		for j, b := range elems {
			_, bShadowed := shadowed[b.keptIdx]
			if i == j || bShadowed {
				continue
			}
			if a.topName != b.topName {
				continue
			}
			// Same top-level name at different depths: shallower wins
			if a.depth < b.depth {
				shadowed[b.keptIdx] = struct{}{}
			} else if b.depth < a.depth {
				shadowed[a.keptIdx] = struct{}{}
				break
			}
		}
	}

	if len(shadowed) > 0 {
		filtered := make([]fieldBinding, 0, len(kept))
		for i, b := range kept {
			if _, ok := shadowed[i]; !ok {
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
		if spec == "" || spec == xmlNameField {
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

func collectFieldBindings(t reflect.Type, parentIndex []int, out *[]fieldBinding, seen map[reflect.Type]struct{}) {
	if _, ok := seen[t]; ok {
		return
	}
	seen[t] = struct{}{}
	defer func() {
		delete(seen, t)
	}()

	for i := range t.NumField() {
		f := t.Field(i)
		idx := append(append([]int(nil), parentIndex...), i)

		if shouldFlattenEmbeddedField(f) {
			embeddedType := derefType(f.Type)
			if _, ok := seen[embeddedType]; !ok {
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
	b.matchSpace, b.matchLocal, b.matchHasSpace = parseTagNameSpec(b.rawName)

	if f.Name == xmlNameField {
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
			b.matchSpace, b.matchLocal, b.matchHasSpace = parseTagNameSpec(b.rawName)
			return b
		}
		// If the field type is a struct with an XMLName tag, use that tag
		// name for element matching (stdlib precedence: XMLName tag > field name).
		if xmlNameTag := structXMLNameTag(derefType(f.Type)); xmlNameTag != "" {
			b.rawName = xmlNameTag
			b.nameSpace, b.name, b.hasNameSpace = parseTagNameSpec(xmlNameTag)
			b.matchSpace, b.matchLocal, b.matchHasSpace = parseTagNameSpec(b.rawName)
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
		b.matchSpace, b.matchLocal, b.matchHasSpace = parseTagNameSpec(b.rawName)

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
	for f := range t.Fields() {
		if f.Name != xmlNameField {
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
		if name != "" && name != xmlNameField {
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
	for child := range helium.Children(elem) {
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
	if elem == nil || elem.FirstChild() == nil {
		return ""
	}
	// Serialize elem as a whole and strip its own start and end tags, rather
	// than serializing each child in isolation. This keeps elem's in-scope
	// namespaces (including any inherited from an ancestor) established while the
	// children are written, so a child inheriting a namespace is not given a
	// spurious xmlns re-declaration — matching encoding/xml's raw innerxml
	// capture. Text and attribute values escape '<' and '>', so the first '>'
	// terminates elem's start tag and the last '<' begins elem's end tag.
	var b bytes.Buffer
	w := helium.NewWriter().XMLDeclaration(false)
	// Seed the namespaces in force on elem's ancestors so an inherited prefix
	// elem itself does not use is not re-declared on a child. encoding/xml's raw
	// ,innerxml capture carries no such re-declaration.
	if inherited := inheritedNamespaces(elem); len(inherited) > 0 {
		w = w.InheritedNamespaces(inherited)
	}
	if err := w.WriteTo(&b, elem); err != nil {
		return ""
	}
	s := b.String()
	start := strings.IndexByte(s, '>')
	end := strings.LastIndexByte(s, '<')
	if start < 0 || end < 0 || start+1 > end {
		return ""
	}
	return s[start+1 : end]
}

// inheritedNamespaces collects the namespace bindings in force on elem's
// ancestors (prefix -> URI; empty prefix = default namespace), with the nearest
// ancestor winning for a repeated prefix. Seeding these into the serializer keeps
// an inherited prefix that elem itself does not declare from being re-declared on
// a child, matching encoding/xml's raw ,innerxml capture. Returns nil when elem
// has no ancestor namespace declarations.
func inheritedNamespaces(elem *helium.Element) map[string]string {
	var out map[string]string
	for anc := elem.Parent(); anc != nil; anc = anc.Parent() {
		nser, ok := anc.(helium.Namespacer)
		if !ok {
			continue
		}
		for _, ns := range nser.Namespaces() {
			prefix := ns.Prefix()
			if _, seen := out[prefix]; seen {
				continue
			}
			if out == nil {
				out = make(map[string]string)
			}
			out[prefix] = ns.URI()
		}
	}
	return out
}

func elementComment(elem *helium.Element) string {
	if elem == nil {
		return ""
	}
	var b strings.Builder
	for child := range helium.Children(elem) {
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
	for child := range helium.Children(elem) {
		if ce, ok := helium.AsNode[*helium.Element](child); ok {
			result = append(result, ce)
		}
	}
	return result
}

// nextUnconsumed returns the first unconsumed child at or after from. The
// any-element pass calls it with a cursor that only ever moves forward, so
// resuming at the cursor instead of restarting at 0 skips only children
// already known to be consumed.
func nextUnconsumed(children []*helium.Element, consumed []bool, from int) (int, *helium.Element) {
	for i := from; i < len(children); i++ {
		if !consumed[i] {
			return i, children[i]
		}
	}
	return -1, nil
}

// pathFrame is one level of a pathScanner's depth-first walk: the sibling
// list at that level (a wrapper's already-materialized children), the index
// of the sibling currently being tried, and which path segment this level
// matches against.
type pathFrame struct {
	children []*helium.Element
	i        int
	level    int
}

// pathScanner walks a multi-segment path (e.g. ["A","B","C"]) through a
// parent's children without consuming wrapper elements, so multiple bindings
// can share the same wrapper. It is the preorder walk findPathLeaf used to
// perform from scratch on every call, made resumable: next resumes exactly
// where the previous call left off instead of re-descending from the root,
// so each wrapper's children are materialized once for the scanner's whole
// life instead of once per leaf claimed. The ns/hasNS fields apply to the
// leaf element only, matching stdlib behavior.
type pathScanner struct {
	path   []string
	ns     string
	hasNS  bool
	frames []pathFrame
}

// newPathScanner seeds a scanner over the parent's already-materialized
// children, ready to walk path from its first segment.
func newPathScanner(children []*helium.Element, path []string, ns string, hasNS bool) *pathScanner {
	return &pathScanner{
		path:  path,
		ns:    ns,
		hasNS: hasNS,
		frames: []pathFrame{
			{children: children, i: 0, level: 0},
		},
	}
}

// next returns the wrapper index (the position of the top-level path segment
// among the parent's children) and the next unconsumed, namespace-matching
// leaf, resuming the walk from where the previous call stopped. It returns
// (-1, nil) once the whole path has been exhausted, and every call after
// that returns the same sentinel without rescanning.
func (s *pathScanner) next(consumedLeaves map[*helium.Element]struct{}) (int, *helium.Element) {
	for len(s.frames) > 0 {
		top := &s.frames[len(s.frames)-1]
		if top.i >= len(top.children) {
			s.frames = s.frames[:len(s.frames)-1]
			if len(s.frames) == 0 {
				return -1, nil
			}
			s.frames[len(s.frames)-1].i++
			continue
		}

		child := top.children[top.i]
		if localName(child.Name()) != s.path[top.level] {
			top.i++
			continue
		}

		if top.level == len(s.path)-1 {
			// Leaf level: the top-level frame's current index is the
			// wrapper index, no matter how deep this frame is.
			wrapperIdx := s.frames[0].i
			top.i++
			if _, ok := consumedLeaves[child]; ok {
				continue
			}
			if s.hasNS && child.URI() != s.ns {
				continue
			}
			return wrapperIdx, child
		}

		// Descend into the matching wrapper to look for the next segment.
		// The parent frame's index advances only once this child frame is
		// exhausted, so a leaf found on a later next() call still resumes
		// inside this same descent.
		s.frames = append(s.frames, pathFrame{
			children: childElements(child),
			i:        0,
			level:    top.level + 1,
		})
	}

	return -1, nil
}

// findFrom scans children for the first unconsumed element matching
// local/space/hasSpace, starting at index from. Within one binding's drain,
// each match is consumed before the next probe, and the consumed set never
// un-marks an index, so resuming at the last claimed index plus one — instead
// of restarting at 0 — sees exactly the same children a from-0 scan would.
// The consumed check still applies at and after from, since a child there may
// already have been claimed by an earlier binding's own drain.
func findFrom(children []*helium.Element, consumed []bool, from int, local, space string, hasSpace bool) (int, *helium.Element) {
	for i := from; i < len(children); i++ {
		if consumed[i] {
			continue
		}
		child := children[i]
		if localName(child.Name()) != local {
			continue
		}
		if hasSpace && child.URI() != space {
			continue
		}
		return i, child
	}
	return -1, nil
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
	if _, after, ok := strings.Cut(name, ":"); ok {
		return after
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

var xmlNameType = reflect.TypeFor[stdxml.Name]()
var attrSliceType = reflect.TypeFor[[]stdxml.Attr]()

func isXMLNameType(t reflect.Type) bool {
	return t == xmlNameType
}

func derefType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}
