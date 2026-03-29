package xpath3

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

func init() {
	registerFn("json-to-xml", 1, 2, fnJSONToXML)
	registerFn("xml-to-json", 1, 2, fnXMLToJSON)
}

func fnJSONToXML(ctx context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return validNilSequence, nil
	}

	opts, err := parseJSONToXMLOptions(args)
	if err != nil {
		return nil, err
	}

	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}
	s = strings.TrimPrefix(s, "\uFEFF")
	s, invalidEsc, err := preprocessJSONStringLiterals(s)
	if err != nil {
		return nil, err
	}
	opts.invalidEsc = invalidEsc

	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	item, err := parseJSONValue(ctx, dec, opts)
	if err != nil {
		return nil, err
	}

	var extra json.Token
	extra, err = dec.Token()
	if err == nil {
		return nil, &XPathError{Code: errCodeFOJS0001, Message: fmt.Sprintf("unexpected trailing content after JSON value: %v", extra)}
	}
	if !errors.Is(err, io.EOF) {
		return nil, &XPathError{Code: errCodeFOJS0001, Message: fmt.Sprintf("invalid trailing content: %v", err)}
	}

	doc := helium.NewDefaultDocument()
	if ec := getFnContext(ctx); ec != nil && ec.baseURI != "" {
		doc.SetURL(ec.baseURI)
	}
	root, err := buildJSONToXMLTree(doc, item, opts, true)
	if err != nil {
		return nil, err
	}
	if err := doc.SetDocumentElement(root); err != nil {
		return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("json-to-xml: failed to build result: %v", err)}
	}
	return ItemSlice{NodeItem{Node: doc}}, nil
}

func buildJSONToXMLTree(doc *helium.Document, item Item, opts jsonOptions, root bool) (*helium.Element, error) {
	name := "null"
	switch v := item.(type) {
	case MapItem:
		name = "map"
		_ = v
	case ArrayItem:
		name = "array"
	case AtomicValue:
		switch v.TypeName {
		case TypeString:
			name = lexicon.TypeString
		case TypeBoolean:
			name = lexicon.TypeBoolean
		default:
			name = lexicon.TypeNumber
		}
	}

	elem := doc.CreateElement(name)
	if root {
		if err := elem.DeclareNamespace("", NSFn); err != nil {
			return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("json-to-xml: failed to build result: %v", err)}
		}
	}
	if err := elem.SetActiveNamespace("", NSFn); err != nil {
		return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("json-to-xml: failed to build result: %v", err)}
	}

	switch v := item.(type) {
	case nil:
		return elem, nil
	case MapItem:
		if err := v.ForEach(func(key AtomicValue, value Sequence) error {
			child, err := buildJSONToXMLTree(doc, jsonSequenceToItem(value), opts, false)
			if err != nil {
				return err
			}
			keyText := key.StringVal()
			_ = child.SetLiteralAttribute("key", keyText)
			if opts.escape {
				_ = child.SetLiteralAttribute("escaped-key", lexicon.ValueTrue)
			}
			if err := elem.AddChild(child); err != nil {
				return &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("json-to-xml: failed to attach child: %v", err)}
			}
			return nil
		}); err != nil {
			return nil, err
		}
	case ArrayItem:
		for _, member := range v.members0() {
			child, err := buildJSONToXMLTree(doc, jsonSequenceToItem(member), opts, false)
			if err != nil {
				return nil, err
			}
			if err := elem.AddChild(child); err != nil {
				return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("json-to-xml: failed to attach child: %v", err)}
			}
		}
	case AtomicValue:
		switch v.TypeName {
		case TypeString:
			if opts.escape {
				_ = elem.SetLiteralAttribute("escaped", lexicon.ValueTrue)
			}
			if err := elem.AppendText([]byte(v.StringVal())); err != nil {
				return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("json-to-xml: failed to append string value: %v", err)}
			}
		case TypeBoolean:
			text := lexicon.ValueFalse
			if v.BooleanVal() {
				text = lexicon.ValueTrue
			}
			if err := elem.AppendText([]byte(text)); err != nil {
				return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("json-to-xml: failed to append boolean value: %v", err)}
			}
		default:
			text, err := atomicToString(v)
			if err != nil {
				return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("json-to-xml: failed to format numeric value: %v", err)}
			}
			if err := elem.AppendText([]byte(text)); err != nil {
				return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("json-to-xml: failed to append numeric value: %v", err)}
			}
		}
	default:
		return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("json-to-xml: unsupported item type %T", item)}
	}

	return elem, nil
}

