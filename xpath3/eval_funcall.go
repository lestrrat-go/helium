package xpath3

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-go/helium/internal/lexicon"
)

func evalFunctionCall(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e FunctionCall) (Sequence, error) {
	// Evaluate arguments
	args := make([]Sequence, len(e.Args))
	hasPlaceholders := false
	for i, argExpr := range e.Args {
		if _, ok := argExpr.(PlaceholderExpr); ok {
			hasPlaceholders = true
			continue
		}
		a, err := evalFn(ctx, ec, argExpr)
		if err != nil {
			return nil, err
		}
		args[i] = enrichNodeItems(ctx, ec, a)
	}

	// Partial application: if any args are placeholders, return FunctionItem
	if hasPlaceholders {
		return partialApply(ctx, ec, e, args)
	}

	// Resolve function, keeping the resolved identity so signature enforcement
	// keys off the resolved URI/local name (handles Q{uri}local) and only applies
	// the built-in signature when the resolved function is the actual built-in.
	r, err := resolveFunctionInfo(ctx, ec, e.Prefix, e.Name, len(args))
	if err != nil {
		return nil, err
	}

	// Enforce declared parameter signatures, mirroring the function-item /
	// named-function-reference path. Coerced values are stored back into args so
	// typed functions observe the converted values (e.g. xs:integer→xs:double).
	paramTypes := lookupParamTypes(r, len(args))
	if paramTypes != nil {
		for i, arg := range args {
			if i < len(paramTypes) {
				coerced, err := coerceFuncallArg(ctx, arg, paramTypes[i], r.name, i, ec)
				if err != nil {
					return nil, err
				}
				args[i] = coerced
			}
		}
	}

	return r.fn.Call(ec.fnContext(ctx), args)
}

// lookupParamTypes returns the declared parameter types for a resolved function.
// The built-in signature registry is consulted only when the resolved function is
// the built-in; user/registered functions use their own TypedFunction metadata.
// It returns nil when no signature is available (no enforcement).
func lookupParamTypes(r resolvedFunction, arity int) []SequenceType {
	if r.isBuiltin {
		if sig := lookupFunctionSignature(r.uri, r.name, arity); sig != nil {
			return sig.ParamTypes
		}
	}
	if tf, ok := r.fn.(TypedFunction); ok {
		return tf.FuncParamTypes()
	}
	if tfa, ok := r.fn.(TypedFunctionByArity); ok {
		return tfa.FuncParamTypesForArity(arity)
	}
	return nil
}

func evalDynamicFunctionCall(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e DynamicFunctionCall) (Sequence, error) {
	funcSeq, err := evalFn(ctx, ec, e.Func)
	if err != nil {
		return nil, err
	}
	if seqLen(funcSeq) != 1 {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "dynamic function call requires single function item"}
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
		a, err := evalFn(ctx, ec, argExpr)
		if err != nil {
			return nil, err
		}
		args[i] = enrichNodeItems(ctx, ec, a)
	}

	if hasPlaceholders {
		fi, ok := asFunctionItem(funcSeq.Get(0))
		if !ok {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "partial application requires function item"}
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
						Code:    lexicon.ErrXPTY0004,
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
		return v.Invoke(withDynamicCall(ec.fnContext(ctx)), args)
	case MapItem:
		// Maps are functions: $map($key) → value
		return mapLookup(v, args)
	case ArrayItem:
		// Arrays are functions: $array($index) → member
		return arrayLookup(v, args)
	default:
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("dynamic function call requires function item, got %T", funcSeq.Get(0))}
	}
}

// asFunctionItem adapts a callee item into a FunctionItem. In XPath 3.1 maps and
// arrays are function items of arity 1 (the lookup function), so they support
// both dynamic invocation and partial function application (e.g. $m(?), $a(?)).
func asFunctionItem(item Item) (FunctionItem, bool) {
	switch v := item.(type) {
	case FunctionItem:
		return v, true
	case MapItem:
		return FunctionItem{
			Arity:  1,
			Invoke: func(_ context.Context, args []Sequence) (Sequence, error) { return mapLookup(v, args) },
		}, true
	case ArrayItem:
		return FunctionItem{
			Arity:  1,
			Invoke: func(_ context.Context, args []Sequence) (Sequence, error) { return arrayLookup(v, args) },
		}, true
	default:
		return FunctionItem{}, false
	}
}

