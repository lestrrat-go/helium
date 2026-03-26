package xpath3

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/unparsedtext"
)

func init() {
	registerFn("parse-json", 1, 2, fnParseJSON)
	registerFn("json-doc", 1, 2, fnJSONDoc)
	registerFn("json-to-xml", 1, 2, fnJSONToXML)
	registerFn("xml-to-json", 1, 2, fnXMLToJSON)
	registerFn("serialize", 1, 2, fnSerialize)
}

// jsonOptions holds parsed options for fn:parse-json.
type jsonOptions struct {
	liberal    bool
	duplicates string // "reject", "use-first", "use-last" (default)
	escape     bool
	fallback   *FunctionItem
	invalidEsc map[rune]string
}

func fnParseJSON(ctx context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return nil, nil
	}
	s, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}

	opts, err := parseJSONOptions(args)
	if err != nil {
		return nil, err
	}

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

	// Check for trailing content after the JSON value
	var extra json.Token
	extra, err = dec.Token()
	if err == nil {
		return nil, &XPathError{Code: errCodeFOJS0001, Message: fmt.Sprintf("unexpected trailing content after JSON value: %v", extra)}
	}
	if !errors.Is(err, io.EOF) {
		return nil, &XPathError{Code: errCodeFOJS0001, Message: fmt.Sprintf("invalid trailing content: %v", err)}
	}
	if item == nil {
		return nil, nil
	}

	return ItemSlice{item}, nil
}

// parseJSONOptions extracts and validates options from the second argument.
func parseJSONOptions(args []Sequence) (jsonOptions, error) {
	opts := jsonOptions{
		duplicates: "use-first", // XPath spec default
	}
	if len(args) <= 1 || seqLen(args[1]) == 0 {
		return opts, nil
	}
	m, ok := args[1].Get(0).(MapItem)
	if !ok {
		return opts, nil
	}

	// Parse "liberal" option — must be xs:boolean
	liberalKey := AtomicValue{TypeName: TypeString, Value: "liberal"}
	if v, found := m.Get(liberalKey); found {
		if seqLen(v) != 1 {
			return opts, &XPathError{Code: errCodeFOJS0005, Message: "option 'liberal' must be a single xs:boolean"}
		}
		av, ok := v.Get(0).(AtomicValue)
		if !ok {
			return opts, &XPathError{Code: errCodeFOJS0005, Message: "option 'liberal' must be xs:boolean"}
		}
		if b, ok := av.Value.(bool); ok {
			opts.liberal = b
		} else {
			return opts, &XPathError{Code: errCodeFOJS0005,
				Message: "option 'liberal' must be xs:boolean, got " + av.TypeName}
		}
	}

	// Parse "duplicates" option
	dupKey := AtomicValue{TypeName: TypeString, Value: "duplicates"}
	if v, found := m.Get(dupKey); found {
		if seqLen(v) != 1 {
			return opts, &XPathError{Code: errCodeFOJS0005, Message: "option 'duplicates' must be a single string"}
		}
		av, ok := v.Get(0).(AtomicValue)
		if !ok {
			return opts, &XPathError{Code: errCodeFOJS0005, Message: "option 'duplicates' must be a string"}
		}
		s, _ := atomicToString(av)
		switch s {
		case "reject", "use-first", "use-last":
			opts.duplicates = s
		default:
			return opts, &XPathError{Code: errCodeFOJS0005,
				Message: fmt.Sprintf("invalid value for 'duplicates' option: %q", s)}
		}
	}

	// Parse "escape" option — must be xs:boolean
	escKey := AtomicValue{TypeName: TypeString, Value: "escape"}
	if v, found := m.Get(escKey); found {
		if seqLen(v) != 1 {
			return opts, &XPathError{Code: errCodeFOJS0005, Message: "option 'escape' must be a single xs:boolean"}
		}
		av, ok := v.Get(0).(AtomicValue)
		if !ok {
			return opts, &XPathError{Code: errCodeFOJS0005, Message: "option 'escape' must be xs:boolean"}
		}
		if b, ok := av.Value.(bool); ok {
			opts.escape = b
		} else {
			return opts, &XPathError{Code: errCodeFOJS0005,
				Message: "option 'escape' must be xs:boolean, got " + av.TypeName}
		}
	}

	// Parse "fallback" option — must be a function item
	fbKey := AtomicValue{TypeName: TypeString, Value: "fallback"}
	if v, found := m.Get(fbKey); found {
		if seqLen(v) != 1 {
			return opts, &XPathError{Code: errCodeFOJS0005,
				Message: "option 'fallback' must be a single function item"}
		}
		fi, ok := v.Get(0).(FunctionItem)
		if !ok {
			return opts, &XPathError{Code: errCodeFOJS0005,
				Message: "option 'fallback' must be a function item"}
		}
		if fi.Arity != 1 {
			return opts, &XPathError{Code: errCodeFOJS0005,
				Message: fmt.Sprintf("option 'fallback' must have arity 1, got %d", fi.Arity)}
		}
		opts.fallback = &fi
	}

	return opts, nil
}

