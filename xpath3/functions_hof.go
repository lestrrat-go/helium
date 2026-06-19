package xpath3

import (
	"context"
	"fmt"

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

// fnMaxNodes returns the node-set/sequence length limit that accumulating
// built-ins (for-each, for-each-pair, map:for-each, array:flatten, ...) must
// honor via appendBounded. When the function is called outside an evaluation
// (ec == nil) or the evaluation did not set an explicit limit, the package
// default applies so unbounded materialization is still rejected.
func fnMaxNodes(ec *evalContext) int {
	if ec == nil || ec.maxNodes <= 0 {
		return maxNodeSetLength
	}
	return ec.maxNodes
}

// fnCountOp charges one operation against the evaluation's op-counter and
// honors context cancellation. It is a no-op (other than the cancellation
// check) when called outside an evaluation. Accumulating built-ins call it once
// per iteration so a long-running higher-order call respects op/time limits.
func fnCountOp(ctx context.Context, ec *evalContext) error {
	if ec != nil {
		return ec.countOps(ctx, 1)
	}
	return ctx.Err()
}

// fnCountOps charges n operations against the evaluation's op-counter and
// honors context cancellation. It is a no-op (other than the cancellation
// check) when called outside an evaluation. Built-ins that clone or materialize
// a whole sub-sequence in one shot (array:for-each, array:for-each-pair,
// array:join, array:flat-map, map:find) call it with the sub-sequence length
// BEFORE the bulk clone/append so the work is charged against OpLimit — a length
// below maxNodes but above OpLimit is still rejected with ErrOpLimit.
func fnCountOps(ctx context.Context, ec *evalContext, n int) error {
	if ec != nil {
		return ec.countOps(ctx, n)
	}
	return ctx.Err()
}

func fnForEach(ctx context.Context, args []Sequence) (Sequence, error) {
	seq := args[0]
	fi, err := extractFunctionItem(args[1])
	if err != nil {
		return nil, err
	}
	ec := getFnContext(ctx)
	maxNodes := fnMaxNodes(ec)
	var result ItemSlice
	callArgs := make([]Sequence, 1)
	for item := range seqItems(seq) {
		if err := fnCountOp(ctx, ec); err != nil {
			return nil, err
		}
		callArgs[0] = ItemSlice{item}
		r, err := fi.Invoke(ctx, callArgs)
		if err != nil {
			return nil, err
		}
		result, err = appendBoundedSeq(ctx, ec, result, r, maxNodes)
		if err != nil {
			return nil, err
		}
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
	ec := getFnContext(ctx)
	maxNodes := fnMaxNodes(ec)
	var result ItemSlice
	callArgs := make([]Sequence, 1)
	for item := range seqItems(seq) {
		if err := fnCountOp(ctx, ec); err != nil {
			return nil, err
		}
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
			result, err = appendBounded(result, []Item{item}, maxNodes)
			if err != nil {
				return nil, err
			}
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
	ec := getFnContext(ctx)
	maxNodes := fnMaxNodes(ec)
	callArgs := make([]Sequence, 2)
	for item := range seqItems(seq) {
		if err := fnCountOp(ctx, ec); err != nil {
			return nil, err
		}
		callArgs[0] = acc
		callArgs[1] = ItemSlice{item}
		acc, err = fi.Invoke(ctx, callArgs)
		if err != nil {
			return nil, err
		}
		// The accumulator can grow without bound across iterations; reject once
		// it would exceed the configured sequence/node-set limit.
		if maxNodes > 0 && seqLen(acc) > maxNodes {
			return nil, ErrNodeSetLimit
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
	ec := getFnContext(ctx)
	maxNodes := fnMaxNodes(ec)
	// Iterate from last to first WITHOUT materializing the input sequence, so a
	// lazy/borrowed input is never fully realized before the per-item op-count
	// and accumulator size-bound checks run (mirrors fnFoldLeft's streaming).
	callArgs := make([]Sequence, 2)
	for i := seqLen(seq); i > 0; i-- {
		if err := fnCountOp(ctx, ec); err != nil {
			return nil, err
		}
		callArgs[0] = ItemSlice{seq.Get(i - 1)}
		callArgs[1] = acc
		acc, err = fi.Invoke(ctx, callArgs)
		if err != nil {
			return nil, err
		}
		// The accumulator can grow without bound across iterations; reject once
		// it would exceed the configured sequence/node-set limit.
		if maxNodes > 0 && seqLen(acc) > maxNodes {
			return nil, ErrNodeSetLimit
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
	ec := getFnContext(ctx)
	maxNodes := fnMaxNodes(ec)
	size := min(seqLen(seq1), seqLen(seq2))
	var result ItemSlice
	callArgs := make([]Sequence, 2)
	for i := range size {
		if err := fnCountOp(ctx, ec); err != nil {
			return nil, err
		}
		callArgs[0] = ItemSlice{seq1.Get(i)}
		callArgs[1] = ItemSlice{seq2.Get(i)}
		r, err := fi.Invoke(ctx, callArgs)
		if err != nil {
			return nil, err
		}
		result, err = appendBoundedSeq(ctx, ec, result, r, maxNodes)
		if err != nil {
			return nil, err
		}
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
	// Reject negative or out-of-int arities (no such function); checking on
	// int64 first avoids a wrap to a valid-looking arity on 32-bit platforms.
	const maxInt = int(^uint(0) >> 1)
	if arityVal < 0 || arityVal > int64(maxInt) {
		return validNilSequence, nil
	}
	arity := int(arityVal)

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
	var capturedEC *evalContext
	if ec := getFnContext(ctx); ec != nil {
		capturedECValue := *ec
		capturedEC = &capturedECValue
		capturedCtx = withFnContext(ctx, capturedEC)
	}

	var paramTypes []SequenceType
	var returnType *SequenceType
	if sig := lookupFunctionSignature(qv.URI, qv.Local, arity); sig != nil {
		paramTypes = sig.ParamTypes
		returnType = sig.ReturnType
	} else if tf, ok := fn.(TypedFunction); ok {
		// User-defined typed functions expose their signature directly;
		// mirror evalNamedFunctionRef so function-lookup enforces the same
		// argument-type checks as the named-reference path (f#1).
		paramTypes = tf.FuncParamTypes()
		returnType = tf.FuncReturnType()
	} else if tfa, ok := fn.(TypedFunctionByArity); ok {
		paramTypes = tfa.FuncParamTypesForArity(arity)
		returnType = tfa.FuncReturnTypeForArity(arity)
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
	fi.Invoke = func(_ context.Context, callArgs []Sequence) (Sequence, error) {
		if len(callArgs) != arity {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("fn:%s requires %d arguments, got %d", qv.Local, arity, len(callArgs))}
		}
		// Enforce the recorded parameter types, mirroring the named
		// function-reference path in eval_funcall.go. Coercion may convert an
		// argument (e.g. xs:untypedAtomic -> xs:integer); the coerced value must
		// be what the function observes, so store it back into a copy of the
		// argument slice rather than discarding it and invoking with the original.
		if paramTypes != nil {
			coerced := make([]Sequence, len(callArgs))
			copy(coerced, callArgs)
			for i, arg := range callArgs {
				if i < len(paramTypes) {
					c, ok := coerceToSequenceType(arg, paramTypes[i], capturedEC)
					if !ok {
						return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("fn:%s: argument %d does not match required type %v", qv.Local, i+1, paramTypes[i])}
					}
					coerced[i] = c
				}
			}
			callArgs = coerced
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
