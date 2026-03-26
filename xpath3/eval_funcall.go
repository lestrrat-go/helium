package xpath3

import (
	"context"
	"fmt"
)

func evalFunctionCall(evalFn exprEvaluator, ec *evalContext, e FunctionCall) (Sequence, error) {
	// Evaluate arguments
	args := make([]Sequence, len(e.Args))
	hasPlaceholders := false
	for i, argExpr := range e.Args {
		if _, ok := argExpr.(PlaceholderExpr); ok {
			hasPlaceholders = true
			continue
		}
		a, err := evalFn(ec, argExpr)
		if err != nil {
			return nil, err
		}
		args[i] = enrichNodeItems(ec, a)
	}

	// Partial application: if any args are placeholders, return FunctionItem
	if hasPlaceholders {
		return partialApply(ec, e, args)
	}

	// Resolve function
	fn, err := resolveFunction(ec, e.Prefix, e.Name, len(args))
	if err != nil {
		return nil, err
	}

	return fn.Call(ec.fnContext(), args)
}

func evalDynamicFunctionCall(evalFn exprEvaluator, ec *evalContext, e DynamicFunctionCall) (Sequence, error) {
	funcSeq, err := evalFn(ec, e.Func)
	if err != nil {
		return nil, err
	}
	if seqLen(funcSeq) != 1 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "dynamic function call requires single function item"}
	}

	// Check for placeholder arguments (partial application)
	hasPlaceholders := false
	for _, argExpr := range e.Args {
		if _, ok := argExpr.(PlaceholderExpr); ok {
			hasPlaceholders = true
			break
		}
	}

	// Evaluate non-placeholder arguments
	args := make([]Sequence, len(e.Args))
	for i, argExpr := range e.Args {
		if _, ok := argExpr.(PlaceholderExpr); ok {
			continue
		}
		a, err := evalFn(ec, argExpr)
		if err != nil {
			return nil, err
		}
		args[i] = enrichNodeItems(ec, a)
	}

	if hasPlaceholders {
		fi, ok := funcSeq.Get(0).(FunctionItem)
		if !ok {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "partial application requires function item"}
		}
		// Check that the number of supplied arguments matches the function's arity
		if fi.Arity >= 0 && len(e.Args) != fi.Arity {
			return nil, fmt.Errorf("%w: expected %d arguments, got %d", ErrArityMismatch, fi.Arity, len(e.Args))
		}
		var placeholderIndices []int
		for i, argExpr := range e.Args {
			if _, ok := argExpr.(PlaceholderExpr); ok {
				placeholderIndices = append(placeholderIndices, i)
			}
		}
		fixedArgs := make([]Sequence, len(args))
		copy(fixedArgs, args)
		// Per XPath 3.1, partial applications are anonymous functions
		result := FunctionItem{
			Arity: len(placeholderIndices),
			Invoke: func(ctx context.Context, partialArgs []Sequence) (Sequence, error) {
				if len(partialArgs) != len(placeholderIndices) {
					return nil, &XPathError{
						Code:    errCodeXPTY0004,
						Message: fmt.Sprintf("arity mismatch: expected %d arguments, got %d", len(placeholderIndices), len(partialArgs)),
					}
				}
				fullArgs := make([]Sequence, len(e.Args))
				copy(fullArgs, fixedArgs)
				for pi, idx := range placeholderIndices {
					fullArgs[idx] = partialArgs[pi]
				}
				return fi.Invoke(ctx, fullArgs)
			},
		}
		return ItemSlice{result}, nil
	}

	switch v := funcSeq.Get(0).(type) {
	case FunctionItem:
		if v.Arity >= 0 && len(args) != v.Arity {
			return nil, fmt.Errorf("%w: expected %d arguments, got %d", ErrArityMismatch, v.Arity, len(args))
		}
		return v.Invoke(withDynamicCall(ec.fnContext()), args)
	case MapItem:
		// Maps are functions: $map($key) → value
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
	case ArrayItem:
		// Arrays are functions: $array($index) → member
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
		idx, err := checkedArrayIndex(key)
		if err != nil {
			return nil, err
		}
		return v.Get(idx)
	default:
		return nil, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("dynamic function call requires function item, got %T", funcSeq.Get(0))}
	}
}

