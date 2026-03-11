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
	registerNS(NSArray, "flat-map", 2, 2, fnArrayFlatMap)
	registerNS(NSArray, "filter", 2, 2, fnArrayFilter)
	registerNS(NSArray, "fold-left", 3, 3, fnArrayFoldLeft)
	registerNS(NSArray, "fold-right", 3, 3, fnArrayFoldRight)
	registerNS(NSArray, "for-each", 2, 2, fnArrayForEach)
	registerNS(NSArray, "for-each-pair", 3, 3, fnArrayForEachPair)
	registerNS(NSArray, "sort", 1, 3, fnArraySort)
}

func extractArray(seq Sequence) (ArrayItem, error) {
	if len(seq) != 1 {
		return ArrayItem{}, &XPathError{Code: "XPTY0004", Message: "expected single array"}
	}
	a, ok := seq[0].(ArrayItem)
	if !ok {
		return ArrayItem{}, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("expected array, got %T", seq[0])}
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
	idx := int(seqToDouble(args[1]))
	return a.Get(idx)
}

func fnArrayPut(_ context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	idx := int(seqToDouble(args[1]))
	result, err := a.Put(idx, args[2])
	if err != nil {
		return nil, err
	}
	return Sequence{result}, nil
}

func fnArrayAppend(_ context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	return Sequence{a.Append(args[1])}, nil
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
	return Sequence{sub}, nil
}

func fnArrayRemove(_ context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	// args[1] is a sequence of positions to remove
	positions := make(map[int]bool)
	for _, item := range args[1] {
		av, err := AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		positions[int(av.ToFloat64())] = true
	}
	members := a.Members()
	var result []Sequence
	for i, m := range members {
		if !positions[i+1] {
			result = append(result, m)
		}
	}
	return Sequence{NewArray(result)}, nil
}

func fnArrayInsertBefore(_ context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	pos := int(seqToDouble(args[1]))
	members := a.Members()
	if pos < 1 || pos > len(members)+1 {
		return nil, &XPathError{Code: "FOAY0001", Message: "array index out of bounds"}
	}
	result := make([]Sequence, 0, len(members)+1)
	result = append(result, members[:pos-1]...)
	result = append(result, args[2])
	result = append(result, members[pos-1:]...)
	return Sequence{NewArray(result)}, nil
}

func fnArrayHead(_ context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	if a.Size() == 0 {
		return nil, &XPathError{Code: "FOAY0001", Message: "array:head on empty array"}
	}
	return a.Get(1)
}

func fnArrayTail(_ context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	if a.Size() == 0 {
		return nil, &XPathError{Code: "FOAY0001", Message: "array:tail on empty array"}
	}
	sub, err := a.SubArray(2, a.Size()-1)
	if err != nil {
		return nil, err
	}
	return Sequence{sub}, nil
}

func fnArrayReverse(_ context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	members := a.Members()
	reversed := make([]Sequence, len(members))
	for i, m := range members {
		reversed[len(members)-1-i] = m
	}
	return Sequence{NewArray(reversed)}, nil
}

func fnArrayJoin(_ context.Context, args []Sequence) (Sequence, error) {
	var allMembers []Sequence
	for _, item := range args[0] {
		a, ok := item.(ArrayItem)
		if !ok {
			return nil, &XPathError{Code: "XPTY0004", Message: "array:join requires sequence of arrays"}
		}
		allMembers = append(allMembers, a.Members()...)
	}
	return Sequence{NewArray(allMembers)}, nil
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
	for _, m := range a.Members() {
		r, err := fi.Invoke(ctx, []Sequence{m})
		if err != nil {
			return nil, err
		}
		// Each result should be an array; collect members
		for _, item := range r {
			if ra, ok := item.(ArrayItem); ok {
				allMembers = append(allMembers, ra.Members()...)
			} else {
				allMembers = append(allMembers, Sequence{item})
			}
		}
	}
	return Sequence{NewArray(allMembers)}, nil
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
	for _, m := range a.Members() {
		r, err := fi.Invoke(ctx, []Sequence{m})
		if err != nil {
			return nil, err
		}
		// Per XPath 3.1, the callback must return exactly one xs:boolean
		if len(r) != 1 {
			return nil, &XPathError{Code: "XPTY0004", Message: "array:filter callback must return a single xs:boolean value"}
		}
		av, ok := r[0].(AtomicValue)
		if !ok || av.TypeName != TypeBoolean {
			return nil, &XPathError{Code: "XPTY0004", Message: "array:filter callback must return xs:boolean"}
		}
		if av.BooleanVal() {
			result = append(result, m)
		}
	}
	return Sequence{NewArray(result)}, nil
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
	for _, m := range a.Members() {
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
	members := a.Members()
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
	for _, m := range a.Members() {
		r, err := fi.Invoke(ctx, []Sequence{m})
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return Sequence{NewArray(results)}, nil
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
	size := a1.Size()
	if a2.Size() < size {
		size = a2.Size()
	}
	var results []Sequence
	for i := 1; i <= size; i++ {
		m1, _ := a1.Get(i)
		m2, _ := a2.Get(i)
		r, err := fi.Invoke(ctx, []Sequence{m1, m2})
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return Sequence{NewArray(results)}, nil
}

func fnArraySort(ctx context.Context, args []Sequence) (Sequence, error) {
	a, err := extractArray(args[0])
	if err != nil {
		return nil, err
	}
	members := make([]Sequence, len(a.Members()))
	copy(members, a.Members())

	// Optional collation (2nd arg) — validate but we only support codepoint
	if len(args) > 1 && len(args[1]) > 0 {
		uri := seqToString(args[1])
		if uri != "" && uri != codepointCollationURI {
			return nil, &XPathError{Code: "FOCH0002", Message: fmt.Sprintf("unsupported collation: %s", uri)}
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
			entries[i].key = make(Sequence, len(atoms))
			for j, av := range atoms {
				entries[i].key[j] = av
			}
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
		minLen := len(ki)
		if len(kj) < minLen {
			minLen = len(kj)
		}
		for idx := 0; idx < minLen; idx++ {
			ai, errI := AtomizeItem(ki[idx])
			aj, errJ := AtomizeItem(kj[idx])
			if errI != nil {
				sortErr = errI
				return false
			}
			if errJ != nil {
				sortErr = errJ
				return false
			}
			lt, err := ValueCompare(TokenLt, ai, aj)
			if err != nil {
				sortErr = err
				return false
			}
			if lt {
				return true
			}
			gt, err := ValueCompare(TokenGt, ai, aj)
			if err != nil {
				sortErr = err
				return false
			}
			if gt {
				return false
			}
			// equal — continue to next element
		}
		return len(ki) < len(kj)
	})
	if sortErr != nil {
		return nil, sortErr
	}

	sorted := make([]Sequence, len(entries))
	for i, e := range entries {
		sorted[i] = e.member
	}
	return Sequence{NewArray(sorted)}, nil
}
