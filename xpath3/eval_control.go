package xpath3

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

var (
	maxArrayIndex = big.NewInt(int64(^uint(0) >> 1))
	minArrayIndex = big.NewInt(-int64(^uint(0)>>1) - 1)
)

func checkedArrayIndex(a AtomicValue) (int, error) {
	switch v := a.Value.(type) {
	case int64:
		return int(v), nil
	case *big.Int:
		if v.Cmp(minArrayIndex) < 0 || v.Cmp(maxArrayIndex) > 0 {
			return 0, &XPathError{Code: errCodeFOAY0001, Message: "array index out of range"}
		}
		return int(v.Int64()), nil
	default:
		return 0, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("array lookup key must be xs:integer, got %s", a.TypeName)}
	}
}

func evalLookupExpr(evalFn exprEvaluator, goCtx context.Context, ec *evalContext, e LookupExpr) (Sequence, error) {
	base, err := evalFn(goCtx, ec, e.Expr)
	if err != nil {
		return nil, err
	}
	var result ItemSlice
	for item := range seqItems(base) {
		r, err := lookupItem(evalFn, goCtx, ec, item, e.Key, e.All)
		if err != nil {
			return nil, err
		}
		result = append(result, seqMaterialize(r)...)
	}
	return result, nil
}

func evalUnaryLookupExpr(evalFn exprEvaluator, goCtx context.Context, ec *evalContext, e UnaryLookupExpr) (Sequence, error) {
	if ec.contextItem != nil {
		return lookupItem(evalFn, goCtx, ec, ec.contextItem, e.Key, e.All)
	}
	if ec.node == nil {
		return nil, &XPathError{Code: errCodeXPDY0002, Message: "context item is absent"}
	}
	return lookupItem(evalFn, goCtx, ec, NodeItem{Node: ec.node}, e.Key, e.All)
}

func lookupItem(evalFn exprEvaluator, goCtx context.Context, ec *evalContext, item Item, keyExpr Expr, all bool) (Sequence, error) {
	switch v := item.(type) {
	case MapItem:
		if all {
			var result ItemSlice
			_ = v.ForEach(func(_ AtomicValue, val Sequence) error {
				result = append(result, seqMaterialize(val)...)
				return nil
			})
			return result, nil
		}
		keySeq, err := evalFn(goCtx, ec, keyExpr)
		if err != nil {
			return nil, err
		}
		if seqLen(keySeq) == 0 {
			return nil, nil //nolint:nilnil
		}
		var result ItemSlice
		for keyItem := range seqItems(keySeq) {
			ka, err := AtomizeItem(keyItem)
			if err != nil {
				return nil, err
			}
			val, ok := v.Get(ka)
			if ok {
				result = append(result, seqMaterialize(val)...)
			}
		}
		return result, nil
	case ArrayItem:
		if all {
			var result ItemSlice
			for _, m := range v.members0() {
				result = append(result, seqMaterialize(m)...)
			}
			return result, nil
		}
		keySeq, err := evalFn(goCtx, ec, keyExpr)
		if err != nil {
			return nil, err
		}
		if seqLen(keySeq) == 0 {
			return nil, nil //nolint:nilnil
		}
		var result ItemSlice
		for keyItem := range seqItems(keySeq) {
			ka, err := AtomizeItem(keyItem)
			if err != nil {
				return nil, err
			}
			if ka.TypeName == TypeUntypedAtomic {
				ka, err = CastAtomic(ka, TypeInteger)
				if err != nil {
					return nil, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("array lookup key must be xs:integer, got %s", ka.TypeName)}
				}
			}
			if !isIntegerDerived(ka.TypeName) {
				return nil, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("array lookup key must be xs:integer, got %s", ka.TypeName)}
			}
			idx, err := checkedArrayIndex(ka)
			if err != nil {
				return nil, err
			}
			member, err := v.get0(idx)
			if err != nil {
				return nil, err
			}
			result = append(result, seqMaterialize(member)...)
		}
		return result, nil
	default:
		return nil, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("lookup requires map or array, got %T", item)}
	}
}

// tupleConsumer receives completed variable scopes from the FLWOR clause
// pipeline. Each call to ConsumeTuple represents one binding tuple that
// has passed through all clauses.
type tupleConsumer interface {
	ConsumeTuple(scope *variableScope) error
}

