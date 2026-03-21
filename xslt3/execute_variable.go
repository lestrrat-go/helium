package xslt3

import (
	"context"
	"errors"
	"strings"

	"github.com/lestrrat-go/helium"
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
			// document-node()* or document-node()+: evaluate as sequence
			// so that copy-of of multiple documents produces separate items.
			if strings.HasSuffix(inst.As, "*") || strings.HasSuffix(inst.As, "+") {
				val, evalErr = ec.evaluateBodyAsSequence(ctx, inst.Body)
			} else {
				// document-node() or document-node()?: wrap body in document node
				val, evalErr = ec.evaluateBodyAsDocument(ctx, inst.Body)
				// When the as type allows zero occurrences (e.g. document-node()?)
				// and the body produced an empty document (no children), return
				// an empty sequence instead of the empty document. This handles
				// xsl:where-populated discarding all content.
				if evalErr == nil && len(val) == 1 {
					if docItem, ok := val[0].(xpath3.NodeItem); ok {
						if doc, ok := docItem.Node.(*helium.Document); ok && doc.FirstChild() == nil {
							if strings.HasSuffix(inst.As, "?") {
								val = nil
							}
						}
					}
				}
			}
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
		// No select, no body (or empty body after whitespace stripping).
		// XSLT 3.0 §9.3: if as specifies a sequence type whose occurrence
		// indicator is ? or *, the effective value is an empty sequence.
		if inst.As != "" && (strings.HasSuffix(inst.As, "?") || strings.HasSuffix(inst.As, "*")) {
			val = nil
		} else {
			val = xpath3.SingleString("")
		}
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