func evalNamedFunctionRef(ec *evalContext, e NamedFunctionRef) (Sequence, error) {
	fn, err := resolveFunction(ec, e.Prefix, e.Name, e.Arity)
	if err != nil {
		return nil, err
	}

	// Check if the function needs to capture state at reference creation time.
	if sp, ok := fn.(DynamicRefSnapshotProvider); ok {
		if fi, ok := sp.DynamicRefSnapshot(ec.goCtx, e.Arity); ok {
			return ItemSlice{fi}, nil
		}
	}

	// Check if the function restricts dynamic references (e.g. current-group#0).
	// If so, create a function item that always raises the specified error.
	if dr, ok := fn.(DynamicRefRestricted); ok && dr.NoDynamicRef() {
		errCode := dr.DynRefErrorCode()
		fnName := e.Name
		ns, _ := resolvePrefix(ec, e.Prefix)
		fi := FunctionItem{
			Arity:     e.Arity,
			Name:      fnName,
			Namespace: ns,
			Invoke: func(_ context.Context, _ []Sequence) (Sequence, error) {
				return nil, &XPathError{Code: errCode, Message: fmt.Sprintf("%s: dynamic call to %s is not allowed", errCode, fnName)}
			},
		}
		return ItemSlice{fi}, nil
	}

	minArity := fn.MinArity()
	ns, _ := resolvePrefix(ec, e.Prefix)
	// Per XPath 3.1 Section 3.1.6: if the function is focus-dependent,
	// the dynamic context (including focus) is fixed at reference creation time.
	// Capture the evalContext snapshot for focus/variables, but use the caller's
	// context.Context at invocation time for cancellation propagation.
	capturedECValue := *ec
	capturedEC := &capturedECValue
	// Populate type signature from built-in registry or TypedFunction interface
	var paramTypes []SequenceType
	var returnType *SequenceType
	if sig := lookupFunctionSignature(ns, e.Name, e.Arity); sig != nil {
		paramTypes = sig.ParamTypes
		returnType = sig.ReturnType
	} else if tf, ok := fn.(TypedFunction); ok {
		paramTypes = tf.FuncParamTypes()
		returnType = tf.FuncReturnType()
	} else if tfa, ok := fn.(TypedFunctionByArity); ok {
		paramTypes = tfa.FuncParamTypesForArity(e.Arity)
		returnType = tfa.FuncReturnTypeForArity(e.Arity)
	}
	fi := FunctionItem{
		Arity:      e.Arity,
		Name:       e.Name,
		Namespace:  ns,
		ParamTypes: paramTypes,
		ReturnType: returnType,
		Invoke: func(ctx context.Context, args []Sequence) (Sequence, error) {
			if len(args) < minArity {
				return nil, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("fn:%s requires at least %d arguments, got %d", e.Name, minArity, len(args))}
			}
			// Type-check arguments against declared parameter types
			if paramTypes != nil {
				for i, arg := range args {
					if i < len(paramTypes) {
						if _, ok := coerceToSequenceType(arg, paramTypes[i], nil); !ok {
							return nil, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("fn:%s: argument %d does not match required type %v", e.Name, i+1, paramTypes[i])}
						}
					}
				}
			}
			// Use caller's ctx for cancellation, captured ec for focus/eval state
			return fn.Call(withFnContext(ctx, capturedEC), args)
		},
	}
	return ItemSlice{fi}, nil
}

func evalInlineFunctionExpr(evalFn exprEvaluator, ec *evalContext, e InlineFunctionExpr) (Sequence, error) {
	// Capture current variable scope snapshot
	closedVars := ec.vars
	// Collect parameter types for subtype checking
	var paramTypes []SequenceType
	for _, p := range e.Params {
		if p.TypeHint != nil {
			paramTypes = append(paramTypes, *p.TypeHint)
		}
	}
	if len(paramTypes) != len(e.Params) {
		paramTypes = nil // All-or-nothing: only set if all params are typed
	}
	fi := FunctionItem{
		Arity:      len(e.Params),
		ParamTypes: paramTypes,
		ReturnType: e.ReturnType,
		Invoke: func(ctx context.Context, args []Sequence) (Sequence, error) {
			if len(args) != len(e.Params) {
				return nil, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("inline function requires %d arguments, got %d", len(e.Params), len(args))}
			}
			// Use caller's evalContext for mutable state (opCount, docOrder, docCache)
			// if available, otherwise fall back to the captured context.
			capturedECValue := *ec
			baseEC := &capturedECValue
			if callerEC := getFnContext(ctx); callerEC != nil {
				baseEC = callerEC
			}
			innerCtx := *baseEC
			innerCtx.goCtx = ctx
			innerCtx.node = nil
			innerCtx.contextItem = nil
			innerCtx.position = 0
			innerCtx.size = 0
			innerCtx.vars = closedVars
			for i, param := range e.Params {
				arg := args[i]
				// Apply function coercion rules if type specified
				if param.TypeHint != nil {
					coerced, ok := coerceToSequenceType(arg, *param.TypeHint, &innerCtx)
					if !ok {
						return nil, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("inline function parameter $%s: value does not match required type %v", param.Name, *param.TypeHint)}
					}
					arg = coerced
				}
				innerCtx.vars = scopeWithBinding(innerCtx.vars, param.Name, arg)
			}
			result, err := evalFn(&innerCtx, e.Body)
			if err != nil {
				return nil, err
			}
			// Apply function coercion rules for return type if specified
			if e.ReturnType != nil {
				coerced, ok := coerceToSequenceType(result, *e.ReturnType, &innerCtx)
				if !ok {
					return nil, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("inline function return value does not match required type %v", *e.ReturnType)}
				}
				result = coerced
			}
			return result, nil
		},
	}
	return ItemSlice{fi}, nil
}