// mapLookup implements the map-as-function call $map($key) → value.
func mapLookup(m MapItem, args []Sequence) (Sequence, error) {
	if len(args) != 1 || seqLen(args[0]) != 1 {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "map lookup requires exactly one argument"}
	}
	key, err := AtomizeItem(args[0].Get(0))
	if err != nil {
		return nil, err
	}
	val, ok := m.Get(key)
	if !ok {
		return validNilSequence, nil
	}
	return val, nil
}

// arrayLookup implements the array-as-function call $array($index) → member.
func arrayLookup(a ArrayItem, args []Sequence) (Sequence, error) {
	if len(args) != 1 || seqLen(args[0]) != 1 {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "array lookup requires exactly one argument"}
	}
	key, err := AtomizeItem(args[0].Get(0))
	if err != nil {
		return nil, err
	}
	if key.TypeName == TypeUntypedAtomic {
		key, err = CastAtomic(key, TypeInteger)
		if err != nil {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "array lookup requires xs:integer index"}
		}
	}
	if !isIntegerDerived(key.TypeName) {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "array lookup requires xs:integer index"}
	}
	idx, err := checkedArrayIndex(key)
	if err != nil {
		return nil, err
	}
	return a.Get(idx)
}

func evalNamedFunctionRef(ctx context.Context, ec *evalContext, e NamedFunctionRef) (Sequence, error) {
	r, err := resolveFunctionInfo(ctx, ec, e.Prefix, e.Name, e.Arity)
	if err != nil {
		return nil, err
	}
	fn := r.fn

	// Check if the function needs to capture state at reference creation time.
	if sp, ok := fn.(DynamicRefSnapshotProvider); ok {
		if fi, ok := sp.DynamicRefSnapshot(ctx, e.Arity); ok {
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
	// Populate type signature. The built-in signature registry is consulted only
	// when the resolved function is the built-in itself; a user override of a
	// built-in name (e.g. abs#1) binds its own signature, mirroring the direct
	// call path (lookupParamTypes).
	var paramTypes []SequenceType
	var returnType *SequenceType
	var sig *functionSignature
	if r.isBuiltin {
		sig = lookupFunctionSignature(r.uri, r.name, e.Arity)
	}
	if sig != nil {
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
				return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("fn:%s requires at least %d arguments, got %d", e.Name, minArity, len(args))}
			}
			// Type-check arguments against declared parameter types. Coercion may
			// convert an argument (e.g. xs:integer -> xs:double); the coerced value
			// must be what the function observes, so store it back into a copy of the
			// argument slice rather than discarding it and invoking with the original
			// — mirroring the direct call path and fn:function-lookup.
			if paramTypes != nil {
				coerced := make([]Sequence, len(args))
				copy(coerced, args)
				for i, arg := range args {
					if i < len(paramTypes) {
						c, ok := coerceToSequenceType(ctx, arg, paramTypes[i], capturedEC)
						if !ok {
							return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("fn:%s: argument %d does not match required type %v", e.Name, i+1, paramTypes[i])}
						}
						coerced[i] = c
					}
				}
				args = coerced
			}
			// Use caller's ctx for cancellation, captured ec for focus/eval state
			return fn.Call(withFnContext(ctx, capturedEC), args)
		},
	}
	return ItemSlice{fi}, nil
}

