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
	var result ItemSlice
	for item := range seqItems(seq) {
		r, err := fi.Invoke(ctx, []Sequence{ItemSlice{item}})
		if err != nil {
			return nil, err
		}
		result = append(result, seqMaterialize(r)...)
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
	var result ItemSlice
	for item := range seqItems(seq) {
		r, err := fi.Invoke(ctx, []Sequence{ItemSlice{item}})
		if err != nil {
			return nil, err
		}
		// Per XPath 3.1, the callback must return exactly one xs:boolean
		if seqLen(r) != 1 {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "fn:filter callback must return a single xs:boolean value"}
		}
		av, ok := r.Get(0).(AtomicValue)
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
	for item := range seqItems(seq) {
		acc, err = fi.Invoke(ctx, []Sequence{acc, ItemSlice{item}})
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
	items := seqMaterialize(seq)
	for i := len(items) - 1; i >= 0; i-- {
		acc, err = fi.Invoke(ctx, []Sequence{ItemSlice{items[i]}, acc})
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
	size := seqLen(seq1)
	if seqLen(seq2) < size {
		size = seqLen(seq2)
	}
	var result ItemSlice
	for i := 0; i < size; i++ {
		r, err := fi.Invoke(ctx, []Sequence{ItemSlice{seq1.Get(i)}, ItemSlice{seq2.Get(i)}})
		if err != nil {
			return nil, err
		}
		result = append(result, seqMaterialize(r)...)
	}
	return result, nil
}

func fnApply(ctx context.Context, args []Sequence) (Sequence, error) {
	fi, err := extractFunctionItem(args[0])
	if err != nil {
		return nil, err
	}
	if seqLen(args[1]) != 1 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "apply() second argument must be a single array"}
	}
	arr, ok := args[1].Get(0).(ArrayItem)
	if !ok {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "apply() second argument must be array"}
	}
	if fi.Arity >= 0 && arr.Size() != fi.Arity {
		return nil, &XPathError{Code: errCodeFOAP0001, Message: fmt.Sprintf("fn:apply: function has arity %d but array has %d members", fi.Arity, arr.Size())}
	}
	members := arr.members0()
	fnArgs := make([]Sequence, len(members))
	copy(fnArgs, members)
	return fi.Invoke(ctx, fnArgs)
}

// CallFunctionLookup is an exported wrapper around the built-in
// function-lookup implementation. It is used by the XSLT layer to
// delegate to the standard function-lookup before applying
// package-specific adjustments.
func CallFunctionLookup(ctx context.Context, args []Sequence) (Sequence, error) {
	return fnFunctionLookup(ctx, args)
}

func fnFunctionLookup(ctx context.Context, args []Sequence) (Sequence, error) {
	nameArg, err := extractSingleAtomicArg(args[0], "function-lookup()")
	if err != nil {
		return nil, err
	}
	if nameArg.TypeName != TypeQName {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "function-lookup() first argument must be QName"}
	}
	arityArg, err := extractSingleAtomicArg(args[1], "function-lookup()")
	if err != nil {
		return nil, err
	}
	arityArg, err = coerceToInteger(arityArg)
	if err != nil {
		return nil, err
	}
	arityBig := arityArg.BigInt()
	if !arityBig.IsInt64() {
		return nil, nil
	}
	arity := int(arityBig.Int64())
	if arity < 0 {
		return nil, nil
	}

	fi, ok := lookupFunctionItem(ctx, nameArg.QNameVal(), arity)
	if !ok {
		return nil, nil
	}
	return ItemSlice{fi}, nil
}

func lookupFunctionItem(ctx context.Context, qv QNameValue, arity int) (FunctionItem, bool) {
	var (
		fn Function
		ok bool
	)

	if ec := getFnContext(ctx); ec != nil {
		if qv.URI == "" && ec.functions != nil {
			fn, ok = ec.functions[qv.Local]
			if ok && checkArity(fn, qv.Local, arity) != nil {
				ok = false
			}
		}
		if !ok && ec.fnsNS != nil {
			fn, ok = ec.fnsNS[QualifiedName{URI: qv.URI, Name: qv.Local}]
			if ok && checkArity(fn, qv.Local, arity) != nil {
				ok = false
			}
		}
	}
	if !ok {
		fn, ok = builtinFunctions3[QualifiedName{URI: qv.URI, Name: qv.Local}]
		if ok && checkArity(fn, qv.Local, arity) != nil {
			ok = false
		}
	}
	if !ok {
		return FunctionItem{}, false
	}

	capturedCtx := ctx
	if ec := getFnContext(ctx); ec != nil {
		capturedECValue := *ec
		capturedCtx = withFnContext(ec.goCtx, &capturedECValue)
	}

	var paramTypes []SequenceType
	var returnType *SequenceType
	if sig := lookupFunctionSignature(qv.URI, qv.Local, arity); sig != nil {
		paramTypes = sig.ParamTypes
		returnType = sig.ReturnType
	}

	fi := FunctionItem{
		Arity:      arity,
		Name:       qv.Local,
		Namespace:  qv.URI,
		ParamTypes: paramTypes,
		ReturnType: returnType,
		Invoke: func(ctx context.Context, callArgs []Sequence) (Sequence, error) {
			if len(callArgs) != arity {
				return nil, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("fn:%s requires %d arguments, got %d", qv.Local, arity, len(callArgs))}
			}
			return fn.Call(ctx, callArgs)
		},
	}
	_ = capturedCtx
	fi.Invoke = func(_ context.Context, callArgs []Sequence) (Sequence, error) {
		if len(callArgs) != arity {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("fn:%s requires %d arguments, got %d", qv.Local, arity, len(callArgs))}
		}
		return fn.Call(capturedCtx, callArgs)
	}
	return fi, true
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
	if seqLen(seq) != 1 {
		return FunctionItem{}, &XPathError{Code: errCodeXPTY0004, Message: "expected single function item"}
	}
	switch v := seq.Get(0).(type) {
	case FunctionItem:
		return v, nil
	case MapItem:
		// Maps are functions: map($key) → value
		return FunctionItem{
			Arity: 1,
			Invoke: func(ctx context.Context, args []Sequence) (Sequence, error) {
				if len(args) != 1 || seqLen(args[0]) != 1 {
					return nil, &XPathError{Code: errCodeXPTY0004, Message: "map lookup requires exactly one argument"}
				}
				key, err := AtomizeItem(args[0].Get(0))
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
				if len(args) != 1 || seqLen(args[0]) != 1 {
					return nil, &XPathError{Code: errCodeXPTY0004, Message: "array lookup requires exactly one argument"}
				}
				key, err := AtomizeItem(args[0].Get(0))
				if err != nil {
					return nil, err
				}
				if key.TypeName == TypeUntypedAtomic {
					key, err = CastAtomic(key, TypeInteger)
					if err != nil {
						return nil, &XPathError{Code: errCodeXPTY0004, Message: "array lookup requires xs:integer index"}
					}
				}
				if !isIntegerDerived(key.TypeName) {
					return nil, &XPathError{Code: errCodeXPTY0004, Message: "array lookup requires xs:integer index"}
				}
				bi := key.BigInt()
				if !bi.IsInt64() {
					return nil, &XPathError{Code: errCodeFOAY0001, Message: "array index out of range"}
				}
				idx := int(bi.Int64())
				return v.Get(idx)
			},
		}, nil
	default:
		return FunctionItem{}, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("expected function item, got %T", seq.Get(0))}
	}
}