func jsonSequenceToItem(seq Sequence) Item {
	if seqLen(seq) == 0 {
		return nil
	}
	return seq.Get(0)
}

func parseJSONToXMLOptions(args []Sequence) (jsonOptions, error) {
	opts := jsonOptions{
		duplicates: "use-first",
	}
	if len(args) <= 1 || seqLen(args[1]) == 0 {
		return opts, nil
	}
	if seqLen(args[1]) != 1 {
		return opts, &XPathError{Code: errCodeXPTY0004, Message: "json-to-xml options must be a single map"}
	}

	m, ok := args[1].Get(0).(MapItem)
	if !ok {
		return opts, &XPathError{Code: errCodeXPTY0004, Message: "json-to-xml options must be a map"}
	}

	readBool := func(name string) (bool, bool, error) {
		key := AtomicValue{TypeName: TypeString, Value: name}
		v, found := m.Get(key)
		if !found {
			return false, false, nil
		}
		if seqLen(v) != 1 {
			return false, true, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("option '%s' must be a single xs:boolean", name)}
		}
		av, ok := v.Get(0).(AtomicValue)
		if !ok || av.TypeName != TypeBoolean {
			return false, true, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("option '%s' must be xs:boolean", name)}
		}
		return av.BooleanVal(), true, nil
	}

	if liberal, found, err := readBool("liberal"); err != nil {
		return opts, err
	} else if found {
		opts.liberal = liberal
	}

	validate := false
	validateSet := false
	if validateValue, found, err := readBool("validate"); err != nil {
		return opts, err
	} else if found {
		validateSet = true
		validate = validateValue
	}

	if escape, found, err := readBool("escape"); err != nil {
		return opts, err
	} else if found {
		opts.escape = escape
	}

	dupKey := AtomicValue{TypeName: TypeString, Value: "duplicates"}
	if v, found := m.Get(dupKey); found {
		if seqLen(v) != 1 {
			return opts, &XPathError{Code: errCodeXPTY0004, Message: "option 'duplicates' must be a single xs:string"}
		}
		s, err := coerceArgToString(v)
		if err != nil {
			return opts, &XPathError{Code: errCodeXPTY0004, Message: "option 'duplicates' must be xs:string"}
		}
		switch s {
		case "reject", "use-first", "retain":
			opts.duplicates = s
		case "use-last":
			return opts, &XPathError{Code: errCodeFOJS0005, Message: "option 'duplicates' must not be 'use-last' for json-to-xml"}
		default:
			return opts, &XPathError{Code: errCodeFOJS0005, Message: fmt.Sprintf("invalid value for 'duplicates' option: %q", s)}
		}
	}

	// When validate=true() is set and no explicit duplicates option was
	// provided, default duplicates to 'reject' per the spec.
	// We silently accept validate=true() even without schema support.
	if validateSet && validate {
		dupKey2 := AtomicValue{TypeName: TypeString, Value: "duplicates"}
		if _, found := m.Get(dupKey2); !found {
			opts.duplicates = "reject"
		}
	}

	fbKey := AtomicValue{TypeName: TypeString, Value: "fallback"}
	if v, found := m.Get(fbKey); found {
		if seqLen(v) != 1 {
			return opts, &XPathError{Code: errCodeXPTY0004, Message: "option 'fallback' must be a single function item"}
		}
		fi, ok := v.Get(0).(FunctionItem)
		if !ok {
			return opts, &XPathError{Code: errCodeXPTY0004, Message: "option 'fallback' must be a function item"}
		}
		if fi.Arity != 1 {
			return opts, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("option 'fallback' must have arity 1, got %d", fi.Arity)}
		}
		opts.fallback = &fi
	}

	if opts.escape && opts.fallback != nil {
		return opts, &XPathError{Code: errCodeFOJS0005, Message: "fallback cannot be combined with escape=true()"}
	}

	return opts, nil
}

type xmlToJSONOptions struct {
	indent bool
}

type xmlJSONInherited struct {
	escaped    bool
	escapedKey bool
}

type xmlJSONMeta struct {
	kind       string
	inherited  xmlJSONInherited
	hasKey     bool
	keyActual  string
	keyEncoded string
}