func parseJSONValue(ctx context.Context, dec *json.Decoder, opts jsonOptions) (Item, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, &XPathError{Code: errCodeFOJS0001, Message: fmt.Sprintf("invalid JSON: %v", err)}
	}
	return parseJSONToken(ctx, tok, dec, opts)
}

func parseJSONToken(ctx context.Context, tok json.Token, dec *json.Decoder, opts jsonOptions) (Item, error) {
	switch v := tok.(type) {
	case json.Delim:
		switch v {
		case '{':
			entries := make([]MapEntry, 0)
			index := make(map[string]int)
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return nil, &XPathError{Code: errCodeFOJS0001, Message: fmt.Sprintf("invalid JSON: %v", err)}
				}
				key, ok := keyTok.(string)
				if !ok {
					return nil, &XPathError{Code: errCodeFOJS0001, Message: "invalid JSON object key"}
				}
				key, err = applyJSONStringOptions(ctx, key, opts)
				if err != nil {
					return nil, err
				}

				valueItem, err := parseJSONValue(ctx, dec, opts)
				if err != nil {
					return nil, err
				}
				var value Sequence
				if valueItem != nil {
					value = ItemSlice{valueItem}
				}

				if prev, found := index[key]; found {
					switch opts.duplicates {
					case "reject":
						return nil, &XPathError{Code: errCodeFOJS0003, Message: fmt.Sprintf("duplicate key in JSON object: %q", key)}
					case "use-first":
						continue
					case "use-last":
						entries[prev].Value = value
						continue
					}
				}

				index[key] = len(entries)
				entries = append(entries, MapEntry{
					Key:   AtomicValue{TypeName: TypeString, Value: key},
					Value: value,
				})
			}
			endTok, err := dec.Token()
			if err != nil {
				return nil, &XPathError{Code: errCodeFOJS0001, Message: fmt.Sprintf("invalid JSON: %v", err)}
			}
			if end, ok := endTok.(json.Delim); !ok || end != '}' {
				return nil, &XPathError{Code: errCodeFOJS0001, Message: "invalid JSON: expected object close delimiter"}
			}
			return NewMap(entries), nil
		case '[':
			members := make([]Sequence, 0)
			for dec.More() {
				item, err := parseJSONValue(ctx, dec, opts)
				if err != nil {
					return nil, err
				}
				if item == nil {
					members = append(members, nil)
					continue
				}
				members = append(members, ItemSlice{item})
			}
			endTok, err := dec.Token()
			if err != nil {
				return nil, &XPathError{Code: errCodeFOJS0001, Message: fmt.Sprintf("invalid JSON: %v", err)}
			}
			if end, ok := endTok.(json.Delim); !ok || end != ']' {
				return nil, &XPathError{Code: errCodeFOJS0001, Message: "invalid JSON: expected array close delimiter"}
			}
			return NewArray(members), nil
		default:
			return nil, &XPathError{Code: errCodeFOJS0001, Message: fmt.Sprintf("unexpected JSON delimiter: %q", v)}
		}
	default:
		if s, ok := v.(string); ok {
			s, err := applyJSONStringOptions(ctx, s, opts)
			if err != nil {
				return nil, err
			}
			return AtomicValue{TypeName: TypeString, Value: s}, nil
		}
		return jsonToXDM(v)
	}
}