// tupleConsumerFunc adapts a plain function to the tupleConsumer interface.
type tupleConsumerFunc func(scope *variableScope) error

func (f tupleConsumerFunc) ConsumeTuple(scope *variableScope) error {
	return f(scope)
}

func evalFLWOR(evalFn exprEvaluator, goCtx context.Context, ec *evalContext, e FLWORExpr) (Sequence, error) {
	var result ItemSlice
	consumer := tupleConsumerFunc(func(scope *variableScope) error {
		oldScope := ec.pushScope(scope)
		r, err := evalFn(goCtx, ec, e.Return)
		ec.restoreScope(oldScope)
		if err != nil {
			return err
		}
		result = append(result, seqMaterialize(r)...)
		return nil
	})

	if err := iterateFLWORClauses(evalFn, goCtx, ec, e.Clauses, 0, ec.vars, consumer); err != nil {
		return nil, err
	}
	return result, nil
}

// iterateFLWORClauses processes clauses[i..] recursively, streaming each
// completed scope to the consumer instead of materializing all tuples.
func iterateFLWORClauses(evalFn exprEvaluator, goCtx context.Context, ec *evalContext, clauses []FLWORClause, i int, scope *variableScope, consumer tupleConsumer) error {
	if i >= len(clauses) {
		return consumer.ConsumeTuple(scope)
	}

	if err := ec.countOps(goCtx, 1); err != nil {
		return err
	}

	switch c := clauses[i].(type) {
	case ForClause:
		oldScope := ec.pushScope(scope)
		domain, err := evalFn(goCtx, ec, c.Expr)
		ec.restoreScope(oldScope)
		if err != nil {
			return err
		}
		pos := 0
		for item := range seqItems(domain) {
			inner := scopeWithBinding(scope, c.Var, ItemSlice{item})
			if c.PosVar != "" {
				inner = scopeWithBinding(inner, c.PosVar, ItemSlice{AtomicValue{TypeName: TypeInteger, Value: int64(pos + 1)}})
			}
			if err := iterateFLWORClauses(evalFn, goCtx, ec, clauses, i+1, inner, consumer); err != nil {
				return err
			}
			pos++
		}
		return nil

	case LetClause:
		oldScope := ec.pushScope(scope)
		val, err := evalFn(goCtx, ec, c.Expr)
		ec.restoreScope(oldScope)
		if err != nil {
			return err
		}
		inner := scopeWithBinding(scope, c.Var, val)
		return iterateFLWORClauses(evalFn, goCtx, ec, clauses, i+1, inner, consumer)

	default:
		return iterateFLWORClauses(evalFn, goCtx, ec, clauses, i+1, scope, consumer)
	}
}

func evalQuantifiedExpr(evalFn exprEvaluator, goCtx context.Context, ec *evalContext, e QuantifiedExpr) (Sequence, error) {
	return evalQuantifiedBindings(evalFn, goCtx, ec, e, 0)
}

func evalQuantifiedBindings(evalFn exprEvaluator, goCtx context.Context, ec *evalContext, e QuantifiedExpr, idx int) (Sequence, error) {
	if idx >= len(e.Bindings) {
		// All bindings bound — evaluate satisfies
		r, err := evalFn(goCtx, ec, e.Satisfies)
		if err != nil {
			return nil, err
		}
		b, err := EBV(r)
		if err != nil {
			return nil, err
		}
		return SingleBoolean(b), nil
	}
	binding := e.Bindings[idx]
	domain, err := evalFn(goCtx, ec, binding.Domain)
	if err != nil {
		return nil, err
	}
	for item := range seqItems(domain) {
		oldScope := ec.pushScope(scopeWithBinding(ec.vars, binding.Var, ItemSlice{item}))
		result, err := evalQuantifiedBindings(evalFn, goCtx, ec, e, idx+1)
		ec.restoreScope(oldScope)
		if err != nil {
			return nil, err
		}
		b, err := EBV(result)
		if err != nil {
			return nil, err
		}
		if e.Some && b {
			return SingleBoolean(true), nil
		}
		if !e.Some && !b {
			return SingleBoolean(false), nil
		}
	}
	if e.Some {
		return SingleBoolean(false), nil
	}
	return SingleBoolean(true), nil
}