func evalInlineFunctionExpr(evalFn exprEvaluator, _ context.Context, ec *evalContext, e InlineFunctionExpr) (Sequence, error) {
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
				return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("inline function requires %d arguments, got %d", len(e.Params), len(args))}
			}
			// Use caller's evalContext for mutable state (opCount, docOrder, docCache)
			// if available, otherwise fall back to the captured context.
			capturedECValue := *ec
			baseEC := &capturedECValue
			if callerEC := getFnContext(ctx); callerEC != nil {
				baseEC = callerEC
			}
			innerCtx := *baseEC
			innerCtx.node = nil
			innerCtx.contextItem = nil
			innerCtx.position = 0
			innerCtx.size = 0
			innerCtx.vars = closedVars
			for i, param := range e.Params {
				arg := args[i]
				// Apply function coercion rules if type specified
				if param.TypeHint != nil {
					coerced, ok := coerceToSequenceType(ctx, arg, *param.TypeHint, &innerCtx)
					if !ok {
						return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("inline function parameter $%s: value does not match required type %v", param.Name, *param.TypeHint)}
					}
					arg = coerced
				}
				innerCtx.vars = scopeWithBinding(innerCtx.vars, param.Name, arg)
			}
			result, err := evalFn(ctx, &innerCtx, e.Body)
			if err != nil {
				return nil, err
			}
			// Apply function coercion rules for return type if specified
			if e.ReturnType != nil {
				coerced, ok := coerceToSequenceType(ctx, result, *e.ReturnType, &innerCtx)
				if !ok {
					return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("inline function return value does not match required type %v", *e.ReturnType)}
				}
				result = coerced
			}
			return result, nil
		},
	}
	return ItemSlice{fi}, nil
}

func partialApply(ctx context.Context, ec *evalContext, e FunctionCall, fixedArgs []Sequence) (Sequence, error) {
	// Count placeholders to determine new arity
	var placeholderIndices []int
	for i, argExpr := range e.Args {
		if _, ok := argExpr.(PlaceholderExpr); ok {
			placeholderIndices = append(placeholderIndices, i)
		}
	}

	// Resolve the target function keeping its identity, so signature enforcement
	// keys off the resolved URI/local name and consults TypedFunction metadata —
	// mirroring evalFunctionCall. This closes the partial-application bypass where
	// a registered TypedFunction's parameter types went unenforced.
	r, err := resolveFunctionInfo(ctx, ec, e.Prefix, e.Name, len(e.Args))
	if err != nil {
		return nil, err
	}
	paramTypes := lookupParamTypes(r, len(e.Args))

	// Coerce the fixed (curried) arguments now, before the placeholders are
	// supplied: they are known at partial-application time, so the function body
	// must observe the converted values (e.g. xs:integer→xs:double) just like a
	// direct call.
	coercedFixed := make([]Sequence, len(fixedArgs))
	copy(coercedFixed, fixedArgs)
	if paramTypes != nil {
		for i, argExpr := range e.Args {
			if _, ok := argExpr.(PlaceholderExpr); ok {
				continue // placeholder slot: coerced at invocation time
			}
			if i >= len(paramTypes) {
				continue
			}
			coerced, err := coerceFuncallArg(ctx, fixedArgs[i], paramTypes[i], r.name, i, ec)
			if err != nil {
				return nil, err
			}
			coercedFixed[i] = coerced
		}
	}

	// Per XPath 3.1, partial applications are anonymous functions
	fi := FunctionItem{
		Arity: len(placeholderIndices),
		Invoke: func(ctx context.Context, partialArgs []Sequence) (Sequence, error) {
			if len(partialArgs) != len(placeholderIndices) {
				return nil, &XPathError{
					Code:    lexicon.ErrXPTY0004,
					Message: fmt.Sprintf("arity mismatch in partial application: expected %d arguments, got %d", len(placeholderIndices), len(partialArgs)),
				}
			}
			fullArgs := make([]Sequence, len(e.Args))
			copy(fullArgs, coercedFixed)
			for pi, idx := range placeholderIndices {
				fullArgs[idx] = partialArgs[pi]
			}
			// Coerce the placeholder-supplied arguments against the declared
			// parameter types and store the converted values back, mirroring
			// evalFunctionCall so typed functions observe promoted values and
			// typed errors (FOTY0013/FORG0001) propagate unchanged.
			if paramTypes != nil {
				for _, idx := range placeholderIndices {
					if idx >= len(paramTypes) {
						continue
					}
					coerced, err := coerceFuncallArg(ctx, fullArgs[idx], paramTypes[idx], r.name, idx, ec)
					if err != nil {
						return nil, err
					}
					fullArgs[idx] = coerced
				}
			}
			return r.fn.Call(ctx, fullArgs)
		},
	}
	return ItemSlice{fi}, nil
}

