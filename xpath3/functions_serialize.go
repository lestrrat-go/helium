package xpath3

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

func init() {
	registerFn("serialize", 1, 2, fnSerialize)
}

func fnSerialize(_ context.Context, args []Sequence) (Sequence, error) {
	opts, err := parseSerializeOptions(args)
	if err != nil {
		return nil, err
	}

	var result string
	switch opts.method {
	case "", "adaptive":
		result, err = serializeAdaptiveSequence(args[0], opts)
	case "json":
		result, err = serializeJSONSequence(args[0], opts)
	default:
		result, err = serializeXMLSequence(args[0], opts)
	}
	if err != nil {
		return nil, err
	}
	return SingleString(result), nil
}

type serializeOptions struct {
	method              string
	itemSeparator       string
	indent              bool
	omitXMLDeclaration  bool
	allowDuplicateNames bool
	encoding            string
}

func parseSerializeOptions(args []Sequence) (serializeOptions, error) {
	opts := serializeOptions{
		method:        "adaptive",
		itemSeparator: " ",
	}
	if len(args) <= 1 || seqLen(args[1]) == 0 {
		return opts, nil
	}

	m, ok := args[1].Get(0).(MapItem)
	if ok {
		return parseSerializeOptionsMap(opts, m)
	}
	if seqLen(args[1]) != 1 {
		return opts, &XPathError{Code: errCodeXPTY0004, Message: "serialize options must be a singleton"}
	}
	node, ok := args[1].Get(0).(NodeItem)
	if !ok {
		return opts, nil
	}
	return parseSerializeOptionsNode(opts, node.Node)
}

func parseSerializeOptionsMap(opts serializeOptions, m MapItem) (serializeOptions, error) {
	readString := func(name string) (string, bool, error) {
		v, found := m.Get(AtomicValue{TypeName: TypeString, Value: name})
		if !found {
			return "", false, nil
		}
		if seqLen(v) != 1 {
			return "", true, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("serialize option %q must be a singleton", name)}
		}
		av, ok := v.Get(0).(AtomicValue)
		if !ok {
			return "", true, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("serialize option %q must be atomic", name)}
		}
		s, err := atomicToString(av)
		if err != nil {
			return "", true, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("serialize option %q must be string-like", name)}
		}
		return s, true, nil
	}

	readBool := func(name string) (bool, bool, error) {
		v, found := m.Get(AtomicValue{TypeName: TypeString, Value: name})
		if !found {
			return false, false, nil
		}
		if seqLen(v) != 1 {
			return false, true, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("serialize option %q must be a single xs:boolean", name)}
		}
		av, ok := v.Get(0).(AtomicValue)
		if !ok {
			return false, true, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("serialize option %q must be xs:boolean", name)}
		}
		switch av.TypeName {
		case TypeBoolean:
			return av.BooleanVal(), true, nil
		case TypeUntypedAtomic:
			s, err := atomicToString(av)
			if err != nil {
				return false, true, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("serialize option %q must be xs:boolean", name)}
			}
			switch s {
			case "true", "1":
				return true, true, nil
			case "false", "0":
				return false, true, nil
			default:
				return false, true, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("serialize option %q must be xs:boolean", name)}
			}
		default:
			return false, true, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("serialize option %q must be xs:boolean", name)}
		}
	}

	if method, found, err := readString("method"); err != nil {
		return opts, err
	} else if found {
		opts.method = method
	}
	if sep, found, err := readString("item-separator"); err != nil {
		return opts, err
	} else if found {
		opts.itemSeparator = sep
	}
	if indent, found, err := readBool("indent"); err != nil {
		return opts, err
	} else if found {
		opts.indent = indent
	}
	if omit, found, err := readBool("omit-xml-declaration"); err != nil {
		return opts, err
	} else if found {
		opts.omitXMLDeclaration = omit
	}
	if allow, found, err := readBool("allow-duplicate-names"); err != nil {
		return opts, err
	} else if found {
		opts.allowDuplicateNames = allow
	}
	if encoding, found, err := readString("encoding"); err != nil {
		return opts, err
	} else if found {
		opts.encoding = encoding
	}
	if v, found := m.Get(AtomicValue{TypeName: TypeString, Value: "use-character-maps"}); found {
		if err := validateSerializeCharacterMaps(v); err != nil {
			return opts, err
		}
	}

	return opts, nil
}