func partialApply(ec *evalContext, e FunctionCall, fixedArgs []Sequence) (Sequence, error) {
	// Count placeholders to determine new arity
	var placeholderIndices []int
	for i, argExpr := range e.Args {
		if _, ok := argExpr.(PlaceholderExpr); ok {
			placeholderIndices = append(placeholderIndices, i)
		}
	}

	fn, err := resolveFunction(ec, e.Prefix, e.Name, len(e.Args))
	if err != nil {
		return nil, err
	}

	// Look up type signature for type checking placeholder arguments
	ns, _ := resolvePrefix(ec, e.Prefix)
	var paramTypes []SequenceType
	if sig := lookupFunctionSignature(ns, e.Name, len(e.Args)); sig != nil {
		paramTypes = sig.ParamTypes
	}

	// Per XPath 3.1, partial applications are anonymous functions
	fi := FunctionItem{
		Arity: len(placeholderIndices),
		Invoke: func(ctx context.Context, partialArgs []Sequence) (Sequence, error) {
			if len(partialArgs) != len(placeholderIndices) {
				return nil, &XPathError{
					Code:    errCodeXPTY0004,
					Message: fmt.Sprintf("arity mismatch in partial application: expected %d arguments, got %d", len(placeholderIndices), len(partialArgs)),
				}
			}
			fullArgs := make([]Sequence, len(e.Args))
			copy(fullArgs, fixedArgs)
			for pi, idx := range placeholderIndices {
				fullArgs[idx] = partialArgs[pi]
			}
			// Type-check placeholder arguments against declared parameter types
			if paramTypes != nil {
				for pi, idx := range placeholderIndices {
					if idx < len(paramTypes) {
						if _, ok := coerceToSequenceType(partialArgs[pi], paramTypes[idx], nil); !ok {
							return nil, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("fn:%s: argument %d does not match required type %v", e.Name, idx+1, paramTypes[idx])}
						}
					}
				}
			}
			return fn.Call(ctx, fullArgs)
		},
	}
	return ItemSlice{fi}, nil
}

func evalMapConstructorExpr(evalFn exprEvaluator, ec *evalContext, e MapConstructorExpr) (Sequence, error) {
	entries := make([]MapEntry, len(e.Pairs))
	seen := make(map[mapKey]struct{}, len(e.Pairs))
	for i, pair := range e.Pairs {
		keySeq, err := evalFn(ec, pair.Key)
		if err != nil {
			return nil, err
		}
		if seqLen(keySeq) != 1 {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "map key must be a single atomic value"}
		}
		ka, err := AtomizeItem(keySeq.Get(0))
		if err != nil {
			return nil, err
		}
		// Per XPath 3.1, duplicate keys in a map constructor raise XQDY0137
		nk := normalizeMapKey(ka)
		if _, dup := seen[nk]; dup {
			return nil, &XPathError{Code: "XQDY0137", Message: fmt.Sprintf("duplicate key in map constructor: %v", ka.Value)}
		}
		seen[nk] = struct{}{}
		valSeq, err := evalFn(ec, pair.Value)
		if err != nil {
			return nil, err
		}
		entries[i] = MapEntry{Key: ka, Value: valSeq}
	}
	return ItemSlice{NewMap(entries)}, nil
}

func evalArrayConstructorExpr(evalFn exprEvaluator, ec *evalContext, e ArrayConstructorExpr) (Sequence, error) {
	if e.SquareBracket {
		// [a, b, c] — each expr is one member
		members := make([]Sequence, len(e.Items))
		for i, item := range e.Items {
			seq, err := evalFn(ec, item)
			if err != nil {
				return nil, err
			}
			members[i] = seq
		}
		return ItemSlice{NewArray(members)}, nil
	}
	// array { expr } — evaluate as sequence, each item is singleton member
	if len(e.Items) == 0 {
		return ItemSlice{NewArray(nil)}, nil
	}
	seq, err := evalFn(ec, e.Items[0])
	if err != nil {
		return nil, err
	}
	members := make([]Sequence, seqLen(seq))
	for i := range seqLen(seq) {
		members[i] = ItemSlice{seq.Get(i)}
	}
	return ItemSlice{NewArray(members)}, nil
}