func fnXMLToJSON(_ context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return validNilSequence, nil
	}
	if seqLen(args[0]) != 1 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "xml-to-json expects zero or one node"}
	}

	opts, err := parseXMLToJSONOptions(args)
	if err != nil {
		return nil, err
	}

	ni, ok := args[0].Get(0).(NodeItem)
	if !ok {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "xml-to-json expects an element or document node"}
	}

	root, err := xmlToJSONRootElement(ni.Node)
	if err != nil {
		return nil, err
	}

	jsonText, _, err := serializeJSONXMLElement(root, xmlJSONInherited{}, opts, 0)
	if err != nil {
		return nil, err
	}
	return SingleString(jsonText), nil
}

func parseXMLToJSONOptions(args []Sequence) (xmlToJSONOptions, error) {
	var opts xmlToJSONOptions
	if len(args) <= 1 {
		return opts, nil
	}
	if seqLen(args[1]) != 1 {
		return opts, &XPathError{Code: errCodeXPTY0004, Message: "xml-to-json options must be a single map"}
	}
	m, ok := args[1].Get(0).(MapItem)
	if !ok {
		return opts, &XPathError{Code: errCodeXPTY0004, Message: "xml-to-json options must be a map"}
	}

	key := AtomicValue{TypeName: TypeString, Value: "indent"}
	if v, found := m.Get(key); found {
		if seqLen(v) != 1 {
			return opts, &XPathError{Code: errCodeXPTY0004, Message: "option 'indent' must be a single xs:boolean"}
		}
		av, ok := v.Get(0).(AtomicValue)
		if !ok || av.TypeName != TypeBoolean {
			return opts, &XPathError{Code: errCodeXPTY0004, Message: "option 'indent' must be xs:boolean"}
		}
		opts.indent = av.BooleanVal()
	}
	return opts, nil
}

func xmlToJSONRootElement(node helium.Node) (*helium.Element, error) {
	switch n := node.(type) {
	case *helium.Document:
		root := n.DocumentElement()
		if root == nil {
			return nil, &XPathError{Code: errCodeFOJS0006, Message: "xml-to-json: document has no document element"}
		}
		// FOJS0006: document must have exactly one element child.
		elemCount := 0
		for child := range helium.Children(n) {
			if child.Type() == helium.ElementNode {
				elemCount++
			}
		}
		if elemCount != 1 {
			return nil, &XPathError{Code: errCodeFOJS0006, Message: "xml-to-json: document must have exactly one element child"}
		}
		return root, nil
	case *helium.Element:
		return n, nil
	default:
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "xml-to-json expects an element or document node"}
	}
}

func serializeJSONXMLElement(elem *helium.Element, inherited xmlJSONInherited, opts xmlToJSONOptions, depth int) (string, xmlJSONMeta, error) {
	meta, err := parseXMLJSONMeta(elem, inherited)
	if err != nil {
		return "", xmlJSONMeta{}, err
	}

	switch meta.kind {
	case "map":
		children, err := jsonElementChildren(elem, true)
		if err != nil {
			return "", xmlJSONMeta{}, err
		}
		parts := make([]string, 0, len(children))
		seen := make(map[string]struct{}, len(children))
		for _, child := range children {
			value, childMeta, err := serializeJSONXMLElement(child, meta.inherited, opts, depth+1)
			if err != nil {
				return "", xmlJSONMeta{}, err
			}
			if !childMeta.hasKey {
				return "", xmlJSONMeta{}, &XPathError{Code: errCodeFOJS0006, Message: "xml-to-json: map child is missing key attribute"}
			}
			if _, exists := seen[childMeta.keyActual]; exists {
				return "", xmlJSONMeta{}, &XPathError{Code: errCodeFOJS0006, Message: fmt.Sprintf("xml-to-json: duplicate key %q", childMeta.keyActual)}
			}
			seen[childMeta.keyActual] = struct{}{}
			parts = append(parts, `"`+childMeta.keyEncoded+`":`+valueSeparator(opts.indent)+value)
		}
		return formatJSONComposite("{", "}", parts, depth, opts.indent), meta, nil
	case "array":
		children, err := jsonElementChildren(elem, false)
		if err != nil {
			return "", xmlJSONMeta{}, err
		}
		parts := make([]string, 0, len(children))
		for _, child := range children {
			value, _, err := serializeJSONXMLElement(child, meta.inherited, opts, depth+1)
			if err != nil {
				return "", xmlJSONMeta{}, err
			}
			parts = append(parts, value)
		}
		return formatJSONComposite("[", "]", parts, depth, opts.indent), meta, nil
	case lexicon.TypeString:
		content, err := scalarElementText(elem, true)
		if err != nil {
			return "", xmlJSONMeta{}, err
		}
		if meta.inherited.escaped {
			encoded, _, err := validateEscapedJSONContent(content)
			if err != nil {
				return "", xmlJSONMeta{}, err
			}
			return `"` + encoded + `"`, meta, nil
		}
		return `"` + encodeJSONStringContent(content) + `"`, meta, nil
	case "number":
		content, err := scalarElementText(elem, true)
		if err != nil {
			return "", xmlJSONMeta{}, err
		}
		number, err := canonicalizeXMLJSONNumber(content)
		if err != nil {
			return "", xmlJSONMeta{}, err
		}
		return number, meta, nil
	case "boolean":
		content, err := scalarElementText(elem, true)
		if err != nil {
			return "", xmlJSONMeta{}, err
		}
		boolean, err := canonicalizeXMLJSONBoolean(content)
		if err != nil {
			return "", xmlJSONMeta{}, err
		}
		return boolean, meta, nil
	case "null":
		if err := validateNullElement(elem); err != nil {
			return "", xmlJSONMeta{}, err
		}
		return "null", meta, nil
	default:
		return "", xmlJSONMeta{}, &XPathError{Code: errCodeFOJS0006, Message: fmt.Sprintf("xml-to-json: unsupported element %s", meta.kind)}
	}
}

