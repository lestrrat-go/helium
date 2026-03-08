package xpath3

import (
	"context"
	"fmt"
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
	pos := int(promoteToDouble(a))
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
	pos := int(promoteToDouble(a))

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
	start := int(promoteToDouble(a))
	if start < 1 {
		start = 1
	}

	length := len(seq) - start + 1
	if len(args) > 2 {
		la, err := AtomizeItem(args[2][0])
		if err != nil {
			return nil, err
		}
		length = int(promoteToDouble(la))
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
			result = append(result, AtomicValue{TypeName: TypeInteger, Value: int64(i + 1)})
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
