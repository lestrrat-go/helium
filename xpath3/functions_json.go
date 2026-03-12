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
}

func fnParseJSON(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	s := seqToString(args[0])

	opts, err := parseJSONOptions(args)
	if err != nil {
		return nil, err
	}

	// Pre-validate the JSON string for issues Go's decoder doesn't catch
	if err := validateJSONString(s); err != nil {
		return nil, err
	}

	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	item, err := parseJSONValue(dec, opts)
	if err != nil {
		return nil, err
	}

	// Check for trailing content after the JSON value
	var extra json.Token
	extra, err = dec.Token()
	if err == nil {
		return nil, &XPathError{Code: "FOJS0001", Message: fmt.Sprintf("unexpected trailing content after JSON value: %v", extra)}
	}
	if !errors.Is(err, io.EOF) {
		return nil, &XPathError{Code: "FOJS0001", Message: fmt.Sprintf("invalid trailing content: %v", err)}
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
			return opts, &XPathError{Code: "FOJS0005", Message: "option 'liberal' must be a single xs:boolean"}
		}
		av, ok := v[0].(AtomicValue)
		if !ok {
			return opts, &XPathError{Code: "FOJS0005", Message: "option 'liberal' must be xs:boolean"}
		}
		if b, ok := av.Value.(bool); ok {
			opts.liberal = b
		} else {
			return opts, &XPathError{Code: "FOJS0005",
				Message: "option 'liberal' must be xs:boolean, got " + av.TypeName}
		}
	}

	// Parse "duplicates" option
	dupKey := AtomicValue{TypeName: TypeString, Value: "duplicates"}
	if v, found := m.Get(dupKey); found {
		if len(v) != 1 {
			return opts, &XPathError{Code: "FOJS0005", Message: "option 'duplicates' must be a single string"}
		}
		av, ok := v[0].(AtomicValue)
		if !ok {
			return opts, &XPathError{Code: "FOJS0005", Message: "option 'duplicates' must be a string"}
		}
		s, _ := atomicToString(av)
		switch s {
		case "reject", "use-first", "use-last":
			opts.duplicates = s
		default:
			return opts, &XPathError{Code: "FOJS0005",
				Message: fmt.Sprintf("invalid value for 'duplicates' option: %q", s)}
		}
	}

	// Parse "escape" option — must be xs:boolean
	escKey := AtomicValue{TypeName: TypeString, Value: "escape"}
	if v, found := m.Get(escKey); found {
		if len(v) != 1 {
			return opts, &XPathError{Code: "FOJS0005", Message: "option 'escape' must be a single xs:boolean"}
		}
		av, ok := v[0].(AtomicValue)
		if !ok {
			return opts, &XPathError{Code: "FOJS0005", Message: "option 'escape' must be xs:boolean"}
		}
		if b, ok := av.Value.(bool); ok {
			opts.escape = b
		} else {
			return opts, &XPathError{Code: "FOJS0005",
				Message: "option 'escape' must be xs:boolean, got " + av.TypeName}
		}
	}

	// Parse "fallback" option — must be a function item
	fbKey := AtomicValue{TypeName: TypeString, Value: "fallback"}
	if v, found := m.Get(fbKey); found {
		if len(v) != 1 {
			return opts, &XPathError{Code: "FOJS0005",
				Message: "option 'fallback' must be a single function item"}
		}
		fi, ok := v[0].(FunctionItem)
		if !ok {
			return opts, &XPathError{Code: "FOJS0005",
				Message: "option 'fallback' must be a function item"}
		}
		if fi.Arity != 1 {
			return opts, &XPathError{Code: "FOJS0005",
				Message: fmt.Sprintf("option 'fallback' must have arity 1, got %d", fi.Arity)}
		}
		// We accept the function but don't use it yet.
	}

	return opts, nil
}

func parseJSONValue(dec *json.Decoder, opts jsonOptions) (Item, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, &XPathError{Code: "FOJS0001", Message: fmt.Sprintf("invalid JSON: %v", err)}
	}
	return parseJSONToken(tok, dec, opts)
}