func parseSerializeOptionsNode(opts serializeOptions, n helium.Node) (serializeOptions, error) {
	elem, ok := n.(*helium.Element)
	if !ok {
		return opts, &XPathError{Code: errCodeXPTY0004, Message: "serialize options node must be an element"}
	}
	if elem.URI() != lexicon.NamespaceSerialization || elem.LocalName() != "serialization-parameters" {
		return opts, &XPathError{Code: errCodeXPTY0004, Message: "serialize options root must be output:serialization-parameters"}
	}
	if len(elem.Attributes()) != 0 {
		return opts, &XPathError{Code: errCodeXPTY0004, Message: "serialize options root must not have attributes"}
	}

	seen := make(map[string]struct{})
	for child := range helium.Children(elem) {
		switch child.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			if strings.TrimSpace(string(child.Content())) == "" {
				continue
			}
			return opts, &XPathError{Code: errCodeXPTY0004, Message: "serialize options root must not contain text"}
		case helium.CommentNode, helium.ProcessingInstructionNode:
			continue
		case helium.ElementNode:
		default:
			return opts, &XPathError{Code: errCodeXPTY0004, Message: "serialize options root has invalid child node"}
		}

		param := child.(*helium.Element)
		if param.URI() == "" {
			return opts, &XPathError{Code: errCodeXPTY0004, Message: "serialize options parameters must be namespace-qualified"}
		}

		key := param.URI() + "|" + param.LocalName()
		if _, exists := seen[key]; exists {
			return opts, &XPathError{Code: errCodeXPTY0004, Message: "serialize options parameter must not appear more than once"}
		}
		seen[key] = struct{}{}

		if param.URI() != lexicon.NamespaceSerialization {
			if _, err := readSerializeParamValue(param); err != nil {
				return opts, err
			}
			continue
		}

		switch param.LocalName() {
		case "method":
			value, err := readSerializeParamValue(param)
			if err != nil {
				return opts, err
			}
			opts.method = value
		case "item-separator":
			value, err := readSerializeParamValue(param)
			if err != nil {
				return opts, err
			}
			opts.itemSeparator = value
		case "indent":
			value, err := readSerializeParamYesNo(param)
			if err != nil {
				return opts, err
			}
			opts.indent = value
		case "omit-xml-declaration":
			value, err := readSerializeParamYesNo(param)
			if err != nil {
				return opts, err
			}
			opts.omitXMLDeclaration = value
		case "allow-duplicate-names":
			value, err := readSerializeParamYesNo(param)
			if err != nil {
				return opts, err
			}
			opts.allowDuplicateNames = value
		case "encoding":
			value, err := readSerializeParamValue(param)
			if err != nil {
				return opts, err
			}
			opts.encoding = value
		case "byte-order-mark", "cdata-section-elements", "doctype-public", "doctype-system",
			"json-node-output-method", "media-type", "normalization-form",
			"suppress-indentation", "version":
			if _, err := readSerializeParamValue(param); err != nil {
				return opts, err
			}
		case "standalone":
			if _, err := readSerializeParamStandalone(param); err != nil {
				return opts, err
			}
		case "use-character-maps":
			if err := validateSerializeCharacterMapsElement(param); err != nil {
				return opts, err
			}
		default:
			return opts, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("unsupported serialize parameter %q", param.LocalName())}
		}
	}

	return opts, nil
}

func readSerializeParamValue(elem *helium.Element) (string, error) {
	if hasNonWhitespaceContent(elem) {
		return "", &XPathError{Code: errCodeXPTY0004, Message: "serialize parameter must not have child content"}
	}
	attrs := elem.Attributes()
	if len(attrs) != 1 {
		return "", &XPathError{Code: errCodeXPTY0004, Message: "serialize parameter must have exactly one value attribute"}
	}
	attr := attrs[0]
	if attr.URI() != "" || attr.LocalName() != "value" {
		return "", &XPathError{Code: errCodeXPTY0004, Message: "serialize parameter must use an unqualified value attribute"}
	}
	return attr.Value(), nil
}

func readSerializeParamYesNo(elem *helium.Element) (bool, error) {
	value, err := readSerializeParamValue(elem)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case lexicon.ValueYes, "true", "1":
		return true, nil
	case lexicon.ValueNo, "false", "0":
		return false, nil
	default:
		return false, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("serialize parameter %q must be yes/no", elem.LocalName())}
	}
}

