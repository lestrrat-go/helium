package xpath3

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

func init() {
	registerFn("parse-json", 1, 2, fnParseJSON)
	registerFn("json-doc", 1, 2, fnJSONDoc)
	registerFn("serialize", 1, 2, fnSerialize)
}

func fnParseJSON(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	s := seqToString(args[0])

	// Parse options from second argument (map) if present
	liberal := false
	if len(args) > 1 && len(args[1]) > 0 {
		if m, ok := args[1][0].(MapItem); ok {
			liberalKey := AtomicValue{TypeName: TypeString, Value: "liberal"}
			if v, found := m.Get(liberalKey); found && len(v) > 0 {
				if av, ok := v[0].(AtomicValue); ok {
					if b, ok := av.Value.(bool); ok {
						liberal = b
					}
				}
			}
		}
	}
	_ = liberal // TODO: liberal mode is parsed but not yet implemented

	var raw interface{}
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		return nil, &XPathError{Code: "FOJS0001", Message: fmt.Sprintf("invalid JSON: %v", err)}
	}

	item, err := jsonToXDM(raw)
	if err != nil {
		return nil, err
	}
	return Sequence{item}, nil
}

func fnJSONDoc(_ context.Context, _ []Sequence) (Sequence, error) {
	return nil, &XPathError{Code: "FODC0002", Message: "json-doc: URI resolution not supported"}
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

func jsonToXDM(v interface{}) (Item, error) {
	switch val := v.(type) {
	case nil:
		return nil, nil
	case bool:
		return AtomicValue{TypeName: TypeBoolean, Value: val}, nil
	case json.Number:
		s := val.String()
		if strings.ContainsAny(s, ".eE") {
			f, err := val.Float64()
			if err != nil {
				return nil, &XPathError{Code: "FOJS0001", Message: fmt.Sprintf("invalid JSON number: %s", s)}
			}
			return AtomicValue{TypeName: TypeDouble, Value: f}, nil
		}
		n, ok := new(big.Int).SetString(s, 10)
		if !ok {
			f, err := val.Float64()
			if err != nil {
				return nil, &XPathError{Code: "FOJS0001", Message: fmt.Sprintf("invalid JSON number: %s", s)}
			}
			return AtomicValue{TypeName: TypeDouble, Value: f}, nil
		}
		return AtomicValue{TypeName: TypeInteger, Value: n}, nil
	case string:
		return AtomicValue{TypeName: TypeString, Value: val}, nil
	case []interface{}:
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
	case map[string]interface{}:
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
