package xslt3

import (
	"context"
	"errors"
	"strings"

	"github.com/lestrrat-go/helium/xpath3"
)

func (ec *execContext) execVariable(ctx context.Context, inst *VariableInst) error {
	var val xpath3.Sequence
	var evalErr error

	if inst.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			// XSLT 3.0 §9.5: circular references are only errors when
			// the variable is actually used. Defer the error so that
			// unused variables with circular dependencies do not cause
			// the transformation to fail.
			if errors.Is(err, ErrCircularRef) {
				ec.setVarDeferred(inst.Name, err)
				return nil
			}
			return err
		}
		val = result.Sequence()
	} else if len(inst.Body) > 0 {
		ec.temporaryOutputDepth++
		if inst.As == "" {
			// Per XSLT spec: variable with content body (no select, no as)
			// produces a document node (temporary tree)
			val, evalErr = ec.evaluateBodyAsDocument(ctx, inst.Body)
		} else if strings.HasPrefix(inst.As, "document-node") {
			// document-node() type: wrap body in document node
			val, evalErr = ec.evaluateBodyAsDocument(ctx, inst.Body)
		} else {
			// With as attribute: evaluate as sequence constructor,
			// keeping each node as a separate item
			val, evalErr = ec.evaluateBodyAsSequence(ctx, inst.Body)
		}
		ec.temporaryOutputDepth--
		if evalErr != nil {
			if errors.Is(evalErr, ErrCircularRef) {
				ec.setVarDeferred(inst.Name, evalErr)
				return nil
			}
			return evalErr
		}
	} else {
		val = xpath3.SingleString("")
	}

	// Type check against the declared as type
	if inst.As != "" {
		st := parseSequenceType(inst.As)
		checked, err := checkSequenceType(val, st, errCodeXTTE0570, "variable $"+inst.Name, ec)
		if err != nil {
			return err
		}
		val = checked
	}

	ec.setVar(inst.Name, val)
	return nil
}

func (ec *execContext) execParam(ctx context.Context, inst *ParamInst) error {
	// Check if already set (by with-param)
	if _, ok := ec.localVars.lookup(inst.Name); ok {
		return nil
	}
	// Use default
	return ec.execVariable(ctx, &VariableInst{
		Name:   inst.Name,
		Select: inst.Select,
		Body:   inst.Body,
		As:     inst.As,
	})
}
