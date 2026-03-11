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
	// Per XPath 3.1, the callback must have arity 1
	if fi.Arity >= 0 && fi.Arity != 1 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("fn:filter callback must have arity 1, got %d", fi.Arity)}
	}
	var result Sequence
	for _, item := range seq {
		r, err := fi.Invoke(ctx, []Sequence{{item}})
		if err != nil {
			return nil, err
		}
		// Per XPath 3.1, the callback must return exactly one xs:boolean
		if len(r) != 1 {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "fn:filter callback must return a single xs:boolean value"}
		}
		av, ok := r[0].(AtomicValue)
		if !ok || av.TypeName != TypeBoolean {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "fn:filter callback must return a single xs:boolean value"}
		}
		if av.BooleanVal() {
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
	// Per XPath 3.1, the callback must have arity 2
	if fi.Arity >= 0 && fi.Arity != 2 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("fn:for-each-pair callback must have arity 2, got %d", fi.Arity)}
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
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "apply() second argument must be array"}
	}
	if fi.Arity >= 0 && arr.Size() != fi.Arity {
		return nil, &XPathError{Code: "FOAP0001", Message: fmt.Sprintf("fn:apply: function has arity %d but array has %d members", fi.Arity, arr.Size())}
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
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "function-lookup() first argument must be QName"}
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
	ns := fi.Namespace
	if ns == "" {
		ns = NSFn
	}
	prefix := namespacePrefixFor(ns)
	return SingleAtomic(AtomicValue{
		TypeName: TypeQName,
		Value:    QNameValue{Prefix: prefix, Local: fi.Name, URI: ns},
	}), nil
}

func extractFunctionItem(seq Sequence) (FunctionItem, error) {
	if len(seq) != 1 {
		return FunctionItem{}, &XPathError{Code: errCodeXPTY0004, Message: "expected single function item"}
	}
	switch v := seq[0].(type) {
	case FunctionItem:
		return v, nil
	case MapItem:
		// Maps are functions: map($key) → value
		return FunctionItem{
			Arity: 1,
			Invoke: func(ctx context.Context, args []Sequence) (Sequence, error) {
				if len(args) != 1 || len(args[0]) != 1 {
					return nil, &XPathError{Code: errCodeXPTY0004, Message: "map lookup requires exactly one argument"}
				}
				key, err := AtomizeItem(args[0][0])
				if err != nil {
					return nil, err
				}
				val, ok := v.Get(key)
				if !ok {
					return nil, nil
				}
				return val, nil
			},
		}, nil
	case ArrayItem:
		// Arrays are functions: array($index) → member
		return FunctionItem{
			Arity: 1,
			Invoke: func(ctx context.Context, args []Sequence) (Sequence, error) {
				if len(args) != 1 || len(args[0]) != 1 {
					return nil, &XPathError{Code: errCodeXPTY0004, Message: "array lookup requires exactly one argument"}
				}
				key, err := AtomizeItem(args[0][0])
				if err != nil {
					return nil, err
				}
				if !isIntegerDerived(key.TypeName) {
					return nil, &XPathError{Code: errCodeXPTY0004, Message: "array lookup requires xs:integer index"}
				}
				bi := key.BigInt()
				if !bi.IsInt64() {
					return nil, &XPathError{Code: "FOAY0001", Message: "array index out of range"}
				}
				idx := int(bi.Int64())
				return v.Get(idx)
			},
		}, nil
	default:
		return FunctionItem{}, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("expected function item, got %T", seq[0])}
	}
}
