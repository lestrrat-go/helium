package xpath3

import (
	"context"
	"fmt"
	"sort"
)

func init() {
	registerNS(NSArray, "size", 1, 1, fnArraySize)
	registerNS(NSArray, "get", 2, 2, fnArrayGet)
	registerNS(NSArray, "put", 3, 3, fnArrayPut)
	registerNS(NSArray, "append", 2, 2, fnArrayAppend)
	registerNS(NSArray, "subarray", 2, 3, fnArraySubarray)
	registerNS(NSArray, "remove", 2, 2, fnArrayRemove)
	registerNS(NSArray, "insert-before", 3, 3, fnArrayInsertBefore)
	registerNS(NSArray, "head", 1, 1, fnArrayHead)
	registerNS(NSArray, "tail", 1, 1, fnArrayTail)
	registerNS(NSArray, "reverse", 1, 1, fnArrayReverse)
	registerNS(NSArray, "join", 1, 1, fnArrayJoin)
	registerNS(NSArray, "flatten", 1, 1, fnArrayFlatten)
	registerNS(NSArray, "flat-map", 2, 2, fnArrayFlatMap)
	registerNS(NSArray, "filter", 2, 2, fnArrayFilter)
	registerNS(NSArray, "fold-left", 3, 3, fnArrayFoldLeft)
	registerNS(NSArray, "fold-right", 3, 3, fnArrayFoldRight)
	registerNS(NSArray, "for-each", 2, 2, fnArrayForEach)
	registerNS(NSArray, "for-each-pair", 3, 3, fnArrayForEachPair)
	registerNS(NSArray, "sort", 1, 3, fnArraySort)
}

func extractArray(seq Sequence) (ArrayItem, error) {
	if seqLen(seq) != 1 {
		return ArrayItem{}, &XPathError{Code: errCodeXPTY0004, Message: "expected single array"}
	}
	a, ok := seq.Get(0).(ArrayItem)
	if !ok {
		return ArrayItem{}, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("expected array, got %T", seq.Get(0))}
	}
	return a, nil
}

func fnArraySize(_ context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	return SingleInteger(int64(a.Size())), nil
}

func fnArrayGet(_ context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	idx, err := extractArrayIndex(args[1])
	if err != nil {
		return nil, err
	}
	return a.Get(idx)
}

// extractArrayIndex extracts a single xs:integer index from a sequence, validating
// that it is exactly one integer (not a decimal, sequence, etc.).
func extractArrayIndex(seq Sequence) (int, error) {
	if seqLen(seq) != 1 {
		return 0, &XPathError{Code: errCodeXPTY0004, Message: "array index must be a single xs:integer"}
	}
	av, err := AtomizeItem(seq.Get(0))
	if err != nil {
		return 0, err
	}
	if !isIntegerDerived(av.TypeName) {
		return 0, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("array index must be xs:integer, got %s", av.TypeName)}
	}
	iv, ok := av.Int64Val()
	if !ok {
		return 0, &XPathError{Code: errCodeFOAY0001, Message: "array index out of range"}
	}
	return int(iv), nil
}

func fnArrayPut(_ context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	idx, err := extractArrayIndex(args[1])
	if err != nil {
		return nil, err
	}
	result, err := a.Put(idx, args[2])
	if err != nil {
		return nil, err
	}
	return ItemSlice{result}, nil
}

func fnArrayAppend(_ context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	return ItemSlice{a.Append(args[1])}, nil
}

func fnArraySubarray(_ context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	start := int(seqToDouble(args[1]))
	length := a.Size() - start + 1
	if len(args) > 2 {
		length = int(seqToDouble(args[2]))
	}
	sub, err := a.SubArray(start, length)
	if err != nil {
		return nil, err
	}
	return ItemSlice{sub}, nil
}

func fnArrayRemove(_ context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	// args[1] is a sequence of positions to remove
	positions := make(map[int]struct{})
	size := a.Size()
	for item := range seqItems(args[1]) {
		av, err := AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		pos := int(av.ToFloat64())
		if pos < 1 || pos > size {
			return nil, &XPathError{Code: errCodeFOAY0001, Message: fmt.Sprintf("array:remove: position %d out of range 1..%d", pos, size)}
		}
		positions[pos] = struct{}{}
	}
	members := a.members0()
	var result []Sequence
	for i, m := range members {
		if _, skip := positions[i+1]; !skip {
			result = append(result, m)
		}
	}
	return ItemSlice{NewArray(result)}, nil
}

