package xpath3

import (
	"context"
	"fmt"
	"math/big"
	"sort"
)

func init() {
	registerFn("empty", 1, 1, fnEmpty)
	registerFn("exists", 1, 1, fnExists)
	registerFn("head", 1, 1, fnHead)
	registerFn("tail", 1, 1, fnTail)
	registerFn("insert-before", 3, 3, fnInsertBefore)
	registerFn("remove", 2, 2, fnRemove)
	registerFn("reverse", 1, 1, fnReverse)
	registerFn("subsequence", 2, 3, fnSubsequence)
	registerFn("unordered", 1, 1, fnUnordered)
	registerFn("zero-or-one", 1, 1, fnZeroOrOne)
	registerFn("one-or-more", 1, 1, fnOneOrMore)
	registerFn("exactly-one", 1, 1, fnExactlyOne)
	registerFn("deep-equal", 2, 3, fnDeepEqual)
	registerFn("index-of", 2, 3, fnIndexOf)
	registerFn("last", 0, 0, fnLast)
	registerFn("position", 0, 0, fnPosition)
	registerFn("sort", 1, 3, fnSort)
	registerFn("flatten", 1, 1, fnFlatten)
}

func fnEmpty(_ context.Context, args []Sequence) (Sequence, error) {
	return SingleBoolean(len(args[0]) == 0), nil
}

func fnExists(_ context.Context, args []Sequence) (Sequence, error) {
	return SingleBoolean(len(args[0]) > 0), nil
}

func fnHead(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	return Sequence{args[0][0]}, nil
}

func fnTail(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) <= 1 {
		return nil, nil
	}
	return args[0][1:], nil
}

func fnInsertBefore(_ context.Context, args []Sequence) (Sequence, error) {
	target := args[0]
	if len(args[1]) == 0 {
		return nil, fmt.Errorf("xpath3: fn:insert-before: position argument is an empty sequence")
	}
	a, err := AtomizeItem(args[1][0])
	if err != nil {
		return nil, err
	}
	pos := int(a.ToFloat64())
	inserts := args[2]

	if pos < 1 {
		pos = 1
	}
	if pos > len(target)+1 {
		pos = len(target) + 1
	}

	result := make(Sequence, 0, len(target)+len(inserts))
	result = append(result, target[:pos-1]...)
	result = append(result, inserts...)
	result = append(result, target[pos-1:]...)
	return result, nil
}

func fnRemove(_ context.Context, args []Sequence) (Sequence, error) {
	target := args[0]
	if len(args[1]) == 0 {
		return nil, fmt.Errorf("xpath3: fn:remove: position argument is an empty sequence")
	}
	a, err := AtomizeItem(args[1][0])
	if err != nil {
		return nil, err
	}
	pos := int(a.ToFloat64())

	if pos < 1 || pos > len(target) {
		return target, nil
	}

	result := make(Sequence, 0, len(target)-1)
	result = append(result, target[:pos-1]...)
	result = append(result, target[pos:]...)
	return result, nil
}

func fnReverse(_ context.Context, args []Sequence) (Sequence, error) {
	seq := args[0]
	result := make(Sequence, len(seq))
	for i, item := range seq {
		result[len(seq)-1-i] = item
	}
	return result, nil
}

func fnSubsequence(_ context.Context, args []Sequence) (Sequence, error) {
	seq := args[0]
	a, err := AtomizeItem(args[1][0])
	if err != nil {
		return nil, err
	}
	start := int(a.ToFloat64())
	if start < 1 {
		start = 1
	}

	length := len(seq) - start + 1
	if len(args) > 2 {
		la, err := AtomizeItem(args[2][0])
		if err != nil {
			return nil, err
		}
		length = int(la.ToFloat64())
	}

	if start > len(seq) || length <= 0 {
		return nil, nil
	}

	end := start - 1 + length
	if end > len(seq) {
		end = len(seq)
	}
	return seq[start-1 : end], nil
}

func fnUnordered(_ context.Context, args []Sequence) (Sequence, error) {
	return args[0], nil
}

func fnZeroOrOne(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) > 1 {
		return nil, &XPathError{Code: "FORG0003", Message: fmt.Sprintf("zero-or-one() called with sequence of length %d", len(args[0]))}
	}
	return args[0], nil
}

func fnOneOrMore(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, &XPathError{Code: "FORG0004", Message: "one-or-more() called with empty sequence"}
	}
	return args[0], nil
}