// coerceFuncallArg coerces one argument against its declared parameter type,
// translating the result into the funcall-enforcement error contract: a typed
// atomization/cast error (FOTY0013, FORG0001, …) surfaces unchanged so try/catch
// can dispatch on it; a plain type/cardinality mismatch becomes XPTY0004. This is
// the shared helper for evalFunctionCall's direct path and partialApply.
func coerceFuncallArg(ctx context.Context, arg Sequence, st SequenceType, fnName string, idx int, ec *evalContext) (Sequence, error) {
	coerced, err := coerceToSequenceTypeE(ctx, arg, st, ec)
	if err != nil {
		if !errors.Is(err, errCoerceMismatch) {
			return nil, err
		}
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fmt.Sprintf("fn:%s: argument %d does not match required type %v", fnName, idx+1, st)}
	}
	return coerced, nil
}

func evalMapConstructorExpr(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e MapConstructorExpr) (Sequence, error) {
	maxNodes := ec.maxNodes
	entries := make([]MapEntry, 0, len(e.Pairs))
	seen := make(map[mapKey]struct{}, len(e.Pairs))
	for _, pair := range e.Pairs {
		keySeq, err := evalFn(ctx, ec, pair.Key)
		if err != nil {
			return nil, err
		}
		if seqLen(keySeq) != 1 {
			return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "map key must be a single atomic value"}
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
		valSeq, err := evalFn(ctx, ec, pair.Value)
		if err != nil {
			return nil, err
		}
		// NewMap clones every value (materializing it), so bound the value length
		// against maxNodes and charge its length (min 1) of ops — honoring
		// OpLimit / context cancellation — BEFORE storing the entry. A huge/lazy
		// value is rejected here via the O(1) seqLen check instead of OOMing the
		// engine when NewMap materializes it.
		vLen := seqLen(valSeq)
		if maxNodes > 0 && vLen > maxNodes {
			return nil, ErrNodeSetLimit
		}
		if err := fnCountOps(ctx, ec, max(vLen, 1)); err != nil {
			return nil, err
		}
		entries = append(entries, MapEntry{Key: ka, Value: valSeq})
	}
	return ItemSlice{NewMap(entries)}, nil
}

func evalArrayConstructorExpr(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e ArrayConstructorExpr) (Sequence, error) {
	maxNodes := ec.maxNodes
	if e.SquareBracket {
		// [a, b, c] — each expr is one member. appendArrayMember bounds both the
		// member count and the total item count (NewArray clones every member,
		// materializing it) and charges the per-member op cost, so a huge/lazy
		// member cannot OOM the engine or bypass OpLimit / context cancellation.
		var members []Sequence
		total := 0
		for _, item := range e.Items {
			seq, err := evalFn(ctx, ec, item)
			if err != nil {
				return nil, err
			}
			members, total, err = appendArrayMember(ctx, ec, maxNodes, members, total, seq)
			if err != nil {
				return nil, err
			}
		}
		return ItemSlice{NewArray(members)}, nil
	}
	// array { expr } — evaluate as sequence, each item is a singleton member.
	if len(e.Items) == 0 {
		return ItemSlice{NewArray(nil)}, nil
	}
	seq, err := evalFn(ctx, ec, e.Items[0])
	if err != nil {
		return nil, err
	}
	// Drain the source lazily and bound each appended member: a huge/lazy source
	// is rejected once the member count would exceed maxNodes (or OpLimit /
	// cancellation fires) rather than materializing the whole structure up front.
	var members []Sequence
	total := 0
	for item := range seqItems(seq) {
		members, total, err = appendArrayMember(ctx, ec, maxNodes, members, total, ItemSlice{item})
		if err != nil {
			return nil, err
		}
	}
	return ItemSlice{NewArray(members)}, nil
}