func preprocessJSONStringLiterals(s string) (string, map[rune]string, error) {
	var b strings.Builder
	invalidEsc := make(map[rune]string)
	inString := false
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if !inString {
			b.WriteString(s[i : i+size])
			i += size
			if r == '"' {
				inString = true
			}
			continue
		}

		switch r {
		case '"':
			inString = false
			b.WriteRune(r)
			i += size
		case '\\':
			if i+1 >= len(s) {
				return "", nil, &XPathError{Code: errCodeFOJS0001, Message: "invalid JSON: trailing backslash in string"}
			}
			next := s[i+1]
			switch next {
			case '"', '\\', '/', 'n', 'r', 't':
				b.WriteString(s[i : i+2])
				i += 2
			case 'b', 'f':
				ph := nextJSONPlaceholderRune(s, invalidEsc)
				invalidEsc[ph] = s[i : i+2]
				b.WriteRune(ph)
				i += 2
			case 'u':
				if i+6 > len(s) {
					return "", nil, &XPathError{Code: errCodeFOJS0001, Message: "invalid JSON: incomplete \\u escape"}
				}
				escapeText := s[i : i+6]
				cp, err := parseJSONHexEscape(s[i+2 : i+6])
				if err != nil {
					return "", nil, err
				}
				switch {
				case cp >= 0xD800 && cp <= 0xDBFF:
					if i+12 <= len(s) && s[i+6] == '\\' && s[i+7] == 'u' {
						cp2, err := parseJSONHexEscape(s[i+8 : i+12])
						if err == nil && cp2 >= 0xDC00 && cp2 <= 0xDFFF {
							b.WriteString(s[i : i+12])
							i += 12
							continue
						}
					}
					ph := nextJSONPlaceholderRune(s, invalidEsc)
					invalidEsc[ph] = escapeText
					b.WriteRune(ph)
					i += 6
				case cp >= 0xDC00 && cp <= 0xDFFF, !isValidXMLCodepoint(int(cp)):
					ph := nextJSONPlaceholderRune(s, invalidEsc)
					invalidEsc[ph] = escapeText
					b.WriteRune(ph)
					i += 6
				default:
					b.WriteString(escapeText)
					i += 6
				}
			default:
				return "", nil, &XPathError{Code: errCodeFOJS0001, Message: fmt.Sprintf("invalid JSON: invalid escape sequence \\%c", next)}
			}
		default:
			if r < 0x20 {
				return "", nil, &XPathError{Code: errCodeFOJS0001, Message: "invalid JSON: unescaped control character in string"}
			}
			b.WriteString(s[i : i+size])
			i += size
		}
	}
	return b.String(), invalidEsc, nil
}

func parseJSONHexEscape(hex string) (uint32, error) {
	var cp uint32
	for _, c := range hex {
		cp <<= 4
		switch {
		case c >= '0' && c <= '9':
			cp += uint32(c - '0')
		case c >= 'a' && c <= 'f':
			cp += uint32(c-'a') + 10
		case c >= 'A' && c <= 'F':
			cp += uint32(c-'A') + 10
		default:
			return 0, &XPathError{Code: errCodeFOJS0001, Message: fmt.Sprintf("invalid JSON: invalid hex digit in \\u escape: %c", c)}
		}
	}
	return cp, nil
}

func nextJSONPlaceholderRune(source string, used map[rune]string) rune {
	for r := rune(0xF0000); r <= 0xFFFFD; r++ {
		if _, exists := used[r]; exists {
			continue
		}
		if !strings.ContainsRune(source, r) {
			return r
		}
	}
	panic("exhausted JSON placeholder runes")
}

