package xpath3

import (
	"context"
	"fmt"
)

func init() {
	registerNS(NSMap, "merge", 1, 2, fnMapMerge)
	registerNS(NSMap, "size", 1, 1, fnMapSize)
	registerNS(NSMap, "keys", 1, 1, fnMapKeys)
	registerNS(NSMap, "contains", 2, 2, fnMapContains)
	registerNS(NSMap, "get", 2, 2, fnMapGet)
	registerNS(NSMap, "put", 3, 3, fnMapPut)
	registerNS(NSMap, "entry", 2, 2, fnMapEntry)
	registerNS(NSMap, "remove", 2, 2, fnMapRemove)
	registerNS(NSMap, "for-each", 2, 2, fnMapForEach)
	registerNS(NSMap, "find", 2, 2, fnMapFind)
}

func extractMap(seq Sequence) (MapItem, error) {
	if seqLen(seq) != 1 {
		return MapItem{}, &XPathError{Code: errCodeXPTY0004, Message: "expected single map"}
	}
	m, ok := seq.Get(0).(MapItem)
	if !ok {
		return MapItem{}, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("expected map, got %T", seq.Get(0))}
	}
	return m, nil
}

func fnMapMerge(_ context.Context, args []Sequence) (Sequence, error) {
	duplicates := MergeUseFirst
	if len(args) > 1 {
		if seqLen(args[1]) == 0 {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "map:merge: options argument must be a map, got empty sequence"}
		}
		// The options map should contain "duplicates" key
		optMap, ok := args[1].Get(0).(MapItem)
		if ok {
			key := AtomicValue{TypeName: TypeString, Value: "duplicates"}
			if val, found := optMap.Get(key); found {
				s, err := coerceArgToStringRequired(val)
				if err != nil {
					return nil, err
				}
				switch s {
				case "use-first":
					duplicates = MergeUseFirst
				case "use-last":
					duplicates = MergeUseLast
				case "reject":
					duplicates = MergeReject
				case "combine":
					duplicates = MergeCombine
				}
			}
		}
	}
	builder := NewMapBuilder(duplicates, seqLen(args[0]))
	for item := range seqItems(args[0]) {
		m, ok := item.(MapItem)
		if !ok {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "map:merge requires sequence of maps"}
		}
		mergeErr := m.ForEach(func(k AtomicValue, v Sequence) error {
			return builder.Add(k, v)
		})
		if mergeErr != nil {
			return nil, mergeErr
		}
	}
	return ItemSlice{builder.Build()}, nil
}

func fnMapSize(_ context.Context, args []Sequence) (Sequence, error) {
	m, err := extractMap(args[0])
	if err != nil {
		return nil, err
	}
	return SingleInteger(int64(m.Size())), nil
}

func fnMapKeys(_ context.Context, args []Sequence) (Sequence, error) {
	m, err := extractMap(args[0])
	if err != nil {
		return nil, err
	}
	keys := m.Keys()
	result := make(ItemSlice, len(keys))
	for i, k := range keys {
		result[i] = k
	}
	return result, nil
}

func fnMapContains(_ context.Context, args []Sequence) (Sequence, error) {
	m, err := extractMap(args[0])
	if err != nil {
		return nil, err
	}
	ka, err := extractSingleAtomicArg(args[1], "map:contains")
	if err != nil {
		return nil, err
	}
	return SingleBoolean(m.Contains(ka)), nil
}

func fnMapGet(_ context.Context, args []Sequence) (Sequence, error) {
	m, err := extractMap(args[0])
	if err != nil {
		return nil, err
	}
	if seqLen(args[1]) == 0 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "map:get requires a key argument"}
	}
	if seqLen(args[1]) > 1 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "map:get key must be a single atomic value"}
	}
	ka, err := AtomizeItem(args[1].Get(0))
	if err != nil {
		return nil, err
	}
	val, ok := m.Get(ka)
	if !ok {
		return nil, nil //nolint:nilnil
	}
	return val, nil
}

func fnMapPut(_ context.Context, args []Sequence) (Sequence, error) {
	m, err := extractMap(args[0])
	if err != nil {
		return nil, err
	}
	if seqLen(args[1]) == 0 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "map:put requires key"}
	}
	ka, err := AtomizeItem(args[1].Get(0))
	if err != nil {
		return nil, err
	}
	return ItemSlice{m.Put(ka, args[2])}, nil
}

func fnMapEntry(_ context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) == 0 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "map:entry requires key"}
	}
	ka, err := AtomizeItem(args[0].Get(0))
	if err != nil {
		return nil, err
	}
	return ItemSlice{newSingleEntryMap(ka, args[1])}, nil
}

func fnMapRemove(_ context.Context, args []Sequence) (Sequence, error) {
	m, err := extractMap(args[0])
	if err != nil {
		return nil, err
	}
	for item := range seqItems(args[1]) {
		ka, err := AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		m = m.Remove(ka)
	}
	return ItemSlice{m}, nil
}

func fnMapForEach(ctx context.Context, args []Sequence) (Sequence, error) {
	m, err := extractMap(args[0])
	if err != nil {
		return nil, err
	}
	fi, err := extractFunctionItem(args[1])
	if err != nil {
		return nil, err
	}
	var result ItemSlice
	mapErr := m.ForEach(func(k AtomicValue, v Sequence) error {
		r, err := fi.Invoke(ctx, []Sequence{ItemSlice{k}, v})
		if err != nil {
			return err
		}
		result = append(result, seqMaterialize(r)...)
		return nil
	})
	if mapErr != nil {
		return nil, mapErr
	}
	return result, nil
}

func fnMapFind(_ context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[1]) == 0 {
		return ItemSlice{NewArray(nil)}, nil
	}
	ka, err := AtomizeItem(args[1].Get(0))
	if err != nil {
		return nil, err
	}
	var results []Sequence
	mapFindRecurse(args[0], ka, &results)
	return ItemSlice{NewArray(results)}, nil
}

// mapFindRecurse recursively searches for a key in maps within items.
// Per XPath 3.1, map:find searches recursively through maps and arrays.
func mapFindRecurse(items Sequence, key AtomicValue, results *[]Sequence) {
	for item := range seqItems(items) {
		switch v := item.(type) {
		case MapItem:
			if val, found := v.Get(key); found {
				*results = append(*results, val)
			}
			// Also recurse into map values
			_ = v.ForEach(func(_ AtomicValue, val Sequence) error {
				mapFindRecurse(val, key, results)
				return nil
			})
		case ArrayItem:
			// Recurse into array members
			for i := 1; i <= v.Size(); i++ {
				member, err := v.Get(i)
				if err != nil {
					continue
				}
				mapFindRecurse(member, key, results)
			}
		}
	}
}
