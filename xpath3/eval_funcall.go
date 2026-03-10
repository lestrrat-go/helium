package xpath3

import (
	"context"
	"fmt"
)

func evalFunctionCall(ec *evalContext, e FunctionCall) (Sequence, error) {
	// Evaluate arguments
	args := make([]Sequence, len(e.Args))
	hasPlaceholders := false
	for i, argExpr := range e.Args {
		if _, ok := argExpr.(PlaceholderExpr); ok {
			hasPlaceholders = true
			continue
		}
		a, err := eval(ec, argExpr)
		if err != nil {
			return nil, err
		}
		args[i] = a
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

	fctx := withFnContext(ec.goCtx, ec)
	return fn.Call(fctx, args)
}

func evalDynamicFunctionCall(ec *evalContext, e DynamicFunctionCall) (Sequence, error) {
	funcSeq, err := eval(ec, e.Func)
	if err != nil {
		return nil, err
	}
	if len(funcSeq) != 1 {
		return nil, &XPathError{Code: "XPTY0004", Message: "dynamic function call requires single function item"}
	}
	// Evaluate arguments first (needed for all call types)
	args := make([]Sequence, len(e.Args))
	for i, argExpr := range e.Args {
		a, err := eval(ec, argExpr)
		if err != nil {
			return nil, err
		}
		args[i] = a
	}

	switch v := funcSeq[0].(type) {
	case FunctionItem:
		if v.Arity >= 0 && len(args) != v.Arity {
			return nil, fmt.Errorf("%w: expected %d arguments, got %d", ErrArityMismatch, v.Arity, len(args))
		}
		return v.Invoke(ec.goCtx, args)
	case MapItem:
		// Maps are functions: $map($key) → value
		if len(args) != 1 || len(args[0]) != 1 {
			return nil, &XPathError{Code: "XPTY0004", Message: "map lookup requires exactly one argument"}
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
	case ArrayItem:
		// Arrays are functions: $array($index) → member
		if len(args) != 1 || len(args[0]) != 1 {
			return nil, &XPathError{Code: "XPTY0004", Message: "array lookup requires exactly one argument"}
		}
		key, err := AtomizeItem(args[0][0])
		if err != nil {
			return nil, err
		}
		idx := int(key.ToFloat64())
		return v.Get(idx)
	default:
		return nil, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("dynamic function call requires function item, got %T", funcSeq[0])}
	}
}

func evalNamedFunctionRef(ec *evalContext, e NamedFunctionRef) (Sequence, error) {
	fn, err := resolveFunction(ec, e.Prefix, e.Name, e.Arity)
	if err != nil {
		return nil, err
	}
	minArity := fn.MinArity()
	fi := FunctionItem{
		Arity: e.Arity,
		Name:  e.Name,
		Invoke: func(ctx context.Context, args []Sequence) (Sequence, error) {
			if len(args) < minArity {
				return nil, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("fn:%s requires at least %d arguments, got %d", e.Name, minArity, len(args))}
			}
			return fn.Call(ctx, args)
		},
	}
	return Sequence{fi}, nil
}

func evalInlineFunctionExpr(ec *evalContext, e InlineFunctionExpr) (Sequence, error) {
	// Capture current variable scope snapshot
	closedVars := copyVars(ec.vars)
	fi := FunctionItem{
		Arity: len(e.Params),
		Invoke: func(ctx context.Context, args []Sequence) (Sequence, error) {
			innerCtx := &evalContext{
				goCtx:      ctx,
				node:       ec.node,
				position:   ec.position,
				size:       ec.size,
				vars:       copyVars(closedVars),
				namespaces: ec.namespaces,
				functions:  ec.functions,
				fnsNS:      ec.fnsNS,
				opCount:    ec.opCount,
				opLimit:    ec.opLimit,
				docOrder:   ec.docOrder,
				maxNodes:   ec.maxNodes,
			}
			for i, param := range e.Params {
				if i < len(args) {
					innerCtx.vars[param.Name] = args[i]
				}
			}
			return eval(innerCtx, e.Body)
		},
	}
	return Sequence{fi}, nil
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

	fi := FunctionItem{
		Arity: len(placeholderIndices),
		Name:  e.Name,
		Invoke: func(ctx context.Context, partialArgs []Sequence) (Sequence, error) {
			fullArgs := make([]Sequence, len(e.Args))
			copy(fullArgs, fixedArgs)
			pi := 0
			for _, idx := range placeholderIndices {
				if pi < len(partialArgs) {
					fullArgs[idx] = partialArgs[pi]
					pi++
				}
			}
			return fn.Call(ctx, fullArgs)
		},
	}
	return Sequence{fi}, nil
}

func evalMapConstructorExpr(ec *evalContext, e MapConstructorExpr) (Sequence, error) {
	entries := make([]MapEntry, len(e.Pairs))
	for i, pair := range e.Pairs {
		keySeq, err := eval(ec, pair.Key)
		if err != nil {
			return nil, err
		}
		if len(keySeq) != 1 {
			return nil, &XPathError{Code: "XPTY0004", Message: "map key must be a single atomic value"}
		}
		ka, err := AtomizeItem(keySeq[0])
		if err != nil {
			return nil, err
		}
		valSeq, err := eval(ec, pair.Value)
		if err != nil {
			return nil, err
		}
		entries[i] = MapEntry{Key: ka, Value: valSeq}
	}
	return Sequence{NewMap(entries)}, nil
}

func evalArrayConstructorExpr(ec *evalContext, e ArrayConstructorExpr) (Sequence, error) {
	if e.SquareBracket {
		// [a, b, c] — each expr is one member
		members := make([]Sequence, len(e.Items))
		for i, item := range e.Items {
			seq, err := eval(ec, item)
			if err != nil {
				return nil, err
			}
			members[i] = seq
		}
		return Sequence{NewArray(members)}, nil
	}
	// array { expr } — evaluate as sequence, each item is singleton member
	if len(e.Items) == 0 {
		return Sequence{NewArray(nil)}, nil
	}
	seq, err := eval(ec, e.Items[0])
	if err != nil {
		return nil, err
	}
	members := make([]Sequence, len(seq))
	for i, item := range seq {
		members[i] = Sequence{item}
	}
	return Sequence{NewArray(members)}, nil
}