func applyJSONStringOptions(ctx context.Context, s string, opts jsonOptions) (string, error) {
	if opts.escape {
		var b strings.Builder
		for _, r := range s {
			if raw, found := opts.invalidEsc[r]; found {
				if opts.fallback != nil {
					return "", &XPathError{Code: errCodeFOJS0001, Message: "fallback cannot be used with escape=true() for escaped invalid characters"}
				}
				b.WriteString(canonicalInvalidJSONEscape(raw))
				continue
			}
			appendEscapedJSONStringRune(&b, r)
		}
		return b.String(), nil
	}

	var b strings.Builder
	for _, r := range s {
		raw, found := opts.invalidEsc[r]
		if !found {
			b.WriteRune(r)
			continue
		}

		repl := string(rune(0xFFFD))
		if opts.fallback != nil {
			seq, err := opts.fallback.Invoke(ctx, []Sequence{ItemSlice{AtomicValue{TypeName: TypeString, Value: raw}}})
			if err != nil {
				return "", err
			}
			repl, err = seqToStringErr(seq)
			if err != nil {
				return "", err
			}
		}
		b.WriteString(repl)
	}
	return b.String(), nil
}

func appendEscapedJSONStringRune(b *strings.Builder, r rune) {
	switch r {
	case '\\':
		b.WriteString(`\\`)
	case '\b':
		b.WriteString(`\b`)
	case '\f':
		b.WriteString(`\f`)
	case '\n':
		b.WriteString(`\n`)
	case '\r':
		b.WriteString(`\r`)
	case '\t':
		b.WriteString(`\t`)
	default:
		if (r >= 0x00 && r <= 0x1F) || !isValidXMLCodepoint(int(r)) {
			fmt.Fprintf(b, `\u%04x`, r)
			return
		}
		b.WriteRune(r)
	}
}

func appendSerializedJSONStringRune(b *strings.Builder, r rune) {
	switch r {
	case '"':
		b.WriteString(`\"`)
	case '\\':
		b.WriteString(`\\`)
	case '/':
		b.WriteString(`\/`)
	case '\b':
		b.WriteString(`\b`)
	case '\f':
		b.WriteString(`\f`)
	case '\n':
		b.WriteString(`\n`)
	case '\r':
		b.WriteString(`\r`)
	case '\t':
		b.WriteString(`\t`)
	default:
		if (r >= 0x00 && r <= 0x1F) || (r >= 0x7F && r <= 0x9F) || !isValidXMLCodepoint(int(r)) {
			fmt.Fprintf(b, `\u%04X`, r)
			return
		}
		b.WriteRune(r)
	}
}

func canonicalInvalidJSONEscape(raw string) string {
	switch {
	case strings.EqualFold(raw, `\u0008`):
		return `\b`
	case strings.EqualFold(raw, `\u000c`):
		return `\f`
	default:
		return raw
	}
}

func fnJSONDoc(ctx context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return nil, nil
	}
	uri, err := coerceArgToString(args[0])
	if err != nil {
		return nil, err
	}

	cfg := unparsedTextConfig(ctx)
	resolvedURI, err := unparsedtext.ResolveURI(ctx, cfg, uri)
	if err != nil {
		return nil, &XPathError{Code: errCodeFODC0002, Message: fmt.Sprintf("json-doc: cannot resolve URI: %v", err)}
	}
	body, err := unparsedtext.ReadURI(ctx, cfg, resolvedURI)
	if err != nil {
		return nil, &XPathError{Code: errCodeFODC0002, Message: fmt.Sprintf("json-doc: cannot retrieve resource: %v", err)}
	}

	// Delegate to parse-json logic, preserving any caller-supplied options.
	parseArgs := []Sequence{ItemSlice{AtomicValue{TypeName: TypeString, Value: string(body)}}}
	if len(args) > 1 {
		parseArgs = append(parseArgs, args[1])
	}
	return fnParseJSON(ctx, parseArgs)
}

