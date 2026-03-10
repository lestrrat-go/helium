package xpath3

import (
	"context"
	"fmt"
)

func init() {
	registerFn("for-each", 2, 2, fnForEach)
	registerFn("filter", 2, 2, fnFilter)
	registerFn("fold-left", 3, 3, fnFoldLeft)
	registerFn("fold-right", 3, 3, fnFoldRight)
	registerFn("for-each-pair", 3, 3, fnForEachPair)
	registerFn("apply", 2, 2, fnApply)
	registerFn("function-lookup", 2, 2, fnFunctionLookup)
	registerFn("function-arity", 1, 1, fnFunctionArity)
	registerFn("function-name", 1, 1, fnFunctionName)
}

func fnForEach(ctx context.Context, args []Sequence) (Sequence, error) {
	seq := args[0]
	fi, err := extractFunctionItem(args[1])
	if err != nil {
		return nil, err
	}
	var result Sequence
	for _, item := range seq {
		r, err := fi.Invoke(ctx, []Sequence{{item}})
		if err != nil {
			return nil, err
		}
		result = append(result, r...)
	}
	return result, nil
}

func fnFilter(ctx context.Context, args []Sequence) (Sequence, error) {
	seq := args[0]
	fi, err := extractFunctionItem(args[1])
	if err != nil {
		return nil, err
	}
	var result Sequence
	for _, item := range seq {
		r, err := fi.Invoke(ctx, []Sequence{{item}})
		if err != nil {
			return nil, err
		}
		b, err := EBV(r)
		if err != nil {
			return nil, err
		}
		if b {
			result = append(result, item)
		}
	}
	return result, nil
}

func fnFoldLeft(ctx context.Context, args []Sequence) (Sequence, error) {
	seq := args[0]
	acc := args[1]
	fi, err := extractFunctionItem(args[2])
	if err != nil {
		return nil, err
	}
	for _, item := range seq {
		acc, err = fi.Invoke(ctx, []Sequence{acc, {item}})
		if err != nil {
			return nil, err
		}
	}
	return acc, nil
}

func fnFoldRight(ctx context.Context, args []Sequence) (Sequence, error) {
	seq := args[0]
	acc := args[1]
	fi, err := extractFunctionItem(args[2])
	if err != nil {
		return nil, err
	}
	for i := len(seq) - 1; i >= 0; i-- {
		acc, err = fi.Invoke(ctx, []Sequence{{seq[i]}, acc})
		if err != nil {
			return nil, err
		}
	}
	return acc, nil
}

func fnForEachPair(ctx context.Context, args []Sequence) (Sequence, error) {
	seq1 := args[0]
	seq2 := args[1]
	fi, err := extractFunctionItem(args[2])
	if err != nil {
		return nil, err
	}
	size := len(seq1)
	if len(seq2) < size {
		size = len(seq2)
	}
	var result Sequence
	for i := 0; i < size; i++ {
		r, err := fi.Invoke(ctx, []Sequence{{seq1[i]}, {seq2[i]}})
		if err != nil {
			return nil, err
		}
		result = append(result, r...)
	}
	return result, nil
}

func fnApply(ctx context.Context, args []Sequence) (Sequence, error) {
	fi, err := extractFunctionItem(args[0])
	if err != nil {
		return nil, err
	}
	arr, ok := args[1][0].(ArrayItem)
	if !ok {
		return nil, &XPathError{Code: "XPTY0004", Message: "apply() second argument must be array"}
	}
	fnArgs := make([]Sequence, arr.Size())
	for i := range fnArgs {
		v, err := arr.Get(i + 1)
		if err != nil {
			return nil, err
		}
		fnArgs[i] = v
	}
	return fi.Invoke(ctx, fnArgs)
}

func fnFunctionLookup(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 || len(args[1]) == 0 {
		return nil, nil
	}
	a, err := AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	if a.TypeName != TypeQName {
		return nil, &XPathError{Code: "XPTY0004", Message: "function-lookup() first argument must be QName"}
	}
	qv := a.QNameVal()
	arityA, err := AtomizeItem(args[1][0])
	if err != nil {
		return nil, err
	}
	arity := int(arityA.ToFloat64())

	qn := QualifiedName{URI: qv.URI, Name: qv.Local}
	fn, ok := builtinFunctions3[qn]
	if !ok {
		// Try fn: namespace if URI is empty
		if qv.URI == "" {
			qn.URI = NSFn
			fn, ok = builtinFunctions3[qn]
		}
		if !ok {
			return nil, nil
		}
	}

	fi := FunctionItem{
		Arity: arity,
		Name:  qv.Local,
		Invoke: func(ctx context.Context, callArgs []Sequence) (Sequence, error) {
			return fn.Call(ctx, callArgs)
		},
	}
	return Sequence{fi}, nil
}

func fnFunctionArity(_ context.Context, args []Sequence) (Sequence, error) {
	fi, err := extractFunctionItem(args[0])
	if err != nil {
		return nil, err
	}
	return SingleInteger(int64(fi.Arity)), nil
}

func fnFunctionName(_ context.Context, args []Sequence) (Sequence, error) {
	fi, err := extractFunctionItem(args[0])
	if err != nil {
		return nil, err
	}
	if fi.Name == "" {
		return nil, nil
	}
	return SingleAtomic(AtomicValue{
		TypeName: TypeQName,
		Value:    QNameValue{Local: fi.Name, URI: NSFn},
	}), nil
}

func extractFunctionItem(seq Sequence) (FunctionItem, error) {
	if len(seq) != 1 {
		return FunctionItem{}, &XPathError{Code: "XPTY0004", Message: "expected single function item"}
	}
	fi, ok := seq[0].(FunctionItem)
	if !ok {
		return FunctionItem{}, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("expected function item, got %T", seq[0])}
	}
	return fi, nil
}