func parseXMLJSONMeta(elem *helium.Element, inherited xmlJSONInherited) (xmlJSONMeta, error) {
	if elem.URI() != NSFn {
		return xmlJSONMeta{}, &XPathError{Code: errCodeFOJS0006, Message: fmt.Sprintf("xml-to-json: invalid namespace %q", elem.URI())}
	}

	meta := xmlJSONMeta{
		kind:      elem.LocalName(),
		inherited: inherited,
	}

	switch meta.kind {
	case "map", "array", lexicon.TypeString, "number", "boolean", "null":
	default:
		return xmlJSONMeta{}, &XPathError{Code: errCodeFOJS0006, Message: fmt.Sprintf("xml-to-json: invalid element %q", meta.kind)}
	}

	for _, attr := range elem.Attributes() {
		switch attr.URI() {
		case "":
			switch attr.LocalName() {
			case "key":
				meta.hasKey = true
				meta.keyActual = attr.Value()
			case "escaped":
				v, err := parseXMLJSONBooleanAttr(attr.Value())
				if err != nil {
					return xmlJSONMeta{}, err
				}
				meta.inherited.escaped = v
			case "escaped-key":
				v, err := parseXMLJSONBooleanAttr(attr.Value())
				if err != nil {
					return xmlJSONMeta{}, err
				}
				meta.inherited.escapedKey = v
			default:
				return xmlJSONMeta{}, &XPathError{Code: errCodeFOJS0006, Message: fmt.Sprintf("xml-to-json: invalid attribute %q", attr.Name())}
			}
		case NSFn:
			return xmlJSONMeta{}, &XPathError{Code: errCodeFOJS0006, Message: fmt.Sprintf("xml-to-json: invalid attribute namespace for %q", attr.Name())}
		default:
			// Attributes in other namespaces are ignored.
		}
	}

	if meta.hasKey {
		if meta.inherited.escapedKey {
			encoded, actual, err := validateEscapedJSONContent(meta.keyActual)
			if err != nil {
				return xmlJSONMeta{}, err
			}
			meta.keyEncoded = encoded
			meta.keyActual = actual
		} else {
			meta.keyEncoded = encodeJSONStringContent(meta.keyActual)
		}
	}
	return meta, nil
}

func jsonElementChildren(elem *helium.Element, insideMap bool) ([]*helium.Element, error) {
	var children []*helium.Element
	for child := range helium.Children(elem) {
		switch v := child.(type) {
		case *helium.Element:
			children = append(children, v)
		case *helium.Text, *helium.CDATASection:
			if strings.TrimSpace(string(child.Content())) != "" {
				kind := "array"
				if insideMap {
					kind = "map"
				}
				return nil, &XPathError{Code: errCodeFOJS0006, Message: fmt.Sprintf("xml-to-json: %s contains non-whitespace text", kind)}
			}
		case *helium.Comment, *helium.ProcessingInstruction:
			continue
		default:
			return nil, &XPathError{Code: errCodeFOJS0006, Message: "xml-to-json: invalid child node"}
		}
	}
	return children, nil
}

