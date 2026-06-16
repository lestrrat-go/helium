package xpath3

import (
	"context"
	"fmt"
	"slices"

	"github.com/lestrrat-go/helium/internal/lexicon"
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
	callArgs := make([]Sequence, 1)
	for item := range seqItems(seq) {
		callArgs[0] = ItemSlice{item}
		r, err := fi.Invoke(ctx, callArgs)
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
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("fn:filter callback must have arity 1, got %d", fi.Arity)}
	}
	var result ItemSlice
	callArgs := make([]Sequence, 1)
	for item := range seqItems(seq) {
		callArgs[0] = ItemSlice{item}
		r, err := fi.Invoke(ctx, callArgs)
		if err != nil {
			return nil, err
		}
		// Per XPath 3.1, the callback must return exactly one xs:boolean
		if seqLen(r) != 1 {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "fn:filter callback must return a single xs:boolean value"}
		}
		av, ok := r.Get(0).(AtomicValue)
		if !ok || av.TypeName != TypeBoolean {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "fn:filter callback must return a single xs:boolean value"}
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
	callArgs := make([]Sequence, 2)
	for item := range seqItems(seq) {
		callArgs[0] = acc
		callArgs[1] = ItemSlice{item}
		acc, err = fi.Invoke(ctx, callArgs)
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
	callArgs := make([]Sequence, 2)
	for _, v := range slices.Backward(items) {
		callArgs[0] = ItemSlice{v}
		callArgs[1] = acc
		acc, err = fi.Invoke(ctx, callArgs)
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
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("fn:for-each-pair callback must have arity 2, got %d", fi.Arity)}
	}
	size := min(seqLen(seq1), seqLen(seq2))
	var result ItemSlice
	callArgs := make([]Sequence, 2)
	for i := range size {
		callArgs[0] = ItemSlice{seq1.Get(i)}
		callArgs[1] = ItemSlice{seq2.Get(i)}
		r, err := fi.Invoke(ctx, callArgs)
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
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "apply() second argument must be a single array"}
	}
	arr, ok := args[1].Get(0).(ArrayItem)
	if !ok {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "apply() second argument must be array"}
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
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "function-lookup() first argument must be QName"}
	}
	arityArg, err := extractSingleAtomicArg(args[1], "function-lookup()")
	if err != nil {
		return nil, err
	}
	arityArg, err = coerceToInteger(arityArg)
	if err != nil {
		return nil, err
	}
	arityVal, ok := arityArg.Int64Val()
	if !ok {
		return validNilSequence, nil
	}
	arity := int(arityVal)
	if arity < 0 {
		return validNilSequence, nil
	}

	fi, ok := lookupFunctionItem(ctx, nameArg.QNameVal(), arity)
	if !ok {
		return validNilSequence, nil
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
		capturedCtx = withFnContext(ctx, &capturedECValue)
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
				return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("fn:%s requires %d arguments, got %d", qv.Local, arity, len(callArgs))}
			}
			return fn.Call(ctx, callArgs)
		},
	}
	_ = capturedCtx
	fi.Invoke = func(_ context.Context, callArgs []Sequence) (Sequence, error) {
		if len(callArgs) != arity {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("fn:%s requires %d arguments, got %d", qv.Local, arity, len(callArgs))}
		}
		// Enforce the recorded parameter types, mirroring the named
		// function-reference path in eval_funcall.go.
		if paramTypes != nil {
			for i, arg := range callArgs {
				if i < len(paramTypes) {
					if _, ok := coerceToSequenceType(arg, paramTypes[i], nil); !ok {
						return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("fn:%s: argument %d does not match required type %v", qv.Local, i+1, paramTypes[i])}
					}
				}
			}
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
		return validNilSequence, nil
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
		return FunctionItem{}, &XPathError{Code: lexicon.ErrXPTY0004, Message: "expected single function item"}
	}
	// asFunctionItem is the single source of truth for adapting maps and arrays
	// (arity-1 lookup functions) into FunctionItem; see eval_funcall.go.
	fi, ok := asFunctionItem(seq.Get(0))
	if !ok {
		return FunctionItem{}, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("expected function item, got %T", seq.Get(0))}
	}
	return fi, nil
}
