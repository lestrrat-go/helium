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

	var raw any
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		return nil, &XPathError{Code: "FOJS0001", Message: fmt.Sprintf("invalid JSON: %v", err)}
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

	// Handle duplicate keys if needed
	if opts.duplicates == "reject" {
		if err := checkDuplicateKeys(s); err != nil {
			return nil, err
		}
	}

	item, err := jsonToXDM(raw)
	if err != nil {
		return nil, err
	}

	// Validate string values don't contain control characters
	if err := validateJSONResultStrings(item); err != nil {
		return nil, err
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
	if v, found := m.Get(liberalKey); found && len(v) > 0 {
		if av, ok := v[0].(AtomicValue); ok {
			if b, ok := av.Value.(bool); ok {
				opts.liberal = b
			} else {
				return opts, &XPathError{Code: "FOJS0005",
					Message: "option 'liberal' must be xs:boolean, got " + av.TypeName}
			}
		}
	}

	// Parse "duplicates" option
	dupKey := AtomicValue{TypeName: TypeString, Value: "duplicates"}
	if v, found := m.Get(dupKey); found && len(v) > 0 {
		if av, ok := v[0].(AtomicValue); ok {
			s, _ := atomicToString(av)
			switch s {
			case "reject", "use-first", "use-last":
				opts.duplicates = s
			default:
				return opts, &XPathError{Code: "FOJS0005",
					Message: fmt.Sprintf("invalid value for 'duplicates' option: %q", s)}
			}
		}
	}

	// Parse "escape" option — must be xs:boolean
	escKey := AtomicValue{TypeName: TypeString, Value: "escape"}
	if v, found := m.Get(escKey); found && len(v) > 0 {
		if av, ok := v[0].(AtomicValue); ok {
			if b, ok := av.Value.(bool); ok {
				opts.escape = b
			} else {
				return opts, &XPathError{Code: "FOJS0005",
					Message: "option 'escape' must be xs:boolean, got " + av.TypeName}
			}
		}
	}

	// Parse "fallback" option — must be a function item
	fbKey := AtomicValue{TypeName: TypeString, Value: "fallback"}
	if v, found := m.Get(fbKey); found && len(v) > 0 {
		if _, ok := v[0].(FunctionItem); !ok {
			return opts, &XPathError{Code: "FOJS0005",
				Message: "option 'fallback' must be a function item"}
		}
		// We accept the function but don't use it yet
	}

	return opts, nil
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

// checkDuplicateKeys recursively checks JSON for duplicate object keys.
func checkDuplicateKeys(s string) *XPathError {
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	return checkDuplicateKeysDecoder(dec)
}

func checkDuplicateKeysDecoder(dec *json.Decoder) *XPathError {
	tok, err := dec.Token()
	if err != nil {
		return nil //nolint:nilerr // decoder error means no more tokens; return no duplicate error
	}
	if v, ok := tok.(json.Delim); ok {
		switch v {
		case '{':
			keys := make(map[string]bool)
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return nil //nolint:nilerr // decoder error means no more tokens; return no duplicate error
				}
				key, ok := keyTok.(string)
				if !ok {
					return nil
				}
				if keys[key] {
					return &XPathError{Code: "FOJS0003",
						Message: fmt.Sprintf("duplicate key in JSON object: %q", key)}
				}
				keys[key] = true
				// Recursively check the value
				if xerr := checkDuplicateKeysDecoder(dec); xerr != nil {
					return xerr
				}
			}
			// Consume closing '}'
			dec.Token() //nolint:errcheck
		case '[':
			for dec.More() {
				if xerr := checkDuplicateKeysDecoder(dec); xerr != nil {
					return xerr
				}
			}
			// Consume closing ']'
			dec.Token() //nolint:errcheck
		}
	}
	return nil
}

// validateJSONResultStrings checks that string values in the result don't
// contain XML-invalid control characters (U+0000..U+001F except U+0009, U+000A, U+000D).
func validateJSONResultStrings(item Item) *XPathError {
	if item == nil {
		return nil
	}
	switch v := item.(type) {
	case AtomicValue:
		if v.TypeName == TypeString {
			s, ok := v.Value.(string)
			if ok {
				for _, r := range s {
					if r < 0x20 && r != 0x09 && r != 0x0A && r != 0x0D {
						return &XPathError{Code: "FOJS0001",
							Message: fmt.Sprintf("JSON string contains control character U+%04X", r)}
					}
					// Check for replacement character (from surrogates)
					// U+FFFF and U+FFFE are not valid XML characters
				}
			}
		}
	case ArrayItem:
		for i := 1; i <= v.Size(); i++ {
			members, _ := v.Get(i)
			for _, m := range members {
				if xerr := validateJSONResultStrings(m); xerr != nil {
					return xerr
				}
			}
		}
	case MapItem:
		for _, k := range v.Keys() {
			// Check key strings too
			if xerr := validateJSONResultStrings(k); xerr != nil {
				return xerr
			}
			val, _ := v.Get(k)
			for _, vi := range val {
				if xerr := validateJSONResultStrings(vi); xerr != nil {
					return xerr
				}
			}
		}
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