func scalarElementText(elem *helium.Element, allowComments bool) (string, error) {
	var b strings.Builder
	for child := range helium.Children(elem) {
		switch child.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			b.Write(child.Content())
		case helium.CommentNode, helium.ProcessingInstructionNode:
			if !allowComments {
				return "", &XPathError{Code: errCodeFOJS0006, Message: "xml-to-json: invalid child node"}
			}
		case helium.ElementNode:
			return "", &XPathError{Code: errCodeFOJS0006, Message: "xml-to-json: scalar value contains element children"}
		default:
			return "", &XPathError{Code: errCodeFOJS0006, Message: "xml-to-json: invalid child node"}
		}
	}
	return b.String(), nil
}

func validateNullElement(elem *helium.Element) error {
	for child := range helium.Children(elem) {
		switch child.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			if strings.TrimSpace(string(child.Content())) != "" {
				return &XPathError{Code: errCodeFOJS0006, Message: "xml-to-json: null element must not contain text"}
			}
		case helium.CommentNode, helium.ProcessingInstructionNode:
			continue
		case helium.ElementNode:
			return &XPathError{Code: errCodeFOJS0006, Message: "xml-to-json: null element must not contain child elements"}
		default:
			return &XPathError{Code: errCodeFOJS0006, Message: "xml-to-json: invalid child node"}
		}
	}
	return nil
}

func parseXMLJSONBooleanAttr(s string) (bool, error) {
	switch strings.TrimSpace(s) {
	case lexicon.ValueTrue, "1":
		return true, nil
	case lexicon.ValueFalse, "0":
		return false, nil
	default:
		return false, &XPathError{Code: errCodeFOJS0006, Message: fmt.Sprintf("xml-to-json: invalid boolean attribute value %q", s)}
	}
}

func canonicalizeXMLJSONBoolean(s string) (string, error) {
	switch strings.TrimSpace(s) {
	case lexicon.ValueTrue, "1":
		return lexicon.ValueTrue, nil
	case lexicon.ValueFalse, "0":
		return lexicon.ValueFalse, nil
	default:
		return "", &XPathError{Code: errCodeFOJS0006, Message: fmt.Sprintf("xml-to-json: invalid boolean value %q", s)}
	}
}

func canonicalizeXMLJSONNumber(s string) (string, error) {
	norm := strings.TrimSpace(s)
	if norm == "" {
		return "", &XPathError{Code: errCodeFOJS0006, Message: "xml-to-json: invalid number value"}
	}
	if strings.EqualFold(norm, "nan") || strings.Contains(strings.ToUpper(norm), "INF") {
		return "", &XPathError{Code: errCodeFOJS0006, Message: fmt.Sprintf("xml-to-json: invalid number value %q", s)}
	}
	norm = preprocessXMLJSONNumber(norm)
	av, err := CastFromString(norm, TypeDouble)
	if err != nil {
		return "", &XPathError{Code: errCodeFOJS0006, Message: fmt.Sprintf("xml-to-json: invalid number value %q", s)}
	}
	return formatXPathDouble(av.DoubleVal()), nil
}

func preprocessXMLJSONNumber(s string) string {
	sign := ""
	if strings.HasPrefix(s, "+") || strings.HasPrefix(s, "-") {
		sign = s[:1]
		s = s[1:]
	}

	exp := ""
	if idx := strings.IndexAny(s, "eE"); idx >= 0 {
		exp = s[idx:]
		s = s[:idx]
	}

	if strings.HasPrefix(s, ".") {
		s = "0" + s
	}
	s = strings.TrimSuffix(s, ".")
	if s == "" {
		s = "0"
	}

	intPart := s
	fracPart := ""
	if before, after, ok := strings.Cut(s, "."); ok {
		intPart = before
		fracPart = after
	}
	if intPart == "" {
		intPart = "0"
	}
	intPart = strings.TrimLeft(intPart, "0")
	if intPart == "" {
		intPart = "0"
	}
	if fracPart != "" {
		return sign + intPart + "." + fracPart + exp
	}
	return sign + intPart + exp
}