func evalIfExpr(evalFn exprEvaluator, goCtx context.Context, ec *evalContext, e IfExpr) (Sequence, error) {
	cond, err := evalFn(goCtx, ec, e.Cond)
	if err != nil {
		return nil, err
	}
	b, err := EBV(cond)
	if err != nil {
		return nil, err
	}
	if b {
		return evalFn(goCtx, ec, e.Then)
	}
	return evalFn(goCtx, ec, e.Else)
}

func evalTryCatchExpr(evalFn exprEvaluator, goCtx context.Context, ec *evalContext, e TryCatchExpr) (Sequence, error) {
	result, err := evalFn(goCtx, ec, e.Try)
	if err == nil {
		return result, nil
	}
	xpErr, ok := errors.AsType[*XPathError](err)
	if !ok {
		return nil, err // non-XPath errors propagate through
	}
	for _, catch := range e.Catches {
		if catchMatchesError(catch, xpErr) {
			return evalFn(goCtx, buildCatchContext(ec, xpErr), catch.Expr)
		}
	}
	return nil, err // no matching catch
}

// catchMatchesError checks if a catch clause matches an error.
func catchMatchesError(catch CatchClause, xpErr *XPathError) bool {
	if len(catch.Codes) == 0 {
		return true // wildcard catch (*)
	}
	errQName := xpErr.qname()
	for _, code := range catch.Codes {
		if catchCodeMatches(code, errQName) {
			return true
		}
	}
	return false
}

// catchCodeMatches compares a catch clause code against an XPath error QName.
// The catch code may be:
//   - "*" — matches anything
//   - "err:FOAR0002" or "FOAR0002" — matches specific error code
//   - "err:*" — matches any code in the err namespace (all XPath errors)
//   - "*:FOAR0002" — matches any namespace with that local name
//   - "Q{http://...}FOAR0002" — matches by URI + local name
//   - "Q{http://...}*" — matches any code in that namespace
func catchCodeMatches(catchCode string, errQName QNameValue) bool {
	if catchCode == "*" {
		return true
	}

	// Handle Q{uri}local and Q{uri}*
	if strings.HasPrefix(catchCode, "Q{") {
		if idx := strings.Index(catchCode, "}"); idx >= 0 {
			uri := catchCode[2:idx]
			local := catchCode[idx+1:]
			if local == "*" {
				return errQName.URI == uri
			}
			return errQName.URI == uri && errQName.Local == local
		}
	}

	// Extract local part from catch code (strip prefix)
	catchLocal := catchCode
	catchPrefix := ""
	if p, l, ok := strings.Cut(catchCode, ":"); ok {
		catchPrefix = p
		catchLocal = l
	}

	// Wildcard forms
	if catchLocal == "*" {
		// prefix:* — for err:*, matches all XPath errors
		return catchPrefix == "" || (catchPrefix == "err" && errQName.URI == NSErr)
	}
	if catchPrefix == "*" {
		return catchLocal == errQName.Local // *:CODE matches the bare code
	}
	if catchPrefix == "err" {
		return errQName.URI == NSErr && catchLocal == errQName.Local
	}
	if catchPrefix != "" {
		return false
	}

	// Compare the local part of the catch code against the error code
	return catchLocal == errQName.Local
}

// buildCatchContext creates an eval context with standard $err:* variables.
func buildCatchContext(ec *evalContext, xpErr *XPathError) *evalContext {
	errQName := xpErr.qname()
	if errQName.Local == "" {
		errQName = QNameValue{Prefix: "err", URI: NSErr, Local: errCodeFOER0000}
	}
	errQN := SingleAtomic(AtomicValue{
		TypeName: TypeQName,
		Value:    errQName,
	})
	ctx := ec.withVar("err:code", errQN)
	ctx = ctx.withVar("err:description", SingleString(xpErr.Message))
	ctx = ctx.withVar("err:value", nil)
	ctx = ctx.withVar("err:module", nil)
	ctx = ctx.withVar("err:line-number", nil)
	ctx = ctx.withVar("err:column-number", nil)
	ctx = ctx.withVar("err:additional", nil)
	return ctx
}
