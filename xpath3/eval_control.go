package xpath3

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
)

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
		return nil, &XPathError{Code: "XPDY0002", Message: "context item is absent"}
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
		ka, err := AtomizeItem(keySeq[0])
		if err != nil {
			return nil, err
		}
		val, ok := v.Get(ka)
		if !ok {
			return nil, nil
		}
		return val, nil
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
		ka, err := AtomizeItem(keySeq[0])
		if err != nil {
			return nil, err
		}
		idx := int(ka.ToFloat64())
		return v.Get(idx)
	default:
		return nil, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("lookup requires map or array, got %T", item)}
	}
}

func evalFLWOR(ec *evalContext, e FLWORExpr) (Sequence, error) {
	// Build tuples by processing clauses
	tuples := []flworTuple{{vars: copyVars(ec.vars)}}

	for _, clause := range e.Clauses {
		switch c := clause.(type) {
		case ForClause:
			var newTuples []flworTuple
			for _, tup := range tuples {
				subCtx := ec.withVars(tup.vars)
				domain, err := eval(subCtx, c.Expr)
				if err != nil {
					return nil, err
				}
				for i, item := range domain {
					newVars := copyVars(tup.vars)
					newVars[c.Var] = Sequence{item}
					if c.PosVar != "" {
						newVars[c.PosVar] = Sequence{AtomicValue{TypeName: TypeInteger, Value: big.NewInt(int64(i + 1))}}
					}
					newTuples = append(newTuples, flworTuple{vars: newVars})
				}
			}
			tuples = newTuples
		case LetClause:
			for i := range tuples {
				subCtx := ec.withVars(tuples[i].vars)
				val, err := eval(subCtx, c.Expr)
				if err != nil {
					return nil, err
				}
				tuples[i].vars = copyVars(tuples[i].vars)
				tuples[i].vars[c.Var] = val
			}
		case WhereClause:
			var filtered []flworTuple
			for _, tup := range tuples {
				subCtx := ec.withVars(tup.vars)
				r, err := eval(subCtx, c.Predicate)
				if err != nil {
					return nil, err
				}
				b, err := EBV(r)
				if err != nil {
					return nil, err
				}
				if b {
					filtered = append(filtered, tup)
				}
			}
			tuples = filtered
		case OrderByClause:
			sorted, sortErr := sortTuples(ec, tuples, c)
			if sortErr != nil {
				return nil, sortErr
			}
			tuples = sorted
		}
	}

	// Evaluate return expression for each tuple
	var result Sequence
	for _, tup := range tuples {
		retCtx := ec.withVars(tup.vars)
		r, err := eval(retCtx, e.Return)
		if err != nil {
			return nil, err
		}
		result = append(result, r...)
	}
	return result, nil
}

type flworTuple struct {
	vars map[string]Sequence
}

func sortTuples(ec *evalContext, tuples []flworTuple, ob OrderByClause) ([]flworTuple, error) {
	type sortKey struct {
		values []AtomicValue
	}
	keys := make([]sortKey, len(tuples))
	for i, tup := range tuples {
		var vals []AtomicValue
		for _, spec := range ob.Specs {
			subCtx := ec.withVars(tup.vars)
			r, err := eval(subCtx, spec.Expr)
			if err != nil {
				return nil, err
			}
			if len(r) > 0 {
				a, err := AtomizeItem(r[0])
				if err != nil {
					return nil, err
				}
				vals = append(vals, a)
			} else {
				vals = append(vals, AtomicValue{})
			}
		}
		keys[i] = sortKey{values: vals}
	}
	sort.SliceStable(tuples, func(i, j int) bool {
		for k, spec := range ob.Specs {
			cmp := compareAtomicOrder(keys[i].values[k], keys[j].values[k])
			if cmp == 0 {
				continue
			}
			if spec.Descending {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})
	return tuples, nil
}

func compareAtomicOrder(a, b AtomicValue) int {
	// String types: compare lexicographically
	sa, saOK := a.Value.(string)
	sb, sbOK := b.Value.(string)
	if saOK && sbOK {
		if sa < sb {
			return -1
		}
		if sa > sb {
			return 1
		}
		return 0
	}

	// Numeric types: compare as float64
	af := a.ToFloat64()
	bf := b.ToFloat64()
	if af < bf {
		return -1
	}
	if af > bf {
		return 1
	}
	return 0
}

func copyVars(m map[string]Sequence) map[string]Sequence {
	cp := make(map[string]Sequence, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
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
		subCtx := ec.withVar(binding.Var, Sequence{item})
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
	for _, code := range catch.Codes {
		if catchCodeMatches(code, xpErr.Code) {
			return true
		}
	}
	return false
}

// catchCodeMatches compares a catch clause code against an XPath error code.
// The catch code may be:
//   - "*" — matches anything
//   - "err:FOAR0002" or "FOAR0002" — matches specific error code
//   - "err:*" — matches any code in the err namespace (all XPath errors)
//   - "*:FOAR0002" — matches any namespace with that local name
//   - "Q{http://...}FOAR0002" — matches by URI + local name
//   - "Q{http://...}*" — matches any code in that namespace
func catchCodeMatches(catchCode, errCode string) bool {
	if catchCode == "*" {
		return true
	}

	// Handle Q{uri}local and Q{uri}*
	if strings.HasPrefix(catchCode, "Q{") {
		if idx := strings.Index(catchCode, "}"); idx >= 0 {
			uri := catchCode[2:idx]
			local := catchCode[idx+1:]
			if local == "*" {
				return uri == NSErr // Q{err-ns}* matches all XPath errors
			}
			// Q{err-ns}CODE matches if the URI is the error namespace and local matches
			if uri == NSErr {
				return local == errCode
			}
			return false
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
		return catchPrefix == "" || catchPrefix == "err"
	}
	if catchPrefix == "*" {
		return catchLocal == errCode // *:CODE matches the bare code
	}

	// Compare the local part of the catch code against the error code
	return catchLocal == errCode
}

// buildCatchContext creates an eval context with standard $err:* variables.
func buildCatchContext(ec *evalContext, xpErr *XPathError) *evalContext {
	// $err:code should be a QName, but for practical purposes store as string
	// with the err: prefix to match test expectations
	errQN := SingleAtomic(AtomicValue{
		TypeName: TypeQName,
		Value:    QNameValue{Prefix: "err", URI: NSErr, Local: xpErr.Code},
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