func validateEscapedJSONContent(s string) (string, string, error) {
	var out strings.Builder
	var decoded strings.Builder

	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r != '\\' {
			appendSerializedJSONStringRune(&out, r)
			decoded.WriteRune(r)
			i += size
			continue
		}

		if i+1 >= len(s) {
			return "", "", &XPathError{Code: errCodeFOJS0007, Message: "xml-to-json: invalid trailing backslash in escaped string"}
		}

		switch s[i+1] {
		case '"':
			out.WriteString(`\"`)
			decoded.WriteByte('"')
			i += 2
		case '\\':
			out.WriteString(`\\`)
			decoded.WriteByte('\\')
			i += 2
		case '/':
			out.WriteString(`\/`)
			decoded.WriteByte('/')
			i += 2
		case 'b':
			out.WriteString(`\b`)
			decoded.WriteByte('\b')
			i += 2
		case 'f':
			out.WriteString(`\f`)
			decoded.WriteByte('\f')
			i += 2
		case 'n':
			out.WriteString(`\n`)
			decoded.WriteByte('\n')
			i += 2
		case 'r':
			out.WriteString(`\r`)
			decoded.WriteByte('\r')
			i += 2
		case 't':
			out.WriteString(`\t`)
			decoded.WriteByte('\t')
			i += 2
		case 'u':
			encoded, actual, consumed, err := parseEscapedUnicodeSequence(s[i:])
			if err != nil {
				return "", "", err
			}
			out.WriteString(encoded)
			decoded.WriteString(actual)
			i += consumed
		default:
			return "", "", &XPathError{Code: errCodeFOJS0007, Message: fmt.Sprintf("xml-to-json: invalid escape sequence \\%c", s[i+1])}
		}
	}

	return out.String(), decoded.String(), nil
}

func parseEscapedUnicodeSequence(s string) (string, string, int, error) {
	if len(s) < 6 {
		return "", "", 0, &XPathError{Code: errCodeFOJS0007, Message: "xml-to-json: incomplete \\u escape"}
	}
	if s[0] != '\\' || s[1] != 'u' || !isHexQuad(s[2:6]) {
		return "", "", 0, &XPathError{Code: errCodeFOJS0007, Message: "xml-to-json: invalid \\u escape"}
	}
	cp, _ := parseJSONHexEscape(s[2:6])
	encoded := s[:6]

	switch {
	case cp >= 0xD800 && cp <= 0xDBFF:
		if len(s) < 12 || s[6] != '\\' || s[7] != 'u' || !isHexQuad(s[8:12]) {
			return "", "", 0, &XPathError{Code: errCodeFOJS0007, Message: "xml-to-json: invalid surrogate pair"}
		}
		cp2, _ := parseJSONHexEscape(s[8:12])
		if cp2 < 0xDC00 || cp2 > 0xDFFF {
			return "", "", 0, &XPathError{Code: errCodeFOJS0007, Message: "xml-to-json: invalid surrogate pair"}
		}
		r := rune(((cp - 0xD800) << 10) + (cp2 - 0xDC00) + 0x10000)
		return encoded + s[6:12], string(r), 12, nil
	case cp >= 0xDC00 && cp <= 0xDFFF:
		return "", "", 0, &XPathError{Code: errCodeFOJS0007, Message: "xml-to-json: invalid low surrogate"}
	default:
		return encoded, string(rune(cp)), 6, nil
	}
}

func isHexQuad(s string) bool {
	if len(s) != 4 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

func encodeJSONStringContent(s string) string {
	var b strings.Builder
	for _, r := range s {
		appendSerializedJSONStringRune(&b, r)
	}
	return b.String()
}

func formatJSONComposite(open, closeBracket string, parts []string, depth int, indent bool) string {
	if !indent || len(parts) == 0 {
		return open + strings.Join(parts, ",") + closeBracket
	}

	var b strings.Builder
	childIndent := strings.Repeat("  ", depth+1)
	parentIndent := strings.Repeat("  ", depth)
	b.WriteString(open)
	b.WriteByte('\n')
	for i, part := range parts {
		if i > 0 {
			b.WriteString(",\n")
		}
		b.WriteString(childIndent)
		b.WriteString(part)
	}
	b.WriteByte('\n')
	b.WriteString(parentIndent)
	b.WriteString(closeBracket)
	return b.String()
}

func valueSeparator(indent bool) string {
	if indent {
		return " "
	}
	return ""
}
