package xpath3

import (
	"fmt"
	"math/big"
	"strings"
)

var (
	maxArrayIndex = big.NewInt(int64(^uint(0) >> 1))
	minArrayIndex = big.NewInt(-int64(^uint(0)>>1) - 1)
)

// atomicToInteger extracts an int from an AtomicValue that is an integer type.
// Returns (index, true) if the value is xs:integer or derived, (0, false) otherwise.
func atomicToInteger(a AtomicValue) (int, bool) {
	if !isIntegerDerived(a.TypeName) {
		return 0, false
	}
	n, ok := a.Value.(*big.Int)
	if !ok {
		return 0, false
	}
	return int(n.Int64()), true
}

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

func evalLookupExpr(ec *evalContext, e LookupExpr) (Sequence, error) {
	base, err := eval(ec, e.Expr)
	if err != nil {
		return nil, err
	}
	var result Sequence
	for _, item := range base {
		r, err := lookupItem(ec, item, e.Key, e.All)
		if err != nil {
			return nil, err
		}
		result = append(result, r...)
	}
	return result, nil
}

func evalUnaryLookupExpr(ec *evalContext, e UnaryLookupExpr) (Sequence, error) {
	if ec.contextItem != nil {
		return lookupItem(ec, ec.contextItem, e.Key, e.All)
	}
	if ec.node == nil {
		return nil, &XPathError{Code: errCodeXPDY0002, Message: "context item is absent"}
	}
	return lookupItem(ec, NodeItem{Node: ec.node}, e.Key, e.All)
}

func lookupItem(ec *evalContext, item Item, keyExpr Expr, all bool) (Sequence, error) {
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
		keySeq, err := eval(ec, keyExpr)
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
			for _, m := range v.Members() {
				result = append(result, m...)
			}
			return result, nil
		}
		keySeq, err := eval(ec, keyExpr)
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
			if !isIntegerDerived(ka.TypeName) {
				return nil, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("array lookup key must be xs:integer, got %s", ka.TypeName)}
			}
			idx, err := checkedArrayIndex(ka)
			if err != nil {
				return nil, err
			}
			member, err := v.Get(idx)
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

func evalFLWOR(ec *evalContext, e FLWORExpr) (Sequence, error) {
	// Build tuples by processing clauses
	tuples := []flworTuple{{scope: ec.vars}}

	for _, clause := range e.Clauses {
		switch c := clause.(type) {
		case ForClause:
			var newTuples []flworTuple
			for _, tup := range tuples {
				subCtx := ec.withScope(tup.scope)
				domain, err := eval(subCtx, c.Expr)
				if err != nil {
					return nil, err
				}
				for i, item := range domain {
					scope := scopeWithBinding(tup.scope, c.Var, Sequence{item})
					if c.PosVar != "" {
						scope = scopeWithBinding(scope, c.PosVar, Sequence{AtomicValue{TypeName: TypeInteger, Value: big.NewInt(int64(i + 1))}})
					}
					newTuples = append(newTuples, flworTuple{scope: scope})
				}
			}
			tuples = newTuples
		case LetClause:
			for i := range tuples {
				subCtx := ec.withScope(tuples[i].scope)
				val, err := eval(subCtx, c.Expr)
				if err != nil {
					return nil, err
				}
				tuples[i].scope = scopeWithBinding(tuples[i].scope, c.Var, val)
			}
		}
	}

	// Evaluate return expression for each tuple
	var result Sequence
	for _, tup := range tuples {
		retCtx := ec.withScope(tup.scope)
		r, err := eval(retCtx, e.Return)
		if err != nil {
			return nil, err
		}
		result = append(result, r...)
	}
	return result, nil
}

type flworTuple struct {
	scope *variableScope
}

func evalQuantifiedExpr(ec *evalContext, e QuantifiedExpr) (Sequence, error) {
	return evalQuantifiedBindings(ec, e, 0)
}

func evalQuantifiedBindings(ec *evalContext, e QuantifiedExpr, idx int) (Sequence, error) {
	if idx >= len(e.Bindings) {
		// All bindings bound — evaluate satisfies
		r, err := eval(ec, e.Satisfies)
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
	domain, err := eval(ec, binding.Domain)
	if err != nil {
		return nil, err
	}
	for _, item := range domain {
		subCtx := ec.withScope(scopeWithBinding(ec.vars, binding.Var, Sequence{item}))
		result, err := evalQuantifiedBindings(subCtx, e, idx+1)
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

func evalIfExpr(ec *evalContext, e IfExpr) (Sequence, error) {
	cond, err := eval(ec, e.Cond)
	if err != nil {
		return nil, err
	}
	b, err := EBV(cond)
	if err != nil {
		return nil, err
	}
	if b {
		return eval(ec, e.Then)
	}
	return eval(ec, e.Else)
}

func evalTryCatchExpr(ec *evalContext, e TryCatchExpr) (Sequence, error) {
	result, err := eval(ec, e.Try)
	if err == nil {
		return result, nil
	}
	xpErr, ok := err.(*XPathError)
	if !ok {
		return nil, err // non-XPath errors propagate through
	}
	for _, catch := range e.Catches {
		if catchMatchesError(catch, xpErr) {
			return eval(buildCatchContext(ec, xpErr), catch.Expr)
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
