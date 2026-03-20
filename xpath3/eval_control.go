package xpath3

import (
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
	n, ok := a.Value.(*big.Int)
	if !ok {
		return 0, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("array lookup key must be xs:integer, got %s", a.TypeName)}
	}
	if n.Cmp(minArrayIndex) < 0 || n.Cmp(maxArrayIndex) > 0 {
		return 0, &XPathError{Code: errCodeFOAY0001, Message: "array index out of range"}
	}
	return int(n.Int64()), nil
}

func evalLookupExpr(evalFn exprEvaluator, ec *evalContext, e LookupExpr) (Sequence, error) {
	base, err := evalFn(ec, e.Expr)
	if err != nil {
		return nil, err
	}
	var result Sequence
	for _, item := range base {
		r, err := lookupItem(evalFn, ec, item, e.Key, e.All)
		if err != nil {
			return nil, err
		}
		result = append(result, r...)
	}
	return result, nil
}

func evalUnaryLookupExpr(evalFn exprEvaluator, ec *evalContext, e UnaryLookupExpr) (Sequence, error) {
	if ec.contextItem != nil {
		return lookupItem(evalFn, ec, ec.contextItem, e.Key, e.All)
	}
	if ec.node == nil {
		return nil, &XPathError{Code: errCodeXPDY0002, Message: "context item is absent"}
	}
	return lookupItem(evalFn, ec, NodeItem{Node: ec.node}, e.Key, e.All)
}

func lookupItem(evalFn exprEvaluator, ec *evalContext, item Item, keyExpr Expr, all bool) (Sequence, error) {
	switch v := item.(type) {
	case MapItem:
		if all {
			var result Sequence
			_ = v.ForEach(func(_ AtomicValue, val Sequence) error {
				result = append(result, val...)
				return nil
			})
			return result, nil
		}
		keySeq, err := evalFn(ec, keyExpr)
		if err != nil {
			return nil, err
		}
		if len(keySeq) == 0 {
			return nil, nil
		}
		var result Sequence
		for _, keyItem := range keySeq {
			ka, err := AtomizeItem(keyItem)
			if err != nil {
				return nil, err
			}
			val, ok := v.Get(ka)
			if ok {
				result = append(result, val...)
			}
		}
		return result, nil
	case ArrayItem:
		if all {
			var result Sequence
			for _, m := range v.members0() {
				result = append(result, m...)
			}
			return result, nil
		}
		keySeq, err := evalFn(ec, keyExpr)
		if err != nil {
			return nil, err
		}
		if len(keySeq) == 0 {
			return nil, nil
		}
		var result Sequence
		for _, keyItem := range keySeq {
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
			result = append(result, member...)
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

func evalFLWOR(evalFn exprEvaluator, ec *evalContext, e FLWORExpr) (Sequence, error) {
	var result Sequence
	consumer := tupleConsumerFunc(func(scope *variableScope) error {
		retCtx := ec.withScope(scope)
		r, err := evalFn(retCtx, e.Return)
		if err != nil {
			return err
		}
		result = append(result, r...)
		return nil
	})

	if err := iterateFLWORClauses(evalFn, ec, e.Clauses, 0, ec.vars, consumer); err != nil {
		return nil, err
	}
	return result, nil
}

// iterateFLWORClauses processes clauses[i..] recursively, streaming each
// completed scope to the consumer instead of materializing all tuples.
func iterateFLWORClauses(evalFn exprEvaluator, ec *evalContext, clauses []FLWORClause, i int, scope *variableScope, consumer tupleConsumer) error {
	if i >= len(clauses) {
		return consumer.ConsumeTuple(scope)
	}

	if err := ec.countOps(1); err != nil {
		return err
	}

	switch c := clauses[i].(type) {
	case ForClause:
		subCtx := ec.withScope(scope)
		domain, err := evalFn(subCtx, c.Expr)
		if err != nil {
			return err
		}
		for pos, item := range domain {
			inner := scopeWithBinding(scope, c.Var, Sequence{item})
			if c.PosVar != "" {
				inner = scopeWithBinding(inner, c.PosVar, Sequence{AtomicValue{TypeName: TypeInteger, Value: big.NewInt(int64(pos + 1))}})
			}
			if err := iterateFLWORClauses(evalFn, ec, clauses, i+1, inner, consumer); err != nil {
				return err
			}
		}
		return nil

	case LetClause:
		subCtx := ec.withScope(scope)
		val, err := evalFn(subCtx, c.Expr)
		if err != nil {
			return err
		}
		inner := scopeWithBinding(scope, c.Var, val)
		return iterateFLWORClauses(evalFn, ec, clauses, i+1, inner, consumer)

	default:
		return iterateFLWORClauses(evalFn, ec, clauses, i+1, scope, consumer)
	}
}

func evalQuantifiedExpr(evalFn exprEvaluator, ec *evalContext, e QuantifiedExpr) (Sequence, error) {
	return evalQuantifiedBindings(evalFn, ec, e, 0)
}

func evalQuantifiedBindings(evalFn exprEvaluator, ec *evalContext, e QuantifiedExpr, idx int) (Sequence, error) {
	if idx >= len(e.Bindings) {
		// All bindings bound — evaluate satisfies
		r, err := evalFn(ec, e.Satisfies)
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
	domain, err := evalFn(ec, binding.Domain)
	if err != nil {
		return nil, err
	}
	for _, item := range domain {
		subCtx := ec.withScope(scopeWithBinding(ec.vars, binding.Var, Sequence{item}))
		result, err := evalQuantifiedBindings(evalFn, subCtx, e, idx+1)
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

func evalIfExpr(evalFn exprEvaluator, ec *evalContext, e IfExpr) (Sequence, error) {
	cond, err := evalFn(ec, e.Cond)
	if err != nil {
		return nil, err
	}
	b, err := EBV(cond)
	if err != nil {
		return nil, err
	}
	if b {
		return evalFn(ec, e.Then)
	}
	return evalFn(ec, e.Else)
}

func evalTryCatchExpr(evalFn exprEvaluator, ec *evalContext, e TryCatchExpr) (Sequence, error) {
	result, err := evalFn(ec, e.Try)
	if err == nil {
		return result, nil
	}
	xpErr, ok := errors.AsType[*XPathError](err)
	if !ok {
		return nil, err // non-XPath errors propagate through
	}
	for _, catch := range e.Catches {
		if catchMatchesError(catch, xpErr) {
			return evalFn(buildCatchContext(ec, xpErr), catch.Expr)
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
	if idx := strings.IndexByte(catchCode, ':'); idx >= 0 {
		catchPrefix = catchCode[:idx]
		catchLocal = catchCode[idx+1:]
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