func readSerializeParamStandalone(elem *helium.Element) (string, error) {
	value, err := readSerializeParamValue(elem)
	if err != nil {
		return "", err
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case lexicon.ValueYes, lexicon.ValueNo, "omit":
		return value, nil
	default:
		return "", &XPathError{Code: errCodeXPTY0004, Message: "serialize parameter \"standalone\" must be yes/no/omit"}
	}
}

func validateSerializeCharacterMapsElement(elem *helium.Element) error {
	if len(elem.Attributes()) != 0 {
		return &XPathError{Code: errCodeXPTY0004, Message: "serialize parameter use-character-maps must not have attributes"}
	}
	seen := make(map[string]struct{})
	for child := range helium.Children(elem) {
		switch child.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			if strings.TrimSpace(string(child.Content())) == "" {
				continue
			}
			return &XPathError{Code: errCodeXPTY0004, Message: "use-character-maps must not contain text"}
		case helium.CommentNode, helium.ProcessingInstructionNode:
			continue
		case helium.ElementNode:
		default:
			return &XPathError{Code: errCodeXPTY0004, Message: "use-character-maps has invalid child node"}
		}

		charMap := child.(*helium.Element)
		if charMap.URI() != lexicon.NamespaceSerialization || charMap.LocalName() != "character-map" {
			return &XPathError{Code: errCodeXPTY0004, Message: "use-character-maps children must be output:character-map"}
		}
		if hasNonWhitespaceContent(charMap) {
			return &XPathError{Code: errCodeXPTY0004, Message: "character-map must not have child content"}
		}

		var character, mapString string
		for _, attr := range charMap.Attributes() {
			if attr.URI() != "" {
				return &XPathError{Code: errCodeXPTY0004, Message: "character-map attributes must be unqualified"}
			}
			switch attr.LocalName() {
			case "character":
				character = attr.Value()
			case "map-string":
				mapString = attr.Value()
			default:
				return &XPathError{Code: errCodeXPTY0004, Message: "character-map has unsupported attribute"}
			}
		}
		if character == "" || mapString == "" {
			return &XPathError{Code: errCodeXPTY0004, Message: "character-map requires character and map-string"}
		}
		if utf8.RuneCountInString(character) != 1 {
			return &XPathError{Code: errCodeXPTY0004, Message: "character-map character must be a single character"}
		}
		if _, exists := seen[character]; exists {
			return &XPathError{Code: errCodeXPTY0004, Message: "character-map entries must be unique"}
		}
		seen[character] = struct{}{}
	}
	return nil
}

func hasNonWhitespaceContent(elem *helium.Element) bool {
	for child := range helium.Children(elem) {
		switch child.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			if strings.TrimSpace(string(child.Content())) != "" {
				return true
			}
		case helium.CommentNode, helium.ProcessingInstructionNode:
			continue
		default:
			return true
		}
	}
	return false
}

func validateSerializeCharacterMaps(v Sequence) error {
	if seqLen(v) != 1 {
		return &XPathError{Code: errCodeXPTY0004, Message: "serialize option 'use-character-maps' must be a singleton map"}
	}
	m, ok := v.Get(0).(MapItem)
	if !ok {
		return &XPathError{Code: errCodeXPTY0004, Message: "serialize option 'use-character-maps' must be a map"}
	}
	return m.ForEach(func(key AtomicValue, value Sequence) error {
		if key.TypeName != TypeString && key.TypeName != TypeUntypedAtomic {
			return &XPathError{Code: errCodeXPTY0004, Message: "serialize use-character-maps keys must be strings"}
		}
		keyString, err := atomicToString(key)
		if err != nil || utf8.RuneCountInString(keyString) != 1 {
			return &XPathError{Code: errCodeXPTY0004, Message: "serialize use-character-maps keys must be single characters"}
		}
		if seqLen(value) != 1 {
			return &XPathError{Code: errCodeXPTY0004, Message: "serialize use-character-maps values must be singleton strings"}
		}
		av, ok := value.Get(0).(AtomicValue)
		if !ok || (av.TypeName != TypeString && av.TypeName != TypeUntypedAtomic) {
			return &XPathError{Code: errCodeXPTY0004, Message: "serialize use-character-maps values must be strings"}
		}
		_, err = atomicToString(av)
		return err
	})
}