func fnExactlyOne(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) != 1 {
		return nil, &XPathError{Code: "FORG0005", Message: fmt.Sprintf("exactly-one() called with sequence of length %d", len(args[0]))}
	}
	return args[0], nil
}

func fnDeepEqual(_ context.Context, args []Sequence) (Sequence, error) {
	a := args[0]
	b := args[1]
	if len(a) != len(b) {
		return SingleBoolean(false), nil
	}
	for i := range a {
		aa, err1 := AtomizeItem(a[i])
		ba, err2 := AtomizeItem(b[i])
		if err1 != nil || err2 != nil {
			return SingleBoolean(false), nil
		}
		sa, _ := atomicToString(aa)
		sb, _ := atomicToString(ba)
		if sa != sb {
			return SingleBoolean(false), nil
		}
	}
	return SingleBoolean(true), nil
}

func fnIndexOf(_ context.Context, args []Sequence) (Sequence, error) {
	seq := args[0]
	if len(args[1]) == 0 {
		return nil, nil
	}
	search, err := AtomizeItem(args[1][0])
	if err != nil {
		return nil, err
	}
	ss, _ := atomicToString(search)

	var result Sequence
	for i, item := range seq {
		a, err := AtomizeItem(item)
		if err != nil {
			continue
		}
		as, _ := atomicToString(a)
		if as == ss && a.TypeName == search.TypeName {
			result = append(result, AtomicValue{TypeName: TypeInteger, Value: big.NewInt(int64(i + 1))})
		}
	}
	return result, nil
}

func fnLast(ctx context.Context, _ []Sequence) (Sequence, error) {
	fc := GetFnContext(ctx)
	if fc == nil {
		return SingleInteger(0), nil
	}
	return SingleInteger(int64(fc.size)), nil
}

func fnPosition(ctx context.Context, _ []Sequence) (Sequence, error) {
	fc := GetFnContext(ctx)
	if fc == nil {
		return SingleInteger(0), nil
	}
	return SingleInteger(int64(fc.position)), nil
}

// fnSort implements fn:sort($input [, $collation [, $key]])
func fnSort(ctx context.Context, args []Sequence) (Sequence, error) {
	input := args[0]
	if len(input) <= 1 {
		return input, nil
	}

	// Optional key function (3rd argument)
	var keyFn *FunctionItem
	if len(args) >= 3 && len(args[2]) > 0 {
		fi, err := extractFunctionItem(args[2])
		if err != nil {
			return nil, err
		}
		keyFn = &fi
	}

	// Compute sort keys
	type sortPair struct {
		item Item
		key  Sequence
	}
	pairs := make([]sortPair, len(input))
	for i, item := range input {
		if keyFn != nil {
			k, err := keyFn.Invoke(ctx, []Sequence{{item}})
			if err != nil {
				return nil, err
			}
			pairs[i] = sortPair{item: item, key: k}
		} else {
			// Default key: atomize the item
			a, err := AtomizeItem(item)
			if err != nil {
				pairs[i] = sortPair{item: item, key: nil}
			} else {
				pairs[i] = sortPair{item: item, key: Sequence{a}}
			}
		}
	}

	// Stable sort using comparison
	var sortErr error
	sort.SliceStable(pairs, func(i, j int) bool {
		if sortErr != nil {
			return false
		}
		ki := pairs[i].key
		kj := pairs[j].key
		if len(ki) == 0 && len(kj) == 0 {
			return false
		}
		if len(ki) == 0 {
			return true // empty < non-empty
		}
		if len(kj) == 0 {
			return false
		}
		ai, err1 := AtomizeItem(ki[0])
		aj, err2 := AtomizeItem(kj[0])
		if err1 != nil || err2 != nil {
			return false
		}
		less, err := ValueCompare(TokenLt, ai, aj)
		if err != nil {
			// Fall back to string comparison
			si, _ := atomicToString(ai)
			sj, _ := atomicToString(aj)
			return si < sj
		}
		return less
	})
	if sortErr != nil {
		return nil, sortErr
	}

	result := make(Sequence, len(pairs))
	for i, p := range pairs {
		result[i] = p.item
	}
	return result, nil
}

func fnFlatten(_ context.Context, args []Sequence) (Sequence, error) {
	var result Sequence
	flattenItems(args[0], &result)
	return result, nil
}

func flattenItems(seq Sequence, result *Sequence) {
	for _, item := range seq {
		if arr, ok := item.(ArrayItem); ok {
			for _, m := range arr.Members() {
				flattenItems(m, result)
			}
		} else {
			*result = append(*result, item)
		}
	}
}