func fnArrayInsertBefore(_ context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	pos := int(seqToDouble(args[1]))
	members := a.members0()
	if pos < 1 || pos > len(members)+1 {
		return nil, &XPathError{Code: errCodeFOAY0001, Message: "array index out of bounds"}
	}
	result := make([]Sequence, 0, len(members)+1)
	result = append(result, members[:pos-1]...)
	result = append(result, args[2])
	result = append(result, members[pos-1:]...)
	return ItemSlice{NewArray(result)}, nil
}

func fnArrayHead(_ context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	if a.Size() == 0 {
		return nil, &XPathError{Code: errCodeFOAY0001, Message: "array:head on empty array"}
	}
	return a.Get(1)
}

func fnArrayTail(_ context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	if a.Size() == 0 {
		return nil, &XPathError{Code: errCodeFOAY0001, Message: "array:tail on empty array"}
	}
	sub, err := a.SubArray(2, a.Size()-1)
	if err != nil {
		return nil, err
	}
	return ItemSlice{sub}, nil
}

func fnArrayReverse(_ context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	members := a.members0()
	reversed := make([]Sequence, len(members))
	for i, m := range members {
		reversed[len(members)-1-i] = m
	}
	return ItemSlice{NewArray(reversed)}, nil
}

func fnArrayJoin(_ context.Context, args []Sequence) (Sequence, error) {
	var allMembers []Sequence
	for item := range seqItems(args[0]) {
		a, ok := item.(ArrayItem)
		if !ok {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "array:join requires sequence of arrays"}
		}
		allMembers = append(allMembers, a.members0()...)
	}
	return ItemSlice{NewArray(allMembers)}, nil
}

func fnArrayFlatten(_ context.Context, args []Sequence) (Sequence, error) {
	var result ItemSlice
	for item := range seqItems(args[0]) {
		flattenArrayItem(&result, item)
	}
	return result, nil
}

func flattenArrayItem(dst *ItemSlice, item Item) {
	arr, ok := item.(ArrayItem)
	if !ok {
		*dst = append(*dst, item)
		return
	}
	for _, member := range arr.members0() {
		for child := range seqItems(member) {
			flattenArrayItem(dst, child)
		}
	}
}

func fnArrayFlatMap(ctx context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	fi, err := extractFunctionItem(args[1])
	if err != nil {
		return nil, err
	}
	var allMembers []Sequence
	for _, m := range a.members0() {
		r, err := fi.Invoke(ctx, []Sequence{m})
		if err != nil {
			return nil, err
		}
		// Each result should be an array; collect members
		for item := range seqItems(r) {
			if ra, ok := item.(ArrayItem); ok {
				allMembers = append(allMembers, ra.members0()...)
			} else {
				allMembers = append(allMembers, ItemSlice{item})
			}
		}
	}
	return ItemSlice{NewArray(allMembers)}, nil
}

func fnArrayFilter(ctx context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	fi, err := extractFunctionItem(args[1])
	if err != nil {
		return nil, err
	}
	var result []Sequence
	for _, m := range a.members0() {
		r, err := fi.Invoke(ctx, []Sequence{m})
		if err != nil {
			return nil, err
		}
		// Per XPath 3.1, the callback must return exactly one xs:boolean
		if seqLen(r) != 1 {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "array:filter callback must return a single xs:boolean value"}
		}
		av, ok := r.Get(0).(AtomicValue)
		if !ok || av.TypeName != TypeBoolean {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "array:filter callback must return xs:boolean"}
		}
		if av.BooleanVal() {
			result = append(result, m)
		}
	}
	return ItemSlice{NewArray(result)}, nil
}

func fnArrayFoldLeft(ctx context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	acc := args[1]
	fi, err := extractFunctionItem(args[2])
	if err != nil {
		return nil, err
	}
	for _, m := range a.members0() {
		acc, err = fi.Invoke(ctx, []Sequence{acc, m})
		if err != nil {
			return nil, err
		}
	}
	return acc, nil
}

