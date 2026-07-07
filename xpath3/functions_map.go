package xpath3

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium/internal/lexicon"
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
		return MapItem{}, &XPathError{Code: lexicon.ErrXPTY0004, Message: "expected single map"}
	}
	m, ok := seq.Get(0).(MapItem)
	if !ok {
		return MapItem{}, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("expected map, got %T", seq.Get(0))}
	}
	return m, nil
}

func fnMapMerge(ctx context.Context, args []Sequence) (Sequence, error) {
	duplicates := MergeUseFirst
	if len(args) > 1 {
		if seqLen(args[1]) == 0 {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "map:merge: options argument must be a map, got empty sequence"}
		}
		if seqLen(args[1]) > 1 {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "map:merge: options argument must be a single map"}
		}
		// The options map should contain "duplicates" key
		optMap, ok := args[1].Get(0).(MapItem)
		if !ok {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("map:merge: options argument must be a map, got %T", args[1].Get(0))}
		}
		key := AtomicValue{TypeName: TypeString, Value: "duplicates"}
		if val, found := optMap.Get(key); found {
			// Per F&O 3.1 option conventions, the 'duplicates' value is type
			// xs:string and is converted with the function conversion rules.
			// So xs:string subtypes (xs:NCName, ...), xs:anyURI, xs:untypedAtomic,
			// and a single-item array all coerce to a string. An empty, multi-item,
			// or non-convertible value is a FOJS0005 error (not XPTY0004).
			s, convErr := coerceDuplicatesOption(ctx, val)
			if convErr != nil {
				return nil, convErr
			}
			switch s {
			case duplicatesUseFirst:
				duplicates = MergeUseFirst
			case "use-last":
				duplicates = MergeUseLast
			case "use-any":
				// use-any allows the implementation to pick any value;
				// reusing use-first behavior is conformant.
				duplicates = MergeUseFirst
			case duplicatesReject:
				duplicates = MergeReject
			case "combine":
				duplicates = MergeCombine
			default:
				return nil, &XPathError{Code: errCodeFOJS0005, Message: fmt.Sprintf("map:merge: invalid value for 'duplicates' option: %q", s)}
			}
		}
	}
	builder := NewMapBuilder(duplicates, seqLen(args[0]))
	for item := range seqItems(args[0]) {
		m, ok := item.(MapItem)
		if !ok {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "map:merge requires sequence of maps"}
		}
		mergeErr := m.forEach0(builder.Add)
		if mergeErr != nil {
			return nil, mergeErr
		}
	}
	return ItemSlice{builder.Build()}, nil
}