func serializeAdaptiveSequence(seq Sequence, opts serializeOptions) (string, error) {
	parts := make([]string, 0, seqLen(seq))
	for item := range seqItems(seq) {
		s, err := serializeAdaptiveItem(item, opts)
		if err != nil {
			return "", err
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, opts.itemSeparator), nil
}

func serializeAdaptiveItem(item Item, opts serializeOptions) (string, error) {
	switch v := item.(type) {
	case AtomicValue:
		if v.TypeName == TypeBoolean {
			if v.BooleanVal() {
				return "true()", nil
			}
			return "false()", nil
		}
		s, err := atomicToString(v)
		if err != nil {
			return "", err
		}
		// Per XPath 3.1 §12.2 (Adaptive): string and untypedAtomic values
		// are serialized enclosed in double quotes, with internal quotes
		// escaped as "".
		if v.TypeName == TypeString || v.TypeName == TypeUntypedAtomic {
			escaped := strings.ReplaceAll(s, `"`, `""`)
			return `"` + escaped + `"`, nil
		}
		return s, nil
	case NodeItem:
		return serializeNodeItem(v, opts)
	case MapItem:
		return serializeAdaptiveMap(v, opts)
	case ArrayItem:
		return serializeAdaptiveArray(v, opts)
	case FunctionItem:
		return "", &XPathError{Code: errCodeFOER0000, Message: "cannot serialize function item"}
	default:
		return "", &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("cannot serialize %T", item)}
	}
}

func serializeAdaptiveMap(m MapItem, opts serializeOptions) (string, error) {
	parts := make([]string, 0, m.Size())
	err := m.ForEach(func(key AtomicValue, value Sequence) error {
		keyText, err := serializeAdaptiveItem(key, opts)
		if err != nil {
			return err
		}
		valText, err := serializeAdaptiveSequence(value, serializeOptions{method: "adaptive", itemSeparator: ","})
		if err != nil {
			return err
		}
		parts = append(parts, keyText+":"+valText)
		return nil
	})
	if err != nil {
		return "", err
	}
	return "map{" + strings.Join(parts, ",") + "}", nil
}

func serializeAdaptiveArray(a ArrayItem, opts serializeOptions) (string, error) {
	parts := make([]string, 0, a.Size())
	for _, member := range a.members0() {
		text, err := serializeAdaptiveSequence(member, serializeOptions{method: "adaptive", itemSeparator: ","})
		if err != nil {
			return "", err
		}
		parts = append(parts, text)
	}
	return "[" + strings.Join(parts, ",") + "]", nil
}

func serializeJSONSequence(seq Sequence, opts serializeOptions) (string, error) {
	if seqLen(seq) > 1 {
		return "", &XPathError{Code: errCodeFOER0000, Message: "json serialization requires at most one top-level item"}
	}
	if seqLen(seq) == 0 {
		return "null", nil
	}
	return serializeJSONItem(seq.Get(0), opts)
}

func serializeJSONItem(item Item, opts serializeOptions) (string, error) {
	switch v := item.(type) {
	case AtomicValue:
		return serializeJSONAtomic(v, opts)
	case ArrayItem:
		parts := make([]string, 0, v.Size())
		for _, member := range v.members0() {
			if seqLen(member) == 0 {
				parts = append(parts, "null")
				continue
			}
			for memberItem := range seqItems(member) {
				text, err := serializeJSONItem(memberItem, opts)
				if err != nil {
					return "", err
				}
				parts = append(parts, text)
			}
		}
		return formatJSONComposite("[", "]", parts, 0, opts.indent), nil
	case MapItem:
		seen := make(map[string]struct{}, v.Size())
		parts := make([]string, 0, v.Size())
		err := v.ForEach(func(key AtomicValue, value Sequence) error {
			keyText, err := atomicToString(key)
			if err != nil {
				return &XPathError{Code: errCodeFOER0000, Message: "json serialization map keys must be stringifiable"}
			}
			if _, exists := seen[keyText]; exists && !opts.allowDuplicateNames {
				return &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("json serialization duplicate key %q", keyText)}
			}
			seen[keyText] = struct{}{}
			if seqLen(value) > 1 {
				return &XPathError{Code: errCodeFOER0000, Message: "json serialization map values must be singleton or empty"}
			}
			valText := "null"
			if seqLen(value) == 1 {
				valText, err = serializeJSONItem(value.Get(0), opts)
				if err != nil {
					return err
				}
			}
			parts = append(parts, `"`+encodeJSONStringContent(keyText)+`":`+valueSeparator(opts.indent)+valText)
			return nil
		})
		if err != nil {
			return "", err
		}
		return formatJSONComposite("{", "}", parts, 0, opts.indent), nil
	case NodeItem:
		text, err := serializeNodeItem(v, serializeOptions{method: "xml", omitXMLDeclaration: true})
		if err != nil {
			return "", err
		}
		return `"` + encodeJSONStringContent(text) + `"`, nil
	case FunctionItem:
		return "", &XPathError{Code: errCodeFOER0000, Message: "cannot serialize function item as JSON"}
	default:
		return "", &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("cannot serialize %T as JSON", item)}
	}
}