func fnArrayFoldRight(ctx context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	acc := args[1]
	fi, err := extractFunctionItem(args[2])
	if err != nil {
		return nil, err
	}
	members := a.members0()
	for i := len(members) - 1; i >= 0; i-- {
		acc, err = fi.Invoke(ctx, []Sequence{members[i], acc})
		if err != nil {
			return nil, err
		}
	}
	return acc, nil
}

func fnArrayForEach(ctx context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	fi, err := extractFunctionItem(args[1])
	if err != nil {
		return nil, err
	}
	var results []Sequence
	for _, m := range a.members0() {
		r, err := fi.Invoke(ctx, []Sequence{m})
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return ItemSlice{NewArray(results)}, nil
}

func fnArrayForEachPair(ctx context.Context, args []Sequence) (Sequence, error) {
	a1, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	a2, err := extractArray(args[1])
	if err != nil {
		return nil, err
	}
	fi, err := extractFunctionItem(args[2])
	if err != nil {
		return nil, err
	}
	if fi.Arity >= 0 && fi.Arity != 2 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("array:for-each-pair callback must have arity 2, got %d", fi.Arity)}
	}
	size := a1.Size()
	if a2.Size() < size {
		size = a2.Size()
	}
	var results []Sequence
	for i := 1; i <= size; i++ {
		m1, _ := a1.get0(i)
		m2, _ := a2.get0(i)
		r, err := fi.Invoke(ctx, []Sequence{m1, m2})
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return ItemSlice{NewArray(results)}, nil
}

func fnArraySort(ctx context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	src := a.members0()
	members := make([]Sequence, len(src))
	copy(members, src)

	// Optional collation (2nd arg)
	var coll *collationImpl
	if len(args) > 1 && seqLen(args[1]) > 0 {
		uri, err := coerceArgToString(args[1])
		if err != nil {
			return nil, err
		}
		if uri != "" {
			coll, err = resolveCollation(uri, "")
			if err != nil {
				return nil, err
			}
		}
	}

	// Optional key function (3rd arg)
	var keyFn *FunctionItem
	if len(args) > 2 {
		fi, err := extractFunctionItem(args[2])
		if err != nil {
			return nil, err
		}
		keyFn = &fi
	}

	type sortEntry struct {
		member Sequence
		key    Sequence // atomized key sequence for comparison
	}
	entries := make([]sortEntry, len(members))
	for i, m := range members {
		entries[i].member = m
		if keyFn != nil {
			r, err := keyFn.Invoke(ctx, []Sequence{m})
			if err != nil {
				return nil, err
			}
			entries[i].key = r
		} else {
			// Default: atomize the member
			atoms, err := AtomizeSequence(m)
			if err != nil {
				return nil, err
			}
			keySlice := make(ItemSlice, len(atoms))
			for j, av := range atoms {
				keySlice[j] = av
			}
			entries[i].key = keySlice
		}
	}

	var sortErr error
	sort.SliceStable(entries, func(i, j int) bool {
		if sortErr != nil {
			return false
		}
		ki := entries[i].key
		kj := entries[j].key
		// Compare element-by-element per XPath 3.1 sort key comparison
		minLen := seqLen(ki)
		if seqLen(kj) < minLen {
			minLen = seqLen(kj)
		}
		for idx := 0; idx < minLen; idx++ {
			ai, errI := AtomizeItem(ki.Get(idx))
			aj, errJ := AtomizeItem(kj.Get(idx))
			if errI != nil {
				sortErr = errI
				return false
			}
			if errJ != nil {
				sortErr = errJ
				return false
			}
			cmp, err := valueCompareThreeWay(ai, aj, coll)
			if err != nil {
				sortErr = err
				return false
			}
			if cmp < 0 {
				return true
			}
			if cmp > 0 {
				return false
			}
			// equal — continue to next element
		}
		return seqLen(ki) < seqLen(kj)
	})
	if sortErr != nil {
		return nil, sortErr
	}

	sorted := make([]Sequence, len(entries))
	for i, e := range entries {
		sorted[i] = e.member
	}
	return ItemSlice{NewArray(sorted)}, nil
}