// coerceDuplicatesOption converts the 'duplicates' option value of map:merge to
// a string. The option is declared as xs:string, so only xs:untypedAtomic,
// xs:string (and subtypes), and xs:anyURI (and subtypes) are accepted; any other
// atomic type — even a custom type whose Go payload happens to be a string — is a
// FOJS0005 error rather than being silently accepted. A single-item array still
// flattens to its member via atomization. Atomization is ctx-aware: an
// element-only-typed node has no typed value, so it surfaces err:FOTY0012
// unchanged rather than being masked as the FOJS0005 invalid-option error.
func coerceDuplicatesOption(ctx context.Context, val Sequence) (string, error) {
	var first AtomicValue
	count := 0
	_, err := atomizeStreamCont(val, typedValueItemCheck(ctx), func(av AtomicValue) (bool, error) {
		count++
		if count == 1 {
			first = av
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		if isNoTypedValueError(err) {
			return "", err
		}
		return "", &XPathError{Code: errCodeFOJS0005, Message: fmt.Sprintf("map:merge: 'duplicates' option must be a single string: %s", err)}
	}
	if count != 1 {
		return "", &XPathError{Code: errCodeFOJS0005, Message: fmt.Sprintf("map:merge: 'duplicates' option must be a single string, got sequence of length %d", count)}
	}
	av := first
	if av.TypeName != TypeUntypedAtomic &&
		!isAtomicSubtypeOf(av, TypeString) &&
		!isAtomicSubtypeOf(av, TypeAnyURI) {
		return "", &XPathError{Code: errCodeFOJS0005, Message: fmt.Sprintf("map:merge: 'duplicates' option must be a string, got %s", av.TypeName)}
	}
	s, convErr := atomicToString(av)
	if convErr != nil {
		return "", &XPathError{Code: errCodeFOJS0005, Message: fmt.Sprintf("map:merge: 'duplicates' option must be a single string: %s", convErr)}
	}
	return s, nil
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

func fnMapGet(ctx context.Context, args []Sequence) (Sequence, error) {
	m, err := extractMap(args[0])
	if err != nil {
		return nil, err
	}
	if seqLen(args[1]) == 0 {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "map:get requires a key argument"}
	}
	if seqLen(args[1]) > 1 {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "map:get key must be a single atomic value"}
	}
	ka, err := AtomizeItem(args[1].Get(0))
	if err != nil {
		return nil, err
	}
	// Borrow the stored value WITHOUT cloning (get0), then drain it through
	// appendBoundedClonedSeq so maxNodes / OpLimit / cancellation fire BEFORE the
	// value is materialized. Public Get deep-clones by materializing eagerly,
	// which would defeat the bound for a borrowed lazy value (and panic if its
	// Materialize panics). Cloning each appended item keeps value semantics, so
	// mutating the result cannot reach the source map.
	val, ok := m.get0(ka)
	if !ok {
		return validNilSequence, nil
	}
	ec := getFnContext(ctx)
	return appendBoundedClonedSeq(ctx, ec, nil, val, fnMaxNodes(ec))
}

func fnMapPut(_ context.Context, args []Sequence) (Sequence, error) {
	m, err := extractMap(args[0])
	if err != nil {
		return nil, err
	}
	ka, err := extractSingleAtomicArg(args[1], "map:put key")
	if err != nil {
		return nil, err
	}
	return ItemSlice{m.Put(ka, args[2])}, nil
}

func fnMapEntry(_ context.Context, args []Sequence) (Sequence, error) {
	ka, err := extractSingleAtomicArg(args[0], "map:entry key")
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
	ec := getFnContext(ctx)
	maxNodes := fnMaxNodes(ec)
	var result ItemSlice
	// Iterate without cloning (forEach0) so a borrowed lazy value is never
	// materialized by ForEach's clone before the bound/op guards run. For each
	// entry: bound the value length against maxNodes and charge the value's
	// length (min 1) of ops BEFORE cloning the key/value into the callback, so a
	// pathological lazy value is rejected rather than eagerly materialized.
	mapErr := m.forEach0(func(k AtomicValue, v Sequence) error {
		vLen := seqLen(v)
		if maxNodes > 0 && vLen > maxNodes {
			return ErrNodeSetLimit
		}
		if err := fnCountOps(ctx, ec, max(vLen, 1)); err != nil {
			return err
		}
		r, err := fi.Invoke(ctx, []Sequence{ItemSlice{cloneMapKey(k)}, cloneSequence(v)})
		if err != nil {
			return err
		}
		result, err = appendBoundedSeq(ctx, ec, result, r, maxNodes)
		if err != nil {
			return err
		}
		return nil
	})
	if mapErr != nil {
		return nil, mapErr
	}
	return result, nil
}

func fnMapFind(ctx context.Context, args []Sequence) (Sequence, error) {
	ka, err := extractSingleAtomicArg(args[1], "map:find key")
	if err != nil {
		return nil, err
	}
	ec := getFnContext(ctx)
	maxNodes := fnMaxNodes(ec)
	results, err := mapFindIter(ctx, ec, maxNodes, args[0], ka)
	if err != nil {
		return nil, err
	}
	return ItemSlice{NewArray(results)}, nil
}

// mapFindIter searches for a key in maps within items. Per XPath 3.1, map:find
// searches recursively through maps and arrays, in document order. The walk is
// performed iteratively with an explicit work stack rather than recursively, so
// a pathologically nested input cannot exhaust the goroutine stack when no op
// limit is set. The traversal charges an op per item and bounds the accumulated
// result item count so a deeply or widely nested input cannot exhaust resources.
//
// Each frame is a one-item-at-a-time cursor over a list of sequences: the root
// input, an array's members, or — for maps — the map's own entries accessed in
// place (so a wide map is never copied into a temporary []Sequence). Items are
// consumed via Sequence.Get so a lazy value/member sequence is never expanded
// into a temporary slice. A matched value is looked up without cloning (get0)
// and its length bound-checked before it is cloned, so a borrowed lazy value is
// rejected rather than materialized. The stack is LIFO, so on descending into a
// nested map/array the parent frame is left in place beneath the child frame,
// yielding document order.
func mapFindIter(ctx context.Context, ec *evalContext, maxNodes int, root Sequence, key AtomicValue) ([]Sequence, error) {
	var results []Sequence
	// total counts items already accumulated across all matched value sequences,
	// so a single huge (possibly lazy) matched value is bounded the same as many
	// small matches.
	total := 0

	stack := []seqCursor{{members: []Sequence{root}}}
	for len(stack) > 0 {
		top := &stack[len(stack)-1]
		item, ok, skippedEmpty := top.next()
		// Charge one op per empty value the cursor stepped past, so scanning N
		// empty map values costs ~N ops and trips OpLimit. Without this a map
		// with thousands of empty values is searched for free even when no key
		// matches.
		if skippedEmpty > 0 {
			if err := fnCountOps(ctx, ec, skippedEmpty); err != nil {
				return nil, err
			}
		}
		if !ok {
			stack = stack[:len(stack)-1]
			continue
		}
		if err := fnCountOp(ctx, ec); err != nil {
			return nil, err
		}
		switch v := item.(type) {
		case MapItem:
			// The map's own match comes before its descendants in document order.
			// Look the value up WITHOUT cloning (get0) and bound-check its length
			// before materializing: a borrowed lazy value (e.g. a huge integer
			// range stored via map:entry, which does not clone) must be rejected
			// with ErrNodeSetLimit rather than materialized by the clone in Get.
			if val, found := v.get0(key); found {
				// Each matched value becomes one member of the result array.
				// appendArrayMember bounds both the member count (many empty
				// matches must not exceed maxNodes members) and the item count (a
				// borrowed lazy value must be rejected before NewArray clones it),
				// and charges max(valLen, 1) ops so empty matches still cost an op.
				// The borrowed (uncloned) value is appended; the single defensive
				// clone happens once in NewArray(results) at the call site.
				var aerr error
				results, total, aerr = appendArrayMember(ctx, ec, maxNodes, results, total, val)
				if aerr != nil {
					return nil, aerr
				}
			}
			// Then descend into the map's values, in insertion order, iterating
			// the entries in place so a wide map is not duplicated into a
			// temporary []Sequence before traversal.
			stack = append(stack, seqCursor{mapEntries: v.entries0()})
		case ArrayItem:
			stack = append(stack, seqCursor{members: v.members0()})
		}
	}
	return results, nil
}