func serializeJSONAtomic(v AtomicValue, opts serializeOptions) (string, error) {
	switch {
	case v.TypeName == TypeBoolean:
		if v.BooleanVal() {
			return "true", nil
		}
		return "false", nil
	case v.IsNumeric():
		s, err := atomicToString(v)
		if err != nil {
			return "", err
		}
		if s == "NaN" || s == "INF" || s == "-INF" {
			return "", &XPathError{Code: errCodeFOER0000, Message: "cannot serialize NaN or infinity as JSON"}
		}
		return s, nil
	default:
		s, err := atomicToString(v)
		if err != nil {
			return "", err
		}
		return `"` + encodeJSONStringForSerialization(s, opts.encoding) + `"`, nil
	}
}

func serializeXMLSequence(seq Sequence, opts serializeOptions) (string, error) {
	parts := make([]string, 0, seqLen(seq))
	for item := range seqItems(seq) {
		switch v := item.(type) {
		case FunctionItem:
			return "", &XPathError{Code: errCodeFOER0000, Message: "cannot serialize function item"}
		case NodeItem:
			text, err := serializeNodeItem(v, opts)
			if err != nil {
				return "", err
			}
			parts = append(parts, text)
		default:
			s, err := serializeAdaptiveItem(item, opts)
			if err != nil {
				return "", err
			}
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, opts.itemSeparator), nil
}

func serializeNodeItem(item NodeItem, opts serializeOptions) (string, error) {
	switch n := item.Node.(type) {
	case *helium.Attribute:
		return fmt.Sprintf(`%s="%s"`, n.Name(), n.Value()), nil
	case *helium.Document:
		writer := helium.NewWriter()
		if opts.omitXMLDeclaration {
			writer = writer.XMLDeclaration(false)
		}
		if opts.indent {
			writer = writer.Format(true)
		}
		var buf strings.Builder
		if err := writer.WriteTo(&buf, n); err != nil {
			return "", err
		}
		return strings.TrimSuffix(buf.String(), "\n"), nil
	default:
		var buf strings.Builder
		writer := helium.NewWriter()
		if opts.omitXMLDeclaration {
			writer = writer.XMLDeclaration(false)
		}
		if opts.indent {
			writer = writer.Format(true)
		}
		if err := writer.WriteTo(&buf, item.Node); err != nil {
			return "", err
		}
		return strings.TrimSuffix(buf.String(), "\n"), nil
	}
}

func encodeJSONStringForSerialization(s, encoding string) string {
	if encoding == "" || strings.EqualFold(encoding, "utf-8") || strings.EqualFold(encoding, "utf8") {
		return encodeJSONStringContent(s)
	}

	var b strings.Builder
	for _, r := range s {
		switch {
		case r <= 0x7F:
			appendSerializedJSONStringRune(&b, r)
		case r <= 0xFFFF:
			fmt.Fprintf(&b, `\u%04X`, r)
		default:
			hi, lo := utf16SurrogatePair(r)
			fmt.Fprintf(&b, `\u%04X\u%04X`, hi, lo)
		}
	}
	return b.String()
}

func utf16SurrogatePair(r rune) (uint16, uint16) {
	cp := uint32(r - 0x10000)
	hi := uint16(0xD800 + (cp >> 10))
	lo := uint16(0xDC00 + (cp & 0x3FF))
	return hi, lo
}