func fnJSONToXML(ctx context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return nil, nil
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
			name = "string"
		case TypeBoolean:
			name = "boolean"
		default:
			name = "number"
		}
	}

	elem, err := doc.CreateElement(name)
	if err != nil {
		return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("json-to-xml: failed to build result: %v", err)}
	}
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
			child.SetLiteralAttribute("key", keyText)
			if opts.escape {
				child.SetLiteralAttribute("escaped-key", "true")
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
				elem.SetLiteralAttribute("escaped", "true")
			}
			if err := elem.AppendText([]byte(v.StringVal())); err != nil {
				return nil, &XPathError{Code: errCodeFOER0000, Message: fmt.Sprintf("json-to-xml: failed to append string value: %v", err)}
			}
		case TypeBoolean:
			text := "false"
			if v.BooleanVal() {
				text = "true"
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
		return nil, nil
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
	case "string":
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
	case "map", "array", "string", "number", "boolean", "null":
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
	case "true", "1":
		return true, nil
	case "false", "0":
		return false, nil
	default:
		return false, &XPathError{Code: errCodeFOJS0006, Message: fmt.Sprintf("xml-to-json: invalid boolean attribute value %q", s)}
	}
}

func canonicalizeXMLJSONBoolean(s string) (string, error) {
	switch strings.TrimSpace(s) {
	case "true", "1":
		return "true", nil
	case "false", "0":
		return "false", nil
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
	if idx := strings.IndexByte(s, '.'); idx >= 0 {
		intPart = s[:idx]
		fracPart = s[idx+1:]
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

func formatJSONComposite(open, close string, parts []string, depth int, indent bool) string {
	if !indent || len(parts) == 0 {
		return open + strings.Join(parts, ",") + close
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
	b.WriteString(close)
	return b.String()
}

func valueSeparator(indent bool) string {
	if indent {
		return " "
	}
	return ""
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
		s, err := n.XMLString(writer)
		if err != nil {
			return "", err
		}
		return strings.TrimSuffix(s, "\n"), nil
	default:
		var buf bytes.Buffer
		writer := helium.NewWriter()
		if opts.omitXMLDeclaration {
			writer = writer.XMLDeclaration(false)
		}
		if opts.indent {
			writer = writer.Format(true)
		}
		if err := writer.WriteNode(&buf, item.Node); err != nil {
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

func jsonToXDM(v any) (Item, error) {
	switch val := v.(type) {
	case nil:
		return nil, nil //nolint:nilnil // JSON null maps to empty sequence (nil Item, nil error) per XPath spec
	case bool:
		return AtomicValue{TypeName: TypeBoolean, Value: val}, nil
	case json.Number:
		s := val.String()
		if strings.ContainsAny(s, ".eE") {
			f, err := val.Float64()
			if err != nil {
				return nil, &XPathError{Code: errCodeFOJS0001, Message: "invalid JSON number: " + s}
			}
			return AtomicValue{TypeName: TypeDouble, Value: NewDouble(f)}, nil
		}
		n, ok := new(big.Int).SetString(s, 10)
		if !ok {
			f, err := val.Float64()
			if err != nil {
				return nil, &XPathError{Code: errCodeFOJS0001, Message: "invalid JSON number: " + s}
			}
			return AtomicValue{TypeName: TypeDouble, Value: NewDouble(f)}, nil
		}
		return AtomicValue{TypeName: TypeInteger, Value: n}, nil
	case string:
		return AtomicValue{TypeName: TypeString, Value: val}, nil
	case []any:
		members := make([]Sequence, len(val))
		for i, elem := range val {
			item, err := jsonToXDM(elem)
			if err != nil {
				return nil, err
			}
			if item == nil {
				members[i] = nil // JSON null → empty sequence
			} else {
				members[i] = ItemSlice{item}
			}
		}
		return NewArray(members), nil
	case map[string]any:
		entries := make([]MapEntry, 0, len(val))
		for k, v := range val {
			item, err := jsonToXDM(v)
			if err != nil {
				return nil, err
			}
			var value Sequence
			if item == nil {
				value = nil // JSON null → empty sequence
			} else {
				value = ItemSlice{item}
			}
			entries = append(entries, MapEntry{
				Key:   AtomicValue{TypeName: TypeString, Value: k},
				Value: value,
			})
		}
		return NewMap(entries), nil
	default:
		return nil, &XPathError{Code: errCodeFOJS0001, Message: fmt.Sprintf("unexpected JSON type: %T", v)}
	}
}
