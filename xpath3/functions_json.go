package xpath3

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
	"unicode/utf8"
)

func init() {
	registerFn("parse-json", 1, 2, fnParseJSON)
	registerFn("json-doc", 1, 2, fnJSONDoc)
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
	if len(args[0]) == 0 {
		return nil, nil
	}
	s := seqToString(args[0])

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

	return Sequence{item}, nil
}

// parseJSONOptions extracts and validates options from the second argument.
func parseJSONOptions(args []Sequence) (jsonOptions, error) {
	opts := jsonOptions{
		duplicates: "use-first", // XPath spec default
	}
	if len(args) <= 1 || len(args[1]) == 0 {
		return opts, nil
	}
	m, ok := args[1][0].(MapItem)
	if !ok {
		return opts, nil
	}

	// Parse "liberal" option — must be xs:boolean
	liberalKey := AtomicValue{TypeName: TypeString, Value: "liberal"}
	if v, found := m.Get(liberalKey); found {
		if len(v) != 1 {
			return opts, &XPathError{Code: errCodeFOJS0005, Message: "option 'liberal' must be a single xs:boolean"}
		}
		av, ok := v[0].(AtomicValue)
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
		if len(v) != 1 {
			return opts, &XPathError{Code: errCodeFOJS0005, Message: "option 'duplicates' must be a single string"}
		}
		av, ok := v[0].(AtomicValue)
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
		if len(v) != 1 {
			return opts, &XPathError{Code: errCodeFOJS0005, Message: "option 'escape' must be a single xs:boolean"}
		}
		av, ok := v[0].(AtomicValue)
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
		if len(v) != 1 {
			return opts, &XPathError{Code: errCodeFOJS0005,
				Message: "option 'fallback' must be a single function item"}
		}
		fi, ok := v[0].(FunctionItem)
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
					value = Sequence{valueItem}
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
				members = append(members, Sequence{item})
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

func escapeParsedJSONString(s string) string {
	var b strings.Builder
	for _, r := range s {
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
			if (r >= 0x00 && r <= 0x1F) || (r >= 0x7F && r <= 0x9F) || !isValidXMLCodepoint(int(r)) {
				fmt.Fprintf(&b, `\u%04x`, r)
				continue
			}
			b.WriteRune(r)
		}
	}
	return b.String()
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
			seq, err := opts.fallback.Invoke(ctx, []Sequence{{AtomicValue{TypeName: TypeString, Value: raw}}})
			if err != nil {
				return "", err
			}
			repl = seqToString(seq)
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
	if len(args[0]) == 0 {
		return nil, nil
	}
	uri := seqToString(args[0])

	ec := getFnContext(ctx)
	if ec == nil || ec.httpClient == nil {
		return nil, &XPathError{Code: errCodeFODC0002, Message: "json-doc: no HTTP client configured for URI: " + uri}
	}

	resp, err := ec.httpClient.Get(uri)
	if err != nil {
		return nil, &XPathError{Code: errCodeFODC0002, Message: fmt.Sprintf("json-doc: failed to fetch %s: %v", uri, err)}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		return nil, &XPathError{Code: errCodeFODC0002, Message: fmt.Sprintf("json-doc: HTTP %d for %s", resp.StatusCode, uri)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &XPathError{Code: errCodeFODC0002, Message: fmt.Sprintf("json-doc: error reading %s: %v", uri, err)}
	}

	// Delegate to parse-json logic, preserving any caller-supplied options.
	parseArgs := []Sequence{{AtomicValue{TypeName: TypeString, Value: string(body)}}}
	if len(args) > 1 {
		parseArgs = append(parseArgs, args[1])
	}
	return fnParseJSON(ctx, parseArgs)
}

func fnSerialize(_ context.Context, args []Sequence) (Sequence, error) {
	seq := args[0]
	var parts []string
	for _, item := range seq {
		parts = append(parts, serializeItem(item))
	}
	return SingleString(strings.Join(parts, " ")), nil
}

func serializeItem(item Item) string {
	switch v := item.(type) {
	case AtomicValue:
		s, err := atomicToString(v)
		if err != nil {
			return fmt.Sprintf("%v", v.Value)
		}
		return s
	case NodeItem:
		a, err := AtomizeItem(v)
		if err != nil {
			return ""
		}
		s, _ := atomicToString(a)
		return s
	case ArrayItem:
		var parts []string
		for _, m := range v.Members() {
			var mParts []string
			for _, mi := range m {
				mParts = append(mParts, serializeItem(mi))
			}
			parts = append(parts, strings.Join(mParts, " "))
		}
		return "[" + strings.Join(parts, ",") + "]"
	case MapItem:
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", item)
	}
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
				members[i] = Sequence{item}
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
				value = Sequence{item}
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