func parseJSONToken(tok json.Token, dec *json.Decoder, opts jsonOptions) (Item, error) {
	switch v := tok.(type) {
	case json.Delim:
		switch v {
		case '{':
			entries := make([]MapEntry, 0)
			index := make(map[string]int)
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return nil, &XPathError{Code: "FOJS0001", Message: fmt.Sprintf("invalid JSON: %v", err)}
				}
				key, ok := keyTok.(string)
				if !ok {
					return nil, &XPathError{Code: "FOJS0001", Message: "invalid JSON object key"}
				}
				if opts.escape {
					key = escapeParsedJSONString(key)
				}

				valueItem, err := parseJSONValue(dec, opts)
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
						return nil, &XPathError{Code: "FOJS0003", Message: fmt.Sprintf("duplicate key in JSON object: %q", key)}
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
				return nil, &XPathError{Code: "FOJS0001", Message: fmt.Sprintf("invalid JSON: %v", err)}
			}
			if end, ok := endTok.(json.Delim); !ok || end != '}' {
				return nil, &XPathError{Code: "FOJS0001", Message: "invalid JSON: expected object close delimiter"}
			}
			return NewMap(entries), nil
		case '[':
			members := make([]Sequence, 0)
			for dec.More() {
				item, err := parseJSONValue(dec, opts)
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
				return nil, &XPathError{Code: "FOJS0001", Message: fmt.Sprintf("invalid JSON: %v", err)}
			}
			if end, ok := endTok.(json.Delim); !ok || end != ']' {
				return nil, &XPathError{Code: "FOJS0001", Message: "invalid JSON: expected array close delimiter"}
			}
			return NewArray(members), nil
		default:
			return nil, &XPathError{Code: "FOJS0001", Message: fmt.Sprintf("unexpected JSON delimiter: %q", v)}
		}
	default:
		if s, ok := v.(string); ok {
			if opts.escape {
				s = escapeParsedJSONString(s)
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

// validateJSONString pre-validates JSON for issues Go's decoder doesn't catch.
func validateJSONString(s string) *XPathError {
	// Check for invalid escape sequences in JSON strings
	inString := false
	escaped := false
	i := 0
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if escaped {
			escaped = false
			if inString {
				switch r {
				case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
					// valid escapes
				case 'u':
					// validate 4 hex digits and check for surrogates
					if i+size+4 > len(s) {
						return &XPathError{Code: "FOJS0001", Message: "invalid JSON: incomplete \\u escape"}
					}
					hex := s[i+size : i+size+4]
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
							return &XPathError{Code: "FOJS0001", Message: fmt.Sprintf("invalid JSON: invalid hex digit in \\u escape: %c", c)}
						}
					}
					// Check for lone surrogates
					if cp >= 0xD800 && cp <= 0xDBFF {
						// High surrogate — check for low surrogate pair
						afterHex := i + size + 4
						if afterHex+6 <= len(s) && s[afterHex] == '\\' && s[afterHex+1] == 'u' {
							hex2 := s[afterHex+2 : afterHex+6]
							var cp2 uint32
							valid := true
							for _, c := range hex2 {
								cp2 <<= 4
								switch {
								case c >= '0' && c <= '9':
									cp2 += uint32(c - '0')
								case c >= 'a' && c <= 'f':
									cp2 += uint32(c-'a') + 10
								case c >= 'A' && c <= 'F':
									cp2 += uint32(c-'A') + 10
								default:
									valid = false
								}
							}
							if !valid || cp2 < 0xDC00 || cp2 > 0xDFFF {
								return &XPathError{Code: "FOJS0001", Message: "invalid JSON: lone high surrogate in \\u escape"}
							}
							// Skip past the low surrogate
							i = afterHex + 6
							continue
						}
						return &XPathError{Code: "FOJS0001", Message: "invalid JSON: lone high surrogate in \\u escape"}
					}
					if cp >= 0xDC00 && cp <= 0xDFFF {
						return &XPathError{Code: "FOJS0001", Message: "invalid JSON: lone low surrogate in \\u escape"}
					}
					// Check for \u0000 (null character not valid in XPath strings)
					if cp == 0 {
						return &XPathError{Code: "FOJS0001", Message: "invalid JSON: null character (\\u0000) not valid in XPath strings"}
					}
				default:
					return &XPathError{Code: "FOJS0001", Message: fmt.Sprintf("invalid JSON: invalid escape sequence \\%c", r)}
				}
			}
			i += size
			continue
		}

		if r == '\\' && inString {
			escaped = true
			i += size
			continue
		}
		if r == '"' {
			inString = !inString
		}
		i += size
	}
	return nil
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
				return nil, &XPathError{Code: "FOJS0001", Message: "invalid JSON number: " + s}
			}
			return AtomicValue{TypeName: TypeDouble, Value: NewDouble(f)}, nil
		}
		n, ok := new(big.Int).SetString(s, 10)
		if !ok {
			f, err := val.Float64()
			if err != nil {
				return nil, &XPathError{Code: "FOJS0001", Message: "invalid JSON number: " + s}
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
		return nil, &XPathError{Code: "FOJS0001", Message: fmt.Sprintf("unexpected JSON type: %T", v)}
	}
}
